package calculomeritos

import (
	"encoding/json"
	"testing"
)

func TestDiferenciaV2ExponeCadaCambioQueAfectaBaremo(t *testing.T) {
	casos := []struct {
		nombre, ambito, campo string
		cambiar               func(*Conjunto)
	}{
		{"categoria", "conjunto", "categoria_ref", func(c *Conjunto) { c.CategoriaRef = "cat_00000000000000000000000000000042" }},
		{"plaza", "conjunto", "plaza_ref", func(c *Conjunto) { c.PlazaRef = "pla_00000000000000000000000000000043" }},
		{"corte", "conjunto", "fecha_corte_inclusiva", func(c *Conjunto) { c.FechaCorteInclusiva = *fecha(t, 2024, 10, 9) }},
		{"relacion", "regla", "conceptos_relacionados", func(c *Conjunto) {
			c.Reglas[0].ConceptosRelacionados = []string{"con_00000000000000000000000000000039"}
		}},
		{"ventana", "regla", "temporal", func(c *Conjunto) { c.Reglas[0].Temporal.DesdeBonificacion = *fecha(t, 2018, 10, 9) }},
		{"efecto", "regla", "temporal", func(c *Conjunto) { c.Reglas[0].Temporal.EfectoAnterior = "excluir" }},
		{"tarifa", "regla", "temporal", func(c *Conjunto) { c.Reglas[0].Temporal.PuntosRecientes = *puntos(t, 90_000) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			anterior, _ := psicologiaV2(t)
			material, err := json.Marshal(anterior)
			if err != nil {
				t.Fatal(err)
			}
			var nueva Conjunto
			if err := json.Unmarshal(material, &nueva); err != nil {
				t.Fatal(err)
			}
			nueva.Version++
			caso.cambiar(&nueva)
			diff, err := CompararConjuntos(anterior, nueva)
			if err != nil {
				t.Fatal(err)
			}
			if diff.Esquema != "vec.bolsa.diferencia_reglas_meritos.v2" || len(diff.Cambios) != 1 || diff.Cambios[0].Ambito != caso.ambito || diff.Cambios[0].Campo != caso.campo {
				t.Fatalf("cambio oculto o ambiguo: %+v", diff.Cambios)
			}
		})
	}
}
