package bootstrap

import (
	"net/http"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	clavePoliticaBorradorLlamamientoBolsaDesarrollo = "politica-bolsa-bback-borrador-llamamiento"
	claveFronteraCrearBorradorLlamamientoBolsa      = "bolsa-bback-borrador-crear"
	claveFronteraConsultarBorradorLlamamientoBolsa  = "bolsa-bback-borrador-consultar"
	claveCapacidadCrearBorradorLlamamientoBolsa     = "capacidad-bolsa-bback-borrador-crear"
	claveCapacidadConsultarBorradorLlamamientoBolsa = "capacidad-bolsa-bback-borrador-consultar"

	dominioMaterialCrearBorradorLlamamientoBolsa     = "vec.bolsa.borrador-llamamiento.crear.desarrollo.capacidad-v3"
	prefijoMaterialCrearBorradorLlamamientoBolsa     = "clave:capacidad:bolsa-borrador-crear:"
	dominioMaterialConsultarBorradorLlamamientoBolsa = "vec.bolsa.borrador-llamamiento.consultar.desarrollo.capacidad-v3"
	prefijoMaterialConsultarBorradorLlamamientoBolsa = "clave:capacidad:bolsa-borrador-consultar:"
)

// descriptoresFronterasBorradorLlamamientoBolsaDesarrollo declara las dos
// fronteras B-BACK. El perfil procede del contexto Bolsa ya resuelto y nunca
// se sustituye por el perfil de Contratación temporal.
func descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(
	perfilActivoRef string,
) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfilActivoRef) {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	return []descriptorFronteraComunDesarrollo{
		{
			Clave:      claveFronteraCrearBorradorLlamamientoBolsa,
			Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo:     http.MethodPost, Ruta: bolsahttp.RutaBorradoresLlamamiento,
			PerfilesActivosRef: []string{perfilActivoRef},
			ClavePolitica:      clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad:     claveCapacidadCrearBorradorLlamamientoBolsa,
		},
		{
			Clave:      claveFronteraConsultarBorradorLlamamientoBolsa,
			Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo:     http.MethodGet, Ruta: bolsahttp.RutaBorradoresLlamamiento,
			PerfilesActivosRef: []string{perfilActivoRef},
			ClavePolitica:      clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad:     claveCapacidadConsultarBorradorLlamamientoBolsa,
			DetalleColeccion:   true,
		},
	}, nil
}

// descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo enlaza sólo las
// acciones B-BACK exactas con la política completa que compone Bolsa.
func descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(
	politica politicaAutorizacionSolicitudLigadaV3Desarrollo,
) ([]descriptorAutorizacionComunDesarrollo, error) {
	if !politica.valida() {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	return []descriptorAutorizacionComunDesarrollo{
		{
			Accion:         puertosbolsa.AccionCrearBorradorLlamamientoInterno,
			ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: claveCapacidadCrearBorradorLlamamientoBolsa,
			Fronteras:      []string{claveFronteraCrearBorradorLlamamientoBolsa},
			Politica:       politica,
		},
		{
			Accion:         puertosbolsa.AccionConsultarBorradorLlamamientoInterno,
			ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: claveCapacidadConsultarBorradorLlamamientoBolsa,
			Fronteras:      []string{claveFronteraConsultarBorradorLlamamientoBolsa},
			Politica:       politica,
		},
	}, nil
}

// descriptoresMaterialBorradorLlamamientoBolsaDesarrollo conserva las dos
// audiencias B-BACK y sus separaciones de dominio y clave nominales.
func descriptoresMaterialBorradorLlamamientoBolsaDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{
			Audiencia:        puertosbolsa.AudienciaCrearBorradorLlamamientoInterno,
			Dominio:          dominioMaterialCrearBorradorLlamamientoBolsa,
			Prefijo:          prefijoMaterialCrearBorradorLlamamientoBolsa,
			ProveedorNominal: "proveedor-material-borrador-llamamiento-bolsa-crear",
		},
		{
			Audiencia:        puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno,
			Dominio:          dominioMaterialConsultarBorradorLlamamientoBolsa,
			Prefijo:          prefijoMaterialConsultarBorradorLlamamientoBolsa,
			ProveedorNominal: "proveedor-material-borrador-llamamiento-bolsa-consultar",
		},
	}
}
