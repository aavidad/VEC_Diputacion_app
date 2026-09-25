package ports

import (
	"bytes"
	"testing"
)

// Repetir la misma clave con otras observaciones es otro material: la
// identidad semántica de la clave debe cambiar y el ámbito no. Sin
// observaciones el canon no añade ningún campo, para que las huellas de las
// operaciones ya confirmadas sin observaciones sigan siendo las mismas.
func TestIdempotenciaAnalisisLigaObservaciones(t *testing.T) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	consulta := func(observaciones string) PreimagenesOperacionAnalisis {
		t.Helper()
		datos := solicitud.DatosFuncionales
		datos.Observaciones = observaciones
		preimagenes, err := NuevasPreimagenesConsultaOperacionAnalisis(
			DatosPreimagenesConsultaOperacionAnalisis{
				ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
				Operacion:         OperacionRegistrarAnalisis,
				OrganizacionRef:   solicitud.OrganizacionRef,
				ExpedienteRef:     solicitud.ExpedienteRef,
				VersionExpediente: solicitud.VersionExpediente,
				ActorRef:          "persona:tecnica-rrhh-sintetica-001",
				PerfilRef:         "perfil:tecnica-rrhh-sintetica-001",
				ArtefactoRef:      solicitud.ArtefactoRef,
				DatosFuncionales:  datos,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		return preimagenes
	}
	operacion := func(observaciones string) PreimagenesOperacionAnalisis {
		t.Helper()
		conObservaciones := solicitud
		conObservaciones.DatosFuncionales.Observaciones = observaciones
		artefacto := prepararArtefactoAnalisisPrueba(t, capacidad, conObservaciones)
		preimagenes, err := NuevasPreimagenesOperacionAnalisis(
			DatosPreimagenesOperacionAnalisis{
				ClaveIdempotencia:  "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b",
				Operacion:          OperacionRegistrarAnalisis,
				ActorRef:           "persona:tecnica-rrhh-sintetica-001",
				PerfilRef:          "perfil:tecnica-rrhh-sintetica-001",
				SolicitudArtefacto: conObservaciones,
				Artefacto:          artefacto,
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		return preimagenes
	}
	for nombre, construir := range map[string]func(string) PreimagenesOperacionAnalisis{
		"consulta": consulta, "operacion": operacion,
	} {
		sin, primera, segunda := construir(""), construir("Primera versión."), construir("Segunda versión.")
		ambitoSin, _ := sin.BytesAmbito()
		ambitoCon, _ := segunda.BytesAmbito()
		semSin, _ := sin.BytesSemantica()
		semPrimera, _ := primera.BytesSemantica()
		semSegunda, _ := segunda.BytesSemantica()
		if !bytes.Equal(ambitoSin, ambitoCon) {
			t.Fatalf("%s: las observaciones no deben cambiar el ámbito de la clave", nombre)
		}
		if bytes.Equal(semPrimera, semSegunda) {
			t.Fatalf("%s: observaciones distintas producen la misma identidad semántica", nombre)
		}
		if !bytes.HasPrefix(semPrimera, semSin) || bytes.Equal(semPrimera, semSin) {
			t.Fatalf("%s: sin observaciones el canon debe quedar como antes", nombre)
		}
	}
}
