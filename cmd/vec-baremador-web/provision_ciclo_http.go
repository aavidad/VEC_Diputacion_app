package main

import (
	"bytes"
	"net/http"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

const rutaCiclosLocales = "/api/provision/v1/ciclos-locales"
const rutaSimularCiclo = rutaCiclosLocales + "/simulaciones"

func configurarCiclosLocales(w http.ResponseWriter) {
	catalogo, err := leerCatalogoEnsayos()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	p, err := simulacion.EjemploCiclo()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	type ejemplo struct {
		EjemploRef    string                `json:"ejemplo_ref"`
		Configuracion domain.Configuracion  `json:"configuracion"`
		Catalogo      domain.CatalogoCausas `json:"catalogo_causas"`
		Casos         []casoCicloLocal      `json:"casos"`
	}
	responderEnsayo(w, struct {
		Ejemplos []ejemplo `json:"ejemplos"`
	}{[]ejemplo{{catalogo.CicloRef, p.Configuracion, p.CatalogoCausas, catalogo.Casos}}})
}

func simularCicloLocal(w http.ResponseWriter, datos []byte) {
	var s struct {
		EjemploRef string `json:"ejemplo_ref"`
		CasoRef    string `json:"caso_ref"`
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
	if s.EjemploRef != catalogo.CicloRef {
		responderError(w, http.StatusBadRequest, "ejemplo_no_admitido")
		return
	}
	var decision string
	for _, caso := range catalogo.Casos {
		if caso.CasoRef == s.CasoRef {
			decision = caso.Decision
			break
		}
	}
	p, err := simulacion.EjemploCiclo()
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "configuracion_no_disponible")
		return
	}
	switch decision {
	case "pendiente":
		p.Decisiones = []domain.DecisionRevision{}
	case string(domain.MantenerValoracion):
		for i := range p.Decisiones {
			p.Decisiones[i].Tipo = domain.MantenerValoracion
			p.Decisiones[i].EntradaCorregida = nil
		}
	case string(domain.RectificarValoracion):
	default:
		responderError(w, http.StatusBadRequest, "caso_no_admitido")
		return
	}
	r, err := application.EnsayarCiclo(p)
	if err != nil {
		responderError(w, http.StatusUnprocessableEntity, "preparacion_invalida")
		return
	}
	responderEnsayo(w, r)
}
