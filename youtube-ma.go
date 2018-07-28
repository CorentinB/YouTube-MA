package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/labstack/gommon/color"
	"github.com/spf13/cast"
	"golang.org/x/net/html"
)

// Video structure containing all metadata for the video
type Video struct {
	ID          string
	Title       string
	Annotations string
	Thumbnail   string
	Description string
	Path        string
	RawHTML     string
	STS         float64
	InfoJSON    infoJSON
	playerArgs  map[string]interface{}
	RawFormats  []url.Values
}

// Tracklist structure containing all subtitles tracks for the video
type Tracklist struct {
	Tracks []Track `xml:"track"`
}

// Track structure for data about single subtitle
type Track struct {
	LangCode string `xml:"lang_code,attr"`
	Lang     string `xml:"lang_translated,attr"`
}

// infoJSON structure containing the generated json data
type infoJSON struct {
	ID            string                `json:"id"`
	Uploader      string                `json:"uploader"`
	UploaderID    string                `json:"uploader_id"`
	UploaderURL   string                `json:"uploader_url"`
	UploadDate    string                `json:"upload_date"`
	License       string                `json:"license,omitempty"`
	Creator       string                `json:"creator,omitempty"`
	Title         string                `json:"title"`
	AltTitle      string                `json:"alt_title,omitempty"`
	Thumbnail     string                `json:"thumbnail"`
	Description   string                `json:"description"`
	Category      string                `json:"category"`
	Tags          []string              `json:"tags"`
	Subtitles     map[string][]Subtitle `json:"subtitles"`
	Duration      float64               `json:"duration"`
	AgeLimit      float64               `json:"age_limit"`
	Annotations   string                `json:"annotations"`
	WebpageURL    string                `json:"webpage_url"`
	ViewCount     float64               `json:"view_count"`
	LikeCount     float64               `json:"like_count"`
	DislikeCount  float64               `json:"dislike_count"`
	AverageRating float64               `json:"average_rating"`
	Formats       []Format              `json:"formats"`
	subLock       sync.Mutex
}

type Subtitle struct {
	URL string `json:"url"`
	Ext string `json:"ext"`
}

// Format structure for all different formats informations
type Format struct {
	FormatID     string  `json:"format_id"`
	Ext          string  `json:"ext"`
	URL          string  `json:"url"`
	Height       float64 `json:"height,omitempty"`
	Width        float64 `json:"width,omitempty"`
	FormatNote   string  `json:"format_note"`
	Bitrate      float64 `json:"bitrate"`
	Fps          float64 `json:"fps,omitempty"`
	Format       string  `json:"format"`
	Clen         float64 `json:"clen,omitempty"`
	EOTF         string  `json:"eotf,omitempty"`
	Index        string  `json:"index,omitempty"`
	Init         string  `json:"init,omitempty"`
	Lmt          float64 `json:"lmt,omitempty"`
	Primaries    string  `json:"primaries,omitempty"`
	QualityLabel string  `json:"quality_label,omitempty"`
	Type         string  `json:"type"`
	Size         string  `json:"size,omitempty"`
}

func fetchAnnotations(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	// requesting annotations from YouTube
	resp, err := http.Get("http://www.youtube.com/annotations_invideo?features=1&legacy=1&video_id=" + video.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	defer resp.Body.Close()
	// checking response status code
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			runtime.Goexit()
		}
		annotations := string(bodyBytes)
		video.Annotations = annotations
	} else {
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Red(" Unable to fetch annotations!"))
		video.Annotations = "Unable to fetch annotations."
	}
}

func writeFiles(video *Video) {
	// write annotations
	annotationsFile, errAnno := os.Create(video.Path + video.ID + "_" + video.Title + ".annotations.xml")
	if errAnno != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errAnno)
		runtime.Goexit()
	}
	defer annotationsFile.Close()
	// write description
	descriptionFile, errDescription := os.Create(video.Path + video.ID + "_" + video.Title + ".description")
	if errDescription != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errDescription)
		runtime.Goexit()
	}
	defer descriptionFile.Close()
	// write info json file
	infoFile, errInfo := os.Create(video.Path + video.ID + "_" + video.Title + ".info.json")
	if errInfo != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", errInfo)
		runtime.Goexit()
	}
	defer infoFile.Close()
	fmt.Fprintf(annotationsFile, "%s", video.Annotations)
	fmt.Fprintf(descriptionFile, "%s", video.Description)
	JSON, _ := JSONMarshalIndentNoEscapeHTML(video.InfoJSON, "", "  ")
	fmt.Fprintf(infoFile, "%s", string(JSON))
}

func downloadThumbnail(video *Video) {
	if len(video.Thumbnail) < 1 {
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + ".jpg")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	defer out.Close()
	// get the data
	resp, err := http.Get(video.Thumbnail)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	defer resp.Body.Close()
	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
}

func parseDescription(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	video.Description = ""
	// extract description
	desc := document.Find("#eow-description").Contents()
	desc.Each(func(i int, s *goquery.Selection) {
		switch s.Nodes[0].Type {
		case html.TextNode:
			video.Description += s.Text()
		case html.ElementNode:
			switch s.Nodes[0].Data {
			case "a":
				video.Description += s.Text()
			case "br":
				video.Description += "\n"
			default:
				fmt.Println("Unknown data type", s.Nodes[0].Data)
				panic("unknown data type")
			}
		default:
			fmt.Println("Unknown node type", s.Nodes[0].Type)
			panic("unknown node type")
		}
	})
}

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

func addFormats(video *Video) {
	for _, rawFormat := range video.RawFormats {
		tmpFormat := Format{}
		for k, v := range rawFormat {
			switch k {
			case "bitrate":
				tmpFormat.Bitrate, _ = strconv.ParseFloat(v[0], 64)
			case "clen":
				tmpFormat.Clen, _ = strconv.ParseFloat(v[0], 64)
			case "eotf":
				tmpFormat.EOTF = v[0]
			case "fps":
				tmpFormat.Fps, _ = strconv.ParseFloat(v[0], 64)
			case "index":
				tmpFormat.Index = v[0]
			case "init":
				tmpFormat.Init = v[0]
			case "itag":
				tmpFormat.FormatID = v[0]
				if v[0] == "82" || v[0] == "83" || v[0] == "84" ||
					v[0] == "85" || v[0] == "100" || v[0] == "101" ||
					v[0] == "102" {
					tmpFormat.FormatNote = "3D"
					tmpFormat.Format = tmpFormat.FormatID + " - " + tmpFormat.FormatNote
				} else if v[0] == "91" || v[0] == "92" ||
					v[0] == "93" || v[0] == "94" || v[0] == "95" ||
					v[0] == "96" || v[0] == "132" || v[0] == "151" {
					tmpFormat.FormatNote = "HLS"
					tmpFormat.Format = tmpFormat.FormatID + " - " + tmpFormat.FormatNote
				} else if v[0] == "139" || v[0] == "140" ||
					v[0] == "141" || v[0] == "256" || v[0] == "258" ||
					v[0] == "325" || v[0] == "328" || v[0] == "249" ||
					v[0] == "250" || v[0] == "251" {
					tmpFormat.FormatNote = "DASH audio"
					tmpFormat.Format = tmpFormat.FormatID + " - " + tmpFormat.FormatNote
				} else if v[0] == "133" || v[0] == "134" ||
					v[0] == "135" || v[0] == "136" || v[0] == "137" ||
					v[0] == "138" || v[0] == "160" || v[0] == "212" ||
					v[0] == "264" || v[0] == "298" || v[0] == "299" ||
					v[0] == "266" || v[0] == "167" || v[0] == "168" ||
					v[0] == "169" || v[0] == "170" || v[0] == "218" ||
					v[0] == "219" || v[0] == "278" || v[0] == "242" ||
					v[0] == "245" || v[0] == "244" || v[0] == "243" ||
					v[0] == "246" || v[0] == "247" || v[0] == "248" ||
					v[0] == "271" || v[0] == "272" || v[0] == "302" ||
					v[0] == "303" || v[0] == "308" || v[0] == "313" ||
					v[0] == "315" {
					tmpFormat.FormatNote = "DASH video"
					tmpFormat.Format = tmpFormat.FormatID + " - " + tmpFormat.FormatNote
				} else {
					tmpFormat.Format = tmpFormat.FormatID + " - " + tmpFormat.Type
				}
			case "lmt":
				tmpFormat.Lmt, _ = strconv.ParseFloat(v[0], 64)
			case "primaries":
				tmpFormat.Primaries = v[0]
			case "quality_label":
				tmpFormat.QualityLabel = v[0]
			case "size":
				tmpFormat.Size = v[0]
				sizes := strings.Split(v[0], "x")
				tmpFormat.Width, _ = strconv.ParseFloat(sizes[0], 64)
				tmpFormat.Height, _ = strconv.ParseFloat(sizes[1], 64)
			case "type":
				tmpFormat.Type = v[0]
				s := strings.Index(v[0], "/")
				e := strings.Index(v[0], ";")
				tmpFormat.Ext = v[0][s+1 : e]
			case "url":
				tmpFormat.URL = v[0]
			}
		}
		video.InfoJSON.Formats = append(video.InfoJSON.Formats, tmpFormat)
	}
}

func parseFormats(video *Video, wg *sync.WaitGroup) {
	defer wg.Done()
	if l, ok := video.playerArgs["adaptive_fmts"]; ok {
		formats := strings.Split(l.(string), ",")
		for _, format := range formats {
			args, _ := url.ParseQuery(format)
			video.RawFormats = append(video.RawFormats, args)
		}
	}
	addFormats(video)
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
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(m[0][1]))
	if err != nil {
		panic(err)
	}
	video.InfoJSON.Category = doc.Find("a").Text()
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

func parseThumbnailURL(video *Video, document *goquery.Document, workers *sync.WaitGroup) {
	defer workers.Done()
	// extract thumbnail url
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if name, _ := s.Attr("property"); name == "og:image" {
			thumbnailURL, _ := s.Attr("content")
			video.Thumbnail = strings.Replace(thumbnailURL, "https", "http", -1)
		}
	})
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
		log.Fatalf("Error: %v\n", err)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	// check status, exit if != 200
	if html.StatusCode != 200 {
		log.Fatalf("Status code error for %s: %d %s", video.ID, html.StatusCode, html.Status)
		os.RemoveAll(video.Path)
		runtime.Goexit()
	}
	body, err := ioutil.ReadAll(html.Body)
	// store raw html in video struct
	video.RawHTML = string(body)
	// start goquery in the page
	document, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Error: %v\n", err)
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

func genPath(video *Video) {
	firstChar := video.ID[:1]
	video.Path = firstChar + "/" + video.ID + "/"
	// create directory if it doesnt exist
	if _, err := os.Stat(video.Path); os.IsNotExist(err) {
		err = os.MkdirAll(video.Path, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			runtime.Goexit()
		}
	}
}

func addSubToJSON(video *Video, langCode string) {
	urlXML := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID
	urlTTML := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID + "&fmt=ttml&name="
	urlVTT := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID + "&fmt=vtt&name="
	video.InfoJSON.subLock.Lock()
	video.InfoJSON.Subtitles[langCode] = append(video.InfoJSON.Subtitles[langCode], Subtitle{urlXML, "xml"}, Subtitle{urlTTML, "ttml"}, Subtitle{urlVTT, "vtt"})
	video.InfoJSON.subLock.Unlock()
}

func downloadSub(video *Video, langCode string, lang string, wg *sync.WaitGroup) {
	defer wg.Done()
	addSubToJSON(video, langCode)
	color.Green("Downloading " + lang + " subtitle.." + "[" + langCode + "]")
	// generate subtitle URL
	url := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID
	color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Green(" Downloading ") + color.Yellow(lang) + color.Green(" subtitle.."))
	// get the data
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	defer resp.Body.Close()
	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + "." + langCode + ".xml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
	defer out.Close()
	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		runtime.Goexit()
	}
}

func fetchSubsList(video *Video) {
	var wg sync.WaitGroup
	// request subtitles list
	res, err := http.Get("http://video.google.com/timedtext?hl=en&type=list&v=" + video.ID)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	// defer it!
	defer res.Body.Close()
	// check status, exit if != 200
	if res.StatusCode != 200 {
		log.Fatalf("Status code error while fetching subtitles for %s: %d %s", video.ID, res.StatusCode, res.Status)
	}
	// reading tracks list as a byte array
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalf("Error: %v\n", err)
	}
	var tracks Tracklist
	xml.Unmarshal(data, &tracks)
	wg.Add(len(tracks.Tracks))
	for _, track := range tracks.Tracks {
		go downloadSub(video, track.LangCode, track.Lang, &wg)
	}
	wg.Wait()
}

func processSingleID(ID string, worker *sync.WaitGroup) {
	defer worker.Done()
	var wg sync.WaitGroup
	video := new(Video)
	video.ID = ID
	video.InfoJSON.Subtitles = make(map[string][]Subtitle)
	video.playerArgs = make(map[string]interface{})
	logInfo("-", video, "Archiving started.")
	wg.Add(2)
	logInfo("~", video, "Fetching annotations..")
	go fetchAnnotations(video, &wg)
	logInfo("~", video, "Parsing infos, description, title and thumbnail..")
	go parseHTML(video, &wg)
	wg.Wait()
	genPath(video)
	logInfo("~", video, "Fetching subtitles..")
	fetchSubsList(video)
	logInfo("~", video, "Writing informations locally..")
	writeFiles(video)
	logInfo("~", video, "Downloading thumbnail..")
	downloadThumbnail(video)
	logInfo("✓", video, "Archiving complete!")
}

func logInfo(info string, video *Video, log string) {
	if info == "-" || info == "✓" {
		color.Println(color.Yellow("[") + color.Green(info) + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("] ") + color.Green(log))
	} else {
		color.Println(color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("] ") + color.Green(log))
	}
}

func processList(path string, worker *sync.WaitGroup) {
	defer worker.Done()
	var count int
	// start workers group
	var wg sync.WaitGroup
	// open file
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	// scan the list line by line
	scanner := bufio.NewScanner(file)
	// scan the list line by line
	for scanner.Scan() {
		count++
		wg.Add(1)
		go processSingleID(scanner.Text(), &wg)
	}
	// log if error
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	wg.Wait()
}

func argumentParsing(args []string) {
	// start workers group
	var wg sync.WaitGroup
	wg.Add(1)
	if len(args) > 1 {
		color.Red("Usage: ./youtube-ma [ID or list of IDs]")
		os.Exit(1)
	}
	if _, err := os.Stat(args[0]); err == nil {
		go processList(args[0], &wg)
	} else {
		go processSingleID(args[0], &wg)
	}
	wg.Wait()
}

// JSONMarshalIndentNoEscapeHTML allow proper json formatting
func JSONMarshalIndentNoEscapeHTML(i interface{}, prefix string, indent string) ([]byte, error) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent(prefix, indent)
	err := encoder.Encode(i)
	return buf.Bytes(), err
}

func main() {
	start := time.Now()
	argumentParsing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
