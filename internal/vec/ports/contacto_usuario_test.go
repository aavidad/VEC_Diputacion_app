package ports

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestSolicitudAccesoContactoEsDTOYConservaContextoActor(t *testing.T) {
	ahora := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: domain.AuthMethodCertificate, Garantia: domain.AuthAssuranceSubstantial}
	instantanea := domain.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1, PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1, Estado: domain.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	s := SolicitudAccesoContactoUsuario{SujetoRef: actor.PersonaRef, ContextoActor: actor, FinalidadRef: "finalidad_contacto_001"}
	if s.SujetoRef != actor.PersonaRef || s.ContextoActor.PersonaRef != actor.PersonaRef || s.FinalidadRef == "" {
		t.Fatal("el DTO perdió el contexto")
	}
}

func TestOrdenRegistroContactoSoloContienePreparacionProtegida(t *testing.T) {
	orden := OrdenRegistroContactoUsuario{Preparacion: PreparacionRegistroContactoUsuario{
		SujetoRef: "per_0123456789abcdefghijkl", VersionNueva: 1,
		Sobre:          SobreContactoUsuario{Version: 1, ClaveRef: "kms://contacto", Nonce: []byte{1}, Cifrado: []byte{2}},
		PayloadNegocio: []byte("preimagen-cifrada"),
	}}
	if orden.Preparacion.SujetoRef == "" || len(orden.Preparacion.Sobre.Cifrado) == 0 || len(orden.Preparacion.PayloadNegocio) == 0 {
		t.Fatal("orden protegida incompleta")
	}
}
