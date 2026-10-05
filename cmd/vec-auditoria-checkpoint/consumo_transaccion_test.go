package main

import "testing"

func TestCLIPaqueteFirmadoConConsumosAD193Mixtos(t *testing.T) {
	verificarPaqueteExportacionPrueba(t, "../vec-auditoria-verificar/testdata/consumo_transaccion_ad193.json", true)
}
