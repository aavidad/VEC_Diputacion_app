package ports

import (
	"context"
	"errors"
	"regexp"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var patronCodigoPuestoRPTAlta = regexp.MustCompile(`^[A-Z0-9][A-Z0-9-]{0,63}$`)
var patronHuellaPuestoRPTAlta = regexp.MustCompile(`^[a-f0-9]{64}$`)

var ErrFuenteNecesidadesAltaNoDisponible = errors.New("contratacion temporal: fuente de necesidades de alta no disponible")

const EsquemaAltaNecesidadV1 = "vec.ct.alta_necesidad.v1"

func materialNecesidadAltaValido(esquema string, necesidad *domain.DatosNecesidadAlta, causa domain.ClaveCatalogo) bool {
	if esquema == "" {
		return necesidad == nil
	}
	return esquema == EsquemaAltaNecesidadV1 && necesidad != nil &&
		necesidad.CausaClave == causa && necesidad.ValidarInstantanea() == nil
}

// ConsultaInstantaneaNecesidadConfirmada usa exclusivamente el ámbito sellado
// y la identidad resuelta por servidor. Sólo devuelve el catálogo público
// congelado en una alta ya confirmada; no autoriza otro acceso al expediente.
type ConsultaInstantaneaNecesidadConfirmada struct {
	AmbitosHMAC     ColeccionSellosHMAC
	OrganizacionRef string
	ActorRef        string
	PerfilRef       string
}

type EstadoInstantaneaNecesidadAlta string

const (
	InstantaneaNecesidadAusente      EstadoInstantaneaNecesidadAlta = "ausente"
	InstantaneaNecesidadLegadoV2     EstadoInstantaneaNecesidadAlta = "legado_v2"
	InstantaneaNecesidadConfirmadaV3 EstadoInstantaneaNecesidadAlta = "confirmada_v3"
)

type ResultadoInstantaneaNecesidadAlta struct {
	Estado      EstadoInstantaneaNecesidadAlta
	Instantanea []byte
}

func (r ResultadoInstantaneaNecesidadAlta) Validar() error {
	switch r.Estado {
	case InstantaneaNecesidadAusente, InstantaneaNecesidadLegadoV2:
		if len(r.Instantanea) != 0 {
			return ErrFuenteNecesidadesAltaNoDisponible
		}
	case InstantaneaNecesidadConfirmadaV3:
		if len(r.Instantanea) == 0 || len(r.Instantanea) > domain.MaximoInstantaneaCatalogoNecesidadesAltaBytes {
			return ErrFuenteNecesidadesAltaNoDisponible
		}
	default:
		return ErrFuenteNecesidadesAltaNoDisponible
	}
	return nil
}

func (c ConsultaInstantaneaNecesidadConfirmada) Validar() error {
	if c.AmbitosHMAC.ValidarDominio("vec.contratacion-temporal.ambito-idempotencia") != nil ||
		!domain.ReferenciaOpacaValida(c.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(c.ActorRef) ||
		!domain.ReferenciaOpacaValida(c.PerfilRef) {
		return ErrFuenteNecesidadesAltaNoDisponible
	}
	return nil
}

// FuenteNecesidadesAlta resuelve una versión publicada exacta o recupera la
// instantánea original por la misma intención de idempotencia confirmada.
type FuenteNecesidadesAlta interface {
	ResolverCatalogoNecesidadesAlta(context.Context, string, uint64, string) (domain.CatalogoNecesidadesAlta, error)
	RecuperarInstantaneaNecesidadConfirmada(context.Context, ConsultaInstantaneaNecesidadConfirmada) (ResultadoInstantaneaNecesidadAlta, error)
}

// SolicitudVerificarPuestoRPTAlta sólo comprueba que un código figura en una
// publicación exacta. La RPT pública no acredita ocupación ni vacancia.
type SolicitudVerificarPuestoRPTAlta struct {
	CatalogoRef          string
	CatalogoVersion      uint64
	CatalogoHuellaSHA256 string
	PuestoCodigo         string
}

func (s SolicitudVerificarPuestoRPTAlta) Validar() error {
	if !domain.ReferenciaOpacaValida(s.CatalogoRef) || s.CatalogoVersion == 0 ||
		s.CatalogoVersion > 1<<53-1 ||
		!patronHuellaPuestoRPTAlta.MatchString(s.CatalogoHuellaSHA256) ||
		!patronCodigoPuestoRPTAlta.MatchString(s.PuestoCodigo) {
		return ErrFuenteNecesidadesAltaNoDisponible
	}
	return nil
}

type PuestoRPTAltaVerificado struct {
	CatalogoRef          string
	CatalogoVersion      uint64
	CatalogoHuellaSHA256 string
	PuestoCodigo         string
	ExisteEnPublicacion  bool
}

func (p PuestoRPTAltaVerificado) ValidarPara(s SolicitudVerificarPuestoRPTAlta) error {
	if s.Validar() != nil || p.CatalogoRef != s.CatalogoRef ||
		p.CatalogoVersion != s.CatalogoVersion || p.CatalogoHuellaSHA256 != s.CatalogoHuellaSHA256 ||
		p.PuestoCodigo != s.PuestoCodigo || !p.ExisteEnPublicacion {
		return ErrFuenteNecesidadesAltaNoDisponible
	}
	return nil
}

type VerificadorPuestoRPTAlta interface {
	VerificarPuestoRPTAlta(context.Context, SolicitudVerificarPuestoRPTAlta) (PuestoRPTAltaVerificado, error)
}

func SolicitudPuestoRPTDesdeCampos(campos map[string]string) (SolicitudVerificarPuestoRPTAlta, error) {
	version, err := strconv.ParseUint(campos["rpt_catalogo_version"], 10, 64)
	if err != nil {
		return SolicitudVerificarPuestoRPTAlta{}, ErrFuenteNecesidadesAltaNoDisponible
	}
	s := SolicitudVerificarPuestoRPTAlta{
		CatalogoRef: campos["rpt_catalogo_ref"], CatalogoVersion: version,
		CatalogoHuellaSHA256: campos["rpt_catalogo_huella_sha256"], PuestoCodigo: campos["puesto_codigo"],
	}
	return s, s.Validar()
}
