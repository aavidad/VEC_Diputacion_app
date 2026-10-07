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
	claveFronteraSolicitudesDocumentalesRRHH        = "bolsa-b77-solicitudes-documentales-consultar"
	claveFronteraContratosParticipacionBolsa        = "bolsa-b13-contratos-consultar"
	claveFronteraReincorporacionesTitularBolsa      = "bolsa-b46-reincorporaciones-titular-consultar"
	claveFronteraConsultarSancionesBolsa            = "bolsa-b24-sanciones-consultar"
	claveFronteraRecursoSancionBolsa                = "bolsa-b24-sancion-recurso"
	claveCapacidadSituacionParticipacionBolsa       = "capacidad-bolsa-b2-situacion-cambiar"
	claveCapacidadSolicitudesDocumentalesRRHH       = "capacidad-bolsa-b77-solicitudes-documentales-consultar"
	claveCapacidadConsultaReincorporacionBolsa      = "capacidad-bolsa-b55-reincorporacion-consultar"
	claveFronteraConsultarContactosBolsa            = "bolsa-b3-contactos-consultar"
	claveFronteraConsultarContactosB5Bolsa          = "bolsa-b3-contactos-b5-consultar"
	claveFronteraConsultarContactosOfertaBolsa      = "bolsa-b72-contactos-oferta-consultar"
	claveCapacidadConsultarContactosBolsa           = "capacidad-bolsa-b3-contactos-consultar"
	claveFronteraConsultarDatosContactoBolsa        = "bolsa-b4-datos-contacto-consultar"
	claveCapacidadConsultarDatosContactoBolsa       = "capacidad-bolsa-b4-datos-contacto-consultar"
	claveFronteraEmisionLlamamientoBolsa            = "bolsa-b7-llamamiento-emitir"
	claveFronteraRecuperarEmisionLlamamientoBolsa   = "bolsa-b7-llamamiento-recuperar"
	claveCapacidadEmisionLlamamientoBolsa           = "capacidad-bolsa-b7-llamamiento-emitir"
	claveFronteraPlantillaCorreoLlamamientoBolsa    = "bolsa-b7-correo-plantilla"
	claveFronteraVistaPreviaCorreoLlamamientoBolsa  = "bolsa-b7-correo-vista-previa"
	claveFronteraConsultarPoliticaOfertasBolsa      = "bolsa-b47-politica-ofertas-consultar"
	claveFronteraPublicarPoliticaOfertasBolsa       = "bolsa-b47-politica-ofertas-publicar"
	claveFronteraCapacidadPoliticaOfertasBolsa      = "bolsa-b47-politica-ofertas-capacidad-publicar"
	claveCapacidadPoliticaOfertasBolsa              = "capacidad-bolsa-b47-politica-ofertas-publicar"
	claveCapacidadConsultarPoliticaOfertasBolsa     = "capacidad-bolsa-b51-politica-ofertas-consultar"
	envBolsaPoliticaOfertasEnabled                  = "VEC_BOLSA_POLITICA_OFERTAS_ENABLED"
	claveFronteraVistaPreviaCargaConvocaBolsa       = "bolsa-carga-convoca-vista-previa"
	claveFronteraConfirmarCargaConvocaBolsa         = "bolsa-carga-convoca-confirmar"
	claveCapacidadCargaConvocaBolsa                 = "capacidad-bolsa-carga-convoca-confirmar"

	dominioMaterialCrearBorradorLlamamientoBolsa      = "vec.bolsa.borrador-llamamiento.crear.desarrollo.capacidad-v3"
	prefijoMaterialCrearBorradorLlamamientoBolsa      = "clave:capacidad:bolsa-borrador-crear:"
	dominioMaterialConsultarBorradorLlamamientoBolsa  = "vec.bolsa.borrador-llamamiento.consultar.desarrollo.capacidad-v3"
	prefijoMaterialConsultarBorradorLlamamientoBolsa  = "clave:capacidad:bolsa-borrador-consultar:"
	dominioMaterialSituacionParticipacionBolsa        = "vec.bolsa.situacion-participacion.cambiar.desarrollo.capacidad-v3"
	prefijoMaterialSituacionParticipacionBolsa        = "clave:capacidad:bolsa-situacion-cambiar:"
	dominioMaterialSolicitudesDocumentalesRRHH        = "vec.bolsa.solicitudes-documentales.consultar-rrhh.desarrollo.capacidad-v3"
	prefijoMaterialSolicitudesDocumentalesRRHH        = "clave:capacidad:bolsa-solicitudes-documentales-consultar-rrhh:"
	dominioMaterialContactoParticipacionBolsa         = "vec.bolsa.contacto-participacion.registrar.desarrollo.capacidad-v3"
	prefijoMaterialContactoParticipacionBolsa         = "clave:capacidad:bolsa-contacto-registrar:"
	dominioMaterialConsultaContactoParticipacionBolsa = "vec.bolsa.contacto-participacion.consultar.desarrollo.capacidad-v3"
	prefijoMaterialConsultaContactoParticipacionBolsa = "clave:capacidad:bolsa-contacto-consultar:"
	dominioMaterialDatosContactoParticipacionBolsa    = "vec.bolsa.datos-contacto-participacion.registrar.desarrollo.capacidad-v3"
	prefijoMaterialDatosContactoParticipacionBolsa    = "clave:capacidad:bolsa-datos-contacto-registrar:"
	dominioMaterialConsultaDatosContactoBolsa         = "vec.bolsa.datos-contacto-participacion.consultar.desarrollo.capacidad-v3"
	prefijoMaterialConsultaDatosContactoBolsa         = "clave:capacidad:bolsa-datos-contacto-consultar:"
	dominioMaterialEmisionLlamamientoBolsa            = "vec.bolsa.llamamiento.emitir.desarrollo.capacidad-v3"
	prefijoMaterialEmisionLlamamientoBolsa            = "clave:capacidad:bolsa-llamamiento-emitir:"
	dominioMaterialPoliticaOfertasBolsa               = "vec.bolsa.politica-ofertas.publicar.desarrollo.capacidad-v3"
	prefijoMaterialPoliticaOfertasBolsa               = "clave:capacidad:bolsa-politica-ofertas-publicar:"
	dominioMaterialConsultaPoliticaOfertasBolsa       = "vec.bolsa.politica-ofertas.consultar.desarrollo.capacidad-v3"
	prefijoMaterialConsultaPoliticaOfertasBolsa       = "clave:capacidad:bolsa-politica-ofertas-consultar:"
	dominioMaterialConsultaReincorporacionBolsa       = "vec.bolsa.reincorporacion-titular.consultar.desarrollo.capacidad-v3"
	prefijoMaterialConsultaReincorporacionBolsa       = "clave:capacidad:bolsa-reincorporacion-titular-consultar:"
	dominioMaterialCargaConvocaBolsa                  = "vec.bolsa.carga-convoca.confirmar.desarrollo.capacidad-v3"
	prefijoMaterialCargaConvocaBolsa                  = "clave:capacidad:bolsa-carga-convoca-confirmar:"
)

func descriptorMaterialConsultaReincorporacionTitularBolsaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia: puertosbolsa.AudienciaConsultarReincorporacionTitular,
		Dominio:   dominioMaterialConsultaReincorporacionBolsa, Prefijo: prefijoMaterialConsultaReincorporacionBolsa,
		ProveedorNominal: "proveedor-material-consulta-reincorporacion-titular-bolsa",
	}
}

func descriptorMaterialPoliticaOfertasBolsaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: puertosbolsa.AudienciaPublicarPoliticaOfertas,
		Dominio: dominioMaterialPoliticaOfertasBolsa, Prefijo: prefijoMaterialPoliticaOfertasBolsa,
		ProveedorNominal: "proveedor-material-politica-ofertas-bolsa"}
}

func descriptorMaterialConsultaPoliticaOfertasBolsaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: puertosbolsa.AudienciaConsultarPoliticaOfertas,
		Dominio: dominioMaterialConsultaPoliticaOfertasBolsa, Prefijo: prefijoMaterialConsultaPoliticaOfertasBolsa,
		ProveedorNominal: "proveedor-material-consulta-politica-ofertas-bolsa"}
}

func descriptorMaterialCargaConvocaBolsaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{Audiencia: puertosbolsa.AudienciaConfirmarCargaConvoca,
		Dominio: dominioMaterialCargaConvocaBolsa, Prefijo: prefijoMaterialCargaConvocaBolsa,
		ProveedorNominal: "proveedor-material-carga-convoca-bolsa-confirmar"}
}

// descriptoresFronterasBorradorLlamamientoBolsaDesarrollo declara las dos
// fronteras B-BACK. El perfil procede del contexto Bolsa ya resuelto y nunca
// se sustituye por el perfil de Contratación temporal.
func descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(
	perfilActivoRef string,
	capacidades ...bool,
) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfilActivoRef) {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	descriptores := append([]descriptorFronteraComunDesarrollo{
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
		{Clave: claveFronteraSolicitudesDocumentalesRRHH, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaSolicitudesDocumentalesPendientesRRHH, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSolicitudesDocumentalesRRHH},
		{Clave: claveFronteraContratosParticipacionBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "contratos"}},
		{Clave: claveFronteraConsultarSancionesBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "sanciones"}},
		{Clave: claveFronteraRecursoSancionBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "sanciones", "*", "recursos"}},
		{Clave: claveFronteraConsultarContactosBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarContactosBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "contactos"}},
		{Clave: claveFronteraConsultarContactosB5Bolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarContactosBolsa, PlantillaDetalle: []string{"*", "candidatos"}},
		{Clave: claveFronteraConsultarContactosOfertaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaContactosOferta, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarContactosBolsa},
		{Clave: claveFronteraConsultarDatosContactoBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarDatosContactoBolsa, PlantillaDetalle: []string{"*", "candidatos", "*", "datos-contacto"}},
		{Clave: claveFronteraEmisionLlamamientoBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaEmisionesLlamamiento, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{Clave: claveFronteraRecuperarEmisionLlamamientoBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaEmisionesLlamamiento, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{Clave: claveFronteraPlantillaCorreoLlamamientoBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaPlantillaCorreoLlamamiento, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{Clave: claveFronteraVistaPreviaCorreoLlamamientoBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaVistaPreviaCorreoLlamamiento, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa},
		{
			Clave:      claveFronteraConsultarBorradorLlamamientoBolsa,
			Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo:     http.MethodGet, Ruta: bolsahttp.RutaBorradoresLlamamiento,
			PerfilesActivosRef: []string{perfilActivoRef},
			ClavePolitica:      clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad:     claveCapacidadConsultarBorradorLlamamientoBolsa,
			DetalleColeccion:   true,
		},
	}, descriptoresFronterasOfertasBolsaDesarrollo(perfilActivoRef)...)
	descriptores = append(descriptores,
		descriptorFronteraComunDesarrollo{Clave: claveFronteraVistaPreviaCargaConvocaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaVistaPreviaCargaConvoca, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadCargaConvocaBolsa},
		descriptorFronteraComunDesarrollo{Clave: claveFronteraConfirmarCargaConvocaBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaConfirmarCargaConvoca, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadCargaConvocaBolsa})
	if len(capacidades) > 1 && capacidades[1] {
		descriptores = append(descriptores, descriptorFronteraComunDesarrollo{
			Clave: claveFronteraReincorporacionesTitularBolsa, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodGet, Ruta: bolsahttp.RutaBolsasGestion,
			PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad:   claveCapacidadConsultaReincorporacionBolsa,
			PlantillaDetalle: []string{"*", "candidatos", "*", "reincorporaciones-titular"},
		})
	}
	if len(capacidades) != 0 && capacidades[0] {
		descriptores = append(descriptores,
			descriptorFronteraComunDesarrollo{Clave: claveFronteraConsultarPoliticaOfertasBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodGet, Ruta: bolsahttp.RutaPoliticaOfertas, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarPoliticaOfertasBolsa},
			descriptorFronteraComunDesarrollo{Clave: claveFronteraPublicarPoliticaOfertasBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaPoliticaOfertas, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadPoliticaOfertasBolsa},
			descriptorFronteraComunDesarrollo{Clave: claveFronteraCapacidadPoliticaOfertasBolsa, Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost, Ruta: bolsahttp.RutaCapacidadPoliticaOfertas, PerfilesActivosRef: []string{perfilActivoRef}, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadPoliticaOfertasBolsa})
	}
	return descriptores, nil
}

// descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo enlaza sólo las
// acciones B-BACK exactas con la política completa que compone Bolsa.
func descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(
	politica politicaAutorizacionSolicitudLigadaV3Desarrollo,
	capacidades ...bool,
) ([]descriptorAutorizacionComunDesarrollo, error) {
	if !politica.valida() {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	descriptores := []descriptorAutorizacionComunDesarrollo{
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
		{Accion: puertosbolsa.AccionConfirmarCargaConvoca, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadCargaConvocaBolsa, Fronteras: []string{claveFronteraVistaPreviaCargaConvocaBolsa, claveFronteraConfirmarCargaConvocaBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionCambiarSituacionParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, Fronteras: []string{claveFronteraSituacionParticipacionBolsa, claveFronteraOperacionesSituacionBolsa, claveFronteraContratosParticipacionBolsa, claveFronteraConsultarSancionesBolsa, claveFronteraRecursoSancionBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionConsultarSolicitudesDocumentalesRRHH, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSolicitudesDocumentalesRRHH, Fronteras: []string{claveFronteraSolicitudesDocumentalesRRHH}, Politica: politica},
		{Accion: puertosbolsa.AccionRegistrarContactoParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, Fronteras: []string{claveFronteraSituacionParticipacionBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionConsultarContactoParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarContactosBolsa, Fronteras: []string{claveFronteraConsultarContactosBolsa, claveFronteraConsultarContactosB5Bolsa, claveFronteraConsultarContactosOfertaBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionRegistrarDatosContactoParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadSituacionParticipacionBolsa, Fronteras: []string{claveFronteraSituacionParticipacionBolsa}, Politica: politica},
		// La lectura GET de datos-contacto solo admite la acción de consulta,
		// nunca la de registro (la consulta completa consume esta decisión).
		{Accion: puertosbolsa.AccionConsultarDatosContactoParticipacion, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarDatosContactoBolsa, Fronteras: []string{claveFronteraConsultarDatosContactoBolsa}, Politica: politica},
		{Accion: puertosbolsa.AccionEmitirLlamamiento, ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadEmisionLlamamientoBolsa, Fronteras: []string{claveFronteraEmisionLlamamientoBolsa, claveFronteraRecuperarEmisionLlamamientoBolsa, claveFronteraPublicarOfertaBolsa, claveFronteraConsultarOfertaBolsa, claveFronteraResolverOfertaBolsa, claveFronteraPlantillaCorreoLlamamientoBolsa, claveFronteraVistaPreviaCorreoLlamamientoBolsa}, Politica: politica},
	}
	if len(capacidades) > 1 && capacidades[1] {
		descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{
			Accion:         puertosbolsa.AccionConsultarReincorporacionTitular,
			ClavePolitica:  clavePoliticaBorradorLlamamientoBolsaDesarrollo,
			ClaveCapacidad: claveCapacidadConsultaReincorporacionBolsa,
			Fronteras:      []string{claveFronteraReincorporacionesTitularBolsa}, Politica: politica,
		})
	}
	if len(capacidades) != 0 && capacidades[0] {
		descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{Accion: puertosbolsa.AccionConsultarPoliticaOfertas,
			ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadConsultarPoliticaOfertasBolsa,
			Fronteras: []string{claveFronteraConsultarPoliticaOfertasBolsa}, Politica: politica})
		descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{Accion: puertosbolsa.AccionPublicarPoliticaOfertas,
			ClavePolitica: clavePoliticaBorradorLlamamientoBolsaDesarrollo, ClaveCapacidad: claveCapacidadPoliticaOfertasBolsa,
			Fronteras: []string{claveFronteraPublicarPoliticaOfertasBolsa, claveFronteraCapacidadPoliticaOfertasBolsa}, Politica: politica})
	}
	return descriptores, nil
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
		{Audiencia: puertosbolsa.AudienciaConsultarSolicitudesDocumentalesRRHH, Dominio: dominioMaterialSolicitudesDocumentalesRRHH, Prefijo: prefijoMaterialSolicitudesDocumentalesRRHH, ProveedorNominal: "proveedor-material-solicitudes-documentales-rrhh-bolsa"},
		{Audiencia: puertosbolsa.AudienciaRegistrarContactoParticipacion, Dominio: dominioMaterialContactoParticipacionBolsa, Prefijo: prefijoMaterialContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaConsultarContactoParticipacion, Dominio: dominioMaterialConsultaContactoParticipacionBolsa, Prefijo: prefijoMaterialConsultaContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-consulta-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaRegistrarDatosContactoParticipacion, Dominio: dominioMaterialDatosContactoParticipacionBolsa, Prefijo: prefijoMaterialDatosContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-datos-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaConsultarDatosContactoParticipacion, Dominio: dominioMaterialConsultaDatosContactoBolsa, Prefijo: prefijoMaterialConsultaDatosContactoBolsa, ProveedorNominal: "proveedor-material-consulta-datos-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaEmitirLlamamiento, Dominio: dominioMaterialEmisionLlamamientoBolsa, Prefijo: prefijoMaterialEmisionLlamamientoBolsa, ProveedorNominal: "proveedor-material-emision-llamamiento-bolsa"},
		{
			Audiencia:        puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno,
			Dominio:          dominioMaterialConsultarBorradorLlamamientoBolsa,
			Prefijo:          prefijoMaterialConsultarBorradorLlamamientoBolsa,
			ProveedorNominal: "proveedor-material-borrador-llamamiento-bolsa-consultar",
		},
	}
}
