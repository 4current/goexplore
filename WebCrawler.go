package main

import (
	"fmt"
	"sync"
)

// SafeRegister is safe to use concurrently.
type SafeRegister struct {
	mu sync.Mutex
	v  map[string]bool
}

// Add increments the counter for the given key.
func (c *SafeRegister) Add(url string) {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	c.v[url] = true
	c.mu.Unlock()
}

// In determines presence of a url
func (c *SafeRegister) In(url string) bool {
	c.mu.Lock()
	// Lock so only one goroutine at a time can access the map c.v.
	defer c.mu.Unlock()
	return c.v[url]
}

var (
	sreg = SafeRegister{v: make(map[string]bool, 0)}
)

type Fetcher interface {
	// Fetch returns the body of URL and
	// a slice of URLs found on that page.
	Fetch(url string) (body string, urls []string, err error)
}

// Crawl uses fetcher to recursively crawl
// pages starting with url, to a maximum of depth.
func Crawl(url string, depth int, fetcher Fetcher) <-chan string {

	// Create a channel to be returned by function
	ch := make(chan string)
	chin := make(<-chan string)

	go func() {
		defer close(ch)

		if depth <= 0 {
			return
		}

		var (
			body string
			err  error
			urls []string
		)

		if sreg.In(url) {
			ch <- fmt.Sprintf("already fetched %s", url)
			return
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
			chin = Crawl(u, depth-1, fetcher)
			for res := range chin {
				ch <- res
			}
		}
	}()
	return ch
}

func main() {
	ch := Crawl("https://golang.org/", 4, fetcher)

	// now we must create an event loop
	for res := range ch {
		fmt.Println(res)
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
