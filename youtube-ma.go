package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/savaki/jq"
)

type Video struct {
	Title       string
	Author      string
	Annotations string
	ThumbURL    string
}

func fetchingBasic(id string) {
	// Declaring jq operations
	getTitle, _ := jq.Parse(".title")
	getAuthor, _ := jq.Parse(".author_name")
	getThumb, _ := jq.Parse(".thumbnail_url")
	// Requesting data from oembed (allow getting title, author, thumbnail url)
	resp, err := http.Get("https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=" + id + "&format=json")
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()
	// Checking response status code
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			// handle err
		}
		bodyString := string(bodyBytes)
		output := []byte(bodyString)
		// Parsing data
		title, _ := getTitle.Apply(output)
		authorName, _ := getAuthor.Apply(output)
		thumbnailUrl, _ := getThumb.Apply(output)
		fmt.Println("\nTitle: " + string(title))
		fmt.Println("Author: " + string(authorName))
		fmt.Println("Thumbnail: " + string(thumbnailUrl))
	}
	//author=$(curl -s "https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=${id}&format=json" | jq -r '.author_name')
	//thumb_url=$(curl -s "https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=${id}&format=json" | jq -r '.thumbnail_url')
	//annotations=$(curl -s "https://www.youtube.com/annotations_invideo?features=1&legacy=1&video_id=${id}")

}

func main() {
	args := os.Args[1:]
	id := args[0]
	key := args[1]
	fmt.Println("ID: " + id)
	fmt.Println("API key: " + key)
	fetchingBasic(id)

}
