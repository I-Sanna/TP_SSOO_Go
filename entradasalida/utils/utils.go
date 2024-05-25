package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"entradasalida/globals"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type BodyRequestTime struct {
	TIME int `json:"tiempo"`
}

func IniciarConfiguracion(filePath string) *globals.Config {
	var config *globals.Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func LeerConsola() string {
	// Leer de la consola
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = text[:len(text)-1]
	return text
}

func ConfigurarLogger() {
	logFile, err := os.OpenFile("logs/entradasalida.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func IO_GEN_SLEEP(w http.ResponseWriter, r *http.Request) {

	cantidad := r.PathValue("units")
	pid := r.PathValue("pid")

	cantidadInt, err := strconv.Atoi(cantidad)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}

	log.Printf("PID: %d - Operación: IO_GEN_SLEEP", pidInt)

	var tiempoAEsperar = cantidadInt * globals.ClientConfig.UnitWorkTime
	time.Sleep(time.Duration(tiempoAEsperar) * time.Millisecond)

	respuesta, err := json.Marshal("OK")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

type BodyRequestIO struct {
	NombreDispositivo    string `json:"nombre_dispositivo"`
	PuertoDispositivo    int    `json:"puerto_dispositivo"`
	CategoriaDispositivo string `json:"categoria_dispositivo"`
}

func EstablecerConexion(nombre string, puerto int) {
	var datosConexion BodyRequestIO

	datosConexion.NombreDispositivo = nombre
	datosConexion.PuertoDispositivo = puerto
	datosConexion.CategoriaDispositivo = globals.ClientConfig.Type

	body, err := json.Marshal(datosConexion)
	if err != nil {
		log.Printf("error codificando mensajes: %s", err.Error())
		return
	}

	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortKernel) + "/nuevoIO"
	_, err = http.Post(url, "application/json", bytes.NewBuffer(body)) // Enviando nil como el cuerpo
	if err != nil {
		log.Printf("error enviando: %s", err.Error())
		return
	}
}

func ValidarConexion(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
