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
	lista[0].Codigo = &CodigoPendiente{VenceUTC: ahora.Add(time.Hour), IntentosRestantes: 5}
	if err := ValidarConjuntoCorreos(lista); err != nil {
		t.Fatal(err)
	}
	lista[0].Codigo.IntentosRestantes = 0
	if ValidarConjuntoCorreos(lista) == nil {
		t.Fatal("código sin intentos presentado como vigente")
	}
	lista[0].Codigo = nil
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
	lista[1].Direccion = "dos@example.org"
	lista[1].Estado = CorreoRetirado
	if ValidarConjuntoCorreos(lista) == nil {
		t.Fatal("retirado presentado al titular")
	}
	lista[1].Estado = CorreoVerificado
	for len(lista) <= MaxCorreosVivos {
		lista = append(lista, CorreoPropio{CorreoRef: "correo:" + string(rune('a'+len(lista))), Direccion: string(rune('a'+len(lista))) + "@example.org", Estado: CorreoPendiente, CreadoUTC: ahora})
	}
	if ValidarConjuntoCorreos(lista) == nil {
		t.Fatal("más direcciones que el máximo")
	}
}

func TestDireccionCorreoRechazaPresentacionYCabeceras(t *testing.T) {
	for _, direccion := range []string{"Nombre <a@example.org>", "a@example.org\r\nBcc: x@example.org", "a@localhost", " a@example.org", "\"a b\"@example.org", "a@example.c", "a..b@example.org", "a@-ejemplo.org", "ñandú@example.org", "a@example.org,b@example.org"} {
		if DireccionCorreoValida(direccion) {
			t.Fatalf("direccion peligrosa aceptada: %q", direccion)
		}
	}
	for _, direccion := range []string{"persona@example.org", "nombre.apellido+vec@dipgra.es", "A_B-c@sub.dominio.example"} {
		if !DireccionCorreoValida(direccion) {
			t.Fatalf("direccion simple rechazada: %q", direccion)
		}
	}
}

func TestNormalizarCodigoCorreo(t *testing.T) {
	for entrada, esperado := range map[string]string{"12345678": "12345678", "1234 5678": "12345678", "1234-5678": "12345678", " 12 34 56 78 ": "12345678"} {
		if codigo, ok := NormalizarCodigoCorreo(entrada); !ok || codigo != esperado {
			t.Fatalf("%q -> %q %v", entrada, codigo, ok)
		}
	}
	for _, entrada := range []string{"", "1234567", "123456789", "1234567a", "１２３４５６７８", "12345678901234567890123456789012345"} {
		if _, ok := NormalizarCodigoCorreo(entrada); ok {
			t.Fatalf("código inválido aceptado: %q", entrada)
		}
	}
}
