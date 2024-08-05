package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type DummyData struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

var db *sql.DB

func main() {
	var err error
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresPort := os.Getenv("POSTGRES_PORT")

	if postgresUser == "" || postgresPassword == "" || postgresDB == "" || postgresHost == "" || postgresPort == "" {
		log.Fatal("Missing one or more required environment variables")
	}

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		postgresHost, postgresPort, postgresUser, postgresPassword, postgresDB)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("Unable to ping database: %v\n", err)
	}

	// Create table if not exists
	createTableSQL := `CREATE TABLE IF NOT EXISTS dummy_data (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL,
        value TEXT NOT NULL
    );`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Unable to create table: %v\n", err)
	}

	router := mux.NewRouter()

	// Wrap the existing handler functions with the CORS middleware
	router.HandleFunc("/rows", getRowsHandler).Methods("GET")
	router.HandleFunc("/rows", postRowHandler).Methods("POST")
	http.Handle("/", corsMiddleware(router))

	log.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func getRowsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, value FROM dummy_data")
	if err != nil {
		http.Error(w, "Failed to query database", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var dummyDataList []DummyData
	for rows.Next() {
		var d DummyData
		err := rows.Scan(&d.ID, &d.Name, &d.Value)
		if err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}
		dummyDataList = append(dummyDataList, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dummyDataList)
}

func postRowHandler(w http.ResponseWriter, r *http.Request) {
	var d DummyData
	err := json.NewDecoder(r.Body).Decode(&d)
	if err != nil {
		http.Error(w, "Failed to decode JSON body", http.StatusBadRequest)
		return
	}

	err = db.QueryRow("INSERT INTO dummy_data (name, value) VALUES ($1, $2) RETURNING id", d.Name, d.Value).Scan(&d.ID)
	if err != nil {
		http.Error(w, "Failed to insert row", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(d)
}
