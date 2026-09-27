package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	publicacion "vec-diputacion-granada/internal/modules/bolsa/publico/aplicacion"
	"vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
)

type capturadorCeseB10Prueba struct{ captura CapturaFuenteCeseB10 }

func (c capturadorCeseB10Prueba) CapturarCeseB10(context.Context, publicacion.CesePendienteB10) (CapturaFuenteCeseB10, error) {
	return c.captura, nil
}

func TestFuenteCeseB10ProyectaTodasLasParticipacionesConEstadoEfectivoSinIdentificadores(t *testing.T) {
	corte := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	desde := corte.Add(24 * time.Hour)
	captura := CapturaFuenteCeseB10{
		BolsaRef: "bolsa:administrativo:2026", ParticipacionOrigen: "participacion:segunda", Corte: corte,
		Bolsas: []BolsaFuenteCeseB10{{
			BolsaRef: "bolsa:administrativo:2026", Categoria: "Administrativo", CategoriaClave: "administrativo",
			Grupos: []string{"C1"}, TipoLista: "cerrada", VigenteDesde: corte.Add(-time.Hour), Total: 2,
			Participaciones: []ParticipacionFuenteCeseB10{
				{ParticipacionRef: "participacion:segunda", Orden: 2, Documento: "***5678**", EstadoEfectivo: "disponible_desde", FechaDisponible: &desde},
				{ParticipacionRef: "participacion:primera", Orden: 1, Documento: "***1234**", EstadoEfectivo: "trabajando"},
			},
		}},
	}
	bolsas, err := proyectarBolsasCeseB10(captura)
	if err != nil || len(bolsas.Bolsas) != 1 || bolsas.Bolsas[0].Total != 2 ||
		bolsas.Bolsas[0].Posiciones[0].EstadoClave != "ocupado" ||
		bolsas.Bolsas[0].Posiciones[1].EstadoClave != "no_disponible" {
		t.Fatalf("proyección B45: %+v; %v", bolsas, err)
	}
	contenido, err := json.Marshal(bolsas)
	if err != nil || strings.Contains(string(contenido), "participacion:") || strings.Contains(string(contenido), "Nombre") {
		t.Fatalf("material público con referencia privada: %s; %v", contenido, err)
	}
	captura.ParticipacionOrigen = "participacion:ajena"
	if _, err := proyectarBolsasCeseB10(captura); !errors.Is(err, ErrFuenteCeseB10NoDisponible) {
		t.Fatalf("origen no incluido aceptado: %v", err)
	}
}

func TestFuenteCeseB10RechazaCapturaSinAutoridadExacta(t *testing.T) {
	if _, err := NuevaFuenteInstantaneaCeseB10(nil); !errors.Is(err, ErrFuenteCeseB10NoDisponible) {
		t.Fatalf("sin capturador: %v", err)
	}
	evento := publicacion.CesePendienteB10{
		OrigenPosicion: 7, OrigenRef: "evento:ct:contrato-bolsa:" + strings.Repeat("a", 64),
		EventoRef: "evento:ct:contrato-bolsa:" + strings.Repeat("a", 64), BolsaRef: "bolsa:administrativo:2026", Fase: "cese",
	}
	fuente, err := NuevaFuenteInstantaneaCeseB10(capturadorCeseB10Prueba{captura: CapturaFuenteCeseB10{
		OrigenPosicion: 7, OrigenRef: evento.OrigenRef, EventoRef: "evento:distinto", Fase: "cese", BolsaRef: evento.BolsaRef,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fuente.PrepararInstantaneaCeseB10(context.Background(), evento); !errors.Is(err, ErrFuenteCeseB10NoDisponible) {
		t.Fatalf("evento diferente aceptado: %v", err)
	}
}

func TestFuenteCeseB10DisponibilidadUsaMedianocheDeGranadaEnUTC(t *testing.T) {
	for _, inicio := range []time.Time{
		time.Date(2026, 6, 1, 22, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 1, 23, 0, 0, 0, time.UTC),
	} {
		participacion := ParticipacionFuenteCeseB10{EstadoEfectivo: "disponible_desde", FechaDisponible: &inicio}
		antes, err := estadoEfectivoPublicoCeseB10(participacion, inicio.Add(-time.Microsecond))
		if err != nil || antes != "no_disponible" {
			t.Fatalf("antes de medianoche civil %s: %s; %v", inicio, antes, err)
		}
		desde, err := estadoEfectivoPublicoCeseB10(participacion, inicio)
		if err != nil || desde != "disponible" {
			t.Fatalf("en medianoche civil %s: %s; %v", inicio, desde, err)
		}
	}
}

func TestFuenteCeseB10PrepararConV2GobernadoConservaReplay(t *testing.T) {
	corte := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	categoria := canonico.CategoriaCatalogoV1{
		Clave: "administrativo", Etiqueta: "Administrativo", Descripcion: "Administración.",
		Semantica: "informacion", Orden: 1, Area: "administracion", AreaEtiqueta: "Administración",
		VigenteDesde: corte.Add(-time.Hour),
	}
	catalogo, err := canonico.NuevoCatalogoCategoriasV1("categorias-publicas", 1, []canonico.CategoriaCatalogoV1{categoria})
	if err != nil {
		t.Fatal(err)
	}
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto := canonico.ManifiestoPublicoV2{
		Esquema: canonico.EsquemaManifiestoPublicoV2,
		Fuente:  canonico.FuenteManifiestoPublicoV2{Revision: "revision-b10-cese", ActualizadaEn: corte},
		Catalogos: []canonico.CatalogoManifiestoV2{{
			Referencia: "tipos_convocatoria", Version: 1,
			Entradas: []canonico.EntradaCatalogoManifiestoV2{{
				Clave: "bolsa_temporal", Etiqueta: "Bolsa temporal", Descripcion: "Bolsa.", Semantica: "informacion", Orden: 1,
			}},
		}},
		Categorias: canonico.CategoriasManifiestoPublicoV2{
			Actual: canonico.ReferenciaCatalogoCategoriasManifiestoV2{
				CatalogoID: catalogo.CatalogoID, CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("a", 64), CatalogoHuellaProyeccionSHA256: huella,
			},
			Snapshots: []canonico.SnapshotCategoriasManifiestoV2{{
				HuellaGobernadaSHA256: strings.Repeat("a", 64), HuellaProyeccionSHA256: huella, Catalogo: catalogo,
			}},
		},
		Convocatorias: []canonico.ConvocatoriaManifiestoPublicoV2{},
	}
	if err := manifiesto.Validar(); err != nil {
		t.Fatal(err)
	}
	manifiestoBytes, err := json.Marshal(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	proyeccion, err := json.Marshal(map[string]any{
		"fuente": manifiesto.Fuente, "catalogos": manifiesto.Catalogos,
		"categorias": map[string]any{"actual": manifiesto.Categorias.Actual,
			"snapshots": []any{map[string]any{
				"catalogo_id": catalogo.CatalogoID, "version": 1,
				"huella_gobernada_sha256":          strings.Repeat("a", 64),
				"huella_proyeccion_publica_sha256": huella, "categorias": catalogo.Categorias,
			}},
		},
		"convocatorias": []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	evento := publicacion.CesePendienteB10{
		OrigenPosicion: 3, OrigenRef: "evento:ct:contrato-bolsa:" + strings.Repeat("b", 64),
		EventoRef: "evento:ct:contrato-bolsa:" + strings.Repeat("c", 64),
		BolsaRef:  "bolsa:administrativo:2026", Fase: "cese",
	}
	captura := CapturaFuenteCeseB10{
		OrigenPosicion: evento.OrigenPosicion, OrigenRef: evento.OrigenRef, EventoRef: evento.EventoRef,
		Fase: evento.Fase, BolsaRef: evento.BolsaRef, ParticipacionOrigen: "participacion:origen",
		Corte: corte, AnteriorActualizadaEn: corte.Add(-time.Microsecond),
		ProyeccionV2: proyeccion, ManifiestoV2: manifiestoBytes,
		Bolsas: []BolsaFuenteCeseB10{{
			BolsaRef: evento.BolsaRef, Categoria: "Administrativo", CategoriaClave: "administrativo",
			Grupos: []string{"C1"}, TipoLista: "cerrada", VigenteDesde: corte.Add(-time.Hour), Total: 1,
			Participaciones: []ParticipacionFuenteCeseB10{{ParticipacionRef: "participacion:origen", Orden: 1,
				Documento: "***1234**", EstadoEfectivo: "disponible"}},
		}},
	}
	fuente, err := NuevaFuenteInstantaneaCeseB10(capturadorCeseB10Prueba{captura})
	if err != nil {
		t.Fatal(err)
	}
	primera, err := fuente.PrepararInstantaneaCeseB10(context.Background(), evento)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := fuente.PrepararInstantaneaCeseB10(context.Background(), evento)
	if err != nil || !bytes.Equal(primera.ProyeccionV2, segunda.ProyeccionV2) ||
		!bytes.Equal(primera.BolsasV1, segunda.BolsasV1) || !bytes.Equal(primera.ManifiestoV2, segunda.ManifiestoV2) ||
		primera.OrigenPosicionIncluida != evento.OrigenPosicion || primera.OrigenRefIncluido != evento.OrigenRef {
		t.Fatalf("material o cursor distinto en replay: %v", err)
	}
	if strings.Contains(string(primera.BolsasV1), "participacion:origen") {
		t.Fatal("referencia privada en B10")
	}
}
