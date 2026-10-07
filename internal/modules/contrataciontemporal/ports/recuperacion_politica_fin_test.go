package ports

import (
	"bytes"
	"testing"
)

func TestAmbitoPoliticaFinAnalisisConservaCanonPublicado(t *testing.T) {
	solicitud, _ := capacidadArtefactoAnalisisPrueba(t)
	const identificadorOperacion = "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b"
	const actor = "persona:tecnica-rrhh-sintetica-001"
	const perfil = "perfil:tecnica-rrhh-sintetica-001"
	nueva, err := AmbitoPoliticaFinAnalisis(identificadorOperacion, solicitud.OrganizacionRef, solicitud.ExpedienteRef, actor, perfil)
	if err != nil {
		t.Fatal(err)
	}
	anterior, err := NuevasPreimagenesConsultaOperacionAnalisis(DatosPreimagenesConsultaOperacionAnalisis{
		ClaveIdempotencia: identificadorOperacion, Operacion: OperacionRegistrarAnalisis,
		OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef,
		VersionExpediente: solicitud.VersionExpediente, ActorRef: actor, PerfilRef: perfil,
		ArtefactoRef: solicitud.ArtefactoRef, DatosFuncionales: solicitud.DatosFuncionales,
	})
	if err != nil {
		t.Fatal(err)
	}
	ambitoNuevo, err := nueva.BytesAmbito()
	if err != nil {
		t.Fatal(err)
	}
	ambitoAnterior, err := anterior.BytesAmbito()
	if err != nil || !bytes.Equal(ambitoNuevo, ambitoAnterior) {
		t.Fatal("la búsqueda histórica debe usar exactamente el ámbito HMAC original")
	}
}
