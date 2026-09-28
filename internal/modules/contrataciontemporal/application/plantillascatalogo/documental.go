package plantillascatalogo

import (
	"regexp"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var huellaConsultaDocumental = regexp.MustCompile(`^[0-9a-f]{64}$`)
var claveTipoDocumental = regexp.MustCompile(`^[a-z][a-z0-9._-]{1,79}$`)

// SolicitudDocumental nace del detalle ya consultado con V3. El actor no
// procede de HTTP: la fuente nominal lo resuelve otra vez en la autorización
// independiente de lectura del catálogo publicado.
type SolicitudDocumental struct {
	Operacion            string `json:"operacion"`
	OrganizacionRef      string `json:"organizacion_ref"`
	ClaseAmbito          string `json:"clase_ambito"`
	AmbitoRef            string `json:"ambito_ref"`
	ExpedienteRef        string `json:"expediente_ref"`
	VersionObservada     uint64 `json:"version_observada"`
	ConsultaHuellaSHA256 string `json:"consulta_huella_sha256"`
	Tipo                 string `json:"tipo,omitempty"`
	Formato              string `json:"formato,omitempty"`
}

func (s SolicitudDocumental) Validar() error {
	if !ctdomain.ReferenciaOpacaValida(s.ExpedienteRef) || !ctdomain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		s.ClaseAmbito != string(ports.AmbitoOrganizacionRRHH) || s.AmbitoRef != s.OrganizacionRef ||
		s.VersionObservada < 1 || s.VersionObservada > 9_007_199_254_740_991 || !huellaConsultaDocumental.MatchString(s.ConsultaHuellaSHA256) {
		return ErrEntradaInvalida
	}
	switch s.Operacion {
	case "listar":
		if s.Tipo != "" || s.Formato != "" {
			return ErrEntradaInvalida
		}
	case "descargar":
		if !claveTipoDocumental.MatchString(s.Tipo) || (s.Formato != "pdf" && s.Formato != "docx") {
			return ErrEntradaInvalida
		}
	default:
		return ErrEntradaInvalida
	}
	return nil
}
