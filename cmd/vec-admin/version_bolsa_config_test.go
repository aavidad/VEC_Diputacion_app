package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestVersionBolsaOverlaySeparaLoginYMaterialDeGobiernoRol(t *testing.T) {
	gobierno, base, u, runtime, lote, plan := gobiernoRolesConfigPrueba(t)
	var meta metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(gobierno.ConfianzaJSON, &meta); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(gobierno.PoolGobierno)
	meta.EntradasCapacidad[0].Audiencia = pg.AudienciaVersionarRolBolsaProponer
	meta.EntradasCapacidad[0].MaterialArchivo = filepath.Join(dir, "version-bolsa-proponer.bin")
	meta.EntradasCapacidad[1].Audiencia = pg.AudienciaVersionarRolBolsaAprobar
	meta.EntradasCapacidad[1].MaterialArchivo = filepath.Join(dir, "version-bolsa-aprobar.bin")
	confianza, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	motivo := motivoLotePrueba(u)
	version := configuracionVersionBolsaPrivada{
		PoolGobierno:  filepath.Join(dir, "version-bolsa.json"),
		PoolCatalogo:  filepath.Join(dir, "catalogo-bolsa-lector.json"),
		ConfianzaJSON: confianza,
		Motivos: map[string]domain.ReferenciaEntradaCatalogo{
			pg.AudienciaVersionarRolBolsaProponer: motivo,
			pg.AudienciaVersionarRolBolsaAprobar:  motivo,
		},
	}
	if err := validarConfiguracionVersionBolsaPrivada(version, base, u, runtime, &lote, &plan, nil, &gobierno); err != nil {
		t.Fatalf("overlays segregados rechazados: %v", err)
	}
	version.PoolGobierno = gobierno.PoolGobierno
	if validarConfiguracionVersionBolsaPrivada(version, base, u, runtime, &lote, &plan, nil, &gobierno) == nil {
		t.Fatal("LOGIN compartido con definición nueva")
	}
	version.PoolGobierno = filepath.Join(dir, "version-bolsa.json")
	version.ConfianzaJSON = gobierno.ConfianzaJSON
	if validarConfiguracionVersionBolsaPrivada(version, base, u, runtime, &lote, &plan, nil, &gobierno) == nil {
		t.Fatal("material o audiencia de definición nueva prestado a B1")
	}
}
