package tpm_sync

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/beevik/ntp"
	"github.com/sourcegraph/conc/pool"
)

func SimulateOnStart(db *sql.DB, seed int64) {

	max_iterations := 100_000
	max_threads := 3

	neuronConfigs := [][]int{
		{3},
		{5, 3},
		{7, 5, 3},
		{13, 11, 9, 7, 5, 3},
		{15, 13, 11, 9, 7, 5, 3},
	}
	nConfigs := []int{2, 3, 4, 5, 6, 7, 8, 9}
	mConfigs := []int{1}

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

	tpmTypes := []string{
		"fully_connected",
		"partially_connected",
		"no_overlap",
	}

	learnRules := []string{
		"Hebbian",
		"Anti-Hebbian",
		"Random-Walk",
	}

	workerPool := pool.New().WithMaxGoroutines(max_threads)

	for _, tpm_type := range tpmTypes {
		for _, rule := range learnRules {
			for _, k := range neuronConfigs {
				for _, l := range lConfigs {
					for _, n_0 := range nConfigs {
						for _, m := range mConfigs {
							tpmSettings, err := SettingsFactory(k, n_0, l, m, tpm_type, rule)
							if err != nil {
								continue
							}
							workerPool.Go(func() {
								startTime, ntpErr := getCurrentTimeFromNTP()
								if ntpErr != nil {
									startTime = time.Now()
								}
								session := SyncSession(tpmSettings, max_iterations, seed)
								endTime, ntpErr := getCurrentTimeFromNTP()
								if ntpErr != nil {
									endTime = time.Now()
								}
								insertIntoDB(db, tpmSettings, session, startTime, endTime)
								fmt.Println(session.Status)
							})
							// fmt.Println(session.FinalWeights)
						}
					}
				}
			}
		}
	}
	workerPool.Wait()
	fmt.Println("-- All automatic configs finished --")
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

func insertIntoDB(db *sql.DB, config TPMmSettings, session SessionData, startTime time.Time, endTime time.Time) {

	kJSON, err := json.Marshal(config.K)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to marshal K: %v", err))
	}

	initialStateJSON, err := json.Marshal(session.InitialState)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to marshal K: %v", err))

	}

	finalStateJSON, err := json.Marshal(session.FinalState)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to marshal K: %v", err))

	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = os.Getenv("HOSTNAME")
	}

	sqlData := map[string]interface{}{
		"host":                 hostname,
		"seed":                 session.Seed,
		"version":              runtime.Version(),
		"k":                    string(kJSON),
		"n_0":                  config.N[0],
		"l":                    config.L,
		"m":                    config.M,
		"tpm_type":             config.LinkType,
		"learn_rule":           config.LearnRule,
		"start_time":           startTime.Format("2006-01-02 15:04:05"),
		"end_time":             endTime.Format("2006-01-02 15:04:05"),
		"status":               session.Status,
		"stimulate_iterations": session.StimulateIterations,
		"learn_iterations":     session.LearnIterations,
		"initial_state":        string(initialStateJSON),
		"final_state":          string(finalStateJSON),
	}
	query := fmt.Sprintf("INSERT INTO %s (host, seed, version, k, n_0, l, m, tpm_type, learn_rule, start_time, end_time, status, stimulate_iterations, learn_iterations, initial_state, final_state) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", os.Getenv("DB_NAME"))
	_, err = db.Exec(query, sqlData["host"], sqlData["seed"], sqlData["version"], sqlData["k"], sqlData["n_0"], sqlData["l"], sqlData["m"], sqlData["tpm_type"], sqlData["learn_rule"], sqlData["start_time"], sqlData["end_time"], sqlData["status"], sqlData["stimulate_iterations"], sqlData["learn_iterations"], sqlData["initial_state"], sqlData["final_state"])
	if err != nil {
		fmt.Println(fmt.Errorf("failed to insert data into MySQL: %v", err))
	}
}
