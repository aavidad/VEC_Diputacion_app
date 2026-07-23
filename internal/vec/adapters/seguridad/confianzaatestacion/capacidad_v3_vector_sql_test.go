package confianzaatestacion

import (
	"bytes"
	"encoding/base64"
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

// TestGenerarVectorO205ParaSQL usa exactamente el emisor MAC de producción.
// Solo actúa cuando el runner PostgreSQL aporta ficheros efímeros; una
// ejecución ordinaria no lee secretos ni genera artefactos.
func TestGenerarVectorO205ParaSQL(t *testing.T) {
	rutaEntrada := os.Getenv("VEC_O205_VECTOR_ENTRADA")
	rutaClave := os.Getenv("VEC_O205_CLAVE_ENTRADA")
	rutaSalida := os.Getenv("VEC_O205_VECTOR_SALIDA")
	if rutaEntrada == "" || rutaClave == "" || rutaSalida == "" {
		t.Skip("solo se ejecuta desde la integración PostgreSQL O2-05")
	}
	entradaB64, err := os.ReadFile(rutaEntrada)
	if err != nil {
		t.Fatalf("leer capacidad de entrada: %v", err)
	}
	claveB64, err := os.ReadFile(rutaClave)
	if err != nil {
		t.Fatalf("leer clave efímera: %v", err)
	}
	entrada, err := base64.StdEncoding.DecodeString(
		string(bytes.TrimSpace(entradaB64)),
	)
	if err != nil {
		t.Fatalf("decodificar capacidad de entrada: %v", err)
	}
	clave, err := base64.StdEncoding.DecodeString(
		string(bytes.TrimSpace(claveB64)),
	)
	if err != nil {
		t.Fatalf("decodificar clave efímera: %v", err)
	}
	defer borrarBytesConfianzaAtestacion(clave)
	documento, err := interpretarExportacionCapacidadV3(entrada)
	if err != nil {
		t.Fatalf("interpretar capacidad de entrada: %v", err)
	}
	documento.MACSHA256 = calcularMACCapacidadAtestacionV3(documento, clave)
	capacidad, err := nuevaCapacidadBreveAtestacionAutorizacionV3(documento)
	if err != nil {
		t.Fatalf("emitir capacidad con MAC Go real: %v", err)
	}
	salida, err := capacidad.ExportacionCanonicaParaConsumidor()
	if err != nil {
		t.Fatalf("exportar capacidad Go real: %v", err)
	}
	if err := os.WriteFile(rutaSalida, salida, 0o600); err != nil {
		t.Fatalf("escribir capacidad Go real: %v", err)
	}
}
