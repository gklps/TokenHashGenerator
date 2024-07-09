package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/ipfs/kubo/client/rpc"
)

const (
	hashLimit         = 4300000 // Maximum token number for hash calculation
	concurrency       = 100     // Maximum concurrent hash calculations
	hashFileName      = "token_hashes.txt"
	bufferSize        = 1024 * 1024 // 1MB buffer size
	channelBufferSize = 1000        // Buffer size for the channel
)

var ipfs *rpc.Client

// Initialize IPFS connection
func init() {
	var err error
	ipfs, err = rpc.NewClientWithOpts(context.Background(), rpc.WithHost("localhost"), rpc.WithPort("5002"))
	if err != nil {
		log.Fatalf("Error connecting to IPFS daemon: %v", err)
	}
}

// calculateSHA256 calculates SHA256 hash
func calculateSHA256(input string) string {
	hasher := sha256.New()
	hasher.Write([]byte(input))
	return hex.EncodeToString(hasher.Sum(nil))
}

// calculateAndStoreHashes calculates and stores hashes concurrently to a file
func calculateAndStoreHashes(limit int, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriterSize(file, bufferSize)
	defer writer.Flush()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	hashChannel := make(chan string, channelBufferSize)

	// Writer goroutine
	go func() {
		for hash := range hashChannel {
			if _, err := writer.WriteString(hash); err != nil {
				log.Printf("Error writing hash: %v", err)
			}
		}
		writer.Flush()
	}()

	// Worker goroutines
	for i := 0; i <= limit; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(tokenNumber int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			hash := calculateSHA256(strconv.Itoa(tokenNumber))
			hashChannel <- fmt.Sprintf("%d:%s\n", tokenNumber, hash)
		}(i)
	}

	wg.Wait()
	close(hashChannel) // Close channel when all goroutines are done

	return nil
}

// handleGetTokenByHash handles API request for token lookup
func handleGetTokenByHash(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var hashes []string
		if err := json.NewDecoder(r.Body).Decode(&hashes); err != nil {
			hash := strings.TrimPrefix(r.URL.Path, "/token/")
			if hash != "" {
				hashes = append(hashes, hash)
			} else {
				http.Error(w, "Invalid hash input", http.StatusBadRequest)
				return
			}
		}

		response := make(map[string]interface{}) // Use interface{} for flexibility
		for _, hash := range hashes {
			tokenNumber, err := getTokenByHashFromFile(hash, filename)
			if err != nil {
				response[hash] = err.Error() // Store error message for the hash
			} else {
				response[hash] = tokenNumber // Store token number if found
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response) // Encode the entire map
	}
}

// handleGetLevelTokenByHash handles API request for level and token lookup by hash
func handleGetLevelTokenByHash(filename string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input := strings.TrimPrefix(r.URL.Path, "/leveltoken/")
		if input == "" {
			http.Error(w, "Missing level and hash value", http.StatusBadRequest)
			return
		}
		parts := strings.Split(input, "-")
		if len(parts) != 2 {
			http.Error(w, "Invalid input format (expected: level-hash)", http.StatusBadRequest)
			return
		}
		level, hash := parts[0], parts[1]
		if _, err := strconv.Atoi(level); err != nil {
			http.Error(w, "Invalid level format", http.StatusBadRequest)
			return
		}

		tokenNumber, err := getTokenByHashFromFile(hash, filename)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		result := fmt.Sprintf("%s%d", level, tokenNumber)
		cid, err := addStringToIPFS(result)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error adding to IPFS: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]string{"Result": result, "CID": cid}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// getTokenByHashFromFile retrieves token number from a hash file
func getTokenByHashFromFile(hash, filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) == 2 && parts[1] == hash {
			tokenNumber, err := strconv.Atoi(parts[0])
			if err != nil {
				return 0, fmt.Errorf("invalid token number in file: %v", err)
			}
			return tokenNumber, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error scanning file: %v", err)
	}
	return 0, fmt.Errorf("hash not found")
}

// addStringToIPFS adds a string to IPFS and returns the CID
func addStringToIPFS(data string) (string, error) {
	// Add the data to IPFS using the new client
	result, err := ipfs.Add(context.Background(), strings.NewReader(data))
	if err != nil {
		return "", err
	}
	return result.Cid.String(), nil
}

func main() {
	// Calculate and store hashes (optional, run this manually once)
	if err := calculateAndStoreHashes(hashLimit, hashFileName); err != nil {
		log.Fatalf("Error calculating and storing hashes: %v", err)
	}

	// API handlers
	http.HandleFunc("/token/", handleGetTokenByHash(hashFileName))
	http.HandleFunc("/leveltoken/", handleGetLevelTokenByHash(hashFileName))

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server listening on port :%s", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
