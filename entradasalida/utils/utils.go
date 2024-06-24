package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"entradasalida/globals"
	"fmt"
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
type BodyEscritura struct {
	PID       int    `json:"pid"`
	Info      []byte `json:"info"`
	Tamaño    int    `json:"tamaño"`
	Direccion int    `json:"direccion"`
}

type BodyRequest struct {
	PID       int `json:"pid"`
	Tamaño    int `json:"tamaño"`
	Direccion int `json:"direccion"`
}
type BodyRequestIO struct {
	NombreDispositivo    string `json:"nombre_dispositivo"`
	PuertoDispositivo    int    `json:"puerto_dispositivo"`
	CategoriaDispositivo string `json:"categoria_dispositivo"`
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

func IO_STDIN_READ(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	tamaño := r.PathValue("tamaño")
	direccion := r.PathValue("direccion")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	tamañoInt, err := strconv.Atoi(tamaño)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	//fmt.Print("El tamaño pedido es: ", tamañoInt)
	direccionInt, err := strconv.Atoi(direccion)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	fmt.Print("\nEsperando texto en consola... ")
	textoIngresado := LeerConsola()
	textoBytes := []byte(textoIngresado)
	if len(textoBytes) > tamañoInt {
		http.Error(w, "El texto ingresado por consola excede el tamaño en bytes", http.StatusInternalServerError)
		return
	}
	fmt.Print("\nEl texto ingresado es: ", textoIngresado)
	fmt.Print("\nGuardando texto en memoria... ")
	var requestBody = BodyEscritura{
		PID:       pidInt,
		Info:      textoBytes,
		Tamaño:    tamañoInt,
		Direccion: direccionInt,
	}
	body, err := json.Marshal(requestBody)
	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/escribir"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("Error guardando el texto ingresado ", err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error al guardar el mensaje ", resp.Status)
		return
	} else {
		log.Printf("Se guardo correctamente el mensaje")
	}
	respuesta, err := json.Marshal("OK")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
	log.Printf("PID: %d - Operación: IO_STDIN_READ", pidInt)
	//log.Printf("Direccion: %s - Operación: IO_STDIN_READ", direccion)

}

func IO_STDOUT_WRITE(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	tamaño := r.PathValue("tamaño")
	direccion := r.PathValue("direccion")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}

	tamañoInt, err := strconv.Atoi(tamaño)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	direccionInt, err := strconv.Atoi(direccion)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}

	var requestBody = BodyEscritura{
		PID:       pidInt,
		Tamaño:    tamañoInt,
		Direccion: direccionInt,
	}
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}

	body, err := json.Marshal(requestBody)
	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/leer"
	fmt.Print("URL: ", url)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando: %s", err.Error())
		return
	}
	response := make([]byte, 0, tamañoInt)
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		return
	}
	time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
	respString := string(response)
	//log.Printf("Direccion: %s - Operación: IO_STDOUT_WRITE", direccion)
	log.Printf("PID: %d - Operación: IO_STDOUT_WRITE", pidInt)
	log.Printf("El texto leido es: %s", respString)
}

// DIAL FS
func IO_FS_CREATE(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	nombreArchivo := r.PathValue("nombre")
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	log.Printf("PID: %d - Crear Archivo: %s", pidInt, nombreArchivo)
	/*

		var requestBody = BodyEscritura{
			PID:       pidInt,
			Tamaño:    tamañoInt,
			Direccion: direccionInt,
		}
		if err != nil {
			http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
			return
		}

		body, err := json.Marshal(requestBody)
		url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/fscreate"
		fmt.Print("URL: ", url)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			fmt.Print("error enviando: %s", err.Error())
			return
		}
		response := make([]byte, 0, tamañoInt)
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			log.Printf("Error al decodificar mensaje: %s\n", err.Error())
			return
		}
		time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
		respString := string(response)
		log.Printf("Direccion: %s - Operación: IO_FS_CREATE", direccion)
		log.Printf("El texto leido es: %s", respString)*/
}
func IO_FS_DELETE(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	nombreArchivo := r.PathValue("nombre")
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	log.Printf("PID: %d - Eliminar Archivo: %s", pidInt, nombreArchivo)
	/*

		var requestBody = BodyEscritura{
			PID:       pidInt,
			Tamaño:    tamañoInt,
			Direccion: direccionInt,
		}
		if err != nil {
			http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
			return
		}

		body, err := json.Marshal(requestBody)
		url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/fscreate"
		fmt.Print("URL: ", url)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			fmt.Print("error enviando: %s", err.Error())
			return
		}
		response := make([]byte, 0, tamañoInt)
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			log.Printf("Error al decodificar mensaje: %s\n", err.Error())
			return
		}
		time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
		respString := string(response)
		log.Printf("Direccion: %s - Operación: IO_FS_CREATE", direccion)
		log.Printf("El texto leido es: %s", respString)*/
}
func IO_FS_TRUNCATE(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	nombreArchivo := r.PathValue("nombre")
	tamaño := r.PathValue("tamaño")
	tamañoInt, err := strconv.Atoi(tamaño)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	log.Printf("PID: %d - Truncar Archivo: %s Tamaño: %d", pidInt, nombreArchivo, tamañoInt)
	/*

		var requestBody = BodyEscritura{
			PID:       pidInt,
			Tamaño:    tamañoInt,
			Direccion: direccionInt,
		}
		if err != nil {
			http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
			return
		}

		body, err := json.Marshal(requestBody)
		url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/fscreate"
		fmt.Print("URL: ", url)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			fmt.Print("error enviando: %s", err.Error())
			return
		}
		response := make([]byte, 0, tamañoInt)
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			log.Printf("Error al decodificar mensaje: %s\n", err.Error())
			return
		}
		time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
		respString := string(response)
		log.Printf("Direccion: %s - Operación: IO_FS_CREATE", direccion)
		log.Printf("El texto leido es: %s", respString)*/
}
func IO_FS_WRITE(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	nombreArchivo := r.PathValue("nombre")
	tamaño := r.PathValue("tamaño")
	//Ver puntero si lo recibe o lo busca, y si lo trabaja como %s o %d
	puntero := r.PathValue("puntero")
	tamañoInt, err := strconv.Atoi(tamaño)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	log.Printf("PID: %d - Leer Archivo: %s - Tamaño a Leer: %d - Puntero Archivo: %s", pidInt, nombreArchivo, tamañoInt, puntero)
	/*

		var requestBody = BodyEscritura{
			PID:       pidInt,
			Tamaño:    tamañoInt,
			Direccion: direccionInt,
		}
		if err != nil {
			http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
			return
		}

		body, err := json.Marshal(requestBody)
		url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/fscreate"
		fmt.Print("URL: ", url)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			fmt.Print("error enviando: %s", err.Error())
			return
		}
		response := make([]byte, 0, tamañoInt)
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			log.Printf("Error al decodificar mensaje: %s\n", err.Error())
			return
		}
		time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
		respString := string(response)
		log.Printf("Direccion: %s - Operación: IO_FS_CREATE", direccion)
		log.Printf("El texto leido es: %s", respString)*/
}
func IO_FS_READ(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	nombreArchivo := r.PathValue("nombre")
	tamaño := r.PathValue("tamaño")
	//Ver puntero si lo recibe o lo busca, y si lo trabaja como %s o %d
	puntero := r.PathValue("puntero")
	tamañoInt, err := strconv.Atoi(tamaño)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	log.Printf("PID: %d - Escribir Archivo: %s - Tamaño a Escribir: %d - Puntero Archivo: %s", pidInt, nombreArchivo, tamañoInt, puntero)
	/*
		var requestBody = BodyEscritura{
			PID:       pidInt,
			Tamaño:    tamañoInt,
			Direccion: direccionInt,
		}
		if err != nil {
			http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
			return
		}

		body, err := json.Marshal(requestBody)
		url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/fscreate"
		fmt.Print("URL: ", url)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			log.Printf("error enviando: %s", err.Error())
			fmt.Print("error enviando: %s", err.Error())
			return
		}
		response := make([]byte, 0, tamañoInt)
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			log.Printf("Error al decodificar mensaje: %s\n", err.Error())
			return
		}
		time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
		respString := string(response)
		log.Printf("Direccion: %s - Operación: IO_FS_CREATE", direccion)
		log.Printf("El texto leido es: %s", respString)*/
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
