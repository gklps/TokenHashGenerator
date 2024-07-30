package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

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

func main() {
	// Open database connection
	db, err := sql.Open("sqlite3", "./token_data.db")
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()
	// HTTP handler for the verification endpoint
	http.HandleFunc("/verify", verifyTokensHandler)

	// Start the server
	fmt.Println("Server listening on port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func verifyTokensHandler(w http.ResponseWriter, r *http.Request) {
	var input []string
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Open the token hash file only once
	file, err := os.Open("token_hashes.txt")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error opening file: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	results := make(map[string]bool, len(input))
	for _, tokenHash := range input {

		//tokenHash = strings.TrimPrefix(tokenHash, "00")

		levelStr := strings.TrimLeft(tokenHash[:3], "0") // Trim leading zeros
		level, _ := strconv.Atoi(levelStr)               // Convert to integer
		hash := tokenHash[3:]
		fmt.Printf("Verifying token: %s (level %d)\n", hash, level) // Logging the token and level

		// Efficiently search for the hash in the file
		found := false
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, ":")
			if len(parts) == 2 && parts[1] == tokenHash[3:] {
				fmt.Println("Hash found in file:", line) // Logging the matching line from the file
				tokenNumber, _ := strconv.Atoi(parts[0])
				fmt.Printf("Token number: %d, Max value for level %d: %d\n", tokenNumber, level, TokenMap[level])
				maxTokenValue := TokenMap[level]
				results[tokenHash] = 0 < tokenNumber && tokenNumber <= maxTokenValue
				found = true
				break
			}
		}

		if !found {
			fmt.Printf("Hash not found in file: %s\n", tokenHash[3:]) // Logging when the hash is not found
			results[tokenHash] = false
		}

		// Reset the scanner to the beginning of the file for the next token
		_, err = file.Seek(0, 0) // 0 means start of the file and 0 is the reference point
		if err != nil {
			http.Error(w, fmt.Sprintf("Error resetting file pointer: %v", err), http.StatusInternalServerError)
			return
		}
		scanner = bufio.NewScanner(file)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}
