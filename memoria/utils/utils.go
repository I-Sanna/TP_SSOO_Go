package utils

import (
	"encoding/json"
	"io"
	"log"
	"memoria/globals"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

var memoria []byte
var bitArray []int

var instruccionesProcesos map[int][]string
var tablasPaginasProcesos map[int]map[int]int

type BodyRequest struct {
	PID  int    `json:"pid"`
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
	memoria = make([]byte, globals.ClientConfig.MemorySize)
	bitArray = make([]int, globals.ClientConfig.MemorySize/globals.ClientConfig.PageSize)
	instruccionesProcesos = make(map[int][]string)
	tablasPaginasProcesos = make(map[int]map[int]int)
}

func readFile(fileName string) []string {
	file, err := os.Open(globals.ClientConfig.InstructionsPath + fileName) // For read access.
	if err != nil {
		log.Fatal(err)
	}
	fileData, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}
	data := make([]byte, fileData.Size())
	_, err = file.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	strData := string(data)
	instrucciones := strings.Split(strings.TrimRight(strData, "\x00"), "\n")

	return instrucciones
}

func BuscarMarco(w http.ResponseWriter, r *http.Request) {
	delayMemoria()
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

	marco, ok = tablasPaginasProcesos[pidInt][numeroPagina]
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
	delayMemoria()
	pid := r.PathValue("pid")
	tam := r.PathValue("tamaño")

	pidInt, err := strconv.Atoi(pid)

	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	tamaño, err := strconv.Atoi(tam)

	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	cantidadPaginas := tamaño / globals.ClientConfig.PageSize //Se asume que la instruccion RESIZE pasa un valor divisible por el tamaño de las paginas
	if tamaño%globals.ClientConfig.PageSize != 0 {
		cantidadPaginas++
	}

	tablaPaginas := tablasPaginasProcesos[pidInt]

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
				w.WriteHeader(http.StatusInsufficientStorage) // Si no devuelve http.StatusOk a la CPU es que se quedo sin memoria
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
	delayMemoria()
	var request BodyRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tablaPaginas := make(map[int]int)

	instrucciones := readFile(request.Path)
	instruccionesProcesos[request.PID] = instrucciones
	tablasPaginasProcesos[request.PID] = tablaPaginas

	log.Printf("PID: %d - Tamaño: %d", request.PID, globals.ClientConfig.MemorySize/globals.ClientConfig.PageSize)

	w.WriteHeader(http.StatusOK)
}

func DevolverInstruccion(w http.ResponseWriter, r *http.Request) {
	delayMemoria()
	pid := r.PathValue("pid")
	pc := r.PathValue("pc")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	subindice, err := strconv.Atoi(pc)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	respuesta, err := json.Marshal(instruccionesProcesos[pidInt][subindice])
	if err != nil {
		http.Error(w, "Error al codificar los datos como JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func LiberarRecursos(w http.ResponseWriter, r *http.Request) {
	delayMemoria()
	pid := r.PathValue("pid")

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		http.Error(w, "Error al convertir de json a Int", http.StatusInternalServerError)
		return
	}

	delete(instruccionesProcesos, pidInt)
	liberarPaginas(tablasPaginasProcesos[pidInt])
	delete(tablasPaginasProcesos, pidInt)

	w.WriteHeader(http.StatusOK)
}

func liberarPaginas(paginas map[int]int) {
	for _, value := range paginas {
		bitArray[value] = 0
	}
}

type BodyEscritura struct {
	PID       int    `json:"pid"`
	Info      []byte `json:"info"`
	Tamaño    int    `json:"tamaño"`
	Direccion int    `json:"direccion"`
}

func EscribirMemoria(w http.ResponseWriter, r *http.Request) {
	delayMemoria()
	var request BodyEscritura

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	marco := request.Direccion / globals.ClientConfig.PageSize
	desplazamiento := request.Direccion % globals.ClientConfig.PageSize
	infoBytes := make([]byte, request.Tamaño)
	copy(infoBytes[0:len(request.Info)], request.Info)

	log.Printf("PID: %d - Accion: Escribir - Direccion fisica: %d - Tamaño a escribir: %d", request.PID, request.Direccion, request.Tamaño)

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

	inicio := false
	fin := false
	tablaProceso := tablasPaginasProcesos[request.PID]
	marcosModificados := 0
	for pagina := 0; pagina < len(tablaProceso); pagina++ {
		if tablaProceso[pagina] == marco {
			inicio = true
			// Interpreto que la info se carga en paginas contiguas y que no vuelvo a la primera pagina si llego a la ultima
			if len(tablaProceso)-pagina < marcosNecesarios {
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
			llenarPagina(tablaProceso[pagina], desplazamiento, infoBytesArray[marcosModificados])
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
	delayMemoria()
	var request BodyEscritura

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	marco := request.Direccion / globals.ClientConfig.PageSize
	desplazamiento := request.Direccion % globals.ClientConfig.PageSize
	infoBytes := make([]byte, 0, request.Tamaño)

	log.Printf("PID: %d - Accion: Leer - Direccion fisica: %d - Tamaño a leer: %d", request.PID, request.Direccion, request.Tamaño)

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

	inicio := false
	fin := false
	tamañoRestante := request.Tamaño
	var listaBytes []byte
	tablaProceso := tablasPaginasProcesos[request.PID]

	for pagina := 0; pagina < len(tablaProceso); pagina++ {
		if tablaProceso[pagina] == marco {
			inicio = true
			// Interpreto que la info se carga en paginas contiguas y que no vuelvo a la primera pagina si llego a la ultima
			if len(tablaProceso)-pagina < marcosNecesarios {
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
				listaBytes = leerPagina(tablaProceso[pagina], desplazamiento, tamañoRestante)
				fin = true
			} else {
				listaBytes = leerPagina(tablaProceso[pagina], desplazamiento, globals.ClientConfig.PageSize-desplazamiento)
				tamañoRestante = tamañoRestante - (globals.ClientConfig.PageSize - desplazamiento)
				desplazamiento = 0
			}
			infoBytes = slices.Concat(infoBytes, listaBytes)
		}
		if fin {
			break
		}
	}

	respuesta, err := json.Marshal(infoBytes)
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

func PageSize(w http.ResponseWriter, r *http.Request) {
	delayMemoria()

	respuesta, err := json.Marshal(globals.ClientConfig.PageSize)
	if err != nil {
		http.Error(w, "Error al codificar el tamaño de la página como JSON", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(respuesta)
}

func delayMemoria() { //Retardo memoria ante cada peticion
	time.Sleep(time.Duration(globals.ClientConfig.DelayResponse) * time.Millisecond)
}
