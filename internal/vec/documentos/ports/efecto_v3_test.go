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

// Vectores fijos compartidos con pruebas_sql/frontera_000004.sql, que los
// recalcula con vec_documentos.huella_efecto_v1: una preimagen de lista real y
// otra con texto no ASCII (UTF-8 de varios bytes).
func TestHuellaEfectoV3VectoresCompartidosConSQL(t *testing.T) {
	casos := []struct{ preimagen, huella string }{
		{`{"accion":"documentos.expediente.listar","expediente_ref":"exp:00000000-0000-4000-8000-000000000001","cursor":"","limite":1}`,
			"c0fd576c1fb94f16eceb0da654d6b5bf53013383f6bfd5ad3730730fd05b440a"},
		{`{"motivo":"año"}`, "eed395327dbdfbda3586ec4ceaeae44086c90f7dceed7bd78c90eb7aa5898178"},
	}
	for _, c := range casos {
		if got := HuellaEfectoV3([]byte(c.preimagen)); got != c.huella {
			t.Fatalf("huella de %s: %s, esperado %s", c.preimagen, got, c.huella)
		}
	}
}
