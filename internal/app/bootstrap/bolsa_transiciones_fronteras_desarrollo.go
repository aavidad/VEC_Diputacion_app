package bootstrap

import (
	"net/http"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	claveFronteraConsultaTransicionesBolsa  = "bolsa-s6-politica-transiciones-consultar"
	claveFronteraPublicaTransicionesBolsa   = "bolsa-s6-politica-transiciones-publicar"
	claveCapacidadConsultaTransicionesBolsa = "capacidad-bolsa-s6-politica-transiciones-consultar"
	claveCapacidadPublicaTransicionesBolsa  = "capacidad-bolsa-s6-politica-transiciones-publicar"
)

// La composición añade estas dos fronteras y acciones antes de crear el PDP.
// La ausencia del selector técnico deja la ruta en 503; no oculta la frontera
// ni concede por sí misma las acciones.
func fronterasGobiernoTransicionesBolsaDesarrollo(perfilActivoRef string) []descriptorFronteraComunDesarrollo {
	if !perfilActivoSeguridadComunValido(perfilActivoRef) {
		return nil
	}
	fronteras := []descriptorFronteraComunDesarrollo{
		{Clave: claveFronteraConsultaTransicionesBolsa, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodGet, Ruta: bolsahttp.RutaGobiernoTransicionesSituacion,
			PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: claveCapacidadConsultaTransicionesBolsa},
		{Clave: claveFronteraPublicaTransicionesBolsa, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodPost, Ruta: bolsahttp.RutaGobiernoTransicionesSituacion,
			PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: claveCapacidadPublicaTransicionesBolsa},
	}
	return fronteras
}

func autorizacionesGobiernoTransicionesBolsaDesarrollo(politica politicaAutorizacionSolicitudLigadaV3Desarrollo) []descriptorAutorizacionComunDesarrollo {
	if !politica.valida() {
		return nil
	}
	return []descriptorAutorizacionComunDesarrollo{
		{Accion: puertosbolsa.AccionConsultarTransicionesSituacion,
			ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: claveCapacidadConsultaTransicionesBolsa,
			Fronteras:      []string{claveFronteraConsultaTransicionesBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionPublicarTransicionesSituacion,
			ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: claveCapacidadPublicaTransicionesBolsa,
			Fronteras:      []string{claveFronteraPublicaTransicionesBolsa}, Politica: politica},
	}
}
