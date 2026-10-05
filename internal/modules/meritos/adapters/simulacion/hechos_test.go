package simulacion

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
)

func TestLectorAislaFuenteYProyecciones(t *testing.T) {
	b, err := os.ReadFile("../../../../../data/ejemplos/meritos/preparacion.json")
	if err != nil {
		t.Fatal(err)
	}
	var p domain.Paquete
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	l, err := NuevoLectorHechos(p)
	if err != nil {
		t.Fatal(err)
	}
	selector := ports.SelectorHechosPreparacion{PersonaRef: "persona:ensayo:01", FechaCorte: p.FechaCorte,
		Hechos: []ports.ReferenciaHechoPreparacion{{Referencia: "hecho:ensayo:02", VersionEsperada: 1}}}
	// La fuente original y cada resultado tienen propiedad separada.
	*p.Hechos[2].Horas = 999
	p.Hechos[2].Evidencias[0].ID = "documento:otro"
	primera, err := l.LeerHechosSinteticos(context.Background(), selector)
	if err != nil {
		t.Fatal(err)
	}
	if *primera.Hechos[0].Horas != 20 || primera.Hechos[0].Evidencias[0].ID != "documento:ensayo:curso" {
		t.Fatal("alias de la fuente")
	}
	*primera.Hechos[0].Horas = 300
	primera.Hechos[0].Evidencias[0].ID = "documento:modificado"
	primera.Hechos[0].Pendientes[0] = "modificado"
	segunda, err := l.LeerHechosSinteticos(context.Background(), selector)
	if err != nil || *segunda.Hechos[0].Horas != 20 || segunda.Hechos[0].Evidencias[0].ID != "documento:ensayo:curso" || segunda.Hechos[0].Pendientes[0] != "meritos.pendiente.persona" {
		t.Fatal("alias entre lecturas")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if r, err := l.LeerHechosSinteticos(ctx, selector); err != context.Canceled || r.Hechos != nil {
		t.Fatal("cancelación devuelve datos")
	}
}
