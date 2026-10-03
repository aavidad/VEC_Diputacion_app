package httpcopias

import (
	"strings"
	"testing"
	"time"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	"vec-diputacion-granada/internal/vec/domain"
)

func sesionPrueba(t *testing.T) p.Sesion {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	persona := "per_" + strings.Repeat("a", 22)
	perfil := "prf_" + strings.Repeat("b", 22)
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_" + strings.Repeat("c", 22), Metodo: domain.AuthMethodCertificate,
		Garantia: domain.AuthAssuranceHigh,
	}
	i := domain.InstantaneaContextoActor{
		VinculoRef: "vca_" + strings.Repeat("d", 22), VinculoVersion: 1,
		CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: persona, PersonaVersion: 1,
		PerfilActivoRef: perfil, PerfilVersion: 1, Estado: domain.EstadoVinculoContextoActorActivo,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}
	actor, err := domain.NuevoContextoActor(cuenta, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	rol := domain.VersionRol{
		RolID: "operador_plataforma", Version: 1, Nombre: "Operador plataforma",
		Estado:       domain.EstadoVersionRolPublicada,
		Concesiones:  []domain.ConcesionRol{{Accion: "copias_consultar", ModuloID: "administracion", TipoRecurso: "copias", Finalidades: []string{"operacion_plataforma"}, GarantiaMinima: domain.AuthAssuranceSubstantial}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-admin", Version: 1, PerfilActivoRef: perfil, PrincipalID: persona,
			VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
			Ambitos:      []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "responsable-seguridad", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: rol,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: rol.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
	if err := snapshot.Validar(); err != nil {
		t.Fatal(err)
	}
	return p.Sesion{Actor: actor, Instantanea: snapshot,
		CorrelacionRef: "correlacion_" + strings.Repeat("e", 32)}
}
