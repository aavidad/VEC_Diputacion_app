package config

// EnvPortalProceso elige qué portal atiende el proceso: vacío para los dos
// (funcionamiento histórico), «interno» para RRHH y empleados o «externo»
// para el Área personal y la consulta pública. Se lee sin recortar espacios:
// un valor mal escrito impide arrancar en lugar de tomarse como vacío.
const EnvPortalProceso = "VEC_PORTAL_PROCESO"

// El campo Config.PortalProceso guarda el valor tal cual; lo interpreta y
// valida internal/app/separacionportales antes de componer nada.
