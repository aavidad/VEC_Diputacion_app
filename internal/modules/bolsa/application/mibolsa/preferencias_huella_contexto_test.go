package mibolsa

import (
	"strings"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// B143 recalcula en SQL la huella del contexto fijo de Ordenar. Este vector
// es el mismo de la prueba b143_huella_contexto_orden_bolsas.sql.
func TestHuellaContextoOrdenarCoincideConB143(t *testing.T) {
	candidato := "can_" + strings.Repeat("a", 43)
	recurso := dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:" + candidato,
		ModuloID: bolsa.ModuloMiBolsa, Tipo: bolsa.TipoRecursoMiBolsa,
		Ambitos: map[string]string{"candidato_ref": candidato}, Atributos: map[string]string{"propiedad": "candidato"}}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huella != "b0084709beee455ac6927bc55e23d95e7e1e727a80b2f798db16f0ba8898b19c" {
		t.Fatalf("huella de Ordenar distinta de B143: %s, %v", huella, err)
	}
	recurso.Atributos = map[string]string{"propiedad": "candidato", "tipo_preferencia": "modalidades",
		"material_sha256": strings.Repeat("c", 64)}
	modalidades, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || modalidades == huella {
		t.Fatalf("la huella de modalidades debe diferir de la de Ordenar: %v", err)
	}
}
