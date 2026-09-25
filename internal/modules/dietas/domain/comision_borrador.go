// Package domain contiene las reglas puras de Dietas.
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var ErrComisionBorradorInvalida = errors.New("dietas: comision borrador invalida")

var (
	referenciaComision = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)
	codigoRuta         = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,64}$`)
	numeroDocumento    = regexp.MustCompile(`^VEC-D-[0-9]{4}-[0-9]{6,18}$`)
)

// ComisionBorrador conserva la declaración y, desde v2, el cálculo orientativo
// versionado del servidor. Ningún importe acredita liquidación ni pago.
type ComisionBorrador struct {
	Referencia      string                   `json:"referencia"`
	NumeroDocumento string                   `json:"numero_documento,omitempty"`
	FechaApertura   string                   `json:"fecha_apertura,omitempty"`
	Version         uint64                   `json:"version,omitempty"`
	Estado          string                   `json:"estado"`
	FechaInicio     string                   `json:"fecha_inicio"`
	FechaFin        string                   `json:"fecha_fin"`
	Motivo          string                   `json:"motivo"`
	CodigosRuta     []string                 `json:"codigos_ruta"`
	RelacionRef     string                   `json:"relacion_ref"`
	CentroRef       string                   `json:"centro_ref,omitempty"`
	UnidadRef       string                   `json:"unidad_ref,omitempty"`
	Calculo         *CalculoComision         `json:"calculo,omitempty"`
	Documento       *DocumentoComision       `json:"documento,omitempty"`
	VehiculoPropio  *bool                    `json:"vehiculo_propio,omitempty"`
	Rutas           *[]RutaDeclaradaComision `json:"rutas,omitempty"`
	Devolucion      *DevolucionComision      `json:"devolucion,omitempty"`
}

// DevolucionComision es la última devolución del circuito que la persona
// titular aún no ha reenviado: etapa, motivo, versión devuelta y fecha,
// tomados de la historia. Solo acompaña a un documento devuelto o en
// corrección; el reenvío vuelve siempre a la revisión del administrativo.
type DevolucionComision struct {
	Etapa      EtapaCircuito `json:"etapa"`
	Motivo     string        `json:"motivo"`
	Version    uint64        `json:"version"`
	DevueltaEn string        `json:"devuelta_en"`
}

func (d DevolucionComision) validarPara(c ComisionBorrador) error {
	if (c.Estado != EstadoDevuelta && c.Estado != "borrador") || d.validarForma() != nil || d.Version > c.Version {
		return ErrComisionBorradorInvalida
	}
	return nil
}

// ValidarAnteriorA comprueba la devolución que acompaña a un documento
// reenviado en el circuito: la última devuelta antes de su versión actual.
func (d DevolucionComision) ValidarAnteriorA(version uint64) error {
	if d.validarForma() != nil || d.Version >= version {
		return ErrComisionBorradorInvalida
	}
	return nil
}

func (d DevolucionComision) validarForma() error {
	if d.Etapa.EstadoPendiente() == "" || len(d.Motivo) < 3 || len(d.Motivo) > 600 || !textoVisible(d.Motivo) ||
		!TextoSinBordes(d.Motivo) || d.Version < 3 || !fechaAperturaValida(d.DevueltaEn) {
		return ErrComisionBorradorInvalida
	}
	return nil
}

func (c ComisionBorrador) Validar() error {
	if !referenciaComision.MatchString(c.Referencia) || (c.Estado != "borrador" && c.Estado != "eliminado" && c.Estado != "enviado_pendiente_revision" && c.Estado != "devuelta" && c.Estado != "pendiente_autorizacion" && c.Estado != "pendiente_liquidacion" && c.Estado != "pendiente_fiscalizacion" && c.Estado != "fiscalizada") ||
		!fechaCivilValida(c.FechaInicio) || !fechaCivilValida(c.FechaFin) ||
		c.FechaInicio > c.FechaFin || len(c.Motivo) < 3 || len(c.Motivo) > 600 ||
		!textoVisible(c.Motivo) || !referenciaRelacionValida(c.RelacionRef) ||
		len(c.CodigosRuta) > 16 {
		return ErrComisionBorradorInvalida
	}
	if (c.NumeroDocumento != "" || c.FechaApertura != "") && (!numeroDocumento.MatchString(c.NumeroDocumento) || !fechaAperturaValida(c.FechaApertura)) {
		return ErrComisionBorradorInvalida
	}
	if c.Version >= 2 && (c.NumeroDocumento == "" || c.FechaApertura == "") {
		return ErrComisionBorradorInvalida
	}
	vistos := map[string]bool{}
	for _, codigo := range c.CodigosRuta {
		if !codigoRuta.MatchString(codigo) || vistos[codigo] {
			return ErrComisionBorradorInvalida
		}
		vistos[codigo] = true
	}
	if c.Documento != nil {
		if c.Version < 2 || c.CentroRef == "" || c.UnidadRef == "" || c.Calculo == nil || c.VehiculoPropio == nil || c.Rutas == nil || c.Documento.VehiculoPropio != *c.VehiculoPropio || c.Calculo.ValidarDocumento(c.CodigosRuta, *c.Rutas, *c.VehiculoPropio) != nil || c.Documento.Validar(*c.Calculo, c.CodigosRuta) != nil {
			return ErrComisionBorradorInvalida
		}
	} else if c.Calculo != nil && c.Calculo.Validar(c.CodigosRuta) != nil {
		return ErrComisionBorradorInvalida
	}
	if c.Estado == "enviado_pendiente_revision" && c.Documento == nil {
		return ErrComisionBorradorInvalida
	}
	if c.Devolucion != nil {
		return c.Devolucion.validarPara(c)
	}
	return nil
}

func fechaAperturaValida(s string) bool {
	t, err := time.Parse("2006-01-02T15:04:05.000000Z", s)
	return err == nil && t.Location() == time.UTC && t.Format("2006-01-02T15:04:05.000000Z") == s
}

func NuevaComisionBorrador(referencia, inicio, fin, motivo, relacion string, codigos []string) (ComisionBorrador, error) {
	// Una ruta vacía es un dato explícito del contrato, no la ausencia del
	// campo. Así se conserva la misma preimagen y la misma respuesta en una
	// recuperación idempotente.
	copia := append([]string{}, codigos...)
	c := ComisionBorrador{Referencia: referencia, Estado: "borrador", FechaInicio: inicio, FechaFin: fin, Motivo: motivo, CodigosRuta: copia, RelacionRef: relacion}
	if c.Validar() != nil {
		return ComisionBorrador{}, ErrComisionBorradorInvalida
	}
	return c, nil
}

func fechaCivilValida(valor string) bool {
	if len(valor) != len("2006-01-02") {
		return false
	}
	_, err := time.Parse("2006-01-02", valor)
	return err == nil
}

func referenciaRelacionValida(valor string) bool { return referenciaOpacaValida(valor, "rel_") }

func referenciaOpacaValida(valor, prefijo string) bool {
	if len(valor) < len(prefijo)+22 || len(valor) > len(prefijo)+128 || len(valor) <= len(prefijo) || valor[:len(prefijo)] != prefijo {
		return false
	}
	for _, r := range valor[len(prefijo):] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// espacioDeBorde reúne los blancos que recorta strings.TrimSpace (Unicode
// White_Space, con U+0085) y los que recorta String.prototype.trim del
// cliente web (además U+FEFF). El cliente recorta este mismo conjunto.
func espacioDeBorde(r rune) bool { return unicode.IsSpace(r) || r == '\uFEFF' }

// TextoSinBordes indica que un texto libre no empieza ni acaba por un blanco
// que Go o el navegador recortarían: así un motivo aceptado aquí es el mismo
// que la persona titular ve y que su cliente acepta.
func TextoSinBordes(valor string) bool { return strings.TrimFunc(valor, espacioDeBorde) == valor }

func textoVisible(valor string) bool {
	for _, r := range valor {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}
