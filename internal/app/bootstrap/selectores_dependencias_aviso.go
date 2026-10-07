package bootstrap

import (
	"log/slog"

	"vec-diputacion-granada/config"
)

// avisarDependenciasSelectoresBolsaCT deja en el registro un aviso por cada
// variable activa de Bolsa o Contratación temporal a la que le falta otra de
// la que depende, según config/selectores_dependencias_bolsa_ct_v1.json.
// Solo avisa: no detiene el arranque ni cambia la composición, y nunca
// escribe valores, solo nombres de variables. Devuelve cuántos avisos dio.
func avisarDependenciasSelectoresBolsaCT(registro *slog.Logger, entorno func(string) string) int {
	if registro == nil || entorno == nil {
		return 0
	}
	inventario, err := config.CargarInventarioSelectoresBolsaCT()
	if err != nil {
		registro.Warn("inventario de selectores de Bolsa y CT ilegible; no se comprueban dependencias")
		return 0
	}
	ausentes := inventario.DependenciasAusentes(entorno)
	for _, a := range ausentes {
		registro.Warn("variable de entorno activa sin otra de la que depende",
			"variable", a.Variable, "falta", a.Falta)
	}
	return len(ausentes)
}
