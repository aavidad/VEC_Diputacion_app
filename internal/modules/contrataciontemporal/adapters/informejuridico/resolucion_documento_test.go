package informejuridico

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	// La descarga lee el acto desde la proyección posterior a él (v8).
	d8 := d.Clonar()
	d8.Resumen.Version, d8.Resumen.ActualizadoEn = 8, recibo.RegistradaEn
	d8.Hitos = append(d8.Hitos, ports.HitoExpedienteRRHH{Secuencia: 8, VersionExpediente: 8, AccionClave: "registrar_resolucion_formalizacion",
		RealizadaEn: recibo.RegistradaEn, FaseOrigen: "nombramiento", FaseDestino: "nombramiento", EstadoOrigen: d.Resumen.EstadoClave, EstadoDestino: d.Resumen.EstadoClave})
	word, err := (RenderizadorBorradorDOCXDesarrollo{DOCX: docxvec.Renderizador{}, Plantillas: plantillas, Resolucion: r.Resolucion}).RenderizarBorradorDOCX(context.Background(), ports.BorradorResolucion, d8)
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

func plantillasConResolucionPrueba(t *testing.T) *PlantillasBorrador {
	t.Helper()
	catalogo := catalogoPlantillasPrueba(t)
	for i := range catalogo.Entradas {
		if catalogo.Entradas[i].Clave == "resolucion" {
			catalogo.Entradas[i].Atributos["parrafo.01"] = "Número {{resolucion_numero}}; fecha {{resolucion_fecha}}"
		}
	}
	plantillas, err := NuevasPlantillasBorrador(catalogo, instantePlantillasPrueba)
	if err != nil {
		t.Fatal(err)
	}
	return plantillas
}

func reciboResolucionDocumentoPrueba(d ports.DetalleExpedienteRRHH, huella string) ports.ResultadoResolucionFormalizacion {
	s := ports.SolicitudResolucionFormalizacion{ExpedienteRef: d.Resumen.ExpedienteRef, VersionEsperada: 7,
		PropuestaRef: "propuesta:prueba", ClaveIdempotencia: "31111111-1111-4111-8111-111111111111",
		NumeroResolucion: "2026/345", FechaResolucion: "2026-09-06", Motivo: "Revisión del expediente.",
		ConfirmaRevisionPropuesta: true, ConfirmaEjercicioManual: true}
	return ports.ResultadoResolucionFormalizacion{Solicitud: s, Estado: "registrada", ResolucionRef: "resolucion:prueba", DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoVersion: 7, DocumentoSHA256: huella, ActuacionRef: "resolucion:prueba", AuditoriaRef: "auditoria:prueba", OutboxRef: "evento:prueba", ReciboRef: "recibo:prueba", VersionResultante: 8, RegistradaEn: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)}
}

// A0 (CT261) conserva el acto legible tras v9: la plantilla nueva lo combina.
func TestResolucionDocumentoTrasV9LeeElActo(t *testing.T) {
	d := detalleInformeDefinitivoPruebaVersionNueve()
	recibo := reciboResolucionDocumentoPrueba(d, strings.Repeat("b", 64))
	lecturas := 0
	fuente := func(_ context.Context, expediente string) (*ports.ResultadoResolucionFormalizacion, error) {
		lecturas++
		if expediente != d.Resumen.ExpedienteRef {
			t.Fatalf("expediente cruzado: %s", expediente)
		}
		r := recibo
		return &r, nil
	}
	datos, err := resolucionBorrador(context.Background(), ports.BorradorResolucion, d, fuente, plantillasConResolucionPrueba(t), false)
	if err != nil || datos == nil || datos.Solicitud.NumeroResolucion != "2026/345" || datos.Solicitud.FechaResolucion != "2026-09-06" || lecturas != 1 {
		t.Fatalf("acto no recuperado tras v9: %+v %v %d", datos, err, lecturas)
	}
	if datos, err := resolucionBorrador(context.Background(), ports.BorradorResolucion, d, fuente, plantillasPrueba(t), false); datos != nil || err != nil || lecturas != 1 {
		t.Fatal("plantilla anterior añadió una lectura del acto", err)
	}
	if _, err := resolucionBorrador(context.Background(), ports.BorradorResolucion, d, func(context.Context, string) (*ports.ResultadoResolucionFormalizacion, error) {
		return nil, ports.ErrResultadoResolucionFormalizacionNoConfiable
	}, plantillasConResolucionPrueba(t), false); !errors.Is(err, ports.ErrResultadoResolucionFormalizacionNoConfiable) {
		t.Fatal("rechazo del lector ocultado", err)
	}
}

// La proyección v7 descargada es la actual previa al acto: campos vacíos, sin lectura.
func TestResolucionDocumentoDescargaV7SinActoNoLee(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	fuente := func(context.Context, string) (*ports.ResultadoResolucionFormalizacion, error) {
		t.Fatal("descarga previa al acto consultó el acto")
		return nil, nil
	}
	if datos, err := resolucionBorrador(context.Background(), ports.BorradorResolucion, d, fuente, plantillasConResolucionPrueba(t), false); datos != nil || err != nil {
		t.Fatal("se inventó un acto", err)
	}
}

// El original firmable de la resolución exige el acto y su huella exacta.
func TestResolucionDocumentoOriginalExigeActo(t *testing.T) {
	d := detalleInformeDefinitivoPrueba()
	r := RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{}, Plantillas: plantillasPrueba(t), ComprobarHuellaResolucion: true}
	if _, err := r.RenderizarBorrador(context.Background(), ports.BorradorResolucion, d); !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
		t.Fatal("original de resolución sin acto admitido", err)
	}
	r.Resolucion = func(context.Context, string) (*ports.ResultadoResolucionFormalizacion, error) {
		return nil, ports.ErrResolucionFormalizacionNoDisponible
	}
	if _, err := r.RenderizarBorrador(context.Background(), ports.BorradorResolucion, d); !errors.Is(err, ports.ErrResolucionFormalizacionNoDisponible) {
		t.Fatal("fallo del lector ocultado", err)
	}
	contenido := []byte("%PDF-prueba")
	h := sha256.Sum256(contenido)
	datos := &ports.DatosResolucionDocumentoRRHH{OriginalSHA256: hex.EncodeToString(h[:])}
	if err := cotejarOriginalResolucion(ports.BorradorResolucion, d, true, datos, contenido); err != nil {
		t.Fatal("original idéntico rechazado", err)
	}
	if err := cotejarOriginalResolucion(ports.BorradorResolucion, detalleInformeDefinitivoPruebaVersionNueve(), true, datos, contenido); !errors.Is(err, ports.ErrResultadoResolucionFormalizacionNoConfiable) {
		t.Fatal("original de otra versión admitido", err)
	}
	// Plantilla anterior: el cotejo lee el acto, pero solo para la resolución.
	fuente := func(context.Context, string) (*ports.ResultadoResolucionFormalizacion, error) {
		t.Fatal("otro documento consultó el acto sin campos combinables")
		return nil, nil
	}
	for _, tipo := range []ports.TipoBorradorRRHH{ports.BorradorDiligencia, ports.BorradorTomaPosesion, ports.BorradorNotificacion, ports.BorradorComunicacionCentro} {
		if datos, err := resolucionBorrador(context.Background(), tipo, detalleInformeDefinitivoPruebaVersionNueve(), fuente, plantillasPrueba(t), true); datos != nil || err != nil {
			t.Fatalf("%s: %v", tipo, err)
		}
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
