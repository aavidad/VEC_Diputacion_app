package bootstrap

import (
	"testing"
	"time"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
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
	original.VersionRol.Concesiones[0].CamposPermitidos = []string{"plan"}
	original.VersionRol.Concesiones[0].Obligaciones = []string{"registrar_auditoria"}
	original.Politicas = []dominiovec.PoliticaRestrictiva{{CamposPermitidos: []string{"plan"}, Obligaciones: []string{"registrar_auditoria"}}}
	copia := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(original)
	if len(copia.VersionRol.Concesiones) == 0 || len(copia.AsignacionPerfil.Ambitos) == 0 {
		t.Fatal("la instantanea de prueba no cubre las colecciones copiadas")
	}
	copia.VersionRol.Concesiones[0].Finalidades[0] = "finalidad:ajena"
	copia.AsignacionPerfil.Ambitos[0].Valores[0] = "ambito:ajeno"
	copia.VersionRol.Concesiones[0].CamposPermitidos[0] = "campo_ajeno"
	copia.VersionRol.Concesiones[0].Obligaciones[0] = "obligacion_ajena"
	copia.Politicas[0].CamposPermitidos[0] = "campo_ajeno"
	copia.Politicas[0].Obligaciones[0] = "obligacion_ajena"
	if original.VersionRol.Concesiones[0].Finalidades[0] == "finalidad:ajena" ||
		original.AsignacionPerfil.Ambitos[0].Valores[0] == "ambito:ajeno" ||
		original.VersionRol.Concesiones[0].CamposPermitidos[0] != "plan" || original.VersionRol.Concesiones[0].Obligaciones[0] != "registrar_auditoria" ||
		original.Politicas[0].CamposPermitidos[0] != "plan" || original.Politicas[0].Obligaciones[0] != "registrar_auditoria" {
		t.Fatal("el clon comun comparte memoria con la instantanea de origen")
	}
	original.VersionRol.Concesiones[0].CamposPermitidos[0] = "campo_del_origen"
	original.VersionRol.Concesiones[0].Obligaciones[0] = "obligacion_del_origen"
	original.Politicas[0].CamposPermitidos[0] = "campo_del_origen"
	original.Politicas[0].Obligaciones[0] = "obligacion_del_origen"
	if copia.VersionRol.Concesiones[0].CamposPermitidos[0] != "campo_ajeno" || copia.VersionRol.Concesiones[0].Obligaciones[0] != "obligacion_ajena" ||
		copia.Politicas[0].CamposPermitidos[0] != "campo_ajeno" || copia.Politicas[0].Obligaciones[0] != "obligacion_ajena" {
		t.Fatal("cambiar el origen alteró los campos u obligaciones del clon")
	}
}
