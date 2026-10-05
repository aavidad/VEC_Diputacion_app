package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/app/administracion"
)

func planFirmaADMINPrueba(t *testing.T, c configuracionLotePrivada) configuracionPlanFirmaPrivada {
	t.Helper()
	var m metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(c.ConfianzaJSON, &m); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(c.PoolLote)
	m.EntradasCapacidad[0].Audiencia = administracion.AudienciaGobiernoPlanFirmaV3
	m.EntradasCapacidad[0].MaterialArchivo = filepath.Join(dir, "plan-firma.bin")
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return configuracionPlanFirmaPrivada{Pool: filepath.Join(dir, "plan-firma.json"), ConfianzaJSON: b, Motivo: c.MotivoLote}
}

// El gobierno del plan sólo se abre con pool, LOGIN y material propios y la
// capacidad exacta de su audiencia; cualquier cruce con otro pool o material
// cierra el arranque.
func TestOverlayPlanFirmaFallaCerrado(t *testing.T) {
	c, base, u, runtime := loteADMINPrueba(t)
	p := planFirmaADMINPrueba(t, c)
	if err := validarConfiguracionPlanFirmaPrivada(p, c, base, u, runtime); err != nil {
		t.Fatalf("gobierno del plan válido rechazado: %v", err)
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if leida, err := cargarConfiguracionPlanFirmaPrivada(archivoPrivadoPrueba(t, string(b)), c, base, u, runtime); err != nil || leida.Pool != p.Pool {
		t.Fatalf("gobierno del plan no conservado: %v", err)
	}
	otra := func(mutar func(*metadatosConfianzaPerfilesPrivados)) json.RawMessage {
		var m metadatosConfianzaPerfilesPrivados
		if err := json.Unmarshal(p.ConfianzaJSON, &m); err != nil {
			t.Fatal(err)
		}
		mutar(&m)
		b, _ := json.Marshal(m)
		return b
	}
	var ml metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(c.ConfianzaJSON, &ml); err != nil {
		t.Fatal(err)
	}
	for nombre, mutar := range map[string]func(*configuracionPlanFirmaPrivada){
		"pool_del_lote":        func(x *configuracionPlanFirmaPrivada) { x.Pool = c.PoolLote },
		"pool_de_usuarios":     func(x *configuracionPlanFirmaPrivada) { x.Pool = u.PoolLector },
		"pool_relativo":        func(x *configuracionPlanFirmaPrivada) { x.Pool = "plan.json" },
		"motivo_otro_catalogo": func(x *configuracionPlanFirmaPrivada) { x.Motivo.CatalogoID = "otro_catalogo" },
		"audiencia_del_lote": func(x *configuracionPlanFirmaPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].Audiencia = administracion.AudienciaLoteOrdinarioV3
			})
		},
		"material_del_lote": func(x *configuracionPlanFirmaPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].MaterialArchivo = ml.EntradasCapacidad[0].MaterialArchivo
			})
		},
		"otra_raiz": func(x *configuracionPlanFirmaPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.Raiz.ClaveID, m.Cabecera.ClaveID = "otra_clave", "otra_clave"
			})
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := p
			mutar(&alterado)
			if validarConfiguracionPlanFirmaPrivada(alterado, c, base, u, runtime) == nil {
				t.Fatalf("%s aceptado", nombre)
			}
		})
	}
}
