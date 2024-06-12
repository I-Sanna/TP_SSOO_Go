package utils

import (
	"encoding/json"
	"io"
	"log"
	"memoria/globals"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var memoria []byte
var bitArray []int

var contadorPID int
var listaPID []int
var instruccionesProcesos [][]string
var tablasPaginasProcesos []map[int]int

type BodyRequest struct {
	Path string `json:"path"`
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

func ConfigurarLogger() {
	logFile, err := os.OpenFile("logs/memoria.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func InicializarMemoriaYTablas() {
	contadorPID = 0
	memoria = make([]byte, globals.ClientConfig.MemorySize)
	bitArray = make([]int, globals.ClientConfig.MemorySize/globals.ClientConfig.PageSize)
	//tablaPaginas = make(map[int]int)
}

func readFile(fileName string) []string {
	file, err := os.Open(fileName) // For read access.
	if err != nil {
		log.Fatal(err)
	}
	data := make([]byte, 300)
	_, err = file.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	strData := string(data)
	instrucciones := strings.Split(strings.TrimRight(strData, "\x00"), "\n")

	return instrucciones
}

func BuscarMarco(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	pagina := r.PathValue("pagina")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}

	numeroPagina, err := strconv.Atoi(pagina)
	if err != nil {
		http.Error(w, "Error al transformar un string en int", http.StatusInternalServerError)
		return
	}

	var marco int
	var ok bool

	for index, valor := range listaPID {
		if valor == pidInt {
			marco, ok = tablasPaginasProcesos[index][numeroPagina]
			if !ok {
				http.Error(w, "Error: la pagina buscada no existe", http.StatusInternalServerError)

				respuesta, err := json.Marshal(marco)
				if err != nil {
					http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write(respuesta)
				return
			}
		}
	}

	respuesta, err := json.Marshal(marco)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	log.Printf("PID: %d - Pagina: %d - Marco: %d", pidInt, numeroPagina, marco)
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func ReservarMemoria(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	tam := r.PathValue("tamaño")

	pidInt, err := strconv.Atoi(pid)
	log.Printf("%s %d\n\n\n\n", pid, pidInt)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	tamaño, err := strconv.Atoi(tam)
	log.Printf("%s %d\n\n\n\n", tam, tamaño)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	cantidadPaginas := tamaño / globals.ClientConfig.PageSize //Se asume que la instruccion RESIZE pasa un valor divisible por el tamaño de las paginas

	var tablaPaginas map[int]int
	log.Printf("hola")
	index := obtenerIndexProceso(pidInt)
	log.Printf("chau %d", index)
	tablaPaginas = tablasPaginasProcesos[index]

	if tamaño > len(tablaPaginas)*globals.ClientConfig.PageSize {
		log.Printf("PID: %d - Tamaño Actual: %d - Tamaño a Ampliar: %d", pidInt, len(tablaPaginas)*globals.ClientConfig.PageSize, tamaño)
	} else {
		log.Printf("PID: %d - Tamaño Actual: %d - Tamaño a Reducir: %d", pidInt, len(tablaPaginas)*globals.ClientConfig.PageSize, tamaño)
	}

	paginasNecesarias := cantidadPaginas - len(tablaPaginas) //Si es positivo se agregan paginas, si es negativo es que se quitan

	if paginasNecesarias > 0 {
		for i := 0; i < paginasNecesarias; i++ {
			marcoLibre := obtenerMarcoLibre()

			if marcoLibre == -1 {
				http.Error(w, "Error: Out of memory", http.StatusInternalServerError)
				w.WriteHeader(http.StatusBadRequest) // Si no devuelve http.StatusOk a la CPU es que se quedo sin memoria
				return
			}

			bitArray[marcoLibre] = 1
			tablaPaginas[len(tablaPaginas)] = marcoLibre
		}
	} else if paginasNecesarias < 0 {
		for i := 0; i < Abs(paginasNecesarias); i++ {
			bitArray[tablaPaginas[len(tablaPaginas)-1]] = 0
			delete(tablaPaginas, len(tablaPaginas)-1)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func Abs(number int) int {
	if number < 0 {
		number = -number
	}
	return number
}

func obtenerMarcoLibre() int {
	for index, value := range bitArray {
		if value == 0 {
			return index
		}
	}
	return -1
}

func CrearProceso(w http.ResponseWriter, r *http.Request) {
	var request BodyRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tablaPaginas := make(map[int]int)

	instrucciones := readFile(request.Path)
	listaPID = append(listaPID, contadorPID)
	instruccionesProcesos = append(instruccionesProcesos, instrucciones)
	tablasPaginasProcesos = append(tablasPaginasProcesos, tablaPaginas)

	log.Printf("PID: %d - Tamaño: %d", contadorPID, globals.ClientConfig.MemorySize/globals.ClientConfig.PageSize)

	contadorPID++
	w.WriteHeader(http.StatusOK)
}

func DevolverInstruccion(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")
	pc := r.PathValue("pc")

	var indice int

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	for index, valor := range listaPID {
		if valor == pidInt {
			indice = index
			break
		}
	}

	subindice, err := strconv.Atoi(pc)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	respuesta, err := json.Marshal(instruccionesProcesos[indice][subindice])
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func LiberarRecursos(w http.ResponseWriter, r *http.Request) {
	pid := r.PathValue("pid")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	index := obtenerIndexProceso(pidInt)
	listaPID = removerIndexInt(listaPID, index)
	instruccionesProcesos = removerIndexString(instruccionesProcesos, index)
	liberarPaginas(tablasPaginasProcesos[index])
	tablasPaginasProcesos = removerIndexMap(tablasPaginasProcesos, index)

	w.WriteHeader(http.StatusOK)
}

func liberarPaginas(paginas map[int]int) {
	for _, value := range paginas {
		bitArray[value] = 0
	}
}

func removerIndexInt(s []int, index int) []int {
	ret := make([]int, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

func removerIndexString(s [][]string, index int) [][]string {
	ret := make([][]string, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

func removerIndexMap(s []map[int]int, index int) []map[int]int {
	ret := make([]map[int]int, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

type BodyEscritura struct {
	PID       int    `json:"pid"`
	Info      string `json:"info"`
	Tamaño    int    `json:"tamaño"`
	Direccion int    `json:"direccion"`
}

func EscribirMemoria(w http.ResponseWriter, r *http.Request) {
	var request BodyEscritura

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	marco := int(request.Direccion / globals.ClientConfig.PageSize)
	desplazamiento := request.Direccion % globals.ClientConfig.PageSize
	infoBytes := []byte(request.Info)

	log.Printf("PID: %d - Accion: Escribir - Direccion fisica: %d - Tamaño a escribir: %d", request.PID, request.Direccion, request.Tamaño)

	contador := len(infoBytes)
	marcosNecesarios := 0
	desplazamientoTemp := desplazamiento
	for contador > 0 {
		if desplazamiento != 0 {
			contador = contador - (globals.ClientConfig.PageSize - desplazamientoTemp)
			desplazamientoTemp = 0
		} else {
			contador = contador - globals.ClientConfig.PageSize
		}
		marcosNecesarios++
	}

	contador = 0
	infoBytesArray := make([][]byte, 0, marcosNecesarios)

	for i := 0; i < marcosNecesarios; i++ {
		if i == 0 {
			if request.Tamaño < globals.ClientConfig.PageSize-desplazamiento {
				infoBytesArray = append(infoBytesArray, infoBytes[0:request.Tamaño])
			} else {
				infoBytesArray = append(infoBytesArray, infoBytes[0:globals.ClientConfig.PageSize-desplazamiento])
			}
		} else if i == marcosNecesarios-1 {
			infoBytesArray = append(infoBytesArray, infoBytes[globals.ClientConfig.PageSize*i-desplazamiento:])
		} else {
			infoBytesArray = append(infoBytesArray, infoBytes[globals.ClientConfig.PageSize*i-desplazamiento:globals.ClientConfig.PageSize*(i+1)-desplazamiento])
		}
	}

	index := obtenerIndexProceso(request.PID)
	inicio := false
	fin := false
	marcosModificados := 0
	for key, value := range tablasPaginasProcesos[index] {
		if value == marco {
			inicio = true
			// Interpreto que la info se carga en paginas contiguas y que no vuelvo a la primera pagina si llego a la ultima
			if len(tablasPaginasProcesos[index])-key < marcosNecesarios {
				http.Error(w, "Error: no hay suficientes paginas contiguas", http.StatusInternalServerError)

				respuesta, err := json.Marshal("Error: Out of Memory")
				if err != nil {
					http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusBadRequest)
				w.Write(respuesta)
				return
			}
		}
		if inicio {
			llenarPagina(value, desplazamiento, infoBytesArray[marcosModificados])
			marcosModificados++
			desplazamiento = 0
			if marcosModificados == len(infoBytesArray) {
				fin = true
			}
		}
		if fin {
			break
		}
	}

	respuesta, err := json.Marshal("OK")
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func LeerMemoria(w http.ResponseWriter, r *http.Request) {
	var request BodyEscritura

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	marco := int(request.Direccion / globals.ClientConfig.PageSize)
	desplazamiento := request.Direccion % globals.ClientConfig.PageSize
	infoBytes := make([]byte, 0, request.Tamaño)

	log.Printf("PID: %d - Accion: Leer - Direccion fisica: %d - Tamaño a escribir: %d", request.PID, request.Direccion, request.Tamaño)

	contador := request.Tamaño
	marcosNecesarios := 0
	desplazamientoTemp := desplazamiento
	for contador > 0 {
		if desplazamiento != 0 {
			contador = contador - (globals.ClientConfig.PageSize - desplazamientoTemp)
			desplazamientoTemp = 0
		} else {
			contador = contador - globals.ClientConfig.PageSize
		}
		marcosNecesarios++
	}

	index := obtenerIndexProceso(request.PID)
	inicio := false
	fin := false
	tamañoRestante := request.Tamaño
	var listaBytes []byte

	for key, value := range tablasPaginasProcesos[index] {
		if value == marco {
			inicio = true
			// Interpreto que la info se carga en paginas contiguas y que no vuelvo a la primera pagina si llego a la ultima
			if len(tablasPaginasProcesos[index])-key < marcosNecesarios {
				http.Error(w, "Error: no hay suficientes paginas contiguas", http.StatusInternalServerError)

				respuesta, err := json.Marshal("Error: Out of Memory")
				if err != nil {
					http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusBadRequest)
				w.Write(respuesta)
				return
			}
		}
		if inicio {
			if tamañoRestante < globals.ClientConfig.PageSize-desplazamiento {
				listaBytes = leerPagina(value, desplazamiento, tamañoRestante)
				fin = true
			} else {
				listaBytes = leerPagina(value, desplazamiento, globals.ClientConfig.PageSize-desplazamiento)
				tamañoRestante = tamañoRestante - (globals.ClientConfig.PageSize - desplazamiento)
				desplazamiento = 0
			}
			infoBytes = append(infoBytes, listaBytes...)
		}
		if fin {
			break
		}
	}

	mensaje := string(infoBytes)

	log.Print(mensaje)

	respuesta, err := json.Marshal(mensaje)
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func llenarPagina(marco int, desplazamiento int, infoBytes []byte) {
	posicionInicial := marco*globals.ClientConfig.PageSize + desplazamiento
	copy(memoria[posicionInicial:posicionInicial+len(infoBytes)], infoBytes)
}

func leerPagina(marco int, desplazamiento int, tamaño int) []byte {
	var lista = make([]byte, tamaño)
	posicionInicial := marco*globals.ClientConfig.PageSize + desplazamiento
	copy(lista, memoria[posicionInicial:posicionInicial+tamaño])
	return lista
}

func obtenerIndexProceso(pid int) int {
	for index, value := range listaPID {
		if value == pid {
			return index
		}
	}
	return -1
}

func PageSize(w http.ResponseWriter, r *http.Request) {
	log.Printf("aa %d", globals.ClientConfig.PageSize)
	respuesta, err := json.Marshal(globals.ClientConfig.PageSize)
	if err != nil {
		http.Error(w, "Error al codificar el tamaño de la página como JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}
