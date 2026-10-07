package main

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

type envoltura struct {
	Estado           string           `json:"estado"`
	Codigo           *string          `json:"codigo"`
	Recibo           *reciboFuentes   `json:"recibo"`
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
type reciboFuentes struct {
	Esquema               string   `json:"esquema"`
	CA                    reciboCA `json:"ca"`
	IS                    reciboIS `json:"is"`
	ReciboRef             string   `json:"recibo_ref"`
	OperacionRef          string   `json:"operacion_ref"`
	PlanSHA256            string   `json:"plan_sha256"`
	PreimagenSHA256       string   `json:"preimagen_sha256"`
	ConfiguracionSHA256   string   `json:"configuracion_sha256"`
	OperadorLogin         string   `json:"operador_login"`
	AprobacionRef         string   `json:"aprobacion_ref"`
	AuditoriaRef          string   `json:"auditoria_ref"`
	AuditoriaSecuencia    uint64   `json:"auditoria_secuencia"`
	AuditoriaHuellaSHA256 string   `json:"auditoria_huella_sha256"`
	RegistradaEn          string   `json:"registrada_en"`
}
type reciboCA struct {
	Esquema       string  `json:"esquema"`
	Version       uint64  `json:"version"`
	ReciboRef     string  `json:"recibo_ref"`
	OperacionRef  string  `json:"operacion_ref"`
	PlanSHA256    string  `json:"plan_sha256"`
	AprobacionRef string  `json:"aprobacion_ref"`
	AlcanceFuente string  `json:"alcance_fuente"`
	RegistradaEn  string  `json:"registrada_en"`
	Datos         datosCA `json:"datos"`
	HuellaSHA256  string  `json:"huella_sha256"`
}
type reciboIS struct {
	Esquema       string  `json:"esquema"`
	Version       uint64  `json:"version"`
	ReciboRef     string  `json:"recibo_ref"`
	OperacionRef  string  `json:"operacion_ref"`
	PlanSHA256    string  `json:"plan_sha256"`
	AprobacionRef string  `json:"aprobacion_ref"`
	AlcanceFuente string  `json:"alcance_fuente"`
	RegistradaEn  string  `json:"registrada_en"`
	Datos         datosIS `json:"datos"`
	HuellaSHA256  string  `json:"huella_sha256"`
}
type datosIS struct {
	Personas    [2]cuentaPersona `json:"personas"`
	PoliticaRef string           `json:"politica_ref"`
}
type datosCA struct {
	OrganizacionRef     string             `json:"organizacion_ref"`
	OrganizacionVersion uint64             `json:"organizacion_version"`
	Personas            [2]cuentaPersonaCA `json:"personas"`
}
type cuentaPersona struct {
	PersonaRef            string `json:"persona_ref"`
	CuentaOrdinariaRef    string `json:"cuenta_ordinaria_ref"`
	CuentaPrivilegiadaRef string `json:"cuenta_privilegiada_ref"`
	VersionTitularidad    uint64 `json:"version_titularidad"`
}
type cuentaPersonaCA struct {
	PersonaRef              string `json:"persona_ref"`
	CuentaOrdinariaRef      string `json:"cuenta_ordinaria_ref"`
	CuentaPrivilegiadaRef   string `json:"cuenta_privilegiada_ref"`
	VersionTitularidad      uint64 `json:"version_titularidad"`
	PersonaVersion          uint64 `json:"persona_version"`
	ProyeccionCuentaVersion uint64 `json:"proyeccion_cuenta_version"`
}

var errEnvoltura = errors.New("envoltura_invalida")

func validarEnvoltura(b []byte, plan domain.PlanFuentesInicialesAdminV1, sha string) (envoltura, error) {
	var e envoltura
	if len(b) == 0 || len(b) > limiteDocumento || decodificarEstricto(b, &e) != nil {
		return e, errEnvoltura
	}
	a := e.AuditoriaIntento
	if !refHex(a.AuditoriaRef, "aud_v3_fi_") || a.Secuencia == 0 || a.Secuencia > 1<<53-1 || !hashValido(a.HuellaSHA256) || !refHex(a.CorrelacionRef, "correlacion_") || !fechaRecibo(a.RegistradaEn) {
		return e, errEnvoltura
	}
	switch e.Estado {
	case "permitido":
		if e.Codigo != nil || e.Recibo == nil || !reciboValido(e.Recibo, plan, sha) {
			return e, errEnvoltura
		}
	case "denegado":
		if e.Codigo == nil || *e.Codigo != "fuentes_rechazadas" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	case "error":
		if e.Codigo == nil || *e.Codigo != "fuentes_no_disponibles" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	default:
		return e, errEnvoltura
	}
	return e, nil
}
func reciboValido(r *reciboFuentes, p domain.PlanFuentesInicialesAdminV1, sha string) bool {
	if r.Esquema != "vec.admin.fuentes-confirmadas.v1" || !refHex(r.ReciboRef, "recibo_fuentes:") || r.OperacionRef != p.OperacionRef || r.PlanSHA256 != sha ||
		!hashValido(r.PreimagenSHA256) || !hashValido(r.ConfiguracionSHA256) || !utf8.ValidString(r.OperadorLogin) || len(r.OperadorLogin) == 0 || len(r.OperadorLogin) > 63 ||
		len(r.AprobacionRef) == 0 || len(r.AprobacionRef) > 128 || !refHex(r.AuditoriaRef, "aud_v3_f_") || r.AuditoriaSecuencia == 0 || r.AuditoriaSecuencia > 1<<53-1 ||
		!hashValido(r.AuditoriaHuellaSHA256) || !fechaRecibo(r.RegistradaEn) {
		return false
	}
	ca, id := r.CA, r.IS
	if ca.Esquema != "vec.ca.fuentes-iniciales-admin.v1" || id.Esquema != "vec.is.fuentes-iniciales-admin.v1" || ca.Version != 1 || id.Version != 1 ||
		!refHex(ca.ReciboRef, "recibo_ca_fuentes:") || !refHex(id.ReciboRef, "recibo_is_fuentes:") || ca.OperacionRef != p.OperacionRef || id.OperacionRef != p.OperacionRef ||
		ca.PlanSHA256 != sha || id.PlanSHA256 != sha || ca.AprobacionRef != r.AprobacionRef || id.AprobacionRef != r.AprobacionRef ||
		ca.AlcanceFuente != p.AlcanceFuente || id.AlcanceFuente != p.AlcanceFuente || !fechaRecibo(ca.RegistradaEn) || !fechaRecibo(id.RegistradaEn) ||
		!hashValido(ca.HuellaSHA256) || !hashValido(id.HuellaSHA256) || ca.Datos.OrganizacionRef != p.Organizacion.OrganizacionRef || ca.Datos.OrganizacionVersion != 1 ||
		id.Datos.PoliticaRef != p.PoliticaADMIN.PoliticaRef {
		return false
	}
	cuentas := map[string]bool{}
	personas := map[string]bool{}
	caPersonas := map[string]cuentaPersonaCA{}
	for _, x := range ca.Datos.Personas {
		caPersonas[x.PersonaRef] = x
	}
	if len(caPersonas) != 2 {
		return false
	}
	for _, x := range id.Datos.Personas {
		y := caPersonas[x.PersonaRef]
		admitida := x.PersonaRef == p.Personas[0].PersonaRef || x.PersonaRef == p.Personas[1].PersonaRef
		if !admitida || personas[x.PersonaRef] || x.VersionTitularidad != 1 || !refOpaca(x.CuentaOrdinariaRef, "cta_") || !refOpaca(x.CuentaPrivilegiadaRef, "cta_") ||
			y.PersonaRef != x.PersonaRef || y.CuentaOrdinariaRef != x.CuentaOrdinariaRef || y.CuentaPrivilegiadaRef != x.CuentaPrivilegiadaRef || y.VersionTitularidad != 1 || y.PersonaVersion != 1 || y.ProyeccionCuentaVersion != 1 {
			return false
		}
		personas[x.PersonaRef] = true
		for _, ref := range []string{x.CuentaOrdinariaRef, x.CuentaPrivilegiadaRef} {
			if cuentas[ref] {
				return false
			}
			cuentas[ref] = true
		}
	}
	return true
}
func fechaRecibo(valor string) bool {
	t, e := time.Parse(time.RFC3339Nano, valor)
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
func refOpaca(v, p string) bool {
	if !strings.HasPrefix(v, p) || len(v) < len(p)+22 || len(v) > len(p)+128 {
		return false
	}
	for _, c := range v[len(p):] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
