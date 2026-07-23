package ports

import (
	"context"
	"crypto/hmac"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	dominioSelloRespuestaCobertura   = "fuente-cobertura-respuesta/v"
	VigenciaMaximaRespuestaCobertura = 5 * time.Second
)

type MetadatosAtestacionRespuestaCobertura struct {
	AutoridadRef string
	Generacion   uint32
	ReciboRef    string
	EmitidaEn    time.Time
	ValidaHasta  time.Time
}

func (m MetadatosAtestacionRespuestaCobertura) Validar() error {
	if !domain.ReferenciaOpacaValida(m.AutoridadRef) ||
		m.Generacion == 0 ||
		!domain.ReferenciaOpacaValida(m.ReciboRef) ||
		!instanteFuenteAnalisisCanonico(m.EmitidaEn) ||
		!instanteFuenteAnalisisCanonico(m.ValidaHasta) ||
		!m.ValidaHasta.After(m.EmitidaEn) ||
		m.ValidaHasta.Sub(m.EmitidaEn) > VigenciaMaximaRespuestaCobertura {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

type AtestacionRespuestaCobertura struct {
	Metadatos MetadatosAtestacionRespuestaCobertura
	SelloHMAC string
}

func NuevaAtestacionRespuestaCobertura(
	metadatos MetadatosAtestacionRespuestaCobertura,
	selloHMAC string,
) (AtestacionRespuestaCobertura, error) {
	atestacion := AtestacionRespuestaCobertura{
		Metadatos: metadatos,
		SelloHMAC: selloHMAC,
	}
	if atestacion.Validar() != nil {
		return AtestacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return atestacion, nil
}

func (a AtestacionRespuestaCobertura) Validar() error {
	if a.Metadatos.Validar() != nil ||
		!selloRespuestaCoberturaValido(
			a.SelloHMAC,
			a.Metadatos.Generacion,
		) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (AtestacionRespuestaCobertura) String() string {
	return "[ATESTACION-RESPUESTA-COBERTURA-REDACTADA]"
}

func (a AtestacionRespuestaCobertura) GoString() string { return a.String() }
func (a AtestacionRespuestaCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, a.String())
}
func (a AtestacionRespuestaCobertura) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

func selloRespuestaCoberturaValido(sello string, generacion uint32) bool {
	dominio := dominioSelloRespuestaCobertura + strconv.FormatUint(
		uint64(generacion),
		10,
	)
	return SelloHMACSHA256Valido(sello) &&
		strings.HasPrefix(sello, "hmac-sha256:"+dominio+":")
}

type SolicitudVerificarRespuestaCobertura struct {
	preimagen  PreimagenRespuestaCobertura
	atestacion AtestacionRespuestaCobertura
}

func nuevaSolicitudVerificarRespuestaCobertura(
	preimagen PreimagenRespuestaCobertura,
	atestacion AtestacionRespuestaCobertura,
) (SolicitudVerificarRespuestaCobertura, error) {
	solicitud := SolicitudVerificarRespuestaCobertura{
		preimagen:  preimagen,
		atestacion: atestacion,
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificarRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return solicitud, nil
}

func (s SolicitudVerificarRespuestaCobertura) Validar() error {
	if _, err := s.preimagen.Bytes(); err != nil ||
		s.atestacion.Validar() != nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (s SolicitudVerificarRespuestaCobertura) Material() (
	PreimagenRespuestaCobertura,
	AtestacionRespuestaCobertura,
	error,
) {
	if s.Validar() != nil {
		return PreimagenRespuestaCobertura{},
			AtestacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return PreimagenRespuestaCobertura{
		contenido: append([]byte(nil), s.preimagen.contenido...),
	}, s.atestacion, nil
}

type DatosConfirmacionRespuestaCobertura struct {
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

type ConfirmacionRespuestaCobertura struct {
	datos *DatosConfirmacionRespuestaCobertura
}

func NuevaConfirmacionRespuestaCobertura(
	solicitud SolicitudVerificarRespuestaCobertura,
	verificadorRef string,
	verificadaEn time.Time,
) (ConfirmacionRespuestaCobertura, error) {
	preimagen, atestacion, err := solicitud.Material()
	huella, errHuella := preimagen.huellaSHA256()
	datos := DatosConfirmacionRespuestaCobertura{
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
		validarDatosConfirmacionCobertura(
			datos,
			solicitud,
			verificadaEn,
		) != nil {
		return ConfirmacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return ConfirmacionRespuestaCobertura{datos: &datos}, nil
}

func (c ConfirmacionRespuestaCobertura) ValidarPara(
	solicitud SolicitudVerificarRespuestaCobertura,
	comprobadaEn time.Time,
) error {
	if c.datos == nil {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return validarDatosConfirmacionCobertura(
		*c.datos,
		solicitud,
		comprobadaEn,
	)
}

func validarDatosConfirmacionCobertura(
	datos DatosConfirmacionRespuestaCobertura,
	solicitud SolicitudVerificarRespuestaCobertura,
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
		!hmac.Equal(
			[]byte(datos.SelloRespuestaHMAC),
			[]byte(atestacion.SelloHMAC),
		) ||
		datos.HuellaMaterialSHA256 != huella ||
		!datos.EmitidaEn.Equal(atestacion.Metadatos.EmitidaEn) ||
		!datos.ValidaHasta.Equal(atestacion.Metadatos.ValidaHasta) ||
		!instanteFuenteAnalisisCanonico(datos.VerificadaEn) ||
		datos.VerificadaEn.Before(datos.EmitidaEn) ||
		!datos.VerificadaEn.Before(datos.ValidaHasta) ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) ||
		comprobadaEn.Before(datos.VerificadaEn) ||
		!comprobadaEn.Before(datos.ValidaHasta) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func (c ConfirmacionRespuestaCobertura) Datos() (
	DatosConfirmacionRespuestaCobertura,
	error,
) {
	if c.datos == nil {
		return DatosConfirmacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return *c.datos, nil
}

func (ConfirmacionRespuestaCobertura) String() string {
	return "[CONFIRMACION-RESPUESTA-COBERTURA-REDACTADA]"
}

func (c ConfirmacionRespuestaCobertura) GoString() string { return c.String() }
func (c ConfirmacionRespuestaCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
func (c ConfirmacionRespuestaCobertura) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

type VerificadorRespuestaCobertura interface {
	PresentadorAutoridadFuenteAnalisis
	VerificarRespuestaCobertura(
		context.Context,
		SolicitudVerificarRespuestaCobertura,
	) (ConfirmacionRespuestaCobertura, error)
}
