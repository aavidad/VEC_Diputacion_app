package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var ErrPlanUnidadInicialAdminInvalido = errors.New("plan_unidad_inicial_admin_invalido")
var patronOperacionUnidadInicialAdmin = regexp.MustCompile(`^pui_[A-Za-z0-9_-]{22,124}$`)
var patronActoUnidadInicialAdmin = regexp.MustCompile(`^acto_tecnico:[a-z0-9_:-]{1,147}$`)

func (p PlanUnidadInicialAdminV1) Validar() error {
	if p.Version != 1 || !patronOperacionUnidadInicialAdmin.MatchString(p.OperacionRef) ||
		!entornoUnidadInicialAdmin(p.Entorno, p.AlcanceFuente) || !instanteUnidadInicialAdmin(p.PreparadoEn) || !instanteUnidadInicialAdmin(p.CaducaEn) || !p.CaducaEn.After(p.PreparadoEn) ||
		p.Unidad.Validar() != nil || !patronActoUnidadInicialAdmin.MatchString(p.ActoTecnicoRef) || !patronAmbitoOrganizacionHistorica.MatchString(p.Fuente.Referencia) || p.Fuente.Version != 1 || !patronHuellaOrganizacionHistorica.MatchString(p.Fuente.HuellaSHA256) {
		return ErrPlanUnidadInicialAdminInvalido
	}
	desde, _ := fechaUnidadInicialAdmin(p.Unidad.VigenteDesde)
	hasta, _ := fechaUnidadInicialAdmin(p.Unidad.VigenteHasta)
	if desde.After(p.PreparadoEn) || hasta.Before(p.CaducaEn) {
		return ErrPlanUnidadInicialAdminInvalido
	}
	return nil
}
func (u UnidadInicialAdmin) Validar() error {
	if !patronUUIDRegistroB2.MatchString(u.NodoRef) || !patronAmbitoOrganizacionHistorica.MatchString(u.OrganizacionRef) || !patronAmbitoOrganizacionHistorica.MatchString(u.UnidadRef) ||
		u.CatalogoRef != idEstructuraOrganizativa || u.CatalogoVersion != 1 || u.CatalogoRevision != 1 || u.RevisionEsperada != 0 ||
		!patronReferenciaFuente.MatchString(u.CatalogoEntradaClave) || !denominacionUnidadInicialAdmin(u.Denominacion) {
		return ErrPlanUnidadInicialAdminInvalido
	}
	switch u.Clase {
	case "centro", "delegacion", "puesto_responsabilidad":
	default:
		return ErrPlanUnidadInicialAdminInvalido
	}
	desde, e := fechaUnidadInicialAdmin(u.VigenteDesde)
	hasta, err := fechaUnidadInicialAdmin(u.VigenteHasta)
	if e != nil || err != nil || !hasta.After(desde) {
		return ErrPlanUnidadInicialAdminInvalido
	}
	return nil
}
func (f FuenteUnidadInicialAdminV1) Validar() error {
	if f.Version != 1 || !patronAmbitoOrganizacionHistorica.MatchString(f.Referencia) || !entornoUnidadInicialAdmin(f.Entorno, f.AlcanceFuente) ||
		f.Unidad.Validar() != nil || !patronActoUnidadInicialAdmin.MatchString(f.ActoTecnicoRef) {
		return ErrPlanUnidadInicialAdminInvalido
	}
	return nil
}
func (p PlanUnidadInicialAdminV1) ValidarEn(ahora time.Time) error {
	if p.Validar() != nil || ahora.IsZero() || ahora.Before(p.PreparadoEn) || !ahora.Before(p.CaducaEn) {
		return ErrPlanUnidadInicialAdminInvalido
	}
	return nil
}

// ValidarConFuente sólo coteja los bytes y metadatos preparados. No acredita
// la procedencia institucional ni sustituye la configuración externa del DBA.
func (p PlanUnidadInicialAdminV1) ValidarConFuente(f FuenteUnidadInicialAdminV1) error {
	if p.Validar() != nil || f.Validar() != nil || p.Fuente.Referencia != f.Referencia || p.Entorno != f.Entorno || p.AlcanceFuente != f.AlcanceFuente || p.Unidad != f.Unidad || p.ActoTecnicoRef != f.ActoTecnicoRef {
		return ErrPlanUnidadInicialAdminInvalido
	}
	_, sha, e := f.CanonicoYHuella()
	if e != nil || sha != p.Fuente.HuellaSHA256 {
		return ErrPlanUnidadInicialAdminInvalido
	}
	return nil
}
func (p PlanUnidadInicialAdminV1) CanonicoYHuella() ([]byte, string, error) {
	if p.Validar() != nil {
		return nil, "", ErrPlanUnidadInicialAdminInvalido
	}
	return canonUnidadInicialAdmin(p)
}
func (f FuenteUnidadInicialAdminV1) CanonicoYHuella() ([]byte, string, error) {
	if f.Validar() != nil {
		return nil, "", ErrPlanUnidadInicialAdminInvalido
	}
	return canonUnidadInicialAdmin(f)
}
func canonUnidadInicialAdmin(v any) ([]byte, string, error) {
	b, e := json.Marshal(v)
	if e != nil || len(b) > 64<<10 {
		return nil, "", ErrPlanUnidadInicialAdminInvalido
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}
func instanteUnidadInicialAdmin(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond() == 0 && t.Year() >= 1 && t.Year() <= 9999
}
func fechaUnidadInicialAdmin(f FechaCivil) (time.Time, error) {
	t, e := time.Parse("2006-01-02", string(f))
	if e != nil || t.Year() < 1 || t.Year() > 9999 || t.Format("2006-01-02") != string(f) {
		return time.Time{}, ErrPlanUnidadInicialAdminInvalido
	}
	return t, nil
}
func entornoUnidadInicialAdmin(entorno, alcance string) bool {
	return entorno == "desarrollo" && alcance == "sintetico_declarado"
}
func denominacionUnidadInicialAdmin(v string) bool {
	if v == "" || !utf8.ValidString(v) || utf8.RuneCountInString(v) > 300 || strings.TrimSpace(v) != v {
		return false
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
