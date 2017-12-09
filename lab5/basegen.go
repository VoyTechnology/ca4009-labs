package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/golang/glog"
)

var (
	queryURLFlag = flag.String("query-url",
		"http://computing.dcu.ie/~sprocheta/lab5/query/Query_%s.txt",
		"query url format string")
	qrelURLFlag = flag.String("qrel-url",
		"http://computing.dcu.ie/~sprocheta/lab5/qrels/qrel_%s.txt",
		"qrel url format string")
	tokenFlag        = flag.String("token", "", "token to use")
	retrievalURLFlag = flag.String("retrieval-url",
		"http://clueweb.adaptcentre.ie/ClueWebNew/search",
		"retrieval url base path")
	trecEvalPathFlag = flag.String("trec_eval", "", "path to trec_eval")
)

func main() {
	flag.Parse()
	glog.Info("Starting base generator")

	if *tokenFlag == "" {
		glog.Fatal("token must be provided")
	}

	tep, err := trecEvalPath(*trecEvalPathFlag)
	if err != nil {
		glog.Fatal(err)
	}

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

	//////////////////////////////////////////////////////////////////////////////

	// Get qrel data
	qrel, err := getData(qrelURL)
	if err != nil {
		glog.Fatalf("error getting qrel: %v", err)
	}
	glog.V(1).Infof("Successfully acquired qrel data from %s", qrelURL)

	rawQueryRes, err := getData(queryURL)
	if err != nil {
		glog.Fatalf("error getting raw query data: %v", err)
	}
	glog.V(1).Infof("Got query data from %s", queryURL)
	queries, err := getQueries(rawQueryRes)
	if err != nil {
		glog.Fatalf("error parsing queries: %v", err)
	}
	baseline := []BaseData{}
	for _, query := range queries {
		glog.Infof("Running for query %s", query.Title)

		q := retrievalURL.Query()
		q.Add("query", query.Title)
		retrievalURL.RawQuery = q.Encode()

		rawSearchData, err := getData(retrievalURL)
		if err != nil {
			glog.Warning(err)
			continue
		}
		searchData, err := getSearch(rawSearchData)
		if err != nil {
			glog.Warning(err)
			continue
		}
		for rank, sd := range searchData {
			baseline = append(baseline, BaseData{
				QID:       query.Num,
				Q:         "Q0",
				DocID:     sd.ID,
				Rank:      rank + 1,
				Relevance: sd.Score,
				ModelName: "lm",
			})
		}
	}
	base := ""
	for _, b := range baseline {
		base += b.String()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = trecEval(ctx, tep, qrel, base); err != nil {
		glog.Fatalf("Unable to run trec_eval: %v", err)
	}
}

// BaseData which is used for running trec_eval
type BaseData struct {
	QID       string
	Q         string
	DocID     string
	Rank      int
	Relevance float64
	ModelName string
}

func (b BaseData) String() string {
	return fmt.Sprintf(
		"%s\t%s\t%s\t%d\t%f\t%s\n",
		b.QID, b.Q, b.DocID, b.Rank, b.Relevance, b.ModelName,
	)
}

// QueryResult of getting the query file
type QueryResult struct {
	Num   string `xml:"num"`
	Title string `xml:"title"`
}

// queryData can be ignored as it just has the data of the full query including
// the unnecessary top
type queryData struct {
	Top []QueryResult `xml:"top"`
}

func getData(path *url.URL) (string, error) {
	resp, err := http.Get(path.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body), err
}

func getQueries(body string) ([]QueryResult, error) {
	body = fmt.Sprintf("<data>%s</data>", body)
	var qd queryData
	if err := xml.Unmarshal([]byte(body), &qd); err != nil {
		return nil, err
	}
	return qd.Top, nil
}

// SearchResult containing the parsed JSON from the search page.
type SearchResult struct {
	Title   string  `json:"title,omitempty"`
	Snippet string  `json:"snippet,omitempty"`
	ID      string  `json:"id,omitempty"`
	Score   float64 `json:"score,omitempty"`
	URL     string  `json:"url,omitempty"`
}

func getSearch(body string) ([]SearchResult, error) {
	var sr [][]SearchResult
	// sr := make(map[string]interface{})
	if err := json.Unmarshal([]byte(body), &sr); err != nil {
		return nil, err
	}

	res := []SearchResult{}
	for _, l1 := range sr {
		res = append(res, l1...)
	}
	return res, nil
}

func trecEval(ctx context.Context, tep string, qrel string, base string) error {
	qrelTmp, err := ioutil.TempFile("", "qrel")
	if err != nil {
		return fmt.Errorf("can't create qrel tmp file: %v", err)
	}
	// defer os.Remove(qrelTmp.Name())
	if _, err = qrelTmp.WriteString(qrel); err != nil {
		return fmt.Errorf("can't write qrel: %v", err)
	}

	baseTmp, err := ioutil.TempFile("", "base")
	if err != nil {
		return fmt.Errorf("can't create base tmp file: %v", err)
	}
	// defer os.Remove(baseTmp.Name())
	if _, err = baseTmp.WriteString(base); err != nil {
		return fmt.Errorf("can't write base: %v", err)
	}

	cmd := exec.CommandContext(ctx, tep, "-q", qrelTmp.Name(), baseTmp.Name())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running trec_eval: %v", err)
	}
	fmt.Println(out)

	return nil
}

func trecEvalPath(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(path)
	return path, err
}
