package main

import (
	"github.com/terencechow/crawl/crawler"
	"github.com/terencechow/crawl/parser"
	"github.com/terencechow/crawl/writer"
	"io/ioutil"
	"log"
)

func main() {
	url, numHandlers, err := parser.GetCliArguments()
	if err != nil {
		log.Fatal(err)
	}

	sitemap := crawler.CreateSiteMap(url, numHandlers)
	prettified := writer.PrettifySiteMap(sitemap, 0)

	// write to file
	data := []byte(prettified)
	err = ioutil.WriteFile("./sitemap.txt", data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
