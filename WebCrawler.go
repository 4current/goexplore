package main

import (
"fmt"
"sync"
)

// SafeRegister is safe to use concurrently.
type SafeRegister struct {
	mu sync.Mutex
	v  []string
	o  int
}

// Add increments the counter for the given key.
func (c *SafeRegister) Add(url string) {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	c.v = append(c.v, url)
	c.mu.Unlock()
}

// In determines presence of a url
func (c *SafeRegister) In(url string) bool {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	defer c.mu.Unlock()
	for _, u := range c.v {
		if url == u {
			return true
		}
	}
	return false
}

func (c *SafeRegister) Inc() {
	c.mu.Lock()
	c.o++
	c.mu.Unlock()
}

func (c *SafeRegister) Dec() {
	c.mu.Lock()
	c.o--
	c.mu.Unlock()
}

func (c *SafeRegister) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.o
}


var (
	sreg     = SafeRegister{v: make([]string, 0)}
	ch = make(chan string)
)

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher) {

	sreg.Inc()
	defer sreg.Dec()

	if depth <= 0 {
		return
	}

	var (
		body string
		err  error
		urls []string
	)

	if sreg.In(url) {
		err = fmt.Errorf("already fetched %s", url)
	} else {
		body, urls, err = fetcher.Fetch(url)
	}

	if err != nil {
		ch <- err.Error()
		return
	}
	sreg.Add(url)

	ch <- fmt.Sprintf("found: %s %q\n", url, body)

	for _, u := range urls {
		go Crawl(u, depth-1, fetcher)
	}

	return
}

func main() {
	go Crawl("https://golang.org/", 4, fetcher)

	// now we must create an event loop
	for sreg.Count() == 0 {}
	for sreg.Count() > 0 {
		for res := range ch {
			fmt.Println(res)
			if sreg.Count() == 0 {
				break
			}
		}
	}
}

// fakeFetcher is Fetcher that returns canned results.
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	if res, ok := f[url]; ok {
		return res.body, res.urls, nil
	}
	return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher is a populated fakeFetcher.
var fetcher = fakeFetcher{
	"https://golang.org/": &fakeResult{
		"The Go Programming Language",
		[]string{
			"https://golang.org/pkg/",
			"https://golang.org/cmd/",
		},
	},
	"https://golang.org/pkg/": &fakeResult{
		"Packages",
		[]string{
			"https://golang.org/",
			"https://golang.org/cmd/",
			"https://golang.org/pkg/fmt/",
			"https://golang.org/pkg/os/",
		},
	},
	"https://golang.org/pkg/fmt/": &fakeResult{
		"Package fmt",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
	"https://golang.org/pkg/os/": &fakeResult{
		"Package os",
		[]string{
			"https://golang.org/",
			"https://golang.org/pkg/",
		},
	},
}
