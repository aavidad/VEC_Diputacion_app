package main

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"testing"

	postgres "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestGobiernoInscripcionNoReutilizaPoolNiAudiencia(t *testing.T) {
	gobierno, base, u, runtime, lote, plan := gobiernoRolesConfigPrueba(t)
	var meta metadatosConfianzaPerfilesPrivados
	if json.Unmarshal(gobierno.ConfianzaJSON, &meta) != nil || len(meta.EntradasCapacidad) != 2 {
		t.Fatal("confianza de fixture")
	}
	dir := filepath.Dir(lote.PoolLote)
	meta.EntradasCapacidad[0].Audiencia = postgres.AudienciaGobiernoInscripcionProponer
	meta.EntradasCapacidad[0].MaterialArchivo = filepath.Join(dir, "inscripcion-proponer.bin")
	meta.EntradasCapacidad[1].Audiencia = postgres.AudienciaGobiernoInscripcionAprobar
	meta.EntradasCapacidad[1].MaterialArchivo = filepath.Join(dir, "inscripcion-aprobar.bin")
	confianza, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	motivo := motivoLotePrueba(u)
	c := configuracionGobiernoInscripcionPrivada{PoolGobierno: filepath.Join(dir, "inscripcion-gobierno.json"),
		PoolCatalogo: filepath.Join(dir, "inscripcion-catalogo.json"), ConfianzaJSON: confianza,
		Motivos: map[string]domain.ReferenciaEntradaCatalogo{
			postgres.AudienciaGobiernoInscripcionProponer: motivo,
			postgres.AudienciaGobiernoInscripcionAprobar:  motivo}}
	if err := validarConfiguracionGobiernoInscripcionPrivada(c, base, u, runtime, &lote, &plan, nil, &gobierno); err != nil {
		t.Fatalf("overlay propio válido: %v", err)
	}
	for nombre, mutar := range map[string]func(*configuracionGobiernoInscripcionPrivada){
		"pool_base":      func(x *configuracionGobiernoInscripcionPrivada) { x.PoolGobierno = lote.PoolLote },
		"pool_gobierno":  func(x *configuracionGobiernoInscripcionPrivada) { x.PoolCatalogo = gobierno.PoolCatalogo },
		"pool_duplicado": func(x *configuracionGobiernoInscripcionPrivada) { x.PoolCatalogo = x.PoolGobierno },
		"motivo_ausente": func(x *configuracionGobiernoInscripcionPrivada) {
			delete(x.Motivos, postgres.AudienciaGobiernoInscripcionAprobar)
		},
		"audiencia_rol": func(x *configuracionGobiernoInscripcionPrivada) {
			var m metadatosConfianzaPerfilesPrivados
			_ = json.Unmarshal(x.ConfianzaJSON, &m)
			m.EntradasCapacidad[1].Audiencia = "vec_autorizacion.gobierno_rol_nuevo.cierre.v1"
			x.ConfianzaJSON, _ = json.Marshal(m)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := c
			alterada.Motivos = maps.Clone(c.Motivos)
			mutar(&alterada)
			if validarConfiguracionGobiernoInscripcionPrivada(alterada, base, u, runtime, &lote, &plan, nil, &gobierno) == nil {
				t.Fatal("se aceptó otro pool, audiencia o motivo")
			}
		})
	}
}
