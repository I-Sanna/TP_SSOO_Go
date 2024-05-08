package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type BodyRequestPath struct {
	Path string `json:"path"`
}

type BodyRequestPid struct {
	PID int `json:"pid"`
}

func main() {
	body, err := json.Marshal(BodyRequestPath{
		Path: "kernel/virus.exe",
	})
	if err != nil {
		return
	}

	body = body

	cliente := &http.Client{}
	url := fmt.Sprintf("http://localhost:8001/enviar")
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	respuesta, err := cliente.Do(req)
	if err != nil {
		return
	}

	// Verificar el código de estado de la respuesta
	if respuesta.StatusCode != http.StatusOK {
		return
	}

	bodyBytes, err := io.ReadAll(respuesta.Body)
	if err != nil {
		return
	}

	fmt.Println(string(bodyBytes))
}
