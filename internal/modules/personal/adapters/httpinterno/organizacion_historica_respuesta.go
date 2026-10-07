package httpinterno

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

var referenciaOH = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
var huellaOH = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validarResultadoOrganizacionHistorica(s domain.SelectorOrganizacionHistorica, r ports.ResultadoConsultaOrganizacionHistorica) error {
	p, e := r.Pagina, r.Evidencia
	a := p.Selector
	if a.Validar() != nil || a.OrganismoRef != s.OrganismoRef || a.UnidadClave != s.UnidadClave || a.VigenteEn != s.VigenteEn || !a.ConocidoEn.Equal(s.ConocidoEn) || a.VersionRPTRef != s.VersionRPTRef || a.VersionPlantillaRef != s.VersionPlantillaRef || a.Limite != s.Limite || a.Cursor != s.Cursor || p.VersionRPTRef != "" && !referenciaOH.MatchString(p.VersionRPTRef) || p.VersionPlantillaRef != "" && !referenciaOH.MatchString(p.VersionPlantillaRef) || s.VersionRPTRef != "" && p.VersionRPTRef != s.VersionRPTRef || s.VersionPlantillaRef != "" && p.VersionPlantillaRef != s.VersionPlantillaRef || !referenciaOH.MatchString(e.ReciboRef) || !referenciaOH.MatchString(e.DecisionRef) || !referenciaOH.MatchString(e.AuditoriaRef) || e.EfectoRef != s.OrganismoRef || !huellaOH.MatchString(e.ConsumoHuellaSHA256) || !instanteOH(e.ConsultadaEn) {
		return domain.ErrOrganizacionHistoricaNoDisponible
	}
	// Reuse the existing selector validation for the next cursor.
	siguiente := s
	siguiente.Cursor = p.CursorSiguiente
	if siguiente.Validar() != nil || p.CursorSiguiente != "" && p.CursorSiguiente == s.Cursor {
		return domain.ErrOrganizacionHistoricaNoDisponible
	}
	ids := map[string]bool{}
	agregar := func(t domain.TrazaOrganizacionHistorica) bool {
		if t.ValidarEn(s) != nil || ids[t.ID] {
			return false
		}
		ids[t.ID] = true
		return true
	}
	for _, v := range p.Unidades {
		if !agregar(v.Traza) {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	for _, v := range p.PuestosTipo {
		if !agregar(v.Traza) {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	for _, v := range p.Dotaciones {
		if !agregar(v.Traza) {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	for _, v := range p.Plazas {
		if !agregar(v.Traza) {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	for _, v := range p.PuestosIndividuales {
		if !agregar(v.Traza) {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	for _, v := range p.Vinculos {
		if !agregar(v.Traza) {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	if len(ids) > s.Limite || len(ids) == 0 && p.CursorSiguiente != "" || p.VersionRPTRef == "" && (len(p.PuestosTipo)+len(p.Dotaciones)+len(p.PuestosIndividuales) != 0 || p.Cobertura.PuestosTipo != "sin_datos" || p.Cobertura.Dotaciones != "sin_datos" || p.Cobertura.PuestosIndividuales != "sin_datos") || p.VersionPlantillaRef == "" && (len(p.Plazas) != 0 || p.Cobertura.Plazas != "sin_datos") {
		return domain.ErrOrganizacionHistoricaNoDisponible
	}
	i := domain.InstantaneaComparacionOrganizacion{Selector: s, Cobertura: domain.CoberturaComparacionOrganizacion(p.Cobertura), Unidades: p.Unidades, PuestosTipo: p.PuestosTipo, Dotaciones: p.Dotaciones, Plazas: p.Plazas, PuestosIndividuales: p.PuestosIndividuales, Vinculos: p.Vinculos}
	i.Selector.Cursor = ""
	i.Selector.VersionRPTRef = p.VersionRPTRef
	i.Selector.VersionPlantillaRef = p.VersionPlantillaRef
	if _, err := domain.CompararOrganizacionHistorica(i, i); err != nil {
		return domain.ErrOrganizacionHistoricaNoDisponible
	}
	return nil
}

func instanteOH(t time.Time) bool {
	_, offset := t.Zone()
	return !t.IsZero() && offset == 0 && t.Nanosecond()%1000 == 0
}

func formaRespuestaOrganizacionHistorica(b []byte) error {
	var top, data, pagina map[string]json.RawMessage
	if json.Unmarshal(b, &top) != nil || len(top) != 1 || json.Unmarshal(top["data"], &data) != nil || len(data) != 2 || data["evidencia"] == nil || json.Unmarshal(data["pagina"], &pagina) != nil {
		return domain.ErrOrganizacionHistoricaNoDisponible
	}
	for _, k := range []string{"selector", "version_rpt_ref", "version_plantilla_ref", "cobertura", "unidades", "puestos_tipo", "dotaciones", "plazas", "puestos_individuales", "vinculos"} {
		if _, ok := pagina[k]; !ok {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	var selector, evidencia map[string]json.RawMessage
	if json.Unmarshal(pagina["selector"], &selector) != nil || len(selector) != 8 || json.Unmarshal(data["evidencia"], &evidencia) != nil || len(evidencia) != 6 {
		return domain.ErrOrganizacionHistoricaNoDisponible
	}
	for _, k := range []string{"organismo_ref", "unidad_clave", "vigente_en", "conocido_en", "version_rpt_ref", "version_plantilla_ref", "cursor"} {
		v := bytes.TrimSpace(selector[k])
		if len(v) == 0 || v[0] != '"' {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	for _, k := range []string{"version_rpt_ref", "version_plantilla_ref"} {
		v := bytes.TrimSpace(pagina[k])
		if len(v) == 0 || v[0] != '"' {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
	}
	return nil
}

// The duplicate-key/depth constraint matches the existing CLI parser. Unknown
// members are subsequently rejected by the typed decoder.
func jsonUnicoOrganizacionHistorica(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var visitar func(int) error
	visitar = func(n int) error {
		if n > 64 {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
		t, err := d.Token()
		if err != nil {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
		v, ok := t.(json.Delim)
		if !ok {
			return nil
		}
		if v == '{' {
			vistos := map[string]bool{}
			for d.More() {
				t, err := d.Token()
				if err != nil {
					return domain.ErrOrganizacionHistoricaNoDisponible
				}
				k, ok := t.(string)
				if !ok || k != strings.ToLower(k) || vistos[k] {
					return domain.ErrOrganizacionHistoricaNoDisponible
				}
				vistos[k] = true
				if err := visitar(n + 1); err != nil {
					return domain.ErrOrganizacionHistoricaNoDisponible
				}
			}
		} else if v == '[' {
			for d.More() {
				if err := visitar(n + 1); err != nil {
					return domain.ErrOrganizacionHistoricaNoDisponible
				}
			}
		}
		if _, err := d.Token(); err != nil {
			return domain.ErrOrganizacionHistoricaNoDisponible
		}
		return nil
	}
	if visitar(0) != nil {
		return domain.ErrOrganizacionHistoricaNoDisponible
	}
	if _, err := d.Token(); err != io.EOF {
		return domain.ErrOrganizacionHistoricaNoDisponible
	}
	return nil
}
