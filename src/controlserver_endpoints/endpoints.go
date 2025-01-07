package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"tpm_sync/tpm_controllers"

	"github.com/joho/godotenv"
	"github.com/sourcegraph/conc/pool"
)

var sessionMap = tpm_controllers.NewSessionMap()

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
		fmt.Println("Error loading .env file")
	}

	welcomeMessage := " -  -  TPM Control Server V2  -  - "
	fmt.Println(welcomeMessage)

	dbController, err := tpm_controllers.NewDatabaseController(
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))
	if err != nil {
		fmt.Println(err)
		return
	}

	defer dbController.CloseDb()

	MAX_GOROUTINES, err := strconv.Atoi(os.Getenv("MAX_GOROUTINES"))

	if err != nil {
		fmt.Println("Error while parsing MAX_GOROUTINES")
		return
	}

	workerPool := pool.New().WithMaxGoroutines(MAX_GOROUTINES)

	simController := tpm_controllers.SimulationController{
		SyncController:     tpm_controllers.SyncController{},
		DatabaseController: *dbController,
		WorkerPool:         workerPool,
	}

	http.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
		listSessionMapHandler(w, r, sessionMap)
	})

	http.HandleFunc("/track-sessions", func(w http.ResponseWriter, r *http.Request) {
		trackAllSessionsHandler(w, r, sessionMap)
	})

	http.HandleFunc("/query3DGraph", func(w http.ResponseWriter, r *http.Request) {
		get3DGraphHandler(w, r, dbController)
	})

	http.HandleFunc("/newNoOverlap", func(w http.ResponseWriter, r *http.Request) {
		createNewNoOverlapSession(w, r, &simController)
	})

	http.HandleFunc("/newOverlap", func(w http.ResponseWriter, r *http.Request) {
		createNewOverlapSession(w, r, &simController)
	})

	http.HandleFunc("/iterationHistogram", func(w http.ResponseWriter, r *http.Request) {
		getIterationHistogram(w, r, dbController)
	})

	http.HandleFunc("/events", realTimeSessionHandler)
	http.HandleFunc("/get-config", settingsByUidHandler)

	// go simController.SimulateOnStart(sessionMap)

	http.ListenAndServe(":8080", nil)

}

func listSessionMapHandler(w http.ResponseWriter, r *http.Request, sessionMap *tpm_controllers.SessionMap) {
	sessionMap.Mutex.RLock()
	jsonString, err := json.Marshal(sessionMap.Sessions)
	sessionMap.Mutex.RUnlock()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprint(w, string(jsonString))
}

func trackAllSessionsHandler(w http.ResponseWriter, r *http.Request, sessionMap *tpm_controllers.SessionMap) {
	// Set http headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// You may need this locally for CORS requests
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for client disconnection
	clientGone := r.Context().Done()

	rc := http.NewResponseController(w)
	t := time.NewTicker(time.Second * 5)
	defer t.Stop()
	for {
		select {
		case <-clientGone:
			// fmt.Println("Client disconnected")
			return
		case <-t.C:
			sessionMap.Mutex.RLock()
			jsonString, err := json.Marshal(sessionMap.Sessions)
			sessionMap.Mutex.RUnlock()
			if err != nil {
				fmt.Println(err)
				return
			}

			_, err = fmt.Fprintf(w, "data: %s\n\n", string(jsonString))
			if err != nil {
				return
			}
			err = rc.Flush()
			if err != nil {
				return
			}
		}
	}
}

func realTimeSessionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	// Set http headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// You may need this locally for CORS requests
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for client disconnection
	clientGone := r.Context().Done()

	rc := http.NewResponseController(w)

	var session tpm_controllers.OpenSession
	sessionMap.Mutex.Lock()

	sessionPointer, ok := sessionMap.Sessions[id]

	if !ok {
		sessionMap.Mutex.Unlock()
		fmt.Println("Session UID not found: ", id)
		http.NotFound(w, r)
		return
	}

	session = *sessionPointer

	sessionMap.Sessions[session.Uid].Tracking = true
	sessionMap.Mutex.Unlock()
	session.EnableStateChannel <- true

	for {
		select {
		case <-clientGone:
			// fmt.Println("Client disconnected")
			session.EnableStateChannel <- false
			sessionMap.Mutex.Lock()
			sessionMap.Sessions[session.Uid].Tracking = false
			sessionMap.Mutex.Unlock()

			return
		case currentState := <-session.CurrentStateChannel:
			parsedState, err := json.Marshal(currentState)
			if err != nil {
				return
			}
			_, err = fmt.Fprintf(w, "data: %s\n\n", parsedState)
			if err != nil {
				return
			}
			err = rc.Flush()
			if err != nil {
				return
			}
		}
	}
}

func settingsByUidHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	var session tpm_controllers.OpenSession
	sessionMap.Mutex.RLock()

	sessionPointer, ok := sessionMap.Sessions[id]

	if !ok {
		sessionMap.Mutex.RUnlock()
		fmt.Println("Session UID not found: ", id)
		http.NotFound(w, r)
		return
	}

	session = *sessionPointer

	sessionMap.Mutex.RUnlock()
	parsedConfig, err := json.Marshal(session.Config)
	if err != nil {
		return
	}
	_, err = fmt.Fprint(w, string(parsedConfig))
	if err != nil {
		fmt.Println("Error while sending sessionConfig")
		return
	}
}

type GraphRequestBody struct {
	X         string `json:"X"`
	Y         string `json:"Y"`
	LearnRule string `json:"LearnRule"`
	Scenario  string `json:"Scenario"`
	TableName string `json:"TableName"`
}

func get3DGraphHandler(w http.ResponseWriter, r *http.Request, dbController *tpm_controllers.DatabaseController) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode the incoming JSON request body into the RequestBody struct
	var requestBody GraphRequestBody
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	axisX := strings.ToUpper(requestBody.X)
	axisY := strings.ToUpper(requestBody.Y)
	learnRule := strings.ToUpper(requestBody.LearnRule)
	scenario := strings.ToUpper(requestBody.Scenario)
	// Process the received X and Y values (you can add your logic here)
	fmt.Printf("Received X: %s, Y: %s\n", axisX, axisY)

	validXAxis := dbController.ValidateGraphAxis(axisX)
	validYAxis := dbController.ValidateGraphAxis(axisY)
	if !(validXAxis && validYAxis) {
		fmt.Println("Invalid Axis Requested")
		return //error!!! invalid axis, bad request
	}

	validRule := dbController.ValidateLearnRule(learnRule)
	validScenario := dbController.ValidateScenario(scenario)

	if !(validRule && validScenario) {
		fmt.Println("Invalid TPM Config")
		return //error!!! invalid axis, bad request
	}

	response, err := dbController.QuerySurfaceGraph(axisX, axisY, requestBody.TableName, learnRule, scenario)

	if err != nil {
		fmt.Println("Error while querying graph")
		fmt.Println(err)
		return
	}

	// Send a response back with the received data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// response := map[string]string{
	// 	"message": fmt.Sprintf("Received X: %s, Y: %s", requestBody.X, requestBody.Y),
	// }
	json.NewEncoder(w).Encode(response)
}

type NewNoOverlapRequestBody struct {
	N               []int
	K_last          int
	L               int
	M               int
	Rule            string
	MaxSessionCount int
	MaxIterations   int
}

func createNewNoOverlapSession(w http.ResponseWriter, r *http.Request, simController *tpm_controllers.SimulationController) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode the incoming JSON request body into the RequestBody struct
	var requestBody NewNoOverlapRequestBody
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	tpmInstanceSettings, err := simController.SyncController.SettingsFactory(requestBody.N, requestBody.K_last, requestBody.L, requestBody.M, "NO_OVERLAP", requestBody.Rule)
	if err != nil {
		fmt.Println("Error while creating settings for an instance: ", err)
		http.Error(w, "Error while creating settings for an instance", http.StatusInternalServerError)

		return
	}

	baseSettings := tpm_controllers.BaseSettings{
		TpmType:         "NO_OVERLAP",
		MaxSessionCount: requestBody.MaxSessionCount,
		MaxIterations:   requestBody.MaxIterations,
		MaxWorkerCount:  1,
		LearnRules:      []string{requestBody.Rule},
		MConfigs:        []int{requestBody.M},
		LConfigs:        []int{requestBody.L},
	}

	newSessionToken := simController.SimulateOnDemand(sessionMap, tpmInstanceSettings, baseSettings)

	fmt.Println("Simulating on Demand:", newSessionToken)
	// Send a response back with the received data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"sessionToken": newSessionToken,
	}
	json.NewEncoder(w).Encode(response)

}

type NewOverlapRequestBody struct {
	K               []int
	N_0             int
	L               int
	M               int
	Rule            string
	MaxSessionCount int
	MaxIterations   int
	Scenario        string
}

func createNewOverlapSession(w http.ResponseWriter, r *http.Request, simController *tpm_controllers.SimulationController) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode the incoming JSON request body into the RequestBody struct
	var requestBody NewOverlapRequestBody
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	tpmInstanceSettings, err := simController.SyncController.SettingsFactory(requestBody.K, requestBody.N_0, requestBody.L, requestBody.M, requestBody.Scenario, requestBody.Rule)
	if err != nil {
		fmt.Println("Error while creating settings for an instance: ", err)
		http.Error(w, "Error while creating settings for an instance", http.StatusInternalServerError)

		return
	}

	baseSettings := tpm_controllers.BaseSettings{
		TpmType:         requestBody.Scenario,
		MaxSessionCount: requestBody.MaxSessionCount,
		MaxIterations:   requestBody.MaxIterations,
		MaxWorkerCount:  1,
		LearnRules:      []string{requestBody.Rule},
		MConfigs:        []int{requestBody.M},
		LConfigs:        []int{requestBody.L},
	}

	newSessionToken := simController.SimulateOnDemand(sessionMap, tpmInstanceSettings, baseSettings)

	fmt.Println("Simulating on Demand:", newSessionToken)
	// Send a response back with the received data
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"sessionToken": newSessionToken,
	}
	json.NewEncoder(w).Encode(response)

}

type SuccessIterationCorrelationRequestBody struct {
	TableName   string
	Scenario    string
	LearnRule   string
	Min         int
	Max         int
	BucketCount int
}

func getIterationHistogram(w http.ResponseWriter, r *http.Request, dbController *tpm_controllers.DatabaseController) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Decode the incoming JSON request body into the RequestBody struct
	var requestBody SuccessIterationCorrelationRequestBody
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&requestBody)
	if err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	histogram := dbController.QuerySuccessIterationCorrelation(requestBody.TableName, requestBody.Scenario, requestBody.LearnRule, requestBody.Min, requestBody.Max, requestBody.BucketCount)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string][]tpm_controllers.HistogramEntry{
		"histogram": histogram,
	}
	json.NewEncoder(w).Encode(response)

}
