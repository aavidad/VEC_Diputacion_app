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

func TestClaseIntentoSQLVersionBolsaSoloFormatoSQLSTATE(t *testing.T) {
	for _, caso := range []struct{ estado, sqlstate, clase string }{
		{"error", "42703", "sql_intento_error_42703"},
		{"error", "22P02", "sql_intento_error_22P02"},
		{"denegado", "42501", "sql_intento_denegado_42501"},
		{"error", "", "sql_intento_error"},
		{"denegado", "", "sql_intento_denegado"},
		{"error", "4270a", "sql_intento_error"},
		{"error", "4270", "sql_intento_error"},
		{"error", "427031", "sql_intento_error"},
		{"error", "42 03", "sql_intento_error"},
		{"error", "42703\n", "sql_intento_error"},
		{"otro", "42703", "sql_intento_error_42703"},
	} {
		if c := ClaseIntentoSQLVersionBolsa(caso.estado, caso.sqlstate); c != caso.clase {
			t.Fatalf("%q/%q: clase %q, esperada %q", caso.estado, caso.sqlstate, c, caso.clase)
		}
		causa := errors.New("record h has no field asignacion_id")
		if ClaseFalloVersionBolsa(ConClaseVersionBolsa(caso.clase, causa)) != caso.clase {
			t.Fatalf("%q no se admite como clase", caso.clase)
		}
	}
	for _, clase := range []string{"sql_intento_error_", "sql_intento_error_4270a", "sql_intento_error_427031",
		"sql_intento_otro_42703", "v3_emision_42703", "sql_intento_denegado_42501x", "sql_intento_error42703"} {
		if ClaseVersionBolsaAdmitida(clase) || ClaseFalloVersionBolsa(ConClaseVersionBolsa(clase, errors.New("x"))) != "" {
			t.Fatalf("se admitió %q", clase)
		}
	}
}
