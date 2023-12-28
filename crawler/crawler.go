package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

var wg sync.WaitGroup

const httpTimeoutSeconds = 10

// crawlData : struct to store the visited URLs, final result and locks to access the data
type crawlData struct {
	sync.RWMutex
	visited map[string]bool
	result  []string
}

// CrawlWebpage : crawls the webpage as per the provided flags and returns the string slice of crawled links
func CrawlWebpage(rootURL string, maxDepth int, sameSite bool, loadDynamicContent bool) ([]string, error) {
	var crawlSessionData = crawlData{visited: make(map[string]bool), result: []string{}}
	fmt.Printf("CrawlWebpage: rootURL:: %v, maxDepth:: %v, sameSite:: %v, loadDynamicContent:: %v\n", rootURL, maxDepth, sameSite, loadDynamicContent)

	// Add the initial URL to the results
	crawlSessionData.Lock()
	crawlSessionData.result = append(crawlSessionData.result, rootURL)
	crawlSessionData.Unlock()

	// crawl asynchronously for better performance
	wg.Add(1)
	go crawl(rootURL, rootURL, 0, maxDepth, sameSite, loadDynamicContent, &crawlSessionData)

	c := make(chan struct{})
	// Default timeout added as 1 minute, modify if required
	timeout := 1 * time.Minute
	go func() {
		fmt.Printf("CrawlWebpage: Waiting for active goroutines to finish.\n")
		defer close(c)
		// Wait until all goroutines are finished executing
		wg.Wait()
	}()
	select {
	case <-c:
		fmt.Printf("CrawlWebpage: all goroutines finished executing.\n")
	case <-time.After(timeout):
		fmt.Printf("CrawlWebpage: timedout after %s. Returning the processed results.\n", timeout)
	}

	crawlSessionData.Lock()
	crawlSessionData.result = removeDuplicates(crawlSessionData.result)
	crawlSessionData.Unlock()

	return crawlSessionData.result, nil
}

// crawl : function to crawl the webpage and find links. This is expected to be called recursively.
func crawl(url, rootURL string, depth, maxDepth int, sameSite, loadDynamicContent bool, crawlSessionData *crawlData) {
	defer wg.Done()

	crawlSessionData.RLock()
	urlVisited := crawlSessionData.visited[url]
	crawlSessionData.RUnlock()

	if depth > maxDepth || urlVisited {
		return
	}

	crawlSessionData.Lock()
	crawlSessionData.result = append(crawlSessionData.result, url)
	crawlSessionData.visited[url] = true
	crawlSessionData.Unlock()

	fmt.Printf("CrawlWebpage: Crawling %s (depth %d)\n", url, depth)

	if loadDynamicContent {
		// Load the page using Chrome Devtools Protocol to get the fully loaded page
		processDynamicContent(url, rootURL, depth, maxDepth, sameSite, loadDynamicContent, crawlSessionData)
	} else {
		// Load the initial HTML and extract links from it
		processStaticContent(url, rootURL, depth, maxDepth, sameSite, loadDynamicContent, crawlSessionData)
	}
}

// processDynamicContent : Load the page using Chrome Devtools Protocol and get the links in the page
func processDynamicContent(url, rootURL string, depth, maxDepth int, sameSite, loadDynamicContent bool, crawlSessionData *crawlData) {
	var nodes []*cdp.Node

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
	)
	if err != nil {
		fmt.Printf("CrawlWebpage: Error in navigating to URL:: %v, err:: %v\n", url, err)
		return
	}

	// @todo: Add a delay here if required if the dynamic content isn't loaded

	err = chromedp.Run(ctx,
		chromedp.Nodes(`a[href]`, &nodes, chromedp.ByQueryAll),
	)
	if err != nil {
		fmt.Printf("CrawlWebpage: Error in getting links for URL:: %v, err:: %v\n", url, err)
		return
	}

	processNodes(url, rootURL, nodes, depth, maxDepth, sameSite, loadDynamicContent, crawlSessionData)
}

// processStaticContent : Get the HTML response for the URL and get the links from it.
func processStaticContent(url, rootURL string, depth, maxDepth int, sameSite, loadDynamicContent bool, crawlSessionData *crawlData) {
	doc, err := fetchHTML(url)
	if err != nil {
		fmt.Printf("Error fetching %s: %v\n", url, err)
		return
	}

	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")
		if link != "" {
			absURL := resolveURL(url, link)
			processLink(absURL, rootURL, depth, maxDepth, sameSite, loadDynamicContent, crawlSessionData)
		}
	})
}

// processLink : Checks if the URL has been visited, marks it as visited and added the data to the result.
// crawls to the next link if its not visited
func processLink(absURL, rootURL string, depth, maxDepth int, sameSite, loadDynamicContent bool, crawlSessionData *crawlData) {
	crawlSessionData.RLock()
	urlVisited := crawlSessionData.visited[absURL]
	crawlSessionData.RUnlock()

	if !urlVisited && (!sameSite || isSameSite(rootURL, absURL)) {
		crawlSessionData.Lock()
		crawlSessionData.result = append(crawlSessionData.result, absURL)
		crawlSessionData.Unlock()

		wg.Add(1)
		go crawl(absURL, rootURL, depth+1, maxDepth, sameSite, loadDynamicContent, crawlSessionData)
	}
}

// processNodes : processes the nodes of the page and gets links from it
func processNodes(url, rootURL string, nodes []*cdp.Node, depth, maxDepth int, sameSite, loadDynamicContent bool, crawlSessionData *crawlData) {
	for _, node := range nodes {
		link := node.AttributeValue("href")
		if link != "" {
			absURL := resolveURL(url, link)
			processLink(absURL, rootURL, depth, maxDepth, sameSite, loadDynamicContent, crawlSessionData)
		}
	}
}

func fetchHTML(url string) (*goquery.Document, error) {
	HTTPRequestTimeout := httpTimeoutSeconds * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), HTTPRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status code %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	return doc, nil
}

// isSameSite : checks if the absURL is of the same website as the rootURL
func isSameSite(rootURL, absURL string) bool {
	root, _ := url.Parse(rootURL)
	abs, _ := url.Parse(absURL)
	return root.Host == abs.Host
}

// resolveURL : Resolves the path WRT the baseURL and returns the final resolved URL
func resolveURL(baseURL, link string) string {
	base, _ := url.Parse(baseURL)
	rel, _ := url.Parse(link)

	if rel.IsAbs() {
		return rel.String()
	}

	// Otherwise, resolve it relative to the base URL
	return base.ResolveReference(rel).String()
}
