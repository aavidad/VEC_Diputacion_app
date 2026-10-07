package main

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var errEnvoltura = errors.New("envoltura_invalida")

type envoltura struct {
	Estado           string               `json:"estado"`
	Codigo           *string              `json:"codigo"`
	Recibo           *reciboMantenimiento `json:"recibo"`
	Replay           bool                 `json:"replay"`
	AuditoriaIntento auditoriaIntento     `json:"auditoria_intento"`
}
type auditoriaIntento struct {
	AuditoriaRef   string `json:"auditoria_ref"`
	Secuencia      uint64 `json:"secuencia"`
	HuellaSHA256   string `json:"huella_sha256"`
	CorrelacionRef string `json:"correlacion_ref"`
	RegistradaEn   string `json:"registrada_en"`
}
type reciboMantenimiento struct {
	Esquema            string                 `json:"esquema"`
	OperacionRef       string                 `json:"operacion_ref"`
	PlanSHA256         string                 `json:"plan_sha256"`
	RolOrigenRef       string                 `json:"rol_origen_ref"`
	RolDestinoRef      string                 `json:"rol_destino_ref"`
	RolDestinoSHA      string                 `json:"rol_destino_sha256"`
	Asignaciones       []referenciaAsignacion `json:"asignaciones"`
	AuditoriaRef       string                 `json:"auditoria_ref"`
	AuditoriaSecuencia uint64                 `json:"auditoria_secuencia"`
	AuditoriaHuella    string                 `json:"auditoria_huella_sha256"`
	ConfirmadoEn       string                 `json:"confirmado_en"`
}
type referenciaAsignacion struct {
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

func validarEnvoltura(b []byte, p documentoPlan, sha string) (envoltura, error) {
	var e envoltura
	variante, ok := varianteMantenimiento(p.Version)
	if !ok {
		return e, errEnvoltura
	}
	if len(b) < 1 || len(b) > limiteDocumento || decodificarEstricto(b, &e) != nil {
		return e, errEnvoltura
	}
	a := e.AuditoriaIntento
	if !refHex(a.AuditoriaRef, "aud_v3_mfi_") || a.Secuencia == 0 || a.Secuencia > 1<<53-1 || !hashValido(a.HuellaSHA256) || !refHex(a.CorrelacionRef, "correlacion_") || !fechaValida(a.RegistradaEn) {
		return e, errEnvoltura
	}
	switch e.Estado {
	case "permitido":
		r := e.Recibo
		if e.Codigo != nil || r == nil || r.Esquema != variante.Esquema || r.OperacionRef != p.OperacionRef || r.PlanSHA256 != sha || r.RolOrigenRef != variante.OrigenRef || r.RolDestinoRef != variante.DestinoRef || !hashValido(r.RolDestinoSHA) || len(r.Asignaciones) != 2 || !refHex(r.AuditoriaRef, "aud_v3_mf_") || r.AuditoriaSecuencia == 0 || !hashValido(r.AuditoriaHuella) || !fechaValida(r.ConfirmadoEn) {
			return e, errEnvoltura
		}
		var target struct {
			Version int `json:"version"`
		}
		if json.Unmarshal(p.RolDestino, &target) != nil || target.Version != variante.VersionRolDestino {
			return e, errEnvoltura
		}
		expected := map[string]bool{}
		for _, x := range p.Asignaciones {
			if !strings.HasSuffix(x.AsignacionRef, variante.OrigenSufijo) {
				return e, errEnvoltura
			}
			expected[strings.TrimSuffix(x.AsignacionRef, variante.OrigenSufijo)+variante.DestinoSufijo] = true
		}
		for _, x := range r.Asignaciones {
			if !expected[x.Ref] || !hashValido(x.SHA) {
				return e, errEnvoltura
			}
			delete(expected, x.Ref)
		}
		if len(expected) != 0 {
			return e, errEnvoltura
		}
	case "denegado":
		if e.Codigo == nil || *e.Codigo != "mantenimiento_rechazado" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	case "error":
		if e.Codigo == nil || *e.Codigo != "mantenimiento_no_disponible" || e.Recibo != nil || e.Replay {
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
