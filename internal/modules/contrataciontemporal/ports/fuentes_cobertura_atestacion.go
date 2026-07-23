package ports

import (
	"context"
	"crypto/ed25519"
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
	dominioSelloRespuestaCobertura        = "fuente-cobertura-respuesta/v"
	dominioConfirmacionRespuestaCobertura = "VEC-CT-CONFIRMACION-COBERTURA-V1"
	VigenciaMaximaRespuestaCobertura      = 5 * time.Second
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
	huellaPeticionSHA256 string
	preimagen            PreimagenRespuestaCobertura
	atestacion           AtestacionRespuestaCobertura
}

func nuevaSolicitudVerificarRespuestaCobertura(
	huellaPeticionSHA256 string,
	preimagen PreimagenRespuestaCobertura,
	atestacion AtestacionRespuestaCobertura,
) (SolicitudVerificarRespuestaCobertura, error) {
	solicitud := SolicitudVerificarRespuestaCobertura{
		huellaPeticionSHA256: huellaPeticionSHA256,
		preimagen:            preimagen,
		atestacion:           atestacion,
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificarRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return solicitud, nil
}

func (s SolicitudVerificarRespuestaCobertura) Validar() error {
	if !huellaSHA256FuenteAnalisisValida(s.huellaPeticionSHA256) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
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

// PreimagenConfirmacionRespuestaCobertura es el material que debe firmar la
// misma clave Ed25519 cuya posesión acreditó el verificador ante el TCB.
type PreimagenConfirmacionRespuestaCobertura struct {
	contenido []byte
}

func (p PreimagenConfirmacionRespuestaCobertura) Bytes() ([]byte, error) {
	if len(p.contenido) == 0 || len(p.contenido) > 64*1024 {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	return append([]byte(nil), p.contenido...), nil
}

func (PreimagenConfirmacionRespuestaCobertura) String() string {
	return "[PREIMAGEN-CONFIRMACION-COBERTURA-REDACTADA]"
}

func (p PreimagenConfirmacionRespuestaCobertura) GoString() string {
	return p.String()
}
func (p PreimagenConfirmacionRespuestaCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, p.String())
}
func (p PreimagenConfirmacionRespuestaCobertura) LogValue() slog.Value {
	return slog.StringValue(p.String())
}

type DatosConfirmacionRespuestaCobertura struct {
	VerificadorRef       string
	HuellaPeticionSHA256 string
	AutoridadRef         string
	Generacion           uint32
	ReciboRef            string
	SelloRespuestaHMAC   string
	HuellaMaterialSHA256 string
	EmitidaEn            time.Time
	ValidaHasta          time.Time
	VerificadaEn         time.Time
	FirmaEd25519         []byte
}

type ConfirmacionRespuestaCobertura struct {
	datos *DatosConfirmacionRespuestaCobertura
}

func NuevaPreimagenConfirmacionRespuestaCobertura(
	solicitud SolicitudVerificarRespuestaCobertura,
	verificadorRef string,
	verificadaEn time.Time,
) (PreimagenConfirmacionRespuestaCobertura, error) {
	datos, err := datosConfirmacionCobertura(
		solicitud,
		verificadorRef,
		verificadaEn,
	)
	if err != nil {
		return PreimagenConfirmacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	contenido, err := canonConfirmacionRespuestaCobertura(datos)
	if err != nil {
		return PreimagenConfirmacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return PreimagenConfirmacionRespuestaCobertura{contenido: contenido}, nil
}

func NuevaConfirmacionRespuestaCobertura(
	solicitud SolicitudVerificarRespuestaCobertura,
	verificadorRef string,
	verificadaEn time.Time,
	firmaEd25519 []byte,
) (ConfirmacionRespuestaCobertura, error) {
	datos, err := datosConfirmacionCobertura(
		solicitud,
		verificadorRef,
		verificadaEn,
	)
	if err != nil || len(firmaEd25519) != ed25519.SignatureSize {
		return ConfirmacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	datos.FirmaEd25519 = append([]byte(nil), firmaEd25519...)
	return ConfirmacionRespuestaCobertura{datos: &datos}, nil
}

func datosConfirmacionCobertura(
	solicitud SolicitudVerificarRespuestaCobertura,
	verificadorRef string,
	verificadaEn time.Time,
) (DatosConfirmacionRespuestaCobertura, error) {
	preimagen, atestacion, err := solicitud.Material()
	huella, errHuella := preimagen.huellaSHA256()
	datos := DatosConfirmacionRespuestaCobertura{
		VerificadorRef:       verificadorRef,
		HuellaPeticionSHA256: solicitud.huellaPeticionSHA256,
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
		!domain.ReferenciaOpacaValida(datos.VerificadorRef) ||
		datos.VerificadorRef == datos.AutoridadRef ||
		!huellaSHA256FuenteAnalisisValida(datos.HuellaPeticionSHA256) ||
		!instanteFuenteAnalisisCanonico(datos.VerificadaEn) ||
		datos.VerificadaEn.Before(datos.EmitidaEn) ||
		!datos.VerificadaEn.Before(datos.ValidaHasta) {
		return DatosConfirmacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	return datos, nil
}

func canonConfirmacionRespuestaCobertura(
	datos DatosConfirmacionRespuestaCobertura,
) ([]byte, error) {
	if datos.FirmaEd25519 != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	escritor := nuevoEscritorCanonFuenteAnalisis()
	escritor.texto(dominioConfirmacionRespuestaCobertura)
	escritor.texto(datos.VerificadorRef)
	escritor.texto(datos.HuellaPeticionSHA256)
	escritor.texto(datos.AutoridadRef)
	escritor.entero64(uint64(datos.Generacion))
	escritor.texto(datos.ReciboRef)
	escritor.texto(datos.SelloRespuestaHMAC)
	escritor.texto(datos.HuellaMaterialSHA256)
	escritor.instante(datos.EmitidaEn)
	escritor.instante(datos.ValidaHasta)
	escritor.instante(datos.VerificadaEn)
	contenido, err := escritor.resultado()
	if err != nil {
		return nil, ErrResultadoFuenteCoberturaNoConfiable
	}
	return contenido, nil
}

func (c ConfirmacionRespuestaCobertura) ValidarPara(
	solicitud SolicitudVerificarRespuestaCobertura,
	comprobadaEn time.Time,
	claveVerificadorEd25519 ed25519.PublicKey,
) error {
	if c.datos == nil ||
		len(c.datos.FirmaEd25519) != ed25519.SignatureSize ||
		len(claveVerificadorEd25519) != ed25519.PublicKeySize ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) ||
		comprobadaEn.Before(c.datos.VerificadaEn) ||
		!comprobadaEn.Before(c.datos.ValidaHasta) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	esperados, err := datosConfirmacionCobertura(
		solicitud,
		c.datos.VerificadorRef,
		c.datos.VerificadaEn,
	)
	if err != nil || !datosConfirmacionCoberturaCoinciden(
		*c.datos,
		esperados,
	) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	canon, err := canonConfirmacionRespuestaCobertura(esperados)
	if err != nil || !ed25519.Verify(
		claveVerificadorEd25519,
		canon,
		c.datos.FirmaEd25519,
	) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func datosConfirmacionCoberturaCoinciden(
	recibidos DatosConfirmacionRespuestaCobertura,
	esperados DatosConfirmacionRespuestaCobertura,
) bool {
	return recibidos.VerificadorRef == esperados.VerificadorRef &&
		recibidos.HuellaPeticionSHA256 == esperados.HuellaPeticionSHA256 &&
		recibidos.AutoridadRef == esperados.AutoridadRef &&
		recibidos.Generacion == esperados.Generacion &&
		recibidos.ReciboRef == esperados.ReciboRef &&
		hmac.Equal(
			[]byte(recibidos.SelloRespuestaHMAC),
			[]byte(esperados.SelloRespuestaHMAC),
		) &&
		recibidos.HuellaMaterialSHA256 == esperados.HuellaMaterialSHA256 &&
		recibidos.EmitidaEn.Equal(esperados.EmitidaEn) &&
		recibidos.ValidaHasta.Equal(esperados.ValidaHasta) &&
		recibidos.VerificadaEn.Equal(esperados.VerificadaEn)
}

func (c ConfirmacionRespuestaCobertura) Datos() (
	DatosConfirmacionRespuestaCobertura,
	error,
) {
	if c.datos == nil {
		return DatosConfirmacionRespuestaCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	datos := *c.datos
	datos.FirmaEd25519 = append([]byte(nil), c.datos.FirmaEd25519...)
	return datos, nil
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
