package main

import (
	"flag"
	"fmt"
	"github.com/ankurnarkhede/golang-web-crawler/crawler"
	"log"
)

func main() {
	const (
		defaultURL                = "https://duckduckgo.com"
		defaultMaxDepth           = 1
		defaultSameSite           = false
		defaultLoadDynamicContent = false
	)
	urlFlag := flag.String("url", defaultURL, "the url that you want to crawl")
	maxDepth := flag.Int("depth", defaultMaxDepth, "the maximum number of links deep to traverse")
	sameSiteFlag := flag.Bool("sameSite", defaultSameSite, "Check if links are under the same site")
	loadDynamicContentFlag := flag.Bool("loadDynamicContent", defaultLoadDynamicContent, "Load dynamic content on the webpage. This may slow down the execution speed.")
	flag.Parse()

	links, err := crawler.CrawlWebpage(*urlFlag, *maxDepth, *sameSiteFlag, *loadDynamicContentFlag)
	if err != nil {
		log.Fatalln("ERROR:", err)
	}
	fmt.Println("Links")
	fmt.Println("-----")
	for i, l := range links {
		fmt.Printf("%03d. %s\n", i+1, l)
	}
	fmt.Println()
}
