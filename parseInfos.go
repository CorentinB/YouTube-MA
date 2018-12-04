package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cast"
)

func parsePlayerArgs(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()

	const pre = "var ytplayer = ytplayer || {};ytplayer.config = "
	const post = ";ytplayer.load "

	// extract ytplayer.config
	script := document.Find("div#player").Find("script")
	script.Each(func(i int, s *goquery.Selection) {
		js := s.Text()
		if strings.HasPrefix(js, pre) {
			i := strings.Index(js, post)
			if i == -1 {
				return
			}
			strCfg := js[len(pre):i]
			var cfg struct {
				Args map[string]interface{}
			}
			err := json.Unmarshal([]byte(strCfg), &cfg)
			if err != nil {
				fmt.Println(err)
				return
			}
			video.playerArgs = cfg.Args
		}
	})
}

func parseUploaderInfo(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("a").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("class"); name == "yt-uix-sessionlink       spf-link " {
			uploader := s.Text()
			if strings.Contains(uploader, "https://www.youtube.com/watch?v=") == false && strings.Contains(uploader, "https://youtu.be") == false {
				video.InfoJSON.Uploader = uploader
			}
			uploaderID, _ := s.Attr("href")
			if strings.Contains(uploaderID, "/channel/") == true {
				video.InfoJSON.UploaderID = uploaderID[9:len(uploaderID)]
				video.InfoJSON.UploaderURL = "https://www.youtube.com" + uploaderID
			}
		}
	})
}

func parseLikeDislike(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()

	document.Find("button.like-button-renderer-like-button").Each(func(i int, s *goquery.Selection) {
		likeCount := strings.TrimSpace(s.Text())
		video.InfoJSON.LikeCount = cast.ToFloat64(strings.Replace(likeCount, ",", "", -1))

	})
	document.Find("button.like-button-renderer-dislike-button").Each(func(i int, s *goquery.Selection) {
		dislikeCount := strings.TrimSpace(s.Text())
		video.InfoJSON.DislikeCount = cast.ToFloat64(strings.Replace(dislikeCount, ",", "", -1))
	})
}

func parseDatePublished(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("itemprop"); name == "datePublished" {
			date, _ := s.Attr("content")
			date = strings.Replace(date, "-", "", -1)
			video.InfoJSON.UploadDate = date
		}
	})
}

func parseViewCount(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("div").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("class"); name == "watch-view-count" {
			viewCount := s.Text()
			reg, err := regexp.Compile("[^0-9]+")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				runtime.Goexit()
			}
			video.InfoJSON.ViewCount = cast.ToFloat64(reg.ReplaceAllString(viewCount, ""))
		}
	})
}

func parseAverageRating(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	if l, ok := video.playerArgs["avg_rating"]; ok {
		dur, _ := strconv.ParseFloat(l.(string), 64)
		video.InfoJSON.AverageRating = dur
	}
}

func parseTags(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("property"); name == "og:video:tag" {
			tag, _ := s.Attr("content")
			video.InfoJSON.Tags = append(video.InfoJSON.Tags, tag)
		}
	})
}

func parseCategory(video *Video, document *goquery.Document, wg *sync.WaitGroup) {
	defer wg.Done()
	pattern, _ := regexp.Compile(`(?s)<h4[^>]*>\s*Category\s*</h4>\s*<ul[^>]*>(.*?)</ul>`)
	m := pattern.FindAllStringSubmatch(video.RawHTML, -1)
	if len(m) == 1 && len(m[0]) == 2 {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[0][1]))
		if err != nil {
			panic(err)
		}
		video.InfoJSON.Category = doc.Find("a").Text()
	}
}

func parseAgeLimit(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	pattern, _ := regexp.Compile(`(?s)<h4[^>]*>\s*Notice\s*</h4>\s*<ul[^>]*>(.*?)</ul>`)
	m := pattern.FindAllStringSubmatch(video.RawHTML, -1)
	if len(m) == 1 && len(m[0]) == 2 {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[0][1]))
		if err != nil {
			panic(err)
		}
		isLicense := doc.Find("a").Text()
		if strings.Contains(isLicense, "Age-restricted video (based on Community Guidelines)") == true {
			video.InfoJSON.AgeLimit = 18
		}
	}
}
func parseDuration(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	if l, ok := video.playerArgs["length_seconds"]; ok {
		dur, _ := strconv.ParseFloat(l.(string), 64)
		video.InfoJSON.Duration = dur
	}
}

func parseLicense(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	pattern, _ := regexp.Compile(`<h4[^>]+class="title"[^>]*>\s*License\s*</h4>\s*<ul[^>]*>\s*<li>(.+?)</li`)
	m := pattern.FindAllStringSubmatch(video.RawHTML, -1)
	if len(m) == 1 && len(m[0]) == 2 {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[0][1]))
		if err != nil {
			panic(err)
		}
		video.InfoJSON.License = doc.Find("a").Text()
	}
}

func parseVariousInfo(video *Video, document *goquery.Document) {
	var wg sync.WaitGroup
	wg.Add(11)
	video.InfoJSON.ID = video.ID
	video.InfoJSON.Description = video.Description
	video.InfoJSON.Annotations = video.Annotations
	video.InfoJSON.Thumbnail = video.Thumbnail
	video.InfoJSON.WebpageURL = "https://www.youtube.com/watch?v=" + video.ID
	go parseUploaderInfo(video, document, &wg)
	go parseLikeDislike(video, document, &wg)
	go parseDatePublished(video, document, &wg)
	go parseLicense(video, &wg)
	go parseViewCount(video, document, &wg)
	go parseAverageRating(video, &wg)
	go parseFormats(video, &wg)
	go parseTags(video, document, &wg)
	go parseCategory(video, document, &wg)
	go parseAgeLimit(video, &wg)
	go parseDuration(video, &wg)
	wg.Wait()
}

func parseTitle(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract title
	title := strings.TrimSpace(document.Find("#eow-title").Text())
	video.InfoJSON.Title = title
	video.Title = strings.Replace(title, " ", "_", -1)
	video.Title = strings.Replace(video.Title, "/", "_", -1)
}

func parseHTML(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	var workers sync.WaitGroup
	workers.Add(4)
	// request video html page
	html, err := http.Get("http://youtube.com/watch?v=" + video.ID + "&gl=US&hl=en&has_verified=1&bpctr=9999999999")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	// check status, exit if != 200
	if html.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Status code error for %s: %d %s", video.ID, html.StatusCode, html.Status)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	body, err := ioutil.ReadAll(html.Body)
	// store raw html in video struct
	video.RawHTML = string(body)
	// start goquery in the page
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	go parseTitle(video, document, &workers)
	go parseDescription(video, document, &workers)
	go parseThumbnailURL(video, document, &workers)
	go parsePlayerArgs(video, document, &workers)
	workers.Wait()
	parseVariousInfo(video, document)
	defer html.Body.Close()
}
