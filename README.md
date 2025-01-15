# TP SSOO UTN 1c2024

Durante este cuatrimestre se desarrollo una API que simula los siguientes componentes de una computadora y sus comportamientos:

- Kernel
- Memoria
- CPU
- I/O

##Dependencias

Lo única dependencia es la instalacion del lenguaje [Golang](https://go.dev/doc/install)

##Ejecución

Para ejecutar el código, cada carpeta contendrá el archivo necesario para levantar el Sistema Operativo y deberá ser realizado en el siguiente orden:

```
go run kernel.go
go run cpu.go
go run memoria.go
go run entradasalida.go
```

___Importante___: Por cada comando deberá haber una instancia de la consola ejecutandoló

##Requests API

___Aclaraciónes:___ Las instrucciones están almacenadas como archivos *.txt en la carpeta de memoria y los archivos de configuración en la carpeta de entradasalida como *.json

###Iniciar Proceso (Kernel)

```golang
PUT /process

{
"pid": 0
"path": "instrucciones.txt"
}
```

###Finalizar Proceso (Kernel)
```
DELETE /process/{pid}
```

###Listar Procesos(Kernel)
```
GET /process
```

Los request mencionados son algunos ejemplos para poder visualizar el funcionamiento del programa. El resto de ellos se encuentra en la documentación

##Documentación

La consigna del trabajo práctico junto con las limitaciones del mismo se encuentra en el repositorio bajo el nombre de "A.L.GO -v1.1.pdf"
