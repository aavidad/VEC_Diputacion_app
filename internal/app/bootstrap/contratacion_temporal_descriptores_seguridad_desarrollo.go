package bootstrap

import (
	"net/http"

	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
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

// descriptoresFronterasContratacionTemporalDesarrollo declara las diez
// fronteras CT que comparten contexto autenticado y PDP. Las ocho escrituras
// usan sólo el perfil base; cuadro y detalle usan exclusivamente los perfiles
// de consulta recibidos. Cada una conserva la acción nominal como capacidad.
func descriptoresFronterasContratacionTemporalDesarrollo(
	perfilCT string,
	perfilesConsulta []string,
) []descriptorFronteraComunDesarrollo {
	perfilesConsulta = append([]string(nil), perfilesConsulta...)
	return append([]descriptorFronteraComunDesarrollo{
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
	}, descriptoresFronterasSeguimientoCeseDesarrollo(perfilCT)...)
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
) []descriptorAutorizacionComunDesarrollo {
	fronteras := descriptoresFronterasContratacionTemporalDesarrollo(
		"prf_catalogo_ct", []string{"prf_catalogo_ct"},
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
