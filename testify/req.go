package testify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

const DefaultServerUrl = "http://127.0.0.1:8021"
const DefaultURLPrefix = "server?Action"

func sendRequest[RespT interface{}](action string, req interface{}) (*BizResponse[RespT], error) {
	if action == "" {
		return nil, errors.New("aciont cant be empty")
	}
	jsonData, err := json.Marshal(req)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return nil, err
	}
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = DefaultServerUrl
	}
	url := fmt.Sprintf("%s/%s=%s", serverURL, DefaultURLPrefix, action)
	fmt.Println("Sending request to:", url)
	fmt.Println("Request data:", string(jsonData))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	fmt.Println("Response status:", resp.Status)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}
	fmt.Println("Response body:", string(bodyBytes))
	mateResp := &BizResponse[RespT]{}
	if err = json.Unmarshal(bodyBytes, &mateResp); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return nil, err
	}
	return mateResp, nil
}

type BizResponse[RespT any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail"`
	Data    RespT  `json:"data"`
}

func ptr[T any](v T) *T {
	return &v
}
