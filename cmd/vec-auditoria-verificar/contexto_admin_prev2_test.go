package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIContextoPreV2NullableCerradoSinFuenteDTO(t *testing.T) {
	b, err := os.ReadFile("../../internal/vec/auditoria/testdata/contexto_admin_prev2.json")
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
	args := []string{"--checkpoint", p, "--max-bytes", "16384", "--max-registros", "3"}
	for _, caso := range []string{"valido", "omitir_nullable", "dato_extra", "doble", "actor_null_success"} {
		t.Run(caso, func(t *testing.T) {
			entrada := string(b)
			want := 2
			switch caso {
			case "valido":
				want = 0
			case "omitir_nullable":
				entrada = strings.Replace(entrada, `"perfil_activo_ref": null,`, "", 1)
			case "dato_extra":
				entrada = strings.Replace(entrada, `"actor_ref":`, `"certificado":"SECRET","actor_ref":`, 1)
			case "doble":
				entrada = strings.Replace(entrada, `"actor_ref":`, `"actor_ref":"SECRET","actor_ref":`, 1)
			case "actor_null_success":
				entrada = strings.Replace(entrada, `"actor_ref": "per_aaaaaaaaaaaaaaaaaaaaaaaa"`, `"actor_ref": null`, 1)
				want = 1
			}
			var out bytes.Buffer
			if code := ejecutar(args, strings.NewReader(entrada), &out); code != want || strings.Contains(out.String(), "SECRET") {
				t.Fatalf("caso%s exit%d wanted%d salida%s", caso, code, want, out.String())
			}
		})
	}
}
