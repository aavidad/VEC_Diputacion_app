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
