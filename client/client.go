package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)

	if err != nil {
		panic(err)
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		if err, ok := err.(*url.Error); ok && err.Timeout() {
			fmt.Println("Context timeout was exceeded")
			return
		}
		panic(err)
	}

	defer res.Body.Close()

	f, err := os.Create("arquivo.txt")

	if err != nil {
		panic(err)
	}

	// Temporary variable to access internal keys
	var result map[string]interface{}

	// Convert
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	dolarString := fmt.Sprintf("Dólar: %.4f", result["Bid"].(float64))

	_, err = f.Write([]byte(dolarString))

}
