package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const ServerHost = "https://openapi.vito.ai/v1/transcribe"
const ClientId = "YOUR_CLIEND_ID"
const ClientSecret = "YOUR_CLIENT_SECRET"

type SttResult struct {
	Id      string      `json:"id"`
	Status  string      `json:"status"`
	Results interface{} `json:"results"`
}

func MakeReq(file, token string) (*http.Request, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	audiofile, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer audiofile.Close()

	fw, err := writer.CreateFormFile("file", audiofile.Name())
	if err != nil {
		log.Fatal(err)
	}

	if _, err := io.Copy(fw, audiofile); err != nil {
		log.Fatal(err)
	}

	fw, err = writer.CreateFormField("config")
	if err != nil {
		log.Fatal(err)
	}

	config := map[string]interface{}{
		"model_name": "sommers",
		"use_itn":    true,
	}

	j, err := json.Marshal(config)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := fw.Write(j); err != nil {
		log.Fatal(err)
	}

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, ServerHost, &buf)

	req.Header.Add("Content-Type", writer.FormDataContentType())
	req.Header.Add("Authorization", fmt.Sprintf("%s %v", "bearer", token))
	if err != nil {
		return nil, err
	}

	return req, err
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <AUDIOFILE>\n", filepath.Base(os.Args[0]))
		fmt.Fprintf(os.Stderr, "<AUDIOFILE> must be a path to a local audio file. Audio file must be a 16-bit signed little-endian encoded with a sample rate of 16000.\n")
	}
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("Please pass path to your local audio file as a command line argument")
	}

	filePath := flag.Arg(0)

	//JWT 토큰을 받기 위한 과정
	data := map[string][]string{
		"client_id":     {ClientId},
		"client_secret": {ClientSecret},
	}

	resp, _ := http.PostForm("https://openapi.vito.ai/v1/authenticate", data)
	if resp.StatusCode != 200 {
		panic("Failed to authenticate")
	}

	// 결과값 중에서 access_token 값만을 result 에 할당
	resByte, _ := io.ReadAll(resp.Body)
	var result struct {
		Token string `json:"access_token"`
	}
	json.Unmarshal(resByte, &result)

	req, _ := MakeReq(filePath, result.Token)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error resposne")
	}
	defer response.Body.Close()

	resByte, _ = io.ReadAll(response.Body)

	var resultId struct {
		Id string `json:"id"`
	}

	json.Unmarshal(resByte, &resultId)

	// Use Polling to get response
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(5 * time.Second)
		var buf bytes.Buffer
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%v", ServerHost, resultId.Id), &buf)
		if err != nil {
			return
		}
		req.Header.Set("Authorization", fmt.Sprintf("%s %v", "bearer", result.Token))
		client := &http.Client{}
		res, _ := client.Do(req)

		if res.StatusCode != 200 {
			continue
		}

		resByte, _ := io.ReadAll(res.Body)

		result := new(SttResult)
		json.Unmarshal(resByte, &result)
		fmt.Println(result.Status)
		if result.Status == "completed" {
			fmt.Println(result.Results)
			return
		}
	}
}
