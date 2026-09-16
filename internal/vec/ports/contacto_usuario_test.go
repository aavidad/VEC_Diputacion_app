package ports

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestSolicitudAccesoContactoConsumeContextoActor(t *testing.T) {
	ahora := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceSubstantial}
	instantanea := domain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: domain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	s := SolicitudAccesoContactoUsuario{SujetoRef: actor.PersonaRef, ContextoActor: actor, FinalidadRef: "finalidad_contacto_001"}
	if s.Validar() != nil {
		t.Fatal("solicitud valida rechazada")
	}
	s.ContextoActor.PersonaRef = ""
	if s.Validar() == nil {
		t.Fatal("acepto contexto invalido")
	}
}
