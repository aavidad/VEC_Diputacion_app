package domain

import "testing"

func TestCatalogoCerradoYDefaults(t *testing.T) {
	c := CatalogoBasePreferencias()
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
	if err := c.ValidarValores(c.Predeterminados); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		name   string
		cambia func(*CatalogoPreferencias)
	}{
		{"codigo libre", func(x *CatalogoPreferencias) {
			x.Temas = append(x.Temas, OpcionPreferencia{"css:rojo", "ui.usuarios.preferencias.tema.rojo"})
		}},
		{"destino arbitrario", func(x *CatalogoPreferencias) { x.Inicios[0].Codigo = "https://example.invalid" }},
		{"opcion repetida", func(x *CatalogoPreferencias) { x.Idiomas = append(x.Idiomas, x.Idiomas[0]) }},
		{"clave sin catalogo i18n", func(x *CatalogoPreferencias) { x.Idiomas[0].NombreKey = "<script>" }},
		{"default no ofertado", func(x *CatalogoPreferencias) { x.Filas = []int{50} }},
		{"filas ampliadas", func(x *CatalogoPreferencias) { x.Filas = append(x.Filas, 500) }},
	}
	for _, tc := range casos {
		t.Run(tc.name, func(t *testing.T) {
			x := c.Clonar()
			tc.cambia(&x)
			if x.Validar() == nil {
				t.Fatal("catalogo aceptado")
			}
		})
	}
	v := c.Predeterminados
	v.Idioma = "fr"
	if c.ValidarValores(v) == nil {
		t.Fatal("idioma no ofertado")
	}
	v = c.Predeterminados
	v.Inicio = "admin"
	if c.ValidarValores(v) == nil {
		t.Fatal("ruta no ofertada")
	}
}

func TestCatalogoConfigurableNoAgregaCodigos(t *testing.T) {
	c := CatalogoBasePreferencias()
	c.VersionRef = "usuarios-preferencias-v2"
	c.Inicios = c.Inicios[:2]
	c.Filas = []int{20, 50}
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
	v := c.Predeterminados
	v.Inicio = "bolsas"
	if c.ValidarValores(v) == nil {
		t.Fatal("opcion retirada aceptada")
	}
	v = c.Predeterminados
	v.Filas = 100
	if c.ValidarValores(v) == nil {
		t.Fatal("filas retiradas aceptadas")
	}
}

func TestCatalogoV2TemasCerradosYContrasteIndependiente(t *testing.T) {
	c := CatalogoBasePreferencias()
	c.VersionRef = "usuarios-preferencias-v2"
	for _, tema := range []string{"diputacion_granada", "arena", "salvia", "lavanda", "azul_sereno", "noche_suave"} {
		c.Temas = append(c.Temas, OpcionPreferencia{Codigo: tema, NombreKey: "ui.usuarios.preferencias.tema." + tema})
	}
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
	for _, opcion := range c.Temas {
		v := c.Predeterminados
		v.Tema, v.AltoContraste = opcion.Codigo, true
		if err := c.ValidarValores(v); err != nil {
			t.Fatalf("tema %q con contraste: %v", opcion.Codigo, err)
		}
	}
	v := c.Predeterminados
	v.Tema = "url(https://example.invalid)"
	if c.ValidarValores(v) == nil {
		t.Fatal("tema libre aceptado")
	}
	c.Temas = c.Temas[:len(c.Temas)-1]
	v.Tema = "noche_suave"
	if c.ValidarValores(v) == nil {
		t.Fatal("tema no ofertado aceptado")
	}
}
