package ports

import (
	"strings"
	"testing"
)

func TestHuellaEfectoV3CoincideConRecursoYVectorSQL(t *testing.T) {
	// Mismo vector que pruebas_sql/frontera_000004.sql y huella_efecto_v1.
	const vector = "c2535d333933e853c1cd0ecb0ed11250173705932d435f6f4310bf3cf4f1d614"
	if got := HuellaEfectoV3([]byte("abc")); got != vector {
		t.Fatalf("huella de efecto %s, esperado %s", got, vector)
	}
	for _, accion := range []string{AccionAlta, AccionListar, AccionDescargar, AccionPrepararNotificacion, AccionRegistrarExterno} {
		recurso, err := RecursoV3(accion, "ref:"+strings.Repeat("7", 64), []byte("abc"))
		if err != nil {
			t.Fatalf("%s: %v", accion, err)
		}
		huella, err := recurso.HuellaContextoAutorizacionSHA256()
		if err != nil || huella != vector || recurso.ModuloID != "documentos" {
			t.Fatalf("%s: huella de recurso %s %v", accion, huella, err)
		}
	}
	if _, err := RecursoV3("otra.accion", "ref:"+strings.Repeat("7", 64), []byte("abc")); err == nil {
		t.Fatal("acción desconocida aceptada")
	}
	if _, err := RecursoV3(AccionListar, "no/opaca", []byte("abc")); err == nil {
		t.Fatal("referencia no opaca aceptada")
	}
	if HuellaEfectoV3(nil) != "" || HuellaEfectoV3(make([]byte, 16385)) != "" {
		t.Fatal("preimagen no admisible con huella")
	}
}
