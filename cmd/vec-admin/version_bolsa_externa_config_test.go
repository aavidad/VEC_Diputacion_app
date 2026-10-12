package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestVersionBolsaExternaOverlaySegregaPoolsYMateriales(t *testing.T) {
	gobierno, base, u, runtime, lote, plan := gobiernoRolesConfigPrueba(t)
	var meta metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(gobierno.ConfianzaJSON, &meta); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(gobierno.PoolGobierno)
	motivo := motivoLotePrueba(u)
	armar := func(nombre, proponer, aprobar string) configuracionGobiernoRolesPrivada {
		m := meta
		m.EntradasCapacidad = append([]capacidadConfianzaPerfilesPrivada(nil), meta.EntradasCapacidad...)
		m.EntradasCapacidad[0].Audiencia = proponer
		m.EntradasCapacidad[0].MaterialArchivo = filepath.Join(dir, nombre+"-proponer.bin")
		m.EntradasCapacidad[1].Audiencia = aprobar
		m.EntradasCapacidad[1].MaterialArchivo = filepath.Join(dir, nombre+"-aprobar.bin")
		confianza, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return configuracionGobiernoRolesPrivada{PoolGobierno: filepath.Join(dir, nombre+"-ejecutor.json"),
			PoolCatalogo: filepath.Join(dir, nombre+"-catalogo.json"), ConfianzaJSON: confianza,
			Motivos: map[string]domain.ReferenciaEntradaCatalogo{proponer: motivo, aprobar: motivo}}
	}
	b1 := configuracionVersionBolsaPrivada(armar("b1", pg.AudienciaVersionarRolBolsaProponer, pg.AudienciaVersionarRolBolsaAprobar))
	b103 := configuracionVersionBolsaB103Privada(armar("b103", pg.AudienciaVersionarRolBolsaProponer, pg.AudienciaVersionarRolBolsaAprobar))
	ext := configuracionVersionBolsaExternaPrivada(armar("externa", pg.AudienciaVersionarRolBolsaExternaProponer, pg.AudienciaVersionarRolBolsaExternaAprobar))
	validar := func(c configuracionVersionBolsaExternaPrivada) error {
		return validarConfiguracionVersionBolsaExternaPrivada(c, base, u, runtime, &lote, &plan, nil, &gobierno, &b1, &b103)
	}
	if err := validar(ext); err != nil {
		t.Fatalf("overlay externo separado rechazado: %v", err)
	}
	for _, caso := range []struct {
		name  string
		mutar func(*configuracionVersionBolsaExternaPrivada)
	}{
		{"login b1", func(c *configuracionVersionBolsaExternaPrivada) { c.PoolGobierno = b1.PoolGobierno }},
		{"catalogo b103", func(c *configuracionVersionBolsaExternaPrivada) { c.PoolCatalogo = b103.PoolCatalogo }},
		{"material b1", func(c *configuracionVersionBolsaExternaPrivada) { c.ConfianzaJSON = b1.ConfianzaJSON }},
		{"material b103", func(c *configuracionVersionBolsaExternaPrivada) { c.ConfianzaJSON = b103.ConfianzaJSON }},
	} {
		t.Run(caso.name, func(t *testing.T) {
			c := ext
			caso.mutar(&c)
			if validar(c) == nil {
				t.Fatal("configuración compartida aceptada")
			}
		})
	}
}
