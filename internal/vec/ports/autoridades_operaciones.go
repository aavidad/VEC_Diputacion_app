package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrEstadoFuenteAutoridadInvalido        = errors.New("vec: estado exacto de fuente de autoridad invalido")
	ErrOperacionFuenteAutoridadInvalida     = errors.New("vec: operacion de fuente de autoridad invalida")
	ErrOperacionFuenteAutoridadNoEncontrada = errors.New("vec: operacion de fuente de autoridad no encontrada")
	ErrSerializacionOperacionAutoridad      = errors.New("vec: serializacion de operacion de fuente de autoridad prohibida")
)

// ReferenciaEstadoFuenteAutoridad es la precondicion OCC completa. La
// referencia de contenido, por si sola, no detecta una transicion de estado.
type ReferenciaEstadoFuenteAutoridad struct {
	Fuente               domain.ReferenciaFuenteAutoridad
	Revision             uint64
	Estado               domain.EstadoFuenteAutoridad
	HuellaHistoriaSHA256 string
	HuellaEstadoSHA256   string
}

func (r ReferenciaEstadoFuenteAutoridad) Validar() error {
	if r.Fuente.Validar() != nil || r.Revision == 0 || !r.Estado.Valido() ||
		!huellaSHA256PuertoAutoridadValida(r.HuellaHistoriaSHA256) ||
		!huellaSHA256PuertoAutoridadValida(r.HuellaEstadoSHA256) {
		return ErrEstadoFuenteAutoridadInvalido
	}
	return nil
}

func EstadoExactoFuenteAutoridad(
	fuente domain.FuenteAutoridadVersionada,
) (ReferenciaEstadoFuenteAutoridad, error) {
	canonica, err := fuente.ClonarCanonica()
	if err != nil {
		return ReferenciaEstadoFuenteAutoridad{}, ErrEstadoFuenteAutoridadInvalido
	}
	referencia, errReferencia := canonica.ReferenciaExacta()
	huella, errHuella := canonica.HuellaEstadoSHA256()
	estado := ReferenciaEstadoFuenteAutoridad{
		Fuente: referencia, Revision: canonica.Revision, Estado: canonica.Estado,
		HuellaHistoriaSHA256: cabezaHistoriaFuenteAutoridad(canonica), HuellaEstadoSHA256: huella,
	}
	if errReferencia != nil || errHuella != nil || estado.Validar() != nil {
		return ReferenciaEstadoFuenteAutoridad{}, ErrEstadoFuenteAutoridadInvalido
	}
	return estado, nil
}

type EstadoOperacionFuenteAutoridad string

const (
	EstadoOperacionFuenteAutoridadPendiente  EstadoOperacionFuenteAutoridad = "pendiente"
	EstadoOperacionFuenteAutoridadAtestada   EstadoOperacionFuenteAutoridad = "atestada"
	EstadoOperacionFuenteAutoridadConfirmada EstadoOperacionFuenteAutoridad = "confirmada"
	EstadoOperacionFuenteAutoridadCancelada  EstadoOperacionFuenteAutoridad = "cancelada"
	EstadoOperacionFuenteAutoridadExpirada   EstadoOperacionFuenteAutoridad = "expirada"
	EstadoOperacionFuenteAutoridadObsoleta   EstadoOperacionFuenteAutoridad = "obsoleta"
)

func (e EstadoOperacionFuenteAutoridad) Valido() bool {
	switch e {
	case EstadoOperacionFuenteAutoridadPendiente,
		EstadoOperacionFuenteAutoridadAtestada,
		EstadoOperacionFuenteAutoridadConfirmada,
		EstadoOperacionFuenteAutoridadCancelada,
		EstadoOperacionFuenteAutoridadExpirada,
		EstadoOperacionFuenteAutoridadObsoleta:
		return true
	default:
		return false
	}
}

func (e EstadoOperacionFuenteAutoridad) Terminal() bool {
	return e == EstadoOperacionFuenteAutoridadConfirmada ||
		e == EstadoOperacionFuenteAutoridadCancelada ||
		e == EstadoOperacionFuenteAutoridadExpirada ||
		e == EstadoOperacionFuenteAutoridadObsoleta
}

// DatosOperacionFuenteAutoridad es una proyeccion interna reconstruible. Los
// conectores persisten los bytes canonicos de Solicitud, no una copia de sus
// parametros. Atestacion y resolucion son referencias a registros durables.
type DatosOperacionFuenteAutoridad struct {
	bloqueoSerializacionOperacionAutoridad
	OperacionRef           string
	Solicitud              domain.SolicitudTransicionFuenteAutoridadV1
	EstadoEsperado         ReferenciaEstadoFuenteAutoridad
	Estado                 EstadoOperacionFuenteAutoridad
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	ResolucionRef          string
	PreparadaEn            time.Time
	ActualizadaEn          time.Time
}

func (DatosOperacionFuenteAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOperacionAutoridad
}

func (*DatosOperacionFuenteAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionOperacionAutoridad
}

func (DatosOperacionFuenteAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionOperacionAutoridad
}

func (*DatosOperacionFuenteAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionOperacionAutoridad
}

func (DatosOperacionFuenteAutoridad) String() string {
	return "[DATOS-OPERACION-FUENTE-AUTORIDAD-INTERNOS]"
}

func (d DatosOperacionFuenteAutoridad) GoString() string { return d.String() }
func (d DatosOperacionFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosOperacionFuenteAutoridad) LogValue() slog.Value { return slog.StringValue(d.String()) }

// OperacionFuenteAutoridad evita que un callback reconstruya actor, motivo,
// accion o revision a partir de parametros externos.
type OperacionFuenteAutoridad struct {
	bloqueoSerializacionOperacionAutoridad
	datos *DatosOperacionFuenteAutoridad
}

func NuevaOperacionPendienteFuenteAutoridad(
	operacionRef string,
	solicitud domain.SolicitudTransicionFuenteAutoridadV1,
	esperado ReferenciaEstadoFuenteAutoridad,
) (OperacionFuenteAutoridad, error) {
	compromiso, err := solicitud.Compromiso()
	datos := DatosOperacionFuenteAutoridad{
		OperacionRef: operacionRef, Solicitud: solicitud, EstadoEsperado: esperado,
		Estado: EstadoOperacionFuenteAutoridadPendiente,
	}
	if err == nil {
		datos.PreparadaEn = compromiso.PreparadaEn
		datos.ActualizadaEn = compromiso.PreparadaEn
	}
	return RehidratarOperacionFuenteAutoridad(datos)
}

func RehidratarOperacionFuenteAutoridad(
	datos DatosOperacionFuenteAutoridad,
) (OperacionFuenteAutoridad, error) {
	copia, err := clonarDatosOperacionFuenteAutoridad(datos)
	if err != nil || validarDatosOperacionFuenteAutoridad(copia) != nil {
		return OperacionFuenteAutoridad{}, ErrOperacionFuenteAutoridadInvalida
	}
	return OperacionFuenteAutoridad{datos: &copia}, nil
}

func (o OperacionFuenteAutoridad) Datos() (DatosOperacionFuenteAutoridad, error) {
	if o.datos == nil || validarDatosOperacionFuenteAutoridad(*o.datos) != nil {
		return DatosOperacionFuenteAutoridad{}, ErrOperacionFuenteAutoridadInvalida
	}
	return clonarDatosOperacionFuenteAutoridad(*o.datos)
}

func (o OperacionFuenteAutoridad) Terminal() bool {
	datos, err := o.Datos()
	return err == nil && datos.Estado.Terminal()
}

func (OperacionFuenteAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOperacionAutoridad
}

func (*OperacionFuenteAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionOperacionAutoridad
}

func (OperacionFuenteAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionOperacionAutoridad
}

func (*OperacionFuenteAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionOperacionAutoridad
}

func (OperacionFuenteAutoridad) String() string     { return "[OPERACION-FUENTE-AUTORIDAD-INTERNA]" }
func (o OperacionFuenteAutoridad) GoString() string { return o.String() }
func (o OperacionFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, o.String())
}
func (o OperacionFuenteAutoridad) LogValue() slog.Value { return slog.StringValue(o.String()) }

// SelectorOperacionFuenteAutoridad permite validar una referencia recibida
// antes de consultar el repositorio. No concede acceso ni acredita que la
// operacion exista.
type SelectorOperacionFuenteAutoridad struct {
	OperacionRef string
}

func (s SelectorOperacionFuenteAutoridad) Validar() error {
	if !referenciaPuertoAutoridadValida(s.OperacionRef) {
		return ErrOperacionFuenteAutoridadInvalida
	}
	return nil
}

type ConsultaOperacionesFuentesAutoridad interface {
	ObtenerOperacion(context.Context, SelectorOperacionFuenteAutoridad) (OperacionFuenteAutoridad, error)
}

func validarDatosOperacionFuenteAutoridad(datos DatosOperacionFuenteAutoridad) error {
	compromiso, err := datos.Solicitud.Compromiso()
	if err != nil || !referenciaPuertoAutoridadValida(datos.OperacionRef) ||
		datos.EstadoEsperado.Validar() != nil || !datos.Estado.Valido() ||
		!instantePuertoAutoridadCanonico(datos.PreparadaEn) ||
		!instantePuertoAutoridadCanonico(datos.ActualizadaEn) ||
		datos.ActualizadaEn.Before(datos.PreparadaEn) || compromiso.PreparadaEn != datos.PreparadaEn ||
		compromiso.Fuente != datos.EstadoEsperado.Fuente ||
		compromiso.RevisionPrevia != datos.EstadoEsperado.Revision ||
		compromiso.EstadoAnterior != datos.EstadoEsperado.Estado ||
		compromiso.HuellaHistoriaPreviaSHA256 != datos.EstadoEsperado.HuellaHistoriaSHA256 {
		return ErrOperacionFuenteAutoridadInvalida
	}
	tieneAtestacion := referenciaPuertoAutoridadValida(datos.AtestacionRef) &&
		huellaSHA256PuertoAutoridadValida(datos.HuellaAtestacionSHA256)
	atestacionVacia := datos.AtestacionRef == "" && datos.HuellaAtestacionSHA256 == ""
	tieneResolucion := referenciaPuertoAutoridadValida(datos.ResolucionRef)
	switch datos.Estado {
	case EstadoOperacionFuenteAutoridadPendiente:
		if !datos.ActualizadaEn.Equal(datos.PreparadaEn) || !atestacionVacia || datos.ResolucionRef != "" {
			return ErrOperacionFuenteAutoridadInvalida
		}
	case EstadoOperacionFuenteAutoridadAtestada:
		if !tieneAtestacion || tieneResolucion || !datos.ActualizadaEn.Before(compromiso.ExpiraEn) {
			return ErrOperacionFuenteAutoridadInvalida
		}
	case EstadoOperacionFuenteAutoridadConfirmada:
		if !tieneAtestacion || !tieneResolucion || !datos.ActualizadaEn.Before(compromiso.ExpiraEn) {
			return ErrOperacionFuenteAutoridadInvalida
		}
	case EstadoOperacionFuenteAutoridadCancelada:
		if (!tieneAtestacion && !atestacionVacia) || !tieneResolucion ||
			!datos.ActualizadaEn.Before(compromiso.ExpiraEn) {
			return ErrOperacionFuenteAutoridadInvalida
		}
	case EstadoOperacionFuenteAutoridadExpirada:
		if (!tieneAtestacion && !atestacionVacia) || !tieneResolucion ||
			datos.ActualizadaEn.Before(compromiso.ExpiraEn) {
			return ErrOperacionFuenteAutoridadInvalida
		}
	case EstadoOperacionFuenteAutoridadObsoleta:
		if (!tieneAtestacion && !atestacionVacia) || !tieneResolucion {
			return ErrOperacionFuenteAutoridadInvalida
		}
	}
	return nil
}

func cabezaHistoriaFuenteAutoridad(fuente domain.FuenteAutoridadVersionada) string {
	if total := len(fuente.Transiciones); total != 0 {
		return fuente.Transiciones[total-1].HuellaHistoriaNuevaSHA256
	}
	if total := len(fuente.EdicionesBorrador); total != 0 {
		return fuente.EdicionesBorrador[total-1].HuellaHistoriaNuevaSHA256
	}
	return fuente.HuellaHistoriaInicialSHA256
}

func clonarDatosOperacionFuenteAutoridad(
	datos DatosOperacionFuenteAutoridad,
) (DatosOperacionFuenteAutoridad, error) {
	bytesSolicitud, err := datos.Solicitud.BytesCanonicos()
	if err != nil {
		return DatosOperacionFuenteAutoridad{}, ErrOperacionFuenteAutoridadInvalida
	}
	solicitud, err := domain.RehidratarSolicitudTransicionFuenteAutoridadV1(bytesSolicitud)
	if err != nil {
		return DatosOperacionFuenteAutoridad{}, ErrOperacionFuenteAutoridadInvalida
	}
	datos.Solicitud = solicitud
	return datos, nil
}

func referenciaPuertoAutoridadValida(valor string) bool {
	if valor == "" || len(valor) > 512 || valor != strings.TrimSpace(valor) {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') {
			continue
		}
		switch caracter {
		case '-', '_', '.', ':', '/', '@', '+':
			continue
		default:
			return false
		}
	}
	return true
}

func huellaSHA256PuertoAutoridadValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	contenido, err := hex.DecodeString(valor)
	return err == nil && len(contenido) == sha256.Size &&
		valor != strings.Repeat("0", sha256.Size*2)
}

func instantePuertoAutoridadCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 && instante.Nanosecond()%1_000 == 0
}

type bloqueoSerializacionOperacionAutoridad struct{}

func (bloqueoSerializacionOperacionAutoridad) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionOperacionAutoridad
}

func (*bloqueoSerializacionOperacionAutoridad) UnmarshalBinary([]byte) error {
	return ErrSerializacionOperacionAutoridad
}

func (bloqueoSerializacionOperacionAutoridad) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionOperacionAutoridad
}

func (*bloqueoSerializacionOperacionAutoridad) GobDecode([]byte) error {
	return ErrSerializacionOperacionAutoridad
}

func (bloqueoSerializacionOperacionAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionOperacionAutoridad
}

func (*bloqueoSerializacionOperacionAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionOperacionAutoridad
}
