package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type requests struct {
	queue chan string
	kill  chan bool
	file  *os.File
	wait  *sync.WaitGroup
}

func (r *requests) limiter() {
	var count int
	ticker := time.NewTicker(time.Duration(100))
	go func() {
		for {
			select {
			case <-ticker.C:
				id := <-r.queue
				url := fmt.Sprintf("https://job.mynavi.jp/21/pc/search/corp%s/outline.html", id)
				detail, err := getInfo(url)
				if err != nil {
					log.Printf("%s: failed to get details: %v", id, err)
					continue
				}
				_, err = r.file.WriteString(strconv.Itoa(count) + ": \n\tURL:     " + url + detail + "\n")
				if err != nil {
					log.Printf("%s: failed to write detail to file: %v", id, err)
					continue
				}
				count++
			case <-r.kill:
				r.wait.Done()
				return
			}
		}
	}()
}

func main() {
	f, err := os.Create("emails.txt")
	if err != nil {
		log.Fatalf("failed to create email text file: %v", err)
	}
	r := requests{
		queue: make(chan string),
		kill:  make(chan bool, 1),
		file:  f,
		wait:  &sync.WaitGroup{},
	}
	r.wait.Add(1)
	r.limiter()
	for n := 0; n < 100000; n++ {
		r.queue <- strconv.Itoa(n)
	}
	close(r.kill)
	r.wait.Wait()
	if err := f.Close(); err != nil {
		log.Fatalf("failed to close file: %v", err)
	}
}

func getInfo(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get webpage: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("non-200 http code detected")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse response body: %w", err)
	}

	var detail string
	detail += "\n\tCompany:  " + doc.Find(".group .heading1 .heading1-content .heading1-inner h1").Text()

	detail += "\n\tAddress: \n          "
	detail += "〒" + doc.Find("#corpDescDtoListDescText40").Text() + "\n         "
	detail += doc.Find("#corpDescDtoListDescText50").Text()

	detail += "\n\tPhone:    " + doc.Find("#corpDescDtoListDescText220").Text()

	doc.Find(".noLink").Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			detail += "\n\tField:    " + s.Text()
		} else {
			detail += ", " + s.Text()
		}
	})

	detail += "\n\tCapital:  " + doc.Find("#corpDescDtoListDescText260").Text()
	detail += "\n\tSales:    " + doc.Find("#corpDescDtoListDescText300").Text()
	detail += "\n\tEmployee: " + doc.Find("#corpDescDtoListDescText270").Text()

	detail += "\n\tEmail:    " + doc.Find("#corpDescDtoListDescText130").Text() + "\n\t"

	return detail, nil
}

func formatter(words ...string) string {
	var format string
	format += fmt.Sprintf("%9s%s", words[0] + ":", words[1])
	if len(words) > 2 {
		for _, w := range words[2:] {
			format += fmt.Sprintf("\n%9s%s", " ", w)
		}
	}
	return format
}