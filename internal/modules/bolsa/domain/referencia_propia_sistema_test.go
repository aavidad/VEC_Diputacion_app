package domain

import (
	"testing"
	"time"
)

// Huella hallada en el recorrido: sus cifras "31969244d" casan con el patron
// de DNI aunque la referencia la genera el propio sistema.
const llamamientoRefHuellaConFormaDNI = "llamamiento:823ae25fabcdefabcdefabcdefabcdefabcdefabcdefabcda31969244d148227"

func TestReferenciaPropiaSistemaNoSeConfundeConDNI(t *testing.T) {
	if len(llamamientoRefHuellaConFormaDNI) != len("llamamiento:")+64 {
		t.Fatalf("la huella de prueba debe tener 64 caracteres")
	}
	if !patronDocumentoIdentidadEnReferencia.MatchString(llamamientoRefHuellaConFormaDNI) {
		t.Fatalf("la prueba necesita una huella que case por azar con el patron de DNI")
	}
	validas := []string{
		llamamientoRefHuellaConFormaDNI,
		"participacion:bolsa:823ae25fabcdefabcdefabcdefabcdefabcdefabcdefabcda31969244d148227",
	}
	for _, valor := range validas {
		if !ReferenciaPropiaSistema(valor) || !referenciaLlamamientoOpacaValida(valor) {
			t.Errorf("referencia propia rechazada: %s", valor)
		}
	}
	// Fuera de la forma exacta el filtro de documentos sigue vigente.
	invalidas := []string{
		"llamamiento:12345678Z",
		"llamamiento:12345678Zabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcde",
		"Llamamiento:823ae25fabcdefabcdefabcdefabcdefabcdefabcdefabcda31969244d148227",
		"llamamiento:823AE25Fabcdefabcdefabcdefabcdefabcdefabcdefabcda31969244d148227",
		"llamamiento:x:12345678Z:823ae25fabcdefabcdefabcdefabcdefabcdefabcdefabcda31969244d148227",
		// Los UUID no entran en la exención general.
		"recibo:12345678-abcd-4abc-8abc-123456789abc",
		"dni:823ae25fabcdefabcdefabcdefabcdefabcdefabcdefabcda31969244d148227",
	}
	for _, valor := range invalidas {
		if referenciaLlamamientoOpacaValida(valor) {
			t.Errorf("referencia con posible documento aceptada: %s", valor)
		}
	}
}

func TestContactoParticipacionAceptaLlamamientoConHuellaDelSistema(t *testing.T) {
	contacto := ContactoParticipacion{ContactoRef: "contacto:01234567", BolsaRef: "bolsa:01234567", ParticipacionRef: "participacion:01234567",
		LlamamientoRef: llamamientoRefHuellaConFormaDNI, Canal: "telefono", Actor: "per_0123456789abcdefghijkl", Resultado: "contactado",
		Anotacion: "Se explicó la oferta y queda pendiente de respuesta", Instante: time.Date(2026, 9, 21, 20, 0, 0, 0, time.UTC)}
	if err := contacto.Validar(); err != nil {
		t.Fatalf("contacto con llamamiento del sistema rechazado: %v", err)
	}
}
