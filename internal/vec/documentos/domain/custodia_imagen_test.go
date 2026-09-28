package domain

import "testing"

func TestCustodiaImagenReplayYColision(t *testing.T) {
	base := IdentidadCustodiaImagen{PersonaRef: "per_1234567890123456", ClaveOperacion: "operacion-1234567890", HuellaPeticion: hexPrueba('a'), OriginalSHA256: hexPrueba('b'), ContenidoSHA256: hexPrueba('c')}
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	replay := base
	replay.DocumentoRef = "doc_1234567890123456"
	if !base.MismaPeticion(replay) {
		t.Fatal("la referencia asignada no altera la identidad de la petición")
	}
	replay.ContenidoSHA256 = hexPrueba('d')
	if base.MismaPeticion(replay) {
		t.Fatal("misma clave con otros bytes debe rechazar")
	}
	replay = base
	replay.HuellaPeticion = hexPrueba('e')
	if base.MismaPeticion(replay) {
		t.Fatal("misma clave con otra petición debe rechazar")
	}
}
func TestCustodiaImagenEstadosTrasCaida(t *testing.T) {
	if !EstadoImagenReservada.PuedePasarA(EstadoImagenCuarentena) || !EstadoImagenCuarentena.PuedePasarA(EstadoImagenAdmitida) || !EstadoImagenAdmitida.PuedePasarA(EstadoImagenConfirmada) {
		t.Fatal("se rompió la secuencia de reconciliación")
	}
	for _, estado := range []EstadoCustodiaImagen{EstadoImagenReservada, EstadoImagenCuarentena, EstadoImagenAdmitida, EstadoImagenConfirmada} {
		if estado.PuedePasarA(estado) {
			t.Fatalf("replay no es transición: %s", estado)
		}
		if estado == EstadoImagenConfirmada && estado.PuedePasarA(EstadoImagenAdmitida) {
			t.Fatal("confirmación no puede volver a cuarentena")
		}
	}
}
func hexPrueba(r rune) string {
	b := make([]rune, 64)
	for i := range b {
		b[i] = r
	}
	return string(b)
}
