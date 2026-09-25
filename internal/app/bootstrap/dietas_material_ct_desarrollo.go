package bootstrap

// En desarrollo hay un único gobierno V3: Contratación publica la
// configuración vigente con una sola raíz y la renueva cada día (ver
// contratacion_temporal_confianza_renovable_desarrollo.go). Un módulo que
// publicara otra configuración movería el puntero único y el checkpoint
// monotónico, dejando sin firma a Contratación y a Bolsa. Dietas se integra
// como Bolsa: una clave HMAC derivada por audiencia de consumo, con dominio y
// prefijo propios, bajo la misma raíz y el mismo publicador.

const (
	audienciaConsumoPersonalDietasDesarrollo  = "vec_personal.relacion_propia.consultar_dietas.v1"
	audienciaConsumoCrearDietasDesarrollo     = "vec_dietas.borrador_propio.crear.v1"
	audienciaConsumoConsultarDietasDesarrollo = "vec_dietas.borrador_propio.consultar.v1"
	audienciaConsumoEditarDietasDesarrollo    = "vec_dietas.borrador_propio.editar.v1"
	audienciaConsumoBorrarDietasDesarrollo    = "vec_dietas.borrador_propio.borrar.v1"
	audienciaConsumoEnviarDietasDesarrollo    = "vec_dietas.borrador_propio.enviar.v1"
	audienciaConsumoDocumentoDietasDesarrollo = "vec_dietas.documento_propio.consultar.v1"
	audienciaConsumoConsultarAsignacionDietas = "vec_personal.asignacion_dietas.consultar.v1"
	audienciaConsumoRegistrarAsignacionDietas = "vec_personal.asignacion_dietas.registrar_inicial.v1"
	audienciaConsumoCorregirAsignacionDietas  = "vec_personal.asignacion_dietas.corregir.v1"
	audienciaConsumoCorregirGrupoDietas       = "vec_personal.asignacion_dietas.grupo_corregir.v1"
	// Circuito de revisión: una audiencia por acción y por bandeja.
	audienciaConsumoRevisarDietas              = "vec_dietas.documento.revisar.v1"
	audienciaConsumoAutorizarDietas            = "vec_dietas.documento.autorizar.v1"
	audienciaConsumoLiquidarDietas             = "vec_dietas.documento.liquidar.v1"
	audienciaConsumoFiscalizarDietas           = "vec_dietas.documento.fiscalizar.v1"
	audienciaConsumoBandejaRevisionDietas      = "vec_dietas.bandeja.revision.consultar.v1"
	audienciaConsumoBandejaAutorizacionDietas  = "vec_dietas.bandeja.autorizacion.consultar.v1"
	audienciaConsumoBandejaLiquidacionDietas   = "vec_dietas.bandeja.liquidacion.consultar.v1"
	audienciaConsumoBandejaFiscalizacionDietas = "vec_dietas.bandeja.fiscalizacion.consultar.v1"
	audienciaConsumoRevisorDocumentoDietas     = "vec_dietas.circuito.documento.consultar.v1"
)

// audienciasCircuitoDietasDesarrollo enumera las audiencias del circuito en
// el orden de sus descriptores.
func audienciasCircuitoDietasDesarrollo() []string {
	return []string{audienciaConsumoRevisarDietas, audienciaConsumoAutorizarDietas, audienciaConsumoLiquidarDietas, audienciaConsumoFiscalizarDietas,
		audienciaConsumoBandejaRevisionDietas, audienciaConsumoBandejaAutorizacionDietas, audienciaConsumoBandejaLiquidacionDietas, audienciaConsumoBandejaFiscalizacionDietas,
		audienciaConsumoRevisorDocumentoDietas}
}

// materialDietasDesdeCTDesarrollo transporta proveedores nominales gobernados
// desde la composición de Contratación. No expone claves.
type materialDietasDesdeCTDesarrollo struct {
	personal, crear, consultar *proveedorMaterialAltaContratacionTemporalDesarrollo
	adicionales                map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo
}

func (m materialDietasDesdeCTDesarrollo) completo() bool {
	if m.personal == nil || m.crear == nil || m.consultar == nil {
		return false
	}
	for _, audiencia := range append([]string{audienciaConsumoEditarDietasDesarrollo, audienciaConsumoBorrarDietasDesarrollo, audienciaConsumoEnviarDietasDesarrollo, audienciaConsumoDocumentoDietasDesarrollo, audienciaConsumoConsultarAsignacionDietas, audienciaConsumoRegistrarAsignacionDietas, audienciaConsumoCorregirAsignacionDietas, audienciaConsumoCorregirGrupoDietas}, audienciasCircuitoDietasDesarrollo()...) {
		if m.adicionales[audiencia] == nil {
			return false
		}
	}
	return true
}

func dietasBorradoresSolicitadas(selector string) bool { return selector == "true" }

func descriptoresMaterialDietasDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: audienciaConsumoPersonalDietasDesarrollo, Dominio: "vec.personal.relacion-propia-dietas.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-dietas:", ProveedorNominal: "proveedor-material-personal-dietas"},
		{Audiencia: audienciaConsumoCrearDietasDesarrollo, Dominio: "vec.dietas.borrador-propio.crear.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-crear:", ProveedorNominal: "proveedor-material-dietas-crear"},
		{Audiencia: audienciaConsumoConsultarDietasDesarrollo, Dominio: "vec.dietas.borrador-propio.consultar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-consultar:", ProveedorNominal: "proveedor-material-dietas-consultar"},
		{Audiencia: audienciaConsumoEditarDietasDesarrollo, Dominio: "vec.dietas.borrador-propio.editar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-editar:", ProveedorNominal: "proveedor-material-dietas-editar"},
		{Audiencia: audienciaConsumoBorrarDietasDesarrollo, Dominio: "vec.dietas.borrador-propio.borrar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-borrar:", ProveedorNominal: "proveedor-material-dietas-borrar"},
		{Audiencia: audienciaConsumoEnviarDietasDesarrollo, Dominio: "vec.dietas.borrador-propio.enviar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-enviar:", ProveedorNominal: "proveedor-material-dietas-enviar"},
		{Audiencia: audienciaConsumoDocumentoDietasDesarrollo, Dominio: "vec.dietas.documento-propio.consultar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-documento-consultar:", ProveedorNominal: "proveedor-material-dietas-documento-consultar"},
		{Audiencia: audienciaConsumoConsultarAsignacionDietas, Dominio: "vec.personal.asignacion-dietas.consultar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-asignacion-consultar:", ProveedorNominal: "proveedor-material-personal-asignacion-consultar"},
		{Audiencia: audienciaConsumoRegistrarAsignacionDietas, Dominio: "vec.personal.asignacion-dietas.registrar-inicial.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-asignacion-registrar:", ProveedorNominal: "proveedor-material-personal-asignacion-registrar"},
		{Audiencia: audienciaConsumoCorregirAsignacionDietas, Dominio: "vec.personal.asignacion-dietas.corregir.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-asignacion-corregir:", ProveedorNominal: "proveedor-material-personal-asignacion-corregir"},
		{Audiencia: audienciaConsumoCorregirGrupoDietas, Dominio: "vec.personal.asignacion-dietas.grupo-corregir.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-asignacion-grupo:", ProveedorNominal: "proveedor-material-personal-asignacion-grupo"},
		{Audiencia: audienciaConsumoRevisarDietas, Dominio: "vec.dietas.documento.revisar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-revisar:", ProveedorNominal: "proveedor-material-dietas-revisar"},
		{Audiencia: audienciaConsumoAutorizarDietas, Dominio: "vec.dietas.documento.autorizar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-autorizar:", ProveedorNominal: "proveedor-material-dietas-autorizar"},
		{Audiencia: audienciaConsumoLiquidarDietas, Dominio: "vec.dietas.documento.liquidar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-liquidar:", ProveedorNominal: "proveedor-material-dietas-liquidar"},
		{Audiencia: audienciaConsumoFiscalizarDietas, Dominio: "vec.dietas.documento.fiscalizar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-fiscalizar:", ProveedorNominal: "proveedor-material-dietas-fiscalizar"},
		{Audiencia: audienciaConsumoBandejaRevisionDietas, Dominio: "vec.dietas.bandeja.revision.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-bandeja-revision:", ProveedorNominal: "proveedor-material-dietas-bandeja-revision"},
		{Audiencia: audienciaConsumoBandejaAutorizacionDietas, Dominio: "vec.dietas.bandeja.autorizacion.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-bandeja-autorizacion:", ProveedorNominal: "proveedor-material-dietas-bandeja-autorizacion"},
		{Audiencia: audienciaConsumoBandejaLiquidacionDietas, Dominio: "vec.dietas.bandeja.liquidacion.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-bandeja-liquidacion:", ProveedorNominal: "proveedor-material-dietas-bandeja-liquidacion"},
		{Audiencia: audienciaConsumoBandejaFiscalizacionDietas, Dominio: "vec.dietas.bandeja.fiscalizacion.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-bandeja-fiscalizacion:", ProveedorNominal: "proveedor-material-dietas-bandeja-fiscalizacion"},
		{Audiencia: audienciaConsumoRevisorDocumentoDietas, Dominio: "vec.dietas.circuito.documento.consultar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-revisor-documento:", ProveedorNominal: "proveedor-material-dietas-revisor-documento"},
	}
}
