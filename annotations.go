package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"sync"

	"github.com/labstack/gommon/color"
)

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
