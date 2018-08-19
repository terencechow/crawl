package crawler_test

import (
	"fmt"
	"github.com/terencechow/crawl/crawler"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"testing"
)

type Node = crawler.Node
type Info = crawler.Info

func TestGetDomainLinks(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w,
			` <html>
        <body>
          <a href='http://www.external.com'>external link</a>
          <a href='http://www.domain.com/about'>domain link</a>
          <a href='http://subdomain.domain.com/something'>subdomain link</a>
          <a href='http://user:pass@www.domain.com/authenticated'>link with user & pass</a>
          <a href='//www.domain.com/noscheme'>link without scheme</a>
          <a href='/relative'>relative link</a>
          <a href='http://www.domain.com/about'>same link doesnt add twice</a>
        </body>
      </html>`)
	}

	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()
	handler(w, req)
	resp := w.Result()
	currentURL, _ := url.Parse("http://www.domain.com")
	links, err := crawler.GetDomainLinks(currentURL, resp.Body)
	if err != nil {
		t.Error("Expected no error when retrieving links got", err)
	}

	expectedLinks := []string{
		"http://www.domain.com/about",
		"http://user:pass@www.domain.com/authenticated",
		"http://www.domain.com/noscheme",
		"http://www.domain.com/relative",
	}
	sort.Strings(links)
	sort.Strings(expectedLinks)
	if !reflect.DeepEqual(links, expectedLinks) {
		t.Error(fmt.Sprintf("Expected %v. Got %v", expectedLinks, links))
	}
}

func TestGetNodeFromSitemap(t *testing.T) {
	parentmap := map[string]string{
		"root-url": "ROOT",
		"depth1-a": "root-url",
		"depth1-b": "root-url",
		"depth2-a": "depth1-a",
		"depth2-b": "depth1-a",
	}
	sitemap := &Node{
		URL: "root-url",
		Links: map[string]*Node{
			"depth1-a": &Node{
				URL: "depth1-a",
				Links: map[string]*Node{
					"depth2-a": &Node{URL: "depth2-a"},
					"depth2-b": &Node{URL: "depth2-b"},
				},
			},
			"depth1-b": &Node{URL: "depth1-b"},
		},
	}

	info := &Info{
		Sitemap:   sitemap,
		Parentmap: parentmap,
	}
	node := info.GetNodeFromSitemap("root-url")
	expected := sitemap
	if node != expected {
		t.Error(fmt.Sprintf("Expected %v. Got %v", expected, node))
	}

	node = info.GetNodeFromSitemap("depth1-b")
	expected = sitemap.Links["depth1-b"]
	if node != expected {
		t.Error(fmt.Sprintf("Expected %v. Got %v", expected, node))
	}

	node = info.GetNodeFromSitemap("depth2-b")
	expected = sitemap.Links["depth1-a"].Links["depth2-b"]
	if node != expected {
		t.Error(fmt.Sprintf("Expected %v. Got %v", expected, node))
	}
}

func TestSiteMap(t *testing.T) {
	// generate a test server so we can capture and inspect the request
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		var content string
		path := req.URL.EscapedPath()

		if path == "/redirect" {
			http.Redirect(res, req, fmt.Sprintf("http://%s/after-redirect", req.Host), 301)
			return
		}

		res.WriteHeader(http.StatusOK)
		if path == "/" {
			content = fmt.Sprintf(`
        <html>
          <body>
            <a href='http://www.external.com'>external link</a>
            <a href='http://%s/about'>domain link</a>
            <a href='http://%s/redirect'>redirect link</a>
          </body>
        </html>
      `, req.Host, req.Host)
		} else if path == "/about" {
			content = `
        <html>
          <body>
            <a href='/relative'>relative link</a>
          </body>
        </html>
      `
		} else if path == "/relative" {
			content = fmt.Sprintf(`
        <html>
          <body>
            <a href='http://%s/about'>domain link</a>
          </body>
        </html>
      `, req.Host)
		} else if path == "/after-redirect" {
			content = fmt.Sprintf(`
        <html>
          <body>
            <a href='http://%s/about'>domain link</a>
          </body>
        </html>
      `, req.Host)
		}
		res.Write([]byte(content))
	}))
	defer ts.Close()

	sitemap := crawler.CreateSiteMap(ts.URL, 1)

	rootURL := ts.URL
	aboutURL := ts.URL + "/about"
	relativeURL := ts.URL + "/relative"
	redirectedURL := ts.URL + "/after-redirect"
	expected := &Node{
		URL: rootURL,
		Links: map[string]*Node{
			aboutURL: &Node{
				URL: aboutURL,
				Links: map[string]*Node{
					relativeURL: &Node{
						URL: relativeURL,
						Links: map[string]*Node{
							aboutURL: &Node{
								URL:   aboutURL,
								Links: map[string]*Node{},
							},
						},
					},
				},
			},
			redirectedURL: &Node{
				URL: redirectedURL,
				Links: map[string]*Node{
					aboutURL: &Node{
						URL:   aboutURL,
						Links: map[string]*Node{},
					},
				},
			},
		},
	}
	if !reflect.DeepEqual(sitemap, expected) {
		t.Error(fmt.Sprintf("Expected %v. Got %v", expected, sitemap))
	}
}
