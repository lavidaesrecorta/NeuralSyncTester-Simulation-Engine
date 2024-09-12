package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
	"tpm_sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	"golang.org/toolchain/src/math/rand"
)

func main() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	rand.Seed(time.Now().UnixNano())

	welcomeMessage := " -  -  TPM Control Server  -  - "
	fmt.Println(welcomeMessage)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_NAME"))
	// Database connection
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}
	defer db.Close()

	tpm_sync.SimulateOnStart(db, time.Now().UnixNano())

}
