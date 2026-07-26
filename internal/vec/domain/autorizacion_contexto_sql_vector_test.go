package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

const (
	preimagenContextoAutorizacionSQLVector = `{"ambitos":{"orden_a":"primero","orden_z":"segundo"},"atributos":{"comillas_barra":"dijo \"sí\" \\ fin","html":"\u0026\u003c\u003e","separadores":"línea\u2028párrafo\u2029fin","utf8_nfc":"áéíóú ñ 😀"}}`
	huellaContextoAutorizacionSQLVector    = "5dd8dc79912e15e6540f4fdf03b88b1783182188f64ecd74eb0f13141cb2f603"
)

func recursoContextoAutorizacionSQLVector() RecursoAutorizable {
	return RecursoAutorizable{
		Referencia: "recurso:vector:canonico",
		ModuloID:   "contratacion_temporal",
		Tipo:       "vector_prueba",
		Ambitos: map[string]string{
			"orden_z": "segundo",
			"orden_a": "primero",
		},
		Atributos: map[string]string{
			"utf8_nfc":       "áéíóú ñ 😀",
			"separadores":    "línea\u2028párrafo\u2029fin",
			"html":           "&<>",
			"comillas_barra": `dijo "sí" \ fin`,
		},
	}
}

func TestVectorSQLHuellaContextoAutorizacion(t *testing.T) {
	recurso := recursoContextoAutorizacionSQLVector()
	if err := recurso.Validar(); err != nil {
		t.Fatalf("recurso del vector inválido: %v", err)
	}
	preimagen, err := json.Marshal(contextoRecursoAutorizacionCanonico{
		Ambitos:   clonarMapaAutorizacion(recurso.Ambitos),
		Atributos: clonarMapaAutorizacion(recurso.Atributos),
	})
	if err != nil {
		t.Fatalf("serializar preimagen: %v", err)
	}
	if string(preimagen) != preimagenContextoAutorizacionSQLVector {
		t.Fatalf("preimagen divergente:\n%s", preimagen)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("calcular huella: %v", err)
	}
	if huella != huellaContextoAutorizacionSQLVector {
		t.Fatalf("huella compartida pendiente/divergente: %s", huella)
	}

	reordenado := RecursoAutorizable{
		Referencia: recurso.Referencia,
		ModuloID:   recurso.ModuloID,
		Tipo:       recurso.Tipo,
		Ambitos: map[string]string{
			"orden_a": "primero",
			"orden_z": "segundo",
		},
		Atributos: map[string]string{
			"comillas_barra": `dijo "sí" \ fin`,
			"html":           "&<>",
			"separadores":    "línea\u2028párrafo\u2029fin",
			"utf8_nfc":       "áéíóú ñ 😀",
		},
	}
	huellaReordenada, err := reordenado.HuellaContextoAutorizacionSHA256()
	if err != nil || huellaReordenada != huella {
		t.Fatalf("el orden de inserción alteró la huella: %s, %v", huellaReordenada, err)
	}

	noNormalizado := recursoContextoAutorizacionSQLVector()
	noNormalizado.Atributos["utf8_nfc"] =
		strings.Replace(noNormalizado.Atributos["utf8_nfc"], "á", "a\u0301", 1)
	huellaNoNormalizada, err := noNormalizado.HuellaContextoAutorizacionSHA256()
	if err != nil || huellaNoNormalizada == huella {
		t.Fatalf("se normalizó Unicode de forma implícita: %s, %v", huellaNoNormalizada, err)
	}
}
