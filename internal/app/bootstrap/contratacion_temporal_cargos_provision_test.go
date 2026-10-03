package bootstrap

import (
	"context"
	"testing"
	"time"

	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type fuenteCargoPrueba struct{ instantanea core.InstantaneaAutorizacion }

func (f fuenteCargoPrueba) ObtenerInstantaneaAutorizacion(_ context.Context, principal, perfil string) (core.InstantaneaAutorizacion, error) {
	if principal != f.instantanea.AsignacionPerfil.PrincipalID || perfil != f.instantanea.AsignacionPerfil.PerfilActivoRef {
		return core.InstantaneaAutorizacion{}, vecports.ErrFuenteAutorizacionNoDisponible
	}
	return f.instantanea, nil
}

func casoCargoJefatura(t *testing.T, principal, perfil string) (ConfiguracionProvisionCargoCT, instantaneaPublicadaDesarrollo, time.Time) {
	t.Helper()
	ahora := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	i, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(principal, perfil, ahora,
		rolesCargoCT["jefatura_servicio_rrhh"], "Jefatura de Servicio RRHH de prueba", "cargo-jefatura-ct-v1",
		[]core.ConcesionRol{{Accion: "contratacion_temporal.informe.consultar", ModuloID: "contratacion_temporal", TipoRecurso: "informe",
			Finalidades: []string{"tramitar_informe"}, GarantiaMinima: core.AuthAssuranceHigh}},
		[]core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{"org:granada"}}, {Clave: "unidad_ref", Valores: []string{"unidad:rrhh"}}})
	if err != nil {
		t.Fatal(err)
	}
	huella, err := i.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaRol, err := i.VersionRol.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	c := ConfiguracionProvisionCargoCT{Version: 1, Cargo: "jefatura_servicio_rrhh", PrincipalID: principal, PerfilRef: perfil,
		OrganizacionRef: "org:granada", UnidadRef: "unidad:servicio-personal",
		VigenteDesde: ahora.Add(time.Minute), VigenteHasta: ahora.Add(24 * time.Hour),
		AprobacionInstalacionRef: "aprobacion:instalacion:ct:001", AprobadorPrincipalID: "per:responsable-seguridad",
		PreimagenSHA256: huella, VersionRolSHA256: huellaRol}
	return c, instantaneaPublicadaDesarrollo{instantanea: i, actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo,
		actualizadaPor: i.AsignacionPerfil.EmitidaPor, actoControl: actoControlRolCTDesarrollo}, ahora
}

func prepararCargoPrueba(c ConfiguracionProvisionCargoCT, p instantaneaPublicadaDesarrollo, ahora time.Time) (PlanProvisionCargoCT, error) {
	return prepararProvisionCargoCT(context.Background(), fuenteCargoPrueba{p.instantanea},
		func(context.Context, string) (instantaneaPublicadaDesarrollo, bool, error) { return p, true, nil }, c, ahora)
}

func TestProvisionCargoJefaturaVigenciaAmbitoYPreimagen(t *testing.T) {
	c, publicada, ahora := casoCargoJefatura(t, "per:ana-molina", "perfil:ana-molina:jefatura")
	plan, err := prepararCargoPrueba(c, publicada, ahora)
	if err != nil || plan.resumen.Estado != "pendiente_canal_autorizado" ||
		plan.objetivo.AsignacionPerfil.Version != publicada.instantanea.AsignacionPerfil.Version+1 ||
		plan.objetivo.VersionRol.Referencia() != publicada.instantanea.VersionRol.Referencia() {
		t.Fatalf("plan de jefatura no conservador: %+v, %v", plan.resumen, err)
	}
	propio := core.RecursoAutorizable{Referencia: "informe:ejemplo", ModuloID: "contratacion_temporal", Tipo: "informe",
		Ambitos: map[string]string{"organizacion_ref": "org:granada", "unidad_ref": "unidad:servicio-personal"}}
	ajeno := core.RecursoAutorizable{Referencia: "informe:ajeno", ModuloID: "contratacion_temporal", Tipo: "informe",
		Ambitos: map[string]string{"organizacion_ref": "org:otra", "unidad_ref": "unidad:servicio-personal"}}
	if !plan.objetivo.AsignacionPerfil.VigenteEn(c.VigenteDesde) ||
		plan.objetivo.AsignacionPerfil.VigenteEn(c.VigenteHasta) ||
		!plan.objetivo.AsignacionPerfil.Cubre(propio) || plan.objetivo.AsignacionPerfil.Cubre(ajeno) {
		t.Fatal("vigencia o ámbito de jefatura incorrectos")
	}
	c.PreimagenSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := prepararCargoPrueba(c, publicada, ahora); err == nil {
		t.Fatal("la preimagen incorrecta produjo plan")
	}
	c.PreimagenSHA256, _ = publicada.instantanea.AsignacionPerfil.HuellaSHA256()
	c.VersionRolSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := prepararCargoPrueba(c, publicada, ahora); err == nil {
		t.Fatal("la huella incorrecta del rol produjo plan")
	}
}

func TestProvisionCargoRechazaRevocacionYDosIdentidades(t *testing.T) {
	c, publicada, ahora := casoCargoJefatura(t, "per:ana-molina", "perfil:ana-molina:jefatura")
	publicada.instantanea.AsignacionPerfil.Estado = core.EstadoAsignacionPerfilRevocada
	publicada.instantanea.AsignacionPerfil.RevocadaPor = "seguridad:operador"
	publicada.instantanea.AsignacionPerfil.RevocadaEn = ahora
	publicada.instantanea.AsignacionPerfil.RevocacionRef = "revocacion:ejemplo"
	if _, err := prepararCargoPrueba(c, publicada, ahora); err == nil {
		t.Fatal("la asignación revocada produjo plan")
	}
	c, publicada, ahora = casoCargoJefatura(t, "per:ana-molina", "perfil:ana-molina:jefatura")
	c.AprobadorPrincipalID = c.PrincipalID
	if _, err := prepararCargoPrueba(c, publicada, ahora); err == nil {
		t.Fatal("autoconcesión aprobada")
	}
	c, publicada, ahora = casoCargoJefatura(t, "per:ana-molina", "perfil:ana-molina:jefatura")
	c.PrincipalID = "per:lucia-ruiz"
	if _, err := prepararCargoPrueba(c, publicada, ahora); err == nil {
		t.Fatal("preimagen de otra identidad aceptada")
	}
	segunda, otra, _ := casoCargoJefatura(t, "per:lucia-ruiz", "perfil:lucia-ruiz:jefatura")
	if _, err := prepararCargoPrueba(segunda, otra, ahora); err != nil {
		t.Fatalf("segunda identidad con perfil propio rechazada: %v", err)
	}
}
