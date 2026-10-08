package httpseguridad

import (
	"context"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrPresentacionCertificadoNoValida = errors.New("presentacion de certificado no valida")

// AcreditacionCertificadoActual solo procede del certificador fijado al
// construir el servicio. La biblioteca TLS aporta la cadena verificada; el
// certificador acredita el registro privado y la CRL actuales.
type AcreditacionCertificadoActual struct {
	bloqueoSerializacionPresentacionCertificado
	SujetoID               string
	CuentaID               string
	CertificadoSHA256      string
	CASHA256               string
	CertificadoValidoHasta time.Time
	CAValidaHasta          time.Time
	CRLSiguienteEn         time.Time
	RevocacionVerificadaEn time.Time
}

type CertificadorPresentacionActual interface {
	AcreditarCertificadoActual(context.Context, []*x509.Certificate, time.Time) (AcreditacionCertificadoActual, error)
}

// PruebaCertificadoActual queda ligada a una sola instancia de la autoridad,
// al certificado/CA del handshake y al exportador TLS de la peticion actual.
// Ni el cliente ni un DTO de HTTP pueden construirla desde huellas y fechas.
type PruebaCertificadoActual struct {
	bloqueoSerializacionPresentacionCertificado
	datos *datosPruebaCertificadoActual
}

type datosPruebaCertificadoActual struct {
	servicio          *ServicioPresentacionCertificado
	marcador          *marcaPeticionCertificadoActual
	canalRef          string
	certificadoSHA256 string
	caSHA256          string
	sujetoID          string
	cuentaID          string
	verificadaEn      time.Time
	certificadoHasta  time.Time
	caHasta           time.Time
	crlSiguienteEn    time.Time
	consumida         atomic.Bool
}

type clavePeticionCertificadoActual struct{}

// No puede ser de tamaño cero: dos direcciones de valores vacíos pueden ser
// iguales en Go y confundir el cotejo de peticiones distintas.
type marcaPeticionCertificadoActual struct{ indice byte }

// DatosPresentacionCertificado lleva exclusivamente la prueba ya verificada
// hacia el adaptador durable; no contiene DER, claves ni decision ejecutable.
type DatosPresentacionCertificado struct {
	bloqueoSerializacionPresentacionCertificado
	OperacionRef                 string
	AsercionID                   string
	SesionIDAfirmada             string
	SujetoID                     string
	CuentaID                     string
	CertificadoSHA256            string
	CASHA256                     string
	CanalRef                     string
	CanalSHA256                  string
	Superficie                   Superficie
	Emisor                       string
	Audiencia                    string
	ACRVerificado                string
	MetodoObservado              domain.AuthMethod
	GarantiaObservada            domain.AuthAssurance
	PoliticaGarantiaRef          string
	PoliticaGarantiaHuellaSHA256 string
	AsercionActualHuellaSHA256   string
	AsercionActualEmitidaEn      time.Time
	AsercionActualExpiraEn       time.Time
	CertificadoVerificadoEn      time.Time
	CertificadoValidoHasta       time.Time
	CAValidaHasta                time.Time
	CRLSiguienteActualizacion    time.Time
	PoliticaRetiradaEn           time.Time
}

// OrdenInicioCertificado incluye el alta original solo en la primera
// operacion. El registro debe confirmar alta, vinculo y primera presentacion
// en una transaccion. Una sesion previa, incluso revocada, impide esta orden.
type OrdenInicioCertificado struct {
	bloqueoSerializacionPresentacionCertificado
	datos *datosOrdenInicioCertificado
}

type datosOrdenInicioCertificado struct {
	alta         AltaSesionAtomica
	presentacion DatosPresentacionCertificado
}

func (o OrdenInicioCertificado) Datos() (AltaSesionAtomica, DatosPresentacionCertificado, error) {
	if o.datos == nil || o.datos.alta.Validar() != nil || validarDatosPresentacionCertificado(o.datos.presentacion) != nil {
		return AltaSesionAtomica{}, DatosPresentacionCertificado{}, ErrPresentacionCertificadoNoValida
	}
	return o.datos.alta, o.datos.presentacion, nil
}

// OrdenReanudacionCertificado carece deliberadamente de AltaSesionAtomica.
// La sesion original se selecciona en SQL desde el vinculo durable.
type OrdenReanudacionCertificado struct {
	bloqueoSerializacionPresentacionCertificado
	datos *DatosPresentacionCertificado
}

func (o OrdenReanudacionCertificado) Datos() (DatosPresentacionCertificado, error) {
	if o.datos == nil || validarDatosPresentacionCertificado(*o.datos) != nil {
		return DatosPresentacionCertificado{}, ErrPresentacionCertificadoNoValida
	}
	return *o.datos, nil
}

// ReciboPresentacionCertificado corresponde al nuevo nonce consumido y al
// canal actual. Sus referencias no sustituyen las de la sesion original.
type ReciboPresentacionCertificado struct {
	bloqueoSerializacionPresentacionCertificado
	ModoInicio                 string
	OperacionRef               string
	PresentacionRef            string
	Generacion                 uint64
	SesionGeneracion           uint64
	HuellaSHA256               string
	RegistradaEn               time.Time
	ValidaHasta                time.Time
	CanalSHA256                string
	AsercionActualHuellaSHA256 string
}

// ResultadoRegistroPresentacionCertificado es el resultado del puerto durable.
// La autoridad valida cada campo contra la orden antes de emitir la capsula.
type ResultadoRegistroPresentacionCertificado struct {
	bloqueoSerializacionPresentacionCertificado
	SesionOriginal domain.AutenticacionRevalidadaV1
	Recibo         ReciboPresentacionCertificado
}

type RegistroPresentacionesCertificado interface {
	IniciarYConsumirPresentacion(context.Context, OrdenInicioCertificado) (ResultadoRegistroPresentacionCertificado, error)
	ReanudarYConsumirPresentacion(context.Context, OrdenReanudacionCertificado) (ResultadoRegistroPresentacionCertificado, error)
}

// CapsulaPresentacionCertificado es una capacidad por petición. Mantiene los
// datos originales separados de la asercion actual; no se reconstruye desde
// bytes, ni sustituye una decision V3 de recurso.
type CapsulaPresentacionCertificado struct {
	bloqueoSerializacionPresentacionCertificado
	datos *datosCapsulaPresentacionCertificado
}

type datosCapsulaPresentacionCertificado struct {
	servicio        *ServicioPresentacionCertificado
	marcador        *marcaPeticionCertificadoActual
	canalRef        string
	inicioExplicito bool
	tokenEmitido    atomic.Bool
	estadoActual    estadoIdentidadSesion
	resultado       ResultadoRegistroPresentacionCertificado
	presentacion    DatosPresentacionCertificado
	consumida       atomic.Bool
}

type bloqueoSerializacionPresentacionCertificado struct{}

func (bloqueoSerializacionPresentacionCertificado) MarshalJSON() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (*bloqueoSerializacionPresentacionCertificado) UnmarshalJSON([]byte) error {
	return ErrPresentacionCertificadoNoValida
}
func (bloqueoSerializacionPresentacionCertificado) MarshalText() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (*bloqueoSerializacionPresentacionCertificado) UnmarshalText([]byte) error {
	return ErrPresentacionCertificadoNoValida
}
func (bloqueoSerializacionPresentacionCertificado) MarshalBinary() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (*bloqueoSerializacionPresentacionCertificado) UnmarshalBinary([]byte) error {
	return ErrPresentacionCertificadoNoValida
}
func (bloqueoSerializacionPresentacionCertificado) GobEncode() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (*bloqueoSerializacionPresentacionCertificado) GobDecode([]byte) error {
	return ErrPresentacionCertificadoNoValida
}
func (bloqueoSerializacionPresentacionCertificado) MarshalCBOR() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (*bloqueoSerializacionPresentacionCertificado) UnmarshalCBOR([]byte) error {
	return ErrPresentacionCertificadoNoValida
}
func (bloqueoSerializacionPresentacionCertificado) MarshalYAML() (any, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (*bloqueoSerializacionPresentacionCertificado) UnmarshalYAML(func(any) error) error {
	return ErrPresentacionCertificadoNoValida
}
func (bloqueoSerializacionPresentacionCertificado) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrPresentacionCertificadoNoValida
}
func (*bloqueoSerializacionPresentacionCertificado) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrPresentacionCertificadoNoValida
}

const presentacionCertificadoRedactada = "[PRESENTACION CERTIFICADO CONFIDENCIAL]"

func (PruebaCertificadoActual) String() string        { return presentacionCertificadoRedactada }
func (OrdenInicioCertificado) String() string         { return presentacionCertificadoRedactada }
func (OrdenReanudacionCertificado) String() string    { return presentacionCertificadoRedactada }
func (CapsulaPresentacionCertificado) String() string { return presentacionCertificadoRedactada }
func (DatosPresentacionCertificado) String() string   { return presentacionCertificadoRedactada }
func (ResultadoRegistroPresentacionCertificado) String() string {
	return presentacionCertificadoRedactada
}
func (ReciboPresentacionCertificado) String() string          { return presentacionCertificadoRedactada }
func (AcreditacionCertificadoActual) String() string          { return presentacionCertificadoRedactada }
func (p PruebaCertificadoActual) LogValue() slog.Value        { return slog.StringValue(p.String()) }
func (o OrdenInicioCertificado) LogValue() slog.Value         { return slog.StringValue(o.String()) }
func (o OrdenReanudacionCertificado) LogValue() slog.Value    { return slog.StringValue(o.String()) }
func (c CapsulaPresentacionCertificado) LogValue() slog.Value { return slog.StringValue(c.String()) }
func (d DatosPresentacionCertificado) LogValue() slog.Value   { return slog.StringValue(d.String()) }
func (r ResultadoRegistroPresentacionCertificado) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (r ReciboPresentacionCertificado) LogValue() slog.Value { return slog.StringValue(r.String()) }
func (a AcreditacionCertificadoActual) LogValue() slog.Value { return slog.StringValue(a.String()) }
func (p PruebaCertificadoActual) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, p.String()) }
func (o OrdenInicioCertificado) Format(s fmt.State, _ rune)  { _, _ = io.WriteString(s, o.String()) }
func (o OrdenReanudacionCertificado) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, o.String())
}
func (c CapsulaPresentacionCertificado) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
func (d DatosPresentacionCertificado) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, d.String())
}
func (r ResultadoRegistroPresentacionCertificado) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r ReciboPresentacionCertificado) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (a AcreditacionCertificadoActual) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, a.String())
}
func (c CapsulaPresentacionCertificado) GoString() string           { return c.String() }
func (d DatosPresentacionCertificado) GoString() string             { return d.String() }
func (r ResultadoRegistroPresentacionCertificado) GoString() string { return r.String() }
func (r ReciboPresentacionCertificado) GoString() string            { return r.String() }
func (a AcreditacionCertificadoActual) GoString() string            { return a.String() }
func (c CapsulaPresentacionCertificado) MarshalJSON() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (d DatosPresentacionCertificado) MarshalJSON() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (r ResultadoRegistroPresentacionCertificado) MarshalJSON() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (r ReciboPresentacionCertificado) MarshalJSON() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
func (a AcreditacionCertificadoActual) MarshalJSON() ([]byte, error) {
	return nil, ErrPresentacionCertificadoNoValida
}
