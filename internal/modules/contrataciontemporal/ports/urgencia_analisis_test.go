package ports

import (
	"bytes"
	"strings"
	"testing"
)

// La urgencia declarada (CT-000125) liga la identidad semántica de la clave,
// no su ámbito ni la huella del artefacto; sin urgencia el canon no cambia.
func TestIdempotenciaAnalisisLigaUrgencia(t *testing.T) {
	solicitud, capacidad := capacidadArtefactoAnalisisPrueba(t)
	operacion := func(observaciones, motivo string) PreimagenesOperacionAnalisis {
		t.Helper()
		con := solicitud
		con.DatosFuncionales.Observaciones = observaciones
		con.DatosFuncionales.MotivoUrgencia = motivo
		artefacto := prepararArtefactoAnalisisPrueba(t, capacidad, con)
		preimagenes, err := NuevasPreimagenesOperacionAnalisis(DatosPreimagenesOperacionAnalisis{
			ClaveIdempotencia: "018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a5b", Operacion: OperacionRegistrarAnalisis,
			ActorRef: "persona:tecnica-rrhh-sintetica-001", PerfilRef: "perfil:tecnica-rrhh-sintetica-001",
			SolicitudArtefacto: con, Artefacto: artefacto,
		})
		if err != nil {
			t.Fatal(err)
		}
		return preimagenes
	}
	semantica := func(p PreimagenesOperacionAnalisis) []byte {
		t.Helper()
		bytesSemantica, err := p.BytesSemantica()
		if err != nil {
			t.Fatal(err)
		}
		return bytesSemantica
	}
	sin := semantica(operacion("", ""))
	urgente := semantica(operacion("", "Cierre del servicio"))
	otraUrgencia := semantica(operacion("", "Temporal de nieve"))
	if !bytes.HasPrefix(urgente, sin) || bytes.Equal(urgente, sin) || bytes.Equal(urgente, otraUrgencia) {
		t.Fatal("la urgencia debe añadirse al canon y distinguir motivos")
	}
	// Una observación con el mismo texto que el motivo no se confunde con él.
	if bytes.Equal(semantica(operacion("Cierre del servicio", "")), urgente) {
		t.Fatal("observaciones y urgencia se confunden en el canon")
	}
	ambitoSin, _ := operacion("", "").BytesAmbito()
	ambitoCon, _ := operacion("", "Cierre del servicio").BytesAmbito()
	if !bytes.Equal(ambitoSin, ambitoCon) {
		t.Fatal("la urgencia no debe cambiar el ámbito de la clave")
	}
}

func TestMotivoUrgenciaAnalisisValido(t *testing.T) {
	for _, valido := range []string{"", "Cierre del servicio de ayuda a domicilio", strings.Repeat("x", 1000)} {
		if !MotivoUrgenciaAnalisisValido(valido) {
			t.Errorf("%q rechazado", valido)
		}
	}
	for _, invalido := range []string{" relleno ", strings.Repeat("x", 1001), "control\u0000", "salto\r"} {
		if MotivoUrgenciaAnalisisValido(invalido) {
			t.Errorf("%q admitido", invalido)
		}
	}
}
