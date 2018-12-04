package main

import (
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/labstack/gommon/color"
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

// Subtitle struct hold subtitle data
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

func main() {
	start := time.Now()
	argumentParsing(os.Args[1:])
	color.Println(color.Cyan("Done in ") + color.Yellow(time.Since(start)) + color.Cyan("!"))
}
