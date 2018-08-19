package parser

import (
	"errors"
	"flag"
	"net/url"
	"strings"
)

// NormalizeURL removes query strings and fragments from the url
func NormalizeURL(currentURL string) string {
	idx := strings.IndexAny(currentURL, "?#")
	if idx != -1 {
		currentURL = currentURL[:idx]
	}
	return currentURL
}

// GetCliArguments grabs the url and number of workers passed into the cli
func GetCliArguments() (string, int, error) {

	var rawurl string
	var workers int
	flag.StringVar(&rawurl, "url", "", "The URL to crawl")
	flag.IntVar(&workers, "workers", 4, "Number of goroutines to spawn concurrently")
	flag.Parse()

	if workers < 1 || workers > 10 {
		return "", -1, errors.New("workers must be less than 10 and greater than 0")
	}
	currentURL, err := url.ParseRequestURI(NormalizeURL(rawurl))
	if err != nil {
		return "", -1, err
	}

	return currentURL.String(), workers, nil
}
