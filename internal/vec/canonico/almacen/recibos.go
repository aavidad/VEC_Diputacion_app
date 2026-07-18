package almacen

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
)

func HMACSHA256Valido(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" &&
		ReferenciaOpacaValida(partes[1], 64) && SHA256HexadecimalValido(partes[2])
}

// ReciboCargaDirecta es un secreto efimero opaco de un solo uso.
type ReciboCargaDirecta struct {
	valor string
}

func NuevoReciboCargaDirecta(valor string) (ReciboCargaDirecta, error) {
	if !ReferenciaOpacaValida(valor, 1024) {
		return ReciboCargaDirecta{}, ErrReciboCargaDirectaNoValido
	}
	return ReciboCargaDirecta{valor: valor}, nil
}

func (r ReciboCargaDirecta) Valido() bool {
	return ReferenciaOpacaValida(r.valor, 1024)
}

func (r ReciboCargaDirecta) RevelarParaEntregaOConsumo() (string, error) {
	if !r.Valido() {
		return "", ErrReciboCargaDirectaNoValido
	}
	return r.valor, nil
}

func (ReciboCargaDirecta) String() string   { return "[RECIBO-CARGA-DIRECTA-CONFIDENCIAL]" }
func (ReciboCargaDirecta) GoString() string { return "[RECIBO-CARGA-DIRECTA-CONFIDENCIAL]" }

func (r ReciboCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

func (r ReciboCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (ReciboCargaDirecta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (ReciboCargaDirecta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

// RegistroReciboCargaDirecta es la propuesta sin fecha de alta del proceso;
// el repositorio fija RegistradoEn con su reloj transaccional.
type RegistroReciboCargaDirecta struct {
	IndiceHMAC             string
	GrupoHMAC              string
	VinculoHMAC            string
	EvidenciaAltaRef       string
	AutorizacionEmisionRef string
	ExpiraEn               time.Time
}

func (r RegistroReciboCargaDirecta) Validar() error {
	if !HMACSHA256Valido(r.IndiceHMAC) || !HMACSHA256Valido(r.GrupoHMAC) ||
		!HMACSHA256Valido(r.VinculoHMAC) || !ReferenciaOpacaValida(r.EvidenciaAltaRef, 512) ||
		r.ExpiraEn.IsZero() || r.ExpiraEn.Location() != time.UTC ||
		!ReferenciaOpacaValida(r.AutorizacionEmisionRef, 512) {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

// PredecesorReciboCargaDirecta liga el recibo sustituido en la misma
// transaccion que registra el nuevo recibo activo del grupo.
type PredecesorReciboCargaDirecta struct {
	IndiceHMAC             string
	GrupoHMAC              string
	AutorizacionEmisionRef string
	SustituidoEn           time.Time
}

func (p PredecesorReciboCargaDirecta) Validar() error {
	if !HMACSHA256Valido(p.IndiceHMAC) || !HMACSHA256Valido(p.GrupoHMAC) ||
		!ReferenciaOpacaValida(p.AutorizacionEmisionRef, 512) ||
		p.SustituidoEn.IsZero() || p.SustituidoEn.Location() != time.UTC {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

// ResultadoRegistroReciboCargaDirecta acredita el alta durable y su posible
// predecesor, ambos fechados por la misma transaccion.
type ResultadoRegistroReciboCargaDirecta struct {
	IndiceHMAC             string
	GrupoHMAC              string
	AutorizacionEmisionRef string
	RegistradoEn           time.Time
	Predecesor             *PredecesorReciboCargaDirecta
}

func (r ResultadoRegistroReciboCargaDirecta) ValidarContra(registro RegistroReciboCargaDirecta) error {
	if registro.Validar() != nil || r.IndiceHMAC != registro.IndiceHMAC ||
		r.GrupoHMAC != registro.GrupoHMAC || r.AutorizacionEmisionRef != registro.AutorizacionEmisionRef ||
		r.RegistradoEn.IsZero() || r.RegistradoEn.Location() != time.UTC ||
		!r.RegistradoEn.Before(registro.ExpiraEn) ||
		registro.ExpiraEn.Sub(r.RegistradoEn) > DuracionMaximaInstruccionesCargaDirecta {
		return ErrReciboCargaDirectaNoValido
	}
	if r.Predecesor == nil {
		return nil
	}
	if r.Predecesor.Validar() != nil || r.Predecesor.IndiceHMAC == registro.IndiceHMAC ||
		r.Predecesor.GrupoHMAC != registro.GrupoHMAC || !r.Predecesor.SustituidoEn.Equal(r.RegistradoEn) {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

// OrdenConsumoReciboCargaDirecta contiene las ligaduras exactas de la
// escritura condicional que consume un recibo.
type OrdenConsumoReciboCargaDirecta struct {
	IndiceHMAC               string
	GrupoHMAC                string
	VinculoHMAC              string
	EvidenciaConsumoRef      string
	IntencionConfirmacionRef string
	HuellaIntencionHMAC      string
	RegistradoEn             time.Time
	ValidaHasta              time.Time
}

func (o OrdenConsumoReciboCargaDirecta) Validar() error {
	if !HMACSHA256Valido(o.IndiceHMAC) || !HMACSHA256Valido(o.GrupoHMAC) ||
		!HMACSHA256Valido(o.VinculoHMAC) || !ReferenciaOpacaValida(o.EvidenciaConsumoRef, 512) ||
		!ReferenciaOpacaValida(o.IntencionConfirmacionRef, 512) ||
		!HMACSHA256Valido(o.HuellaIntencionHMAC) || o.RegistradoEn.IsZero() ||
		o.RegistradoEn.Location() != time.UTC || o.ValidaHasta.IsZero() || o.ValidaHasta.Location() != time.UTC {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

// ResultadoConsumoReciboCargaDirecta acredita las fechas autoritativas y la
// ligadura exacta persistida por el repositorio.
type ResultadoConsumoReciboCargaDirecta struct {
	IndiceHMAC               string
	GrupoHMAC                string
	VinculoHMAC              string
	EvidenciaConsumoRef      string
	IntencionConfirmacionRef string
	HuellaIntencionHMAC      string
	RegistradoEn             time.Time
	ConsumidoEn              time.Time
	ExpiraEn                 time.Time
}

func (r ResultadoConsumoReciboCargaDirecta) Validar() error {
	if !HMACSHA256Valido(r.IndiceHMAC) || !HMACSHA256Valido(r.GrupoHMAC) ||
		!HMACSHA256Valido(r.VinculoHMAC) || !ReferenciaOpacaValida(r.EvidenciaConsumoRef, 512) ||
		!ReferenciaOpacaValida(r.IntencionConfirmacionRef, 512) ||
		!HMACSHA256Valido(r.HuellaIntencionHMAC) || r.RegistradoEn.IsZero() ||
		r.ConsumidoEn.IsZero() || r.ExpiraEn.IsZero() || r.RegistradoEn.Location() != time.UTC ||
		r.ConsumidoEn.Location() != time.UTC || r.ExpiraEn.Location() != time.UTC ||
		r.ConsumidoEn.Before(r.RegistradoEn) || !r.ConsumidoEn.Before(r.ExpiraEn) ||
		r.ExpiraEn.Sub(r.RegistradoEn) > DuracionMaximaInstruccionesCargaDirecta {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

func (r ResultadoConsumoReciboCargaDirecta) ValidarContra(o OrdenConsumoReciboCargaDirecta) error {
	if o.Validar() != nil || r.Validar() != nil || r.IndiceHMAC != o.IndiceHMAC ||
		r.GrupoHMAC != o.GrupoHMAC || r.VinculoHMAC != o.VinculoHMAC ||
		r.EvidenciaConsumoRef != o.EvidenciaConsumoRef ||
		r.IntencionConfirmacionRef != o.IntencionConfirmacionRef || !r.RegistradoEn.Equal(o.RegistradoEn) ||
		r.HuellaIntencionHMAC != o.HuellaIntencionHMAC || !r.ConsumidoEn.Before(o.ValidaHasta) {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

// DatosComprobanteConsumoReciboCargaDirecta es una copia validada sin recibo
// ni sesion. No permite reconstruir o alterar el comprobante opaco.
type DatosComprobanteConsumoReciboCargaDirecta struct {
	IndiceReciboHMAC    string
	GrupoReciboHMAC     string
	VinculoReciboHMAC   string
	EvidenciaConsumoRef string
	IntencionRef        string
	HuellaIntencionHMAC string
	RegistradoEn        time.Time
	ConsumidoEn         time.Time
	ExpiraEn            time.Time
	ValidaHasta         time.Time
	AtestacionHMAC      string
}

// ComprobanteConsumoReciboCargaDirecta conserva de forma opaca la atestacion
// del consumo durable; su forma valida no prueba por si sola la autenticidad.
type ComprobanteConsumoReciboCargaDirecta struct {
	datos DatosComprobanteConsumoReciboCargaDirecta
}

func NuevoComprobanteConsumoReciboCargaDirecta(
	resultado ResultadoConsumoReciboCargaDirecta,
	validaHasta time.Time,
	atestacionHMAC string,
) (ComprobanteConsumoReciboCargaDirecta, error) {
	datos := DatosComprobanteConsumoReciboCargaDirecta{
		IndiceReciboHMAC: resultado.IndiceHMAC, GrupoReciboHMAC: resultado.GrupoHMAC,
		VinculoReciboHMAC: resultado.VinculoHMAC, EvidenciaConsumoRef: resultado.EvidenciaConsumoRef,
		IntencionRef: resultado.IntencionConfirmacionRef, HuellaIntencionHMAC: resultado.HuellaIntencionHMAC,
		RegistradoEn: resultado.RegistradoEn, ConsumidoEn: resultado.ConsumidoEn, ExpiraEn: resultado.ExpiraEn,
		ValidaHasta: validaHasta, AtestacionHMAC: atestacionHMAC,
	}
	comprobante := ComprobanteConsumoReciboCargaDirecta{datos: datos}
	if resultado.Validar() != nil || comprobante.ValidarEstructura() != nil {
		return ComprobanteConsumoReciboCargaDirecta{}, ErrReciboCargaDirectaNoValido
	}
	return comprobante, nil
}

func (c ComprobanteConsumoReciboCargaDirecta) ValidarEstructura() error {
	return ValidarDatosComprobanteConsumoReciboCargaDirecta(c.datos)
}

// ValidarDatosComprobanteConsumoReciboCargaDirecta aplica las mismas reglas
// a una proyeccion procedente de un contrato que mantiene su envoltorio opaco.
func ValidarDatosComprobanteConsumoReciboCargaDirecta(d DatosComprobanteConsumoReciboCargaDirecta) error {
	if !HMACSHA256Valido(d.IndiceReciboHMAC) || !HMACSHA256Valido(d.GrupoReciboHMAC) ||
		!HMACSHA256Valido(d.VinculoReciboHMAC) || !ReferenciaOpacaValida(d.EvidenciaConsumoRef, 512) ||
		!ReferenciaOpacaValida(d.IntencionRef, 512) || !HMACSHA256Valido(d.HuellaIntencionHMAC) ||
		d.RegistradoEn.IsZero() || d.ConsumidoEn.IsZero() || d.ExpiraEn.IsZero() || d.ValidaHasta.IsZero() ||
		d.RegistradoEn.Location() != time.UTC || d.ConsumidoEn.Location() != time.UTC ||
		d.ExpiraEn.Location() != time.UTC || d.ValidaHasta.Location() != time.UTC ||
		d.ConsumidoEn.Before(d.RegistradoEn) || !d.ConsumidoEn.Before(d.ExpiraEn) ||
		!d.ConsumidoEn.Before(d.ValidaHasta) ||
		d.ExpiraEn.Sub(d.RegistradoEn) > DuracionMaximaInstruccionesCargaDirecta ||
		!HMACSHA256Valido(d.AtestacionHMAC) {
		return ErrReciboCargaDirectaNoValido
	}
	return nil
}

func (c ComprobanteConsumoReciboCargaDirecta) DatosVerificados() (
	DatosComprobanteConsumoReciboCargaDirecta,
	error,
) {
	if err := c.ValidarEstructura(); err != nil {
		return DatosComprobanteConsumoReciboCargaDirecta{}, err
	}
	return c.datos, nil
}

func (c ComprobanteConsumoReciboCargaDirecta) RevelarParaVerificacion() (
	indiceHMAC, grupoHMAC, vinculoHMAC, evidenciaConsumoRef, intencionRef, huellaIntencionHMAC string,
	registradoEn, consumidoEn, expiraEn, validaHasta time.Time,
	atestacionHMAC string,
	err error,
) {
	datos, err := c.DatosVerificados()
	if err != nil {
		return "", "", "", "", "", "", time.Time{}, time.Time{}, time.Time{}, time.Time{}, "", err
	}
	return datos.IndiceReciboHMAC, datos.GrupoReciboHMAC, datos.VinculoReciboHMAC,
		datos.EvidenciaConsumoRef, datos.IntencionRef, datos.HuellaIntencionHMAC,
		datos.RegistradoEn, datos.ConsumidoEn, datos.ExpiraEn, datos.ValidaHasta, datos.AtestacionHMAC, nil
}

func (ComprobanteConsumoReciboCargaDirecta) String() string {
	return "[COMPROBANTE-CONSUMO-RECIBO-CARGA-DIRECTA-ATESTADO]"
}

func (ComprobanteConsumoReciboCargaDirecta) GoString() string {
	return "[COMPROBANTE-CONSUMO-RECIBO-CARGA-DIRECTA-ATESTADO]"
}

func (c ComprobanteConsumoReciboCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (ComprobanteConsumoReciboCargaDirecta) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (ComprobanteConsumoReciboCargaDirecta) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionReciboCargaProhibida
}

func (c ComprobanteConsumoReciboCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(c.String())
}
