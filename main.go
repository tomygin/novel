package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/axgle/mahonia"
	"github.com/gocolly/colly"
	"github.com/imroc/req/v3"
)

var Client = req.C()

func cover(src string) string {
	return mahonia.NewDecoder("gbk").ConvertString(src)
}

func main() {

	start := time.Now()
	defer fmt.Println(time.Since(start))
	var count int

	c0 := colly.NewCollector()

	// 首页层面
	// https://www.wenku8.net/novel/2/2353/index.htm
	c1 := colly.NewCollector(
	// colly.Async(false),
	)

	//小说页面
	//https://www.wenku8.net/novel/2/2353/86807.htm
	c2 := colly.NewCollector(
	// colly.Async(true),
	)
	// extensions.RandomUserAgent(c2)

	//小说一级首页
	//用于下载一本书
	c1.OnHTML("td.ccss", func(h *colly.HTMLElement) {
		h.ForEach("a[href]", func(i int, t *colly.HTMLElement) {
			// name := t.Text
			// name = cover(name)
			url := t.Attr("href")
			// fmt.Println(name, url)
			url = t.Request.AbsoluteURL(url)
			c2.Visit(url)
		})
	})

	c1.OnError(func(r *colly.Response, err error) {
		fmt.Println(r.Request.URL, err)
	})

	//小说一个章节
	c2.OnHTML("body", func(c *colly.HTMLElement) {
		var title, content string
		title = c.DOM.Find("div#title").Text()
		content = c.DOM.Find("div#content").Text()

		title = cover(title)
		content = cover(content)

		content = strings.ReplaceAll(content, "聽", "")
		content = strings.Replace(content, "本文来自 轻小说文库(http://www.wenku8.com)", "", 1)
		content = strings.Replace(content, "最新最全的日本动漫轻小说 轻小说文库(http://www.wenku8.com) 为你一网打尽！", "", 1)

		//清理无效章节
		if strings.Contains(content, "因版权问题，文库不再提供该小说的阅读") {
			return
		}

		path := c.DOM.Find("#linkleft > a:nth-child(3)").Text()
		path = cover(path)

		//创建小说文件目录
		if _, err := os.Stat(path); err != nil {
			os.Mkdir(path, 0777)
			os.Chmod(path, 0777)
		}

		//如果这个章节有图片就把图片也下载了
		c.DOM.Find("a[href]").Each(func(_ int, i *goquery.Selection) {
			link, _ := i.Attr("href")
			tmp := strings.Split(link, "/")
			if len(tmp) <= 1 {
				return
			}
			img := tmp[len(tmp)-1]
			if strings.Contains(img, `.jpg`) {
				_, err := Client.R().SetOutputFile(filepath.Join(path, img)).Get(link)
				if err != nil {
					fmt.Println("[下载错误]", err)
				}

			}
		})

		//保存文本章节到小说目录
		save(filepath.Join(path, title), content)

		//保存下载数量
		count++

	})

	c0.OnRequest(func(r *colly.Request) {
		fmt.Println("[开始访问]", r.URL)
	})

	c0.OnHTML("#content > div:nth-child(1) > div:nth-child(6) > div:nth-child(1) > span:nth-child(1) > fieldset:nth-child(1) > div:nth-child(2) > a:nth-child(1)", func(h *colly.HTMLElement) {
		url := h.Attr("href")
		tmp := cover(h.Text)
		if url == "" || tmp != "小说目录" {
			return
		}
		c1.Visit(url)

	})

	c1.OnScraped(func(r *colly.Response) {
		fmt.Println("[获取完成]", r.Request.URL)
	})

	fast := os.Args[1]
	if fast == "y" {
		c2.Async = true
	}

	if len(os.Args) == 3 {
		httpurl := os.Args[2]
		// https://www.wenku8.net/novel/2/2353/index.htm
		if httpurl == "test" {
			httpurl = "https://www.wenku8.net/novel/2/2353/index.htm"
		}
		c1.Visit(httpurl)

	} else if len(os.Args) == 4 {
		begain, end := os.Args[2], os.Args[3]
		for begain != end {
			// https: //www.wenku8.net/book/1.htm

			c0.Visit("https://www.wenku8.net/book/" + begain + ".htm")
			i, err := strconv.Atoi(begain)
			if err != nil {
				panic(err)
			}
			i++
			begain = strconv.Itoa(i)
		}
	} else {
		fmt.Println(os.Args)
	}

	c1.Wait()
	c2.Wait()
	fmt.Println("[共计下载]", count)
}

func save(name, content string) {
	name += `.txt`
	// name = strings.TrimSpace(name)
	f, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	f.WriteString(content)
	f.Close()
	fmt.Println("[保存完毕]", name)
}
