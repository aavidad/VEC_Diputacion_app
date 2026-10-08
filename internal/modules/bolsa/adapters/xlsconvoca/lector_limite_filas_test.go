package xlsconvoca_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/adapters/xlsconvoca"
)

// El tope propio de filas corta antes de leer las celdas, en XLS y en XLSX.
func TestLectorConLimiteFilasCortaXLSYXLSX(t *testing.T) {
	xls := leerFixture(t, "resumen.xls")
	xlsx := construirXLSXPrueba(t, hojaResumenXLSXPrueba(t), opcionesXLSX{})
	for nombre, libro := range map[string][]byte{"xls": xls, "xlsx": xlsx} {
		if _, err := xlsconvoca.NuevoLectorConLimiteFilas(2).Decodificar(context.Background(), bytes.NewReader(libro)); !errors.Is(err, xlsconvoca.ErrLimiteXLSExcedido) {
			t.Fatalf("%s: tope de 2 filas no aplicado: %v", nombre, err)
		}
		if _, err := xlsconvoca.NuevoLectorConLimiteFilas(4).Decodificar(context.Background(), bytes.NewReader(libro)); err != nil {
			t.Fatalf("%s: tope de 4 filas rechazó cabecera y tres filas: %v", nombre, err)
		}
		if _, err := xlsconvoca.NuevoLectorConLimiteFilas(0).Decodificar(context.Background(), bytes.NewReader(libro)); err != nil {
			t.Fatalf("%s: sin tope propio debe usar el máximo del adaptador: %v", nombre, err)
		}
	}
}
