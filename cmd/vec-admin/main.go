package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
)

func main() {
	retirada, err := time.Parse(time.RFC3339, os.Getenv("VEC_ADMIN_RETIRADA_EN"))
	if err != nil {
		log.Fatal(administracion.ErrConfiguracion)
	}
	configServidor := administracion.Configuracion{
		Entorno:             os.Getenv("VEC_ADMIN_ENTORNO"),
		Escucha:             os.Getenv("VEC_ADMIN_ESCUCHA"),
		Host:                os.Getenv("VEC_ADMIN_HOST"),
		Audiencia:           os.Getenv("VEC_ADMIN_AUDIENCIA"),
		EmisorIdentidad:     os.Getenv("VEC_ADMIN_EMISOR_IDENTIDAD"),
		CertificadoServidor: os.Getenv("VEC_ADMIN_TLS_CERT_FILE"),
		ClaveServidor:       os.Getenv("VEC_ADMIN_TLS_KEY_FILE"),
		CAAdministracion:    os.Getenv("VEC_ADMIN_CA_FILE"),
		CRLAdministracion:   os.Getenv("VEC_ADMIN_CRL_FILE"),
		RedesPermitidas:     strings.Split(os.Getenv("VEC_ADMIN_REDES_PERMITIDAS"), ","),
		RetiradaEn:          retirada,
	}
	privada, err := cargarConfiguracionPerfilesPrivada(os.Getenv("VEC_ADMIN_PERFILES_CONFIG_FILE"))
	if err != nil {
		log.Fatal(administracion.ErrConfiguracion)
	}
	var servidor *http.Server
	var cerrar func()
	if rutaUsuarios := os.Getenv("VEC_ADMIN_USUARIOS_CONFIG_FILE"); rutaUsuarios != "" {
		// El modo lo fija el archivo privado cerrado; una configuración inválida
		// nunca cae al arranque heredado de perfiles.
		usuarios, errorConfig := cargarConfiguracionUsuariosMetadatosPrivada(rutaUsuarios, privada)
		if errorConfig != nil {
			log.Fatal(administracion.ErrConfiguracion)
		}
		servidor, cerrar, err = componerProcesoUsuariosMetadatosADMIN(configServidor, privada, usuarios)
	} else {
		servidor, cerrar, err = componerProcesoADMIN(configServidor, privada)
	}
	if err != nil {
		log.Fatal(administracion.ErrConfiguracion)
	}
	defer cerrar()
	if err := servidor.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(administracion.ErrConfiguracion)
	}
}
