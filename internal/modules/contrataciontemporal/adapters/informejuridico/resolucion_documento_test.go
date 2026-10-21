package informejuridico

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docxvec "vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
)

func TestResolucionDocumentoNumeroFechaYOriginalDeterminista(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	s := ports.SolicitudResolucionFormalizacion{ExpedienteRef: d.Resumen.ExpedienteRef, VersionEsperada: 7,
		PropuestaRef: "propuesta:prueba", ClaveIdempotencia: "31111111-1111-4111-8111-111111111111",
		NumeroResolucion: "2026/345", FechaResolucion: "2026-09-06", Motivo: "Revisión del expediente.",
		ConfirmaRevisionPropuesta: true, ConfirmaEjercicioManual: true}
	catalogo := catalogoPlantillasPrueba(t)
	for i := range catalogo.Entradas {
		if catalogo.Entradas[i].Clave == "resolucion" {
			catalogo.Entradas[i].Atributos["parrafo.01"] = "Número {{resolucion_numero}}; fecha {{resolucion_fecha}}; actuación {{propuesta_actuacion}}; {{propuesta_fecha}}"
		}
	}
	plantillas, err := NuevasPlantillasBorrador(catalogo, instantePlantillasPrueba)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := ports.ConResolucionDocumentoRRHH(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	r := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}, Plantillas: plantillas}
	antes, err := r.RenderizarBorrador(ctx, ports.BorradorResolucion, d)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(antes)
	recibo := ports.ResultadoResolucionFormalizacion{Solicitud: s, Estado: "registrada", ResolucionRef: "resolucion:prueba", DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoVersion: 7, DocumentoSHA256: hex.EncodeToString(h[:]), ActuacionRef: "resolucion:prueba", AuditoriaRef: "auditoria:prueba", OutboxRef: "evento:prueba", ReciboRef: "recibo:prueba", VersionResultante: 8, RegistradaEn: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)}
	r.Resolucion = func(context.Context, string) (*ports.ResultadoResolucionFormalizacion, error) { return &recibo, nil }
	r.ComprobarHuellaResolucion = true
	despues, err := r.RenderizarBorrador(context.Background(), ports.BorradorResolucion, d)
	if err != nil || !bytes.Equal(antes, despues) {
		t.Fatalf("el original v7 cambia tras registrar el acto: %v", err)
	}
	recibo.DocumentoSHA256 = strings.Repeat("a", 64)
	if _, err := r.RenderizarBorrador(context.Background(), ports.BorradorResolucion, d); err == nil {
		t.Fatal("original distinto del acto admitido")
	}
	recibo.DocumentoSHA256 = hex.EncodeToString(h[:])
	for _, tipo := range []ports.TipoBorradorRRHH{ports.BorradorDiligencia, ports.BorradorTomaPosesion, ports.BorradorNotificacion, ports.BorradorComunicacionCentro} {
		if _, err := r.RenderizarBorrador(context.Background(), tipo, d); err != nil {
			t.Fatalf("huella de resolución aplicada a %s: %v", tipo, err)
		}
	}
	contenido, err := contenidoBorradorConResolucion(ports.BorradorResolucion, d, nil, plantillas, false, &s)
	if err != nil || !strings.Contains(contenido.Parrafos[0], "2026/345; fecha 2026-09-06; actuación 7") {
		t.Fatalf("datos propios ausentes: %v", err)
	}
	word, err := (RenderizadorBorradorDOCXDesarrollo{DOCX: docxvec.Renderizador{}, Plantillas: plantillas, Resolucion: r.Resolucion}).RenderizarBorradorDOCX(context.Background(), ports.BorradorResolucion, d)
	if err != nil {
		t.Fatal(err)
	}
	archivo, err := zip.NewReader(bytes.NewReader(word), int64(len(word)))
	if err != nil {
		t.Fatal(err)
	}
	documento, err := archivo.Open("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer documento.Close()
	xml, err := io.ReadAll(documento)
	if err != nil || !bytes.Contains(xml, []byte("2026/345")) || !bytes.Contains(xml, []byte("2026-09-06")) {
		t.Fatalf("datos propios ausentes en Word: %v", err)
	}
}

func TestResolucionDocumentoNoConsultaActoParaOtrasVersiones(t *testing.T) {
	d := detalleInformeDefinitivoPruebaVersionNueve()
	fuente := func(context.Context, string) (*ports.ResultadoResolucionFormalizacion, error) {
		t.Fatal("el lector de preparación solo admite v7/v8")
		return nil, nil
	}
	if s, err := resolucionBorrador(context.Background(), ports.BorradorResolucion, d, fuente, plantillasPrueba(t), false); s != nil || err != nil {
		t.Fatal("se inventó un acto", err)
	}
}

func TestResolucionDocumentoPlantillaAnteriorNoAnadeLectura(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	r := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}, Plantillas: plantillasPrueba(t), Resolucion: func(context.Context, string) (*ports.ResultadoResolucionFormalizacion, error) {
		t.Fatal("consulta añadida sin campos combinables nuevos")
		return nil, nil
	}}
	if _, err := r.RenderizarBorrador(context.Background(), ports.BorradorResolucion, d); err != nil {
		t.Fatal(err)
	}
}
