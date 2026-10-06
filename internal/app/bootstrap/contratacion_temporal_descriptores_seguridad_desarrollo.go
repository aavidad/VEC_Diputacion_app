package bootstrap

import (
	"net/http"

	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctpostgres "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

const (
	clavePoliticaContratacionTemporalDesarrollo = "contratacion-temporal-autorizacion-v3"
	proveedorMaterialContratacionTemporal       = "proveedor-material-alta-contratacion-temporal-desarrollo"
)

// perfilesConsultaContratacionTemporalDesarrollo construye el conjunto finito
// de perfiles para las consultas RRHH. Conserva el perfil CT base una vez y
// añade lectores externos en orden de primera aparición. Una base inválida no
// permite declarar lectores alternativos: el catálogo posterior falla cerrado.
func perfilesConsultaContratacionTemporalDesarrollo(
	perfilCT string,
	lectores []string,
) []string {
	if !perfilActivoSeguridadComunValido(perfilCT) {
		return nil
	}
	perfiles := make([]string, 0, len(lectores)+1)
	vistos := map[string]struct{}{perfilCT: {}}
	perfiles = append(perfiles, perfilCT)
	for _, lector := range lectores {
		if _, existe := vistos[lector]; existe {
			continue
		}
		vistos[lector] = struct{}{}
		perfiles = append(perfiles, lector)
	}
	return perfiles
}

// descriptoresFronterasContratacionTemporalDesarrollo declara las fronteras CT
// que comparten contexto autenticado y PDP. Las escrituras y las lecturas de
// cobertura usan sólo el perfil base; cuadro y detalle usan exclusivamente los perfiles
// de consulta recibidos. Cada una conserva la acción nominal como capacidad.
// La lista de peticiones de los centros para RRHH va la última y es GET.
func descriptoresFronterasContratacionTemporalDesarrollo(
	perfilCT string,
	perfilesConsulta []string,
	firma ...bool,
) []descriptorFronteraComunDesarrollo {
	perfilesConsulta = append([]string(nil), perfilesConsulta...)
	fronteras := append([]descriptorFronteraComunDesarrollo{
		fronteraContratacionTemporalDesarrollo("ct-analisis-registrar", ctports.AccionRegistrarAnalisis, cthttp.RutaRegistroAnalisisRRHH, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-analisis-rectificar", ctports.AccionRectificarAnalisis, cthttp.RutaRectificacionAnalisisRRHH, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-solicitud-crear", ctports.AccionCrearSolicitud, cthttp.RutaAltaSolicitudes, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-asignacion-registrar", ctports.AccionRegistrarAsignacion, cthttp.RutaAsignaciones, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-informe-juridico-emitir", ctports.AccionEmitirInformeJuridico, cthttp.RutaPreparacionesInformeJuridico, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-subsanacion-reparo-registrar", string(ctdomain.AccionRegistrarSubsanacionReparo), cthttp.RutaSubsanacionReparos, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-cobertura-decidir", string(ctdomain.AccionDecidirCoberturaGobernada), cthttp.RutaDecisionCobertura, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-cobertura-rectificar", string(ctdomain.AccionRectificarCoberturaGobernada), cthttp.RutaRectificacionCobertura, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-cuadro-consultar", ctports.AccionConsultarCuadroRRHH, cthttp.RutaConsultaCuadroRRHH, perfilesConsulta),
		fronteraContratacionTemporalDesarrollo("ct-expediente-consultar", ctports.AccionConsultarDetalleRRHH, cthttp.RutaConsultaDetalleRRHH, perfilesConsulta),
		// La propuesta y el resultado de cobertura usan el mismo autorizador
		// que la decisión: sin su frontera, el PDP común las deniega (403).
		fronteraContratacionTemporalDesarrollo("ct-cobertura-proponer", accionPropuestaCoberturaDesarrollo, cthttp.RutaPropuestaCobertura, []string{perfilCT}),
		fronteraContratacionTemporalDesarrollo("ct-cobertura-resultado-consultar", string(ctports.AccionConsultarResultadoCobertura), cthttp.RutaResultadoCobertura, []string{perfilCT}),
	}, append(descriptoresFronterasSeguimientoCeseDesarrollo(perfilCT), descriptoresFronterasCancelacionCTDesarrollo(perfilCT)...)...)
	fronteras = append(fronteras, descriptoresFronterasLlamamientoDesarrollo(perfilCT)...)
	fronteras = append(fronteras, fronteraEntregaPeticionCentroRRHHDesarrollo(perfilCT), fronteraPeticionesCentroRRHHDesarrollo(perfilCT))
	if len(firma) > 0 && firma[0] {
		fronteras = append(fronteras,
			fronteraContratacionTemporalDesarrollo("ct-documento-firmar", ctports.AccionFirmarDocumento, cthttp.RutaFirmaDocumento, []string{perfilCT}),
			fronteraContratacionTemporalDesarrollo("ct-firmas-documento-consultar", ctports.AccionConsultarFirmasDocumento, cthttp.RutaConsultaFirmaDocumento, []string{perfilCT}))
	}
	return fronteras
}

// El POST conserva la acción de entrega. El alta anidada requiere además un
// vínculo explícito de CrearSolicitud a esta misma frontera; GET no lo recibe.
func fronteraEntregaPeticionCentroRRHHDesarrollo(perfilCT string) descriptorFronteraComunDesarrollo {
	return fronteraContratacionTemporalDesarrollo("ct-peticiones-centro-rrhh-entregar",
		ctports.AccionEntregarPeticionRRHH, rutaEntregaPeticionCentro, []string{perfilCT})
}

// fronteraPeticionesCentroRRHHDesarrollo declara la lista de «Peticiones de
// los centros» que consulta RRHH (GET). Con Bolsa compuesta, el autorizador
// CT delega en el PDP común; sin esta frontera, la consulta se denegaba antes
// de decidir y la ruta respondía siempre 503. Solo el perfil CT base la usa;
// la entrega (POST) tiene una frontera distinta para la acción de escritura.
func fronteraPeticionesCentroRRHHDesarrollo(perfilCT string) descriptorFronteraComunDesarrollo {
	f := fronteraContratacionTemporalDesarrollo("ct-peticiones-centro-rrhh-consultar",
		ctports.AccionConsultarPeticionesRRHH, rutaEntregaPeticionCentro, []string{perfilCT})
	f.Metodo = http.MethodGet
	return f
}

// CT131 añade únicamente sus tres fronteras con un perfil derivado distinto.
// El perfil CT base conserva todas las rutas anteriores y su PDP común.
func descriptoresFronterasContratacionTemporalConPlantillasDesarrollo(
	perfilCT string, perfilesConsulta []string, firmaActiva bool,
	plantillasActivas bool, perfilPlantillas string,
) ([]descriptorFronteraComunDesarrollo, error) {
	fronteras := descriptoresFronterasContratacionTemporalDesarrollo(perfilCT, perfilesConsulta, firmaActiva)
	if !plantillasActivas {
		if perfilPlantillas != "" {
			return nil, ErrActivacionDesarrolloInvalida
		}
		return fronteras, nil
	}
	if !perfilActivoSeguridadComunValido(perfilPlantillas) || perfilPlantillas == perfilCT {
		return nil, ErrActivacionDesarrolloInvalida
	}
	for _, perfil := range perfilesConsulta {
		if perfil == perfilPlantillas {
			return nil, ErrActivacionDesarrolloInvalida
		}
	}
	return append(fronteras, descriptoresFronterasPlantillasCTDesarrollo(perfilPlantillas)...), nil
}

// CT133 se publica sólo para su perfil documental. La consulta de detalle
// dentro de estas rutas usa otra decisión V3, con el mismo vínculo nominal.
func anexarFronterasPlantillasDocumentalCTDesarrollo(
	fronteras []descriptorFronteraComunDesarrollo, perfilCT, perfilCatalogo, perfilDocumental string,
	perfilesConsulta []string,
) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfilDocumental) ||
		perfilDocumental == perfilCT || perfilDocumental == perfilCatalogo {
		return nil, ErrActivacionDesarrolloInvalida
	}
	for _, perfil := range perfilesConsulta {
		if perfil == perfilDocumental {
			return nil, ErrActivacionDesarrolloInvalida
		}
	}
	return append(fronteras, descriptoresFronterasPlantillasDocumentalCTDesarrollo(perfilDocumental)...), nil
}

func fronteraContratacionTemporalDesarrollo(
	clave, accion, ruta string,
	perfilesActivosRef []string,
) descriptorFronteraComunDesarrollo {
	return descriptorFronteraComunDesarrollo{
		Clave:              clave,
		Superficie:         superficieInternaSeguridadComunDesarrollo,
		Metodo:             http.MethodPost,
		Ruta:               ruta,
		PerfilesActivosRef: append([]string(nil), perfilesActivosRef...),
		ClavePolitica:      clavePoliticaContratacionTemporalDesarrollo,
		ClaveCapacidad:     accion,
	}
}

// descriptoresAutorizacionContratacionTemporalDesarrollo enlaza cada acción
// CT con una única frontera y con la política completa recibida por composición.
func descriptoresAutorizacionContratacionTemporalDesarrollo(
	politica politicaAutorizacionSolicitudLigadaV3Desarrollo,
	reincorporacion ...bool,
) []descriptorAutorizacionComunDesarrollo {
	fronteras := descriptoresFronterasContratacionTemporalDesarrollo(
		"prf_catalogo_ct", []string{"prf_catalogo_ct"}, len(reincorporacion) > 1 && reincorporacion[1],
	)
	descriptores := make([]descriptorAutorizacionComunDesarrollo, 0, len(fronteras))
	for _, frontera := range fronteras {
		descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{
			Accion:         frontera.ClaveCapacidad,
			ClavePolitica:  frontera.ClavePolitica,
			ClaveCapacidad: frontera.ClaveCapacidad,
			Fronteras:      []string{frontera.Clave},
			Politica:       politica,
		})
	}
	descriptores = append(descriptores, descriptoresAutorizacionAdicionalesLlamamientoDesarrollo(politica)...)
	// El circuito solo recibe entrada PDP cuando su perfil fijo ya quedó
	// compuesto. La frontera se declara en la raíz y se enlaza por clave exacta.
	if len(reincorporacion) > 2 && reincorporacion[2] {
		descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{
			Accion:         ctpostgres.AccionConsultaCircuitoRRHH,
			ClavePolitica:  clavePoliticaContratacionTemporalDesarrollo,
			ClaveCapacidad: ctpostgres.AccionConsultaCircuitoRRHH,
			Fronteras:      []string{"ct-circuito-rrhh-consultar"}, Politica: politica,
		})
	}
	if len(reincorporacion) > 1 && reincorporacion[1] {
		// La firma consulta su antecedente con otra decisión nominal del mismo
		// perfil activo, sin sumar perfiles ni reutilizar el permiso de escritura.
		descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{
			Accion: ctports.AccionConsultarFirmasDocumento, ClavePolitica: clavePoliticaContratacionTemporalDesarrollo,
			ClaveCapacidad: ctports.AccionFirmarDocumento, Fronteras: []string{"ct-documento-firmar"}, Politica: politica,
		})
		// La custodia es otra decisión nominal de la misma petición y perfil de
		// firma. Su predicado exige el documento, el PDF y la decisión exactos.
		descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{
			Accion: docports.AccionCustodiarFirmado, ClavePolitica: clavePoliticaContratacionTemporalDesarrollo,
			ClaveCapacidad: ctports.AccionFirmarDocumento, Fronteras: []string{"ct-documento-firmar"}, Politica: politica,
		})
	}
	// CrearSolicitud desde una petición ratificada conserva el perfil y el
	// sello de la reserva. Su autorización se liga solo al POST de entrega;
	// el alta directa mantiene su propia frontera y perfil fijo.
	entrega := fronteraEntregaPeticionCentroRRHHDesarrollo("prf_catalogo_ct")
	descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{
		Accion: ctports.AccionCrearSolicitud, ClavePolitica: entrega.ClavePolitica,
		ClaveCapacidad: entrega.ClaveCapacidad, Fronteras: []string{entrega.Clave}, Politica: politica,
	})
	if len(reincorporacion) != 0 && reincorporacion[0] {
		for _, ruta := range []string{cthttp.RutaReincorporacionesTitular, cthttp.RutaCapacidadReincorporacionTitular} {
			o, _ := operacionSeguimientoCesePorRuta(ruta)
			descriptores = append(descriptores, descriptorAutorizacionComunDesarrollo{
				Accion: o.accion, ClavePolitica: clavePoliticaContratacionTemporalDesarrollo,
				ClaveCapacidad: o.accion, Fronteras: []string{o.frontera}, Politica: politica,
			})
		}
	}
	return descriptores
}

// descriptoresMaterialAutorizacionContratacionTemporalDesarrollo conserva los
// cuatro consumidores CT que ya derivaban material nominal en este cableado.
func descriptoresMaterialAutorizacionContratacionTemporalDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: ctports.AudienciaConsumoConsultaCuadroRRHHV3, Dominio: "vec.ct.cuadro-rrhh.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-cuadro:", ProveedorNominal: proveedorMaterialContratacionTemporal},
		{Audiencia: ctports.AudienciaConsumoConsultaDetalleRRHHV3, Dominio: "vec.ct.detalle-rrhh.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-detalle:", ProveedorNominal: proveedorMaterialContratacionTemporal},
		{Audiencia: ctapplication.AudienciaDespachoCorreoLlamamientoV3, Dominio: "vec.ct.despacho-correo-llamamiento.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-despacho-correo:", ProveedorNominal: proveedorMaterialContratacionTemporal},
		{Audiencia: ctapplication.AudienciaResultadoCorreoLlamamientoV3, Dominio: "vec.ct.resultado-correo-llamamiento.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:ct-resultado-correo:", ProveedorNominal: proveedorMaterialContratacionTemporal},
	}
}
