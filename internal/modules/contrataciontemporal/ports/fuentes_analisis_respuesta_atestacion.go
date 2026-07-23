package ports

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const dominioSelloRespuestaFuenteAnalisis = "fuente-analisis-respuesta/v"

type MetadatosAtestacionRespuestaFuenteAnalisis struct {
	AutoridadRef string
	Generacion   uint32
	ReciboRef    string
	EmitidaEn    time.Time
	ValidaHasta  time.Time
}

func (m MetadatosAtestacionRespuestaFuenteAnalisis) Validar() error {
	if !domain.ReferenciaOpacaValida(m.AutoridadRef) ||
		m.Generacion == 0 ||
		!domain.ReferenciaOpacaValida(m.ReciboRef) ||
		!instanteFuenteAnalisisCanonico(m.EmitidaEn) ||
		!instanteFuenteAnalisisCanonico(m.ValidaHasta) ||
		!m.ValidaHasta.After(m.EmitidaEn) ||
		m.ValidaHasta.Sub(m.EmitidaEn) > VigenciaMaximaRespuestaFuenteAnalisis {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

type AtestacionRespuestaFuenteAnalisis struct {
	Metadatos MetadatosAtestacionRespuestaFuenteAnalisis
	SelloHMAC string
}

func (AtestacionRespuestaFuenteAnalisis) String() string {
	return "[ATESTACION-RESPUESTA-FUENTE-ANALISIS-REDACTADA]"
}

func (a AtestacionRespuestaFuenteAnalisis) GoString() string { return a.String() }
func (a AtestacionRespuestaFuenteAnalisis) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, a.String())
}
func (a AtestacionRespuestaFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

func NuevaAtestacionRespuestaFuenteAnalisis(
	metadatos MetadatosAtestacionRespuestaFuenteAnalisis,
	selloHMAC string,
) (AtestacionRespuestaFuenteAnalisis, error) {
	atestacion := AtestacionRespuestaFuenteAnalisis{
		Metadatos: metadatos,
		SelloHMAC: selloHMAC,
	}
	if atestacion.Validar() != nil {
		return AtestacionRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return atestacion, nil
}

func (a AtestacionRespuestaFuenteAnalisis) Validar() error {
	if a.Metadatos.Validar() != nil ||
		!selloRespuestaFuenteAnalisisValido(
			a.SelloHMAC,
			a.Metadatos.Generacion,
		) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func selloRespuestaFuenteAnalisisValido(sello string, generacion uint32) bool {
	dominio := dominioSelloRespuestaFuenteAnalisis + strconv.FormatUint(
		uint64(generacion),
		10,
	)
	return SelloHMACSHA256Valido(sello) &&
		strings.HasPrefix(sello, "hmac-sha256:"+dominio+":")
}

type PreimagenRespuestaFuenteAnalisis struct {
	contenido []byte
}

func (p PreimagenRespuestaFuenteAnalisis) Bytes() ([]byte, error) {
	if len(p.contenido) == 0 || len(p.contenido) > 64*1024 {
		return nil, ErrResultadoFuenteAnalisisNoConfiable
	}
	return append([]byte(nil), p.contenido...), nil
}

func (p PreimagenRespuestaFuenteAnalisis) huellaSHA256() (string, error) {
	contenido, err := p.Bytes()
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:]), nil
}

func (PreimagenRespuestaFuenteAnalisis) String() string {
	return "[PREIMAGEN-RESPUESTA-FUENTE-ANALISIS-REDACTADA]"
}

func (p PreimagenRespuestaFuenteAnalisis) GoString() string { return p.String() }
func (p PreimagenRespuestaFuenteAnalisis) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, p.String())
}
func (p PreimagenRespuestaFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

type SolicitudVerificarRespuestaFuenteAnalisis struct {
	preimagen  PreimagenRespuestaFuenteAnalisis
	atestacion AtestacionRespuestaFuenteAnalisis
}

func nuevaSolicitudVerificarRespuestaFuenteAnalisis(
	preimagen PreimagenRespuestaFuenteAnalisis,
	atestacion AtestacionRespuestaFuenteAnalisis,
) (SolicitudVerificarRespuestaFuenteAnalisis, error) {
	solicitud := SolicitudVerificarRespuestaFuenteAnalisis{
		preimagen: preimagen, atestacion: atestacion,
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificarRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return solicitud, nil
}

func (s SolicitudVerificarRespuestaFuenteAnalisis) Validar() error {
	if _, err := s.preimagen.Bytes(); err != nil ||
		s.atestacion.Validar() != nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func (s SolicitudVerificarRespuestaFuenteAnalisis) Material() (
	PreimagenRespuestaFuenteAnalisis,
	AtestacionRespuestaFuenteAnalisis,
	error,
) {
	if s.Validar() != nil {
		return PreimagenRespuestaFuenteAnalisis{},
			AtestacionRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return PreimagenRespuestaFuenteAnalisis{
		contenido: append([]byte(nil), s.preimagen.contenido...),
	}, s.atestacion, nil
}

type ConfirmacionRespuestaFuenteAnalisis struct {
	datos *DatosConfirmacionRespuestaFuenteAnalisis
}

func (ConfirmacionRespuestaFuenteAnalisis) String() string {
	return "[CONFIRMACION-RESPUESTA-FUENTE-ANALISIS-REDACTADA]"
}

func (c ConfirmacionRespuestaFuenteAnalisis) GoString() string { return c.String() }
func (c ConfirmacionRespuestaFuenteAnalisis) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
func (c ConfirmacionRespuestaFuenteAnalisis) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

type DatosConfirmacionRespuestaFuenteAnalisis struct {
	VerificadorRef       string
	AutoridadRef         string
	Generacion           uint32
	ReciboRef            string
	SelloRespuestaHMAC   string
	HuellaMaterialSHA256 string
	EmitidaEn            time.Time
	ValidaHasta          time.Time
	VerificadaEn         time.Time
}

func NuevaConfirmacionRespuestaFuenteAnalisis(
	solicitud SolicitudVerificarRespuestaFuenteAnalisis,
	verificadorRef string,
	verificadaEn time.Time,
) (ConfirmacionRespuestaFuenteAnalisis, error) {
	preimagen, atestacion, err := solicitud.Material()
	huella, errHuella := preimagen.huellaSHA256()
	datos := DatosConfirmacionRespuestaFuenteAnalisis{
		VerificadorRef:       verificadorRef,
		AutoridadRef:         atestacion.Metadatos.AutoridadRef,
		Generacion:           atestacion.Metadatos.Generacion,
		ReciboRef:            atestacion.Metadatos.ReciboRef,
		SelloRespuestaHMAC:   atestacion.SelloHMAC,
		HuellaMaterialSHA256: huella,
		EmitidaEn:            atestacion.Metadatos.EmitidaEn,
		ValidaHasta:          atestacion.Metadatos.ValidaHasta,
		VerificadaEn:         verificadaEn,
	}
	if err != nil || errHuella != nil ||
		validarDatosConfirmacionRespuesta(datos, solicitud, verificadaEn) != nil {
		return ConfirmacionRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return ConfirmacionRespuestaFuenteAnalisis{datos: &datos}, nil
}

func (c ConfirmacionRespuestaFuenteAnalisis) ValidarPara(
	solicitud SolicitudVerificarRespuestaFuenteAnalisis,
	comprobadaEn time.Time,
) error {
	if c.datos == nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return validarDatosConfirmacionRespuesta(*c.datos, solicitud, comprobadaEn)
}

func validarDatosConfirmacionRespuesta(
	datos DatosConfirmacionRespuestaFuenteAnalisis,
	solicitud SolicitudVerificarRespuestaFuenteAnalisis,
	comprobadaEn time.Time,
) error {
	preimagen, atestacion, err := solicitud.Material()
	huella, errHuella := preimagen.huellaSHA256()
	if err != nil || errHuella != nil ||
		!domain.ReferenciaOpacaValida(datos.VerificadorRef) ||
		datos.AutoridadRef != atestacion.Metadatos.AutoridadRef ||
		datos.VerificadorRef == datos.AutoridadRef ||
		datos.Generacion != atestacion.Metadatos.Generacion ||
		datos.ReciboRef != atestacion.Metadatos.ReciboRef ||
		!hmac.Equal([]byte(datos.SelloRespuestaHMAC), []byte(atestacion.SelloHMAC)) ||
		datos.HuellaMaterialSHA256 != huella ||
		!datos.EmitidaEn.Equal(atestacion.Metadatos.EmitidaEn) ||
		!datos.ValidaHasta.Equal(atestacion.Metadatos.ValidaHasta) ||
		!instanteFuenteAnalisisCanonico(datos.VerificadaEn) ||
		datos.VerificadaEn.Before(datos.EmitidaEn) ||
		!datos.VerificadaEn.Before(datos.ValidaHasta) ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) ||
		comprobadaEn.Before(datos.VerificadaEn) ||
		!comprobadaEn.Before(datos.ValidaHasta) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func (c ConfirmacionRespuestaFuenteAnalisis) Datos() (
	DatosConfirmacionRespuestaFuenteAnalisis,
	error,
) {
	if c.datos == nil {
		return DatosConfirmacionRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return *c.datos, nil
}

type VerificadorRespuestaFuenteAnalisis interface {
	PresentadorAutoridadFuenteAnalisis
	VerificarRespuestaFuenteAnalisis(
		context.Context,
		SolicitudVerificarRespuestaFuenteAnalisis,
	) (ConfirmacionRespuestaFuenteAnalisis, error)
}
