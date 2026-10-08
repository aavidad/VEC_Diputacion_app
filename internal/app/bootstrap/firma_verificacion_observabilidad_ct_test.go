package bootstrap

import (
	"context"
	"io"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestCTComparteEmisorResultadosDeDocumentosSinCrearOtro(t *testing.T) {
	f := &firmaDocumentoCTDesarrollo{}
	if f.emisorResultadosFirma() != nil {
		t.Fatal("emisor antes de Documentos")
	}
	vincularResultadosFirmaCT(f, nil)
	if f.emisorResultadosFirma() != nil {
		t.Fatal("Documentos ausente creó emisor")
	}
	emisor, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: io.Discard, Entorno: "pruebas"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		ctx, cancelar := context.WithTimeout(context.Background(), time.Second)
		defer cancelar()
		if err := emisor.Cerrar(ctx); err != nil {
			t.Error(err)
		}
	}()
	d := &autoridadDocumentosDesarrollo{incidencias: emisor}
	vincularResultadosFirmaCT(f, d)
	if f.emisorResultadosFirma() != emisor {
		t.Fatal("CT no reutiliza el emisor de Documentos")
	}
	d.incidencias = vecports.EmisorIncidenciasTecnicasNulo{}
	if f.emisorResultadosFirma() != nil {
		t.Fatal("adaptador sin puerto de resultados admitido")
	}
}
