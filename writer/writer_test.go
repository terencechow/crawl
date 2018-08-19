package writer_test

import (
	"fmt"
	"github.com/terencechow/crawl/crawler"
	"github.com/terencechow/crawl/writer"
	"testing"
)

type Node = crawler.Node

func TestPrettifySiteMap(t *testing.T) {
	rootURL := "https://example.com"
	aboutURL := rootURL + "/about"
	faqURL := rootURL + "/faq"
	anotherURL := rootURL + "/another"
	somethingUnderAnother := rootURL + "/something-under-another"

	sitemap := &Node{
		URL: rootURL,
		Links: map[string]*Node{
			anotherURL: &Node{
				URL: anotherURL,
				Links: map[string]*Node{
					somethingUnderAnother: &Node{
						URL:   somethingUnderAnother,
						Links: map[string]*Node{},
					},
				},
			},
			aboutURL: &Node{
				URL: aboutURL,
				Links: map[string]*Node{
					faqURL: &Node{
						URL: faqURL,
						Links: map[string]*Node{
							aboutURL: &Node{
								URL:   aboutURL,
								Links: map[string]*Node{},
							},
						},
					},
				},
			},
		},
	}

	expected := "" +
		"https://example.com\n" +
		"\thttps://example.com/about\n" +
		"\t\thttps://example.com/faq\n" +
		"\t\t\thttps://example.com/about\n" +
		"\thttps://example.com/another\n" +
		"\t\thttps://example.com/something-under-another\n"

	result := writer.PrettifySiteMap(sitemap, 0)
	if result != expected {
		t.Error(fmt.Sprintf("Expected:\n%q\nGot:\n%q\n", expected, result))
	}
}
