package internagobierno

import (
	"context"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
)

// AmbitoOrganizacionHistorica selecciona un perfil y ámbito privados. No es
// una concesión: la operación requiere una decisión V3 y su consumo nominal.
type AmbitoOrganizacionHistorica struct {
	PerfilActivoRef string `json:"perfil_activo_ref"`
	PerfilVersion   uint64 `json:"perfil_version"`
	OrganismoRef    string `json:"organismo_ref"`
	UnidadClave     string `json:"unidad_clave"`
}

func (a AmbitoOrganizacionHistorica) Validar() error {
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdef0123456789abcdef", Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial}
	if (core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: a.PerfilActivoRef}).Validar() != nil || a.PerfilVersion == 0 {
		return ErrGobiernoInternoNoDisponible
	}
	// La validación del ámbito utiliza el mismo contrato que consumirá Personal.
	s := personal.SelectorOrganizacionHistorica{OrganismoRef: a.OrganismoRef, UnidadClave: a.UnidadClave}
	if !ambitoOHValido(s.OrganismoRef) || s.UnidadClave != "" && !ambitoOHValido(s.UnidadClave) {
		return ErrGobiernoInternoNoDisponible
	}
	return nil
}

func ambitoOHValido(v string) bool {
	if len(v) < 3 || len(v) > 128 || v[0] < 'a' || v[0] > 'z' {
		return false
	}
	for _, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

type claveContextoOrganizacionHistorica struct{}
type contextoOrganizacionHistoricaSellado struct {
	fuente *FuenteF1
	valor  ct.ContextoAutorizacionAltaV3
	ambito AmbitoOrganizacionHistorica
}

// VincularContextoOrganizacionHistorica sella una única resolución F1 después
// de mTLS y de comprobar cuenta, perfil, versión y ámbito del servidor.
func (f *FuenteF1) VincularContextoOrganizacionHistorica(ctx context.Context, ambitos map[string]AmbitoOrganizacionHistorica) (context.Context, error) {
	if f == nil || f.reloj == nil || ctx == nil || ctx.Err() != nil || ctx.Value(claveContextoOrganizacionHistorica{}) != nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	r, err := f.ResolverContexto(ctx)
	if err != nil || r.Resultado.Validar() != nil || r.Vinculo.ValidarPara(r.Resultado) != nil || !r.Vinculo.VigenteEn(f.reloj.Ahora(), r.Resultado) {
		return nil, ErrGobiernoInternoNoDisponible
	}
	a := r.Resultado.Contexto
	scope, ok := ambitos[a.Instantanea.CuentaRef]
	nominal, existe := f.porCuenta[a.Instantanea.CuentaRef]
	if !ok || !existe || scope.Validar() != nil || scope.PerfilActivoRef != a.PerfilActivoRef || scope.PerfilVersion != a.Instantanea.PerfilVersion || nominal.PerfilActivoRef != scope.PerfilActivoRef || nominal.OrganizacionRef != scope.OrganismoRef {
		return nil, httpapi.ErrAccesoRutaExactaDenegado
	}
	clon, err := r.Resultado.Clonar()
	if err != nil {
		return nil, ErrGobiernoInternoNoDisponible
	}
	r.Resultado = clon
	return context.WithValue(ctx, claveContextoOrganizacionHistorica{}, contextoOrganizacionHistoricaSellado{f, r, scope}), nil
}

func (f *FuenteF1) ContextoVinculadoOrganizacionHistorica(ctx context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error) {
	r, organismo, unidad, err := f.ContextoOriginalOrganizacionHistoricaParaAuditoria(ctx)
	if err != nil || !r.Vinculo.VigenteEn(f.reloj.Ahora(), r.Resultado) {
		return ct.ContextoAutorizacionAltaV3{}, "", "", ErrGobiernoInternoNoDisponible
	}
	return r, organismo, unidad, nil
}

// ContextoOriginalOrganizacionHistoricaParaAuditoria copia únicamente el sello
// F1 ya emitido por esta fuente. Conserva identidad histórica tras caducidad,
// sin revalidar ni renovar sesión. No concede lectura: el método actual, el
// proveedor V3 y el consumidor SQL siguen exigiendo vigencia.
func (f *FuenteF1) ContextoOriginalOrganizacionHistoricaParaAuditoria(ctx context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error) {
	var vacio ct.ContextoAutorizacionAltaV3
	if f == nil || f.reloj == nil || ctx == nil || ctx.Err() != nil {
		return vacio, "", "", ErrGobiernoInternoNoDisponible
	}
	s, ok := ctx.Value(claveContextoOrganizacionHistorica{}).(contextoOrganizacionHistoricaSellado)
	if !ok || s.fuente != f || s.ambito.Validar() != nil || s.valor.Resultado.Validar() != nil || s.valor.Vinculo.ValidarPara(s.valor.Resultado) != nil {
		return vacio, "", "", ErrGobiernoInternoNoDisponible
	}
	actor := s.valor.Resultado.Contexto
	if actor.PerfilActivoRef != s.ambito.PerfilActivoRef || actor.Instantanea.PerfilVersion != s.ambito.PerfilVersion {
		return vacio, "", "", ErrGobiernoInternoNoDisponible
	}
	r := s.valor
	clon, err := r.Resultado.Clonar()
	if err != nil {
		return vacio, "", "", ErrGobiernoInternoNoDisponible
	}
	r.Resultado = clon
	return r, s.ambito.OrganismoRef, s.ambito.UnidadClave, nil
}
