package administracionperfiles

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
	"vec-diputacion-granada/internal/vec/domain"
)

var errConsultaPersonasInvalida = errors.New("administracion de perfiles: consulta de personas invalida")

// consultaPersonas valida exclusivamente la forma HTTP. El catálogo y la
// decisión de acceso corresponden a FuenteLecturas, nunca a estos filtros.
func consultaPersonas(raw string) (ConsultaPersonas, error) {
	if len(raw) > 2048 {
		return ConsultaPersonas{}, errConsultaPersonasInvalida
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return ConsultaPersonas{}, err
	}
	for clave, valores := range q {
		switch clave {
		case "q", "cursor", "perfil_ref", "unidad_ref", "estado":
		default:
			return ConsultaPersonas{}, errConsultaPersonasInvalida
		}
		if len(valores) != 1 || valores[0] == "" {
			return ConsultaPersonas{}, errConsultaPersonasInvalida
		}
	}
	x := ConsultaPersonas{Texto: q.Get("q"), Cursor: q.Get("cursor"), PerfilRef: q.Get("perfil_ref"), UnidadRef: q.Get("unidad_ref"), Estado: q.Get("estado")}
	if x.Texto != "" && (len(x.Texto) < 2 || !textoConsulta(x.Texto, 132) || strings.TrimSpace(x.Texto) != x.Texto) {
		return ConsultaPersonas{}, errConsultaPersonasInvalida
	}
	if !textoConsulta(x.Cursor, 256) || x.PerfilRef != "" && !domain.RolVersionAdministracionPerfilesValido(x.PerfilRef) || x.UnidadRef != "" && !referenciaUnidad(x.UnidadRef) || x.Estado != "" && x.Estado != "vigente" && x.Estado != "caducado" {
		return ConsultaPersonas{}, errConsultaPersonasInvalida
	}
	return x, nil
}

func textoConsulta(s string, max int) bool {
	if !utf8.ValidString(s) || len(s) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func referenciaUnidad(s string) bool {
	if !strings.HasPrefix(s, "unidad:") || len(s) <= len("unidad:") || len(s) > 256 {
		return false
	}
	for _, r := range s[len("unidad:"):] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == ':' || r == '.') {
			return false
		}
	}
	return true
}
