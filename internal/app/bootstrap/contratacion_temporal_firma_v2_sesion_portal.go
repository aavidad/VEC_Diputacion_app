package bootstrap

import (
	"context"
	"net/http"

	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
)

type resolvedorSesionPortalFirmaCT struct {
	legacy vechttp.DemoIdentityResolver
	firma  *montajeFirmaVecR5CT
}

func resolvedorCarcasaFirmaCT(legacy vechttp.DemoIdentityResolver, firma *montajeFirmaVecR5CT) vechttp.DemoIdentityResolver {
	if firma == nil {
		return legacy
	}
	return resolvedorSesionPortalFirmaCT{legacy: legacy, firma: firma}
}

// Sólo las tres consultas de la carcasa leen la sesión nominal. Sus permisos
// informativos siguen saliendo de la lista positiva común, sin permisos CT.
func (s resolvedorSesionPortalFirmaCT) ResolveDemoIdentity(ctx context.Context, r *http.Request) (core.Principal, error) {
	if s.firma == nil || r == nil || r.URL == nil || r.Method != http.MethodGet ||
		(r.URL.Path != "/api/vec/session" && r.URL.Path != "/api/vec/modules" && r.URL.Path != "/api/vec/workspace") {
		return s.legacy.ResolveDemoIdentity(ctx, r)
	}
	_, huella, valido := certificadoSesionFirmanteV2(r, s.firma.reloj.Ahora())
	if !valido {
		return core.Principal{}, ErrMaterialDesarrolloInvalido
	}
	if _, nominal := s.firma.nominal.porCertificado[huella]; !nominal {
		return s.legacy.ResolveDemoIdentity(ctx, r)
	}
	preparada, err := s.firma.nominal.prepararPeticionNominal(r.WithContext(ctx))
	if err != nil {
		return core.Principal{}, err
	}
	operativo, err := s.firma.nominal.contexto(preparada.Context())
	if err != nil {
		return core.Principal{}, err
	}
	actor, err := operativo.Vinculo.Datos()
	if err != nil {
		return core.Principal{}, err
	}
	snapshot, err := s.firma.fuenteAutorizacion.ObtenerInstantaneaAutorizacion(preparada.Context(), actor.PrincipalID, actor.PerfilActivoRef)
	if err != nil {
		return core.Principal{}, err
	}
	if snapshot.Validar() != nil || !snapshot.AsignacionPerfil.VigenteEn(s.firma.reloj.Ahora()) ||
		snapshot.ControlVigenciaVersionRol.Estado != core.EstadoControlVigenciaVersionRolHabilitada {
		return core.Principal{}, ErrMaterialDesarrolloInvalido
	}
	principal := operativo.Resultado.Contexto.Principal
	principal.Roles = []string{snapshot.VersionRol.RolID}
	principal.Permissions = nil
	if principal.DisplayName == "" {
		principal.DisplayName = r.TLS.VerifiedChains[0][0].Subject.CommonName
	}
	return principal, principal.Validate()
}
