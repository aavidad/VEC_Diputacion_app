package domain

import (
	"encoding/json"
	"fmt"
	"testing"
)

const sujetoContactoPrueba = "per_0123456789abcdefghijkl"

func TestContactoUsuarioExigeCorreoEstrictoYSeRedacta(t *testing.T) {
	c, err := NuevoContactoUsuario(sujetoContactoPrueba, "persona@prueba.local", 1)
	if err != nil {
		t.Fatal(err)
	}
	if c.SujetoRef() != sujetoContactoPrueba || c.Version() != 1 {
		t.Fatal("metadatos")
	}
	if got := fmt.Sprintf("%v|%+v|%#v", c, c, c); got != "vec.ContactoUsuario{redactado}|vec.ContactoUsuario{redactado}|vec.ContactoUsuario{redactado}" {
		t.Fatal(got)
	}
	b, err := json.Marshal(c)
	if err != nil || string(b) != `{"redactado":true}` {
		t.Fatalf("%s %v", b, err)
	}
	var abierto string
	if err := c.ConDireccion(func(v string) error { abierto = v; return nil }); err != nil || abierto != "persona@prueba.local" {
		t.Fatal("callback")
	}
	if _, err := NuevoContactoUsuario("sujeto:bolsa:012345", "persona@prueba.local", 1); err == nil {
		t.Fatal("sujeto Bolsa aceptado como persona VEC")
	}
	for _, correo := range []string{"", "nombre <persona@prueba.local>", `"con espacio"@prueba.local`, "persona@prueba.local\r\nBcc:x", "á@prueba.local"} {
		if _, err := NuevoContactoUsuario(sujetoContactoPrueba, correo, 1); err == nil {
			t.Fatalf("correo aceptado %q", correo)
		}
	}
}
