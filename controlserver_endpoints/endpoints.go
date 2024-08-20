package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"tpm_sync"

	"github.com/beevik/ntp"
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
	StartTime           string
	MaxSessionCount     int
	CurrentSessionCount int
}

// var sessionMap = make(map[string]OpenSession) //how about a list of pointers?

func main() {
	welcomeMessage := " -  -  TPM Control Server  -  - "
	fmt.Println(welcomeMessage)

	// Database connection
	dsn := "username:password@tcp(127.0.0.1:3306)/yourdatabase"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/config", func(w http.ResponseWriter, r *http.Request) {
		configHandler(w, r, db)
	})

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
			maxSessionCount := 3000
			go CreateSimulationSession(maxSessionCount, tpmCfg, N, db)
			return
		default:
			http.Error(w, "Invalid TPM Type", http.StatusBadRequest)
			return
		}

	} else {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
	}

}

func CreateSimulationSession(MaxSessionCount int, config TPMConfig, N []int, db *sql.DB) {

	startTime, _ := getCurrentTimeFromNTP()
	// idStamp := fmt.Sprintf("%v%d%d%d%s", config.K, config.N_0, config.L, config.M, startTime)
	// h := sha256.New()
	// h.Write([]byte(idStamp))
	// token := hex.EncodeToString(h.Sum(nil))
	// fmt.Println(token)

	// simulationData := OpenSession{
	// 	Uid:                 token,
	// 	Config:              config,
	// 	StartTime:           startTime,
	// 	MaxSessionCount:     MaxSessionCount,
	// 	CurrentSessionCount: 0,
	// }

	for simulation := 0; simulation < MaxSessionCount; simulation++ {
		tpm_sync.SyncSession(config.K, N, config.L, config.M, config.TpmType, config.LearnRule)
		fmt.Println(simulation)
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

func getCurrentRFC3339Time() string {
	ntpTime, err := getCurrentTimeFromNTP()
	if err != nil {
		log.Fatalf("Error getting NTP time: %v", err)
		return ""
	}
	return ntpTime.Format(time.RFC3339)
}

func insertIntoDB(db *sql.DB) {
	// map[string]interface{}{
	//     "k":         c.K,
	//     "n":         c.N,
	//     "l":         c.L,
	//     "m":         c.M,
	//     "timestamp": c.Timestamp.Format("2006-01-02 15:04:05"),
	// }
	query := "INSERT INTO yourtable (k, n, l, m, timestamp) VALUES (?, ?, ?, ?, ?)"
	// _, err := db.Exec(query, sqlData["k"], sqlData["n"], sqlData["l"], sqlData["m"], sqlData["timestamp"])
	// if err != nil {
	// 	return fmt.Errorf("failed to insert data into MySQL: %v", err)
	// }
}
