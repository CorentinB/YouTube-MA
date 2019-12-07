package main

import (
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

func addSubToJSON(video *Video, langCode string) {
	urlXML := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID
	urlTTML := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID + "&fmt=ttml&name="
	urlVTT := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID + "&fmt=vtt&name="
	video.InfoJSON.subLock.Lock()
	video.InfoJSON.Subtitles[langCode] = append(video.InfoJSON.Subtitles[langCode], Subtitle{urlXML, "xml"}, Subtitle{urlTTML, "ttml"}, Subtitle{urlVTT, "vtt"})
	video.InfoJSON.subLock.Unlock()
}

func downloadSub(video *Video, langCode string, lang string) error {
	addSubToJSON(video, langCode)

	// generate subtitle URL
	url := "http://www.youtube.com/api/timedtext?lang=" + langCode + "&v=" + video.ID

	// get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + "." + langCode + ".xml")
	if err != nil {
		return err
	}
	defer out.Close()

	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func fetchSubs(video *Video) error {
	var tracks Tracklist

	// request subtitles list
	res, err := http.Get("http://video.google.com/timedtext?hl=en&type=list&v=" + video.ID)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// check status, exit if != 200
	if res.StatusCode != 200 {
		return errors.New("status code of subtitles list != 200, cancelation")
	}

	// reading tracks list as a byte array
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// download the subtitles
	xml.Unmarshal(data, &tracks)
	for _, track := range tracks.Tracks {
		err = downloadSub(video, track.LangCode, track.Lang)
		if err != nil {
			return err
		}
	}

	return nil
}
