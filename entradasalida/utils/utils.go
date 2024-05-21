package utils

import (
	"bufio"
	"encoding/json"
	"entradasalida/globals"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

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
	cantidadIO := 4 //Este lo manda el cpu a kernel y kernel a io

	var tiempoAEsperar = cantidadIO * globals.ClientConfig.UnitWorkTime
	time.Sleep(time.Duration(tiempoAEsperar) * time.Millisecond)

	respuesta, err := json.Marshal("OK")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}
