package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/execute", executeInstruction)

	fmt.Println("CPU running on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func ejecutarInstruccion(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "instruccion ejecutada correctamente")
}
