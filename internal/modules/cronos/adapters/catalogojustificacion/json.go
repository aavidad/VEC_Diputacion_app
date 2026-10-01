package catalogojustificacion

import (
	"errors"
	"strings"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/domain"
)

var ErrEntrada = errors.New("cronos: catalogo de justificacion invalido")

type Escenario struct {
	Referencia      string                       `json:"referencia"`
	Solicitud       domain.SolicitudJustificable `json:"solicitud"`
	Vinculo         domain.VinculoJustificacion  `json:"vinculo"`
	VinculoRevision *domain.VinculoJustificacion `json:"vinculo_revision,omitempty"`
	VersionEsperada int64                        `json:"version_esperada"`
	Decision        domain.EstadoJustificacion   `json:"decision"`
	MotivoRef       string                       `json:"motivo_ref"`
}

type Escenarios struct {
	VersionEsquema int         `json:"version_esquema"`
	Demostracion   bool        `json:"demostracion"`
	Escenarios     []Escenario `json:"escenarios"`
}

type Politica struct {
	VersionEsquema     int      `json:"version_esquema"`
	Demostracion       bool     `json:"demostracion"`
	Referencia         string   `json:"referencia"`
	Version            uint64   `json:"version"`
	CatalogoVersionRef string   `json:"catalogo_version_ref"`
	PermisoRef         string   `json:"permiso_ref"`
	TipoDocumentalRef  string   `json:"tipo_documental_ref"`
	CustodioID         string   `json:"custodio_id"`
	MotivosRef         []string `json:"motivos_ref"`
}

type Textos struct {
	Aviso    string `json:"aviso"`
	Anexo    string `json:"anexo"`
	Revision string `json:"revision"`
	Limite   string `json:"limite"`
}

func CargarEscenarios(b []byte, sha string) (Escenarios, error) {
	var e Escenarios
	if catalogoefectos.DecodificarEstricto(b, sha, &e) != nil || e.VersionEsquema != 1 || !e.Demostracion || len(e.Escenarios) == 0 || len(e.Escenarios) > 8 {
		return Escenarios{}, ErrEntrada
	}
	vistos := make(map[string]bool, len(e.Escenarios))
	for _, x := range e.Escenarios {
		if x.Referencia == "" || len(x.Referencia) > 80 || vistos[x.Referencia] || x.VersionEsperada != 1 || x.MotivoRef == "" {
			return Escenarios{}, ErrEntrada
		}
		vistos[x.Referencia] = true
	}
	return e, nil
}

func CargarPolitica(b []byte, sha string) (domain.PoliticaJustificacion, error) {
	var w Politica
	if catalogoefectos.DecodificarEstricto(b, sha, &w) != nil || w.VersionEsquema != 1 || !w.Demostracion {
		return domain.PoliticaJustificacion{}, ErrEntrada
	}
	p := domain.PoliticaJustificacion{
		Referencia: w.Referencia, Version: w.Version, SHA256: sha,
		CatalogoVersionRef: w.CatalogoVersionRef, PermisoRef: w.PermisoRef,
		TipoDocumentalRef: w.TipoDocumentalRef, CustodioID: w.CustodioID,
		MotivosRef: w.MotivosRef,
	}
	if p.Validar() != nil {
		return domain.PoliticaJustificacion{}, ErrEntrada
	}
	return p, nil
}

func CargarTextos(b []byte, sha string) (Textos, error) {
	var t Textos
	if catalogoefectos.DecodificarEstricto(b, sha, &t) != nil {
		return Textos{}, ErrEntrada
	}
	for _, s := range []string{t.Aviso, t.Anexo, t.Revision, t.Limite} {
		if s == "" || len(s) > 512 || !utf8.ValidString(s) || strings.TrimSpace(s) != s {
			return Textos{}, ErrEntrada
		}
	}
	return t, nil
}
