package httppersonal

import (
	"encoding/json"
	"strings"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestOfertaPortalProyectaClavesF111(t *testing.T) {
	i := puertosbolsa.InstantaneaMiBolsa{Ofertas: []puertosbolsa.OfertaPortalCandidato{{Datos: dominiobolsa.DatosOferta{ModalidadClave: "interinidad", CausaClave: "sustitucion"}}}}
	jsonNuevo, err := json.Marshal(respuestaOfertas(i))
	if err != nil || !strings.Contains(string(jsonNuevo), `"modalidad_clave":"interinidad"`) || !strings.Contains(string(jsonNuevo), `"causa_clave":"sustitucion"`) {
		t.Fatalf("claves omitidas: %s, %v", jsonNuevo, err)
	}
	i.Ofertas[0].Datos = dominiobolsa.DatosOferta{}
	jsonLegado, err := json.Marshal(respuestaOfertas(i))
	if err != nil || strings.Contains(string(jsonLegado), "modalidad_clave") || strings.Contains(string(jsonLegado), "causa_clave") {
		t.Fatalf("claves inventadas en legado: %s, %v", jsonLegado, err)
	}
}
