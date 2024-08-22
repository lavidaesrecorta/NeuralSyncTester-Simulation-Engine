package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"tpm_sync"

	"github.com/beevik/ntp"
	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type TPMConfig struct {
	K         []int  `json:"k"`
	N_0       int    `json:"n"`
	L         int    `json:"l"`
	M         int    `json:"m"`
	TpmType   string `json:"type"`
	LearnRule string `json:"learnRule"`
}

type OpenSession struct {
	Uid                 string
	Config              TPMConfig
	StartTime           time.Time
	MaxSessionCount     int
	CurrentSessionCount int
}

var sessionMap = make(map[string]*OpenSession) //how about a list of pointers?

func main() {

	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	welcomeMessage := " -  -  TPM Control Server  -  - "
	fmt.Println(welcomeMessage, os.Getenv("MAX_SIMULATION_REPETITIONS"))
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	// Database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		configHandler(w, r, db)
	})

	http.HandleFunc("/datadump", func(w http.ResponseWriter, r *http.Request) {
		datadumpHandler(w, r, db)
	})

	go simulateOnStart(db)

	http.ListenAndServe(":8080", nil)

}

func configHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Unable to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Do something with the request body

		var tpmCfg TPMConfig
		err = json.Unmarshal(body, &tpmCfg)
		if err != nil {
			http.Error(w, "Invalid JSON structure", http.StatusBadRequest)
			return
		}

		h := len(tpmCfg.K)
		// Validate the struct fields
		if h == 0 || tpmCfg.N_0 == 0 || tpmCfg.L == 0 || tpmCfg.M == 0 {
			http.Error(w, "Invalid configuration data", http.StatusBadRequest)
			return
		}

		N := make([]int, h)

		switch tpmType := strings.ToUpper(tpmCfg.TpmType); tpmType {
		case "OVERLAP":
			N = tpm_sync.CreateConvolutionalStimulusStructure(tpmCfg.K, tpmCfg.N_0)
			if N == nil {
				http.Error(w, "Invalid layer structure", http.StatusBadRequest)
				return
			}
			// fmt.Fprint(w, "TPM input stim structure:", N)
			maxSessionCount, err := strconv.Atoi(os.Getenv("MAX_SIMULATION_REPETITIONS"))
			if err != nil {
				maxSessionCount = 3000
			}
			go CreateSimulationSession(maxSessionCount, tpmCfg, N, db)
			fmt.Println("New config received in endpoint:", tpmCfg)
			return
		default:
			http.Error(w, "Invalid TPM Type", http.StatusBadRequest)
			return
		}

	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}
}

func simulateOnStart(db *sql.DB) {

	neuronConfigs := [][]int{
		{3},
		{5, 3},
		{7, 5, 3},
		{13, 11, 9, 7, 5, 3},
		{15, 13, 11, 9, 7, 5, 3},
	}

	lConfigs := []int{
		3,
		4,
		5,
		6,
		7,
		8,
		9,
		// 10,
		// 11,
		// 12,
		// 13,
	}

	// fmt.Fprint(w, "TPM input stim structure:", N)
	maxSessionCount, err := strconv.Atoi(os.Getenv("MAX_SIMULATION_REPETITIONS"))
	if err != nil {
		maxSessionCount = 3000
	}

	for i := range neuronConfigs {

		n_0, err := strconv.Atoi(os.Getenv("DEFAULT_N_0"))
		if err != nil {
			n_0 = 5
		}
		N := tpm_sync.CreateConvolutionalStimulusStructure(neuronConfigs[i], n_0)
		if N == nil {
			fmt.Println("Invalid layer structure")
			return
		}

		m, err := strconv.Atoi(os.Getenv("DEFAULT_M"))
		if err != nil {
			m = 5
		}
		for j := range lConfigs {

			tpmCfg := TPMConfig{
				K:         neuronConfigs[i],
				N_0:       n_0,
				L:         lConfigs[j],
				M:         m,
				TpmType:   os.Getenv("DEFAULT_TYPE"),
				LearnRule: os.Getenv("DEFAULT_RULE"),
			}

			CreateSimulationSession(maxSessionCount, tpmCfg, N, db)
		}
		fmt.Println("Config finished:", neuronConfigs[i])
	}
	selectQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", os.Getenv("DB_NAME"))

	var count int
	err = db.QueryRow(selectQuery).Scan(&count)
	if err != nil {
		log.Fatalf("Error getting count: %v", err)
	}

	fmt.Printf("Number of elements in the table: %d\n", count)
	fmt.Println("-- All automatic configs finished --")
}

func datadumpHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	jsonResult, err := FetchTableAsJSON(db, os.Getenv("DB_NAME"))
	if err != nil {
		http.Error(w, "Error fetching table as JSON", http.StatusInternalServerError)
	}

	fmt.Fprint(w, jsonResult)
}

func CreateSimulationSession(MaxSessionCount int, config TPMConfig, N []int, db *sql.DB) {

	maxIterCount, err := strconv.Atoi(os.Getenv("MAX_ITERATION_COUNT"))
	if err != nil {
		maxIterCount = 7 * 1_000_000
	}

	startTime, _ := getCurrentTimeFromNTP()
	idStamp := fmt.Sprintf("%v%d%d%d%s", config.K, config.N_0, config.L, config.M, startTime)
	h := sha256.New()
	h.Write([]byte(idStamp))
	token := hex.EncodeToString(h.Sum(nil))
	// fmt.Println(token)

	simulationData := OpenSession{
		Uid:                 token,
		Config:              config,
		StartTime:           startTime,
		MaxSessionCount:     MaxSessionCount,
		CurrentSessionCount: 0,
	}

	sessionMap[token] = &simulationData

	for simulation := 0; simulation < MaxSessionCount; simulation++ {
		sessionStartTime, _ := getCurrentTimeFromNTP()
		sessionData := tpm_sync.SyncSession(config.K, N, config.L, config.M, config.TpmType, config.LearnRule, maxIterCount)
		insertIntoDB(db, config, sessionData, sessionStartTime)
		simulationData.CurrentSessionCount = 1
	}

}

func getCurrentTimeFromNTP() (time.Time, error) {
	// You can use a specific NTP server or use a pool
	ntpServer := "ntp.shoa.cl"
	// Retrieve the time from the NTP server
	time, err := ntp.Time(ntpServer)
	if err != nil {
		return time, fmt.Errorf("failed to get time from NTP server: %v", err)
	}
	return time, nil
}

func insertIntoDB(db *sql.DB, config TPMConfig, session tpm_sync.SessionData, startTime time.Time) {
	endTime, ntpErr := getCurrentTimeFromNTP()
	if ntpErr != nil {
		endTime = time.Now()
	}

	kJSON, err := json.Marshal(config.K)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to marshal K: %v", err))
	}
	weightsJSON, err := json.Marshal(session.FinalWeights)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to marshal K: %v", err))
	}
	sqlData := map[string]interface{}{
		"host":                 os.Getenv("HOSTNAME"),
		"k":                    string(kJSON),
		"n_0":                  config.N_0,
		"l":                    config.L,
		"m":                    config.M,
		"tpm_type":             config.TpmType,
		"learn_rule":           config.LearnRule,
		"start_time":           startTime.Format("2006-01-02 15:04:05"),
		"end_time":             endTime.Format("2006-01-02 15:04:05"),
		"status":               session.Status,
		"stimulate_iterations": session.StimulateIterations,
		"learn_iterations":     session.LearnIterations,
		"final_weights":        string(weightsJSON),
	}
	query := fmt.Sprintf("INSERT INTO %s (host, k, n_0, l, m, tpm_type, learn_rule, start_time, end_time, status, stimulate_iterations, learn_iterations, final_weights) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", os.Getenv("DB_NAME"))
	_, err = db.Exec(query, sqlData["host"], sqlData["k"], sqlData["n_0"], sqlData["l"], sqlData["m"], sqlData["tpm_type"], sqlData["learn_rule"], sqlData["start_time"], sqlData["end_time"], sqlData["status"], sqlData["stimulate_iterations"], sqlData["learn_iterations"], sqlData["final_weights"])
	if err != nil {
		fmt.Println(fmt.Errorf("failed to insert data into MySQL: %v", err))
	}
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
