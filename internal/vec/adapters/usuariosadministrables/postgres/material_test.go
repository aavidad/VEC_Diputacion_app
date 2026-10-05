package postgres

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestMaterialAUT43CoincideConVectorNominal(t *testing.T) {
	a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
	lista, err := materialListar(a, ports.FiltrosUsuariosAdministrables{})
	if err != nil {
		t.Fatal(err)
	}
	esperada := `{"esquema":"vec.admin.usuarios.listar.v1","organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica","conjunto_ref":"conjunto_admin:dfa8fa3eef2981ce04ad7dcccbee704d","filtros":{"perfil_ref":"","unidad_ref":"","estado":""},"cursor":"","limite":50}`
	if string(lista.material) != esperada {
		t.Fatalf("material lista divergente: %s", lista.material)
	}
	huella, err := lista.recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huella != "a44970b1679b0fb6b534263e3f3923540fa881ba4817b30084a052d24b6c787a" {
		t.Fatalf("contexto lista divergente: %s %v", huella, err)
	}
	ficha, err := materialConsultar(a, "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	esperada = `{"esquema":"vec.admin.usuarios.consultar.v1","organizacion_ref":"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","unidad_ref":"unidad_admin_sintetica","conjunto_ref":"conjunto_admin:dfa8fa3eef2981ce04ad7dcccbee704d","persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	if string(ficha.material) != esperada {
		t.Fatalf("material ficha divergente: %s", ficha.material)
	}
	huella, err = ficha.recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || huella != "23488ca0bca6ef20ede975f7aaabe00976ddadb349c3e757acd4d110c00d82b1" {
		t.Fatalf("contexto ficha divergente: %s %v", huella, err)
	}
}

func TestCursorLigaConjuntoYFiltros(t *testing.T) {
	a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
	conjunto, err := conjuntoUsuarios(a)
	if err != nil {
		t.Fatal(err)
	}
	persona := "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	cursor, err := cursorUsuarios(conjunto, filtrosMaterial{"rol:administracion_perfiles:v5", "", "vigente"}, persona)
	if err != nil {
		t.Fatal(err)
	}
	base := ports.FiltrosUsuariosAdministrables{PerfilRef: "rol:administracion_perfiles:v5", Estado: "vigente", Cursor: cursor}
	if _, err := materialListar(a, base); err != nil {
		t.Fatal(err)
	}
	base.Estado = "caducado"
	if _, err := materialListar(a, base); err == nil {
		t.Fatal("cursor reutilizado con otro estado")
	}
	base.Estado = "vigente"
	a.UnidadRef = "unidad_distinta"
	if _, err := materialListar(a, base); err == nil {
		t.Fatal("cursor reutilizado con otro conjunto")
	}
	if !strings.HasPrefix(cursor, "usuarios:") {
		t.Fatal("cursor sin prefijo propio")
	}
}
