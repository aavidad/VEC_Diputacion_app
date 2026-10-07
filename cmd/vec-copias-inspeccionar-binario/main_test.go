package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

type salidaFallida struct{}

func (salidaFallida) Write([]byte) (int, error) {
	return 0, errors.New("salida_sintetica_no_disponible")
}

func TestCLINoOcultaFalloDeEscritura(t *testing.T) {
	for _, argumentos := range [][]string{nil, {"-raiz", t.TempDir(), "-binario", "ausente", "-max-bytes", "1024"}} {
		if codigo := ejecutar(argumentos, salidaFallida{}); codigo != 4 {
			t.Fatalf("codigo=%d", codigo)
		}
	}
}
