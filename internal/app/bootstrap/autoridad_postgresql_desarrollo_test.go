package bootstrap

import (
	"testing"
	"time"
)

func TestClonarInstantaneaAutorizacionPostgreSQLDesarrolloNoComparteConcesionesNiAmbitos(t *testing.T) {
	original, err := nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
		"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"prf_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	copia := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(original)
	if len(copia.VersionRol.Concesiones) == 0 || len(copia.AsignacionPerfil.Ambitos) == 0 {
		t.Fatal("la instantanea de prueba no cubre las colecciones copiadas")
	}
	copia.VersionRol.Concesiones[0].Finalidades[0] = "finalidad:ajena"
	copia.AsignacionPerfil.Ambitos[0].Valores[0] = "ambito:ajeno"
	if original.VersionRol.Concesiones[0].Finalidades[0] == "finalidad:ajena" ||
		original.AsignacionPerfil.Ambitos[0].Valores[0] == "ambito:ajeno" {
		t.Fatal("el clon comun comparte memoria con la instantanea de origen")
	}
}
