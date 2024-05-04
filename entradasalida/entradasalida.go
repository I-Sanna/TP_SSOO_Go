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

	fmt.Print("Ingrese el archivo de configuracion del dispositivo: ")
	var configDispositivo string = utils.LeerConsola()

	globals.ClientConfig = utils.IniciarConfiguracion(configDispositivo)

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}

	http.HandleFunc("PUT /sleep/{units}", utils.IO_GEN_SLEEP)

	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(globals.ClientConfig.Port), nil))
}
