package confianzaatestacion

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestVectorCanonicoO205EsComunAGoYSQL(t *testing.T) {
	t.Parallel()

	ruta := filepath.Join(
		"testdata",
		"capacidad_v3_canonica_o2_05.json",
	)
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("leer vector compartido: %v", err)
	}
	contenido = bytes.TrimSuffix(contenido, []byte{'\n'})
	documento, err := interpretarExportacionCapacidadV3(contenido)
	if err != nil {
		t.Fatalf("Go rechazó el vector compartido: %v", err)
	}
	canonica, err := json.Marshal(documento)
	if err != nil {
		t.Fatalf("serializar vector compartido: %v", err)
	}
	if !bytes.Equal(canonica, contenido) {
		t.Fatal("Go no conserva exactamente los bytes canónicos compartidos")
	}
}
