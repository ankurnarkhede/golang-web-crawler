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

type crawlData struct {
	sync.RWMutex
	visited map[string]bool
	result  []string
}

func CrawlWebpage(rootURL string, maxDepth int, sameSite bool, loadDynamicContent bool) ([]string, error) {
	var crawlSessionData = crawlData{visited: make(map[string]bool), result: []string{}}
	fmt.Printf("CrawlWebpage: rootURL:: %v, maxDepth:: %v, sameSite:: %v, loadDynamicContent:: %v\n", rootURL, maxDepth, sameSite, loadDynamicContent)

	// Add the initial URL to the results
	crawlSessionData.Lock()
	crawlSessionData.result = append(crawlSessionData.result, rootURL)
	crawlSessionData.Unlock()

	var crawl func(string, int, *sync.WaitGroup)
	crawl = func(url string, depth int, wg *sync.WaitGroup) {
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
			var nodes []*cdp.Node

			// Create context
			ctx, cancel := chromedp.NewContext(context.Background())
			defer cancel()

			// Navigate to the website
			err := chromedp.Run(ctx,
				chromedp.Navigate(url),
			)
			if err != nil {
				fmt.Printf("CrawlWebpage: Error in navigating to URL:: %v, err:: %v", url, err)
				return
			}

			// @todo: Add a delay here if required if the dynamic content isnt loaded

			// Extract links as nodes
			err = chromedp.Run(ctx,
				chromedp.Nodes(`a[href]`, &nodes, chromedp.ByQueryAll),
			)
			if err != nil {
				fmt.Printf("CrawlWebpage: Error in getting links for URL:: %v, err:: %v", url, err)
				return
			}

			// Convert nodes to strings
			for _, node := range nodes {
				link := node.AttributeValue("href")
				if link != "" {
					absURL := resolveURL(rootURL, url, link)
					crawlSessionData.RLock()
					urlVisited := crawlSessionData.visited[absURL]
					crawlSessionData.RUnlock()

					if !urlVisited && (!sameSite || isSameSite(rootURL, absURL)) {
						crawlSessionData.Lock()
						crawlSessionData.result = append(crawlSessionData.result, absURL)
						crawlSessionData.Unlock()

						wg.Add(1)
						go crawl(absURL, depth+1, wg)
					}
				}
			}
		} else {
			doc, err := fetchHTML(url)
			if err != nil {
				fmt.Printf("Error fetching %s: %v\n", url, err)
				return
			}

			// Parse and collect links
			doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
				link, _ := s.Attr("href")
				if link != "" {
					absURL := resolveURL(rootURL, url, link)

					crawlSessionData.RLock()
					urlVisited := crawlSessionData.visited[absURL]
					crawlSessionData.RUnlock()
					if !urlVisited && (!sameSite || isSameSite(rootURL, absURL)) {
						crawlSessionData.Lock()
						crawlSessionData.result = append(crawlSessionData.result, absURL)
						crawlSessionData.Unlock()

						wg.Add(1)
						go crawl(absURL, depth+1, wg)
					}
				}
			})
		}
	}

	wg.Add(1)
	go crawl(rootURL, 0, &wg)

	c := make(chan struct{})
	timeout := 1 * time.Minute
	go func() {
		fmt.Printf("CrawlWebpage: Waiting for active goroutines to finish.\n")
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		fmt.Printf("CrawlWebpage: all goroutines finished executing.\n")
		crawlSessionData.Lock()
		crawlSessionData.result = removeDuplicates(crawlSessionData.result)
		crawlSessionData.Unlock()

		return crawlSessionData.result, nil
	case <-time.After(timeout):
		fmt.Printf("CrawlWebpage: timedout after %s. Returning the processed results.\n", timeout)
		crawlSessionData.Lock()
		crawlSessionData.result = removeDuplicates(crawlSessionData.result)
		crawlSessionData.Unlock()

		return crawlSessionData.result, nil
	}
}

func fetchHTML(url string) (*goquery.Document, error) {
	resp, err := http.Get(url)
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

func isSameSite(rootURL, absURL string) bool {
	root, _ := url.Parse(rootURL)
	abs, _ := url.Parse(absURL)
	return root.Host == abs.Host
}

// Update resolveURL to handle relative URLs more robustly
func resolveURL(rootURL, baseURL, link string) string {
	base, _ := url.Parse(baseURL)
	rel, _ := url.Parse(link)

	// Check if the link is absolute
	if rel.IsAbs() {
		return rel.String()
	}

	// Otherwise, resolve it relative to the base URL
	return base.ResolveReference(rel).String()
}
