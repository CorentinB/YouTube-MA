package main

import (
	"net/url"
	"strconv"
	"strings"
	"sync"
)

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
