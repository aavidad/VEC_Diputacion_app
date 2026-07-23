package ports

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// DatosConfirmacionPublicacionCobertura transporta la publicación completa,
// no una afirmación booleana. El núcleo restaura su canon y demuestra la
// pertenencia exacta de la vía y comprobación solicitadas.
type DatosConfirmacionPublicacionCobertura struct {
	PublicadorRef string
	Publicacion   domain.PublicacionCatalogoViasCobertura
	VerificadaEn  time.Time
}

type ConfirmacionPublicacionCobertura struct {
	datos *DatosConfirmacionPublicacionCobertura
}

func NuevaConfirmacionPublicacionCobertura(
	publicadorRef string,
	publicacion domain.PublicacionCatalogoViasCobertura,
	verificadaEn time.Time,
) (ConfirmacionPublicacionCobertura, error) {
	catalogo, err := domain.RestaurarCatalogoViasCobertura(publicacion)
	datos := DatosConfirmacionPublicacionCobertura{
		PublicadorRef: publicadorRef,
		Publicacion:   publicacion,
		VerificadaEn:  verificadaEn,
	}
	if err != nil || !domain.ReferenciaOpacaValida(publicadorRef) ||
		!instanteFuenteAnalisisCanonico(verificadaEn) ||
		catalogo.Validar() != nil {
		return ConfirmacionPublicacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	datos.Publicacion = catalogo.Publicacion()
	return ConfirmacionPublicacionCobertura{datos: &datos}, nil
}

func (c ConfirmacionPublicacionCobertura) ValidarPara(
	solicitud SolicitudConsultarCobertura,
	comprobadaEn time.Time,
) error {
	if c.datos == nil || solicitud.Validar() != nil ||
		!instanteFuenteAnalisisCanonico(comprobadaEn) ||
		!domain.ReferenciaOpacaValida(c.datos.PublicadorRef) ||
		!instanteFuenteAnalisisCanonico(c.datos.VerificadaEn) ||
		c.datos.VerificadaEn.Before(solicitud.SolicitadaEn) ||
		c.datos.VerificadaEn.After(comprobadaEn) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	catalogo, err := domain.RestaurarCatalogoViasCobertura(c.datos.Publicacion)
	if err != nil ||
		!catalogo.Identidad().CoincideExactamente(solicitud.Catalogo) ||
		catalogo.PublicadoEn().After(solicitud.SolicitadaEn) ||
		!catalogo.VigenteEn(solicitud.SolicitadaEn) ||
		!catalogo.VigenteEn(comprobadaEn) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	via, existe := catalogo.Via(solicitud.ViaClave)
	if !existe || !contieneComprobacionCobertura(
		via,
		solicitud.Comprobacion,
	) {
		return ErrResultadoFuenteCoberturaNoConfiable
	}
	return nil
}

func contieneComprobacionCobertura(
	via domain.DefinicionViaCobertura,
	esperada domain.ComprobacionExigibleCobertura,
) bool {
	for _, comprobacion := range via.Comprobaciones {
		if comprobacion == esperada {
			return true
		}
	}
	return false
}

func (c ConfirmacionPublicacionCobertura) Datos() (
	DatosConfirmacionPublicacionCobertura,
	error,
) {
	if c.datos == nil {
		return DatosConfirmacionPublicacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	datos := *c.datos
	catalogo, err := domain.RestaurarCatalogoViasCobertura(datos.Publicacion)
	if err != nil {
		return DatosConfirmacionPublicacionCobertura{},
			ErrResultadoFuenteCoberturaNoConfiable
	}
	datos.Publicacion = catalogo.Publicacion()
	return datos, nil
}

func (ConfirmacionPublicacionCobertura) String() string {
	return "[CONFIRMACION-PUBLICACION-COBERTURA-REDACTADA]"
}

func (c ConfirmacionPublicacionCobertura) GoString() string { return c.String() }
func (c ConfirmacionPublicacionCobertura) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, c.String())
}
func (c ConfirmacionPublicacionCobertura) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

// PublicadorCatalogoCobertura es una autoridad separada. Debe recuperar una
// publicación vigente, inmutable y gobernada; no recibe raíces de confianza.
type PublicadorCatalogoCobertura interface {
	PresentadorAutoridadFuenteAnalisis
	ConsultarPublicacionCobertura(
		context.Context,
		SolicitudConsultarCobertura,
	) (ConfirmacionPublicacionCobertura, error)
}
