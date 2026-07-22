package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"vec-diputacion-granada/internal/app/composicion/interna"
)

var (
	errArranqueComposicion = errors.New("servidor interno: composicion no disponible")
	errArranqueEscucha     = errors.New("servidor interno: escucha TLS no disponible")
)

func main() {
	if err := ejecutar(); err != nil {
		log.Fatal(err)
	}
}

func ejecutar() error {
	cfg := interna.CargarConfiguracion()
	servidor, err := interna.NuevoServidor(cfg)
	if err != nil {
		return errArranqueComposicion
	}
	if err := validarServidorParaEscucha(servidor); err != nil {
		return errArranqueComposicion
	}

	// net/http incluye direcciones remotas y errores TLS crudos en ErrorLog.
	// La raiz cerrada no publica esos datos mediante el cmd.
	servidor.ErrorLog = log.New(nuevoEscritorEventosHTTPSaneados(os.Stderr), "", 0)
	log.Print("servidor interno VEC iniciando escucha TLS mutua")
	err = servidor.ListenAndServeTLS("", "")
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errArranqueEscucha
	}
	return nil
}

func validarServidorParaEscucha(servidor *http.Server) error {
	return interna.ValidarServidorParaEscucha(servidor)
}
