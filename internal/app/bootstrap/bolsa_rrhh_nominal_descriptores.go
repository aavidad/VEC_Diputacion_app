package bootstrap

import (
	"net/http"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	claveFronteraRRHHBolsasBolsa           = "bolsa-rrhh-bolsas-consultar"
	claveFronteraRRHHEstadisticasBolsa     = "bolsa-rrhh-estadisticas-consultar"
	claveFronteraRRHHCandidatosBolsa       = "bolsa-rrhh-candidatos-consultar"
	claveFronteraRRHHBolsasHEADBolsa       = "bolsa-rrhh-bolsas-head"
	claveFronteraRRHHEstadisticasHEADBolsa = "bolsa-rrhh-estadisticas-head"
	claveFronteraRRHHCandidatosHEADBolsa   = "bolsa-rrhh-candidatos-head"
	claveCapacidadRRHHBolsasBolsa          = "capacidad-bolsa-rrhh-bolsas-consultar"
	claveCapacidadRRHHEstadisticasBolsa    = "capacidad-bolsa-rrhh-estadisticas-consultar"
	claveCapacidadRRHHCandidatosBolsa      = "capacidad-bolsa-rrhh-candidatos-consultar"
	clavePoliticaRRHHNominalBolsa          = "politica-bolsa-rrhh-nominal"

	dominioMaterialRRHHBolsasBolsa       = "vec.bolsa.rrhh.bolsas.consultar.capacidad-v3"
	dominioMaterialRRHHEstadisticasBolsa = "vec.bolsa.rrhh.estadisticas.consultar.capacidad-v3"
	dominioMaterialRRHHCandidatosBolsa   = "vec.bolsa.rrhh.candidatos.consultar.capacidad-v3"
	prefijoMaterialRRHHBolsasBolsa       = "clave:capacidad:bolsa-rrhh-bolsas-consultar:"
	prefijoMaterialRRHHEstadisticasBolsa = "clave:capacidad:bolsa-rrhh-estadisticas-consultar:"
	prefijoMaterialRRHHCandidatosBolsa   = "clave:capacidad:bolsa-rrhh-candidatos-consultar:"
)

// Estas fronteras se anexan a la composición; no cambian las huellas del
// catálogo B-BACK previo ni prestan una acción de Contratación temporal.
func descriptoresFronterasRRHHNominalBolsaDesarrollo(perfilActivoRef string) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfilActivoRef) {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	configurar := func(clave, metodo, ruta, capacidad string, detalle []string) descriptorFronteraComunDesarrollo {
		return descriptorFronteraComunDesarrollo{Clave: clave, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: metodo, Ruta: ruta, PlantillaDetalle: detalle, PerfilesActivosRef: []string{perfilActivoRef},
			ClavePolitica: clavePoliticaRRHHNominalBolsa, ClaveCapacidad: capacidad}
	}
	return []descriptorFronteraComunDesarrollo{
		configurar(claveFronteraRRHHBolsasBolsa, http.MethodGet, rutaBolsasRRHHDesarrollo, claveCapacidadRRHHBolsasBolsa, nil),
		configurar(claveFronteraRRHHBolsasHEADBolsa, http.MethodHead, rutaBolsasRRHHDesarrollo, claveCapacidadRRHHBolsasBolsa, nil),
		configurar(claveFronteraRRHHEstadisticasBolsa, http.MethodGet, rutaEstadisticasBolsaRRHHDesarrollo, claveCapacidadRRHHEstadisticasBolsa, nil),
		configurar(claveFronteraRRHHEstadisticasHEADBolsa, http.MethodHead, rutaEstadisticasBolsaRRHHDesarrollo, claveCapacidadRRHHEstadisticasBolsa, nil),
		configurar(claveFronteraRRHHCandidatosBolsa, http.MethodGet, rutaBolsasRRHHDesarrollo, claveCapacidadRRHHCandidatosBolsa, []string{"*", "candidatos"}),
		configurar(claveFronteraRRHHCandidatosHEADBolsa, http.MethodHead, rutaBolsasRRHHDesarrollo, claveCapacidadRRHHCandidatosBolsa, []string{"*", "candidatos"}),
	}, nil
}

func descriptoresAutorizacionRRHHNominalBolsaDesarrollo(politica politicaAutorizacionSolicitudLigadaV3Desarrollo) ([]descriptorAutorizacionComunDesarrollo, error) {
	if !politica.valida() {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	return []descriptorAutorizacionComunDesarrollo{
		{Accion: puertosbolsa.AccionRRHHBolsasConsultar, ClavePolitica: clavePoliticaRRHHNominalBolsa,
			ClaveCapacidad: claveCapacidadRRHHBolsasBolsa, Fronteras: []string{claveFronteraRRHHBolsasBolsa, claveFronteraRRHHBolsasHEADBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionRRHHEstadisticasConsultar, ClavePolitica: clavePoliticaRRHHNominalBolsa,
			ClaveCapacidad: claveCapacidadRRHHEstadisticasBolsa, Fronteras: []string{claveFronteraRRHHEstadisticasBolsa, claveFronteraRRHHEstadisticasHEADBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionRRHHCandidatosConsultar, ClavePolitica: clavePoliticaRRHHNominalBolsa,
			ClaveCapacidad: claveCapacidadRRHHCandidatosBolsa, Fronteras: []string{claveFronteraRRHHCandidatosBolsa, claveFronteraRRHHCandidatosHEADBolsa}, Politica: politica},
	}, nil
}

func descriptoresMaterialRRHHNominalBolsaDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: puertosbolsa.AudienciaRRHHBolsasConsultar, Dominio: dominioMaterialRRHHBolsasBolsa,
			Prefijo: prefijoMaterialRRHHBolsasBolsa, ProveedorNominal: "proveedor-material-rrhh-bolsas-bolsa"},
		{Audiencia: puertosbolsa.AudienciaRRHHEstadisticasConsultar, Dominio: dominioMaterialRRHHEstadisticasBolsa,
			Prefijo: prefijoMaterialRRHHEstadisticasBolsa, ProveedorNominal: "proveedor-material-rrhh-estadisticas-bolsa"},
		{Audiencia: puertosbolsa.AudienciaRRHHCandidatosConsultar, Dominio: dominioMaterialRRHHCandidatosBolsa,
			Prefijo: prefijoMaterialRRHHCandidatosBolsa, ProveedorNominal: "proveedor-material-rrhh-candidatos-bolsa"},
	}
}
