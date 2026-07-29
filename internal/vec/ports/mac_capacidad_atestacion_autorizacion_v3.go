package ports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"
)

var ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible = errors.New(
	"vec: calculo MAC de capacidad de atestacion V3 no disponible",
)

const (
	TamanoMACCapacidadAtestacionAutorizacionV3           = sha256.Size
	TamanoMaximoPreimagenMACCapacidadAtestacionV3        = 32 * 1024
	versionMaximaExactaMACCapacidadAtestacionV3   uint64 = 1<<53 - 1
)

// DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3 contiene solamente
// metadatos gobernados. No contiene material, referencia interna ni
// credenciales KMS/HSM y, por sí solo, no permite calcular una MAC.
type DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3 struct {
	ClaveID              string
	ClaveVersion         uint64
	RevisionGobierno     uint64
	HuellaGobiernoSHA256 string
	EmisorID             string
	AudienciaConsumo     string
	ValidaDesde          time.Time
	ValidaHasta          time.Time
}

// PerfilEmisionMACCapacidadAtestacionAutorizacionV3 identifica una única
// versión de clave y audiencia. Es informativo y no concede autoridad.
type PerfilEmisionMACCapacidadAtestacionAutorizacionV3 struct {
	bloqueoSerializacionMACCapacidadAtestacionV3
	datos DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3
}

// NuevoPerfilEmisionMACCapacidadAtestacionAutorizacionV3 comprueba forma, no
// procedencia. Fabricar metadatos no concede acceso al calculador inyectado ni
// produce una capacidad verificable por el consumidor.
func NuevoPerfilEmisionMACCapacidadAtestacionAutorizacionV3(
	datos DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3,
) (PerfilEmisionMACCapacidadAtestacionAutorizacionV3, error) {
	perfil := PerfilEmisionMACCapacidadAtestacionAutorizacionV3{datos: datos}
	if perfil.Validar() != nil {
		return PerfilEmisionMACCapacidadAtestacionAutorizacionV3{},
			ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return perfil, nil
}

func (p PerfilEmisionMACCapacidadAtestacionAutorizacionV3) Validar() error {
	d := p.datos
	if !textoResumenCapacidadV3Valido(d.ClaveID) ||
		d.ClaveVersion == 0 ||
		d.ClaveVersion > versionMaximaExactaMACCapacidadAtestacionV3 ||
		d.RevisionGobierno == 0 ||
		d.RevisionGobierno > versionMaximaExactaMACCapacidadAtestacionV3 ||
		!huellaResumenCapacidadV3Valida(d.HuellaGobiernoSHA256) ||
		!textoResumenCapacidadV3Valido(d.EmisorID) ||
		!textoResumenCapacidadV3Valido(d.AudienciaConsumo) ||
		!instanteResumenCapacidadV3Valido(d.ValidaDesde) ||
		!instanteResumenCapacidadV3Valido(d.ValidaHasta) ||
		!d.ValidaHasta.After(d.ValidaDesde) {
		return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return nil
}

func (p PerfilEmisionMACCapacidadAtestacionAutorizacionV3) ValidarEn(
	instante time.Time,
) error {
	if p.Validar() != nil || !instanteResumenCapacidadV3Valido(instante) ||
		instante.Before(p.datos.ValidaDesde) ||
		!instante.Before(p.datos.ValidaHasta) {
		return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return nil
}

func (p PerfilEmisionMACCapacidadAtestacionAutorizacionV3) Datos() (
	DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3,
	error,
) {
	if p.Validar() != nil {
		return DatosPerfilEmisionMACCapacidadAtestacionAutorizacionV3{},
			ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return p.datos, nil
}

// SolicitudCalculoMACCapacidadAtestacionAutorizacionV3 conserva la preimagen
// exacta que el adaptador debe autenticar con su clave ya preligada.
type SolicitudCalculoMACCapacidadAtestacionAutorizacionV3 struct {
	bloqueoSerializacionMACCapacidadAtestacionV3
	perfil       PerfilEmisionMACCapacidadAtestacionAutorizacionV3
	preimagen    []byte
	huella       [sha256.Size]byte
	solicitadaEn time.Time
}

func NuevaSolicitudCalculoMACCapacidadAtestacionAutorizacionV3(
	perfil PerfilEmisionMACCapacidadAtestacionAutorizacionV3,
	preimagen []byte,
	solicitadaEn time.Time,
) (SolicitudCalculoMACCapacidadAtestacionAutorizacionV3, error) {
	solicitud := SolicitudCalculoMACCapacidadAtestacionAutorizacionV3{
		perfil: perfil, preimagen: bytes.Clone(preimagen),
		huella: sha256.Sum256(preimagen), solicitadaEn: solicitadaEn,
	}
	if solicitud.Validar() != nil {
		return SolicitudCalculoMACCapacidadAtestacionAutorizacionV3{},
			ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return solicitud, nil
}

func (s SolicitudCalculoMACCapacidadAtestacionAutorizacionV3) Validar() error {
	huella := sha256.Sum256(s.preimagen)
	if s.perfil.ValidarEn(s.solicitadaEn) != nil ||
		len(s.preimagen) == 0 ||
		len(s.preimagen) > TamanoMaximoPreimagenMACCapacidadAtestacionV3 ||
		bytes.Equal(s.preimagen, make([]byte, len(s.preimagen))) ||
		subtle.ConstantTimeCompare(huella[:], s.huella[:]) != 1 {
		return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return nil
}

// MaterialParaCalculador entrega copias defensivas al adaptador confiable. No
// incluye un selector: el adaptador debe cotejar el perfil con su manejador
// inmutable antes de usar la clave no exportable.
func (s SolicitudCalculoMACCapacidadAtestacionAutorizacionV3) MaterialParaCalculador() (
	PerfilEmisionMACCapacidadAtestacionAutorizacionV3,
	[]byte,
	time.Time,
	string,
	error,
) {
	if s.Validar() != nil {
		return PerfilEmisionMACCapacidadAtestacionAutorizacionV3{},
			nil, time.Time{}, "",
			ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return s.perfil, bytes.Clone(s.preimagen), s.solicitadaEn,
		hex.EncodeToString(s.huella[:]), nil
}

// ResultadoCalculoMACCapacidadAtestacionAutorizacionV3 solo liga forma,
// perfil y preimagen. La autenticidad se vuelve a comprobar en el consumidor.
type ResultadoCalculoMACCapacidadAtestacionAutorizacionV3 struct {
	bloqueoSerializacionMACCapacidadAtestacionV3
	perfil PerfilEmisionMACCapacidadAtestacionAutorizacionV3
	huella [sha256.Size]byte
	mac    [TamanoMACCapacidadAtestacionAutorizacionV3]byte
}

func NuevoResultadoCalculoMACCapacidadAtestacionAutorizacionV3(
	solicitud SolicitudCalculoMACCapacidadAtestacionAutorizacionV3,
	mac []byte,
) (ResultadoCalculoMACCapacidadAtestacionAutorizacionV3, error) {
	var resultado ResultadoCalculoMACCapacidadAtestacionAutorizacionV3
	if solicitud.Validar() != nil ||
		len(mac) != TamanoMACCapacidadAtestacionAutorizacionV3 ||
		bytes.Equal(mac, make([]byte, len(mac))) {
		return resultado,
			ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	resultado.perfil = solicitud.perfil
	resultado.huella = solicitud.huella
	copy(resultado.mac[:], mac)
	return resultado, nil
}

func (r ResultadoCalculoMACCapacidadAtestacionAutorizacionV3) ValidarPara(
	solicitud SolicitudCalculoMACCapacidadAtestacionAutorizacionV3,
) error {
	if solicitud.Validar() != nil || r.perfil.Validar() != nil ||
		r.perfil != solicitud.perfil ||
		subtle.ConstantTimeCompare(r.huella[:], solicitud.huella[:]) != 1 ||
		bytes.Equal(r.mac[:], make([]byte, len(r.mac))) {
		return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return nil
}

func (r ResultadoCalculoMACCapacidadAtestacionAutorizacionV3) MACPara(
	solicitud SolicitudCalculoMACCapacidadAtestacionAutorizacionV3,
) ([]byte, error) {
	if r.ValidarPara(solicitud) != nil {
		return nil, ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
	}
	return bytes.Clone(r.mac[:]), nil
}

// CalculadorMACCapacidadAtestacionAutorizacionV3 representa un manejador de
// una única clave/version/audiencia. Ningún método acepta selectores libres.
type CalculadorMACCapacidadAtestacionAutorizacionV3 interface {
	PerfilEmisionMACCapacidadAtestacionAutorizacionV3(
		context.Context,
	) (PerfilEmisionMACCapacidadAtestacionAutorizacionV3, error)
	CalcularMACCapacidadAtestacionAutorizacionV3(
		context.Context,
		SolicitudCalculoMACCapacidadAtestacionAutorizacionV3,
	) (ResultadoCalculoMACCapacidadAtestacionAutorizacionV3, error)
}

func CalculadorMACCapacidadAtestacionAutorizacionV3Nulo(
	calculador CalculadorMACCapacidadAtestacionAutorizacionV3,
) bool {
	if calculador == nil {
		return true
	}
	valor := reflect.ValueOf(calculador)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

type bloqueoSerializacionMACCapacidadAtestacionV3 struct{}

func (bloqueoSerializacionMACCapacidadAtestacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (*bloqueoSerializacionMACCapacidadAtestacionV3) UnmarshalJSON([]byte) error {
	return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (bloqueoSerializacionMACCapacidadAtestacionV3) MarshalText() ([]byte, error) {
	return nil, ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (*bloqueoSerializacionMACCapacidadAtestacionV3) UnmarshalText([]byte) error {
	return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (bloqueoSerializacionMACCapacidadAtestacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (*bloqueoSerializacionMACCapacidadAtestacionV3) UnmarshalBinary([]byte) error {
	return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (bloqueoSerializacionMACCapacidadAtestacionV3) GobEncode() ([]byte, error) {
	return nil, ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (*bloqueoSerializacionMACCapacidadAtestacionV3) GobDecode([]byte) error {
	return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (bloqueoSerializacionMACCapacidadAtestacionV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (*bloqueoSerializacionMACCapacidadAtestacionV3) UnmarshalCBOR([]byte) error {
	return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (bloqueoSerializacionMACCapacidadAtestacionV3) MarshalYAML() (any, error) {
	return nil, ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (*bloqueoSerializacionMACCapacidadAtestacionV3) UnmarshalYAML(
	func(any) error,
) error {
	return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (bloqueoSerializacionMACCapacidadAtestacionV3) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (*bloqueoSerializacionMACCapacidadAtestacionV3) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrCalculoMACCapacidadAtestacionAutorizacionV3NoDisponible
}
func (bloqueoSerializacionMACCapacidadAtestacionV3) String() string {
	return "[MAC-CAPACIDAD-ATESTACION-V3-OPACA]"
}
func (b bloqueoSerializacionMACCapacidadAtestacionV3) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionMACCapacidadAtestacionV3) Format(
	estado fmt.State,
	_ rune,
) {
	_, _ = io.WriteString(estado, b.String())
}
func (b bloqueoSerializacionMACCapacidadAtestacionV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}
