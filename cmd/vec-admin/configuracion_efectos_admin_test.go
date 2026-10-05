package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/app/administracion"
)

func efectoADMINPrueba(t *testing.T, c configuracionLotePrivada, audiencia, nombre string) configuracionEfectoPrivada {
	t.Helper()
	var m metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(c.ConfianzaJSON, &m); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(c.PoolLote)
	m.EntradasCapacidad[0].Audiencia = audiencia
	m.EntradasCapacidad[0].MaterialArchivo = filepath.Join(dir, nombre+".bin")
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return configuracionEfectoPrivada{Pool: filepath.Join(dir, nombre+".json"), ConfianzaJSON: b, Motivo: c.MotivoLote}
}

// La publicación de cargos sólo se abre con pool, LOGIN y material propios y
// la capacidad exacta de su audiencia; cualquier cruce con otro pool o
// material (también del lote o del plan) cierra el arranque.
func TestOverlayCargosFallaCerrado(t *testing.T) {
	c, base, u, runtime := loteADMINPrueba(t)
	p := planFirmaADMINPrueba(t, c)
	k := efectoADMINPrueba(t, c, administracion.AudienciaCargoCompetencialV3, "cargos")
	ce := efectoADMINPrueba(t, c, administracion.AudienciaCertificadoNominalV3, "certificados")
	aud := administracion.AudienciaCargoCompetencialV3
	for nombre, x := range map[string]struct {
		lote *configuracionLotePrivada
		plan *configuracionPlanFirmaPrivada
	}{"todo": {&c, &p}, "sin_lote": {nil, &p}, "sin_plan": {&c, nil}, "solo": {nil, nil}} {
		if err := validarConfiguracionEfectoPrivada(k, aud, []configuracionEfectoPrivada{ce}, x.lote, x.plan, base, u, runtime); err != nil {
			t.Fatalf("cargos válidos rechazados (%s): %v", nombre, err)
		}
	}
	b, err := json.Marshal(k)
	if err != nil {
		t.Fatal(err)
	}
	if leida, err := cargarConfiguracionEfectoPrivada(archivoPrivadoPrueba(t, string(b)), aud, nil, &c, &p, base, u, runtime); err != nil || leida.Pool != k.Pool {
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
	for nombre, mutar := range map[string]func(*configuracionEfectoPrivada){
		"pool_del_lote":        func(x *configuracionEfectoPrivada) { x.Pool = c.PoolLote },
		"pool_del_plan":        func(x *configuracionEfectoPrivada) { x.Pool = p.Pool },
		"pool_de_certificados": func(x *configuracionEfectoPrivada) { x.Pool = ce.Pool },
		"material_de_certificados": func(x *configuracionEfectoPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].MaterialArchivo = filepath.Join(filepath.Dir(c.PoolLote), "certificados.bin")
			})
		},
		"audiencia_de_certificados": func(x *configuracionEfectoPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].Audiencia = administracion.AudienciaCertificadoNominalV3
			})
		},
		"pool_de_usuarios":     func(x *configuracionEfectoPrivada) { x.Pool = u.PoolLector },
		"pool_relativo":        func(x *configuracionEfectoPrivada) { x.Pool = "cargos.json" },
		"motivo_otro_catalogo": func(x *configuracionEfectoPrivada) { x.Motivo.CatalogoID = "otro_catalogo" },
		"audiencia_del_plan": func(x *configuracionEfectoPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].Audiencia = administracion.AudienciaGobiernoPlanFirmaV3
			})
		},
		"material_del_plan": func(x *configuracionEfectoPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].MaterialArchivo = mp.EntradasCapacidad[0].MaterialArchivo
			})
		},
		"otra_raiz": func(x *configuracionEfectoPrivada) {
			x.ConfianzaJSON = otra(func(m *metadatosConfianzaPerfilesPrivados) {
				m.Raiz.ClaveID, m.Cabecera.ClaveID = "otra_clave", "otra_clave"
			})
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterado := k
			mutar(&alterado)
			if validarConfiguracionEfectoPrivada(alterado, aud, []configuracionEfectoPrivada{ce}, &c, &p, base, u, runtime) == nil {
				t.Fatalf("%s aceptado", nombre)
			}
		})
	}
}

// La lista de efectos es cerrada: cada uno con su variable, audiencia, grupo,
// ruta y contrato coherentes; sólo los cargos piden UTC.
func TestEfectosADMINCoherentes(t *testing.T) {
	vistos := map[string]bool{}
	for _, e := range efectosADMIN {
		c := e.contrato()
		if vistos[e.variable] || vistos[e.ruta] || vistos[e.grupo] || c.Audiencia != e.audiencia || !c.Valido() ||
			e.maximo < 2 || e.utc != (e.nombre == "cargos") {
			t.Fatalf("efecto incoherente: %+v", e.nombre)
		}
		vistos[e.variable], vistos[e.ruta], vistos[e.grupo] = true, true, true
	}
	if len(efectosADMIN) != 2 {
		t.Fatal("lista de efectos distinta")
	}
}
