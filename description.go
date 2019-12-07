package main

import (
	"errors"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func parseDescription(video *Video, document *goquery.Document) error {
	video.Description = ""
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
				video.Description = "unknown data type when parsing description, cancelation"
			}
		default:
			video.Description = "unknown data type when parsing description, cancelation"
		}
	})

	if video.Description == "unknown data type when parsing description, cancelation" {
		return errors.New("unknown data type when parsing description, cancelation")
	}
	return nil
}
