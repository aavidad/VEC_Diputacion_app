package canonico

import (
	"bytes"
	"encoding/json"
	"testing"
)

func manifiestoV2DesdeV3(v3 ManifiestoPublicoV3) ManifiestoPublicoV2 {
	return ManifiestoPublicoV2{Esquema: EsquemaManifiestoPublicoV2, Fuente: v3.Fuente,
		Catalogos: v3.Catalogos, Categorias: v3.Categorias, Convocatorias: v3.Convocatorias}
}

func proyeccionV2CompletaPrueba(t *testing.T, manifiesto ManifiestoPublicoV2) []byte {
	t.Helper()
	snapshots := make([]any, len(manifiesto.Categorias.Snapshots))
	for i, snapshot := range manifiesto.Categorias.Snapshots {
		snapshots[i] = map[string]any{"catalogo_id": snapshot.Catalogo.CatalogoID, "version": snapshot.Catalogo.Version, "huella_gobernada_sha256": snapshot.HuellaGobernadaSHA256, "huella_proyeccion_publica_sha256": snapshot.HuellaProyeccionSHA256, "categorias": snapshot.Catalogo.Categorias}
	}
	// El detalle no se compacta en CLI: representa todos los grupos que exige
	// publicar_proyeccion_v2 y debe llegar byte por byte a PostgreSQL.
	contenido, err := json.Marshal(map[string]any{
		"fuente": manifiesto.Fuente, "catalogos": manifiesto.Catalogos,
		"categorias":    map[string]any{"actual": manifiesto.Categorias.Actual, "snapshots": snapshots},
		"convocatorias": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return contenido
}

func TestPrepararMaterialPublicacionV3ConservaProyeccionCompletaYReconstruyeAncla(t *testing.T) {
	v3 := manifiestoV3Prueba(t)
	v3.Convocatorias = []ConvocatoriaManifiestoPublicoV2{}
	manifiesto, _ := json.Marshal(manifiestoV2DesdeV3(v3))
	proyeccion := proyeccionV2CompletaPrueba(t, manifiestoV2DesdeV3(v3))
	bolsas, _ := json.Marshal(v3.BolsasV1)
	material, err := PrepararMaterialPublicacionV3(proyeccion, manifiesto, bolsas)
	if err != nil {
		t.Fatalf("preparar material: %v", err)
	}
	esperada, _ := v3.HuellaSHA256()
	if material.AnclaManifiestoSHA256 != esperada || !bytes.Equal(material.ProyeccionV2, proyeccion) || !bytes.Contains(material.ProyeccionV2, []byte(`"snapshots"`)) {
		t.Fatal("el detalle V2 o el ancla no se conservaron")
	}
}

func TestPrepararMaterialPublicacionV3RechazaDuplicadosAliasesNullYFuenteDistinta(t *testing.T) {
	v3 := manifiestoV3Prueba(t)
	v3.Convocatorias = []ConvocatoriaManifiestoPublicoV2{}
	manifiesto, _ := json.Marshal(manifiestoV2DesdeV3(v3))
	proyeccion := proyeccionV2CompletaPrueba(t, manifiestoV2DesdeV3(v3))
	bolsas, _ := json.Marshal(v3.BolsasV1)
	for nombre, mutar := range map[string]func(*[]byte, *[]byte, *[]byte){
		"duplicado": func(p, _, _ *[]byte) { *p = append((*p)[:len(*p)-1], []byte(`,"fuente":{}}`)...) },
		"alias":     func(_, m, _ *[]byte) { *m = bytes.Replace(*m, []byte(`"esquema"`), []byte(`"Esquema"`), 1) },
		"bolsas null": func(_, _, b *[]byte) {
			var bolsas BolsasManifiestoV1
			_ = json.Unmarshal(*b, &bolsas)
			bolsas.Bolsas = nil
			*b, _ = json.Marshal(bolsas)
		},
		"fuente distinta": func(p, _, _ *[]byte) { *p = bytes.Replace(*p, []byte(`revision-001`), []byte(`revision-002`), 1) },
	} {
		t.Run(nombre, func(t *testing.T) {
			p, m, b := append([]byte(nil), proyeccion...), append([]byte(nil), manifiesto...), append([]byte(nil), bolsas...)
			mutar(&p, &m, &b)
			if _, err := PrepararMaterialPublicacionV3(p, m, b); err == nil {
				t.Fatal("material invalido aceptado")
			}
		})
	}
}

func TestPrepararMaterialPublicacionV3ConservaFraccionTemporalParaSQL(t *testing.T) {
	v3 := manifiestoV3Prueba(t)
	v3.Convocatorias = []ConvocatoriaManifiestoPublicoV2{}
	manifiesto, _ := json.Marshal(manifiestoV2DesdeV3(v3))
	proyeccion := bytes.Replace(proyeccionV2CompletaPrueba(t, manifiestoV2DesdeV3(v3)), []byte(`2026-07-22T10:00:00Z`), []byte(`2026-07-22T10:00:00.000000Z`), 1)
	bolsas, _ := json.Marshal(v3.BolsasV1)
	material, err := PrepararMaterialPublicacionV3(proyeccion, manifiesto, bolsas)
	if err != nil || !bytes.Contains(material.BolsasV1, []byte(`"generado_en":"2026-07-22T10:00:00.000000Z"`)) {
		t.Fatalf("instante SQL no coherente: %s; %v", material.BolsasV1, err)
	}
}

func TestPrepararMaterialPublicacionV3TransportaArraysVaciosComoListas(t *testing.T) {
	v3 := manifiestoV3Prueba(t)
	v3.Convocatorias = []ConvocatoriaManifiestoPublicoV2{}
	manifiesto, _ := json.Marshal(manifiestoV2DesdeV3(v3))
	proyeccion := proyeccionV2CompletaPrueba(t, manifiestoV2DesdeV3(v3))
	for nombre, bolsas := range map[string]BolsasManifiestoV1{
		"bolsas":     {GeneradoEn: v3.Fuente.ActualizadaEn, Bolsas: []BolsaManifiestoV1{}},
		"posiciones": {GeneradoEn: v3.Fuente.ActualizadaEn, Bolsas: []BolsaManifiestoV1{{BolsaRef: "bolsa:auxiliar:2026", Categoria: "Auxiliar", CategoriaClave: "auxiliar", Grupos: []string{"C2"}, TipoLista: "definitiva", VigenteDesde: v3.Fuente.ActualizadaEn, Total: 0, Posiciones: []PosicionBolsaManifiestoV1{}}}},
	} {
		t.Run(nombre, func(t *testing.T) {
			entrada, _ := json.Marshal(bolsas)
			material, err := PrepararMaterialPublicacionV3(proyeccion, manifiesto, entrada)
			if err != nil {
				t.Fatal(err)
			}
			esperado := []byte(`"bolsas":[]`)
			if nombre == "posiciones" {
				esperado = []byte(`"posiciones":[]`)
			}
			if !bytes.Contains(material.BolsasV1, esperado) {
				t.Fatalf("transporte sin lista vacia: %s", material.BolsasV1)
			}
		})
	}
}
