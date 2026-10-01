package main

import (
	"bytes"
	"net/http"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

const rutaAdjudicacionesLocales = "/api/provision/v1/adjudicaciones-locales"
const rutaSimularAdjudicacion = rutaAdjudicacionesLocales + "/simulaciones"

func configurarAdjudicacionesLocales(w http.ResponseWriter) {
	catalogo, err := leerCatalogoEnsayos()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	p, err := simulacion.EjemploAdjudicacion()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	type preferencia struct {
		VacanteRef string    `json:"vacante_ref"`
		Orden      int       `json:"orden"`
		Total      *b.Puntos `json:"total"`
	}
	type solicitud struct {
		PersonaRef   string        `json:"persona_ref"`
		Preferencias []preferencia `json:"preferencias"`
	}
	type resumen struct {
		Vacantes    []domain.VacanteAdjudicacion `json:"vacantes"`
		Solicitudes []solicitud                  `json:"solicitudes"`
	}
	r := resumen{Vacantes: p.Entrada.Vacantes, Solicitudes: make([]solicitud, 0, len(p.Entrada.Solicitudes))}
	for _, s := range p.Entrada.Solicitudes {
		ps := make([]preferencia, 0, len(s.Preferencias))
		for orden, pref := range s.Preferencias {
			var total *b.Puntos
			if pref.Valoracion != nil {
				total = pref.Valoracion.Total
			}
			ps = append(ps, preferencia{pref.VacanteRef, orden + 1, total})
		}
		r.Solicitudes = append(r.Solicitudes, solicitud{s.PersonaRef, ps})
	}
	type ejemplo struct {
		EjemploRef    string                           `json:"ejemplo_ref"`
		Configuracion domain.ConfiguracionAdjudicacion `json:"configuracion"`
		Resumen       resumen                          `json:"resumen"`
	}
	responderEnsayo(w, struct {
		Ejemplos []ejemplo `json:"ejemplos"`
	}{[]ejemplo{{catalogo.AdjudicacionRef, p.Configuracion, r}}})
}

// El navegador configura una política de ensayo. Las solicitudes, sus hechos y
// resultados de valoración permanecen fijados en el servidor sintético.
func simularAdjudicacionLocal(w http.ResponseWriter, datos []byte) {
	var s struct {
		EjemploRef    string                           `json:"ejemplo_ref"`
		Configuracion domain.ConfiguracionAdjudicacion `json:"configuracion"`
	}
	if err := simulacion.Decodificar(bytes.NewReader(datos), &s); err != nil {
		responderError(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	catalogo, err := leerCatalogoEnsayos()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	if s.EjemploRef != catalogo.AdjudicacionRef {
		responderError(w, http.StatusBadRequest, "ejemplo_no_admitido")
		return
	}
	p, err := simulacion.EjemploAdjudicacion()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	r, err := application.SimularAdjudicacion(s.Configuracion, p.Entrada)
	if err != nil {
		responderError(w, http.StatusUnprocessableEntity, "preparacion_invalida")
		return
	}
	responderEnsayo(w, r)
}
