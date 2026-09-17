package bootstrap

import (
	"testing"
	"vec-diputacion-granada/config"
)

func TestCustodiaImportacionConvocaFallaFueraDeDesarrollo(t *testing.T) {
	if _, e := custodiarImportacionConvocaDesarrollo(config.Config{}, []byte("xls")); e == nil {
		t.Fatal("custodia habilitada fuera de desarrollo")
	}
}
