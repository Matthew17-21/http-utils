package main

import (
	"context"
	"io"
	"log"

	httputils "github.com/Matthew17-21/http-utils"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

func main() {
	// Create a new HTTP client
	client := newClient()
	retryableClient := httputils.NewRetryableHTTPClient(client)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", "https://httpbin.org/brotli", nil)
	if err != nil {
		log.Fatalln("Error getting HTTP request:", err)
	}
	req.Header.Set("Accept-Encoding", "br, gzip, deflate")

	// Make a request
	// resp, err := client.Do(req)
	resp, err := retryableClient.MakeRequest(context.Background(), req)
	if err != nil {
		log.Fatalln("Error making HTTP request:", err)
	}

	// Read the body
	parseResponse(resp)

}

func newClient() tls_client.HttpClient {
	options := []tls_client.HttpClientOption{}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		log.Fatalln("Error getting HTTP client:", err)
	}
	return client
}

func parseResponse(resp *http.Response) {
	log.Printf("HTTP Status: %s\n", resp.Status)
	log.Printf("Content-Encoding: %s\n", resp.Header.Get("Content-Encoding"))
	// NOTE: No need to decompress the response, fhttp handles it automatically
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Error reading HTTP response:", err)
	}
	log.Println("Body:", string(body))
}
