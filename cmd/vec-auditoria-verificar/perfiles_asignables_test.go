package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIPerfilesAsignablesClavesExactas(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/perfiles_asignables_ad196.json")
	if err != nil {
		t.Fatal(err)
	}
	var o map[string]json.RawMessage
	if err := json.Unmarshal(b, &o); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "checkpoint.json")
	if err := os.WriteFile(p, o["manifiesto"], 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"--checkpoint", p, "--max-bytes", "16384", "--max-registros", "4"}
	for caso, want := range map[string]int{"valido": 0, "dato_extra": 2, "doble": 2, "nulo": 2, "alterado": 1} {
		t.Run(caso, func(t *testing.T) {
			entrada := string(b)
			switch caso {
			case "dato_extra":
				entrada = strings.Replace(entrada, `"perfiles_numero":`, `"plan":"SECRET","perfiles_numero":`, 1)
			case "doble":
				entrada = strings.Replace(entrada, `"perfiles_numero":`, `"perfiles_numero":"SECRET","perfiles_numero":`, 1)
			case "nulo":
				entrada = strings.Replace(entrada, `"perfiles_numero": "3"`, `"perfiles_numero": null`, 1)
			case "alterado":
				entrada = strings.Replace(entrada, `"perfiles_numero": "3"`, `"perfiles_numero": "4"`, 1)
			}
			if caso != "valido" && entrada == string(b) {
				t.Fatal("el caso no altera la entrada")
			}
			var out bytes.Buffer
			if code := ejecutar(args, strings.NewReader(entrada), &out); code != want || strings.Contains(out.String(), "SECRET") {
				t.Fatalf("caso %s salida %d esperada %d: %s", caso, code, want, out.String())
			}
		})
	}
}
