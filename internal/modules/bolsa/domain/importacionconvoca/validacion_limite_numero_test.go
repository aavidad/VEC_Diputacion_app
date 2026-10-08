package importacionconvoca

import (
	"strings"
	"testing"
)

// Una celda numérica enorme se rechaza sin recortarla ni recorrerla.
func TestCeldasNumericasEnormesSeRechazanSinRecorrer(t *testing.T) {
	fila := FilaStaging{Numero: 2, Celdas: []CeldaStaging{{Tipo: CeldaTexto, Valor: strings.Repeat(" ", 65536) + "1"}}}
	if _, codigo := leerDecimal(fila, 0, true); codigo != codigoDecimalInvalido {
		t.Fatalf("decimal enorme: %q", codigo)
	}
	if _, codigo := leerOrdenGrupo(fila, 0); codigo != codigoOrdenGrupoInvalido {
		t.Fatalf("orden enorme: %q", codigo)
	}
	fila.Celdas[0].Valor = " 12,5 "
	if valor, codigo := leerDecimal(fila, 0, true); codigo != "" || valor != "12.5" {
		t.Fatalf("decimal normal: %q %q", valor, codigo)
	}
}
