package calculoexperiencia

// Los errores tecnicos de la fase temporal se separan de sus exclusiones y
// bloqueos de negocio. Estos ultimos forman parte del resultado explicable.
const (
	CodigoSeleccionTemporalBloqueada   CodigoError = "seleccion_temporal_bloqueada"
	CodigoSeleccionTemporalInvalida    CodigoError = "seleccion_temporal_invalida"
	CodigoLimiteAplicacionesTemporales CodigoError = "limite_aplicaciones_temporales"
	CodigoLimiteEventosTemporales      CodigoError = "limite_eventos_temporales"
)

var (
	ErrSeleccionTemporalBloqueada   = &ErrorCalculo{codigo: CodigoSeleccionTemporalBloqueada}
	ErrSeleccionTemporalInvalida    = &ErrorCalculo{codigo: CodigoSeleccionTemporalInvalida}
	ErrLimiteAplicacionesTemporales = &ErrorCalculo{codigo: CodigoLimiteAplicacionesTemporales}
	ErrLimiteEventosTemporales      = &ErrorCalculo{codigo: CodigoLimiteEventosTemporales}
)
