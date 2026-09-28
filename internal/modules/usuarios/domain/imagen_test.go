package domain

import "testing"

func TestCatalogoImagenCerradoYContrasteAA(t *testing.T) {
	c := CatalogoBaseImagen()
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, p := range c.Paletas {
		if !ContrasteAA(p.Fondo, p.Texto) {
			t.Fatalf("sin AA: %s", p.Codigo)
		}
	}
	for _, e := range []EleccionImagen{{Modo: ModoIniciales, Paleta: "azul"}, {Modo: ModoIcono, Paleta: "verde", Icono: "hoja"}, {Modo: ModoFoto, Paleta: "gris", DocumentoRef: "doc_0123456789abcdef"}} {
		if err := c.ValidarEleccion(e); err != nil {
			t.Fatalf("elección válida %+v: %v", e, err)
		}
	}
	for _, e := range []EleccionImagen{{Modo: ModoIcono, Paleta: "verde", Icono: "externo"}, {Modo: ModoIniciales, Paleta: "#ffffff"}, {Modo: ModoIniciales, Paleta: "azul", DocumentoRef: "doc_0123456789abcdef"}, {Modo: ModoFoto, Paleta: "azul", DocumentoRef: "/tmp/foto.png"}} {
		if c.ValidarEleccion(e) == nil {
			t.Fatalf("aceptó elección libre %+v", e)
		}
	}
	c.Paletas[0].Fondo = "#ffffff"
	if c.Validar() == nil {
		t.Fatal("aceptó paleta alterada")
	}
}
