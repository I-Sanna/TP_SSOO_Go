package main

import (
	"entradasalida/globals"
	"entradasalida/utils"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

func main() {
	utils.ConfigurarLogger()

	fmt.Print("Ingrese el nombre del dispositivo: ")
	var nombreDispositivo string = utils.LeerConsola()

	fmt.Print("Ingrese el archivo de configuracion del dispositivo: ")
	var configDispositivo string = utils.LeerConsola()

	globals.ClientConfig = utils.IniciarConfiguracion(configDispositivo)

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	} else {
		log.Printf("\nConfiguracion cargada con exito!\n")
	}

	utils.EstablecerConexion(nombreDispositivo, globals.ClientConfig.Port)

	http.HandleFunc("GET /sleep/{units}/{pid}", utils.IO_GEN_SLEEP)
	http.HandleFunc("GET /read/{pid}/{tamaño}/{direccion}", utils.IO_STDIN_READ)
	http.HandleFunc("GET /write/{pid}/{tamaño}/{direccion}", utils.IO_STDOUT_WRITE)
	http.HandleFunc("GET /validar", utils.ValidarConexion)

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.Port), nil))
}
