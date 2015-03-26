package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"log"
	"strconv"
	"time"
)

type Crawler interface {
	Crawl(url string) Urlset
}

type Urlset []Url

type Url struct {
	Loc        string
	Lastmod    time.Time
	Changefreq string
	Priority   float32
}

func (u Url) String() string {
	return fmt.Sprintf("loc: %s\nlastmod: %s\nchangefreq: %s\nprio: %f\n",
		u.Loc, u.Lastmod, u.Changefreq, u.Priority)
}

type MaudCrawler struct {
	url string
}

func (crawler MaudCrawler) Crawl() Urlset {
	// crawl home
	log.Println("Crawling " + crawler.url + "...")
	doc, err := goquery.NewDocument(crawler.url)
	if err != nil {
		log.Fatal(err)
	}
	threads := doc.Find("article.thread-item")
	log.Printf("found %d threads in homepage.\n", len(threads.Nodes))
	// get latest updated thread
	latest := threads.First()
	udatespan := latest.Find("span.date").First().Nodes[0].FirstChild
	log.Printf("first child: %s\n", udatespan.Data)
	udate, err := time.Parse("02/01/2006 15:04", udatespan.Data)
	if err != nil {
		log.Fatal(err)
	}

	urlset := make(Urlset, 0)
	// first insert root url
	urlset = append(urlset, Url{
		Loc:        crawler.url,
		Lastmod:    udate,
		Changefreq: "hourly",
		Priority:   1.0,
	})

	// then crawl /threads/
	urlset = append(urlset, crawler.subCrawl("threads", "article.thread-item", "daily", 0.6)...)
	// and /tags/
	urlset = append(urlset, crawler.subCrawl("tags", "article.tag-item", "daily", 0.5)...)
	// and /stiki/
	urlset = append(urlset, crawler.subCrawl("stiki", "article.thread-item", "monthly", 0.7)...)
	return urlset
}

func (crawler MaudCrawler) subCrawl(suburl, selector, changefreq string, prio float32) Urlset {
	log.Println("Crawling " + crawler.url + suburl + "...")
	urlset := make(Urlset, 0)
	doc, err := goquery.NewDocument(crawler.url + suburl)
	if err != nil {
		log.Fatal(err)
	}
	curpage := 1
	for {
		more := false
		spansel := doc.Find("div.pages").First()
		if spansel != nil && len(spansel.Nodes) > 0 {
			pagespan := spansel.Nodes[0]
			for _, attr := range pagespan.Attr {
				if attr.Key == "data-more" {
					if attr.Val == "yes" {
						more = true
						break
					}
				}
			}
		}
		if curpage > 1 {
			log.Printf("--> Crawling %s ...\n", crawler.url+suburl+"/page/"+strconv.Itoa(curpage))
			doc, err = goquery.NewDocument(crawler.url + suburl + "/page/" + strconv.Itoa(curpage))
			if err != nil {
				log.Fatal(err)
			}
		}
		doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
			anchor := s.Find("a").First().Nodes[0]
			var href string
			for _, attr := range anchor.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}
			udatespan := s.Find("span.date").First().Nodes[0].FirstChild
			udate, err := time.Parse("02/01/2006 15:04", udatespan.Data)
			if err != nil {
				log.Fatal(err)
			}
			urlset = append(urlset, Url{
				Loc:        href,
				Lastmod:    udate,
				Changefreq: changefreq,
				Priority:   prio,
			})
		})
		if !more {
			break
		}
		curpage++
	}
	return urlset
}

func emitSitemap(urlset Urlset) {
	header :=
		`<?xml version="1.0" encoding="UTF-8"?>
    <urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
`
	fmt.Print(header)
	for _, url := range urlset {
		data :=
			`        <url>
            <loc>` + url.Loc + `</loc>
            <lastmod>` + url.Lastmod.Format("2006-01-02T15:04:05-07:00") + `</lastmod>
            <changefreq>` + url.Changefreq + `</changefreq>
            <priority>` + fmt.Sprintf("%.2f", url.Priority) + `</priority>
        </url>`
		fmt.Println(data)
	}
	fmt.Println("    </urlset>\n</xml>")
}

func main() {
	crawler := MaudCrawler{url: "https://crunchy.rocks/"}
	urlset := crawler.Crawl()
	emitSitemap(urlset)
}
