package domain

import "testing"

func TestCustodiaImagenReplayYColision(t *testing.T) {
	base := IdentidadCustodiaImagen{PersonaRef: "per_1234567890123456", PerfilRef: "prf_1234567890123456", Audiencia: "portal_personal_autenticado", Finalidad: "finalidad:usuarios:imagen-propia:v1", VersionEsperada: 3, CatalogoVersionRef: "usuarios-imagen-v1", Paleta: "azul", ClaveOperacion: "operacion-1234567890", HuellaPeticion: hexPrueba('a'), OriginalSHA256: hexPrueba('b'), ContenidoSHA256: hexPrueba('c')}
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
	for _, cambio := range []func(*IdentidadCustodiaImagen){
		func(i *IdentidadCustodiaImagen) { i.VersionEsperada++ },
		func(i *IdentidadCustodiaImagen) { i.CatalogoVersionRef = "usuarios-imagen-v2" },
		func(i *IdentidadCustodiaImagen) { i.Paleta = "verde" },
		func(i *IdentidadCustodiaImagen) { i.PerfilRef = "prf_2222222222222222" },
		func(i *IdentidadCustodiaImagen) { i.Audiencia = "portal_interno_autenticado" },
	} {
		replay = base
		cambio(&replay)
		if base.MismaPeticion(replay) {
			t.Fatal("replay perdió un dato necesario para reconstruir la intención")
		}
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
func TestCustodiaImagenClaveYCatalogoCompatiblesConUsuarios(t *testing.T) {
	id := IdentidadCustodiaImagen{PersonaRef: "per_1234567890123456", PerfilRef: "prf_1234567890123456", Audiencia: "portal_personal_autenticado", Finalidad: "finalidad:usuarios:imagen-propia:v1", CatalogoVersionRef: "usuarios:imagen.v1", Paleta: "azul", ClaveOperacion: "operacion.1234567890", HuellaPeticion: hexPrueba('a'), OriginalSHA256: hexPrueba('b'), ContenidoSHA256: hexPrueba('c')}
	if err := id.Validar(); err != nil {
		t.Fatalf("Usuarios acepta clave/catalogo con ./: %v", err)
	}
	id.DocumentoRef = "doc_1234567890123456"
	id.DocumentoRef += ".ajena"
	if id.Validar() == nil {
		t.Fatal("ampliar claves no debe ampliar referencias opacas")
	}
}
func hexPrueba(r rune) string {
	b := make([]rune, 64)
	for i := range b {
		b[i] = r
	}
	return string(b)
}
