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
		sessionHandler(w, r, sessionMap)
	})

	go simController.SimulateOnStart(sessionMap)

	http.ListenAndServe(":8080", nil)

}

func sessionHandler(w http.ResponseWriter, r *http.Request, sessionMap *tpm_controllers.SessionMap) {
	sessionMap.Mutex.RLock()
	jsonString, err := json.Marshal(sessionMap.Sessions)
	sessionMap.Mutex.RUnlock()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprint(w, string(jsonString))
}
