package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {

	db, err := gorm.Open(sqlite.Open("quote_database.db"), &gorm.Config{})

	if err != nil {
		log.Fatal("Failed to connect to the database", err)
	}

	db.AutoMigrate(&Coin{})

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		GetDollarQuote(w, r, db)
	})

	http.ListenAndServe(":8080", nil)

}

type Coin struct {
	gorm.Model
	Code string
	Bid  float64
}

func GetDollarQuote(w http.ResponseWriter, r *http.Request, db *gorm.DB) {

	rContext := r.Context()

	log.Println("Request started")
	defer log.Println("Request ended")

	// Context for dollar quote request
	ctx, cancel := context.WithTimeout(rContext, 200*time.Millisecond)

	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Do the request
	res, err := http.DefaultClient.Do(req)

	// Check errors
	if err != nil {
		if err == context.DeadlineExceeded {
			log.Printf("The request to the external API exceeded 200ms")

		} else {
			log.Printf("Error when making the request to the external API: %v", err)
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	defer res.Body.Close()

	// Verify if the request was cancelled
	select {
	case <-rContext.Done():
		log.Println("Request was cancelled by the user.")
		http.Error(w, "Request cancelled by the client", http.StatusRequestTimeout)
		return
	default:
	}

	// Temporary variable to access internal keys
	var result map[string]interface{}

	// Convert
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	// Feature to access nested map
	usdbrl := result["USDBRL"].(map[string]interface{})

	// Feature to convert the string to a float64
	bidStr, ok := usdbrl["bid"].(string)
	if !ok {

		// If it isnt a string, try converting it to a string and parsing it
		bidStr = fmt.Sprintf("%v", usdbrl["bid"])
	}

	// Convert to float64
	bid, err := strconv.ParseFloat(bidStr, 64)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Convert the response to a struct
	coin := Coin{
		Code: usdbrl["code"].(string),
		Bid:  bid,
	}

	// Call the function that inserts the coin into the database
	err = insertCoin(db.WithContext(rContext), coin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	coinToEncode, err := getLastCoin(db.WithContext(rContext))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(coinToEncode)
}

func insertCoin(db *gorm.DB, coin Coin) error {

	// Creates a new context with a 10ms timeout from the original context
	ctx, cancel := context.WithTimeout(db.Statement.Context, 10*time.Millisecond)
	defer cancel()

	insertion := db.WithContext(ctx).Create(&coin)

	if insertion.Error != nil {
		if insertion.Error == context.DeadlineExceeded {
			log.Printf("The insertion request into the database exceeded 10ms")
		}
		return insertion.Error
	}

	return nil
}

func getLastCoin(db *gorm.DB) (*Coin, error) {

	// Creates a new context with a 10ms timeout from the original context
	ctx, cancel := context.WithTimeout(db.Statement.Context, 10*time.Millisecond)
	defer cancel()

	var c Coin

	selection := db.WithContext(ctx).Last(&c)

	if selection.Error != nil {
		if selection.Error == context.DeadlineExceeded {
			log.Printf("Database query request exceeded 10ms")
		}
		return nil, selection.Error
	}

	return &c, nil
}
