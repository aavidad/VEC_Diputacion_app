package main

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/app/bootstrap"
)

func TestCargaProvisionUsaHuellaCanonicaDelCatalogoPublicado(t *testing.T) {
	ruta := filepath.Join("..", "..", "data", "demo", "plantillas", "ct_plantillas_documentos.ejemplo.demo.json")
	c, err := bootstrap.CargarCatalogoPlantillasCT(ruta)
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.HuellaSHA256()
	if err != nil || len(h) != 64 || c.Version != 1 || c.Estado != "publicado" || len(c.Entradas) < 6 {
		t.Fatalf("catálogo de origen incompatible: %+v, %v", c, err)
	}
	r := reciboProvision{Resultado: "registrado", ReciboRef: "recibo:11111111-1111-4111-8111-111111111111", Version: int64(c.Version), Revision: int64(c.Revision), CatalogoHuellaSHA256: h, ContenidoJSONSHA256: strings.Repeat("a", 64), ProcedenciaRef: c.FuenteRef, RegistradaEn: time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)}
	if !reciboCompatible(r, c, h, r.ContenidoJSONSHA256) {
		t.Fatal("recibo legítimo rechazado")
	}
	r.CatalogoHuellaSHA256 = strings.Repeat("b", 64)
	if reciboCompatible(r, c, h, r.ContenidoJSONSHA256) {
		t.Fatal("recibo con otra huella aceptado")
	}
	r.CatalogoHuellaSHA256 = h
	if reciboCompatible(r, c, h, strings.Repeat("b", 64)) {
		t.Fatal("recibo con hash JSON forjado aceptado")
	}
	r.Resultado = "replay"
	if !reciboCompatible(r, c, h, r.ContenidoJSONSHA256) {
		t.Fatal("replay con el mismo recibo rechazado")
	}
}

type filaPreflight struct {
	valido bool
	err    error
}

func (f filaPreflight) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*(dest[0].(*bool)) = f.valido
	return nil
}

type consultaPreflight struct{ fila filaPreflight }

func (q consultaPreflight) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if sql != sqlPreflight {
		return filaPreflight{err: errors.New("consulta inesperada")}
	}
	return q.fila
}

func TestPreflightRechazaRolAjenoYErrorSQL(t *testing.T) {
	for _, caso := range []consultaPreflight{
		{fila: filaPreflight{valido: false}},
		{fila: filaPreflight{err: errors.New("rol o esquema no disponible")}},
	} {
		if err := comprobarPreflight(context.Background(), caso); !errors.Is(err, errConexion) {
			t.Fatalf("preflight aceptó identidad o SQL rechazados: %v", err)
		}
	}
	if err := comprobarPreflight(context.Background(), consultaPreflight{fila: filaPreflight{valido: true}}); err != nil {
		t.Fatalf("preflight válido: %v", err)
	}
}

type salidaRota struct{}

func (salidaRota) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestFalloDeSalidaNoConfirmaEntregaDelRecibo(t *testing.T) {
	if err := emitirRecibo(salidaRota{}, reciboProvision{Resultado: "replay"}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("salida rota tratada como recibo entregado: %v", err)
	}
}
