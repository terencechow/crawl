package writer

import (
	"fmt"
	"github.com/terencechow/crawl/crawler"
	"sort"
)

// PrettifySiteMap prints the sitemap to a string using tabs for different depths
func PrettifySiteMap(sitemap *crawler.Node, depth int) string {
	result := ""
	tabs := ""
	for i := 0; i < depth; i++ {
		tabs += "\t"
	}

	if depth == 0 {
		result += fmt.Sprintf("%s%s\n", tabs, sitemap.URL)
	}

	// sort the links alphabetically
	keys := make([]string, len(sitemap.Links))
	i := 0
	for k := range sitemap.Links {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := sitemap.Links[k]
		result += fmt.Sprintf("\t%s%s\n", tabs, k)
		result += PrettifySiteMap(v, depth+1)
	}
	return result
}
