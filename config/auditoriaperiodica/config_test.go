package auditoriaperiodica

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestEjecutorConfiguracionCerradaSinCredenciales(t *testing.T) {
	c := Ejecutor{Version: 1, MaxRegistros: 1000, VersionBinario: "desarrollo:1", PinSPKISHA256: strings.Repeat("a", 64), TimeoutSegundos: 30}
	raw, _ := json.Marshal(c)
	obtenida, err := Decodificar(raw)
	if err != nil || obtenida != c || obtenida.Timeout() != 30*time.Second {
		t.Fatal("configuración válida rechazada")
	}
	for _, mala := range [][]byte{
		append(append([]byte{}, raw...), raw...),
		[]byte(strings.Replace(string(raw), `"version":1`, `"version":0`, 1)),
		[]byte(strings.Replace(string(raw), `"version":1`, `"version":0,"version":1`, 1)),
		[]byte(strings.Replace(string(raw), `"max_registros":1000`, `"max_registros":0`, 1)),
		[]byte(strings.Replace(string(raw), `"timeout_segundos":30`, `"timeout_segundos":301`, 1)),
		[]byte(strings.Replace(string(raw), `"timeout_segundos":30`, `"timeout_segundos":-1`, 1)),
		[]byte(strings.Replace(string(raw), strings.Repeat("a", 64), strings.Repeat("0", 64), 1)),
		[]byte(strings.Replace(string(raw), `"version":1`, `"dsn":"postgres://privado","version":1`, 1)),
		[]byte(strings.Repeat(" ", 16*1024+1)),
	} {
		if _, err := Decodificar(mala); err == nil {
			t.Fatal("configuración inválida aceptada")
		}
	}
	c.TimeoutSegundos = 1 << 62
	if c.Timeout() != 0 {
		t.Fatal("timeout inválido desbordado")
	}
}
