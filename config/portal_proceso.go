package config

// EnvPortalProceso elige qué portal atiende el proceso: vacío para los dos
// (funcionamiento histórico), «interno» para RRHH y empleados o «externo»
// para el Área personal y la consulta pública. Se lee sin recortar espacios:
// un valor mal escrito impide arrancar en lugar de tomarse como vacío.
const EnvPortalProceso = "VEC_PORTAL_PROCESO"

const (
	ValorPortalProcesoInterno = "interno"
	ValorPortalProcesoExterno = "externo"
)

// El campo Config.PortalProceso guarda el valor tal cual; lo interpreta y
// valida internal/app/separacionportales antes de componer nada.

// EnvExternoPreflightV3DatabaseURL es la conexión del proceso externo con la
// que lee el gobierno de autorización (AD3-112), con el LOGIN nominal
// vec_externo_preflight_v3_desarrollo. Lleva el prefijo de las variables del
// portal externo: el proceso interno la rechaza.
const EnvExternoPreflightV3DatabaseURL = "VEC_EXTERNO_PREFLIGHT_V3_DATABASE_URL"

// EnvExternoBolsaPublicaDatabaseURL usa un LOGIN de solo lectura propio del
// proceso externo para la proyección pública B10. La conexión interna no se
// comparte aunque ambas lean la misma base pública gobernada.
const EnvExternoBolsaPublicaDatabaseURL = "VEC_EXTERNO_BOLSA_PUBLICA_DATABASE_URL"

// B11: conexiones nominales del candidato. Cada LOGIN hereda solo su rol de
// consumo externo; la provisión y el gobierno permanecen en el proceso interno.
const (
	EnvExternoBolsaDatabaseURL                 = "VEC_EXTERNO_BOLSA_DATABASE_URL"
	EnvExternoBolsaFronteraDatabaseURL         = "VEC_EXTERNO_BOLSA_FRONTERA_DATABASE_URL"
	EnvExternoAutorizacionFuenteDatabaseURL    = "VEC_EXTERNO_AUTORIZACION_FUENTE_DATABASE_URL"
	EnvExternoAutorizacionRegistroDatabaseURL  = "VEC_EXTERNO_AUTORIZACION_REGISTRO_DATABASE_URL"
	EnvExternoAutorizacionMotivosDatabaseURL   = "VEC_EXTERNO_AUTORIZACION_MOTIVOS_DATABASE_URL"
	EnvExternoIdentidadRegistroDatabaseURL     = "VEC_EXTERNO_IDENTIDAD_REGISTRO_DATABASE_URL"
	EnvExternoIdentidadRevalidacionDatabaseURL = "VEC_EXTERNO_IDENTIDAD_REVALIDACION_DATABASE_URL"
	EnvExternoContextoDatabaseURL              = "VEC_EXTERNO_CONTEXTO_DATABASE_URL"
)
