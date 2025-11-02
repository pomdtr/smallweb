package main

import (
	"encoding/json"
	"fmt"

	"github.com/tailscale/hujson"
)

type JsonPatchOp struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func main() {
	// 1. Your HuJSON input string.
	// It includes comments and a trailing comma, which are handled by hujson.
	jsonString := `{
		// The name of the user
		"name": "John Doe",
		"age": 30, // User's age
		"isStudent": false,
		"courses": [
			"History",
			"Math" // Final course
		],
	}`

	patch, err := json.Marshal([]JsonPatchOp{
		{Op: "replace", Path: "/name", Value: "Jane Doe"},
	})

	if err != nil {
		panic(err)
	}

	// 2. Parse the HuJSON string.
	parsed, err := hujson.Parse([]byte(jsonString))
	if err != nil {
		panic(err)
	}

	if err := parsed.Patch(patch); err != nil {
		panic(err)
	}

	parsed.Format()
	packed := parsed.Pack()

	// 3. Output the modified HuJSON string.
	fmt.Println(string(packed))
}
