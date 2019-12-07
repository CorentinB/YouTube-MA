package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/spf13/cast"
)

func parsePlayerArgs(video *Video, document *goquery.Document) error {
	const pre = "var ytplayer = ytplayer || {};ytplayer.config = "
	const post = ";ytplayer.load "

	// extract ytplayer.config
	script := document.Find("div#player").Find("script")
	script.Each(func(i int, s *goquery.Selection) {
		js := s.Text()
		if strings.HasPrefix(js, pre) {
			i := strings.Index(js, post)
			if i == -1 {
				video.playerArgs = nil
				return
			}
			strCfg := js[len(pre):i]
			var cfg struct {
				Args map[string]interface{}
			}
			err := json.Unmarshal([]byte(strCfg), &cfg)
			if err != nil {
				video.playerArgs = nil
				return
			}
			video.playerArgs = cfg.Args
		}
	})

	if reflect.ValueOf(video.playerArgs).IsNil() {
		return errors.New("error when parsing player arguments, cancelation")
	}
	return nil
}

func parseUploaderInfo(video *Video, document *goquery.Document) error {
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

	if video.InfoJSON.Uploader == "" ||
		video.InfoJSON.UploaderID == "" ||
		video.InfoJSON.UploaderURL == "" {
		return errors.New("error when parsing uploader informations, cancelation")
	}

	return nil
}

func parseLikeDislike(video *Video, document *goquery.Document) error {
	video.InfoJSON.LikeCount = -1
	video.InfoJSON.DislikeCount = -1

	document.Find("button.like-button-renderer-like-button").Each(func(i int, s *goquery.Selection) {
		likeCount := strings.TrimSpace(s.Text())
		video.InfoJSON.LikeCount = cast.ToFloat64(strings.Replace(likeCount, ",", "", -1))

	})
	document.Find("button.like-button-renderer-dislike-button").Each(func(i int, s *goquery.Selection) {
		dislikeCount := strings.TrimSpace(s.Text())
		video.InfoJSON.DislikeCount = cast.ToFloat64(strings.Replace(dislikeCount, ",", "", -1))
	})

	if video.InfoJSON.LikeCount == -1 || video.InfoJSON.LikeCount == -1 {
		return errors.New("error when parsing like/dislike counter, cancelation")
	}
	return nil
}

func parseDatePublished(video *Video, document *goquery.Document) error {
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("itemprop"); name == "datePublished" {
			date, _ := s.Attr("content")
			date = strings.Replace(date, "-", "", -1)
			video.InfoJSON.UploadDate = date
		}
	})

	if video.InfoJSON.UploadDate == "" {
		return errors.New("error when parsing publication date, cancelation")
	}
	return nil
}

func parseViewCount(video *Video, document *goquery.Document) error {
	video.InfoJSON.ViewCount = -1

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

	if video.InfoJSON.ViewCount == -1 {
		errors.New("error when parsing views, cancelation")
	}
	return nil
}

func parseAverageRating(video *Video) error {
	video.InfoJSON.AverageRating = -1

	if l, ok := video.playerArgs["avg_rating"]; ok {
		dur, _ := strconv.ParseFloat(l.(string), 64)
		video.InfoJSON.AverageRating = dur
	}

	if video.InfoJSON.AverageRating == -1 {
		return errors.New("error when parsing average rating, cancelation")
	}
	return nil
}

func parseTags(video *Video, document *goquery.Document) {
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("property"); name == "og:video:tag" {
			tag, _ := s.Attr("content")
			video.InfoJSON.Tags = append(video.InfoJSON.Tags, tag)
		}
	})
}

func parseCategory(video *Video, document *goquery.Document) error {
	pattern, _ := regexp.Compile(`(?s)<h4[^>]*>\s*Category\s*</h4>\s*<ul[^>]*>(.*?)</ul>`)
	m := pattern.FindAllStringSubmatch(video.RawHTML, -1)
	if len(m) == 1 && len(m[0]) == 2 {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[0][1]))
		if err != nil {
			panic(err)
		}
		video.InfoJSON.Category = doc.Find("a").Text()
	}

	if video.InfoJSON.Category == "" {
		return errors.New("error when parsing category, cancelation")
	}
	return nil
}

func parseAgeLimit(video *Video) error {
	pattern, _ := regexp.Compile(`(?s)<h4[^>]*>\s*Notice\s*</h4>\s*<ul[^>]*>(.*?)</ul>`)
	m := pattern.FindAllStringSubmatch(video.RawHTML, -1)
	if len(m) == 1 && len(m[0]) == 2 {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[0][1]))
		if err != nil {
			return err
		}
		isLicense := doc.Find("a").Text()
		if strings.Contains(isLicense, "Age-restricted video (based on Community Guidelines)") == true {
			video.InfoJSON.AgeLimit = 18
		}
	}
	return nil
}

func parseDuration(video *Video) error {
	video.InfoJSON.Duration = -1

	if l, ok := video.playerArgs["length_seconds"]; ok {
		dur, _ := strconv.ParseFloat(l.(string), 64)
		video.InfoJSON.Duration = dur
	}

	if video.InfoJSON.Duration == -1 {
		return errors.New("error when parsing video's duration, cancelation")
	}
	return nil
}

func parseLicense(video *Video) {
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

func parseVariousInfo(video *Video, document *goquery.Document) (err error) {
	video.InfoJSON.ID = video.ID
	video.InfoJSON.Description = video.Description
	video.InfoJSON.Thumbnail = video.Thumbnail
	video.InfoJSON.WebpageURL = "https://www.youtube.com/watch?v=" + video.ID

	err = parseUploaderInfo(video, document)
	if err != nil {
		return err
	}

	err = parseLikeDislike(video, document)
	if err != nil {
		return err
	}

	err = parseDatePublished(video, document)
	if err != nil {
		return err
	}

	parseLicense(video)

	err = parseViewCount(video, document)
	if err != nil {
		return err
	}

	err = parseAverageRating(video)
	if err != nil {
		return err
	}

	err = parseFormats(video)
	if err != nil {
		return err
	}

	parseTags(video, document)

	err = parseCategory(video, document)
	if err != nil {
		return err
	}

	err = parseAgeLimit(video)
	if err != nil {
		return err
	}

	err = parseDuration(video)
	if err != nil {
		return err
	}

	return nil
}

func parseTitle(video *Video, document *goquery.Document) error {
	title := strings.TrimSpace(document.Find("#eow-title").Text())
	if len(title) < 1 {
		return errors.New("title of the video is empty, cancelation")
	}

	video.InfoJSON.Title = title
	video.Title = strings.Replace(title, " ", "_", -1)
	video.Title = strings.Replace(video.Title, "/", "_", -1)
	return nil
}

func parseHTML(video *Video) error {
	// request video html page
	html, err := http.Get("https://youtube.com/watch?v=" + video.ID + "&gl=US&hl=en&has_verified=1&bpctr=9999999999")
	if err != nil {
		return err
	}
	defer html.Body.Close()

	// check status, exit if != 200
	if html.StatusCode != 200 {
		return errors.New("status code != 200 when requesting the video page")
	}
	body, err := ioutil.ReadAll(html.Body)

	// store raw html in video struct
	video.RawHTML = string(body)

	// start goquery in the page
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return err
	}

	err = parseTitle(video, document)
	if err != nil {
		return err
	}

	err = parseDescription(video, document)
	if err != nil {
		return err
	}

	err = parsePlayerArgs(video, document)
	if err != nil {
		return err
	}

	err = parseVariousInfo(video, document)
	if err != nil {
		return err
	}

	return nil
}
