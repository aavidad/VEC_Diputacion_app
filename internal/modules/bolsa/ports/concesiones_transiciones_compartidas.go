package ports

// Estas coordenadas de concesión se cotejan tanto al adoptar el gobierno de
// transiciones como al admitir un acto nuevo sobre la misma autoridad.
const (
	AccionPublicarTransicionesSituacion      = "bolsa.politica_transiciones.publicar"
	AccionConsultarTransicionesSituacion     = "bolsa.politica_transiciones.consultar"
	FinalidadGobiernoTransicionesSituacion   = "gobierno_politica_transiciones_bolsa"
	FinalidadConsultaTransicionesSituacion   = "consultar_politica_transiciones_bolsa"
	CampoConsultaTransicionesSituacion       = "politica_transiciones"
	TipoRecursoGobiernoTransicionesSituacion = "politica_transiciones_global"
)
