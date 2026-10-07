package main

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

var errEnvoltura = errors.New("envoltura_invalida")

// envoltura es la respuesta de publicar_cargos_firma_admin_v1 (AUT53).
type envoltura struct {
	Estado           string           `json:"estado"`
	Codigo           *string          `json:"codigo"`
	Recibo           *reciboCargos    `json:"recibo"`
	Replay           bool             `json:"replay"`
	AuditoriaIntento auditoriaIntento `json:"auditoria_intento"`
}
type auditoriaIntento struct {
	AuditoriaRef   string `json:"auditoria_ref"`
	Secuencia      uint64 `json:"secuencia"`
	HuellaSHA256   string `json:"huella_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
	RegistradaEn   string `json:"registrada_en"`
}
type reciboCargos struct {
	Esquema            string        `json:"esquema"`
	OperacionRef       string        `json:"operacion_ref"`
	PlanSHA256         string        `json:"plan_sha256"`
	AprobacionRef      string        `json:"aprobacion_ref"`
	AprobacionSHA256   string        `json:"aprobacion_sha256"`
	Cargos             []cargoRecibo `json:"cargos"`
	PerfilesSHA256     string        `json:"perfiles_sha256"`
	AuditoriaRef       string        `json:"auditoria_ref"`
	AuditoriaSecuencia uint64        `json:"auditoria_secuencia"`
	AuditoriaHuella    string        `json:"auditoria_huella_sha256"`
	ConfirmadoEn       string        `json:"confirmado_en"`
}
type cargoRecibo struct {
	RolID            string `json:"rol_id"`
	VersionRolRef    string `json:"version_rol_ref"`
	VersionRolSHA256 string `json:"version_rol_sha256"`
	ControlRevision  uint64 `json:"control_revision"`
	ControlSHA256    string `json:"control_sha256"`
	ReglaAsignacion  string `json:"regla_asignacion"`
}

// validarEnvoltura sólo acepta un recibo que publique exactamente los cargos
// del plan, con la huella que se aprobó, o una negativa sin recibo.
func validarEnvoltura(b []byte, p documentoPlan, sha string) (envoltura, error) {
	var e envoltura
	if len(b) < 1 || len(b) > limiteDocumento || decodificarEstricto(b, &e) != nil {
		return e, errEnvoltura
	}
	a := e.AuditoriaIntento
	if !refHex(a.AuditoriaRef, "aud_v3_pai_") || a.Secuencia == 0 || a.Secuencia > 1<<53-1 || !hashValido(a.HuellaSHA256) || !refHex(a.CorrelacionRef, "correlacion_") || !fechaValida(a.RegistradaEn) {
		return e, errEnvoltura
	}
	switch e.Estado {
	case "permitido":
		r := e.Recibo
		if e.Codigo != nil || r == nil || r.Esquema != "vec.admin.cargos-firma.recibo.v1" || r.OperacionRef != p.OperacionRef || r.PlanSHA256 != sha ||
			r.AprobacionRef == "" || !hashValido(r.AprobacionSHA256) || !hashValido(r.PerfilesSHA256) || !refHex(r.AuditoriaRef, "aud_v3_pa_") ||
			r.AuditoriaSecuencia == 0 || !hashValido(r.AuditoriaHuella) || !fechaValida(r.ConfirmadoEn) || len(r.Cargos) != len(p.Cargos) {
			return e, errEnvoltura
		}
		esperados := map[string]cargoPlan{}
		for _, c := range p.Cargos {
			esperados[c.RolID] = c
		}
		for _, x := range r.Cargos {
			c, ok := esperados[x.RolID]
			if !ok || x.VersionRolRef != "rol:"+c.RolID+":v"+strconv.FormatUint(c.Version, 10) || x.VersionRolSHA256 != c.VersionRolSHA256 ||
				x.ControlRevision != 1 || !hashValido(x.ControlSHA256) || x.ReglaAsignacion != c.ReglaAsignacion {
				return e, errEnvoltura
			}
			delete(esperados, x.RolID)
		}
		if len(esperados) != 0 {
			return e, errEnvoltura
		}
	case "denegado":
		if e.Codigo == nil || *e.Codigo != "cargos_firma_rechazado" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	case "error":
		if e.Codigo == nil || *e.Codigo != "cargos_firma_no_disponible" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	default:
		return e, errEnvoltura
	}
	return e, nil
}
func fechaValida(v string) bool {
	t, e := time.Parse(time.RFC3339Nano, v)
	if e != nil || t.IsZero() || t.Year() < 1 || t.Year() > 9999 {
		return false
	}
	_, offset := t.Zone()
	return offset == 0
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
func refHex(v, p string) bool {
	if !strings.HasPrefix(v, p) || len(v) != len(p)+32 {
		return false
	}
	return hashValido(v[len(p):] + strings.Repeat("0", 32))
}
