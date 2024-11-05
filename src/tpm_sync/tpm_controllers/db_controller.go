package tpm_controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
	"tpm_sync/tpm_core"

	_ "github.com/go-sql-driver/mysql"
)

type DatabaseController struct {
	db *sql.DB
}

func NewDatabaseController(username, password, db_host, db_port, db_name string) (*DatabaseController, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	// Database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
		return nil, err
	}

	dbController := DatabaseController{db: db}

	return &dbController, nil
}

func (dc *DatabaseController) CloseDb() error {
	return dc.db.Close()
}

func (dc *DatabaseController) insertIntoDB(config TPMmSettings, session SessionData, startTime time.Time, endTime time.Time) {

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
		"program_version":      runtime.Version(),
		"k":                    string(kJSON),
		"n_0":                  config.N[0],
		"l":                    config.L,
		"m":                    config.M,
		"h":                    config.H,
		"data_size":            tpm_core.GetNetworkDataSize(config.H, config.K, config.N),
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
	query := fmt.Sprintf("INSERT INTO %s (host, seed, program_version, k, n_0, l, m, h, data_size, tpm_type, learn_rule, start_time, end_time, status, stimulate_iterations, learn_iterations, initial_state, final_state) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", os.Getenv("DB_NAME"))
	_, err = dc.db.Exec(query, sqlData["host"], sqlData["seed"], sqlData["program_version"], sqlData["k"], sqlData["n_0"], sqlData["l"], sqlData["m"], sqlData["h"], sqlData["data_size"], sqlData["tpm_type"], sqlData["learn_rule"], sqlData["start_time"], sqlData["end_time"], sqlData["status"], sqlData["stimulate_iterations"], sqlData["learn_iterations"], sqlData["initial_state"], sqlData["final_state"])
	if err != nil {
		fmt.Println(fmt.Errorf("failed to insert data into MySQL: %v", err))
	}

}

func (dc *DatabaseController) FetchFullTableAsJSON(tableName string) (string, error) {
	// Query to retrieve all data from the specified table
	rows, err := dc.db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
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

func (dc *DatabaseController) GetFullIterationStatsFromDB(tableName string) (IterationStatsResponse, error) {
	fmt.Println("Querying iteration stats to DB...")
	// Example queries for each surface graph

	queries := map[string]IterationGroup{
		"H_vs_N0": {X: "H", Y: "N0"},
		"H_vs_L":  {X: "H", Y: "L"},
		"D_vs_H":  {X: "D", Y: "H"},
	}
	graphResponse := IterationStatsResponse{Graphs: make(map[string][][]interface{})}

	for name, query := range queries {
		data, err := dc.QuerySurfaceGraph(query.X, query.Y, tableName)
		if err != nil {
			return graphResponse, err
		}
		// Store each graph data with its corresponding name
		graphResponse.Graphs[name] = data
	}

	return graphResponse, nil

}

// QueryGraph performs the query and returns the data for the graph
func (dc *DatabaseController) QuerySurfaceGraph(X string, Y string, tableName string) ([][]interface{}, error) {
	queryBody := fmt.Sprintf(`MIN(stimulate_iterations), MAX(stimulate_iterations), AVG(stimulate_iterations),
                  	MIN(learn_iterations), MAX(learn_iterations), AVG(learn_iterations)
            		FROM %s 
            		WHERE status = 'FINISHED'`, tableName)
	query := fmt.Sprintf("SELECT %s, %s, %s GROUP BY %s, %s;", X, Y, queryBody, X, Y)
	rows, err := dc.db.Query(query)
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

// QueryFinishedCount retrieves the count of 'FINISHED' rows and total rows
func (dc *DatabaseController) QueryFinishedCount(tableName string) ([]FinishedCountData, error) {
	fmt.Println("Querying session count to DB...")
	query := fmt.Sprintf(`
        SELECT
            learn_rule,
            tpm_type,
            CONCAT(H, '-', L) AS h_l_group,
            COUNT(CASE WHEN status = 'FINISHED' THEN 1 END) AS finished_count,
            COUNT(*) AS total_count
        FROM
            %s
        GROUP BY
            learn_rule, tpm_type, h_l_group;
    `, tableName)

	rows, err := dc.db.Query(query)
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

func (dc *DatabaseController) GetSessionsByK(kValues []int, tableName string, tpmType string) (*SessionAvgsAndCounts, error) {
	// Convert kValues into a JSON array string for querying
	jsonArray := make([]string, len(kValues))
	for i, val := range kValues {
		jsonArray[i] = fmt.Sprintf("%d", val)
	}
	jsonK := fmt.Sprintf("[%s]", strings.Join(jsonArray, ", "))
	query := fmt.Sprintf(`
        SELECT 
            COALESCE(AVG(learn_iterations), 0) AS avg_learn_iterations, 
            COALESCE(AVG(stimulate_iterations), 0) AS avg_stimulate_iterations,
			COUNT(*) AS total_count,
            SUM(CASE WHEN status = 'FINISHED' THEN 1 ELSE 0 END) AS finished_count
		FROM 
            %s
        WHERE 
			tpm_type = %s
			AND CAST(K as CHAR) = ?
    `, tableName, tpmType)

	result := SessionAvgsAndCounts{}
	err := dc.db.QueryRow(query, jsonK).Scan(
		&result.AvgLearnIterations,
		&result.AvgStimulateIterations,
		&result.TotalCount,
		&result.FinishedCount,
		&result.UnfinishedCount,
	)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}

	return &result, nil
}
