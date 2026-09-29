package domain

import "testing"

func TestEleccionImagenSoloCodigosCerrados(t *testing.T) {
	validas := []EleccionImagen{
		{Modo: ModoImagenIniciales, Paleta: "azul"},
		{Modo: ModoImagenIcono, Paleta: "verde", Icono: "hoja"},
		{Modo: ModoImagenFoto, Paleta: "gris"},
	}
	for _, e := range validas {
		if e.ValidarCodigos() != nil || CatalogoBaseImagen().ValidarEleccion(e) != nil {
			t.Fatalf("elección válida rechazada: %+v", e)
		}
	}
	for _, e := range []EleccionImagen{
		{Modo: "svg", Paleta: "azul"},
		{Modo: ModoImagenIniciales, Paleta: "#ff0000"},
		{Modo: ModoImagenIniciales, Paleta: "azul", Icono: "sol"},
		{Modo: ModoImagenIcono, Paleta: "azul"},
		{Modo: ModoImagenIcono, Paleta: "azul", Icono: "../../etc/passwd"},
		{Modo: ModoImagenFoto, Paleta: "azul", Icono: "sol"},
	} {
		if e.ValidarCodigos() == nil {
			t.Fatalf("elección fuera del vocabulario aceptada: %+v", e)
		}
	}
}

func TestCatalogoImagenReduceNuncaAmplia(t *testing.T) {
	c := CatalogoBaseImagen()
	if c.Validar() != nil {
		t.Fatal("catálogo base inválido")
	}
	reducido := c.Clonar()
	reducido.Iconos = reducido.Iconos[:2]
	if reducido.Validar() != nil || reducido.ValidarEleccion(EleccionImagen{Modo: ModoImagenIcono, Paleta: "azul", Icono: "montana"}) == nil {
		t.Fatal("un catálogo reducido debe rechazar lo que no publica")
	}
	ampliado := c.Clonar()
	ampliado.Paletas = append(ampliado.Paletas, OpcionPreferencia{Codigo: "fucsia", NombreKey: "ui.usuarios.imagen.paleta.fucsia"})
	if ampliado.Validar() == nil {
		t.Fatal("el catálogo no puede ampliar el vocabulario")
	}
	foto := c.Clonar()
	foto.Predeterminada = EleccionImagen{Modo: ModoImagenFoto, Paleta: "azul"}
	if foto.Validar() == nil {
		t.Fatal("la imagen predeterminada no puede ser una foto")
	}
	if len(c.Clonar().Paletas) != len(c.Paletas) || &c.Clonar().Paletas[0] == &c.Paletas[0] {
		t.Fatal("Clonar comparte memoria")
	}
}
