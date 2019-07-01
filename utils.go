package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/labstack/gommon/color"
)

var tildePre = color.Yellow("[") + color.Magenta("~") + color.Yellow("]") + color.Yellow("[")
var checkPre = color.Yellow("[") + color.Green("✓") + color.Yellow("]") + color.Yellow("[")
var dashPre = color.Yellow("[") + color.Green("-") + color.Yellow("]") + color.Yellow("[")

func logInfo(info string, video *Video, log string) {
	if info == "-" {
		color.Println(dashPre + color.Cyan(video.ID) + color.Yellow("] ") + color.Green(log))
	} else if info == "✓" {
		color.Println(checkPre + color.Cyan(video.ID) + color.Yellow("] ") + color.Green(log))
	} else {
		color.Println(tildePre + color.Cyan(video.ID) + color.Yellow("] ") + color.Green(log))
	}
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

func genPath(video *Video) {
	// create directory if it doesnt exist
	if _, err := os.Stat(video.Path); os.IsNotExist(err) {
		err = os.MkdirAll(video.Path, 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			runtime.Goexit()
		}
	}
}

func checkFiles(video *Video) {
	firstLayer := video.ID[:1]
	secondLayer := video.ID[:3]
	video.Path = firstLayer + "/" + secondLayer + "/" + video.ID + "/"
	files, err := ioutil.ReadDir(video.Path)
	if err == nil && len(files) >= 4 {
		color.Println(color.Yellow("[") + color.Red("!") + color.Yellow("]") + color.Yellow("[") + color.Cyan(video.ID) + color.Yellow("]") + color.Red(" This video has already been archived!"))
		runtime.Goexit()
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
