package ports

import (
	"errors"
	"testing"
)

func TestClaseVersionBolsaConservaCausaYListaCerrada(t *testing.T) {
	causa := errors.New("causa con datos privados")
	err := ConClaseVersionBolsa("v3_sin_concesion", causa)
	if !errors.Is(err, causa) || err.Error() != causa.Error() || ClaseFalloVersionBolsa(err) != "v3_sin_concesion" {
		t.Fatalf("la clase cambió la causa o no quedó: %v", err)
	}
	if ClaseFalloVersionBolsa(ConClaseVersionBolsa("administrador_rol", err)) != "v3_sin_concesion" {
		t.Fatal("la clase externa sustituyó a la más interna")
	}
	if ClaseFalloVersionBolsa(ConClaseVersionBolsa("clase libre", causa)) != "" ||
		ClaseFalloVersionBolsa(causa) != "" || ConClaseVersionBolsa("v3_emision", nil) != nil {
		t.Fatal("se aceptó una clase fuera de la lista o se inventó un error")
	}
	unido := errors.Join(ErrGobiernoRolIntentoAuditado, ConClaseVersionBolsa("sql_intento_error", causa))
	if ClaseFalloVersionBolsa(unido) != "sql_intento_error" || !errors.Is(unido, ErrGobiernoRolIntentoAuditado) {
		t.Fatal("la clase se perdió dentro de errors.Join")
	}
	if !ClaseVersionBolsaAdmitida("rol_administrable_respuesta") || ClaseVersionBolsaAdmitida("") {
		t.Fatal("lista cerrada inconsistente")
	}
}
