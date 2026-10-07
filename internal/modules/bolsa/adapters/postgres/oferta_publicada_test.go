package postgres

import (
	"context"
	"errors"
	"strings"
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
	 "disposiciones_total":2,"propuesta":null,"resolucion":null,"numero_plazas":2,
	 "politica_plazas":{"llamada":"simultanea","respuesta_horas":24,"tras_renuncia":"siguiente_en_orden"},
	 "plazas":[{"numero_de_plaza":1,"estado":"pendiente_respuesta","secuencia":1,"participacion_ref":"p:3","orden_vigente":2,
	   "responder_antes_de":"2026-09-30T22:00:00.000000Z","puede_sin_respuesta":false,"propuesta":null,
	   "historial":[{"secuencia":1,"tipo":"adjudicada","orden_vigente":2,"registrado_en":"2026-09-29T22:00:00.000000Z","recibo_ref":"recibo:plaza-oferta:1"}]},
	  {"numero_de_plaza":2,"estado":"vacante","secuencia":0,"participacion_ref":null,"orden_vigente":null,"responder_antes_de":null,
	   "puede_sin_respuesta":false,"propuesta":{"tipo":"llamamiento_directo"},"historial":[]}]}`)
	oferta, err := decodificarOferta(salida, true)
	if err != nil || !oferta.Reutilizada || oferta.Propuesta != nil || oferta.NumeroPlazas != 2 || oferta.PoliticaPlazas == nil ||
		oferta.PoliticaPlazas.RespuestaHoras != 24 || len(oferta.Plazas) != 2 || *oferta.Plazas[0].ParticipacionRef != "p:3" ||
		oferta.Plazas[0].ResponderAntesDe == nil || len(oferta.Plazas[0].Historial) != 1 || oferta.Plazas[1].Propuesta.Tipo != "llamamiento_directo" ||
		len(oferta.Disposiciones) != 2 || oferta.Disposiciones[1].OrdenVigente != nil || oferta.Plazo.Cantidad != 2 || oferta.VenceAntesDe.IsZero() {
		t.Fatalf("oferta=%+v err=%v", oferta, err)
	}
	conConfirmacion := []byte(strings.Replace(string(salida), `"estado":"pendiente_resolucion"`, `"confirmacion_adjudicacion":"aceptacion_previa","estado":"pendiente_resolucion"`, 1))
	telematica, err := decodificarOferta(conConfirmacion, false)
	if err != nil || telematica.ConfirmacionAdjudicacion != "aceptacion_previa" {
		t.Fatalf("confirmación de la política perdida: %q, %v", telematica.ConfirmacionAdjudicacion, err)
	}
	modoDesconocido := []byte(strings.Replace(string(conConfirmacion), `"confirmacion_adjudicacion":"aceptacion_previa"`, `"confirmacion_adjudicacion":"otra"`, 1))
	if _, err := decodificarOferta(modoDesconocido, false); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatalf("confirmación no implementada admitida: %v", err)
	}
	// Una proyección sin una entrada por plaza no se acepta.
	incompleta := []byte(`{"oferta_ref":"oferta:1","estado":"abierta","numero_plazas":2,"plazas":[{"numero_de_plaza":1,"estado":"vacante","secuencia":0}]}`)
	if _, err := decodificarOferta(incompleta, false); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatalf("proyección con plazas incompletas aceptada: %v", err)
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
		"VBO07": ports.ErrOfertaRespuestaAbierta, "VBO08": ports.ErrOfertaPoliticaSinPlazas,
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
	if _, err := r.Resolver(context.Background(), ports.ComandoResolverOferta{}); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatal(err)
	}
	if _, err := r.Listar(context.Background(), "bolsa", time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC), 0); !errors.Is(err, ports.ErrOfertaNoDisponible) {
		t.Fatal(err)
	}
}
