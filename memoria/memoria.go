package main

import (
	"net/http"
)

//var memory = make(map[int]string)

func main() {
	//readFile("../leer.txt") //con ../ busca el que esta en general, con ./ o sin aclarar busca el de memoria
	mux := http.NewServeMux()
	//mux.HandleFunc("DELETE /process", finalizarProceso)
	mux.HandleFunc("PUT /process", crearProceso)
	err := http.ListenAndServe(":8002", mux)
	if err != nil {
		panic(err)
	}
}
