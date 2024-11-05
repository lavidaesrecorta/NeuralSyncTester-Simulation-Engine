package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"tpm_sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var sessionMap = tpm_sync.NewSessionMap()

var queryTimeThreshold = 12 * time.Hour
var lastSurfaceQuery time.Time
var lastSurfaceResponse SurfaceGraphResponse
var lastCountQuery time.Time
var lastCountResponse []FinishedCountData

func main() {
	err := godotenv.Load(".env")
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

	http.HandleFunc("/sessionStats", func(w http.ResponseWriter, r *http.Request) {
		surfaceGraphHandler(w, r, db, false)
	})

	http.HandleFunc("/sessionStats-refresh", func(w http.ResponseWriter, r *http.Request) {
		surfaceGraphHandler(w, r, db, true)
	})

	http.HandleFunc("/sessionCount", func(w http.ResponseWriter, r *http.Request) {
		countGraphHandler(w, r, db, false)
	})

	http.HandleFunc("/sessionCount-refresh", func(w http.ResponseWriter, r *http.Request) {
		countGraphHandler(w, r, db, true)
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

func surfaceGraphHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, overrideTimeCheck bool) {

	dbResult := lastSurfaceResponse
	if time.Since(lastSurfaceQuery) > queryTimeThreshold || overrideTimeCheck {
		var err error
		dbResult, err = GetIterationStatsFromDB(db)
		if err != nil {
			http.Error(w, "Error fetching from db", http.StatusInternalServerError)
			return // Add return to stop further execution
		}
		lastSurfaceQuery = time.Now()
		lastSurfaceResponse = dbResult
	}

	response := GraphResponse{
		Result:     dbResult,
		LastUpdate: lastSurfaceQuery,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
	}
}
func countGraphHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, overrideTimeCheck bool) {

	dbResult := lastCountResponse
	if time.Since(lastCountQuery) > queryTimeThreshold || overrideTimeCheck {
		var err error
		dbResult, err = QueryFinishedCount(db)
		if err != nil {
			http.Error(w, "Error fetching from db", http.StatusInternalServerError)
			return // Add return to stop further execution
		}
		lastCountQuery = time.Now()
		lastCountResponse = dbResult
	}
	response := GraphResponse{
		Result:     dbResult,
		LastUpdate: lastCountQuery,
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
	}
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

type GraphResponse struct {
	Result     interface{} `json:"result"`
	LastUpdate time.Time   `json:"last_update"`
}

// SurfaceGraphResponse holds the result for multiple graphs
type SurfaceGraphResponse struct {
	Graphs map[string][][]interface{} `json:"graphs"`
}

func GetIterationStatsFromDB(db *sql.DB) (SurfaceGraphResponse, error) {
	fmt.Println("Querying iteration stats to DB...")
	// Example queries for each surface graph
	queries := map[string]string{
		"H_vs_N0": `
            SELECT H, N_0, 
                   MIN(stimulate_iterations), MAX(stimulate_iterations), AVG(stimulate_iterations),
                   MIN(learn_iterations), MAX(learn_iterations), AVG(learn_iterations)
            FROM sessions 
            WHERE status = 'FINISHED'
            GROUP BY H, N_0;`,
		"H_vs_L": `
            SELECT H, L, 
                   MIN(stimulate_iterations), MAX(stimulate_iterations), AVG(stimulate_iterations),
                   MIN(learn_iterations), MAX(learn_iterations), AVG(learn_iterations)
            FROM sessions 
            WHERE status = 'FINISHED'
            GROUP BY H, L;`,
		"M_vs_N0": `
            SELECT M, N_0, 
                   MIN(stimulate_iterations), MAX(stimulate_iterations), AVG(stimulate_iterations),
                   MIN(learn_iterations), MAX(learn_iterations), AVG(learn_iterations)
            FROM sessions 
            WHERE status = 'FINISHED'
            GROUP BY M, N_0;`,
		"D_vs_H": `
            SELECT data_size, H, 
                   MIN(stimulate_iterations), MAX(stimulate_iterations), AVG(stimulate_iterations),
                   MIN(learn_iterations), MAX(learn_iterations), AVG(learn_iterations)
            FROM sessions 
            WHERE status = 'FINISHED'
            GROUP BY data_size, H;`,
	}

	graphResponse := SurfaceGraphResponse{Graphs: make(map[string][][]interface{})}

	for name, query := range queries {
		data, err := QuerySurfaceGraph(db, query)
		if err != nil {
			return graphResponse, err
		}
		// Store each graph data with its corresponding name
		graphResponse.Graphs[name] = data
	}

	return graphResponse, nil

}

// GraphData represents a row in the graph data with multiple statistics for two Z-values
type GraphData struct {
	X  string // X-axis value
	Y  string // Y-axis value
	Z1 Stats  // Stats for stimulate_iterations
	Z2 Stats  // Stats for learn_iterations
}

// Stats holds the statistical values for a column
type Stats struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
	Avg float64 `json:"avg"`
}

// QueryGraph performs the query and returns the data for the graph
func QuerySurfaceGraph(db *sql.DB, query string) ([][]interface{}, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var graphData [][]interface{}

	// Append the headers ["X", "Y", "stimulate_min", "stimulate_max", "stimulate_avg", "learn_min", "learn_max", "learn_avg"]
	graphData = append(graphData, []interface{}{"X", "Y", "stimulate_min", "stimulate_max", "stimulate_avg", "learn_min", "learn_max", "learn_avg"})

	for rows.Next() {
		var x, y string
		var stimulateMin, stimulateMax, stimulateAvg float64
		var learnMin, learnMax, learnAvg float64
		err := rows.Scan(&x, &y, &stimulateMin, &stimulateMax, &stimulateAvg, &learnMin, &learnMax, &learnAvg)
		if err != nil {
			return nil, err
		}

		// Append a row of results
		graphData = append(graphData, []interface{}{x, y, stimulateMin, stimulateMax, stimulateAvg, learnMin, learnMax, learnAvg})
	}

	return graphData, nil
}

// FinishedCountData holds the result of the modified query
type FinishedCountData struct {
	LearnRule     string `json:"learn_rule"`
	TPMType       string `json:"tpm_type"`
	HLGroup       string `json:"h_l_group"`
	FinishedCount int    `json:"finished_count"`
	TotalCount    int    `json:"total_count"`
}

// QueryFinishedCount retrieves the count of 'FINISHED' rows and total rows
func QueryFinishedCount(db *sql.DB) ([]FinishedCountData, error) {
	fmt.Println("Querying session count to DB...")
	query := `
        SELECT
            learn_rule,
            tpm_type,
            CONCAT(H, '-', L) AS h_l_group,
            COUNT(CASE WHEN status = 'FINISHED' THEN 1 END) AS finished_count,
            COUNT(*) AS total_count
        FROM
            sessions
        GROUP BY
            learn_rule, tpm_type, h_l_group;
    `

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []FinishedCountData

	for rows.Next() {
		var data FinishedCountData
		err := rows.Scan(&data.LearnRule, &data.TPMType, &data.HLGroup, &data.FinishedCount, &data.TotalCount)
		if err != nil {
			return nil, err
		}
		results = append(results, data)
	}

	return results, nil
}
