package main

import (
	"fmt"
	"log"
	"net/http"
)

type IOInterface struct {
	Name string `json:"name"`
}

var ioInterfaces = make(map[string]IOInterface)

func main() {
	http.HandleFunc("/connect", connectInterface)
	http.HandleFunc("/disconnect", disconnectInterface)

	fmt.Println("I/O Interfaces running on :8083")
	log.Fatal(http.ListenAndServe(":8083", nil))
}

func connectInterface(w http.ResponseWriter, r *http.Request) {
	var io IOInterface
	io.Name = "Interface1" // Here you would get the name from the request

	ioInterfaces[io.Name] = io

	fmt.Fprintf(w, "Interface %s connected", io.Name)
}

func disconnectInterface(w http.ResponseWriter, r *http.Request) {
	name := "Interface1" // Here you would get the name from the request
	if _, exists := ioInterfaces[name]; exists {
		delete(ioInterfaces, name)
		fmt.Fprintf(w, "Interface %s disconnected", name)
	} else {
		fmt.Fprintf(w, "Interface %s not found", name)
	}
}
