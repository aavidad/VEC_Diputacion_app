package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	canonicopublico "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
)

func materialPublicacionPrueba(t *testing.T) ([]byte, []byte, []byte) {
	t.Helper()
	instante := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	catalogo, err := canonicopublico.NuevoCatalogoCategoriasV1("categorias", 1, []canonicopublico.CategoriaCatalogoV1{{
		Clave: "administrativo", Etiqueta: "Administrativo", Descripcion: "Categoría administrativa.",
		Semantica: "informacion", Orden: 1, Area: "administracion", AreaEtiqueta: "Administración",
		Suscribible: true, VigenteDesde: instante,
	}})
	if err != nil {
		t.Fatal(err)
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto := canonicopublico.ManifiestoPublicoV2{
		Esquema:   canonicopublico.EsquemaManifiestoPublicoV2,
		Fuente:    canonicopublico.FuenteManifiestoPublicoV2{Revision: "revision-001", ActualizadaEn: instante},
		Catalogos: []canonicopublico.CatalogoManifiestoV2{{Referencia: "bolsa", Version: 1, Entradas: []canonicopublico.EntradaCatalogoManifiestoV2{{Clave: "estado", Etiqueta: "Estado", Descripcion: "Estado publico", Semantica: "estado", Orden: 1}}}},
		Categorias: canonicopublico.CategoriasManifiestoPublicoV2{
			Actual:    canonicopublico.ReferenciaCatalogoCategoriasManifiestoV2{CatalogoID: catalogo.CatalogoID, CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: strings.Repeat("a", 64), CatalogoHuellaProyeccionSHA256: huella},
			Snapshots: []canonicopublico.SnapshotCategoriasManifiestoV2{{HuellaGobernadaSHA256: strings.Repeat("a", 64), HuellaProyeccionSHA256: huella, Catalogo: catalogo}},
		},
	}
	contenido, err := json.Marshal(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	var v2 map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &v2); err != nil {
		t.Fatal(err)
	}
	delete(v2, "esquema")
	snapshots := make([]any, len(manifiesto.Categorias.Snapshots))
	for i, snapshot := range manifiesto.Categorias.Snapshots {
		snapshots[i] = map[string]any{"catalogo_id": snapshot.Catalogo.CatalogoID, "version": snapshot.Catalogo.Version, "huella_gobernada_sha256": snapshot.HuellaGobernadaSHA256, "huella_proyeccion_publica_sha256": snapshot.HuellaProyeccionSHA256, "categorias": snapshot.Catalogo.Categorias}
	}
	categoriasV2, err := json.Marshal(map[string]any{"actual": manifiesto.Categorias.Actual, "snapshots": snapshots})
	if err != nil {
		t.Fatal(err)
	}
	v2["categorias"] = categoriasV2
	v2["convocatorias"] = json.RawMessage(`[]`)
	contenido, err = json.Marshal(v2)
	if err != nil {
		t.Fatal(err)
	}
	bolsas, err := json.Marshal(canonicopublico.BolsasManifiestoV1{GeneradoEn: instante, Bolsas: []canonicopublico.BolsaManifiestoV1{}})
	if err != nil {
		t.Fatal(err)
	}
	manifiestoJSON, err := json.Marshal(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	return contenido, manifiestoJSON, bolsas
}

func configuracionPublicadoraPrueba(t *testing.T) config.Config {
	t.Helper()
	publica, err := config.NuevaConfiguracionPostgreSQLPublica("postgres://usuario:secreto@localhost/publica?sslmode=verify-full")
	if err != nil {
		t.Fatal(err)
	}
	return config.Config{BolsaPublicaPostgreSQL: publica}
}

func TestEjecutarPublicacionProyeccionPublicaPublicaSoloAncla(t *testing.T) {
	directorio := t.TempDir()
	v2, manifiesto, bolsas := materialPublicacionPrueba(t)
	rutaV2, rutaManifiesto, rutaBolsas := filepath.Join(directorio, "v2.json"), filepath.Join(directorio, "manifiesto.json"), filepath.Join(directorio, "b10.json")
	if err := os.WriteFile(rutaV2, v2, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rutaManifiesto, manifiesto, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rutaBolsas, bolsas, 0600); err != nil {
		t.Fatal(err)
	}
	esperado, err := canonicopublico.PrepararMaterialPublicacionV3(v2, manifiesto, bolsas)
	if err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	llamado := false
	err = ejecutarPublicacionProyeccionPublica(context.Background(), []string{"--proyeccion-v2", rutaV2, "--manifiesto-v2", rutaManifiesto, "--bolsas-v1", rutaBolsas}, &salida, configuracionPublicadoraPrueba(t), func(_ context.Context, dsn string, recibidoV2, recibidoB10 []byte, ancla string) error {
		llamado = true
		if dsn == "" || !bytes.Equal(recibidoV2, esperado.ProyeccionV2) || !bytes.Equal(recibidoB10, esperado.BolsasV1) || ancla != esperado.AnclaManifiestoSHA256 {
			t.Fatal("material publicado inesperado")
		}
		return nil
	})
	if err != nil || !llamado || salida.String() != "publicacion_publica ancla="+esperado.AnclaManifiestoSHA256+"\n" {
		t.Fatalf("resultado: llamado=%t salida=%q err=%v", llamado, salida.String(), err)
	}
}

func TestEjecutarPublicacionProyeccionPublicaRechazaSinInvocarNiFiltrar(t *testing.T) {
	directorio := t.TempDir()
	rutaV2, rutaManifiesto, rutaBolsas := filepath.Join(directorio, "secreto-v2.json"), filepath.Join(directorio, "secreto-manifiesto.json"), filepath.Join(directorio, "secreto-b10.json")
	if err := os.WriteFile(rutaV2, []byte(`{"ajeno":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rutaManifiesto, []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rutaBolsas, []byte(`{"bolsas":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	llamado := false
	err := ejecutarPublicacionProyeccionPublica(context.Background(), []string{"--proyeccion-v2", rutaV2, "--manifiesto-v2", rutaManifiesto, "--bolsas-v1", rutaBolsas}, &salida, configuracionPublicadoraPrueba(t), func(context.Context, string, []byte, []byte, string) error {
		llamado = true
		return errors.New("no debe llamarse")
	})
	if !errors.Is(err, errPublicacionPublicaCLI) || llamado || salida.Len() != 0 || strings.Contains(err.Error(), directorio) || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("rechazo inseguro: llamado=%t salida=%q err=%v", llamado, salida.String(), err)
	}
}

func TestEjecutarPublicacionProyeccionPublicaRedactaFalloDelPublicador(t *testing.T) {
	directorio := t.TempDir()
	v2, manifiesto, bolsas := materialPublicacionPrueba(t)
	rutaV2, rutaManifiesto, rutaBolsas := filepath.Join(directorio, "v2.json"), filepath.Join(directorio, "manifiesto.json"), filepath.Join(directorio, "b10.json")
	if err := os.WriteFile(rutaV2, v2, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rutaManifiesto, manifiesto, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rutaBolsas, bolsas, 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	err := ejecutarPublicacionProyeccionPublica(context.Background(), []string{"--proyeccion-v2", rutaV2, "--manifiesto-v2", rutaManifiesto, "--bolsas-v1", rutaBolsas}, &salida, configuracionPublicadoraPrueba(t), func(context.Context, string, []byte, []byte, string) error {
		return errors.New("postgres://usuario:secreto@servidor/publica")
	})
	if !errors.Is(err, errPublicacionPublicaCLI) || salida.Len() != 0 || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("fallo no redactado: salida=%q err=%v", salida.String(), err)
	}
}
