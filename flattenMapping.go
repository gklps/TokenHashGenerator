package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

// flattenKeys processes the input recursively, flattening numeric keys and retaining non-numeric keys.
func flattenKeys(parentKey string, value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		flattenedMap := make(map[string]interface{})
		for k, nestedValue := range v {
			var newKey string
			if isInteger(k) {
				if parentKey != "" {
					newKey = parentKey + "-" + k
				} else {
					newKey = k
				}
				flattenedMap[newKey] = flattenKeys(newKey, nestedValue)
			} else {
				newKey = k
				flattenedMap[newKey] = flattenKeys(parentKey, nestedValue)
			}
		}
		return flattenedMap
	case []interface{}:
		for i, item := range v {
			v[i] = flattenKeys(parentKey, item)
		}
		return v
	default:
		return value
	}
}

// isInteger checks if the given string is an integer.
func isInteger(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func main() {
	// Open the input JSON file
	file, err := os.Open("input.json")
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Read the JSON file
	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return
	}

	// Parse the JSON data
	var data []interface{}
	err = json.Unmarshal(byteValue, &data)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Transform the JSON data
	for i, item := range data {
		data[i] = flattenKeys("", item)
	}

	// Convert the transformed data back to JSON
	output, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Write the output to a file
	err = ioutil.WriteFile("output.json", output, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	fmt.Println("Transformation complete. Check output.json for results.")
}
