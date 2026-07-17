package domain

import "testing"

func TestClaveMotivoAutorizacionV2ValidaPerfilOpaco(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		clave  string
		valida bool
	}{
		{nombre: "opaca", clave: "motivo_0123456789abcdef0123456789abcdef", valida: true},
		{nombre: "semantica", clave: "consulta_expediente_personal"},
		{nombre: "dato personal", clave: "motivo_12345678Z"},
		{nombre: "mayusculas", clave: "motivo_0123456789ABCDEF0123456789ABCDEF"},
		{nombre: "corta", clave: "motivo_0123456789abcdef"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenida := ClaveMotivoAutorizacionV2Valida(caso.clave); obtenida != caso.valida {
				t.Fatalf("validez = %v; esperada %v", obtenida, caso.valida)
			}
		})
	}
}
