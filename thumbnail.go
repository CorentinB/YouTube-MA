package main

import (
	"io"
	"os"
)

func downloadThumbnail(video *Video) error {
	// create the file
	out, err := os.Create(video.Path + video.ID + "_" + video.Title + ".jpg")
	if err != nil {
		return err
	}
	defer out.Close()

	// get the data
	resp, err := getHttpClient().Get(video.Thumbnail)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
