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
	if err != nil {
		log.Printf("Error guardando el texto ingresado %v", err)
		return
	}
	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/escribir"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		log.Printf("Error guardando el texto ingresado %v", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error al guardar el mensaje %s", resp.Status)
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

	body, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error guardando el texto ingresado %v", err)
		return
	}
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

func PathHandler(w http.ResponseWriter, r *http.Request) {
	configResponse := map[string]interface{}{
		"path":       globals.ClientConfig.DialfsPath,
		"tam_block":  globals.ClientConfig.DialfsBlockSize,
		"cant_block": globals.ClientConfig.DialfsBlockCount,
	}

	response, err := json.Marshal(configResponse)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(response)
}

type BodyFileRequest struct {
	Interfaz      string `json:"interfaz"`
	NombreArchivo string `json:"nombreArchivo"`
}
type Metadata struct {
	InitialBlock int `json:"initial_block"`
	Size         int `json:"size"`
}

func IO_FS_CREATE_Handler(w http.ResponseWriter, r *http.Request) {
	var request BodyFileRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Error al decodificar la solicitud", http.StatusBadRequest)
		return
	}

	if request.Interfaz == "" || request.NombreArchivo == "" {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}
	log.Printf("\n\n\nSE ENTRO IOFSCREATE EN IO FUNCION")
	err = CrearArchivoFS(request.Interfaz, request.NombreArchivo)
	if err != nil {
		response := CreateFileResponse{
			Status:  "Error",
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Responder con éxito
	response := CreateFileResponse{
		Status:  "OK",
		Message: "Archivo creado correctamente",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	log.Printf("Archivo '%s' creado en la interfaz '%s'", request.NombreArchivo, request.Interfaz)
}

type CreateFileRequest struct {
	Interfaz      string `json:"interfaz"`
	NombreArchivo string `json:"nombreArchivo"`
}

type CreateFileResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func IO_FS_DELETE_Handler(w http.ResponseWriter, r *http.Request) {
	var request BodyFileRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Error al decodificar la solicitud", http.StatusBadRequest)
		return
	}

	if request.Interfaz == "" || request.NombreArchivo == "" {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	err = EliminarArchivoFS(request.Interfaz, request.NombreArchivo)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al eliminar el archivo: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode("Archivo eliminado correctamente")
	log.Printf("Archivo '%s' eliminado en la interfaz '%s'", request.NombreArchivo, request.Interfaz)
}

type ConfigResponse struct {
	Path       string `json:"path"`
	BlockCount int    `json:"block_count"`
	BlockSize  int    `json:"block_size"`
}

func CrearArchivoFS(interfaz, nombreArchivo string) error {
	// Ruta al archivo de metadata
	metadataPath := fmt.Sprintf("%s/%s", globals.ClientConfig.DialfsPath, nombreArchivo)

	// Verificar si el archivo ya existe
	if _, err := os.Stat(metadataPath); err == nil {
		return fmt.Errorf("el archivo ya existe")
	}

	// Leer el archivo de bloques y el bitmap
	bloquesPath := fmt.Sprintf("%s/bloques.dat", globals.ClientConfig.DialfsPath)
	bitmapPath := fmt.Sprintf("%s/bitmap.dat", globals.ClientConfig.DialfsPath)

	bloquesFile, err := os.OpenFile(bloquesPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo de bloques: %s", err.Error())
	}
	defer bloquesFile.Close()

	bitmapFile, err := os.OpenFile(bitmapPath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("no se pudo abrir el archivo de bitmap: %s", err.Error())
	}
	defer bitmapFile.Close()

	// Leer el bitmap
	bitmap := make([]byte, globals.ClientConfig.DialfsBlockCount)
	_, err = bitmapFile.Read(bitmap)
	if err != nil {
		return fmt.Errorf("no se pudo leer el bitmap: %s", err.Error())
	}

	// Encontrar un bloque libre
	var initialBlock int = -1
	for i := 0; i < globals.ClientConfig.DialfsBlockCount; i++ {
		if bitmap[i] == 0 {
			initialBlock = i
			break
		}
	}
	if initialBlock == -1 {
		return fmt.Errorf("no hay bloques libres disponibles")
	}

	// Marcar el bloque como ocupado en el bitmap
	bitmap[initialBlock] = 1
	_, err = bitmapFile.WriteAt(bitmap, 0)
	if err != nil {
		return fmt.Errorf("no se pudo actualizar el bitmap: %s", err.Error())
	}

	// Crear el archivo de metadata
	metadata := map[string]interface{}{
		"initial_block": initialBlock,
		"size":          0,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("no se pudo codificar la metadata: %s", err.Error())
	}

	err = os.WriteFile(metadataPath, metadataBytes, 0644)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo de metadata: %s", err.Error())
	}

	return nil
}

func EliminarArchivoFS(interfaz, nombreArchivo string) error {
	// Ruta al archivo de metadata
	metadataPath := fmt.Sprintf("%s/%s", globals.ClientConfig.DialfsPath, nombreArchivo)

	// Leer el archivo de metadata
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("error al leer la metadata del archivo: %v", err)
	}

	var metadata Metadata
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return fmt.Errorf("error al decodificar la metadata del archivo: %v", err)
	}

	// Leer el archivo de bitmap
	bitmapPath := fmt.Sprintf("%s/bitmap.dat", globals.ClientConfig.DialfsPath)
	bitmap, err := os.ReadFile(bitmapPath)
	if err != nil {
		return fmt.Errorf("error al leer el bitmap: %v", err)
	}

	// Marcar el bloque como libre en el bitmap
	initialBlock := metadata.InitialBlock
	bitmap[initialBlock] = 0

	err = os.WriteFile(bitmapPath, bitmap, 0644)
	if err != nil {
		return fmt.Errorf("error al actualizar el bitmap: %v", err)
	}

	// Eliminar el archivo de metadata
	err = os.Remove(metadataPath)
	if err != nil {
		return fmt.Errorf("error al eliminar el archivo de metadata: %v", err)
	}

	return nil
}
