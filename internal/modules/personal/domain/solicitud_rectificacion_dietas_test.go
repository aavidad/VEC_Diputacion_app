package domain

import (
	"strings"
	"testing"
)

func TestRectificacionDietasNoPermiteAutodesignarValidador(t *testing.T) {
	actor := actorAsignacionPrueba(t)
	fecha, _ := NuevaFechaCivil("2026-09-20")
	s := SolicitudRectificacionDietas{
		Actor: actor, Operacion: SolicitarRectificacionDietas,
		RelacionRef: "rel_" + strings.Repeat("a", 24), UnidadRef: "unidad_sintetica",
		AsignacionRef: "ads_" + strings.Repeat("a", 24), VersionEsperada: 1,
		FechaReferencia: fecha, ClaveIdempotencia: "12345678-1234-4123-8123-123456789abc",
		CamposARevisar: []string{"responsable_persona_ref", "centro_ref"},
		MotivoRevision: "centro o validador incorrecto",
	}
	m, err := NuevoMaterialRectificacionDietas(s)
	if err != nil || m.Recurso().Ambitos["persona_ref"] != actor.PersonaRef ||
		m.Solicitud().CamposARevisar[0] != "centro_ref" {
		t.Fatalf("petición propia no canónica: %v", err)
	}
	s.PersonaRef = "per_" + strings.Repeat("b", 24)
	if _, err := NuevoMaterialRectificacionDietas(s); err == nil {
		t.Fatal("petición aceptó sujeto indicado por cliente")
	}
	s.PersonaRef = ""
	s.CamposARevisar = []string{"grupo_dieta"}
	if _, err := NuevoMaterialRectificacionDietas(s); err == nil {
		t.Fatal("petición aceptó grupo de dieta sin concesión separada")
	}
	s.CamposARevisar = []string{"responsable_persona_ref"}
	s.DetalleSolicitado = "per_" + strings.Repeat("c", 24)
	if _, err := NuevoMaterialRectificacionDietas(s); err == nil {
		t.Fatal("detalle aceptó referencia personal libre")
	}
	s.DetalleSolicitado = "el centro mostrado no corresponde"
	if _, err := NuevoMaterialRectificacionDietas(s); err != nil {
		t.Fatalf("detalle textual acotado fue rechazado: %v", err)
	}
}

func TestRectificacionDietasResolucionExigeOtroSujeto(t *testing.T) {
	actor := actorAsignacionPrueba(t)
	fecha, _ := NuevaFechaCivil("2026-09-20")
	s := SolicitudRectificacionDietas{
		Actor: actor, Operacion: RechazarRectificacionDietas,
		PersonaRef: actor.PersonaRef, EmpleadoRef: "emp_" + strings.Repeat("a", 24),
		RelacionRef: "rel_" + strings.Repeat("b", 24), UnidadRef: "unidad_sintetica",
		SolicitudRef: "srd_" + strings.Repeat("b", 32), FechaReferencia: fecha,
		ClaveIdempotencia: "12345678-1234-4123-8123-123456789abc",
		MotivoRevision:    "motivo administrativo",
	}
	if _, err := NuevoMaterialRectificacionDietas(s); err == nil {
		t.Fatal("resolución aceptó autodecisión")
	}
	s.PersonaRef, s.EmpleadoRef = "per_"+strings.Repeat("b", 24), "emp_"+strings.Repeat("b", 24)
	m, err := NuevoMaterialRectificacionDietas(s)
	if err != nil || m.Recurso().Ambitos["persona_ref"] != s.PersonaRef {
		t.Fatalf("resolución no ligó sujeto administrativo: %v", err)
	}
}

func TestConfirmacionRectificacionExigeCorreccionD7Ligada(t *testing.T) {
	actor := actorAsignacionPrueba(t)
	fecha, _ := NuevaFechaCivil("2026-09-20")
	ref := "srd_" + strings.Repeat("b", 32)
	s := SolicitudRectificacionDietas{
		Actor: actor, Operacion: ConfirmarRectificacionDietas,
		PersonaRef: "per_" + strings.Repeat("b", 24), EmpleadoRef: "emp_" + strings.Repeat("b", 24),
		RelacionRef: "rel_" + strings.Repeat("b", 24), UnidadRef: "unidad_sintetica",
		AsignacionRef: "ads_" + strings.Repeat("b", 24), VersionEsperada: 1,
		SolicitudRef: ref, FechaReferencia: fecha,
		ClaveIdempotencia: "12345678-1234-4123-8123-123456789abc", MotivoRevision: "corrección administrativa",
	}
	if _, err := NuevoMaterialRectificacionDietas(s); err == nil {
		t.Fatal("confirmación aceptó falta de corrección D7")
	}
	s.Correccion = &SolicitudAsignacionDietas{
		Actor: actor, Operacion: CorregirAsignacionDietas,
		PersonaRef: s.PersonaRef, EmpleadoRef: s.EmpleadoRef, RelacionRef: s.RelacionRef,
		UnidadRef: s.UnidadRef, FechaReferencia: fecha, VersionEsperada: 1,
		ClaveIdempotencia: s.ClaveIdempotencia, MotivoRevision: s.MotivoRevision,
		ProcedenciaActoRef: ref,
	}
	if _, err := NuevoMaterialRectificacionDietas(s); err != nil {
		t.Fatalf("dos materiales ligados: %v", err)
	}
	s.Correccion.ProcedenciaActoRef = "otro_acto"
	if _, err := NuevoMaterialRectificacionDietas(s); err == nil {
		t.Fatal("confirmación aceptó corrección D7 de otro acto")
	}
}
