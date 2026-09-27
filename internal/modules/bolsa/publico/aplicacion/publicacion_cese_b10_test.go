package aplicacion

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
)

type feedCeseB10Prueba struct {
	evento       CesePendienteB10
	confirmada   bool
	ancla        string
	confirmacion int
}

func (f *feedCeseB10Prueba) SiguientePublicacionCeseB10(context.Context) (CesePendienteB10, bool, error) {
	return f.evento, !f.confirmada, nil
}

func (f *feedCeseB10Prueba) ConfirmarPublicacionCeseB10(_ context.Context, evento CesePendienteB10, ancla string) (bool, error) {
	if evento != f.evento || ancla == "" {
		return false, errors.New("confirmacion incorrecta")
	}
	f.confirmada = true
	f.ancla = ancla
	f.confirmacion++
	return false, nil
}

type instantaneaCeseB10Prueba struct{ InstantaneaCeseB10 }

func (f instantaneaCeseB10Prueba) PrepararInstantaneaCeseB10(context.Context, CesePendienteB10) (InstantaneaCeseB10, error) {
	return f.InstantaneaCeseB10, nil
}

func materialCeseB10Prueba(t *testing.T, evento CesePendienteB10) InstantaneaCeseB10 {
	t.Helper()
	fecha := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	catalogo, err := canonico.NuevoCatalogoCategoriasV1("categorias", 1, []canonico.CategoriaCatalogoV1{{
		Clave: "auxiliar", Etiqueta: "Auxiliar", Descripcion: "Categoría auxiliar.",
		Semantica: "informacion", Orden: 1, Area: "administracion", AreaEtiqueta: "Administración",
		Suscribible: true, VigenteDesde: fecha,
	}})
	if err != nil {
		t.Fatal(err)
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto := canonico.ManifiestoPublicoV2{
		Esquema:   canonico.EsquemaManifiestoPublicoV2,
		Fuente:    canonico.FuenteManifiestoPublicoV2{Revision: "revision-001", ActualizadaEn: fecha},
		Catalogos: []canonico.CatalogoManifiestoV2{{Referencia: "bolsa", Version: 1, Entradas: []canonico.EntradaCatalogoManifiestoV2{{Clave: "estado", Etiqueta: "Estado", Descripcion: "Estado público", Semantica: "estado", Orden: 1}}}},
		Categorias: canonico.CategoriasManifiestoPublicoV2{
			Actual:    canonico.ReferenciaCatalogoCategoriasManifiestoV2{CatalogoID: catalogo.CatalogoID, CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: strings.Repeat("a", 64), CatalogoHuellaProyeccionSHA256: huella},
			Snapshots: []canonico.SnapshotCategoriasManifiestoV2{{HuellaGobernadaSHA256: strings.Repeat("a", 64), HuellaProyeccionSHA256: huella, Catalogo: catalogo}},
		},
		Convocatorias: []canonico.ConvocatoriaManifiestoPublicoV2{},
	}
	manifiestoJSON, err := json.Marshal(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	snapshots := []any{map[string]any{"catalogo_id": catalogo.CatalogoID, "version": catalogo.Version, "huella_gobernada_sha256": strings.Repeat("a", 64), "huella_proyeccion_publica_sha256": huella, "categorias": catalogo.Categorias}}
	proyeccionJSON, err := json.Marshal(map[string]any{
		"fuente": manifiesto.Fuente, "catalogos": manifiesto.Catalogos,
		"categorias":    map[string]any{"actual": manifiesto.Categorias.Actual, "snapshots": snapshots},
		"convocatorias": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	bolsasJSON, err := json.Marshal(canonico.BolsasManifiestoV1{
		GeneradoEn: fecha, Bolsas: []canonico.BolsaManifiestoV1{{
			BolsaRef: "bolsa:auxiliar:2026", Categoria: "Auxiliar", CategoriaClave: "auxiliar",
			Grupos: []string{"C2"}, TipoLista: "definitiva", VigenteDesde: fecha,
			Total: 1, Posiciones: []canonico.PosicionBolsaManifiestoV1{{Orden: 1, DocumentoEnmascarado: "***1234**", EstadoClave: "no_disponible"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return InstantaneaCeseB10{OrigenPosicionIncluida: evento.OrigenPosicion, OrigenRefIncluido: evento.OrigenRef, FaseIncluida: evento.Fase,
		ProyeccionV2: proyeccionJSON, ManifiestoV2: manifiestoJSON, BolsasV1: bolsasJSON}
}

func TestPublicarSiguienteCeseB10RecuperaDespuesDeCommitPublico(t *testing.T) {
	evento := CesePendienteB10{OrigenPosicion: 17, OrigenRef: "origen:ct:17", EventoRef: "evento:ct:contrato-bolsa:" + strings.Repeat("a", 64), BolsaRef: "bolsa:auxiliar:2026", Fase: "cese"}
	feed := &feedCeseB10Prueba{evento: evento}
	instantanea := instantaneaCeseB10Prueba{materialCeseB10Prueba(t, evento)}
	publicaciones := 0
	var anclaPublicada string
	publicar := func(_ context.Context, _, bolsas []byte, ancla string) error {
		publicaciones++
		if !strings.Contains(string(bolsas), `"estado_clave":"no_disponible"`) || strings.Contains(string(bolsas), evento.OrigenRef) {
			t.Fatal("material público no minimizado")
		}
		anclaPublicada = ancla
		if publicaciones == 1 {
			return errors.New("commit público incierto")
		}
		return nil
	}
	if _, _, err := PublicarSiguienteCeseB10(context.Background(), feed, instantanea, publicar); err == nil || feed.confirmada {
		t.Fatal("se confirmó B49 sin éxito del destino")
	}
	recibo, ok, err := PublicarSiguienteCeseB10(context.Background(), feed, instantanea, publicar)
	if err != nil || !ok || publicaciones != 2 || feed.confirmacion != 1 || recibo.AnclaSHA256 != anclaPublicada || feed.ancla != anclaPublicada {
		t.Fatalf("replay perdido: ok=%t publicaciones=%d confirmaciones=%d err=%v", ok, publicaciones, feed.confirmacion, err)
	}
	if _, ok, err := PublicarSiguienteCeseB10(context.Background(), feed, instantanea, publicar); err != nil || ok || publicaciones != 2 {
		t.Fatalf("duplicado después de confirmar: ok=%t publicaciones=%d err=%v", ok, publicaciones, err)
	}
}

func TestPublicarSiguienteCeseB10RechazaInstantaneaAtrasada(t *testing.T) {
	evento := CesePendienteB10{OrigenPosicion: 17, OrigenRef: "origen:ct:17", EventoRef: "evento:ct:contrato-bolsa:" + strings.Repeat("a", 64), BolsaRef: "bolsa:auxiliar:2026", Fase: "cese"}
	feed := &feedCeseB10Prueba{evento: evento}
	instantanea := materialCeseB10Prueba(t, evento)
	instantanea.OrigenPosicionIncluida--
	llamado := false
	_, _, err := PublicarSiguienteCeseB10(context.Background(), feed, instantaneaCeseB10Prueba{instantanea}, func(context.Context, []byte, []byte, string) error {
		llamado = true
		return nil
	})
	if !errors.Is(err, ErrPublicacionCeseB10NoDisponible) || llamado || feed.confirmada {
		t.Fatal("instantánea anterior aceptada")
	}
}
