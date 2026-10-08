package importacionconvoca

import (
	"strings"
	"testing"
)

// Un texto enorme se rechaza por longitud sin normalizarlo; uno en el límite
// sigue normalizándose y aceptándose.
func TestLeerTextoDescartaTextoEnormeAntesDeNormalizar(t *testing.T) {
	fila := FilaStaging{Numero: 2, Celdas: []CeldaStaging{{Tipo: CeldaTexto, Valor: strings.Repeat("é", 40*120)}}}
	if _, codigo := leerTexto(fila, 0, true, 120); codigo != codigoTextoExcesivo {
		t.Fatalf("texto enorme: %q", codigo)
	}
	fila.Celdas[0].Valor = strings.Repeat("é", 120)
	if valor, codigo := leerTexto(fila, 0, true, 120); codigo != "" || len([]rune(valor)) != 120 {
		t.Fatalf("texto en el límite: %q %d", codigo, len([]rune(valor)))
	}
}
