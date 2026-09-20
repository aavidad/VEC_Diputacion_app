package domain

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrConsultaEstructuraPublicaInvalida = errors.New("personal: consulta de estructura publica invalida")
	ErrEstructuraPublicaNoDisponible     = errors.New("personal: estructura publica no disponible")
	patronHuellaEstructuraPublica        = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// UnidadEstructuraPublica es la proyección mínima apta para la presentación.
// No representa ocupación, permisos, códigos fuente ni edición local.
type UnidadEstructuraPublica struct {
	Clave            string `json:"clave"`
	Etiqueta         string `json:"etiqueta"`
	Tipo             string `json:"tipo"`
	AdscripcionClave string `json:"adscripcion_clave,omitempty"`
}

type FuenteEstructuraPublica struct {
	Revision      string `json:"revision"`
	ActualizadaEn string `json:"actualizada_en"`
	Demostracion  bool   `json:"demostracion"`
	Aviso         string `json:"aviso"`
	HuellaSHA256  string `json:"huella_sha256"`
}

type EstructuraOrganizativaPublica struct {
	Esquema          string                    `json:"esquema"`
	CatalogoID       string                    `json:"catalogo_id"`
	CatalogoVersion  int                       `json:"catalogo_version"`
	CatalogoRevision int                       `json:"catalogo_revision"`
	FuenteRef        string                    `json:"fuente_ref"`
	Fuente           FuenteEstructuraPublica   `json:"fuente"`
	Unidades         []UnidadEstructuraPublica `json:"unidades"`
}

func (e EstructuraOrganizativaPublica) Validar() error {
	if e.Esquema != "vec.personal.estructura-organizativa-publica.v1" || e.CatalogoID != "estructura-organizativa-dipgra" || e.CatalogoVersion != 1 || e.CatalogoRevision < 1 || !textoEstructuraPublica(e.FuenteRef, 2048) || !textoEstructuraPublica(e.Fuente.Revision, 256) || !textoEstructuraPublica(e.Fuente.ActualizadaEn, 64) || !e.Fuente.Demostracion || !textoEstructuraPublica(e.Fuente.Aviso, 4096) || !patronHuellaEstructuraPublica.MatchString(e.Fuente.HuellaSHA256) || len(e.Unidades) != 66 {
		return ErrEstructuraPublicaNoDisponible
	}
	vistas := make(map[string]struct{}, len(e.Unidades))
	conteos := map[string]int{}
	for _, unidad := range e.Unidades {
		if !textoEstructuraPublica(unidad.Clave, 128) || !textoEstructuraPublica(unidad.Etiqueta, 512) || (unidad.Tipo != "delegacion" && unidad.Tipo != "centro" && unidad.Tipo != "puesto_responsabilidad") || (unidad.AdscripcionClave != "" && !textoEstructuraPublica(unidad.AdscripcionClave, 128)) {
			return ErrEstructuraPublicaNoDisponible
		}
		if _, ok := vistas[unidad.Clave]; ok {
			return ErrEstructuraPublicaNoDisponible
		}
		vistas[unidad.Clave] = struct{}{}
		conteos[unidad.Tipo]++
	}
	if conteos["delegacion"] != 14 || conteos["centro"] != 41 || conteos["puesto_responsabilidad"] != 11 {
		return ErrEstructuraPublicaNoDisponible
	}
	return nil
}

func (e EstructuraOrganizativaPublica) Clonar() EstructuraOrganizativaPublica {
	salida := e
	salida.Unidades = append([]UnidadEstructuraPublica(nil), e.Unidades...)
	return salida
}

func textoEstructuraPublica(valor string, maximo int) bool {
	return valor != "" && valor == strings.TrimSpace(valor) && len([]rune(valor)) <= maximo && !strings.ContainsAny(valor, "\x00\r\n\t")
}
