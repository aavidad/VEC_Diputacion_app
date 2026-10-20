package main

import (
	"strings"
	"testing"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestGobiernoRolesExigeConjuntoSieteYAcuse(t *testing.T) {
	e := nuevoEscenario(t)
	base := destinoRolesPrueba(t, e)
	base.ConjuntoCapacidades = 6
	e.config = e.escribirConfig(t, base)
	if code, d, _, _ := e.ejecutar(e.args("gobierno-roles", "-acuse", "acuse-aplicar.json")); code != 1 || d.Codigo != "configuracion_invalida" || e.llamadas != 0 {
		t.Fatal("acepta gobierno de roles con conjunto anterior")
	}
	base.ConjuntoCapacidades = 7
	e.config = e.escribirConfig(t, base)
	if code, d, _, _ := e.ejecutar(e.args("gobierno-roles")); code != 1 || d.Codigo != "uso_invalido" || e.llamadas != 0 {
		t.Fatal("exporta sin acuse")
	}
	if code, d, _, _ := e.ejecutar(e.args("gobierno-roles", "-acuse", "acuse-aplicar.json")); code != 0 || d.Codigo != "gobierno_roles_escrito" || e.llamadas != 1 {
		t.Fatal("no exporta gobierno de roles confirmado", code, d.Codigo)
	}
}

func destinoRolesPrueba(t *testing.T, e *escenario) configuracionPrivada {
	t.Helper()
	base := configuracionPrivada{Salida: e.salida, ConjuntoCapacidades: 7}
	// Reutiliza el motivo nominal del mismo escenario; no acepta otro par.
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)}
	base.GobiernoRoles = &configuracionVersionBolsa{PoolGobierno: "/srv/privado/pools/roles.json", PoolCatalogo: "/srv/privado/pools/catalogo-roles.json",
		Motivos: map[string]domain.ReferenciaEntradaCatalogo{pg.AudienciaGobiernoRolProponer: motivo, pg.AudienciaGobiernoRolAprobar: motivo}}
	return base
}
