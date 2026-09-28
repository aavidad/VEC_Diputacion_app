package domain

import (
	"testing"
	"time"
)

func TestConjuntoCorreosImpidePendienteActivoYDosActivos(t *testing.T) {
	ahora := time.Now().UTC()
	lista := []CorreoPropio{{CorreoRef: "correo:uno", Direccion: "uno@example.org", Estado: CorreoPendiente, Activo: true, CreadoUTC: ahora}}
	if ValidarConjuntoCorreos(lista) == nil {
		t.Fatal("pendiente usado como activo")
	}
	lista[0].Activo = false
	if err := ValidarConjuntoCorreos(lista); err != nil {
		t.Fatal(err)
	}
	lista[0].Estado, lista[0].Activo, lista[0].VerificadoUTC = CorreoVerificado, true, &ahora
	lista = append(lista, CorreoPropio{CorreoRef: "correo:dos", Direccion: "dos@example.org", Estado: CorreoVerificado, Activo: true, CreadoUTC: ahora, VerificadoUTC: &ahora})
	if ValidarConjuntoCorreos(lista) == nil {
		t.Fatal("dos correos activos")
	}
	lista[1].Activo = false
	lista[1].Direccion = "UNO@example.org"
	if ValidarConjuntoCorreos(lista) == nil {
		t.Fatal("duplicado no detectado")
	}
}

func TestDireccionCorreoRechazaPresentacionYCabeceras(t *testing.T) {
	for _, direccion := range []string{"Nombre <a@example.org>", "a@example.org\r\nBcc: x@example.org", "a@localhost", " a@example.org"} {
		if DireccionCorreoValida(direccion) {
			t.Fatalf("direccion peligrosa aceptada: %q", direccion)
		}
	}
	if !DireccionCorreoValida("persona@example.org") {
		t.Fatal("direccion simple rechazada")
	}
}
