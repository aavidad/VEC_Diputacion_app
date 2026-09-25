// Package documentos adapta el borrador de Dietas a una plantilla PDF gobernada.
package documentos

import (
	"context"
	"strconv"
	"strings"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const TipoDocumentoComisionBorrador = dietasapp.TipoDocumentoComision

// RenderizadorComisionPDF exige una version publicada exacta. La plantilla
// contiene los textos visibles; el catalogo comun localiza fechas y estado.
type RenderizadorComisionPDF struct {
	plantilla vecdomain.PlantillaDocumento
	textos    dietasports.TextosDocumentoComision
	pdf       pdf.Renderizador
}

func NuevoRenderizadorComisionPDF(plantilla vecdomain.PlantillaDocumento, textos dietasports.TextosDocumentoComision) (*RenderizadorComisionPDF, error) {
	if textos == nil || plantilla.Validar() != nil || plantilla.Estado != vecdomain.EstadoPlantillaPublicada ||
		plantilla.ModuloID != "dietas" || plantilla.TipoDocumental != TipoDocumentoComisionBorrador ||
		!plantilla.AdmiteFormato(vecdomain.FormatoDocumentoPDF) {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	for _, campo := range plantilla.Campos {
		if !campoDocumentoComisionAdmitido(campo.Clave) {
			return nil, dietasports.ErrDocumentoComisionNoDisponible
		}
	}
	campos := make(map[string]bool, len(plantilla.Campos))
	for _, campo := range plantilla.Campos {
		campos[campo.Clave] = campo.Obligatorio
	}
	for _, minimo := range []string{"estado", "fecha_inicio", "fecha_fin", "motivo"} {
		if !campos[minimo] {
			return nil, dietasports.ErrDocumentoComisionNoDisponible
		}
	}
	plantilla.Parrafos = append([]string(nil), plantilla.Parrafos...)
	plantilla.Campos = append([]vecdomain.CampoPlantillaDocumento(nil), plantilla.Campos...)
	plantilla.Formatos = append([]vecdomain.FormatoDocumento(nil), plantilla.Formatos...)
	return &RenderizadorComisionPDF{plantilla: plantilla, textos: textos}, nil
}

func campoDocumentoComisionAdmitido(clave string) bool {
	switch clave {
	case "referencia", "version", "recibo_ref", "registrado_en", "fecha_inicio", "fecha_fin", "motivo", "itinerario", "estado":
		return true
	default:
		return false
	}
}

func (r *RenderizadorComisionPDF) Renderizar(ctx context.Context, instantanea dietasports.InstantaneaDocumentoComision) ([]byte, error) {
	if ctx == nil || r == nil || r.textos == nil || instantanea.Validar() != nil {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fechaInicio, err := r.textos.FechaCivil(ctx, instantanea.Borrador.FechaInicio)
	if err != nil || fechaInicio == "" {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	fechaFin, err := r.textos.FechaCivil(ctx, instantanea.Borrador.FechaFin)
	if err != nil || fechaFin == "" {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	estado, err := r.textos.Texto(ctx, "dietas.documento.comision.estado.borrador")
	if err != nil || estado == "" || estado == "dietas.documento.comision.estado.borrador" {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	registradoEn, err := r.textos.FechaHora(ctx, instantanea.RegistradoEn)
	if err != nil || registradoEn == "" {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	disponibles := map[string]string{
		"referencia":    instantanea.Referencia,
		"version":       strconv.FormatUint(instantanea.Version, 10),
		"recibo_ref":    instantanea.ReciboRef,
		"registrado_en": registradoEn,
		"fecha_inicio":  fechaInicio,
		"fecha_fin":     fechaFin,
		"motivo":        instantanea.Borrador.Motivo,
		"itinerario":    strings.Join(instantanea.Borrador.CodigosRuta, " → "),
		"estado":        estado,
	}
	datos := make(map[string]string, len(r.plantilla.Campos))
	for _, campo := range r.plantilla.Campos {
		datos[campo.Clave] = disponibles[campo.Clave]
	}
	contenido, err := r.plantilla.Fusionar(datos)
	if err != nil {
		return nil, dietasports.ErrDocumentoComisionNoDisponible
	}
	return r.pdf.Renderizar(ctx, contenido)
}

var _ dietasports.RenderizadorDocumentoComision = (*RenderizadorComisionPDF)(nil)
