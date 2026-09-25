package documentos

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type textosComisionPrueba struct{ sinEstado bool }

func (t textosComisionPrueba) Texto(_ context.Context, clave string) (string, error) {
	if t.sinEstado {
		return clave, nil
	}
	return "Borrador sin aprobación", nil
}
func (textosComisionPrueba) FechaCivil(_ context.Context, fecha string) (string, error) {
	return fecha[8:] + "/" + fecha[5:7] + "/" + fecha[:4], nil
}
func (textosComisionPrueba) FechaHora(_ context.Context, instante time.Time) (string, error) {
	return instante.UTC().Format("02/01/2006 15:04"), nil
}

func plantillaComisionPrueba() vecdomain.PlantillaDocumento {
	fecha := time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC)
	return vecdomain.PlantillaDocumento{
		ID: "plantilla_comision", Version: 1, ModuloID: "dietas", TipoDocumental: TipoDocumentoComisionBorrador,
		Nombre: "Borrador de comisión", Titulo: "Comisión {{referencia}}",
		Parrafos: []string{"Estado: {{estado}}. Del {{fecha_inicio}} al {{fecha_fin}}.", "Motivo: {{motivo}}.", "Ruta: {{itinerario}}."},
		Campos: []vecdomain.CampoPlantillaDocumento{
			{Clave: "referencia", Etiqueta: "Referencia", Obligatorio: true},
			{Clave: "estado", Etiqueta: "Estado", Obligatorio: true},
			{Clave: "fecha_inicio", Etiqueta: "Inicio", Obligatorio: true},
			{Clave: "fecha_fin", Etiqueta: "Fin", Obligatorio: true},
			{Clave: "motivo", Etiqueta: "Motivo", Obligatorio: true},
			{Clave: "itinerario", Etiqueta: "Itinerario", Obligatorio: true},
		},
		Formatos: []vecdomain.FormatoDocumento{vecdomain.FormatoDocumentoPDF}, PermisoGenerar: "dietas.documentos.generar",
		GarantiaMinima: vecdomain.AuthAssuranceSubstantial, Estado: vecdomain.EstadoPlantillaPublicada,
		CreadaPor: "rrhh-1", CreadaEn: fecha.Add(-time.Hour), PublicadaPor: "rrhh-2", PublicadaEn: fecha,
		AprobacionRef: "aprobacion-plantilla-1", MotivoPublicacion: "Aprobación gobernada",
	}
}

func instantaneaComisionPrueba() ports.InstantaneaDocumentoComision {
	ref := "dco_" + strings.Repeat("a", 22)
	return ports.InstantaneaDocumentoComision{
		Referencia: ref, Version: 1, ReciboRef: "rcd-1", RegistradoEn: time.Date(2026, 9, 22, 11, 0, 0, 0, time.UTC),
		Borrador: domain.ComisionBorrador{Referencia: ref, Estado: "borrador", FechaInicio: "2026-09-22", FechaFin: "2026-09-23", Motivo: "Visita técnica", CodigosRuta: []string{"granada", "motril"}, RelacionRef: "rel_" + strings.Repeat("b", 22)},
	}
}

func TestRenderizarDocumentoComisionPDFRequierePlantillaPublicadaYTraduccion(t *testing.T) {
	plantilla := plantillaComisionPrueba()
	plantilla.Estado = vecdomain.EstadoPlantillaBorrador
	plantilla.PublicadaPor, plantilla.PublicadaEn, plantilla.AprobacionRef, plantilla.MotivoPublicacion = "", time.Time{}, "", ""
	if _, err := NuevoRenderizadorComisionPDF(plantilla, textosComisionPrueba{}); !errors.Is(err, ports.ErrDocumentoComisionNoDisponible) {
		t.Fatalf("plantilla no publicada: %v", err)
	}
	plantilla = plantillaComisionPrueba()
	plSinMinimo := plantillaComisionPrueba()
	plSinMinimo.Campos[1].Obligatorio = false
	if _, err := NuevoRenderizadorComisionPDF(plSinMinimo, textosComisionPrueba{}); !errors.Is(err, ports.ErrDocumentoComisionNoDisponible) {
		t.Fatalf("plantilla sin campo minimo obligatorio: %v", err)
	}
	r, err := NuevoRenderizadorComisionPDF(plantilla, textosComisionPrueba{sinEstado: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.Renderizar(context.Background(), instantaneaComisionPrueba()); !errors.Is(err, ports.ErrDocumentoComisionNoDisponible) {
		t.Fatalf("estado sin traduccion: %v", err)
	}
	r, err = NuevoRenderizadorComisionPDF(plantilla, textosComisionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	uno, err := r.Renderizar(context.Background(), instantaneaComisionPrueba())
	if err != nil || !bytes.HasPrefix(uno, []byte("%PDF-")) {
		t.Fatalf("pdf: %v", err)
	}
	dos, err := r.Renderizar(context.Background(), instantaneaComisionPrueba())
	if err != nil || !bytes.Equal(uno, dos) {
		t.Fatalf("bytes no deterministas: %v", err)
	}
}
