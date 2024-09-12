package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"tpm_sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var sessionMap = tpm_sync.NewSessionMap()

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	welcomeMessage := " -  -  TPM Control Server  -  - "
	fmt.Println(welcomeMessage)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	// Database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/datadump", func(w http.ResponseWriter, r *http.Request) {
		datadumpHandler(w, r, db)
	})

	http.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
		sessionHandler(w, r, sessionMap)
	})

	fmt.Println("Starting simulation...")
	go tpm_sync.SimulateOnStart(db, sessionMap)

	http.ListenAndServe(":8080", nil)

}

func datadumpHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	jsonResult, err := FetchTableAsJSON(db, os.Getenv("DB_NAME"))
	if err != nil {
		http.Error(w, "Error fetching table as JSON", http.StatusInternalServerError)
	}

	fmt.Fprint(w, jsonResult)
}

func sessionHandler(w http.ResponseWriter, r *http.Request, sessionMap *tpm_sync.SessionMap) {
	sessionMap.Mutex.RLock()
	jsonString, err := json.Marshal(sessionMap.Sessions)
	sessionMap.Mutex.RUnlock()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprint(w, string(jsonString))
}

func FetchTableAsJSON(db *sql.DB, tableName string) (string, error) {
	// Query to retrieve all data from the specified table
	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
	if err != nil {
		return "", fmt.Errorf("error retrieving data: %v", err)
	}
	defer rows.Close()

	// Slice to hold the result
	var results []map[string]interface{}

	// Get the column names
	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("error getting columns: %v", err)
	}

	// Iterate over the rows
	for rows.Next() {
		// Create a slice of interface{}'s to hold each value, and a second slice to contain pointers to each item in the interface{} slice
		values := make([]interface{}, len(columns))
		valuePointers := make([]interface{}, len(columns))
		for i := range values {
			valuePointers[i] = &values[i]
		}

		// Scan the result into the value pointers
		if err := rows.Scan(valuePointers...); err != nil {
			return "", fmt.Errorf("error scanning row: %v", err)
		}

		// Create a map and fill it with the row data
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]

			// Convert []byte to string for readability
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}

			rowMap[col] = v
		}

		results = append(results, rowMap)
	}

	// Convert the results slice to JSON
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshaling results to JSON: %v", err)
	}

	return string(jsonData), nil
}
