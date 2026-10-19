package inscripcion

import (
	"strings"
	"testing"
)

// El material conjunto es canónico: el orden de las categorías no cambia sus
// bytes y sólo se reconstruye desde esos mismos bytes.
func TestMaterialPresentacionConjuntaCanonico(t *testing.T) {
	a := PresentacionConjunta{ConvocatoriaRef: "cv1_prueba_v1", CategoriasRefs: []string{"cat_beta", "cat_alpha"},
		CatalogoVersion: 1, ClaveIdempotencia: "clave_inscripcion_prueba_01",
		Declaraciones: []Declaracion{{RequisitoCodigo: "req_b"}, {RequisitoCodigo: "req_a"}}}
	b := a
	b.CategoriasRefs = []string{"cat_alpha", "cat_beta"}
	b.Declaraciones = []Declaracion{{RequisitoCodigo: "req_a"}, {RequisitoCodigo: "req_b"}}
	ma, ha, err := MaterialPresentacionConjunta(a)
	if err != nil {
		t.Fatal(err)
	}
	mb, hb, err := MaterialPresentacionConjunta(b)
	if err != nil || string(ma) != string(mb) || ha != hb ||
		!strings.Contains(string(ma), `"categorias_refs":["cat_alpha","cat_beta"]`) {
		t.Fatalf("material no canónico: %s / %s", ma, mb)
	}
	if p, ok := PresentacionConjuntaDesdeMaterial(ma); !ok || strings.Join(p.Categorias(), ",") != "cat_alpha,cat_beta" {
		t.Fatal("material conjunto no reconstruido")
	}
	if _, ok := PresentacionConjuntaDesdeMaterial([]byte(strings.Replace(string(ma), `"cat_alpha","cat_beta"`, `"cat_beta","cat_alpha"`, 1))); ok {
		t.Fatal("material no canónico aceptado")
	}
	for _, mal := range [][]string{nil, {"cat_alpha", "cat_alpha"}, {"cat_alpha", ""}} {
		c := a
		c.CategoriasRefs = mal
		if c.Validar() == nil {
			t.Fatalf("categorías inválidas aceptadas: %v", mal)
		}
	}
}

// El recibo debe corresponder, categoría a categoría y en orden, a las
// solicitudes de la persona; otro orden o persona se rechaza.
func TestReciboConjuntoValidarCategorias(t *testing.T) {
	p := PresentacionConjunta{ConvocatoriaRef: "cv1_prueba_v1", CategoriasRefs: []string{"cat_beta", "cat_alpha"},
		CatalogoVersion: 1, ClaveIdempotencia: "clave_inscripcion_prueba_01"}
	recibo := ReciboConjunto{PresentacionRef: "presentacion_inscripcion_0001", ReciboRef: "recibo_inscripcion_conjunta_0001",
		ConvocatoriaRef: p.ConvocatoriaRef, BasesRef: "bases.prueba", Estado: EstadoPendiente, Version: 1}
	recibo.RegistradaEn = recibo.RegistradaEn.AddDate(2026, 0, 0)
	for _, categoria := range p.Categorias() {
		ref, err := ReferenciaSolicitud("per_prueba", p.PorCategoria(categoria))
		if err != nil {
			t.Fatal(err)
		}
		recibo.Categorias = append(recibo.Categorias, CategoriaReciboConjunto{CategoriaRef: categoria, Categoria: "Etiqueta",
			SolicitudRef: ref, ReciboRef: "recibo_inscripcion_0001", DeclaracionRef: "declaracion_inscripcion_0001"})
	}
	if err := recibo.Validar("per_prueba", p); err != nil {
		t.Fatal(err)
	}
	if recibo.Validar("per_ajena", p) == nil {
		t.Fatal("recibo de otra persona aceptado")
	}
	recibo.Categorias[0], recibo.Categorias[1] = recibo.Categorias[1], recibo.Categorias[0]
	if recibo.Validar("per_prueba", p) == nil {
		t.Fatal("recibo desordenado aceptado")
	}
}
