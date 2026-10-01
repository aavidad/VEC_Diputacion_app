package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"log/slog"
	"net/http"

	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

const rutaProcesosLocales = "/api/provision/v1/procesos-locales"
const rutaSimulacionProceso = rutaProcesosLocales + "/simulaciones"

//go:embed provision_presentacion.json
var presentacionProcesoJSON []byte

type proyeccionPuestoProceso struct {
	DenominacionClave string `json:"denominacion_clave"`
	CentroClave       string `json:"centro_clave"`
}

type presentacionProceso struct {
	EjemploRef string `json:"ejemplo_ref"`
	Proyeccion struct {
		DenominacionClave string                             `json:"denominacion_clave"`
		Puestos           map[string]proyeccionPuestoProceso `json:"puestos"`
	} `json:"proyeccion"`
}

func leerPresentacionProceso() (presentacionProceso, error) {
	var p presentacionProceso
	err := simulacion.Decodificar(bytes.NewReader(presentacionProcesoJSON), &p)
	return p, err
}

// Sólo proyecta la oferta del ejercicio. El navegador no suministra los hechos
// de Personal, los requisitos comprobados ni una identidad institucional.
func configurarProcesosLocales(w http.ResponseWriter) {
	p, err := simulacion.EjemploProceso()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	proyeccion, err := leerPresentacionProceso()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	type ejemploPublico struct {
		EjemploRef   string                        `json:"ejemplo_ref"`
		Proceso      domain.ProcesoProvision       `json:"proceso"`
		Preferencias []domain.PreferenciaProvision `json:"preferencias"`
		Proyeccion   any                           `json:"proyeccion"`
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Ejemplos []ejemploPublico `json:"ejemplos"`
	}{[]ejemploPublico{{proyeccion.EjemploRef, p.Proceso, p.Solicitud.Preferencias, proyeccion.Proyeccion}}}); err != nil {
		slog.Warn("provision_respuesta_no_entregada", "operacion", "configuracion_local")
		return
	}
}

func simularProcesoLocal(w http.ResponseWriter, datos []byte) {
	var solicitud struct {
		EjemploRef    string                        `json:"ejemplo_ref"`
		Configuracion domain.Configuracion          `json:"configuracion"`
		Preferencias  []domain.PreferenciaProvision `json:"preferencias"`
	}
	if err := simulacion.Decodificar(bytes.NewReader(datos), &solicitud); err != nil {
		responderError(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	proyeccion, err := leerPresentacionProceso()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	if solicitud.EjemploRef != proyeccion.EjemploRef {
		responderError(w, http.StatusBadRequest, "ejemplo_no_admitido")
		return
	}
	p, err := simulacion.EjemploProceso()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	// La configuración cambia el ensayo; la fuente y versión de los hechos
	// permanecen ligadas al ejemplo sintético embebido en el servidor.
	p.Proceso.Configuracion = solicitud.Configuracion
	p.Solicitud.VersionReglas = solicitud.Configuracion.Version
	p.Solicitud.Preferencias = solicitud.Preferencias
	seleccionados := make(map[string]bool, len(solicitud.Preferencias))
	for _, preferencia := range solicitud.Preferencias {
		seleccionados[preferencia.PuestoRef] = true
	}
	valoraciones := make([]domain.EntradaValoracionPuesto, 0, len(p.Solicitud.Valoraciones))
	for _, valoracion := range p.Solicitud.Valoraciones {
		if seleccionados[valoracion.PuestoRef] {
			valoraciones = append(valoraciones, valoracion)
		}
	}
	p.Solicitud.Valoraciones = valoraciones
	resultado, err := application.SimularProceso(p)
	if err != nil {
		responderError(w, http.StatusUnprocessableEntity, "preparacion_invalida")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resultado); err != nil {
		slog.Warn("provision_respuesta_no_entregada", "operacion", "simulacion_local")
		return
	}
}
