package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

var httpClient = &http.Client{}

func MakeAPICall(method, url string, headers map[string]string, body interface{}) (int, []byte, error) {
	req, err := prepareRequest(method, url, headers, body)
	if err != nil {
		log.Println("Error in Preparaing Request")
		return http.StatusInternalServerError, nil, err
	}
	resp, err := httpClient.Do(req)

	if err != nil {
		log.Println("Error in Making API Call ", err)
		return http.StatusInternalServerError, nil, err
	}
	defer resp.Body.Close()
	rBody, err := io.ReadAll(resp.Body)
	return resp.StatusCode, rBody, err

}
func prepareRequest(method, url string, headers map[string]string, body interface{}) (*http.Request, error) {
	marshallBody, marshallError := json.Marshal(body)
	if marshallError != nil {
		return nil, marshallError
	}
	request, err := http.NewRequest(method, url, bytes.NewBuffer(marshallBody))

	if err != nil {
		log.Println("Error in Preparaing Request")
		return nil, err
	}
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	return request, err

}
