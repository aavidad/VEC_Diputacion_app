package conservacion

import (
	"os"
	"strings"
	"testing"
)

// La migración Documentos 000009 reserva a la custodia de documentos firmados
// el tipo de la resolución firmada de Contratación temporal por su referencia
// opaca. Esa constante debe coincidir con la que deriva este catálogo.
func TestTipoReservadoFirmadoCoincideConLaMigracion(t *testing.T) {
	sql, err := os.ReadFile("../../../../deploy/postgresql/documentos/migraciones/000009_custodia_documento_firmado.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	ref := referencia("tipo", "contratacion_temporal.resolucion_firmada.v1")
	if !strings.Contains(string(sql), "('"+ref+"',\n  'contratacion_temporal.resolucion_firmada.v1','contratacion_temporal')") {
		t.Fatalf("la migración no reserva %s para la resolución firmada", ref)
	}
}
