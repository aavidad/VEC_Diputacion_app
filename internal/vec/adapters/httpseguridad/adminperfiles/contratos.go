// Package adminperfiles adapta la sesión ADMIN a las autoridades centrales de
// identidad, contexto y autorización. Una petición nunca provisiona perfiles.
package adminperfiles

import (
	"context"
	"reflect"
	"strings"
	"time"

	h "vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Host es el nombre sin puerto que PostgreSQL contrasta con host_admin.
// Autoridad es la cabecera Host exacta que la frontera exigió (con puerto si el
// público no es 443); no viaja a SQL y el proveedor la vuelve a exigir.
type ObservacionADMIN struct {
	Entorno, Host, Audiencia, CertificadoSHA256, CASHA256 string
	Autoridad                                             string
	AutenticacionVerificadaEn, RevocacionVerificadaEn     time.Time
	CRLVigenteHasta, CertificadoVigenteHasta              time.Time
}

func (o ObservacionADMIN) Valida(ahora time.Time) bool {
	return (o.Entorno == "desarrollo" || o.Entorno == "cidonia") && o.Host != "" && o.Audiencia != "" &&
		huella(o.CertificadoSHA256) && huella(o.CASHA256) && instante(o.AutenticacionVerificadaEn) &&
		instante(o.RevocacionVerificadaEn) && instante(o.CRLVigenteHasta) && instante(o.CertificadoVigenteHasta) &&
		!o.AutenticacionVerificadaEn.After(ahora) && !o.RevocacionVerificadaEn.After(ahora) &&
		!o.RevocacionVerificadaEn.Before(o.AutenticacionVerificadaEn) &&
		ahora.Before(o.CRLVigenteHasta) && ahora.Before(o.CertificadoVigenteHasta)
}

// CuentaADMIN procede de las fachadas IS12/CA23 y de la asignación nominal
// vigente. Rol y perfil nunca se eligen mediante la petición HTTP.
type CuentaADMIN struct {
	materialCuentaSQL                                          string
	SujetoID, CuentaID, CuentaOrdinariaID                      string
	PersonaRef, CuentaRef, CuentaOrdinariaRef, PerfilActivoRef string
	RolID, VinculoRef                                          string
	VinculoVersion                                             uint64
	SeleccionRevision                                          uint64
	PoliticaGarantiaRef, PoliticaGarantiaHuellaSHA256          string
	GarantiaObservada                                          domain.AuthAssurance
	VigenteHasta                                               time.Time
}

func (c CuentaADMIN) Valida(ahora time.Time) bool {
	if !referencia(c.PersonaRef, "per_") || !referencia(c.CuentaRef, "cta_") ||
		!referencia(c.CuentaOrdinariaRef, "cta_") || c.CuentaRef == c.CuentaOrdinariaRef ||
		!referencia(c.PerfilActivoRef, "prf_") || !referencia(c.VinculoRef, "vca_") ||
		c.VinculoVersion == 0 || c.SeleccionRevision == 0 || c.RolID != "administracion_perfiles" ||
		!referencia(c.PoliticaGarantiaRef, "pga_") || !huella(c.PoliticaGarantiaHuellaSHA256) ||
		c.GarantiaObservada != domain.AuthAssuranceHigh || !instante(c.VigenteHasta) || !ahora.Before(c.VigenteHasta) {
		return false
	}
	for _, id := range []string{c.SujetoID, c.CuentaID, c.CuentaOrdinariaID} {
		if id == "" || len(id) > 512 {
			return false
		}
		for _, ch := range id {
			if ch <= 32 || ch >= 127 {
				return false
			}
		}
	}
	return c.CuentaID != c.CuentaOrdinariaID && strings.ToLower(c.CuentaID) == c.CuentaID &&
		strings.ToLower(c.CuentaOrdinariaID) == c.CuentaOrdinariaID
}

type ReferenciasSesionADMIN struct{ AutenticacionRef, SesionRef string }

type FuenteCuentasADMIN interface {
	ResolverCuentaADMIN(context.Context, ObservacionADMIN) (CuentaADMIN, error)
	VincularSesionADMIN(context.Context, ObservacionADMIN, CuentaADMIN, ReferenciasSesionADMIN) error
}

type Dependencias struct {
	Cuentas      FuenteCuentasADMIN
	Registro     h.RegistroSesiones
	Revalidador  domain.RevalidadorAutenticacionActorV1
	Contextos    domain.ResolutorContextoActorRegistradoV2
	Autorizacion ports.FuenteAutorizacion
	Reloj        h.Reloj
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Func, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func referencia(s, prefijo string) bool {
	if !strings.HasPrefix(s, prefijo) || len(s) < len(prefijo)+22 || len(s) > len(prefijo)+128 {
		return false
	}
	for _, ch := range s[len(prefijo):] {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '_' || ch == '-') {
			return false
		}
	}
	return true
}

func huella(s string) bool {
	if len(s) != 64 || strings.Trim(s, "0") == "" {
		return false
	}
	for _, ch := range s {
		if !(ch >= 'a' && ch <= 'f' || ch >= '0' && ch <= '9') {
			return false
		}
	}
	return true
}

func instante(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}
