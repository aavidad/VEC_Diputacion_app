package ports

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

const VigenciaMaximaAtestacionActoFuenteAutoridad = 5 * time.Minute

var (
	ErrSolicitudComprobacionActoAutoridadInvalida = errors.New("vec: solicitud de comprobacion de acto de autoridad invalida")
	ErrAtestacionActoAutoridadInvalida            = errors.New("vec: atestacion de acto de autoridad invalida")
	ErrActoFuenteAutoridadNoDisponible            = errors.New("vec: acto de fuente de autoridad no disponible")
	ErrAtestacionActoAutoridadConsumida           = errors.New("vec: atestacion de acto de autoridad consumida")
	ErrSerializacionAtestacionActoAutoridad       = errors.New("vec: serializacion de atestacion de acto de autoridad prohibida")
)

// SolicitudComprobarActoFuenteAutoridad contiene la solicitud de dominio
// exacta y el snapshot OCC que la produjo. El adaptador recibe este valor desde
// aplicacion; nunca lo reconstruye con datos de un callback.
type SolicitudComprobarActoFuenteAutoridad struct {
	bloqueoSerializacionAtestacionActoAutoridad
	Solicitud      domain.SolicitudTransicionFuenteAutoridadV1
	EstadoEsperado ReferenciaEstadoFuenteAutoridad
	ComprobarEn    time.Time
}

func (s SolicitudComprobarActoFuenteAutoridad) Validar() error {
	compromiso, err := s.Solicitud.Compromiso()
	if err != nil || s.EstadoEsperado.Validar() != nil ||
		!instantePuertoAutoridadCanonico(s.ComprobarEn) ||
		s.ComprobarEn.Before(compromiso.PreparadaEn) || !s.ComprobarEn.Before(compromiso.ExpiraEn) ||
		compromiso.Fuente != s.EstadoEsperado.Fuente ||
		compromiso.RevisionPrevia != s.EstadoEsperado.Revision ||
		compromiso.EstadoAnterior != s.EstadoEsperado.Estado ||
		compromiso.HuellaHistoriaPreviaSHA256 != s.EstadoEsperado.HuellaHistoriaSHA256 {
		return ErrSolicitudComprobacionActoAutoridadInvalida
	}
	return nil
}

func (s SolicitudComprobarActoFuenteAutoridad) Clonar() (
	SolicitudComprobarActoFuenteAutoridad,
	error,
) {
	bytesSolicitud, err := s.Solicitud.BytesCanonicos()
	if err != nil || s.Validar() != nil {
		return SolicitudComprobarActoFuenteAutoridad{}, ErrSolicitudComprobacionActoAutoridadInvalida
	}
	solicitud, err := domain.RehidratarSolicitudTransicionFuenteAutoridadV1(bytesSolicitud)
	if err != nil {
		return SolicitudComprobarActoFuenteAutoridad{}, ErrSolicitudComprobacionActoAutoridadInvalida
	}
	s.Solicitud = solicitud
	return s, nil
}

func (SolicitudComprobarActoFuenteAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAtestacionActoAutoridad
}

func (*SolicitudComprobarActoFuenteAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionAtestacionActoAutoridad
}

func (SolicitudComprobarActoFuenteAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAtestacionActoAutoridad
}

func (*SolicitudComprobarActoFuenteAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionAtestacionActoAutoridad
}

func (SolicitudComprobarActoFuenteAutoridad) String() string {
	return "[SOLICITUD-COMPROBACION-ACTO-AUTORIDAD-INTERNA]"
}
func (s SolicitudComprobarActoFuenteAutoridad) GoString() string { return s.String() }
func (s SolicitudComprobarActoFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudComprobarActoFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// DatosAtestacionActoFuenteAutoridad es la vista que el repositorio relee
// antes de consumir TokenConsumoRef. Las referencias acreditan registros
// externos; este contrato no implementa ni simula criptografia.
type DatosAtestacionActoFuenteAutoridad struct {
	bloqueoSerializacionAtestacionActoAutoridad
	Evidencia                      domain.EvidenciaActoFuenteAutoridad
	RevisionEsperada               uint64
	HuellaEstadoEsperadoSHA256     string
	VerificadorRef                 string
	RegistroAtestacionRef          string
	HuellaRegistroAtestacionSHA256 string
	TokenConsumoRef                string
	EmitidaEn                      time.Time
	ValidaHasta                    time.Time
}

func (DatosAtestacionActoFuenteAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAtestacionActoAutoridad
}

func (*DatosAtestacionActoFuenteAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionAtestacionActoAutoridad
}

func (DatosAtestacionActoFuenteAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAtestacionActoAutoridad
}

func (*DatosAtestacionActoFuenteAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionAtestacionActoAutoridad
}

func (DatosAtestacionActoFuenteAutoridad) String() string {
	return "[DATOS-ATESTACION-ACTO-AUTORIDAD-INTERNOS]"
}

func (d DatosAtestacionActoFuenteAutoridad) GoString() string { return d.String() }
func (d DatosAtestacionActoFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosAtestacionActoFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

type AtestacionActoFuenteAutoridad struct {
	bloqueoSerializacionAtestacionActoAutoridad
	datos *DatosAtestacionActoFuenteAutoridad
}

func NuevaAtestacionActoFuenteAutoridad(
	solicitud SolicitudComprobarActoFuenteAutoridad,
	datos DatosAtestacionActoFuenteAutoridad,
) (AtestacionActoFuenteAutoridad, error) {
	copia, err := clonarDatosAtestacionActoAutoridad(datos)
	if err != nil || validarDatosAtestacionActoAutoridad(copia) != nil ||
		validarAtestacionParaSolicitud(solicitud, copia, solicitud.ComprobarEn) != nil {
		return AtestacionActoFuenteAutoridad{}, ErrAtestacionActoAutoridadInvalida
	}
	return AtestacionActoFuenteAutoridad{datos: &copia}, nil
}

func (a AtestacionActoFuenteAutoridad) DatosParaConsumo() (
	DatosAtestacionActoFuenteAutoridad,
	error,
) {
	if a.datos == nil || validarDatosAtestacionActoAutoridad(*a.datos) != nil {
		return DatosAtestacionActoFuenteAutoridad{}, ErrAtestacionActoAutoridadInvalida
	}
	return clonarDatosAtestacionActoAutoridad(*a.datos)
}

func (a AtestacionActoFuenteAutoridad) Evidencia() (
	domain.EvidenciaActoFuenteAutoridad,
	error,
) {
	datos, err := a.DatosParaConsumo()
	return datos.Evidencia, err
}

func (a AtestacionActoFuenteAutoridad) ValidarPara(
	solicitud SolicitudComprobarActoFuenteAutoridad,
	instante time.Time,
) error {
	datos, err := a.DatosParaConsumo()
	if err != nil || validarAtestacionParaSolicitud(solicitud, datos, instante) != nil {
		return ErrAtestacionActoAutoridadInvalida
	}
	return nil
}

func (AtestacionActoFuenteAutoridad) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAtestacionActoAutoridad
}
func (*AtestacionActoFuenteAutoridad) UnmarshalJSON([]byte) error {
	return ErrSerializacionAtestacionActoAutoridad
}
func (AtestacionActoFuenteAutoridad) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAtestacionActoAutoridad
}
func (*AtestacionActoFuenteAutoridad) UnmarshalText([]byte) error {
	return ErrSerializacionAtestacionActoAutoridad
}
func (AtestacionActoFuenteAutoridad) String() string {
	return "[ATESTACION-ACTO-FUENTE-AUTORIDAD-OPACA]"
}
func (a AtestacionActoFuenteAutoridad) GoString() string { return a.String() }
func (a AtestacionActoFuenteAutoridad) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}
func (a AtestacionActoFuenteAutoridad) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

// ComprobadorActosFuentesAutoridad debe verificar documento, representacion,
// huella, firmas, competencia y procedencia de la atestacion. El repositorio
// vuelve a leer el registro y consume el token en la transaccion de gobierno.
type ComprobadorActosFuentesAutoridad interface {
	ComprobarActo(
		context.Context,
		SolicitudComprobarActoFuenteAutoridad,
	) (AtestacionActoFuenteAutoridad, error)
}

func validarDatosAtestacionActoAutoridad(datos DatosAtestacionActoFuenteAutoridad) error {
	if datos.Evidencia.Validar() != nil || datos.RevisionEsperada == 0 ||
		!huellaSHA256PuertoAutoridadValida(datos.HuellaEstadoEsperadoSHA256) ||
		!referenciaPuertoAutoridadValida(datos.VerificadorRef) ||
		!referenciaPuertoAutoridadValida(datos.RegistroAtestacionRef) ||
		!huellaSHA256PuertoAutoridadValida(datos.HuellaRegistroAtestacionSHA256) ||
		!referenciaPuertoAutoridadValida(datos.TokenConsumoRef) ||
		!referenciasPuertoAutoridadDistintas(
			datos.VerificadorRef, datos.RegistroAtestacionRef, datos.TokenConsumoRef,
			datos.Evidencia.AtestacionRef,
		) || !instantePuertoAutoridadCanonico(datos.EmitidaEn) ||
		!instantePuertoAutoridadCanonico(datos.ValidaHasta) ||
		!datos.ValidaHasta.After(datos.EmitidaEn) ||
		datos.ValidaHasta.Sub(datos.EmitidaEn) > VigenciaMaximaAtestacionActoFuenteAutoridad ||
		datos.EmitidaEn.Before(datos.Evidencia.ComprobadaEn) {
		return ErrAtestacionActoAutoridadInvalida
	}
	return nil
}

func validarAtestacionParaSolicitud(
	solicitud SolicitudComprobarActoFuenteAutoridad,
	datos DatosAtestacionActoFuenteAutoridad,
	instante time.Time,
) error {
	compromiso, errCompromiso := solicitud.Solicitud.Compromiso()
	if solicitud.Validar() != nil || validarDatosAtestacionActoAutoridad(datos) != nil ||
		errCompromiso != nil || datos.ValidaHasta.After(compromiso.ExpiraEn) ||
		!instantePuertoAutoridadCanonico(instante) || instante.Before(datos.EmitidaEn) ||
		!instante.Before(datos.ValidaHasta) || !instante.Before(compromiso.ExpiraEn) ||
		datos.EmitidaEn.After(solicitud.ComprobarEn) ||
		datos.RevisionEsperada != solicitud.EstadoEsperado.Revision ||
		datos.HuellaEstadoEsperadoSHA256 != solicitud.EstadoEsperado.HuellaEstadoSHA256 ||
		validarEvidenciaActoParaSolicitud(solicitud.Solicitud, datos.Evidencia) != nil {
		return ErrAtestacionActoAutoridadInvalida
	}
	return nil
}

func validarEvidenciaActoParaSolicitud(
	solicitud domain.SolicitudTransicionFuenteAutoridadV1,
	evidencia domain.EvidenciaActoFuenteAutoridad,
) error {
	compromiso, err := solicitud.Compromiso()
	huellaCompromiso, errHuella := compromiso.HuellaSHA256()
	mensaje, errMensaje := domain.PrepararMensajeAtestacionActoFuenteAutoridadV1(
		solicitud,
		domain.DatosMensajeAtestacionActoFuenteAutoridadV1{
			EvidenciaRef: evidencia.EvidenciaRef, ActoRef: evidencia.ActoRef,
			DocumentoRef: evidencia.DocumentoRef, RepresentacionRef: evidencia.RepresentacionRef,
			HuellaDocumentoSHA256: evidencia.HuellaDocumentoSHA256, OrganoRef: evidencia.OrganoRef,
			FirmasRefs: evidencia.FirmasRefs, ComprobadorRef: evidencia.ComprobadorRef,
			ActoOcurridoEn: evidencia.ActoOcurridoEn, ComprobadaEn: evidencia.ComprobadaEn,
		},
	)
	huellaMensaje, errMensajeHuella := mensaje.HuellaSHA256()
	if err != nil || errHuella != nil || errMensaje != nil || errMensajeHuella != nil ||
		evidencia.Validar() != nil || evidencia.Accion != compromiso.Accion ||
		evidencia.FuenteID != compromiso.Fuente.FuenteID ||
		evidencia.FuenteVersion != compromiso.Fuente.Version ||
		evidencia.HuellaContenidoSHA256 != compromiso.Fuente.HuellaContenidoSHA256 ||
		evidencia.HuellaCompromisoSHA256 != huellaCompromiso ||
		evidencia.HuellaMensajeAtestadoSHA256 != huellaMensaje ||
		evidencia.ComprobadaEn.Before(compromiso.PreparadaEn) ||
		!evidencia.ComprobadaEn.Before(compromiso.ExpiraEn) {
		return ErrAtestacionActoAutoridadInvalida
	}
	return nil
}

func clonarDatosAtestacionActoAutoridad(
	datos DatosAtestacionActoFuenteAutoridad,
) (DatosAtestacionActoFuenteAutoridad, error) {
	evidencia, err := datos.Evidencia.ClonarCanonica()
	if err != nil {
		return DatosAtestacionActoFuenteAutoridad{}, ErrAtestacionActoAutoridadInvalida
	}
	datos.Evidencia = evidencia
	return datos, nil
}

func referenciasPuertoAutoridadDistintas(referencias ...string) bool {
	vistas := make(map[string]struct{}, len(referencias))
	for _, referencia := range referencias {
		if !referenciaPuertoAutoridadValida(referencia) {
			return false
		}
		if _, repetida := vistas[referencia]; repetida {
			return false
		}
		vistas[referencia] = struct{}{}
	}
	return true
}

type bloqueoSerializacionAtestacionActoAutoridad struct{}

func (bloqueoSerializacionAtestacionActoAutoridad) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAtestacionActoAutoridad
}

func (*bloqueoSerializacionAtestacionActoAutoridad) UnmarshalBinary([]byte) error {
	return ErrSerializacionAtestacionActoAutoridad
}

func (bloqueoSerializacionAtestacionActoAutoridad) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionAtestacionActoAutoridad
}

func (*bloqueoSerializacionAtestacionActoAutoridad) GobDecode([]byte) error {
	return ErrSerializacionAtestacionActoAutoridad
}

func (bloqueoSerializacionAtestacionActoAutoridad) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionAtestacionActoAutoridad
}

func (*bloqueoSerializacionAtestacionActoAutoridad) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionAtestacionActoAutoridad
}
