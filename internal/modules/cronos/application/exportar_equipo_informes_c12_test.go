package application

import (
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/domain"
)

func TestCamposInformeC12CierranDatosSensibles(t *testing.T) {
	casos := []struct {
		tipo   domain.TipoInformeCronos
		equipo bool
		campos []string
		valido bool
	}{
		{domain.InformeSaldo, false, []string{"fecha", "saldo_minutos"}, true},
		{domain.InformeSaldo, false, []string{"empleado_etiqueta"}, false},
		{domain.InformeSaldo, true, []string{"empleado_etiqueta", "fecha"}, true},
		{domain.InformeSaldo, true, []string{"empleado_ref"}, false},
		{domain.InformePermisos, true, []string{"tipo", "motivo"}, false},
		{domain.InformeMovimientos, false, []string{"fecha", "fecha"}, false},
	}
	for _, c := range casos {
		if got := camposInformeC12Validos(c.tipo, c.equipo, c.campos); got != c.valido {
			t.Fatalf("tipo=%s campos=%v: %v", c.tipo, c.campos, got)
		}
	}
}
