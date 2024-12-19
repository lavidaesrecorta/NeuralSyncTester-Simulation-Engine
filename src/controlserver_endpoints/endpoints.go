package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"tpm_sync/tpm_controllers"

	"github.com/joho/godotenv"
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

	simController := tpm_controllers.SimulationController{
		SyncController:     tpm_controllers.SyncController{},
		DatabaseController: *dbController,
	}

	http.HandleFunc("/sessions", func(w http.ResponseWriter, r *http.Request) {
		listSessionMapHandler(w, r, sessionMap)
	})
	http.HandleFunc("/events", realTimeSessionHandler)

	go simController.SimulateOnStart(sessionMap)

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

func realTimeSessionHandler(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	fmt.Println(id)
	// Set http headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// You may need this locally for CORS requests
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for client disconnection
	clientGone := r.Context().Done()

	rc := http.NewResponseController(w)

	var sessionChannel chan tpm_controllers.SessionStateMessage

	sessionMap.Mutex.RLock()
	for k := range sessionMap.Sessions {
		sessionChannel = sessionMap.Sessions[k].CurrentStateChannel
		break
	}
	sessionMap.Mutex.RUnlock()

	// t := time.NewTicker(time.Second)
	// defer t.Stop()

	for {
		select {
		case <-clientGone:
			fmt.Println("Client disconnected")
			return
		case currentState := <-sessionChannel:
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
			// case <-t.C:
			// 	// Send an event to the client
			// 	// Here we send only the "data" field, but there are few others
			// 	_, err := fmt.Fprintf(w, "data: The time is %s\n\n", time.Now().Format(time.UnixDate))
			// 	if err != nil {
			// 		return
			// 	}
			// 	err = rc.Flush()
			// 	if err != nil {
			// 		return
			// 	}
		}
	}
}
