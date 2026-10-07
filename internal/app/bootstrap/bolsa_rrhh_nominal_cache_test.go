package bootstrap

import (
	"testing"
	"time"
)

func TestSeleccionTextoRRHHBolsaLigaActorFiltroInstantaneaYCaducidad(t *testing.T) {
	s := seleccionTextoRRHHBolsa{ids: []string{"participacion:1", "participacion:2"},
		principalID: "certificado:1", personaRef: "persona:1", perfilRef: "perfil:rrhh",
		bolsaRef: "bolsa:1", filtroSHA256: "filtro:1", snapshotSHA256: "snapshot:1",
		expira: time.Now().Add(time.Minute)}
	valida := func(principal, persona, perfil, bolsa, filtro, snapshot string, indice int) bool {
		return seleccionTextoRRHHBolsaValida(s, principal, persona, perfil, bolsa, filtro, snapshot, indice)
	}
	if !valida("certificado:1", "persona:1", "perfil:rrhh", "bolsa:1", "filtro:1", "snapshot:1", 1) ||
		valida("certificado:2", "persona:1", "perfil:rrhh", "bolsa:1", "filtro:1", "snapshot:1", 1) ||
		valida("certificado:1", "persona:2", "perfil:rrhh", "bolsa:1", "filtro:1", "snapshot:1", 1) ||
		valida("certificado:1", "persona:1", "perfil:otro", "bolsa:1", "filtro:1", "snapshot:1", 1) ||
		valida("certificado:1", "persona:1", "perfil:rrhh", "bolsa:2", "filtro:1", "snapshot:1", 1) ||
		valida("certificado:1", "persona:1", "perfil:rrhh", "bolsa:1", "filtro:2", "snapshot:1", 1) ||
		valida("certificado:1", "persona:1", "perfil:rrhh", "bolsa:1", "filtro:1", "snapshot:2", 1) ||
		valida("certificado:1", "persona:1", "perfil:rrhh", "bolsa:1", "filtro:1", "snapshot:1", 2) {
		t.Fatal("selección textual reutilizable fuera de su actor, filtro, corte o página")
	}
	s.expira = time.Now().Add(-time.Second)
	if valida("certificado:1", "persona:1", "perfil:rrhh", "bolsa:1", "filtro:1", "snapshot:1", 1) {
		t.Fatal("selección textual caducada aceptada")
	}
}
