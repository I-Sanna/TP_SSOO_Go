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
type BodyRequestFS struct {
	PID        int    `json:"pid"`
	Archivo    string `json:"nombre_archivo"`
	Tamaño     int    `json:"tamaño"`
	Direccion  int    `json:"direccion"`
	PtrArchivo int    `json:"ptrarchivo"`
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

var tablaSegmentacion map[int]string

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

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func CrearTablaSegmentacion() {
	directory := globals.ClientConfig.DialfsPath // The current directory

	files, err := os.Open(directory) //open the directory to read files in the directory
	if err != nil {
		fmt.Println("error opening directory:", err) //print error if directory is not opened
		return
	}
	defer files.Close() //close the directory opened

	fileInfos, err := files.Readdir(-1) //read the files from the directory
	if err != nil {
		fmt.Println("error reading directory:", err) //if directory is not read properly print error message
		return
	}
	for _, fileInfos := range fileInfos {
		if fileInfos.Name()[len(fileInfos.Name())-5:] == ".json" {
			metadata := obtenerMetadata(globals.ClientConfig.DialfsPath + "/" + fileInfos.Name())
			tablaSegmentacion[metadata.InitialBlock] = fileInfos.Name()[:len(fileInfos.Name())-5]
		}
	}
}

func CrearEstructurasNecesariasFS() {
	tablaSegmentacion = make(map[int]string)
	CrearTablaSegmentacion()

	fileData := make([]byte, globals.ClientConfig.DialfsBlockSize*globals.ClientConfig.DialfsBlockCount)
	bitmapData := make([]byte, globals.ClientConfig.DialfsBlockCount)

	filePath := globals.ClientConfig.DialfsPath + "/bloques.dat"
	bitmapPath := globals.ClientConfig.DialfsPath + "/bitmap.dat"

	_, err := os.Stat(filePath)
	_, err2 := os.Stat(bitmapPath)

	if err == nil && err2 == nil {
		return
	}

	data, err := os.Create(filePath)
	check(err)

	defer data.Close()

	_, err = data.Write(fileData)
	if err != nil {
		log.Printf("Error creando el archivo %s", filePath)
	}

	bitmap, err := os.Create(bitmapPath)
	check(err)

	defer bitmap.Close()

	_, err = bitmap.Write(bitmapData)
	if err != nil {
		log.Printf("Error creando el archivo %s", filePath)
	}

	log.Print("Se crearon los archivos de bloque y bitmap")
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
	fmt.Print("El tamaño pedido es: ", tamañoInt)
	direccionInt, err := strconv.Atoi(direccion)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}
	fmt.Print("\nEsperando texto en consola... ")
	textoIngresado := LeerConsola()
	fmt.Print("\n Paso leer consola ")
	textoBytes := []byte(textoIngresado)
	fmt.Print("\nConvirtió a array de bytes el texto ")
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
		log.Printf("Error al codificar la solicitud %v", err)
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

type BodyTruncate struct {
	Pid           int    `json:"pid"`
	NombreArchivo string `json:"nombre_archivo"`
	Tamaño        int    `json:"tamaño"`
}

func ObtenerBitmap() []byte {
	bitmapFile, err := os.OpenFile(globals.ClientConfig.DialfsPath+"/bitmap.dat", os.O_RDWR, 0644)
	if err != nil {
		log.Printf("no se pudo abrir el archivo de bitmap: %s", err.Error())
	}
	defer bitmapFile.Close()

	// Leer el bitmap
	bitmap := make([]byte, globals.ClientConfig.DialfsBlockCount)
	_, err = bitmapFile.Read(bitmap)
	if err != nil {
		log.Printf("no se pudo leer el bitmap: %s", err.Error())
	}

	return bitmap
}

func ModificarBitmap(bitmap []byte) {
	bitmapFile, err := os.OpenFile(globals.ClientConfig.DialfsPath+"/bitmap.dat", os.O_RDWR, 0644)
	if err != nil {
		log.Printf("no se pudo abrir el archivo de bitmap: %s", err.Error())
	}
	defer bitmapFile.Close()

	_, err = bitmapFile.WriteAt(bitmap, 0)
	if err != nil {
		log.Printf("no se pudo actualizar el bitmap: %s", err.Error())
	}
}

func OcuparEspacioLibreContiguo(bitmap []byte, bloquesOcupados int, bloquesNecesarios int, metadataPath string) []byte {
	metadata := obtenerMetadata(metadataPath)

	bloquesAOcupar := bloquesNecesarios - bloquesOcupados

	espacioContiguoLibre := true
	for i := 0; i < bloquesAOcupar; i++ {
		if metadata.InitialBlock+bloquesOcupados+i == globals.ClientConfig.DialfsBlockCount {
			espacioContiguoLibre = false
			break
		}
		if bitmap[metadata.InitialBlock+bloquesOcupados+i] != 0 {
			espacioContiguoLibre = false
			break
		}
	}

	if espacioContiguoLibre {
		for i := 0; i < bloquesAOcupar; i++ {
			bitmap[metadata.InitialBlock+bloquesOcupados+i] = 1
		}
		return bitmap
	} else {
		bitmapCompactado := Compactar(bitmap, metadata)
		return OcuparEspacioLibreContiguo(bitmapCompactado, bloquesOcupados, bloquesNecesarios, metadataPath)
	}
}

func Compactar(bitmap []byte, metadata Metadata) []byte {
	contadorEspaciosLibres := 0
	inicioEspacioLibre := -1
	cadenaDe0 := false

	bloquesOcupados := metadata.Size / globals.ClientConfig.DialfsBlockSize
	if bloquesOcupados == 0 || metadata.Size%globals.ClientConfig.DialfsBlockSize != 0 {
		bloquesOcupados++
	}

	for i := 0; i < metadata.InitialBlock+bloquesOcupados+1; i++ {
		if i == metadata.InitialBlock+bloquesOcupados {
			bitmap = liberarBloques(bitmap, i, contadorEspaciosLibres)
			actualizarTablaSegmentacion(inicioEspacioLibre, i, contadorEspaciosLibres)
		} else if bitmap[i] == 0 && !cadenaDe0 {
			cadenaDe0 = true

			if inicioEspacioLibre == -1 {
				inicioEspacioLibre = i
			}

			actualizarTablaSegmentacion(inicioEspacioLibre, i, contadorEspaciosLibres)
			inicioEspacioLibre = i - contadorEspaciosLibres
			contadorEspaciosLibres++
			bitmap[i] = 1
		} else if bitmap[i] == 1 {
			cadenaDe0 = false
		} else {
			contadorEspaciosLibres++
			bitmap[i] = 1
		}
	}

	contadorEspaciosLibres = 0
	inicioEspacioLibre = -1

	for i := globals.ClientConfig.DialfsBlockCount - 1; i > metadata.InitialBlock+bloquesOcupados-2; i-- {
		if i == metadata.InitialBlock+bloquesOcupados-1 {
			bitmap = liberarBloques(bitmap, i, -contadorEspaciosLibres)
			actualizarTablaSegmentacion(inicioEspacioLibre, i, -contadorEspaciosLibres)
		} else if bitmap[i] == 0 && !cadenaDe0 {
			cadenaDe0 = true

			if inicioEspacioLibre == -1 {
				inicioEspacioLibre = i
			}

			actualizarTablaSegmentacion(inicioEspacioLibre, i, -contadorEspaciosLibres)
			inicioEspacioLibre = i + contadorEspaciosLibres
			contadorEspaciosLibres++
			bitmap[i] = 1
		} else if bitmap[i] == 1 {
			cadenaDe0 = false
		} else {
			contadorEspaciosLibres++
			bitmap[i] = 1
		}
	}

	return bitmap
}

func actualizarTablaSegmentacion(bloqueInicial int, bloqueFinal int, desplazamiento int) {
	bloquesFile, err := os.OpenFile(globals.ClientConfig.DialfsPath+"/bloques.dat", os.O_RDWR, 0644)
	if err != nil {
		log.Printf("no se pudo abrir el archivo de bloques: %s", err.Error())
	}
	defer bloquesFile.Close()

	if desplazamiento > 0 {
		for i := bloqueInicial; i < bloqueFinal; i++ {
			archivo, ok := tablaSegmentacion[i]
			if ok {
				metadata := obtenerMetadata(globals.ClientConfig.DialfsPath + "/" + archivo + ".json")

				data := make([]byte, metadata.Size)
				_, err = bloquesFile.ReadAt(data, int64(metadata.InitialBlock)*int64(globals.ClientConfig.DialfsBlockSize))
				if err != nil {
					log.Printf("no se pudo leer el archivo de bloques: %s", err.Error())
				}

				metadata.InitialBlock -= desplazamiento

				_, err = bloquesFile.WriteAt(data, int64(metadata.InitialBlock)*int64(globals.ClientConfig.DialfsBlockSize))
				if err != nil {
					log.Printf("no se pudo actualizar el bitmap: %s", err.Error())
				}

				delete(tablaSegmentacion, i)
				tablaSegmentacion[metadata.InitialBlock] = archivo

				metadataBytes, err := json.Marshal(metadata)
				if err != nil {
					log.Printf("no se pudo codificar la metadata: %s", err.Error())
				}

				err = os.WriteFile(globals.ClientConfig.DialfsPath+"/"+archivo+".json", metadataBytes, 0644)
				if err != nil {
					log.Printf("no se pudo crear el archivo de metadata: %s", err.Error())
				}
			}
		}
	} else {
		for i := bloqueInicial; i > bloqueFinal; i-- {
			archivo, ok := tablaSegmentacion[i]
			if ok {
				metadata := obtenerMetadata(globals.ClientConfig.DialfsPath + "/" + archivo + ".json")

				data := make([]byte, metadata.Size)
				_, err = bloquesFile.ReadAt(data, int64(metadata.InitialBlock)*int64(globals.ClientConfig.DialfsBlockSize))
				if err != nil {
					log.Printf("no se pudo leer el archivo de bloques: %s", err.Error())
				}

				metadata.InitialBlock -= desplazamiento

				_, err = bloquesFile.WriteAt(data, int64(metadata.InitialBlock)*int64(globals.ClientConfig.DialfsBlockSize))
				if err != nil {
					log.Printf("no se pudo actualizar el bitmap: %s", err.Error())
				}

				delete(tablaSegmentacion, i)
				tablaSegmentacion[metadata.InitialBlock] = archivo

				metadataBytes, err := json.Marshal(metadata)
				if err != nil {
					log.Printf("no se pudo codificar la metadata: %s", err.Error())
				}

				err = os.WriteFile(globals.ClientConfig.DialfsPath+"/"+archivo+".json", metadataBytes, 0644)
				if err != nil {
					log.Printf("no se pudo crear el archivo de metadata: %s", err.Error())
				}
			}
		}
	}
}

func liberarBloques(bitmap []byte, puntero int, cantidad int) []byte {
	if cantidad > 0 {
		for i := 0; i < cantidad && puntero-(i+1) > 0; i++ {
			bitmap[puntero-(i+1)] = 0
		}
	} else {
		for i := 0; i > cantidad && puntero-(i-1) < globals.ClientConfig.DialfsBlockCount; i-- {
			bitmap[puntero-(i-1)] = 0
		}
	}
	fmt.Printf("Puntero: %d - Cantidad: %d", puntero, cantidad)
	return bitmap
}

func IO_FS_TRUNCATE(w http.ResponseWriter, r *http.Request) {
	var request BodyTruncate
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Error al decodificar la solicitud", http.StatusBadRequest)
		return
	}

	metadataPath := globals.ClientConfig.DialfsPath + "/" + request.NombreArchivo + ".json"

	if _, err := os.Stat(metadataPath); err != nil {
		log.Printf("El archivo: %s no existe", request.NombreArchivo)
		return
	}

	log.Printf("PID: %d - Truncar Archivo: %s - Tamaño: %d", request.Pid, request.NombreArchivo, request.Tamaño)

	metadata := obtenerMetadata(metadataPath)

	bitmap := ObtenerBitmap()

	bloquesOcupados := metadata.Size / globals.ClientConfig.DialfsBlockSize
	if bloquesOcupados == 0 || metadata.Size%globals.ClientConfig.DialfsBlockSize != 0 {
		bloquesOcupados++
	}

	bloquesNecesarios := request.Tamaño / globals.ClientConfig.DialfsBlockSize
	if bloquesNecesarios == 0 || request.Tamaño%globals.ClientConfig.DialfsBlockSize != 0 {
		bloquesNecesarios++
	}

	bloquesLibres := 0

	for i := 0; i < globals.ClientConfig.DialfsBlockCount; i++ {
		if bitmap[i] == 0 {
			bloquesLibres++
		}
	}

	if bloquesLibres < bloquesNecesarios-bloquesOcupados {
		log.Printf("No hay bloques suficientes para el archivo")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if bloquesNecesarios-bloquesOcupados >= 0 {
		bitmap = OcuparEspacioLibreContiguo(bitmap, bloquesOcupados, bloquesNecesarios, metadataPath)
	} else {
		bitmap = liberarBloques(bitmap, metadata.InitialBlock+bloquesOcupados, bloquesOcupados-bloquesNecesarios)
	}

	log.Printf("\n%b", bitmap)

	metadata = obtenerMetadata(globals.ClientConfig.DialfsPath + "/" + request.NombreArchivo + ".json")
	metadata.Size = request.Tamaño

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("no se pudo codificar la metadata: %s", err.Error())
	}

	err = os.WriteFile(globals.ClientConfig.DialfsPath+"/"+request.NombreArchivo+".json", metadataBytes, 0644)
	if err != nil {
		log.Printf("no se pudo crear el archivo de metadata: %s", err.Error())
	}

	ModificarBitmap(bitmap)
	respuesta, err := json.Marshal("OK")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	//time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func obtenerMetadata(filePath string) Metadata {
	var metadata Metadata
	metadataFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer metadataFile.Close()

	jsonParser := json.NewDecoder(metadataFile)
	jsonParser.Decode(&metadata)

	return metadata
}

func writeToFile(filename string, ptr int, data []byte) error {
	metadata := obtenerMetadata(globals.ClientConfig.DialfsPath + "/" + filename + ".json")

	file, err := os.OpenFile(globals.ClientConfig.DialfsPath+"/bloques.dat", os.O_RDWR, 0644)
	check(err)
	defer file.Close()

	_, err = file.WriteAt(data, int64(metadata.InitialBlock*globals.ClientConfig.DialfsBlockSize)+int64(ptr))
	check(err)

	return nil
}

func readFromFile(filename string, ptr int, size int) []byte {
	metadata := obtenerMetadata(globals.ClientConfig.DialfsPath + "/" + filename + ".json")

	textoLeido := make([]byte, size)

	file, err := os.OpenFile(globals.ClientConfig.DialfsPath+"/bloques.dat", os.O_RDWR, 0644)
	check(err)
	defer file.Close()

	_, err = file.ReadAt(textoLeido, int64(metadata.InitialBlock*globals.ClientConfig.DialfsBlockSize)+int64(ptr))
	check(err)

	return textoLeido
}

func IO_FS_WRITE(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Entro al fs write IO")
	var request BodyRequestFS
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Error al decodificar la solicitud", http.StatusBadRequest)
		return
	}

	if request.Archivo == "" {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	var requestBody = BodyEscritura{
		PID:       request.PID,
		Tamaño:    request.Tamaño,
		Direccion: request.Direccion,
	}

	metadata := obtenerMetadata(globals.ClientConfig.DialfsPath + "/" + request.Archivo + ".json")

	if request.PtrArchivo+request.Tamaño >= metadata.Size {
		log.Printf("Error: Intentando leer/escribir más allá del tamaño del archivo")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Error al codificar la solicitud %v", err)
		return
	}
	fmt.Printf("\nLlamando a memoria para leer la direc %d, tamaño %d", request.Direccion, request.Tamaño)
	url := "http://localhost:" + strconv.Itoa(globals.ClientConfig.PortMemory) + "/leer"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando: %s", err.Error())
		return
	}
	response := make([]byte, 0, request.Tamaño)
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		return
	}
	//time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
	respString := string(response)

	log.Printf("El texto leido es: %s", respString)
	log.Printf("Escribo el texto en el archivo")
	writeToFile(request.Archivo, request.PtrArchivo, response)
	log.Printf("PID: %d - Operación: IO_FS_WRITE", request.PID)
	log.Printf("PID: %d - Escribir Archivo: %s - Tamaño a Escribir:  %d - Puntero Archivo: %d", request.PID, request.Archivo, request.Tamaño, request.PtrArchivo)
	respuesta, err := json.Marshal("OK")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func IO_FS_READ(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\nEntro al fs read IO")
	var request BodyRequestFS
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Error al decodificar la solicitud", http.StatusBadRequest)
		return
	}

	if request.Archivo == "" {
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}
	//Leer del archivo desde el ptr indicado
	metadata := obtenerMetadata(globals.ClientConfig.DialfsPath + "/" + request.Archivo + ".json")

	if request.PtrArchivo+request.Tamaño >= metadata.Size {
		log.Printf("Error: Intentando leer/escribir más allá del tamaño del archivo")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	textoLeido := readFromFile(request.Archivo, request.PtrArchivo, request.Tamaño)

	log.Printf("El texto leido es: %s", string(textoLeido))
	fmt.Print("\nGuardando texto en memoria... ")
	var requestBody = BodyEscritura{
		PID:       request.PID,
		Info:      textoLeido,
		Tamaño:    request.Tamaño,
		Direccion: request.Direccion,
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
	//time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
	log.Printf("PID: %d - Operación: IO_FS_READ", request.PID)
	log.Printf("PID: %d - Leer Archivo: %s - Tamaño a Leer: %d - Puntero Archivo: %d ", request.PID, request.Archivo, request.Tamaño, request.PtrArchivo)
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

type Metadata struct {
	InitialBlock int `json:"initial_block"`
	Size         int `json:"size"`
}

type CreateFileRequest struct {
	Interfaz      string `json:"interfaz"`
	NombreArchivo string `json:"nombre_archivo"`
}

type ConfigResponse struct {
	Path       string `json:"path"`
	BlockCount int    `json:"block_count"`
	BlockSize  int    `json:"block_size"`
}

type BodyFileRequest struct {
	PID           int    `json:"pid"`
	NombreArchivo string `json:"nombre_archivo"`
}

type FileResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func IO_FS_CREATE_Handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Se recibió una solicitud en IO_FS_CREATE_Handler")

	var request BodyFileRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("Error al decodificar la solicitud: %v", err)
		http.Error(w, "Error al decodificar la solicitud", http.StatusBadRequest)
		return
	}

	if request.NombreArchivo == "" {
		log.Printf("Parámetros inválidos: nombreArchivo está vacío")
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	log.Printf("Intentando crear archivo con nombre: %s", request.NombreArchivo)

	err = CrearArchivoFS(request.NombreArchivo)
	if err != nil {
		log.Printf("Error al crear archivo: %v", err)
		response := FileResponse{
			Status:  "Error",
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Responder con éxito
	response := FileResponse{
		Status:  "OK",
		Message: "Archivo creado correctamente",
	}
	//time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	log.Printf("Archivo '%s' creado", request.NombreArchivo)
	log.Printf("PID: %d - Crear Archivo: %s", request.PID, request.NombreArchivo)
}

func IO_FS_DELETE_Handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Se recibió una solicitud en IO_FS_DELETE_Handler")

	var request BodyFileRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		log.Printf("Error al decodificar la solicitud: %v", err)
		http.Error(w, "Error al decodificar la solicitud", http.StatusBadRequest)
		return
	}

	if request.NombreArchivo == "" {
		log.Printf("Parámetros inválidos: nombreArchivo está vacío")
		http.Error(w, "Parámetros inválidos", http.StatusBadRequest)
		return
	}

	log.Printf("Intentando eliminar archivo con nombre: %s", request.NombreArchivo)

	err = EliminarArchivoFS(request.NombreArchivo)
	if err != nil {
		log.Printf("Error al eliminar archivo: %v", err)
		response := FileResponse{
			Status:  "Error",
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Responder con éxito
	response := FileResponse{
		Status:  "OK",
		Message: "Archivo eliminado correctamente",
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
	//time.Sleep(time.Duration(globals.ClientConfig.UnitWorkTime) * time.Millisecond)
	log.Printf("Archivo '%s' eliminado", request.NombreArchivo)
	log.Printf("PID: %d - Delete Archivo: %s", request.PID, request.NombreArchivo)
}

func CrearArchivoFS(nombreArchivo string) error {
	log.Printf("Se entró a CrearArchivoFS con nombreArchivo: %s", nombreArchivo)

	// Ruta al archivo de metadata
	metadataPath := globals.ClientConfig.DialfsPath + "/" + nombreArchivo + ".json"

	// Verificar si el archivo ya existe
	if _, err := os.Stat(metadataPath); err == nil {
		return fmt.Errorf("el archivo ya existe")
	}

	log.Printf("Leyendo archivos de bloques y bitmap")

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

	log.Printf("Leyendo el bitmap")

	// Leer el bitmap
	bitmap := make([]byte, globals.ClientConfig.DialfsBlockCount)
	_, err = bitmapFile.Read(bitmap)
	if err != nil {
		return fmt.Errorf("no se pudo leer el bitmap: %s", err.Error())
	}

	log.Printf("Buscando bloque libre")

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

	log.Printf("Bloque libre encontrado: %d", initialBlock)

	// Marcar el bloque como ocupado en el bitmap
	bitmap[initialBlock] = 1
	_, err = bitmapFile.WriteAt(bitmap, 0)
	if err != nil {
		return fmt.Errorf("no se pudo actualizar el bitmap: %s", err.Error())
	}

	log.Printf("Actualización del bitmap completada")

	// Crear el archivo de metadata
	metadata := Metadata{
		InitialBlock: initialBlock,
		Size:         0,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("no se pudo codificar la metadata: %s", err.Error())
	}

	err = os.WriteFile(metadataPath, metadataBytes, 0644)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo de metadata: %s", err.Error())
	}

	tablaSegmentacion[initialBlock] = nombreArchivo

	log.Printf("Archivo de metadata '%s' creado con éxito", nombreArchivo)
	return nil
}

func EliminarArchivoFS(nombreArchivo string) error {
	// Ruta al archivo de metadata
	metadataPath := fmt.Sprintf("%s/%s.json", globals.ClientConfig.DialfsPath, nombreArchivo)

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

	bloquesOcupados := metadata.Size / globals.ClientConfig.DialfsBlockSize
	if bloquesOcupados == 0 || metadata.Size%globals.ClientConfig.DialfsBlockSize != 0 {
		bloquesOcupados++
	}

	// Marcar el bloque como libre en el bitmap
	initialBlock := metadata.InitialBlock
	for i := 0; i < bloquesOcupados; i++ {
		bitmap[initialBlock+i] = 0
	}

	fmt.Printf("%b", bitmap)

	err = os.WriteFile(bitmapPath, bitmap, 0644)
	if err != nil {
		return fmt.Errorf("error al actualizar el bitmap: %v", err)
	}

	// Eliminar el archivo de metadata
	err = os.Remove(metadataPath)
	if err != nil {
		return fmt.Errorf("error al eliminar el archivo de metadata: %v", err)
	}

	delete(tablaSegmentacion, initialBlock)

	return nil
}
