package bootstrap

import (
	"net/http"

	"vec-diputacion-granada/internal/vec/auditoria"
)

const envRRHHAuditoriaEnabled = "VEC_RRHH_AUDITORIA_ENABLED"

const (
	claveFronteraOpcionesAuditoriaRRHH = "rrhh-auditoria-opciones"
	claveFronteraConsultaAuditoriaRRHH = "rrhh-auditoria-consultar"
	clavePoliticaAuditoriaRRHH         = "politica-rrhh-auditoria"
	claveCapacidadOpcionesAuditoria    = "capacidad-rrhh-auditoria-opciones"
	claveCapacidadConsultaAuditoria    = "capacidad-rrhh-auditoria-consultar"
)

// GET expone solo los parámetros del catálogo. POST admite dos perfiles
// nominales distintos; la fuente tipada determina qué identidad y emisor V3
// se usan antes de consultar el almacenamiento propietario.
func descriptoresFronterasAuditoriaRRHHDesarrollo(perfilCT, perfilBolsa string) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfilCT) || !perfilActivoSeguridadComunValido(perfilBolsa) || perfilCT == perfilBolsa {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	return []descriptorFronteraComunDesarrollo{
		{Clave: claveFronteraOpcionesAuditoriaRRHH, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodGet, Ruta: auditoria.RutaOpciones, PerfilesActivosRef: []string{perfilCT},
			ClavePolitica: clavePoliticaAuditoriaRRHH, ClaveCapacidad: claveCapacidadOpcionesAuditoria},
		{Clave: claveFronteraConsultaAuditoriaRRHH, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodPost, Ruta: auditoria.RutaConsulta, PerfilesActivosRef: []string{perfilCT, perfilBolsa},
			ClavePolitica: clavePoliticaAuditoriaRRHH, ClaveCapacidad: claveCapacidadConsultaAuditoria},
	}, nil
}
