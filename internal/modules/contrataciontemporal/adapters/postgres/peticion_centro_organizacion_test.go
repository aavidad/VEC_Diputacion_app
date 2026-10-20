package postgres

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRecursoConsultaPeticionCentroLigaOrganizacionSinCambiarMaterial(t *testing.T) {
	c := ports.ConsultaPeticionCentro{
		Modo: "bandeja",
		Actor: domain.ActorPeticionCentro{
			ActorRef: "actor:centro:001", PerfilRef: "perfil:centro:001",
			CentroRef: "centro:001", PuestoRef: "puesto:001",
		},
		Referencia: "centro:001",
	}
	primero, err := RecursoConsultaPeticionCentro(c, "organizacion:desarrollo:dipgra")
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := RecursoConsultaPeticionCentro(c, "org_"+strings.Repeat("b", 32))
	if err != nil {
		t.Fatal(err)
	}
	h1, err := primero.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h2, err := segundo.HuellaContextoAutorizacionSHA256()
	if err != nil || h1 == h2 || primero.Ambitos["organizacion_ref"] == segundo.Ambitos["organizacion_ref"] ||
		primero.Atributos["material_sha256"] != segundo.Atributos["material_sha256"] {
		t.Fatalf("organizaciones sin separación de ámbito y huella: %v", err)
	}
	if _, err := RecursoConsultaPeticionCentro(c, ""); err == nil {
		t.Fatal("organización vacía aceptada")
	}
}
