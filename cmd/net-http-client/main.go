package main

import (
	"context"
	"log"
	"net/http"

	httputils "github.com/Matthew17-21/http-utils"
)

func main() {
	url := "https://httpbin.org/brotli"

	// Create HTTP client
	client := &http.Client{}
	retryableClient := httputils.NewRetryableHTTPClient(client)

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Set("Accept-Encoding", "br, gzip, deflate")

	// Perform the request
	resp, err := retryableClient.MakeRequest(context.Background(), req)
	if err != nil {
		log.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("HTTP Status: %s\n", resp.Status)
	log.Printf("Content-Encoding: %s\n", resp.Header.Get("Content-Encoding"))

	decompressed, err := httputils.DecompressResponse(resp.Header, resp.Body)
	if err != nil {
		log.Printf("Error decompressing response: %v\n", err)
		return
	}
	log.Println("Decompressed response:", string(decompressed))

}
