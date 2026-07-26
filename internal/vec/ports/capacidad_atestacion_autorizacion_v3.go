package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var ErrResumenCapacidadAtestacionAutorizacionV3Invalido = errors.New(
	"vec: resumen de capacidad de atestacion V3 invalido",
)

// ResumenCapacidadAtestacionAutorizacionV3 permite cruzar ligaduras sin
// reinterpretar en cada módulo el protocolo privado AD-3. Es informativo: no
// verifica MAC, revocación ni consumo y nunca concede autoridad por sí solo.
type ResumenCapacidadAtestacionAutorizacionV3 struct {
	bloqueoSerializacionResumenCapacidadV3
	decisionRef          string
	decisionHuellaSHA256 string
	motivoHuellaSHA256   string
	contextoRef          string
	contextoHuellaSHA256 string
	operacion            string
	efectoRef            string
	efectoHuellaSHA256   string
	audienciaConsumo     string
	emitidaEn            time.Time
	expiraEn             time.Time
}

func NuevoResumenCapacidadAtestacionAutorizacionV3(
	decisionRef, decisionHuellaSHA256, motivoHuellaSHA256 string,
	contextoRef, contextoHuellaSHA256, operacion string,
	efectoRef, efectoHuellaSHA256, audienciaConsumo string,
	emitidaEn, expiraEn time.Time,
) (ResumenCapacidadAtestacionAutorizacionV3, error) {
	r := ResumenCapacidadAtestacionAutorizacionV3{
		decisionRef: decisionRef, decisionHuellaSHA256: decisionHuellaSHA256,
		motivoHuellaSHA256: motivoHuellaSHA256, contextoRef: contextoRef,
		contextoHuellaSHA256: contextoHuellaSHA256, operacion: operacion,
		efectoRef: efectoRef, efectoHuellaSHA256: efectoHuellaSHA256,
		audienciaConsumo: audienciaConsumo, emitidaEn: emitidaEn,
		expiraEn: expiraEn,
	}
	if r.ValidarEstructura() != nil {
		return ResumenCapacidadAtestacionAutorizacionV3{},
			ErrResumenCapacidadAtestacionAutorizacionV3Invalido
	}
	return r, nil
}

func (r ResumenCapacidadAtestacionAutorizacionV3) ValidarEstructura() error {
	if !textoResumenCapacidadV3Valido(r.decisionRef) ||
		!huellaResumenCapacidadV3Valida(r.decisionHuellaSHA256) ||
		!huellaResumenCapacidadV3Valida(r.motivoHuellaSHA256) ||
		!textoResumenCapacidadV3Valido(r.contextoRef) ||
		!huellaResumenCapacidadV3Valida(r.contextoHuellaSHA256) ||
		!textoResumenCapacidadV3Valido(r.operacion) ||
		!textoResumenCapacidadV3Valido(r.efectoRef) ||
		!huellaResumenCapacidadV3Valida(r.efectoHuellaSHA256) ||
		!textoResumenCapacidadV3Valido(r.audienciaConsumo) ||
		!instanteResumenCapacidadV3Valido(r.emitidaEn) ||
		!instanteResumenCapacidadV3Valido(r.expiraEn) ||
		!r.expiraEn.After(r.emitidaEn) ||
		r.expiraEn.Sub(r.emitidaEn) > 5*time.Second {
		return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
	}
	return nil
}

func (r ResumenCapacidadAtestacionAutorizacionV3) DecisionRef() string {
	return r.decisionRef
}
func (r ResumenCapacidadAtestacionAutorizacionV3) DecisionHuellaSHA256() string {
	return r.decisionHuellaSHA256
}
func (r ResumenCapacidadAtestacionAutorizacionV3) MotivoHuellaSHA256() string {
	return r.motivoHuellaSHA256
}
func (r ResumenCapacidadAtestacionAutorizacionV3) ContextoRef() string {
	return r.contextoRef
}
func (r ResumenCapacidadAtestacionAutorizacionV3) ContextoHuellaSHA256() string {
	return r.contextoHuellaSHA256
}
func (r ResumenCapacidadAtestacionAutorizacionV3) Operacion() string {
	return r.operacion
}
func (r ResumenCapacidadAtestacionAutorizacionV3) EfectoRef() string {
	return r.efectoRef
}
func (r ResumenCapacidadAtestacionAutorizacionV3) EfectoHuellaSHA256() string {
	return r.efectoHuellaSHA256
}
func (r ResumenCapacidadAtestacionAutorizacionV3) AudienciaConsumo() string {
	return r.audienciaConsumo
}
func (r ResumenCapacidadAtestacionAutorizacionV3) EmitidaEn() time.Time {
	return r.emitidaEn
}
func (r ResumenCapacidadAtestacionAutorizacionV3) ExpiraEn() time.Time {
	return r.expiraEn
}

// ExportadorCapacidadAtestacionAutorizacionV3 es la unica frontera que el
// futuro consumidor SQL necesita conocer. La exportacion no concede autoridad
// por su tipo Go: el consumidor independiente debe verificar de nuevo formato
// canonico, MAC, clave gobernada, rotacion, revocacion, audiencia, vigencia y
// ligaduras exactas antes del efecto.
//
// El contrato es neutral al cliente. No recibe HTTP, cookies, almacenamiento
// de navegador ni cabeceras de identidad.
type ExportadorCapacidadAtestacionAutorizacionV3 interface {
	fmt.Stringer
	slog.LogValuer
	ExportacionCanonicaParaConsumidor() ([]byte, error)
	// ResumenParaConsumidor se deriva del mismo parser estricto y canónico que
	// protege la exportación. El consumidor SQL sigue obligado a verificar y
	// consumir los bytes originales dentro de su transacción.
	ResumenParaConsumidor() (
		ResumenCapacidadAtestacionAutorizacionV3,
		error,
	)
}

type bloqueoSerializacionResumenCapacidadV3 struct{}

func (bloqueoSerializacionResumenCapacidadV3) MarshalJSON() ([]byte, error) {
	return nil, ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (*bloqueoSerializacionResumenCapacidadV3) UnmarshalJSON([]byte) error {
	return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (bloqueoSerializacionResumenCapacidadV3) MarshalText() ([]byte, error) {
	return nil, ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (*bloqueoSerializacionResumenCapacidadV3) UnmarshalText([]byte) error {
	return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (bloqueoSerializacionResumenCapacidadV3) MarshalBinary() ([]byte, error) {
	return nil, ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (*bloqueoSerializacionResumenCapacidadV3) UnmarshalBinary([]byte) error {
	return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (bloqueoSerializacionResumenCapacidadV3) GobEncode() ([]byte, error) {
	return nil, ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (*bloqueoSerializacionResumenCapacidadV3) GobDecode([]byte) error {
	return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (bloqueoSerializacionResumenCapacidadV3) MarshalCBOR() ([]byte, error) {
	return nil, ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (*bloqueoSerializacionResumenCapacidadV3) UnmarshalCBOR([]byte) error {
	return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (bloqueoSerializacionResumenCapacidadV3) MarshalYAML() (any, error) {
	return nil, ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (*bloqueoSerializacionResumenCapacidadV3) UnmarshalYAML(func(any) error) error {
	return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (bloqueoSerializacionResumenCapacidadV3) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (*bloqueoSerializacionResumenCapacidadV3) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrResumenCapacidadAtestacionAutorizacionV3Invalido
}
func (bloqueoSerializacionResumenCapacidadV3) String() string {
	return "[RESUMEN-CAPACIDAD-ATESTACION-V3-OPACO]"
}
func (b bloqueoSerializacionResumenCapacidadV3) GoString() string {
	return b.String()
}
func (b bloqueoSerializacionResumenCapacidadV3) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, b.String())
}
func (b bloqueoSerializacionResumenCapacidadV3) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

func textoResumenCapacidadV3Valido(valor string) bool {
	if valor == "" || len(valor) > 512 ||
		valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) ||
		strings.ContainsRune(valor, '*') {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return true
}

func huellaResumenCapacidadV3Valida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	contenido, err := hex.DecodeString(valor)
	return err == nil && len(contenido) == sha256.Size
}

func instanteResumenCapacidadV3Valido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}
