package main

import (
	_ "embed"
	"fmt"
	"github.com/ankurnarkhede/golang-web-crawler/crawler"
	"github.com/stretchr/testify/assert"
	"html/template"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
)

// This is a test package that is useful for testing various website configuration

func TestCrawler(t *testing.T) {
	f := func(seed int64, maxDepth int) bool {
		srv, exp := testRandomServer(seed, maxDepth)
		fmt.Printf("srv.URL:: %v, seed:: %v, maxDepth:: %v\n", srv.URL, seed, maxDepth)
		links, err := crawler.CrawlWebpage(srv.URL, maxDepth, false, false)
		srv.Close()
		if err != nil {
			t.Errorf("CrawlWebpage: error=%v", err)
			return false
		}

		// Replaced assert.Equal() with assert.ElementsMatch() to match only with the elements and not the order of the links.
		// The order of the links doesnt matter on our use case, the result does.
		// The crawler is optimised to execute asynchronously which might lead to links not added in the expected order.
		if !assert.ElementsMatch(t, exp, links) {
			return false
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{
		Values: func(v []reflect.Value, r *rand.Rand) {
			v[0] = reflect.ValueOf(r.Int63())        // random seed
			v[1] = reflect.ValueOf(int(r.Int63n(6))) // max depth between [0,5]
		},
	}); err != nil {
		t.Fatal(err)
	}
}

const (
	// genDepth is the number of pages deep testRandomServer will create for its simulated website
	genDepth = 10
	// maxPages is the maximum number of web pages it will create at each depth level (between 1 and maxPages inclusive)
	maxPages = 5
)

// testRandomServer starts up a local http server with a generated website
// and returns the list of expected links the crawler should find on this site given the maxDepth
// the seed value ensures we can re-create the same exact website while still being able to generate
// a random layout each time.
func testRandomServer(seed int64, maxDepth int) (*httptest.Server, []string) {
	fmt.Printf("testRandomServer:: STARTING SERVER")
	type link struct {
		ToNum int
	}
	type page struct {
		Num   int
		Depth int
		Links []link
	}
	rng := rand.New(rand.NewSource(seed))
	var index page
	allPages := make([]*page, 1)
	allPages[0] = &index
	var p []*page
	np := []*page{&index}
	var pageNum int
	for d := 1; d <= genDepth; d++ { // generate genDepth levels of links
		// Pick a random number of new pages at this depth from 1 to maxPages inclusive
		pagesAtDepth := 1 + int(rng.Int63n(maxPages))
		// get the last set of pages and make those the parents [p]
		// make the new set of pages an empty list [np]
		p, np = np, make([]*page, pagesAtDepth)
		for pd := range np {
			// create the new page
			pageNum++
			thisPage := page{
				Num:   pageNum,
				Depth: d,
			}
			allPages = append(allPages, &thisPage)
			np[pd] = &thisPage
			// decide how many inbound links there should be to this new page based on the number of parents we have
			maxLinks := int(rng.Int63n(int64(len(p)))) + 1
			for i := 0; i < maxLinks; i++ { // for each inbound link we want
				// grab next parent and create a link to this page
				parent := p[i]
				parent.Links = append(parent.Links, link{
					ToNum: pageNum,
				})
			}
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.WriteHeader(http.StatusOK)
			_ = pageTemplate.Execute(w, allPages[0])
			return
		}
		path := strings.Trim(r.URL.Path, "/")
		i, err := strconv.Atoi(path)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if i < 0 || i >= len(allPages) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = pageTemplate.Execute(w, allPages[i])
	}))
	var expected []string
	links := make(map[string]struct{})
	for _, thisPage := range allPages {
		// Modified the condition here to > rather than >= so as to also consider the depth 0 while calculating results
		if thisPage.Depth > maxDepth {
			break
		}
		href := srv.URL
		if thisPage.Num > 0 {
			href = fmt.Sprintf("%s/%d", srv.URL, thisPage.Num)
		}
		links[href] = struct{}{}
		for _, l := range thisPage.Links {
			href = fmt.Sprintf("%s/%d", srv.URL, l.ToNum)
			links[href] = struct{}{}
		}
	}
	for k := range links {
		expected = append(expected, k)
	}
	sort.Strings(expected)
	return srv, expected
}

//go:embed testdata/templates/page.tmpl
var pageTemplateStr string
var pageTemplate = template.Must(template.New("page").Parse(pageTemplateStr))
