
### Prerequisites

This was built with go1.10.2. It is recommended to use go1.10.2+ to build and test.

### Download

`go get github.com/terencechow/crawl`

### Testing

cd into that directory and run:

`go test ./...`

### Install

Installing to your gopath/bin

`go install`

### Run with

`YOUR/GO/PATH/bin/crawl -url=https://monzo.com/`

You can optionally pass a workers argument ie `-workers=5` to indicate number of concurrent workers to spawn. Workers must be a number between 1 and 10 inclusive. Default is 4 workers.

`YOUR/GO/PATH/bin/crawl -url=https://monzo.com/ -workers=10`

This command will crawl a website and all related links reachable from the original url where subdomain and domain matches. Once complete, it will write the sitemap to `sitemap.txt`.

Note that I treated subdomains as different urls because of this
[recommendation.](https://webmasters.stackexchange.com/questions/82687/sitemaps-one-per-subdomain-or-one-for-the-base-domain)
