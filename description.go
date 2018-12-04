package main

import (
	"fmt"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

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
