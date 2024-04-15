package main

import (
	"fmt"
	"log"
	"net/http"
)

var memory = make(map[int]string)

func main() {
	http.HandleFunc("/allocate", allocateMemory)
	http.HandleFunc("/free", freeMemory)

	fmt.Println("Memory running on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

func allocateMemory(w http.ResponseWriter, r *http.Request) {
	address := len(memory)
	memory[address] = "DATA"

	fmt.Fprintf(w, "Memory allocated at address %d", address)
}

func freeMemory(w http.ResponseWriter, r *http.Request) {
	address := 0 // Here you would get the address from the request
	if _, exists := memory[address]; exists {
		delete(memory, address)
		fmt.Fprintf(w, "Memory at address %d freed", address)
	} else {
		fmt.Fprintf(w, "Memory at address %d not found", address)
	}
}
