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
	claveFronteraSituacionParticipacionBolsa        = "bolsa-b2-situacion-cambiar"
	claveFronteraOperacionesSituacionBolsa          = "bolsa-b8-operaciones-situacion"
	claveCapacidadSituacionParticipacionBolsa       = "capacidad-bolsa-b2-situacion-cambiar"
	claveFronteraConsultarContactosBolsa            = "bolsa-b3-contactos-consultar"
	claveFronteraConsultarContactosB5Bolsa          = "bolsa-b3-contactos-b5-consultar"
	claveCapacidadConsultarContactosBolsa           = "capacidad-bolsa-b3-contactos-consultar"
	claveFronteraConsultarDatosContactoBolsa        = "bolsa-b4-datos-contacto-consultar"
	claveFronteraEmisionLlamamientoBolsa            = "bolsa-b7-llamamiento-emitir"
	claveFronteraRecuperarEmisionLlamamientoBolsa   = "bolsa-b7-llamamiento-recuperar"
	claveCapacidadEmisionLlamamientoBolsa           = "capacidad-bolsa-b7-llamamiento-emitir"

	dominioMaterialCrearBorradorLlamamientoBolsa      = "vec.bolsa.borrador-llamamiento.crear.desarrollo.capacidad-v3"
	prefijoMaterialCrearBorradorLlamamientoBolsa      = "clave:capacidad:bolsa-borrador-crear:"
	dominioMaterialConsultarBorradorLlamamientoBolsa  = "vec.bolsa.borrador-llamamiento.consultar.desarrollo.capacidad-v3"
	prefijoMaterialConsultarBorradorLlamamientoBolsa  = "clave:capacidad:bolsa-borrador-consultar:"
	dominioMaterialSituacionParticipacionBolsa        = "vec.bolsa.situacion-participacion.cambiar.desarrollo.capacidad-v3"
	prefijoMaterialSituacionParticipacionBolsa        = "clave:capacidad:bolsa-situacion-cambiar:"
	dominioMaterialContactoParticipacionBolsa         = "vec.bolsa.contacto-participacion.registrar.desarrollo.capacidad-v3"
	prefijoMaterialContactoParticipacionBolsa         = "clave:capacidad:bolsa-contacto-registrar:"
	dominioMaterialConsultaContactoParticipacionBolsa = "vec.bolsa.contacto-participacion.consultar.desarrollo.capacidad-v3"
	prefijoMaterialConsultaContactoParticipacionBolsa = "clave:capacidad:bolsa-contacto-consultar:"
	dominioMaterialDatosContactoParticipacionBolsa    = "vec.bolsa.datos-contacto-participacion.registrar.desarrollo.capacidad-v3"
	prefijoMaterialDatosContactoParticipacionBolsa    = "clave:capacidad:bolsa-datos-contacto-registrar:"
	dominioMaterialEmisionLlamamientoBolsa            = "vec.bolsa.llamamiento.emitir.desarrollo.capacidad-v3"
	prefijoMaterialEmisionLlamamientoBolsa            = "clave:capacidad:bolsa-llamamiento-emitir:"
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
	return append([]descriptorFronteraComunDesarrollo{
		{
			Clave:      claveFronteraCrearBorradorLlamamientoBolsa,
			Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo:     http.MethodPost, Ruta: bolsahttp.RutaBorradoresLlamamiento,
			PerfilesActivosRef: []string{perfilActivoRef},
			ClavePolitica:      clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad:     claveCapacidadCrearBorradorLlamamientoBolsa,
		},
		{Clave: claveFronteraSituacionParticipacionBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "*"}},
		{Clave: claveFronteraOperacionesSituacionBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "operaciones"}},
		{Clave: claveFronteraConsultarContactosBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarContactosBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "contactos"}},
		{Clave: claveFronteraConsultarContactosB5Bolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarContactosBolsa, PlantillaDetalle: []string{"*", "candidatos"}},
		{Clave: claveFronteraConsultarDatosContactoBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "datos-contacto"}},
		{Clave: claveFronteraEmisionLlamamientoBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaEmisionesLlamamiento, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{Clave: claveFronteraRecuperarEmisionLlamamientoBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaEmisionesLlamamiento, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{
			Clave:      claveFronteraConsultarBorradorLlamamientoBolsa,
			Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo:     http.MethodGet, Ruta: bolsahttp.RutaBorradoresLlamamiento,
			PerfilesActivosRef: []string{perfilActivoRef},
			ClavePolitica:      clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad:     claveCapacidadConsultarBorradorLlamamientoBolsa,
			DetalleColeccion:   true,
		},
	}, descriptoresFronterasOfertasBolsaDesarrollo(perfilActivoRef)...), nil
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
		{Accion: puertosbolsa.AccionCambiarSituacionParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, Fronteras: []string{claveFronteraSituacionParticipacionBolsa, claveFronteraOperacionesSituacionBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionRegistrarContactoParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, Fronteras: []string{claveFronteraSituacionParticipacionBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionConsultarContactoParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarContactosBolsa, Fronteras: []string{claveFronteraConsultarContactosBolsa, claveFronteraConsultarContactosB5Bolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionRegistrarDatosContactoParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, Fronteras: []string{claveFronteraSituacionParticipacionBolsa, claveFronteraConsultarDatosContactoBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionEmitirLlamamiento, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa, Fronteras: []string{claveFronteraEmisionLlamamientoBolsa, claveFronteraRecuperarEmisionLlamamientoBolsa, claveFronteraPublicarOfertaBolsa, claveFronteraConsultarOfertaBolsa, claveFronteraResolverOfertaBolsa}, Politica: politica},
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
		{Audiencia: puertosbolsa.AudienciaCambiarSituacionParticipacion, Dominio: dominioMaterialSituacionParticipacionBolsa, Prefijo: prefijoMaterialSituacionParticipacionBolsa, ProveedorNominal: "proveedor-material-situacion-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaRegistrarContactoParticipacion, Dominio: dominioMaterialContactoParticipacionBolsa, Prefijo: prefijoMaterialContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaConsultarContactoParticipacion, Dominio: dominioMaterialConsultaContactoParticipacionBolsa, Prefijo: prefijoMaterialConsultaContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-consulta-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaRegistrarDatosContactoParticipacion, Dominio: dominioMaterialDatosContactoParticipacionBolsa, Prefijo: prefijoMaterialDatosContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-datos-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaEmitirLlamamiento, Dominio: dominioMaterialEmisionLlamamientoBolsa, Prefijo: prefijoMaterialEmisionLlamamientoBolsa, ProveedorNominal: "proveedor-material-emision-llamamiento-bolsa"},
		{
			Audiencia:        puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno,
			Dominio:          dominioMaterialConsultarBorradorLlamamientoBolsa,
			Prefijo:          prefijoMaterialConsultarBorradorLlamamientoBolsa,
			ProveedorNominal: "proveedor-material-borrador-llamamiento-bolsa-consultar",
		},
	}
}
