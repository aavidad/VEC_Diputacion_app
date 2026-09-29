package postgres

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRecursoEntregaPostIncluyeAmbitosRatificadosYGETNoLosHereda(t *testing.T) {
	get := ports.MaterialEntregaPeticionCentro{
		Modo: "bandeja", ActorRef: "actor:rrhh", PerfilRef: "perfil:lector",
	}
	rGET, err := RecursoEntregaPeticionCentro(get)
	if err != nil || len(rGET.Ambitos) != 1 ||
		rGET.Ambitos["organizacion_ref"] != "organizacion:desarrollo:dipgra" {
		t.Fatalf("lector GET: recurso=%+v error=%v", rGET, err)
	}
	post := ports.MaterialEntregaPeticionCentro{
		Modo: "preparar", ActorRef: "actor:rrhh", PerfilRef: "perfil:entrega",
		PeticionRef: "peticion:centro:001", CentroRef: "centro-520",
		CategoriaRef: "categoria:desarrollo:c2", VersionEsperada: 2,
		ClaveAltaCandidata: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		AmbitoAltaHMAC:     "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" + strings.Repeat("a", 64),
	}
	rPOST, err := RecursoEntregaPeticionCentro(post)
	if err != nil || len(rPOST.Ambitos) != 3 ||
		rPOST.Ambitos["centro_ref"] != post.CentroRef ||
		rPOST.Ambitos["categoria_ref"] != post.CategoriaRef ||
		rPOST.Ambitos["organizacion_ref"] != "organizacion:desarrollo:dipgra" {
		t.Fatalf("POST: recurso=%+v error=%v", rPOST, err)
	}
	h, err := rPOST.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	post.CategoriaRef = "categoria:otra"
	rOtro, err := RecursoEntregaPeticionCentro(post)
	if err != nil {
		t.Fatal(err)
	}
	hOtro, err := rOtro.HuellaContextoAutorizacionSHA256()
	if err != nil || h == hOtro {
		t.Fatal("cambiar categoría no cambió la huella del contexto")
	}
}
