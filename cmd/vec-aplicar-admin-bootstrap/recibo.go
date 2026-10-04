package main

import (
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type envoltura struct {
	Estado           string           `json:"estado"`
	Codigo           *string          `json:"codigo"`
	Recibo           *reciboBootstrap `json:"recibo"`
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
type reciboBootstrap struct {
	ActoRef           string             `json:"acto_ref"`
	ReciboRef         string             `json:"recibo_ref"`
	HuellaPlanSHA256  string             `json:"huella_plan_sha256"`
	PrimeraPersonaRef string             `json:"primera_persona_ref"`
	SegundaPersonaRef string             `json:"segunda_persona_ref"`
	AuditoriaRef      string             `json:"auditoria_ref"`
	Perfiles          [3]perfilBootstrap `json:"perfiles"`
	ConfirmadoEn      string             `json:"confirmado_en"`
}
type perfilBootstrap struct {
	PersonaRef    string `json:"persona_ref"`
	CuentaRef     string `json:"cuenta_ref"`
	PerfilRef     string `json:"perfil_ref"`
	VinculoRef    string `json:"vinculo_ref"`
	AsignacionRef string `json:"asignacion_ref"`
	RolVersionRef string `json:"rol_version_ref"`
}

var errEnvoltura = errors.New("envoltura_invalida")

func validarEnvoltura(b []byte, p domain.PlanBootstrapAdministracionV3, sha string) (envoltura, error) {
	var e envoltura
	if len(b) == 0 || len(b) > limiteDocumento || decodificarEstricto(b, &e) != nil {
		return e, errEnvoltura
	}
	a := e.AuditoriaIntento
	if !refHex(a.AuditoriaRef, "aud_v3_bi_") || a.Secuencia == 0 || a.Secuencia > 1<<53-1 || !hashValido(a.HuellaSHA256) || !refHex(a.CorrelacionRef, "correlacion_") || !fechaRecibo(a.RegistradaEn) {
		return e, errEnvoltura
	}
	switch e.Estado {
	case "permitido":
		if e.Codigo != nil || e.Recibo == nil || !reciboValido(e.Recibo, p, sha) {
			return e, errEnvoltura
		}
	case "denegado":
		if e.Codigo == nil || *e.Codigo != "bootstrap_rechazado" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	case "error":
		if e.Codigo == nil || *e.Codigo != "bootstrap_no_disponible" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	default:
		return e, errEnvoltura
	}
	return e, nil
}
func reciboValido(r *reciboBootstrap, p domain.PlanBootstrapAdministracionV3, sha string) bool {
	if !refHex(r.ActoRef, "acto_admin:") || !refHex(r.ReciboRef, "recibo_bootstrap:") || r.HuellaPlanSHA256 != sha || r.PrimeraPersonaRef != p.Personas[0].PersonaRef || r.SegundaPersonaRef != p.Personas[1].PersonaRef || !refHex(r.AuditoriaRef, "aud_v3_p_") || !fechaRecibo(r.ConfirmadoEn) {
		return false
	}
	esperado := map[string]perfilBootstrap{}
	for _, persona := range p.Personas {
		esperado[persona.PerfilRef] = perfilBootstrap{PersonaRef: persona.PersonaRef, CuentaRef: persona.CuentaRef, PerfilRef: persona.PerfilRef, VinculoRef: persona.VinculoRef, RolVersionRef: p.Rol.VersionRef}
		for _, sistema := range persona.Sistemas {
			esperado[sistema.PerfilRef] = perfilBootstrap{PersonaRef: persona.PersonaRef, CuentaRef: persona.CuentaRef, PerfilRef: sistema.PerfilRef, VinculoRef: sistema.VinculoRef, RolVersionRef: sistema.Rol.VersionRef}
		}
	}
	if len(esperado) != 3 {
		return false
	}
	vistos := map[string]bool{}
	for _, perfil := range r.Perfiles {
		x, ok := esperado[perfil.PerfilRef]
		if !ok || vistos[perfil.PerfilRef] || perfil.PersonaRef != x.PersonaRef || perfil.CuentaRef != x.CuentaRef || perfil.VinculoRef != x.VinculoRef || perfil.RolVersionRef != x.RolVersionRef || !asignacionValida(perfil.AsignacionRef) {
			return false
		}
		vistos[perfil.PerfilRef] = true
	}
	return true
}
func asignacionValida(v string) bool {
	const prefix = "asignacion:bootstrap_"
	const suffix = ":v1"
	if !strings.HasPrefix(v, prefix) || !strings.HasSuffix(v, suffix) {
		return false
	}
	return refHex(strings.TrimSuffix(v, suffix), prefix)
}
func fechaRecibo(v string) bool {
	t, e := time.Parse(time.RFC3339Nano, v)
	if e != nil || t.IsZero() || t.Year() < 1 || t.Year() > 9999 {
		return false
	}
	_, offset := t.Zone()
	return offset == 0
}
func hashValido(v string) bool { return domain.HuellaAdministracionPerfilesValida(v) }
func refHex(v, p string) bool {
	if !strings.HasPrefix(v, p) || len(v) != len(p)+32 {
		return false
	}
	for _, c := range v[len(p):] {
		if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
