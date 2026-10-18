package domain

import (
	"strings"
	"testing"
)

func TestMaterialPropuestaVersionarRolBolsaB103FijaCuatroSelecciones(t *testing.T) {
	c, i, ahora := escenarioVersionarRolBolsaB103(t)
	plan, err := PrepararPlanVersionarRolBolsaB103(c, i, ahora)
	if err != nil {
		t.Fatal(err)
	}
	m := MaterialPropuestaVersionarRolBolsaB103{
		OperacionRef:         "propuesta_admin:" + strings.Repeat("1", 32),
		ProponentePersonaRef: "per_" + strings.Repeat("p", 22), PerfilActivoRef: "prf_" + strings.Repeat("q", 22),
		AsignacionPerfilRef: "asignacion:admin:uno", Plan: plan,
	}
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	copia := m.Copia()
	copia.Plan.Selecciones[0].EntradaRef = "accion:alterada"
	copia.Plan.Asignaciones[0].Documento.Ambitos[0].Valores[0] = "unidad:alterada"
	if m.Plan.Selecciones[0].EntradaRef == copia.Plan.Selecciones[0].EntradaRef ||
		m.Plan.Asignaciones[0].Documento.Ambitos[0].Valores[0] == copia.Plan.Asignaciones[0].Documento.Ambitos[0].Valores[0] {
		t.Fatal("el material comparte memoria mutable")
	}
	h2, err := m.HuellaSHA256()
	if err != nil || h != h2 {
		t.Fatal("la copia alteró la huella original")
	}
	copia = m.Copia()
	copia.Plan.DefinicionNueva.Concesiones[1].CamposPermitidos[0] = "dni"
	hAlterada, err := copia.HuellaSHA256()
	if err != nil || hAlterada == h {
		t.Fatal("el material alterado debe tener otra huella")
	}
}

func TestSolicitudPropuestaB103AdmiteExtensionContactoDeUnaSeleccion(t *testing.T) {
	_, i, ahora := escenarioVersionarRolBolsaB103(t)
	vinculo, resultado, _ := vinculoAutenticacionActorV2Prueba(t, ahora)
	base := solicitudActoAdministracionPerfilesPrueba(t)
	base.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID = resultado.Contexto.PersonaRef
	base.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef = resultado.Contexto.PerfilActivoRef
	q := SolicitudPropuestaVersionarRolBolsaB103{OperacionRef: "propuesta_admin:" + strings.Repeat("a", 32), Actor: resultado.Contexto, Evidencia: EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo}, InstantaneaAutorizacion: base.InstantaneaAutorizacion, Intencion: i, HuellaPlanEsperada: strings.Repeat("a", 64), CorrelacionRef: base.CorrelacionRef}
	for _, n := range []int{1, 4, 0, 2, 17} {
		q.Intencion.Selecciones = make([]SeleccionAccionAdministracionV1, n)
		err := q.Validar()
		if (n > 0) != (err == nil) {
			t.Fatalf("selecciones=%d error=%v actor=%v evidencia=%v snapshot=%v motivo=%v acto=%v", n, err, q.Actor.Validar(), q.Evidencia.ValidarPara(q.Actor), q.InstantaneaAutorizacion.Validar(), ReferenciaMotivoAutorizacionV2Valida(q.Intencion.Motivo), ReferenciaActoAdministracionValida(q.Intencion.ReferenciaActo))
		}
	}
}
