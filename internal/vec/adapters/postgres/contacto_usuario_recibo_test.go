package postgres

import (
	"context"
	"testing"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestReciboContactoPostgreSQLDeniegaSinPoolYMaterial(t *testing.T) {
	if c, err := nuevoConsultorReciboContactoUsuarioPostgreSQL(nil); err == nil || c != nil {
		t.Fatal("aceptó pool ausente")
	}
	pool := &iniciadorContactoPrueba{}
	c, err := nuevoConsultorReciboContactoUsuarioPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = c.ConsultarReciboContactoUsuario(context.Background(), ports.OrdenConsultaReciboContactoUsuario{}); err == nil || pool.llamadas != 0 {
		t.Fatal("consulta sin autorización abrió transacción")
	}
}
