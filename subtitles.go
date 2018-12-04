package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sync"

	"github.com/labstack/gommon/color"
)

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
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Red(" Unable to fetch subtitles!"))
		runtime.Goexit()
	}
	// defer it!
	defer res.Body.Close()
	// check status, exit if != 200
	if res.StatusCode != 200 {
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Red(" Unable to fetch subtitles!"))
		runtime.Goexit()
	}
	// reading tracks list as a byte array
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Red(" Unable to fetch subtitles!"))
		runtime.Goexit()
	}
	var tracks Tracklist
	xml.Unmarshal(data, &tracks)
	wg.Add(len(tracks.Tracks))
	for _, track := range tracks.Tracks {
		go downloadSub(video, track.LangCode, track.Lang, &wg)
	}
	wg.Wait()
}
