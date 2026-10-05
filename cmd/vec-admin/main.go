package main

import (
	"errors"
	"fmt"
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
		log.Fatal(errorArranque("retirada"))
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
		log.Fatal(errorArranque("perfiles_config"))
	}
	runtime, err := cargarConfiguracionRuntimeADMIN(os.Getenv("VEC_ADMIN_RUNTIME_CONFIG_FILE"), privada)
	if err != nil {
		log.Fatal(errorArranque("runtime_config"))
	}
	var servidor *http.Server
	var cerrar func()
	if rutaUsuarios := os.Getenv("VEC_ADMIN_USUARIOS_CONFIG_FILE"); rutaUsuarios != "" {
		// El modo lo fija el archivo privado cerrado; una configuración inválida
		// nunca cae al arranque heredado de perfiles.
		usuarios, errorConfig := cargarConfiguracionUsuariosMetadatosPrivada(rutaUsuarios, privada)
		if errorConfig != nil {
			log.Fatal(errorArranque("usuarios_config"))
		}
		// El lote sólo existe junto a las lecturas de usuarios y con su propio
		// archivo privado; sin él, el proceso no abre ninguna escritura.
		var lote *configuracionLotePrivada
		if rutaLote := os.Getenv("VEC_ADMIN_LOTE_CONFIG_FILE"); rutaLote != "" {
			c, errorLote := cargarConfiguracionLotePrivada(rutaLote, privada, usuarios, runtime)
			if errorLote != nil {
				log.Fatal(errorArranque("lote_config"))
			}
			lote = &c
		}
		// El gobierno del plan de firma sólo existe junto al lote (comparte su
		// superficie de escritura) y con su propio archivo privado.
		var plan *configuracionPlanFirmaPrivada
		if rutaPlan := os.Getenv("VEC_ADMIN_PLAN_FIRMA_CONFIG_FILE"); rutaPlan != "" {
			if lote == nil {
				log.Fatal(errorArranque("plan_firma_sin_lote"))
			}
			c, errorPlan := cargarConfiguracionPlanFirmaPrivada(rutaPlan, *lote, privada, usuarios, runtime)
			if errorPlan != nil {
				log.Fatal(errorArranque("plan_firma_config"))
			}
			plan = &c
		}
		servidor, cerrar, err = componerProcesoUsuariosMetadatosADMINConLote(configServidor, privada, usuarios, runtime, lote, plan)
	} else if os.Getenv("VEC_ADMIN_LOTE_CONFIG_FILE") != "" || os.Getenv("VEC_ADMIN_PLAN_FIRMA_CONFIG_FILE") != "" {
		log.Fatal(errorArranque("lote_sin_usuarios"))
	} else {
		servidor, cerrar, err = componerProcesoADMINConRuntime(configServidor, privada, runtime)
	}
	if err != nil {
		// La composición ya indica su etapa (por ejemplo, etapa=lector_usuarios).
		log.Fatal(err)
	}
	defer cerrar()
	if err := servidor.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(errorArranque("escucha"))
	}
}

// errorArranque conserva ErrConfiguracion y añade la etapa que falló. Es un
// código fijo del programa: nunca lleva rutas, valores de configuración,
// mensajes de PostgreSQL ni secretos, sólo dice al operador dónde mirar.
func errorArranque(etapa string) error {
	return fmt.Errorf("%w: etapa=%s", administracion.ErrConfiguracion, etapa)
}
