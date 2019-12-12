package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

// AdminRequests is for the /api/admin/requests endpoint
type AdminRequests struct {
	Ok       bool   `json:"ok"`
	Msg      string `json:"msg"`
	Requests []struct {
		ID         int         `json:"ID"`
		VideoID    string      `json:"video_id"`
		RawURL     string      `json:"raw_url"`
		ArchivedAt interface{} `json:"archived_at"`
	} `json:"requests"`
}

// Payload to push IDs
type Payload struct {
	VideoIds []string `json:"video_ids"`
}

func pushIDs(videoIDs []string) error {
	data := new(Payload)
	data.VideoIds = videoIDs
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", "https://youtube.the-eye.eu/api/admin/requests", body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Secret", arguments.Secret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func markIDsArchived(IDs ...string) error {
	data := new(Payload)
	data.VideoIds = IDs
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("PUT", "https://youtube.the-eye.eu/api/admin/requests", body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Secret", arguments.Secret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func getID(secret string, offset, limit int) (IDs []string) {
	URL := "https://youtube.the-eye.eu/api/admin/requests?" +
		"offset=" + strconv.Itoa(offset) +
		"&limit=" + strconv.Itoa(limit)

	spaceClient := http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("X-Secret", secret)

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	requestResponse := AdminRequests{}
	jsonErr := json.Unmarshal(body, &requestResponse)
	if jsonErr != nil {
		log.Println(jsonErr)
		return nil
	}

	for _, response := range requestResponse.Requests {
		IDs = append(IDs, response.VideoID)
	}

	if len(IDs) < 1 {
		log.Println(jsonErr)
		return nil
	}

	return IDs
}
