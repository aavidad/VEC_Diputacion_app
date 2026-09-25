package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestDecodificarOfertaDeLaProyeccionSQL(t *testing.T) {
	salida := []byte(`{"oferta_ref":"oferta:1","recibo_ref":"recibo:oferta:1","bolsa_ref":"bolsa:1",
	 "datos":{"categoria":"Auxiliar","centro":"Residencia","fecha_inicio":"2026-10-01","descripcion":"Sustitución"},
	 "plazo":{"regla_ref":"vec.bolsa.reglas:1:b10.plazo_publicacion","huella_catalogo":"aa","unidad":"dias_habiles","cantidad":2,"computo":"administrativo","ultimo_dia":"2026-09-29","ejemplo":false},
	 "publicada_en":"2026-09-25T10:00:00.000000Z","vence_antes_de":"2026-09-29T22:00:00.000000Z","estado":"pendiente_resolucion",
	 "disposiciones":[{"participacion_ref":"p:3","manifestada_en":"2026-09-26T08:00:00.123456Z","orden_vigente":2,"situacion":"disponible"},{"participacion_ref":"p:1","manifestada_en":"2026-09-26T08:00:00Z","orden_vigente":null,"situacion":"no_disponible"}],
	 "disposiciones_total":2,"propuesta":{"tipo":"adjudicar","participacion_ref":"p:3","orden_vigente":2},"resolucion":null}`)
	oferta, err := decodificarOferta(salida, true)
	if err != nil || !oferta.Reutilizada || oferta.Propuesta == nil || oferta.Propuesta.ParticipacionRef != "p:3" || *oferta.Propuesta.OrdenVigente != 2 ||
		len(oferta.Disposiciones) != 2 || oferta.Disposiciones[1].OrdenVigente != nil || oferta.Plazo.Cantidad != 2 || oferta.VenceAntesDe.IsZero() {
		t.Fatalf("oferta=%+v err=%v", oferta, err)
	}
	if _, err := decodificarOferta([]byte(`{"estado":"abierta"}`), false); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatalf("proyección sin referencia aceptada: %v", err)
	}
	lista, err := decodificarListaOfertas([]byte(`[]`))
	if err != nil || lista == nil || len(lista) != 0 {
		t.Fatalf("lista vacía: %v %v", lista, err)
	}
}

func TestErrorOfertaTraduceCodigosSQL(t *testing.T) {
	casos := map[string]error{
		"42501": dominiovec.ErrAutorizacionDenegada, "VBO01": ports.ErrOfertaConflicto, "VBO02": ports.ErrOfertaYaResuelta,
		"VBO03": ports.ErrOfertaPlazoAbierto, "VBO04": ports.ErrOfertaPropuestaCambiada, "22023": ports.ErrOfertaInvalida,
		"23503": ports.ErrOfertaInvalida, "40001": ports.ErrOfertaNoDisponible,
	}
	for codigo, esperado := range casos {
		if err := errorOferta(&pgconn.PgError{Code: codigo}); !errors.Is(err, esperado) {
			t.Errorf("%s: %v", codigo, err)
		}
	}
}

func TestRepositorioOfertasRechazaSinPoolNiMaterial(t *testing.T) {
	if _, err := NuevoRepositorioOfertasPublicadasPostgreSQL(nil); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatal(err)
	}
	r := &RepositorioOfertasPublicadasPostgreSQL{}
	if _, err := r.Publicar(context.Background(), ports.ComandoPublicarOferta{}); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatal(err)
	}
	if _, err := r.Listar(context.Background(), "bolsa", time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), 0); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatal(err)
	}
}
