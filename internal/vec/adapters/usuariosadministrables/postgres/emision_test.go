package postgres

import (
	"bytes"
	"context"
	"maps"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func emisionPrueba(t *testing.T, p peticion) ports.EmisionUsuariosAdministrables {
	t.Helper()
	ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	r := p.recurso
	r.Ambitos = maps.Clone(r.Ambitos)
	r.Atributos = maps.Clone(r.Atributos)
	return ports.EmisionUsuariosAdministrables{Material: bytes.Clone(p.material), Recurso: r, Accion: p.accion, Audiencia: p.audiencia, Correlacion: correlacion}
}

func TestValidadorEmisionCierraCanonYEntregaCopia(t *testing.T) {
	a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
	lista, err := materialListar(a, ports.FiltrosUsuariosAdministrables{})
	if err != nil {
		t.Fatal(err)
	}
	base := emisionPrueba(t, lista)
	copia, err := ValidarEmisionUsuariosAdministrables(base)
	if err != nil {
		t.Fatal(err)
	}
	base.Material[0] = 'X'
	base.Recurso.Ambitos["unidad_ref"] = "unidad_ajena"
	base.Recurso.Atributos["material_sha256"] = "cambiada"
	if !bytes.Equal(copia.Material, lista.material) || copia.Recurso.Ambitos["unidad_ref"] != a.UnidadRef || copia.Recurso.Atributos["material_sha256"] == "cambiada" {
		t.Fatal("la copia validada comparte material o mapas con el llamador")
	}
	ficha, err := materialConsultar(a, "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidarEmisionUsuariosAdministrables(emisionPrueba(t, ficha)); err != nil {
		t.Fatalf("ficha canónica rechazada: %v", err)
	}

	casos := map[string]func(*ports.EmisionUsuariosAdministrables){
		"accion_ajena":       func(e *ports.EmisionUsuariosAdministrables) { e.Accion = "administracion.usuarios.borrar" },
		"audiencia_prestada": func(e *ports.EmisionUsuariosAdministrables) { e.Audiencia = audienciaFicha },
		"scope_alterado": func(e *ports.EmisionUsuariosAdministrables) {
			e.Material = []byte(strings.Replace(string(e.Material), a.OrganizacionRef, "org_cccccccccccccccccccccccccccccccc", 1))
		},
		"sha_material_alterado": func(e *ports.EmisionUsuariosAdministrables) {
			e.Recurso.Atributos["material_sha256"] = strings.Repeat("0", 64)
		},
		"campo_extra": func(e *ports.EmisionUsuariosAdministrables) {
			e.Material = bytes.Replace(e.Material, []byte(`,"limite":50}`), []byte(`,"limite":50,"permiso":true}`), 1)
		},
		"json_final": func(e *ports.EmisionUsuariosAdministrables) { e.Material = append(e.Material, []byte(` {}`)...) },
		"duplicado": func(e *ports.EmisionUsuariosAdministrables) {
			e.Material = bytes.Replace(e.Material, []byte(`,"limite":50}`), []byte(`,"limite":50,"limite":50}`), 1)
		},
		"cursor_sin_ligadura": func(e *ports.EmisionUsuariosAdministrables) {
			e.Material = bytes.Replace(e.Material, []byte(`"cursor":""`), []byte(`"cursor":"usuarios:prestado"`), 1)
		},
		"correlacion_cero": func(e *ports.EmisionUsuariosAdministrables) {
			e.Correlacion = domain.ReferenciaCorrelacionAutorizacionV2{}
		},
	}
	for nombre, muta := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := emisionPrueba(t, lista)
			muta(&e)
			if salida, err := ValidarEmisionUsuariosAdministrables(e); err == nil || len(salida.Material) != 0 {
				t.Fatalf("material alterado admitido: %+v %v", salida, err)
			}
		})
	}
}
