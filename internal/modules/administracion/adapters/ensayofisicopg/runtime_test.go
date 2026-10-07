package ensayofisicopg

import "testing"

func TestInspeccionNormalizacionNoNuevosPrivilegios(t *testing.T) {
	for _, c := range []struct {
		opciones []string
		esperado bool
	}{
		{[]string{"no-new-privileges"}, true},
		{[]string{"no-new-privileges:true"}, true},
		{[]string{"no-new-privileges:false"}, false},
		{[]string{"no-new-privileges:true", "no-new-privileges:false"}, false},
		{[]string{"no-new-privileges:1"}, false},
		{nil, false},
	} {
		if sinNuevosPrivilegios(c.opciones) != c.esperado {
			t.Fatalf("opción Docker no reconocida: %v", c.opciones)
		}
	}
}
