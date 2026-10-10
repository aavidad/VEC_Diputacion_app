package main

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"testing"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

func gobiernoRolesConfigPrueba(t *testing.T) (configuracionGobiernoRolesPrivada,
	configuracionPerfilesPrivada, configuracionUsuariosMetadatosPrivada,
	configuracionRuntimeADMIN, configuracionLotePrivada, configuracionPlanFirmaPrivada) {
	t.Helper()
	lote, base, u, runtime := loteADMINPrueba(t)
	plan := planFirmaADMINPrueba(t, lote)
	var m metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(u.ConfianzaJSON, &m); err != nil || len(m.EntradasCapacidad) < 2 {
		t.Fatal("metadatos confianza de prueba")
	}
	dir := filepath.Dir(lote.PoolLote)
	m.EntradasCapacidad = m.EntradasCapacidad[:2]
	m.EntradasCapacidad[0].Audiencia = pg.AudienciaGobiernoRolProponer
	m.EntradasCapacidad[0].MaterialArchivo = filepath.Join(dir, "gobierno-rol-proponer.bin")
	m.EntradasCapacidad[1].Audiencia = pg.AudienciaGobiernoRolAprobar
	m.EntradasCapacidad[1].MaterialArchivo = filepath.Join(dir, "gobierno-rol-aprobar.bin")
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	motivo := motivoLotePrueba(u)
	c := configuracionGobiernoRolesPrivada{PoolGobierno: filepath.Join(dir, "gobierno-rol.json"),
		PoolCatalogo: filepath.Join(dir, "catalogo-acciones-lector.json"), ConfianzaJSON: b,
		Motivos: map[string]domain.ReferenciaEntradaCatalogo{
			pg.AudienciaGobiernoRolProponer: motivo, pg.AudienciaGobiernoRolAprobar: motivo}}
	return c, base, u, runtime, lote, plan
}

func TestGobiernoRolesConfiguracionPrivadaCierraPoolsYAudiancias(t *testing.T) {
	c, base, u, runtime, lote, plan := gobiernoRolesConfigPrueba(t)
	if err := validarConfiguracionGobiernoRolesPrivada(c, base, u, runtime, &lote, &plan, nil); err != nil {
		t.Fatalf("configuración nominal válida: %v", err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	leida, err := cargarConfiguracionGobiernoRolesPrivada(archivoPrivadoPrueba(t, string(b)), base, u, runtime, &lote, &plan, nil)
	if err != nil || leida.PoolGobierno != c.PoolGobierno || leida.PoolCatalogo != c.PoolCatalogo {
		t.Fatal("overlay privado no conservado")
	}
	for nombre, mutar := range map[string]func(*configuracionGobiernoRolesPrivada){
		"pool_reutilizado":   func(x *configuracionGobiernoRolesPrivada) { x.PoolGobierno = lote.PoolLote },
		"lector_reutilizado": func(x *configuracionGobiernoRolesPrivada) { x.PoolCatalogo = u.PoolLector },
		"pools_iguales":      func(x *configuracionGobiernoRolesPrivada) { x.PoolCatalogo = x.PoolGobierno },
		"motivo_ausente":     func(x *configuracionGobiernoRolesPrivada) { delete(x.Motivos, pg.AudienciaGobiernoRolAprobar) },
		"audiencia_ajena": func(x *configuracionGobiernoRolesPrivada) {
			var m metadatosConfianzaPerfilesPrivados
			_ = json.Unmarshal(x.ConfianzaJSON, &m)
			m.EntradasCapacidad[1].Audiencia = "vec_autorizacion.administracion_perfiles.cierre.v1"
			x.ConfianzaJSON, _ = json.Marshal(m)
		},
		"material_compartido": func(x *configuracionGobiernoRolesPrivada) {
			var m metadatosConfianzaPerfilesPrivados
			_ = json.Unmarshal(x.ConfianzaJSON, &m)
			m.EntradasCapacidad[1].MaterialArchivo = m.EntradasCapacidad[0].MaterialArchivo
			x.ConfianzaJSON, _ = json.Marshal(m)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := c
			alterada.Motivos = maps.Clone(c.Motivos)
			mutar(&alterada)
			if validarConfiguracionGobiernoRolesPrivada(alterada, base, u, runtime, &lote, &plan, nil) == nil {
				t.Fatal("configuración ajena aceptada")
			}
		})
	}
}
