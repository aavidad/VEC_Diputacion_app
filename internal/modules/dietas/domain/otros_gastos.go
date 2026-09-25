package domain

// D5: otros medios de transporte y otros gastos con justificante.
//
// El catálogo de tipos es versionado y está rotulado provisional: RRHH no ha
// confirmado todavía qué tipos admite ni con qué límites. La misma versión
// está publicada en PostgreSQL (Dietas 000009), que es quien la exige al
// guardar; este espejo solo permite validar antes y ofrecer los tipos al
// cliente. Una prueba compara ambos para que no diverjan.

// VersionCatalogoOtrosGastos identifica la única versión publicada.
const VersionCatalogoOtrosGastos = "provisional:otros-gastos:20260925"

// ClaseOtroMedio y ClaseOtroGasto son los dos apartados del documento.
const (
	ClaseOtroMedio = "otro_medio"
	ClaseOtroGasto = "otro_gasto"
)

// TipoOtroGasto es una entrada del catálogo: el código estable y el apartado
// del documento en el que se suma.
type TipoOtroGasto struct {
	Codigo string `json:"codigo"`
	Clase  string `json:"clase"`
}

// CatalogoOtrosGastos es la proyección pública del catálogo versionado.
type CatalogoOtrosGastos struct {
	Version string          `json:"version"`
	Rotulo  string          `json:"rotulo"`
	Tipos   []TipoOtroGasto `json:"tipos"`
}

var tiposOtrosGastos = []TipoOtroGasto{
	{"tren", ClaseOtroMedio},
	{"autobus", ClaseOtroMedio},
	{"metro_tranvia", ClaseOtroMedio},
	{"taxi", ClaseOtroMedio},
	{"avion", ClaseOtroMedio},
	{"barco", ClaseOtroMedio},
	{"vehiculo_alquiler", ClaseOtroMedio},
	{"aparcamiento", ClaseOtroGasto},
	{"peaje", ClaseOtroGasto},
	{"consigna", ClaseOtroGasto},
}

// CatalogoOtrosGastosVigente devuelve una copia defensiva del catálogo.
func CatalogoOtrosGastosVigente() CatalogoOtrosGastos {
	return CatalogoOtrosGastos{Version: VersionCatalogoOtrosGastos, Rotulo: RotuloTarifaProvisional, Tipos: append([]TipoOtroGasto(nil), tiposOtrosGastos...)}
}

// ClaseDeTipoOtroGasto resuelve el apartado de un tipo en una versión dada.
func ClaseDeTipoOtroGasto(version, codigo string) (string, bool) {
	if version != VersionCatalogoOtrosGastos {
		return "", false
	}
	for _, t := range tiposOtrosGastos {
		if t.Codigo == codigo {
			return t.Clase, true
		}
	}
	return "", false
}

// Catalogado indica si la línea usa la forma D5 (tipo, versión y fecha). Las
// líneas guardadas antes de D5 no la tienen y se siguen pudiendo leer.
func (o OtroGastoDeclarado) Catalogado() bool {
	return o.TipoGasto != "" || o.CatalogoVersion != "" || o.Fecha != ""
}

// ValidarAlta exige la forma D5 completa para toda línea nueva o editada:
// tipo del catálogo coherente con el apartado, fecha del gasto dentro de la
// comisión y justificante por referencia y huella SHA-256 (el documento
// permanece en custodia de la persona; VEC no lo recibe).
func (o OtroGastoDeclarado) ValidarAlta(fechaInicio, fechaFin string) error {
	clase, ok := ClaseDeTipoOtroGasto(o.CatalogoVersion, o.TipoGasto)
	if !ok || clase != o.Tipo || !fechaCivilValida(o.Fecha) || o.Fecha < fechaInicio || o.Fecha > fechaFin ||
		!referenciaJustificante.MatchString(o.JustificanteRef) || !huellaJustificante.MatchString(o.JustificanteSHA256) {
		return ErrDocumentoComisionInvalido
	}
	return o.validarComun()
}

func (o OtroGastoDeclarado) validarComun() error {
	if (o.Tipo != ClaseOtroMedio && o.Tipo != ClaseOtroGasto) || len(o.Concepto) < 3 || len(o.Concepto) > 500 || !textoLinea(o.Concepto) || o.ImporteCentimos < 1 || o.ImporteCentimos > 100000000 {
		return ErrDocumentoComisionInvalido
	}
	if o.Catalogado() {
		clase, ok := ClaseDeTipoOtroGasto(o.CatalogoVersion, o.TipoGasto)
		if !ok || clase != o.Tipo || !fechaCivilValida(o.Fecha) || !referenciaJustificante.MatchString(o.JustificanteRef) || !huellaJustificante.MatchString(o.JustificanteSHA256) {
			return ErrDocumentoComisionInvalido
		}
		return nil
	}
	parejaVacia := o.JustificanteRef == "" && o.JustificanteSHA256 == ""
	parejaValida := referenciaJustificante.MatchString(o.JustificanteRef) && huellaJustificante.MatchString(o.JustificanteSHA256)
	if !parejaVacia && !parejaValida {
		return ErrDocumentoComisionInvalido
	}
	return nil
}
