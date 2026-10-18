package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestVersionBolsaB103OverlaySeparaPoolsYMaterialDeB1(t *testing.T) {
	gobierno, base, u, runtime, lote, plan := gobiernoRolesConfigPrueba(t)
	var meta metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(gobierno.ConfianzaJSON, &meta); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(gobierno.PoolGobierno)
	motivo := motivoLotePrueba(u)
	construir := func(nombre string) configuracionGobiernoRolesPrivada {
		m := meta
		m.EntradasCapacidad = append([]capacidadConfianzaPerfilesPrivada(nil), meta.EntradasCapacidad...)
		m.EntradasCapacidad[0].Audiencia = pg.AudienciaVersionarRolBolsaProponer
		m.EntradasCapacidad[0].MaterialArchivo = filepath.Join(dir, nombre+"-proponer.bin")
		m.EntradasCapacidad[1].Audiencia = pg.AudienciaVersionarRolBolsaAprobar
		m.EntradasCapacidad[1].MaterialArchivo = filepath.Join(dir, nombre+"-aprobar.bin")
		confianza, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return configuracionGobiernoRolesPrivada{
			PoolGobierno:  filepath.Join(dir, nombre+"-ejecutor.json"),
			PoolCatalogo:  filepath.Join(dir, nombre+"-catalogo.json"),
			ConfianzaJSON: confianza,
			Motivos: map[string]domain.ReferenciaEntradaCatalogo{
				pg.AudienciaVersionarRolBolsaProponer: motivo,
				pg.AudienciaVersionarRolBolsaAprobar:  motivo,
			},
		}
	}
	b1 := configuracionVersionBolsaPrivada(construir("b1"))
	b103 := configuracionVersionBolsaB103Privada(construir("b103"))
	validar := func(c configuracionVersionBolsaB103Privada) error {
		return validarConfiguracionVersionBolsaB103Privada(c, base, u, runtime, &lote, &plan, nil, &gobierno, &b1)
	}
	if err := validar(b103); err != nil {
		t.Fatalf("overlay B103 segregado rechazado: %v", err)
	}
	if err := validarConfiguracionVersionBolsaB103Privada(b103, base, u, runtime,
		&lote, &plan, nil, nil, nil); err != nil {
		t.Fatalf("overlay B103 independiente de B1 rechazado: %v", err)
	}
	compartido := b103
	compartido.PoolGobierno = b1.PoolGobierno
	if validar(compartido) == nil {
		t.Fatal("B103 acepta DSN ejecutor B1")
	}
	compartido = b103
	compartido.PoolCatalogo = b1.PoolCatalogo
	if validar(compartido) == nil {
		t.Fatal("B103 acepta DSN lector B1")
	}
	compartido = b103
	compartido.ConfianzaJSON = b1.ConfianzaJSON
	if validar(compartido) == nil {
		t.Fatal("B103 acepta materiales V3 B1")
	}
}
