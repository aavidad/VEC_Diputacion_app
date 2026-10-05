package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/vec/domain"
)

func loteADMINPrueba(t *testing.T) (configuracionLotePrivada, configuracionPerfilesPrivada, configuracionUsuariosMetadatosPrivada, configuracionRuntimeADMIN) {
	t.Helper()
	base := configuracionPrivadaPrueba(t)
	u := usuariosMetadatosPrueba(t, base)
	u.Destinos["preparar_lote_ordinario"] = destinoUsuariosPrivado{Accion: "administracion.perfiles.preparar_lote_ordinario",
		RecursoRef: "administracion:perfiles:preparar_lote_ordinario", FinalidadRef: "gestion_perfiles", TipoRecurso: "fijo"}
	if err := validarConfiguracionUsuariosMetadatosPrivada(u, base); err != nil {
		t.Fatal("overlay de usuarios con destino de preparación:", err)
	}
	dir := filepath.Dir(base.Pools.CuentasADMIN)
	runtime := configuracionRuntimeADMIN{Version: 1, PoolContexto: filepath.Join(dir, "contexto.json"),
		FuenteIdentificadoresArchivo: filepath.Join(dir, "originales.json"),
		FuenteIdentificadoresSHA256:  strings.Repeat("a", 64), ProcesoContexto: "vec_admin"}
	var confianza metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(base.ConfianzaJSON, &confianza); err != nil {
		t.Fatal(err)
	}
	e := confianza.EntradasCapacidad[0]
	e.Audiencia = administracion.AudienciaLoteOrdinarioV3
	e.MaterialArchivo = filepath.Join(dir, "lote.bin")
	confianza.EntradasCapacidad = []capacidadConfianzaPerfilesPrivada{e}
	b, err := json.Marshal(confianza)
	if err != nil {
		t.Fatal(err)
	}
	fuente := func(ref, letra string) fuenteLotePrivada {
		return fuenteLotePrivada{Referencia: ref, Version: 1, HuellaSHA256: strings.Repeat(letra, 64)}
	}
	c := configuracionLotePrivada{Modo: modoLoteADMIN, PoolLote: filepath.Join(dir, "lote.json"), ConfianzaJSON: b,
		MotivoLote: motivoLotePrueba(u), Unidades: []unidadLotePrivada{{UnidadRef: u.UnidadRef,
			FuenteOrganizacion: fuente("prc_fuente_organizacion", "b"), FuenteUnidad: fuente("fuente:unidad:prueba", "c")}}}
	return c, base, u, runtime
}

// El motivo de la decisión del lote es una clave opaca del catálogo V2.
func motivoLotePrueba(u configuracionUsuariosMetadatosPrivada) domain.ReferenciaEntradaCatalogo {
	m := u.MotivoDenegado
	m.EntradaClave = "motivo_" + strings.Repeat("1", 32)
	return m
}

func TestOverlayLotePrivadoCerradoYConservado(t *testing.T) {
	c, base, u, runtime := loteADMINPrueba(t)
	if err := validarConfiguracionLotePrivada(c, base, u, runtime); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	leida, err := cargarConfiguracionLotePrivada(archivoPrivadoPrueba(t, string(b)), base, u, runtime)
	if err != nil || leida.PoolLote != c.PoolLote || len(leida.Unidades) != 1 {
		t.Fatalf("overlay del lote no conservado: %v", err)
	}
	for _, bruto := range []string{`{}`, `{"modo":"lote_v1","modo":"lote_v1"}`, `{"modo":null}`} {
		if _, err := cargarConfiguracionLotePrivada(archivoPrivadoPrueba(t, bruto), base, u, runtime); err == nil {
			t.Fatal("overlay del lote ambiguo o incompleto aceptado")
		}
	}
}

func TestOverlayLoteFallaCerrado(t *testing.T) {
	c, base, u, runtime := loteADMINPrueba(t)
	otraConfianza := func(mutar func(*metadatosConfianzaPerfilesPrivados)) json.RawMessage {
		var m metadatosConfianzaPerfilesPrivados
		if err := json.Unmarshal(c.ConfianzaJSON, &m); err != nil {
			t.Fatal(err)
		}
		mutar(&m)
		b, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	sinDestino := u
	sinDestino.Destinos = map[string]destinoUsuariosPrivado{}
	for k, v := range u.Destinos {
		if k != "preparar_lote_ordinario" {
			sinDestino.Destinos[k] = v
		}
	}
	if validarConfiguracionLotePrivada(c, base, sinDestino, runtime) == nil {
		t.Fatal("lote sin destino de auditoría de la preparación aceptado")
	}
	for nombre, mutar := range map[string]func(*configuracionLotePrivada){
		"modo":                 func(x *configuracionLotePrivada) { x.Modo = "metadatos_v1" },
		"pool_de_usuarios":     func(x *configuracionLotePrivada) { x.PoolLote = u.PoolLector },
		"pool_de_contexto":     func(x *configuracionLotePrivada) { x.PoolLote = runtime.PoolContexto },
		"pool_de_actos":        func(x *configuracionLotePrivada) { x.PoolLote = base.Pools.CuentasADMIN },
		"pool_relativo":        func(x *configuracionLotePrivada) { x.PoolLote = "lote.json" },
		"motivo_otro_catalogo": func(x *configuracionLotePrivada) { x.MotivoLote.CatalogoID = "otro_catalogo" },
		"motivo_no_opaco":      func(x *configuracionLotePrivada) { x.MotivoLote.EntradaClave = "lectura" },
		"sin_unidades":         func(x *configuracionLotePrivada) { x.Unidades = nil },
		"unidad_repetida":      func(x *configuracionLotePrivada) { x.Unidades = append(x.Unidades, x.Unidades[0]) },
		"fuente_sin_version":   func(x *configuracionLotePrivada) { x.Unidades[0].FuenteUnidad.Version = 0 },
		"otra_audiencia": func(x *configuracionLotePrivada) {
			x.ConfianzaJSON = otraConfianza(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].Audiencia = administracion.AudienciaUsuariosListarV3
			})
		},
		"dos_capacidades": func(x *configuracionLotePrivada) {
			x.ConfianzaJSON = otraConfianza(func(m *metadatosConfianzaPerfilesPrivados) {
				f := m.EntradasCapacidad[0]
				f.Audiencia, f.MaterialArchivo = administracion.AudienciaUsuariosListarV3, filepath.Join(filepath.Dir(f.MaterialArchivo), "otra.bin")
				m.EntradasCapacidad = append(m.EntradasCapacidad, f)
			})
		},
		"material_compartido": func(x *configuracionLotePrivada) {
			var mu metadatosConfianzaPerfilesPrivados
			if err := json.Unmarshal(u.ConfianzaJSON, &mu); err != nil {
				t.Fatal(err)
			}
			x.ConfianzaJSON = otraConfianza(func(m *metadatosConfianzaPerfilesPrivados) {
				m.EntradasCapacidad[0].MaterialArchivo = mu.EntradasCapacidad[0].MaterialArchivo
			})
		},
		"otra_raiz": func(x *configuracionLotePrivada) {
			x.ConfianzaJSON = otraConfianza(func(m *metadatosConfianzaPerfilesPrivados) {
				m.Raiz.ClaveID = "otra_clave"
				m.Cabecera.ClaveID = "otra_clave"
			})
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			alterada := c
			alterada.Unidades = append([]unidadLotePrivada(nil), c.Unidades...)
			mutar(&alterada)
			if validarConfiguracionLotePrivada(alterada, base, u, runtime) == nil {
				t.Fatal("overlay del lote inválido aceptado")
			}
		})
	}
}
