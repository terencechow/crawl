package crawler

import (
	"bytes"
	"errors"
	"github.com/terencechow/crawl/parser"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"
)

// Enums for visiting / visited status of a url
type VisitStatus int

const (
	VISITING VisitStatus = 1
	VISITED  VisitStatus = 2
)

// data structure for tracking visiting / visited status
type VisitState struct {
	urlmap map[string]VisitStatus
	sync.Mutex
}

// Tree like data structure representing a node in the sitemap
type Node struct {
	URL   string
	Links map[string]*Node
}

// data structure for tracking the full Sitemap and a "Parentmap"
// the Parentmap tracks each url's parent url, making it easy to
// grab a specific node in the sitemap given a url
type Info struct {
	Sitemap   *Node
	Parentmap map[string]string
	sync.Mutex
}

// Constant to indicate a node in the Parentmap has no parent (ie is the root url)
const ROOT = "ROOT"

// GetNodeFromSitemap returns a specific node in the sitemap given a url
func (self *Info) GetNodeFromSitemap(currentURL string) *Node {
	path := []string{}

	nextURL := currentURL
	for self.Parentmap[nextURL] != "" && self.Parentmap[nextURL] != ROOT {
		path = append(path, nextURL)
		nextURL = self.Parentmap[nextURL]
	}

	// root node, return original Sitemap
	if len(path) == 0 {
		return self.Sitemap
	}

	// not root, return node of Sitemap
	temp := self.Sitemap

	// note the path is from node to root, so we iterate backwards to get from root to node
	for i := len(path) - 1; i > 0; i-- {
		currentNode := path[i]
		temp = temp.Links[currentNode]
	}
	return temp.Links[currentURL]
}

// data structure tracking every link that *WILL* be visited
// This is separate from what's visited / visiting because the locks won't block each other.
// toVisit is for tracking every link that needs to be visited
// in order to compare with what we have visited and figure out when crawling is complete
type ToVisit struct {
	urlmap map[string]bool
	sync.Mutex
}

/* Global variables */
// instances for above data structures
var info = &Info{
	Sitemap:   &Node{},
	Parentmap: make(map[string]string),
}
var toVisit = &ToVisit{urlmap: make(map[string]bool)}
var visitState = &VisitState{urlmap: make(map[string]VisitStatus)}

// queue channel
var queue = make(chan string)

// quit channel
var quit = make(chan bool)

// CreateSiteMap crawls a url, all available links on that page and returns a Sitemap
// If a link is already fetched it does not fetch that link again
func CreateSiteMap(rootURL string, numWorkers int) *Node {

	// initialize Sitemap
	info.Sitemap.URL = rootURL
	info.Sitemap.Links = make(map[string]*Node)

	// initialize Parentmap
	info.Parentmap[rootURL] = ROOT

	// initialize toVisit
	toVisit.urlmap[rootURL] = true

	// create goroutines to wait on queue
	for i := 0; i < numWorkers; i++ {
		go processQueue(i)
	}

	log.Println("Initializing queue...")
	// add rootURL to queue to start processing
	queue <- rootURL

	// block until all channels visited
	<-quit
	log.Println("Done crawling...")

	return info.Sitemap
}

// check if we have crawled every link
func terminateIfComplete() {
	if len(queue) != 0 {
		return
	}

	// if queue is 0 and the numberToVisit matches numberVisited
	toVisit.Lock()
	numToVisit := len(toVisit.urlmap)
	toVisit.Unlock()
	numVisited := 0
	stillVisiting := false

	visitState.Lock()
	for _, visitState := range visitState.urlmap {
		if visitState == VISITING {
			stillVisiting = true
			break
		} else if visitState == VISITED {
			numVisited += 1
		}
	}
	visitState.Unlock()

	if !stillVisiting && numToVisit == numVisited {
		// # of visited sites matches the number toVisit then we are done visiting every site
		quit <- true
	}
}

// processQueue blocks on the queue and crawls one url at a time. Links from the url are then added to the queue
func processQueue(id int) {
	for currentURL := range queue {
		// lock to ensure concurrent handlers don't process same url
		visitState.Lock()
		if status, urlInMap := visitState.urlmap[currentURL]; urlInMap && (status == VISITING || status == VISITED) {
			visitState.Unlock()
			terminateIfComplete()
			continue
		} else {
			visitState.urlmap[currentURL] = VISITING
			visitState.Unlock()
		}

		// crawl & get links
		log.Printf("Goroutine #%v: crawling %s ...\n", id, currentURL)
		links, err := crawl(currentURL, 1)
		if err != nil {
			// for redirects no need to log an error since its not an error and the redirect has been added to queue
			if redirectRegex := regexp.MustCompile(`^3\d\d$`); redirectRegex.MatchString(err.Error()) {
				log.Printf("Goroutine #%v: Error crawling %s, %s\n", id, currentURL, err)
			}

			// for errors we don't add the url back to the queue
			// because the url may be genuinely inaccessible to us and we don't want a circular dependency
			visitState.Lock()
			visitState.urlmap[currentURL] = VISITED
			visitState.Unlock()
			terminateIfComplete()
			continue
		}

		// grab a specific node in the Sitemap
		info.Lock()
		temp := info.GetNodeFromSitemap(currentURL)

		// update links of specific node in Sitemap and update Parentmap for each link
		// we also add each link to the toVisit map so we must lock that as well
		toVisit.Lock()
		for _, link := range links {
			// if a page links to itself no need to include it in Sitemap
			if link != currentURL && temp.Links[link] == nil {
				temp.Links[link] = &Node{URL: link, Links: make(map[string]*Node)}
			}

			if info.Parentmap[link] == "" {
				info.Parentmap[link] = currentURL
			}
			toVisit.urlmap[link] = true
		}
		toVisit.Unlock()
		info.Unlock()

		go func() {
			for _, link := range links {
				queue <- link
			}
		}()

		// update the state of this url to visited
		visitState.Lock()
		visitState.urlmap[currentURL] = VISITED
		visitState.Unlock()

		terminateIfComplete()
	}
}

// handle relative paths
func resolveIfRelativePath(currentURL *url.URL, nextURL *url.URL) *url.URL {
	if nextURL.Host == "" {
		return currentURL.ResolveReference(nextURL)
	}
	return nextURL
}

// GetDomainLinks parses a response body and returns all links within the same domain
func GetDomainLinks(currentURL *url.URL, body io.ReadCloser) ([]string, error) {
	links := map[string]bool{}
	tokenizer := html.NewTokenizer(body)
	var hrefAttr []byte = []byte("href") // used in bytes.Compare to find href attribute
	var anchor []byte = []byte("a")      // used in bytes.Compare to find anchor tags

	for {
		// iterate over tokens
		tokenType := tokenizer.Next()
		switch tokenType {

		case html.ErrorToken:
			err := tokenizer.Err()
			if err.Error() != "EOF" {
				return nil, err
			}

			// done iterating on links, convert links which was a map into an array
			keys := make([]string, len(links))
			i := 0
			for k := range links {
				keys[i] = k
				i++
			}
			// return the array of links
			return keys, nil

		case html.StartTagToken:
			// if token is an anchor tag...
			if tagName, moreAttr := tokenizer.TagName(); bytes.Equal(tagName, anchor) && moreAttr {
				var key, val []byte

				// and if it has attributes
				for moreAttr {
					key, val, moreAttr = tokenizer.TagAttr()
					// check if its a href attribute
					if bytes.Equal(key, hrefAttr) {
						// grab url from href
						nextURL, err := url.Parse(string(val))
						if err != nil {
							log.Print("Error parsing", err)
							return nil, err
						}

						// handle if relative path
						nextURL = resolveIfRelativePath(currentURL, nextURL)

						// if host matches but scheme missing set the scheme
						if nextURL.Host == currentURL.Host && nextURL.Scheme == "" {
							nextURL.Scheme = currentURL.Scheme
						}

						// get Sitemap only for same domain, scheme & if the link isnt a duplicate
						if normalizedURL := parser.NormalizeURL(nextURL.String()); !links[normalizedURL] &&
							nextURL.Host == currentURL.Host && nextURL.Scheme == currentURL.Scheme {
							links[normalizedURL] = true
						}
					}
				}
			}
		}
	}
}

// crawl fetches the page and calls GetDomainLinks to return links form the same domain
func crawl(rawURL string, retryDelay int) ([]string, error) {
	var client = &http.Client{
		Timeout: time.Second * 10,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(rawURL)
	if err != nil {
		log.Print("Error with request", err)
		return nil, err
	}
	defer resp.Body.Close()

	currentURL, err := url.Parse(rawURL)
	if err != nil {
		log.Print("Error parsing URL", err)
		return nil, err
	}

	if resp.StatusCode > 300 && resp.StatusCode < 400 {
		// for redirects we don't follow automatically because they may be a different subdomain and we don't use different subdomains in Sitemaps
		// ie www.blah.com/blog -> blog.blah.com
		nextURL, err := url.Parse(resp.Header.Get("location"))
		if err != nil {
			log.Print("Error parsing URL from header location", err)
			return nil, err
		}
		nextURL = resolveIfRelativePath(currentURL, nextURL)

		if nextRawURL := parser.NormalizeURL(nextURL.String()); nextRawURL != rawURL && currentURL.Host == nextURL.Host {

			// mark the redirect target as something toVisit and add it to the queue
			toVisit.Lock()
			toVisit.urlmap[nextRawURL] = true
			toVisit.Unlock()
			go func() {
				queue <- nextRawURL
			}()

			// update Parentmap & Sitemap, removing the old link and adding the redirect target
			info.Lock()
			if parentURL := info.Parentmap[rawURL]; parentURL != "" && info.Parentmap[nextRawURL] == "" {
				info.Parentmap[nextRawURL] = parentURL

				// update Sitemap to add the final target of redirections
				parentNode := info.GetNodeFromSitemap(parentURL)
				parentNode.Links[nextRawURL] = &Node{
					URL:   nextRawURL,
					Links: make(map[string]*Node),
				}

				// remove old link of redirect
				delete(parentNode.Links, rawURL)
			}
			info.Unlock()
		}

		return nil, errors.New(string(resp.StatusCode))
	} else if resp.StatusCode < 200 || resp.StatusCode > 400 {
		if resp.StatusCode > 499 && retryDelay <= 16 {
			// treat 500 errors as the website's problem not ours, retry the crawl with a delay
			log.Printf("Failed %v on %s. Retrying in %v seconds \n", resp.StatusCode, rawURL, retryDelay)
			time.Sleep(time.Duration(retryDelay) * time.Second)
			return crawl(rawURL, retryDelay*2)
		} else {
			// 400 errors like bad request, unauthorized, etc
			// will never succeed even with a backoff so we just return an error
			return nil, errors.New(resp.Status)
		}
	}

	links, err := GetDomainLinks(currentURL, resp.Body)
	if err != nil {
		log.Print("Error geting domain links", err)
		return nil, err
	}

	return links, nil
}
