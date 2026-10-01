package restauracioncopias

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func propuesta() Propuesta {
	n := time.Date(2026, 10, 1, 18, 0, 0, 0, time.UTC)
	h := strings.Repeat("a", 64)
	return Propuesta{1, "propuesta:1", "copia:1", h, "destino:1", h, "motivo:1", "ventana:1", n, n.Add(time.Hour), "politica:1", h, "persona:1", n, n.Add(time.Hour), true, "operativo"}
}
func TestDobleControlYSello(t *testing.T) {
	s, err := Sellar(propuesta())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		fn   func(Sellada) error
		want error
	}{
		{"misma", func(s Sellada) error { _, e := s.Revisar("persona:1", s.Propuesta.Creada); return e }, ErrMismaPersona},
		{"alterada", func(s Sellada) error { s.Propuesta.DestinoRef = "destino:2"; return s.Comprobar(s.Propuesta.Creada) }, ErrAlterada},
		{"expirada", func(s Sellada) error { return s.Comprobar(s.Propuesta.Caduca) }, ErrCaducada},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if e := tc.fn(s); !errors.Is(e, tc.want) {
				t.Fatalf("%v", e)
			}
		})
	}
	r, e := s.Revisar("persona:2", s.Propuesta.Creada)
	if e != nil || s.ComprobarRevision(r, s.Propuesta.Creada) != nil {
		t.Fatal(e)
	}
	r.PropuestaSHA256 = strings.Repeat("b", 64)
	if !errors.Is(s.ComprobarRevision(r, s.Propuesta.Creada), ErrAlterada) {
		t.Fatal("sello")
	}
}
func TestConfiguracionDefaultYSintetica(t *testing.T) {
	f := false
	for _, c := range []Configuracion{{Entorno: "operativo"}, {Entorno: "sintetico_offline"}} {
		v, e := c.ExigirDobleControl()
		if e != nil || !v {
			t.Fatal(v, e)
		}
	}
	if _, e := (Configuracion{&f, "operativo"}).ExigirDobleControl(); e == nil {
		t.Fatal("operativo")
	}
	if v, e := (Configuracion{&f, "sintetico_offline"}).ExigirDobleControl(); v || e != nil {
		t.Fatal(v, e)
	}
}
