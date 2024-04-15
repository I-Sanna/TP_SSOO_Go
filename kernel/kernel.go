package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("GET /process", iniciarProceso)
	http.HandleFunc("/process/", finalizarProceso)
	http.HandleFunc("/process/", estadoProceso)
	http.HandleFunc("/process", listarProcesos)
	http.HandleFunc("/plani", iniciarPlanificacion)
	http.HandleFunc("/plani", detenerPlanificacion)

	fmt.Println("Kernel escuchando en el puerto 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
