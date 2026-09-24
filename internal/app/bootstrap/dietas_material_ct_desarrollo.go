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
)

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
	for _, audiencia := range []string{audienciaConsumoEditarDietasDesarrollo, audienciaConsumoBorrarDietasDesarrollo, audienciaConsumoEnviarDietasDesarrollo, audienciaConsumoDocumentoDietasDesarrollo, audienciaConsumoConsultarAsignacionDietas, audienciaConsumoRegistrarAsignacionDietas, audienciaConsumoCorregirAsignacionDietas, audienciaConsumoCorregirGrupoDietas} {
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
	}
}
