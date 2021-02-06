// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"log"
)

func main() {
	fileContents, err := ioutil.ReadFile("test/socket.js")
	if err != nil {
		log.Fatal("Could not read socket.js from expected location 'test/socket.js': %v", err)
	}
	outputFile := fmt.Sprintf(`package websocket

var socketJsContents = %s%s%s

`, "`", fileContents, "`")

	ioutil.WriteFile("socket.js.go", []byte(outputFile), 0666)
}
