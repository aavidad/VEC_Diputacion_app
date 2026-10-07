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
	Recibo           *reciboIdentidad `json:"recibo"`
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
type reciboIdentidad struct {
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
	IS                    reciboIS `json:"is"`
	CA                    reciboCA `json:"ca"`
}

var errEnvoltura = errors.New("envoltura_invalida")

func validarEnvoltura(b []byte, p domain.PlanIdentidadInternaSinteticaV1, sha string, cfg conexionPrivada, modo string) (envoltura, error) {
	var e envoltura
	if len(b) == 0 || len(b) > limiteDocumento || decodificarEstricto(b, &e) != nil {
		return e, errEnvoltura
	}
	a := e.AuditoriaIntento
	if a.AuditoriaRef == "" || a.Secuencia == 0 || a.Secuencia > 1<<53-1 || !hashValido(a.HuellaSHA256) || a.CorrelacionRef == "" || !fechaRecibo(a.RegistradaEn) {
		return e, errEnvoltura
	}
	switch e.Estado {
	case "permitido":
		r := e.Recibo
		if e.Codigo != nil || r == nil || r.ReciboRef == "" || r.OperacionRef != p.OperacionRef || r.PlanSHA256 != sha || r.PreimagenSHA256 != cfg.PreimagenSHA256 || r.ConfiguracionSHA256 != cfg.ConfiguracionSHA256 || r.OperadorLogin != cfg.LoginEsperado || r.AprobacionRef != cfg.AprobacionRef || r.AuditoriaRef == "" || r.AuditoriaSecuencia == 0 || r.AuditoriaSecuencia > 1<<53-1 || !hashValido(r.AuditoriaHuellaSHA256) || !fechaRecibo(r.RegistradaEn) || modo == "reconcile" && !e.Replay || !r.IS.valido(p, sha, cfg) || !r.CA.valido(p, sha, cfg) || r.IS.Datos.CuentaOrdinariaRef != r.CA.Datos.CuentaOrdinariaRef {
			return e, errEnvoltura
		}
	case "denegado", "error":
		if e.Codigo == nil || *e.Codigo == "" || e.Recibo != nil || e.Replay {
			return e, errEnvoltura
		}
	default:
		return e, errEnvoltura
	}
	return e, nil
}
func hashValido(s string) bool {
	return domain.HuellaAdministracionPerfilesValida(s) && s != strings.Repeat("0", 64)
}
func fechaRecibo(s string) bool {
	t, e := time.Parse(time.RFC3339Nano, s)
	if e != nil || t.IsZero() {
		return false
	}
	_, o := t.Zone()
	return o == 0
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
type datosIS struct {
	PersonaRef         string `json:"persona_ref"`
	CuentaOrdinariaRef string `json:"cuenta_ordinaria_ref"`
	VersionTitularidad uint64 `json:"version_titularidad"`
}
type datosCA struct {
	OrganizacionRef         string `json:"organizacion_ref"`
	OrganizacionVersion     uint64 `json:"organizacion_version"`
	PersonaRef              string `json:"persona_ref"`
	PersonaVersion          uint64 `json:"persona_version"`
	CuentaOrdinariaRef      string `json:"cuenta_ordinaria_ref"`
	ProyeccionCuentaVersion uint64 `json:"proyeccion_cuenta_version"`
}

func (r reciboIS) valido(p domain.PlanIdentidadInternaSinteticaV1, sha string, c conexionPrivada) bool {
	return r.Esquema == "vec.is.identidad-interna-sintetica.v1" && r.Version == 1 && r.ReciboRef != "" && r.OperacionRef == p.OperacionRef && r.PlanSHA256 == sha && r.AprobacionRef == c.AprobacionRef && r.AlcanceFuente == p.AlcanceFuente && fechaRecibo(r.RegistradaEn) && hashValido(r.HuellaSHA256) && r.Datos.PersonaRef == p.Persona.PersonaRef && r.Datos.VersionTitularidad == 1 && refCuenta(r.Datos.CuentaOrdinariaRef)
}
func (r reciboCA) valido(p domain.PlanIdentidadInternaSinteticaV1, sha string, c conexionPrivada) bool {
	return r.Esquema == "vec.ca.identidad-interna-sintetica.v1" && r.Version == 1 && r.ReciboRef != "" && r.OperacionRef == p.OperacionRef && r.PlanSHA256 == sha && r.AprobacionRef == c.AprobacionRef && r.AlcanceFuente == p.AlcanceFuente && fechaRecibo(r.RegistradaEn) && hashValido(r.HuellaSHA256) && r.Datos.OrganizacionRef == p.Organizacion.OrganizacionRef && r.Datos.OrganizacionVersion == p.Organizacion.VersionEsperada && r.Datos.PersonaRef == p.Persona.PersonaRef && r.Datos.PersonaVersion == 1 && r.Datos.ProyeccionCuentaVersion == 1 && refCuenta(r.Datos.CuentaOrdinariaRef)
}
func refCuenta(v string) bool {
	if !strings.HasPrefix(v, "cta_") || len(v) < 26 || len(v) > 132 {
		return false
	}
	for _, c := range v[4:] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
