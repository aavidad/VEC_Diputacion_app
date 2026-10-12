// Package firmasct compone las vías CT sobre autoridades ya construidas.
package firmasct

import (
	"context"
	"errors"
	"reflect"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	doc "vec-diputacion-granada/internal/vec/documentos/ports"
)

var ErrComposicionNoDisponible = errors.New("composicion CT: firmas R5 no disponibles")

// RegistroDurable excluye el escritor directo de CT172. Ambas vías reciben
// únicamente el decorador que confirma la firma con su plan en CT176.
type RegistroDurable interface {
	ConsultarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2,
		ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error)
	ports.RegistradorFirmaConPlanV2
}

type Dependencias struct {
	Base              *ct.ServicioFirmaDocumento
	Original          ports.FuenteOriginalFirmaAutorizado
	Registro          RegistroDurable
	DescriptorPlan    ports.FuenteDescriptorPlanFijadoFirmaV2
	EmisorPlan        ports.EmisorMaterialPlanFirmaV2
	Consulta          ports.AutorizadorConsultaFirmasR5V2
	Autorizar         ports.AutorizadorFirmaVerificadaV2
	Verificador       doc.VerificadorFirmasDocumento
	PDFAnterior       ports.FuentePDFFirmaAnterior
	Competencia       ports.FuenteCompetenciaFirmante
	PoliticaFirmantes ports.FuentePoliticaMismaPersonaEnPasos
}

type Servicios struct {
	Vec      *ct.ServicioFirmaVec
	Externa  *ct.ServicioFirmaExterna
	Registro ports.RegistroFirmasVerificadasV2
}

// Componer conserva intacta la base y entrega ambas vías juntas. No crea
// identidad, PDP, almacenamiento, perfiles ni rutas HTTP.
func Componer(d Dependencias) (Servicios, error) {
	var cero Servicios
	if d.Base == nil || d.PoliticaFirmantes == nil {
		return cero, ErrComposicionNoDisponible
	}
	// Los constructores V2 admiten política ausente, pero el montaje R5 exige
	// la política gobernada también cuando llega como una interfaz con nil.
	valor := reflect.ValueOf(d.PoliticaFirmantes)
	switch valor.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Func, reflect.Interface, reflect.Chan:
		if valor.IsNil() {
			return cero, ErrComposicionNoDisponible
		}
	}
	registro, err := firmaautorizacionv2.NuevoRegistroConPlanV2(d.DescriptorPlan, d.EmisorPlan,
		d.Registro, consultaSinRegistroDirecto{d.Registro})
	if err != nil {
		return cero, errors.Join(ErrComposicionNoDisponible, err)
	}
	base := *d.Base
	if err := base.ComponerOriginalAutorizado(d.Original); err != nil {
		return cero, errors.Join(ErrComposicionNoDisponible, err)
	}
	externa, err := ct.NuevoServicioFirmaExternaV2(&base, d.Verificador, registro,
		d.Autorizar, d.Consulta, d.PDFAnterior, d.Competencia, d.PoliticaFirmantes)
	if err != nil {
		return cero, errors.Join(ErrComposicionNoDisponible, err)
	}
	vec, err := ct.NuevoServicioFirmaVecV2(&base, d.Verificador, registro,
		d.Autorizar, d.Consulta, d.PDFAnterior, d.Competencia, d.PoliticaFirmantes)
	if err != nil {
		return cero, errors.Join(ErrComposicionNoDisponible, err)
	}
	return Servicios{Vec: vec, Externa: externa, Registro: registro}, nil
}

type consultaSinRegistroDirecto struct{ durable RegistroDurable }

func (consultaSinRegistroDirecto) RegistrarFirmaVerificadaV2(context.Context,
	ports.MaterialFirmaVerificadaV2, ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	return ports.ReciboFirmaDocumento{}, ports.ErrRegistroFirmaDocumentoNoDisponible
}

func (c consultaSinRegistroDirecto) ConsultarFirmasAutorizadasV2(ctx context.Context,
	m ports.MaterialConsultaFirmasR5V2, cap ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	return c.durable.ConsultarFirmasAutorizadasV2(ctx, m, cap)
}
