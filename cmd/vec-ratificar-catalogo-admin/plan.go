package main

import (
	"encoding/json"
	"strings"
	"time"
)

var recursos = map[string]string{
	"administracion.perfiles.aprobar":             "propuesta_perfil",
	"administracion.perfiles.historial.consultar": "historial_perfil",
	"administracion.perfiles.otorgar":             "perfil",
	"administracion.perfiles.proponer":            "perfil",
	"administracion.perfiles.rechazar":            "propuesta_perfil",
	"administracion.perfiles.recibo.consultar":    "recibo_perfil",
	"administracion.perfiles.revocar":             "perfil",
}

func validarPlan(p documentoPlan) bool {
	if p.Esquema != "vec.admin.ratificacion-catalogo.v1" || p.Version != "0600" ||
		p.RolRef != "rol:administracion_perfiles:v7" || !referenciaOperacion(p.OperacionRef) ||
		!hashValido(p.RolSHA) || !hashValido(p.ControlSHA) || !hashValido(p.CatalogoSHA) ||
		!hashValido(p.PreimagenSHA) || len(p.Descriptores) != len(recursos) {
		return false
	}
	revision := p.ControlRevision.String()
	if len(revision) == 0 || len(revision) > 18 || revision[0] < '1' || revision[0] > '9' {
		return false
	}
	for _, c := range revision[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	preparado, ok := instanteUTC(p.PreparadoEn)
	if !ok {
		return false
	}
	caduca, ok := instanteUTC(p.CaducaEn)
	if !ok || !caduca.After(preparado) || !caduca.After(time.Now().UTC()) || preparado.After(time.Now().UTC()) {
		return false
	}
	previa := ""
	for _, d := range p.Descriptores {
		recurso, ok := recursos[d.Concesion.Accion]
		if !ok || d.AccionRef != "accion:"+d.Concesion.Accion || d.AccionRef <= previa ||
			d.Concesion.ModuloID != "administracion" || d.Concesion.TipoRecurso != recurso ||
			d.Concesion.GarantiaMinima != "alto" || len(d.Concesion.Finalidades) != 1 ||
			d.Concesion.Finalidades[0] != "gestion_perfiles" || len(d.DimensionesAmbito) != 1 ||
			d.DimensionesAmbito[0] != "organizacion_ref" {
			return false
		}
		previa = d.AccionRef
	}
	return true
}

func referenciaOperacion(v string) bool {
	if !strings.HasPrefix(v, "rca_") || len(v) < 26 || len(v) > 128 {
		return false
	}
	for _, c := range v[4:] {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func referenciaAprobacion(v string) bool {
	if len(v) == 0 || len(v) > 128 {
		return false
	}
	for i, c := range v {
		alfanum := c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
		if !alfanum && (i == 0 || c != '.' && c != '_' && c != ':' && c != '-') {
			return false
		}
	}
	return true
}

func instanteUTC(v string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil || t.Year() < 1 || t.Year() > 9999 {
		return time.Time{}, false
	}
	_, offset := t.Zone()
	return t, offset == 0
}

func hashValido(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func numeroPositivo(v json.Number) bool {
	n, err := v.Int64()
	return err == nil && n > 0
}
