package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// TokenVerificationResponse represents the structure of the JSON response.
type TokenVerificationResponse struct {
	Results map[string]bool `json:"results"`
}

// TokenMap is your provided mapping of levels to maximum token values.
var TokenMap = map[int]int{
	0:  0,
	1:  5000000,
	2:  2425000,
	3:  2303750,
	4:  2188563,
	5:  2079134,
	6:  1975178,
	7:  1876419,
	8:  1782598,
	9:  1693468,
	10: 1608795,
	11: 1528355,
	12: 1451937,
	13: 1379340,
	14: 1310373,
	15: 1244855,
	16: 1182612,
	17: 1123481,
	18: 1067307,
	19: 1013942,
	20: 963245,
	21: 915082,
	22: 869328,
	23: 825862,
	24: 784569,
	25: 745340,
	26: 708073,
	27: 672670,
	28: 639036,
	29: 607084,
	30: 576730,
	31: 547894,
	32: 520499,
	33: 494474,
	34: 469750,
	35: 446263,
	36: 423950,
	37: 402752,
	38: 382615,
	39: 363484,
	40: 345310,
	41: 328044,
	42: 311642,
	43: 296060,
	44: 281257,
	45: 267194,
	46: 253834,
	47: 241143,
	48: 229085,
	49: 217631,
	50: 206750,
	51: 196412,
	52: 186592,
	53: 177262,
	54: 168399,
	55: 159979,
	56: 151980,
	57: 144381,
	58: 137162,
	59: 130304,
	60: 117273,
	61: 105546,
	62: 94992,
	63: 85492,
	64: 76943,
	65: 69249,
	66: 62324,
	67: 56092,
	68: 50482,
	69: 45434,
	70: 40891,
	71: 36802,
	72: 33121,
	73: 29809,
	74: 26828,
	75: 24146,
	76: 21731,
	77: 19558,
	78: 17602,
}

var db *sql.DB
var mutex sync.Mutex

func main() {
	// Open database connection (once)
	var err error
	db, err = sql.Open("sqlite3", "./token_data.db")
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	// Prepare the database statements (once)
	prepareStatements(db) // Function to prepare statements (see below)

	// HTTP handler
	http.HandleFunc("/verify", verifyTokensHandler)

	// Start server
	fmt.Println("Server listening on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// Prepared statements (for better performance)
var stmtVerifyToken *sql.Stmt

func prepareStatements(db *sql.DB) {
	var err error
	stmtVerifyToken, err = db.Prepare("SELECT token_value FROM tokens WHERE token_hash = ?")
	if err != nil {
		fmt.Printf("Error preparing statement: %v\n", err)
		return // Or handle the error more gracefully
	}
}

func verifyTokensHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Tokens []string `json:"tokens"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := make(map[string]bool, len(input.Tokens))
	// Use WaitGroup for concurrent processing
	var wg sync.WaitGroup
	wg.Add(len(input.Tokens)) // Set the number of goroutines to wait for
	for _, tokenHash := range input.Tokens {
		go func(tokenHash string) {
			defer wg.Done() // Signal when the goroutine is done

			levelStr := strings.TrimLeft(tokenHash[:3], "0")
			level, _ := strconv.Atoi(levelStr)

			hash := tokenHash[3:]
			fmt.Printf("Verifying token: %s (level %d)\n", hash, level)
			// Length check:
			if len(tokenHash) != 67 {
				fmt.Printf("Invalid token length: %s\n", tokenHash)
				results[tokenHash] = false
				return // Skip further processing if the length is invalid
			}

			// Lock the mutex to ensure safe concurrent access to the database
			mutex.Lock()
			var tokenNumberFromFile int
			err := stmtVerifyToken.QueryRow(hash).Scan(&tokenNumberFromFile) // Use the prepared statement
			if err != nil {
				if err == sql.ErrNoRows {
					fmt.Printf("Hash not found in database: %s\n", hash)
				} else {
					// Log or handle the database error
					fmt.Printf("Database error: %v\n", err)
				}
				results[tokenHash] = false
			} else {
				fmt.Printf("Token number: %d (from database), Max value for level %d: %d\n", tokenNumberFromFile, level, TokenMap[level])
				results[tokenHash] = 0 < tokenNumberFromFile && tokenNumberFromFile <= TokenMap[level]
			}
			// Unlock the mutex
			mutex.Unlock()
		}(tokenHash) // Passing the tokenHash argument to the goroutine
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Create a TokenVerificationResponse struct and populate it with the results.
	response := TokenVerificationResponse{Results: results}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response) // Encode the map
}
