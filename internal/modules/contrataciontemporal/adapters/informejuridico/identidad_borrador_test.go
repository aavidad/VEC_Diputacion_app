package informejuridico

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// Lee el catálogo anterior publicado, sin activar v5.
func catalogoV4IdentidadPrueba(t *testing.T) vecdomain.CatalogoConfigurable {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogoProvisionable(
		"../../../../../data/catalogos/contratacion-temporal/plantillas_documentos.v4.json")
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := consulta.ObtenerCatalogo(context.Background(), CatalogoPlantillasBorradorID, 4)
	if err != nil {
		t.Fatal(err)
	}
	return catalogo
}

// Añade solo las marcas de identidad en memoria: v5 requiere también los
// campos de resolución del corte A.
func plantillasIdentidadPrueba(t *testing.T) *PlantillasBorrador {
	t.Helper()
	catalogo := catalogoV4IdentidadPrueba(t)
	for i := range catalogo.Entradas {
		if catalogo.Entradas[i].Clave == string(ports.BorradorResolucion) {
			catalogo.Entradas[i].Atributos["parrafo.11"] =
				"Titular: {{?persona_nombre}}{{persona_nombre}} {{persona_apellidos}}; documento " +
					"{{persona_documento_enmascarado}}{{/persona_nombre}}" +
					"{{!persona_nombre}}identificación pendiente{{/persona_nombre}}."
		}
	}
	plantillas, err := NuevasPlantillasBorrador(catalogo, time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return plantillas
}

type renderizadorPDFContadorIdentidad struct{ llamadas int }

func (*renderizadorPDFContadorIdentidad) Formato() vecdomain.FormatoDocumento {
	return vecdomain.FormatoDocumentoPDF
}
func (r *renderizadorPDFContadorIdentidad) Renderizar(context.Context, vecdomain.ContenidoDocumento) ([]byte, error) {
	r.llamadas++
	return nil, nil
}
func (*renderizadorPDFContadorIdentidad) ValidarSalida(context.Context, []byte) error { return nil }

func TestIdentidadBorradorPDFYWordUsanUnaLecturaYElMismoTexto(t *testing.T) {
	plantillas := plantillasIdentidadPrueba(t)
	detalle := detalleInformeDefinitivoPrueba()
	const nombre, apellidos, mascara = "Yago", "Lozano Hidalgo", "***4567**"
	var textos []string
	for _, formato := range []vecdomain.FormatoDocumento{vecdomain.FormatoDocumentoPDF, vecdomain.FormatoDocumentoDOCX} {
		t.Run(string(formato), func(t *testing.T) {
			lecturas := 0
			fuente := func(_ context.Context, expedienteRef string) (*ports.IdentidadDocumentoRRHH, error) {
				lecturas++
				if expedienteRef != detalle.Resumen.ExpedienteRef {
					t.Fatalf("expediente ajeno: %q", expedienteRef)
				}
				return &ports.IdentidadDocumentoRRHH{Nombre: nombre, Apellidos: apellidos,
					DocumentoEnmascarado: mascara, VersionFuente: 99223344}, nil
			}
			var captura *renderizadorCapturaNombres
			var salida []byte
			var err error
			if formato == vecdomain.FormatoDocumentoPDF {
				captura = &renderizadorCapturaNombres{RenderizadorDocumento: pdf.Renderizador{}}
				salida, err = (RenderizadorBorradorDesarrollo{PDF: captura, Plantillas: plantillas,
					FuenteIdentidad: fuente, ExigirIdentidad: true}).RenderizarBorrador(context.Background(), ports.BorradorResolucion, detalle)
				if !bytes.HasPrefix(salida, []byte("%PDF-")) {
					t.Fatal("el PDF real no se generó")
				}
			} else {
				captura = &renderizadorCapturaNombres{RenderizadorDocumento: docx.Renderizador{}}
				salida, err = (RenderizadorBorradorDOCXDesarrollo{DOCX: captura, Plantillas: plantillas,
					FuenteIdentidad: fuente}).RenderizarBorradorDOCX(context.Background(), ports.BorradorResolucion, detalle)
				if !bytes.HasPrefix(salida, []byte("PK")) {
					t.Fatal("el Word real no se generó")
				}
				archivo, zipErr := zip.NewReader(bytes.NewReader(salida), int64(len(salida)))
				if zipErr != nil {
					t.Fatal(zipErr)
				}
				var xml []byte
				for _, parte := range archivo.File {
					if parte.Name != "word/document.xml" {
						continue
					}
					lector, openErr := parte.Open()
					if openErr != nil {
						t.Fatal(openErr)
					}
					xml, openErr = io.ReadAll(lector)
					_ = lector.Close()
					if openErr != nil {
						t.Fatal(openErr)
					}
				}
				for _, esperado := range []string{nombre, apellidos, mascara} {
					if !bytes.Contains(xml, []byte(esperado)) {
						t.Fatalf("Word sin %q", esperado)
					}
				}
			}
			if err != nil || lecturas != 1 {
				t.Fatalf("generación: error=%v, lecturas=%d", err, lecturas)
			}
			texto := captura.contenido.Titulo + "\n" + strings.Join(captura.contenido.Parrafos, "\n")
			for _, esperado := range []string{nombre, apellidos, mascara, "2026/CT-0001"} {
				if !strings.Contains(texto, esperado) {
					t.Fatalf("representación sin %q", esperado)
				}
			}
			if strings.Contains(texto, "identificación pendiente") || strings.Contains(texto, "99223344") ||
				strings.Contains(texto, "12345678Z") {
				t.Fatal("la representación contiene estado o dato no autorizado")
			}
			textos = append(textos, texto)
		})
	}
	if len(textos) != 2 || textos[0] != textos[1] {
		t.Fatal("PDF y Word recibieron textos distintos")
	}
}

func TestIdentidadOriginalSeDeniegaAntesDeRenderizar(t *testing.T) {
	completa := ports.IdentidadDocumentoRRHH{Nombre: "Yago", Apellidos: "Lozano Hidalgo",
		DocumentoEnmascarado: "***4567**", VersionFuente: 9}
	for _, caso := range []struct {
		nombre    string
		plantilla *PlantillasBorrador
		identidad *ports.IdentidadDocumentoRRHH
	}{
		{"fuente ausente", plantillasIdentidadPrueba(t), nil},
		{"identidad ausente", plantillasIdentidadPrueba(t), nil},
		{"documento completo", plantillasIdentidadPrueba(t), &ports.IdentidadDocumentoRRHH{
			Nombre: completa.Nombre, Apellidos: completa.Apellidos, DocumentoEnmascarado: "12345678Z", VersionFuente: 9}},
		{"nombre inválido", plantillasIdentidadPrueba(t), &ports.IdentidadDocumentoRRHH{
			Nombre: " Yago", Apellidos: completa.Apellidos, DocumentoEnmascarado: completa.DocumentoEnmascarado, VersionFuente: 9}},
		{"sin versión de fuente", plantillasIdentidadPrueba(t), &ports.IdentidadDocumentoRRHH{
			Nombre: completa.Nombre, Apellidos: completa.Apellidos, DocumentoEnmascarado: completa.DocumentoEnmascarado}},
		{"plantilla antigua sin campos", plantillasPrueba(t), &completa},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			doble := &renderizadorPDFContadorIdentidad{}
			lecturas := 0
			var fuente ports.FuenteIdentidadDocumentoRRHH
			if caso.nombre != "fuente ausente" {
				fuente = func(context.Context, string) (*ports.IdentidadDocumentoRRHH, error) {
					lecturas++
					return caso.identidad, nil
				}
			}
			salida, err := (RenderizadorBorradorDesarrollo{PDF: doble, Plantillas: caso.plantilla,
				FuenteIdentidad: fuente, ExigirIdentidad: true}).RenderizarBorrador(
				context.Background(), ports.BorradorResolucion, detalleInformeDefinitivoPrueba())
			if !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) || len(salida) != 0 || doble.llamadas != 0 {
				t.Fatalf("original inválido llegó al render: err=%v, llamadas=%d", err, doble.llamadas)
			}
			if caso.nombre == "plantilla antigua sin campos" && lecturas != 0 {
				t.Fatal("el catálogo antiguo consultó identidad")
			}
		})
	}
}

func TestInformeYDescargaAntiguaNoConsultanIdentidad(t *testing.T) {
	lecturas := 0
	fuente := func(context.Context, string) (*ports.IdentidadDocumentoRRHH, error) {
		lecturas++
		return nil, errors.New("lectura no permitida")
	}
	detalle := detalleInformeDefinitivoPrueba()
	detalle.Hitos = detalle.Hitos[:5]
	detalle.Hitos[4].AccionClave = domain.AccionEmitirInformeJuridico
	detalle.Hitos[4].FaseDestino = domain.FaseInformeJuridico
	detalle.Resumen.FaseClave = domain.FaseInformeJuridico
	detalle.Resumen.Version = 5
	detalle.Resumen.ActualizadoEn = detalle.Hitos[4].RealizadaEn
	informe, err := (RenderizadorBorradorDesarrollo{PDF: pdf.Renderizador{},
		Plantillas: plantillasIdentidadPrueba(t), FuenteIdentidad: fuente}).RenderizarBorrador(
		context.Background(), ports.BorradorInformeDefinitivo, detalle)
	if err != nil || !bytes.HasPrefix(informe, []byte("%PDF-")) || lecturas != 0 {
		t.Fatalf("informe previo consultó identidad: err=%v, lecturas=%d", err, lecturas)
	}
	catalogo := catalogoV4IdentidadPrueba(t)
	proveedor := &proveedorPlantillasVariable{catalogo: catalogo}
	antiguo := RenderizadorBorradorCatalogoVivo{Proveedor: proveedor,
		PDF: pdf.Renderizador{}, DOCX: docx.Renderizador{}, FuenteIdentidad: fuente}
	for _, generar := range []func(context.Context, ports.TipoBorradorRRHH, ports.DetalleExpedienteRRHH) (BorradorCatalogado, error){
		antiguo.GenerarPDF, antiguo.GenerarDOCX,
	} {
		b, err := generar(context.Background(), ports.BorradorResolucion, detalleInformeDefinitivoPrueba())
		if err != nil || len(b.Contenido) == 0 || lecturas != 0 {
			t.Fatalf("catálogo antiguo consultó identidad: err=%v, lecturas=%d", err, lecturas)
		}
	}
}
