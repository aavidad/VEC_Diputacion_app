package informejuridico

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/docx"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type proveedorPlantillasVariable struct {
	catalogo vecdomain.CatalogoConfigurable
	llamadas int
}

func (p *proveedorPlantillasVariable) ObtenerPlantillas(_ context.Context, instante time.Time) (*PlantillasBorrador, error) {
	p.llamadas++
	return NuevasPlantillasBorrador(p.catalogo, instante)
}

func TestRenderizadorVivoUsaVersionPublicadaPorPeticion(t *testing.T) {
	primera := catalogoPlantillasPrueba(t)
	fecha := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	segunda, err := primera.NuevaVersion(2, "configurador:rrhh:001", "fuente:rrhh:plantillas", "Añadir tipo", fecha)
	if err != nil {
		t.Fatal(err)
	}
	segunda, err = segunda.ActualizarBorrador(segunda.Revision, "configurador:rrhh:001", segunda.Nombre, segunda.Descripcion,
		segunda.FuenteRef, "Añadir nuevo tipo", append(segunda.Entradas, vecdomain.EntradaCatalogoConfigurable{
			Clave: "certificacion_servicio", Etiqueta: "Certificación de servicio", Orden: 11, VigenteDesde: fecha,
			Atributos: map[string]string{
				"titulo":     "Certificación de servicio — borrador",
				"parrafo.01": "BORRADOR SIN FIRMA NI VALIDEZ ADMINISTRATIVA.",
				"parrafo.02": "Expediente {{numero_expediente}}. Plantilla {{plantilla_ref}}.",
			},
		}), fecha.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	segunda, err = segunda.Publicar("revisor:rrhh:002", "aprobacion:rrhh:plantillas", "Ejercicio sintético", fecha.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	proveedor := &proveedorPlantillasVariable{catalogo: primera}
	r := RenderizadorBorradorCatalogoVivo{
		Proveedor: proveedor, PDF: pdf.Renderizador{}, DOCX: docx.Renderizador{},
		Ahora: func() time.Time { return fecha.Add(3 * time.Minute) },
	}
	detalle := detalleInformeDefinitivoPrueba()
	primero, err := r.GenerarPDF(context.Background(), ports.BorradorResolucion, detalle)
	if err != nil || primero.CatalogoRef != primera.Referencia() || !strings.HasPrefix(string(primero.Contenido), "%PDF-") {
		t.Fatalf("primera versión: %+v, error=%v", primero, err)
	}
	proveedor.catalogo = segunda
	tipoNuevo := ports.TipoBorradorRRHH("certificacion_servicio")
	word, err := r.GenerarDOCX(context.Background(), tipoNuevo, detalle)
	if err != nil || word.CatalogoRef != segunda.Referencia() || word.PlantillaRef != segunda.Referencia()+":"+string(tipoNuevo) ||
		!strings.HasPrefix(string(word.Contenido), "PK") || len(word.CatalogoHuellaSHA256) != 64 {
		t.Fatalf("nueva versión Word: %+v, error=%v", word, err)
	}
	pdfNuevo, err := r.GenerarPDF(context.Background(), tipoNuevo, detalle)
	if err != nil || pdfNuevo.CatalogoHuellaSHA256 != word.CatalogoHuellaSHA256 ||
		!strings.HasPrefix(string(pdfNuevo.Contenido), "%PDF-") {
		t.Fatalf("nueva versión PDF: %+v, error=%v", pdfNuevo, err)
	}
	if proveedor.llamadas != 3 {
		t.Fatalf("consultas a catálogo = %d; se esperaba una por generación", proveedor.llamadas)
	}
	proveedor.catalogo = primera
	if _, err := r.GenerarPDF(context.Background(), tipoNuevo, detalle); !errors.Is(err, ports.ErrBorradorRRHHNoDisponible) {
		t.Fatalf("tipo no publicado: %v", err)
	}
}

func TestRenderizadorVivoDeniegaSinProveedor(t *testing.T) {
	r := RenderizadorBorradorCatalogoVivo{PDF: pdf.Renderizador{}, DOCX: docx.Renderizador{}}
	if _, err := r.GenerarPDF(context.Background(), ports.BorradorResolucion, detalleInformeDefinitivoPrueba()); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("sin proveedor: %v", err)
	}
}
