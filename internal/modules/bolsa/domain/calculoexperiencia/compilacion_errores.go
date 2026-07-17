package calculoexperiencia

// Los codigos de compilacion separan una configuracion valida pero no
// ejecutable por V1 de un fallo aritmetico producido durante el calculo.
const (
	CodigoCompilacionConjuntoInvalido  CodigoError = "compilacion_conjunto_invalido"
	CodigoCompilacionPlanInvalido      CodigoError = "compilacion_plan_invalido"
	CodigoUnidadBaseNoSoportada        CodigoError = "unidad_base_no_soportada"
	CodigoJornadaNoSoportada           CodigoError = "jornada_no_soportada"
	CodigoRedondeoNoSoportado          CodigoError = "redondeo_no_soportado"
	CodigoMinimoSeccionNoSoportado     CodigoError = "minimo_seccion_no_soportado"
	CodigoRestosRedondeoNoSoportados   CodigoError = "restos_redondeo_no_soportados"
	CodigoCoincidenciaNoSoportada      CodigoError = "coincidencia_no_soportada"
	CodigoSolapeNoSoportado            CodigoError = "solape_no_soportado"
	CodigoTopeUnidadesNoSoportado      CodigoError = "tope_unidades_no_soportado"
	CodigoCatalogoCriterioIncompatible CodigoError = "catalogo_criterio_incompatible"
)

var (
	ErrCompilacionConjuntoInvalido  = &ErrorCalculo{codigo: CodigoCompilacionConjuntoInvalido}
	ErrCompilacionPlanInvalido      = &ErrorCalculo{codigo: CodigoCompilacionPlanInvalido}
	ErrUnidadBaseNoSoportada        = &ErrorCalculo{codigo: CodigoUnidadBaseNoSoportada}
	ErrJornadaNoSoportada           = &ErrorCalculo{codigo: CodigoJornadaNoSoportada}
	ErrRedondeoNoSoportado          = &ErrorCalculo{codigo: CodigoRedondeoNoSoportado}
	ErrMinimoSeccionNoSoportado     = &ErrorCalculo{codigo: CodigoMinimoSeccionNoSoportado}
	ErrRestosRedondeoNoSoportados   = &ErrorCalculo{codigo: CodigoRestosRedondeoNoSoportados}
	ErrCoincidenciaNoSoportada      = &ErrorCalculo{codigo: CodigoCoincidenciaNoSoportada}
	ErrSolapeNoSoportado            = &ErrorCalculo{codigo: CodigoSolapeNoSoportado}
	ErrTopeUnidadesNoSoportado      = &ErrorCalculo{codigo: CodigoTopeUnidadesNoSoportado}
	ErrCatalogoCriterioIncompatible = &ErrorCalculo{codigo: CodigoCatalogoCriterioIncompatible}
)
