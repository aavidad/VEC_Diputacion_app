package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCLIFallosConJSONYLimiteExplicito(t *testing.T) {
	for _, caso := range []struct {
		args   []string
		codigo int
	}{
		{nil, 2},
		{[]string{"-raiz", t.TempDir(), "-binario", "ausente"}, 2},
		{[]string{"-raiz", t.TempDir(), "-binario", "ausente", "-max-bytes", "1024"}, 1},
	} {
		var salida bytes.Buffer
		if codigo := ejecutar(caso.args, &salida); codigo != caso.codigo || !json.Valid(salida.Bytes()) {
			t.Fatalf("codigo=%d salida=%s", codigo, salida.String())
		}
	}
}
