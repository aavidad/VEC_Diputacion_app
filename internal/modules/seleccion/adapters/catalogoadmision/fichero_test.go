package catalogoadmision

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const dirEjemplo = "../../../../../data/catalogos/seleccion"

func TestEjemploCargaYTieneTextoPorIdioma(t *testing.T) {
	f, err := Cargar(dirEjemplo, "admision_ejemplo.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := f.CatalogoAdmision(context.Background(), "seleccion-admision-ejemplo", "ejemplo-1")
	if err != nil || !c.PaqueteEjemplo {
		t.Fatal(err)
	}
	for _, idioma := range []string{"es", "en"} {
		raw, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/motivos-" + c.Referencia + ".json")
		var textos map[string]string
		if err != nil || json.Unmarshal(raw, &textos) != nil || len(textos) != len(c.Motivos) {
			t.Fatalf("%s: textos incompletos", idioma)
		}
		for _, m := range c.Motivos {
			if strings.TrimSpace(textos[m.Codigo]) == "" {
				t.Fatalf("%s: falta %s", idioma, m.Codigo)
			}
		}
	}
	c.Motivos[0].Subsanable = !c.Motivos[0].Subsanable
	if otra, _ := f.CatalogoAdmision(context.Background(), "seleccion-admision-ejemplo", "ejemplo-1"); otra.Motivos[0].Subsanable == c.Motivos[0].Subsanable {
		t.Fatal("el catálogo devuelto comparte memoria")
	}
	if _, err := f.CatalogoAdmision(context.Background(), "seleccion-admision-ejemplo", "ejemplo-9"); err != ErrNoEncontrado {
		t.Fatal(err)
	}
}

func TestLeerRechazaConfiguracionAmbiguaOInvalida(t *testing.T) {
	valido := `{"referencia":"r","version":"1","plazo_subsanacion":{"unidad":"dias_habiles","cantidad":10},"motivos":[{"codigo":"m","subsanable":true}]}`
	casos := []string{
		`{"catalogos":[` + valido + `,` + valido + `]}`,
		`{"catalogos":[` + valido + `],"extra":1}`,
		`{"catalogos":[` + valido + `]}{}`,
		`{"catalogos":[]}`,
		`{"catalogos":[` + strings.Replace(valido, `"cantidad":10`, `"cantidad":0`, 1) + `]}`,
		`{"catalogos":[` + strings.Replace(valido, `"version":"1"`, `"version":"1","paquete_ejemplo":true`, 1) + `]}`,
	}
	if _, err := Leer([]byte(`{"catalogos":[` + valido + `]}`)); err != nil {
		t.Fatal(err)
	}
	for i, c := range casos {
		if _, err := Leer([]byte(c)); err != ErrConfiguracion {
			t.Errorf("caso %d: %v", i, err)
		}
	}
	if _, err := Cargar(dirEjemplo, "../../../../go.mod"); err != ErrConfiguracion {
		t.Fatal("salió de la raíz configurada")
	}
}
