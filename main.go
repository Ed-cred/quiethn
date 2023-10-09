package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/Ed-cred/quiethn/hn"
)

type item struct {
	hn.Item
	Host string
}

type Data struct {
	Stories []item
	Time    time.Duration
}

func main() {
	var port, numStories int
	flag.IntVar(&numStories, "stories", 30, "Number of stories to display on screen")
	flag.IntVar(&port, "port", 3000, "Port number to listen on")
	flag.Parse()

	tpl := template.Must(template.ParseFiles("./index.gohtml"))
	http.HandleFunc("/", handler(numStories, tpl))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handler(numStories int, tpl *template.Template) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		stories, err := getTopStories(numStories)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := Data{
			Stories: stories,
			Time:    time.Now().Sub(start),
		}

		err = tpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

func getTopStories(numStories int) ([]item, error) {
	var client hn.Client
	ids, err := client.TopItems()
	if err != nil {
		return nil, errors.New("failed to load top stories")
	}
	type result struct {
		idx  int
		item item
		err  error
	}
	resChan := make(chan result, len(ids))
	for i := 0; i < numStories; i++ {
		go func(idx, id int) {
			hnItem, err := client.GetItem(id)
			if err != nil {
				resChan <- result{idx: idx, err: err}
			}
			resChan <- result{idx: idx, item: parseHnItem(hnItem)}
		}(i, ids[i])
	}
	var results []result
	for i := 0; i < numStories; i++ {
		results = append(results, <-resChan)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].idx < results[j].idx
	})
	var stories []item
	for _, res := range results {
		if res.err != nil {
			continue
		}
		if isStory(res.item) {
			stories = append(stories, res.item)
		}
	}
	return stories, nil
}

func isStory(item item) bool {
	return item.Type == "story" && item.URL != ""
}

func parseHnItem(hnitem hn.Item) item {
	ret := item{Item: hnitem}
	url, err := url.Parse(ret.URL)
	if err == nil {
		ret.Host = strings.TrimPrefix(url.Hostname(), "www.")
	}
	return ret
}
