package adminperfiles

import (
	"context"
	"strings"
	"time"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
)

// VinculoSesionADMIN procede exclusivamente del acuse IS16 confirmado. El
// contexto interno evita sustituirlo por datos HTTP o buscar otra sesión.
type VinculoSesionADMIN struct {
	Referencia                                                 string
	Version                                                    uint64
	HuellaSHA256                                               string
	AutenticacionRef, SesionRef                                string
	PersonaRef, CuentaRef, CuentaOrdinariaRef, PerfilActivoRef string
	CertificadoSHA256, CASHA256                                string
	VinculoCertificadoRef                                      string
	VinculoCertificadoVersion                                  uint64
	PoliticaRef, PoliticaSHA256                                string
	SeleccionRevision                                          uint64
	ControlSesionRef                                           string
	ControlSesionRevision                                      uint64
	ControlSesionSHA256                                        string
	VinculadaEn, VigenteHasta                                  time.Time
	FuenteRef, FuenteSHA256                                    string
}

func (v VinculoSesionADMIN) Validar() error {
	if !strings.HasPrefix(v.Referencia, "vis_") || len(v.Referencia) != 36 || !hexVinculo(v.Referencia[4:]) || v.Version == 0 ||
		!huella(v.HuellaSHA256) || !referencia(v.AutenticacionRef, "aut_") || !referencia(v.SesionRef, "ses_") ||
		!referencia(v.PersonaRef, "per_") || !referencia(v.CuentaRef, "cta_") || !referencia(v.CuentaOrdinariaRef, "cta_") || v.CuentaRef == v.CuentaOrdinariaRef ||
		!referencia(v.PerfilActivoRef, "prf_") || !huella(v.CertificadoSHA256) || !huella(v.CASHA256) ||
		!referencia(v.VinculoCertificadoRef, "vca_") || v.VinculoCertificadoVersion == 0 || !referencia(v.PoliticaRef, "pga_") || !huella(v.PoliticaSHA256) || v.SeleccionRevision == 0 ||
		!referencia(v.ControlSesionRef, "cse_") || v.ControlSesionRevision == 0 || !huella(v.ControlSesionSHA256) ||
		!instante(v.VinculadaEn) || !instante(v.VigenteHasta) || !v.VigenteHasta.After(v.VinculadaEn) ||
		v.FuenteRef == "" || len(v.FuenteRef) > 128 || !huella(v.FuenteSHA256) {
		return api.ErrConfiguracionIncompleta
	}
	return nil
}
func hexVinculo(s string) bool {
	for _, c := range s {
		if !(c >= 'a' && c <= 'f' || c >= '0' && c <= '9') {
			return false
		}
	}
	return len(s) == 32
}
func (VinculoSesionADMIN) MarshalJSON() ([]byte, error) {
	return []byte(`{"vinculo_sesion":"oculto"}`), nil
}

type claveVinculoSesionADMIN struct{}

func ContextoConVinculoSesionADMIN(ctx context.Context, v VinculoSesionADMIN) (context.Context, error) {
	if ctx == nil || ctx.Err() != nil || v.Validar() != nil {
		return nil, api.ErrConfiguracionIncompleta
	}
	if _, err := VinculoSesionADMINDeContexto(ctx); err == nil {
		return nil, api.ErrConfiguracionIncompleta
	}
	return context.WithValue(ctx, claveVinculoSesionADMIN{}, v), nil
}
func VinculoSesionADMINDeContexto(ctx context.Context) (VinculoSesionADMIN, error) {
	if ctx == nil || ctx.Err() != nil {
		return VinculoSesionADMIN{}, api.ErrConfiguracionIncompleta
	}
	v, ok := ctx.Value(claveVinculoSesionADMIN{}).(VinculoSesionADMIN)
	if !ok || v.Validar() != nil {
		return VinculoSesionADMIN{}, api.ErrConfiguracionIncompleta
	}
	return v, nil
}

// FuenteCuentasADMINConAcuse no modifica los puertos del núcleo V2.
type FuenteCuentasADMINConAcuse interface {
	VincularSesionADMINConAcuse(context.Context, ObservacionADMIN, CuentaADMIN, ReferenciasSesionADMIN) (VinculoSesionADMIN, error)
}
