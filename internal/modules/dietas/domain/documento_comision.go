package domain

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
)

var ErrDocumentoComisionInvalido = errors.New("dietas: documento de comision invalido")
var huellaJustificante = regexp.MustCompile(`^[0-9a-f]{64}$`)
var referenciaJustificante = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9:_-]{2,127}$`)

// Concepto describe el gasto o la razón declarada, también en otro_medio.
// Desde D5 cada línea lleva tipo del catálogo versionado, fecha del gasto y
// justificante obligatorio por referencia y huella: el documento queda en
// custodia de la persona. Las líneas anteriores a D5 (sin tipo ni fecha, con
// justificante opcional) siguen siendo legibles, pero no se pueden guardar.
type OtroGastoDeclarado struct {
	Tipo               string `json:"tipo"`
	Concepto           string `json:"concepto"`
	ImporteCentimos    int64  `json:"importe_centimos"`
	JustificanteRef    string `json:"justificante_ref"`
	JustificanteSHA256 string `json:"justificante_sha256"`
	TipoGasto          string `json:"tipo_gasto,omitempty"`
	CatalogoVersion    string `json:"catalogo_version,omitempty"`
	Fecha              string `json:"fecha,omitempty"`
}

type LineaDocumentoComision struct {
	Tipo               string  `json:"tipo"`
	Grupo              int     `json:"grupo,omitempty"`
	IndiceTramo        *int    `json:"indice_tramo,omitempty"`
	Fecha              string  `json:"fecha,omitempty"`
	Concepto           string  `json:"concepto,omitempty"`
	ImporteCentimos    int64   `json:"importe_centimos"`
	VersionTarifaRef   string  `json:"version_tarifa_ref,omitempty"`
	Rotulo             string  `json:"rotulo,omitempty"`
	OrigenCodigo       string  `json:"origen_codigo,omitempty"`
	DestinoCodigo      string  `json:"destino_codigo,omitempty"`
	Kilometros         string  `json:"kilometros,omitempty"`
	JustificanteRef    *string `json:"justificante_ref,omitempty"`
	JustificanteSHA256 *string `json:"justificante_sha256,omitempty"`
	RutaIndice         int     `json:"ruta_indice,omitempty"`
	KilometrosBase     string  `json:"kilometros_base,omitempty"`
	AjusteKilometros   string  `json:"ajuste_kilometros,omitempty"`
	MotivoAjuste       string  `json:"motivo_ajuste,omitempty"`
	VersionGrafo       string  `json:"version_grafo,omitempty"`
	TipoGasto          string  `json:"tipo_gasto,omitempty"`
	CatalogoVersion    string  `json:"catalogo_version,omitempty"`
}

// DocumentoComision contiene solo los tramos elegidos del grupo acreditado
// por Personal. El total es orientativo; alojamiento es un tope y no pago.
type DocumentoComision struct {
	VehiculoPropio           bool                     `json:"vehiculo_propio"`
	GrupoDieta               int16                    `json:"grupo_dieta"`
	VersionTarifaAceptada    string                   `json:"version_tarifa_aceptada"`
	TramosAceptados          []int                    `json:"tramos_aceptados"`
	Lineas                   []LineaDocumentoComision `json:"lineas"`
	ManutencionCentimos      int64                    `json:"manutencion_centimos"`
	AlojamientoTopeCentimos  int64                    `json:"alojamiento_tope_centimos"`
	KilometrajeCentimos      int64                    `json:"kilometraje_centimos"`
	OtrosCentimos            int64                    `json:"otros_centimos"`
	TotalOrientativoCentimos int64                    `json:"total_orientativo_centimos"`
}

func ConstruirDocumentoComision(calculo CalculoComision, codigos []string, rutas []RutaDeclaradaComision, vehiculo bool, grupo int16, aceptados []int, versionAceptada string, otros []OtroGastoDeclarado) (DocumentoComision, error) {
	var vacio DocumentoComision
	if calculo.ValidarDocumento(codigos, rutas, vehiculo) != nil || len(otros) > 32 || grupo < 1 || grupo > 3 || versionAceptada != calculo.VersionTarifa {
		return vacio, ErrDocumentoComisionInvalido
	}
	opcion := calculo.OpcionesDieta[grupo-1].Calculo
	if (len(opcion.Tramos) == 0 && len(aceptados) != 0) || (len(opcion.Tramos) > 0 && len(aceptados) != 1 && len(aceptados) != len(opcion.Tramos)) {
		return vacio, ErrDocumentoComisionInvalido
	}
	if len(aceptados) == len(opcion.Tramos) && len(aceptados) > 1 {
		for i, indice := range aceptados {
			if indice != i {
				return vacio, ErrDocumentoComisionInvalido
			}
		}
	}
	d := DocumentoComision{VehiculoPropio: vehiculo, GrupoDieta: grupo, VersionTarifaAceptada: versionAceptada, TramosAceptados: append([]int{}, aceptados...), Lineas: make([]LineaDocumentoComision, 0, len(aceptados)+len(rutas)+len(otros)), KilometrajeCentimos: calculo.ImporteKilometrajeCentimos}
	anterior := -1
	for _, indice := range aceptados {
		if indice <= anterior || indice < 0 || indice >= len(opcion.Tramos) {
			return vacio, ErrDocumentoComisionInvalido
		}
		anterior = indice
		tramo := opcion.Tramos[indice]
		if tramo.Tipo == "manutencion" {
			d.ManutencionCentimos += tramo.ImporteCentimos
		} else if tramo.Tipo == "alojamiento_tope_pendiente_justificante" {
			d.AlojamientoTopeCentimos += tramo.ImporteCentimos
		} else {
			return vacio, ErrDocumentoComisionInvalido
		}
		i := indice
		d.Lineas = append(d.Lineas, LineaDocumentoComision{Tipo: "dieta", Grupo: int(grupo), IndiceTramo: &i, Fecha: tramo.Fecha, Concepto: tramo.Tipo, ImporteCentimos: tramo.ImporteCentimos, VersionTarifaRef: tramo.VersionTarifaRef, Rotulo: tramo.Rotulo})
	}
	for i, ruta := range calculo.Rutas {
		d.Lineas = append(d.Lineas, LineaDocumentoComision{Tipo: "kilometraje", RutaIndice: i + 1, OrigenCodigo: ruta.CodigosRuta[0], DestinoCodigo: ruta.CodigosRuta[len(ruta.CodigosRuta)-1], Kilometros: ruta.KilometrosFinales, KilometrosBase: ruta.KilometrosBase, AjusteKilometros: ruta.AjusteKilometros, MotivoAjuste: ruta.MotivoAjuste, ImporteCentimos: ruta.ImporteCentimos, VersionGrafo: ruta.VersionGrafo, VersionTarifaRef: calculo.VersionTarifa, Rotulo: calculo.Rotulo})
	}
	for _, otro := range otros {
		if otro.validarComun() != nil {
			return vacio, ErrDocumentoComisionInvalido
		}
		d.OtrosCentimos += otro.ImporteCentimos
		ref, huella := otro.JustificanteRef, otro.JustificanteSHA256
		d.Lineas = append(d.Lineas, LineaDocumentoComision{Tipo: otro.Tipo, Fecha: otro.Fecha, Concepto: otro.Concepto, ImporteCentimos: otro.ImporteCentimos, JustificanteRef: &ref, JustificanteSHA256: &huella, TipoGasto: otro.TipoGasto, CatalogoVersion: otro.CatalogoVersion})
	}
	d.TotalOrientativoCentimos = d.ManutencionCentimos + d.AlojamientoTopeCentimos + d.KilometrajeCentimos + d.OtrosCentimos
	return d, nil
}

func (d DocumentoComision) Validar(calculo CalculoComision, codigos []string) error {
	rutas := make([]RutaDeclaradaComision, 0, len(calculo.Rutas))
	for _, ruta := range calculo.Rutas {
		rutas = append(rutas, RutaDeclaradaComision{CodigosRuta: append([]string(nil), ruta.CodigosRuta...), AjusteKilometros: ruta.AjusteKilometros, MotivoAjuste: ruta.MotivoAjuste})
	}
	otros := make([]OtroGastoDeclarado, 0)
	for _, linea := range d.Lineas {
		if linea.Tipo == "otro_medio" || linea.Tipo == "otro_gasto" {
			if linea.JustificanteRef == nil || linea.JustificanteSHA256 == nil {
				return ErrDocumentoComisionInvalido
			}
			otros = append(otros, OtroGastoDeclarado{Tipo: linea.Tipo, Concepto: linea.Concepto, ImporteCentimos: linea.ImporteCentimos, JustificanteRef: *linea.JustificanteRef, JustificanteSHA256: *linea.JustificanteSHA256, TipoGasto: linea.TipoGasto, CatalogoVersion: linea.CatalogoVersion, Fecha: linea.Fecha})
		}
	}
	esperado, err := ConstruirDocumentoComision(calculo, codigos, rutas, d.VehiculoPropio, d.GrupoDieta, d.TramosAceptados, d.VersionTarifaAceptada, otros)
	if err != nil || !reflect.DeepEqual(d, esperado) {
		return ErrDocumentoComisionInvalido
	}
	return nil
}

func textoLinea(s string) bool {
	return strings.TrimSpace(s) == s && !strings.ContainsAny(s, "\r\n\x00")
}
