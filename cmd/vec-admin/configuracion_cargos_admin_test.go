package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/app/administracion"
)

func cargosADMINPrueba(t *testing.T, c configuracionLotePrivada) configuracionCargosPrivada {
	t.Helper()
	var m metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(c.ConfianzaJSON, &m); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(c.PoolLote)
	m.EntradasCapacidad[0].Audiencia = administracion.AudienciaCargoCompetencialV3
	m.EntradasCapacidad[0].MaterialArchivo = filepath.Join(dir, "cargos.bin")
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return configuracionCargosPrivada{Pool: filepath.Join(dir, "cargos.json"), ConfianzaJSON: b, Motivo: c.MotivoLote}
}

// La publicación de cargos sólo se abre con pool, LOGIN y material propios y
// la capacidad exacta de su audiencia; cualquier cruce con otro pool o
// material (también del lote o del plan) cierra el arranque.
func TestOverlayCargosFallaCerrado(t *testing.T) {
	c, base, u, runtime := loteADMINPrueba(t)
	p := planFirmaADMINPrueba(t, c)
	k := cargosADMINPrueba(t, c)
	for nombre, x := range map[string]struct {
		lote *configuracionLotePrivada
		plan *configuracionPlanFirmaPrivada
	}{"todo": {&c, &p}, "sin_lote": {nil, &p}, "sin_plan": {&c, nil}, "solo": {nil, nil}} {
		if err := validarConfiguracionCargosPrivada(k, x.lote, x.plan, base, u, runtime); err != nil {
			t.Fatalf("cargos válidos rechazados (%s): %v", nombre, err)
		}
	}
	b, err := json.Marshal(k)
	if err != nil {
		t.Fatal(err)
	}
	if leida, err := cargarConfiguracionCargosPrivada(archivoPrivadoPrueba(t, string(b)), &c, &p, base, u, runtime); err != nil || leida.Pool != k.Pool {
		t.Fatalf("cargos no conservados: %v", err)
	}
	otra := func(mutar func(*metadatosConfianzaPerfilesPrivados)) json.RawMessage {
		var m metadatosConfianzaPerfilesPrivados
		if err := json.Unmarshal(k.ConfianzaJSON, &m); err != nil {
			t.Fatal(err)
		}
		mutar(&m)
		b, _ := json.Marshal(m)
		return b
	}
	var mp metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(p.ConfianzaJSON, &mp); err != nil {
		t.Fatal(err)
	}
	for nombre, mutar := range map[string]func(*configuracionCargosPrivada){
		"pool_del_lote":        func(x *configuracionCargosPrivada) { x.Pool = c.PoolLote },
		"pool_del_plan":        func(x *configuracionCargosPrivada) { x.Pool = p.Pool },
		"pool_de_usuarios":     func(x *configuracionCargosPrivada) { x.Pool = u.PoolLector },
		"pool_relativo":        func(x *configuracionCargosPrivada) { x.Pool = "cargos.json" },
		"motivo_otro_catalogo": func(x *configuracionCargosPrivada) { x.Motivo.CatalogoID = "otro_catalogo" },
		"audiencia_del_plan": func(x *configuracionCargosPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].Audiencia = administracion.AudienciaGobiernoPlanFirmaV3
			})
		},
		"material_del_plan": func(x *configuracionCargosPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].MaterialArchivo = mp.EntradasCapacidad[0].MaterialArchivo
			})
		},
		"otra_raiz": func(x *configuracionCargosPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.Raiz.ClaveID, m.Cabecera.ClaveID = "otra_clave", "otra_clave"
			})
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := k
			mutar(&alterado)
			if validarConfiguracionCargosPrivada(alterado, &c, &p, base, u, runtime) == nil {
				t.Fatalf("%s aceptado", nombre)
			}
		})
	}
}
