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
)

// materialDietasDesdeCTDesarrollo transporta los tres proveedores ya
// gobernados desde la composición de Contratación a la de Dietas. No expone
// claves: solo los proveedores nominales que el emisor renovable consume.
type materialDietasDesdeCTDesarrollo struct {
	personal, crear, consultar *proveedorMaterialAltaContratacionTemporalDesarrollo
}

func (m materialDietasDesdeCTDesarrollo) completo() bool {
	return m.personal != nil && m.crear != nil && m.consultar != nil
}

func dietasBorradoresSolicitadas(selector string) bool { return selector == "true" }

func descriptoresMaterialDietasDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	return []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: audienciaConsumoPersonalDietasDesarrollo, Dominio: "vec.personal.relacion-propia-dietas.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:personal-dietas:", ProveedorNominal: "proveedor-material-personal-dietas"},
		{Audiencia: audienciaConsumoCrearDietasDesarrollo, Dominio: "vec.dietas.borrador-propio.crear.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-crear:", ProveedorNominal: "proveedor-material-dietas-crear"},
		{Audiencia: audienciaConsumoConsultarDietasDesarrollo, Dominio: "vec.dietas.borrador-propio.consultar.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:dietas-consultar:", ProveedorNominal: "proveedor-material-dietas-consultar"},
	}
}
