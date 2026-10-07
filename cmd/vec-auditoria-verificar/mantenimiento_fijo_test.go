package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCLIAD183YActorAjeno(t *testing.T) {
	b, e := os.ReadFile("testdata/mantenimiento_fijo_ad183.json")
	if e != nil {
		t.Fatal(e)
	}
	var d map[string]json.RawMessage
	if json.Unmarshal(b, &d) != nil {
		t.Fatal("fixture")
	}
	p := filepath.Join(t.TempDir(), "checkpoint.json")
	if os.WriteFile(p, d["manifiesto"], 0600) != nil {
		t.Fatal("checkpoint")
	}
	a := []string{"-checkpoint", p, "-max-bytes", "65536", "-max-registros", "5"}
	var out bytes.Buffer
	if c := ejecutar(a, bytes.NewReader(b), &out); c != 0 {
		t.Fatalf("CLI: %d %s", c, out.String())
	}
	for _, clave := range []string{"actor_ref", "perfil_activo_ref", "decision_ref"} {
		alterado := bytes.Replace(b, []byte(`"plan_sha256":`), []byte(`"`+clave+`":"inventado","plan_sha256":`), 1)
		out.Reset()
		if c := ejecutar(a, bytes.NewReader(alterado), &out); c != 2 {
			t.Fatalf("campo ajeno: %d %s", c, out.String())
		}
	}
}
