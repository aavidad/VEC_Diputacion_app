package application

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// Petición RRHH p.4: el registro de contacto declara qué campos cambiaron
// frente a la versión anterior, sin valores.
func TestServicioContactoDeclaraCamposCambiados(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 1, 0, 0, 0, time.UTC)
	repo := &repositorioDatosContactoPrueba{}
	servicio, _ := servicioDatosContactoPrueba(t, ahora, repo, &cifradorDatosContactoPrueba{}, contextoSituacionPrueba{}, true)
	if _, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "traza-0001", datosDatosContactoPrueba())); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repo.ultimo.CamposCambiados, []string{"correo", "telefono_1", "telefono_2"}) {
		t.Fatalf("primera versión: %v", repo.ultimo.CamposCambiados)
	}
	otros := datosDatosContactoPrueba()
	otros.Telefono2 = "611111111"
	if _, err := servicio.Registrar(context.Background(), solicitudDatosContactoPrueba(t, ahora, "traza-0002", otros)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(repo.ultimo.CamposCambiados, []string{"telefono_2"}) {
		t.Fatalf("segunda versión: %v", repo.ultimo.CamposCambiados)
	}
}
