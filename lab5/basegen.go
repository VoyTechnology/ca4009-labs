package main

import (
	"flag"
	"fmt"
	"net/url"

	"github.com/golang/glog"
)

var (
	queryURLFlag = flag.String("query-url",
		"http://computing.dcu.ie/~sprocheta/lab5/query/Query_%s.txt",
		"query url format string")
	qrelURLFlag = flag.String("qrel-url",
		"http://computing.dcu.ie/~sprocheta/lab5/qrel_%s.txt",
		"qrel url format string")
	tokenFlag        = flag.String("token", "", "token to use")
	retrievalURLFlag = flag.String("retrieval-url",
		"http://clueweb.adaptcentre.ie/ClueWebNew/search",
		"retrieval url base path")
)

func main() {
	flag.Parse()
	glog.Info("Starting base generator")

	queryURL, err := url.Parse(fmt.Sprintf(*queryURLFlag, *tokenFlag))
	if err != nil {
		glog.Fatalf("can't parse queryURL: %v", err)
	}

	qrelURL, err := url.Parse(fmt.Sprintf(*qrelURLFlag, *tokenFlag))
	if err != nil {
		glog.Fatalf("can't parse qrelURL: %v", err)
	}

	retrievalURL, err := url.Parse(*retrievalURLFlag)
	if err != nil {
		glog.Fatalf("%v", err)
	}

	_, err = getQrel(qrelURL)
	if err != nil {
		glog.Fatalf("error getting qrel: %v", err)
	}

	baseline := []BaseData{}

	queries, err := getQueries(queryURL)
	if err != nil {
		glog.Fatalf("error getting qrel: %v", err)
	}
	for _, query := range queries {
		q := retrievalURL.Query()
		q.Set("query", query.Title)
		b := BaseData{
			QID: query.Num,
		}
		baseline = append(baseline, b)
	}
}

// SearchResult containing the parsed JSON from the search page.
type SearchResult struct {
	Title   string  `json:"title,omitempty"`
	Snippet string  `json:"snippet,omitempty"`
	ID      string  `json:"id,omitempty"`
	Score   float64 `json:"score,omitempty"`
	URL     string  `json:"url,omitempty"`
}

// BaseData which is used for running trec_eval
type BaseData struct {
	QID string
}

// QueryResult of getting the query file
type QueryResult struct {
	Num   string
	Title string
}

func getQueries(path *url.URL) ([]QueryResult, error) {
	return nil, nil
}

func getQrel(path *url.URL) (string, error) {
	return "", nil
}

func trecEval(qrel string, b *BaseData) error {
	return nil
}
