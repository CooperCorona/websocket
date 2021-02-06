package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func main() {
	var err error
	var s struct {
		Name string `json:"name"`
		Data struct {
			X int    `json:"x"`
			Y string `json:"y"`
			Z bool   `json:"z"`
		} `json:"data"`
	}
	s.Name = "cooperEvent"
	s.Data.X = 1
	s.Data.Y = "two"
	s.Data.Z = true
	b, err := json.Marshal(s)
	fmt.Printf("Bytes: %v\n", b)
	fmt.Printf("Err: %v\n", err)
	var u struct {
		Name string          `json:"name"`
		Data json.RawMessage `json:"data"`
	}
	err = json.Unmarshal(b, &u)
	// err = json.NewDecoder(bytes.NewReader(b)).Decode(&u)
	fmt.Printf("U: %v\n", u)
	fmt.Printf("Err; %v\n", err)
	var st struct {
		Name string `json:"name"`
		Data struct {
			X int    `json:"x"`
			Y string `json:"y"`
			Z bool   `json:"z"`
		} `json:"data"`
	}
	err = json.Unmarshal(b, &st)
	// err = json.NewDecoder(bytes.NewReader(b)).Decode(&st)
	fmt.Printf("ST: %v\n", u)
	fmt.Printf("Err; %v\n", err)
	var str string
	err = json.NewDecoder(bytes.NewReader(b)).Decode(&str)
	fmt.Printf("Str: %v\n", str)
	fmt.Printf("Err: %v\n", err)
}
