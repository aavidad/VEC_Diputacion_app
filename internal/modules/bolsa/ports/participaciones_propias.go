package ports

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	EsquemaParticipacionesPropiasV1          = "vec.bolsa.mi-bolsa.v1"
	AccionConsultarParticipacionesPropias    = "bolsa.candidato.participaciones.consultar"
	FinalidadConsultarParticipacionesPropias = "consulta_posicion_propia_bolsa"
	ModuloParticipacionesPropiasBolsa        = "bolsa"
	TipoRecursoParticipacionesPropias        = "participaciones_propias_candidato"
	AudienciaParticipacionesPropias          = "vec.bolsa.mi-bolsa.v1"
)

var (
	ErrConsultaParticipacionesPropiasInvalida  = errors.New("bolsa: consulta de participaciones propias invalida")
	ErrResultadoParticipacionesPropiasInvalido = errors.New("bolsa: resultado de participaciones propias invalido")
)

// ConsultaParticipacionesPropias es la capacidad que llega al adaptador
// duradero. La referencia procede exclusivamente del contexto resuelto en el
// servidor; no se serializa ni puede construirse desde una petición HTTP.
type ConsultaParticipacionesPropias struct {
	candidatoRef string
	consultadaEn time.Time
}

func NuevaConsultaParticipacionesPropias(candidatoRef string, consultadaEn time.Time) (ConsultaParticipacionesPropias, error) {
	c := ConsultaParticipacionesPropias{candidatoRef: candidatoRef, consultadaEn: consultadaEn.UTC()}
	if c.Validar() != nil {
		return ConsultaParticipacionesPropias{}, ErrConsultaParticipacionesPropiasInvalida
	}
	return c, nil
}

func (c ConsultaParticipacionesPropias) CandidatoRef() (string, error) {
	if c.Validar() != nil {
		return "", ErrConsultaParticipacionesPropiasInvalida
	}
	return c.candidatoRef, nil
}
func (c ConsultaParticipacionesPropias) ConsultadaEn() (time.Time, error) {
	if c.Validar() != nil {
		return time.Time{}, ErrConsultaParticipacionesPropiasInvalida
	}
	return c.consultadaEn, nil
}
func (c ConsultaParticipacionesPropias) Validar() error {
	if !referenciaCandidataValida(c.candidatoRef) || !instanteParticipacionesPropiasCanonico(c.consultadaEn) {
		return ErrConsultaParticipacionesPropiasInvalida
	}
	return nil
}
func (ConsultaParticipacionesPropias) String() string {
	return "[CONSULTA-PARTICIPACIONES-PROPIAS-BOLSA-OPACA]"
}

// RecursoAutorizableParticipacionesPropias liga una única candidatura y la
// finalidad exacta a la decisión. No contiene estado de disponibilidad ni
// datos enviados por el navegador.
func RecursoAutorizableParticipacionesPropias(c ConsultaParticipacionesPropias) (dominiovec.RecursoAutorizable, error) {
	ref, err := c.CandidatoRef()
	if err != nil {
		return dominiovec.RecursoAutorizable{}, err
	}
	r := dominiovec.RecursoAutorizable{Referencia: "candidato:" + ref, ModuloID: ModuloParticipacionesPropiasBolsa,
		Tipo: TipoRecursoParticipacionesPropias, Ambitos: map[string]string{"candidato_ref": ref},
		Atributos: map[string]string{"finalidad": FinalidadConsultarParticipacionesPropias}}
	if r.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrConsultaParticipacionesPropiasInvalida
	}
	return r, nil
}

// AutorizadorParticipacionesPropias devuelve el conjunto AD-3 completo. La
// persistencia vuelve a verificarlo y lo consume junto a la auditoría de la
// lectura; la aplicación no transforma ni reduce sus piezas.
type AutorizadorParticipacionesPropias interface {
	AutorizarOperacion(context.Context, string, dominiovec.RecursoAutorizable) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type ConsultaParticipacionesPropiasPersistente interface {
	ConsultarParticipacionesPropias(context.Context, ConsultaParticipacionesPropias, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ResultadoParticipacionesPropias, error)
}

type ParticipacionPropia struct {
	BolsaRef             string     `json:"bolsa_ref"`
	CategoriaRef         string     `json:"categoria_ref"`
	VersionBolsa         uint64     `json:"version_bolsa"`
	Orden                uint64     `json:"orden"`
	TotalParticipaciones uint64     `json:"total_participaciones"`
	EstadoBolsa          string     `json:"estado_bolsa"`
	VigenteDesde         time.Time  `json:"vigente_desde"`
	VigenteHasta         *time.Time `json:"vigente_hasta,omitempty"`
}

type ResultadoParticipacionesPropias struct {
	Esquema         string                `json:"esquema"`
	ConsultadaEn    time.Time             `json:"consultada_en"`
	Participaciones []ParticipacionPropia `json:"participaciones"`
}

func (r ResultadoParticipacionesPropias) ClonarValidadoPara(c ConsultaParticipacionesPropias) (ResultadoParticipacionesPropias, error) {
	if c.Validar() != nil || r.Esquema != EsquemaParticipacionesPropiasV1 || !instanteParticipacionesPropiasCanonico(r.ConsultadaEn) || r.ConsultadaEn.Before(c.consultadaEn) {
		return ResultadoParticipacionesPropias{}, ErrResultadoParticipacionesPropiasInvalido
	}
	copia := r
	copia.Participaciones = make([]ParticipacionPropia, len(r.Participaciones))
	copy(copia.Participaciones, r.Participaciones)
	for i := range copia.Participaciones {
		p := &copia.Participaciones[i]
		p.VigenteDesde = p.VigenteDesde.UTC()
		if p.VigenteHasta != nil {
			hasta := p.VigenteHasta.UTC()
			p.VigenteHasta = &hasta
		}
		if !participacionPropiaValida(*p) || (i > 0 && participacionPropiaMenor(*p, copia.Participaciones[i-1])) {
			return ResultadoParticipacionesPropias{}, ErrResultadoParticipacionesPropiasInvalido
		}
	}
	return copia, nil
}

func participacionPropiaValida(p ParticipacionPropia) bool {
	return referenciaBolsaValida(p.BolsaRef) && referenciaBolsaValida(p.CategoriaRef) && p.VersionBolsa > 0 && p.Orden > 0 &&
		p.TotalParticipaciones > 0 && p.Orden <= p.TotalParticipaciones && claveBolsaValida(p.EstadoBolsa) &&
		instanteParticipacionesPropiasCanonico(p.VigenteDesde) && (p.VigenteHasta == nil || (instanteParticipacionesPropiasCanonico(*p.VigenteHasta) && p.VigenteHasta.After(p.VigenteDesde)))
}
func participacionPropiaMenor(a, b ParticipacionPropia) bool {
	if a.BolsaRef != b.BolsaRef {
		return a.BolsaRef < b.BolsaRef
	}
	return a.VersionBolsa < b.VersionBolsa
}
func referenciaCandidataValida(v string) bool { return referenciaBolsaConPrefijoValida(v, "can_") }
func referenciaBolsaValida(v string) bool     { return referenciaBolsaConPrefijoValida(v, "") }
func referenciaBolsaConPrefijoValida(v, prefijo string) bool {
	if len(v) < 12 || (prefijo != "" && !strings.HasPrefix(v, prefijo)) {
		return false
	}
	for _, r := range v {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == ':' || r == '.') {
			return false
		}
	}
	return true
}
func claveBolsaValida(v string) bool {
	if len(v) == 0 || len(v) > 128 {
		return false
	}
	for _, r := range v {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}
func instanteParticipacionesPropiasCanonico(v time.Time) bool {
	return !v.IsZero() && v.Location() == time.UTC && v.Nanosecond()%1000 == 0
}
