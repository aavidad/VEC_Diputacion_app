package bootstrap

import (
	"context"
	"errors"
	"testing"

	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type dependenciasR5PresentesPrueba struct{}

func (dependenciasR5PresentesPrueba) ObtenerOriginalFirma(context.Context, ports.SolicitudOriginalFirma) (ports.OriginalFirmaAutorizado, error) {
	return ports.OriginalFirmaAutorizado{}, nil
}
func (dependenciasR5PresentesPrueba) RegistrarFirmaExterna(context.Context, ports.MaterialFirmaExterna, ports.CapacidadFirmaExterna) (ports.ReciboFirmaDocumento, error) {
	return ports.ReciboFirmaDocumento{}, nil
}
func (dependenciasR5PresentesPrueba) RegistrarFirmaVec(context.Context, ports.MaterialFirmaVec, ports.CapacidadFirmaVec) (ports.ReciboFirmaDocumento, error) {
	return ports.ReciboFirmaDocumento{}, nil
}
func (dependenciasR5PresentesPrueba) ConsultarFirmasAutorizadas(context.Context, ports.MaterialConsultaFirmasR5, ports.CapacidadConsultaFirmasR5) (ports.LecturaFirmasR5, error) {
	return ports.LecturaFirmasR5{}, nil
}
func (dependenciasR5PresentesPrueba) AutorizarConsultaFirmasR5(context.Context, ports.MaterialConsultaFirmasR5) (ports.CapacidadConsultaFirmasR5, error) {
	return ports.CapacidadConsultaFirmasR5{}, nil
}
func (dependenciasR5PresentesPrueba) AutorizarRegistroFirmaExterna(context.Context, ports.MaterialFirmaExterna) (ports.CapacidadFirmaExterna, error) {
	return ports.CapacidadFirmaExterna{}, nil
}
func (dependenciasR5PresentesPrueba) AutorizarFirmaVec(context.Context, ports.MaterialFirmaVec) (ports.CapacidadFirmaVec, error) {
	return ports.CapacidadFirmaVec{}, nil
}
func (dependenciasR5PresentesPrueba) AcreditarCompetenciaFirmante(context.Context, ports.SolicitudCompetenciaFirmante) (ports.EvidenciaCompetenciaFirmante, error) {
	return ports.EvidenciaCompetenciaFirmante{}, nil
}
func (dependenciasR5PresentesPrueba) PoliticaMismaPersonaEnPasos(context.Context, string, string) (ports.PoliticaMismaPersonaEnPasos, error) {
	return ports.PoliticaMismaPersonaEnPasos{}, nil
}

func dependenciasCompletasR5Prueba() dependenciasFirmaR5Desarrollo {
	p := dependenciasR5PresentesPrueba{}
	return dependenciasFirmaR5Desarrollo{original: p, registro: p, consulta: p,
		autorizarExterna: p, autorizarVec: p, competencia: p, politicaFirmantes: p}
}

func TestComposicionFirmasR5DeniegaDependenciasIncompletas(t *testing.T) {
	for _, f := range []*firmaDocumentoCTDesarrollo{nil, {}, {servicio: &ctapplication.ServicioFirmaDocumento{}}} {
		if err := f.componerFirmasR5(dependenciasFirmaR5Desarrollo{}); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) {
			t.Fatalf("composición incompleta: %v", err)
		}
		if f != nil && (f.firmaExterna != nil || f.firmaVec != nil) {
			t.Fatal("una dependencia ausente publicó una vía R5")
		}
	}
}

func TestComposicionFirmasR5NoConsumeOriginalSiFaltaBase(t *testing.T) {
	for _, custodiaCompuesta := range []bool{false, true} {
		base := &ctapplication.ServicioFirmaDocumento{}
		f := &firmaDocumentoCTDesarrollo{servicio: base, custodiaR5Compuesta: custodiaCompuesta}
		if err := f.componerFirmasR5(dependenciasCompletasR5Prueba()); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) {
			t.Fatalf("base sin custodia o verificador aceptada: %v", err)
		}
		if f.firmaExterna != nil || f.firmaVec != nil {
			t.Fatal("una vía R5 quedó compuesta tras fallo")
		}
		// La composición fallida no puede ocupar la asignación única del
		// original compartido ni alterar el comportamiento del servicio legado.
		if err := base.ComponerOriginalAutorizado(dependenciasR5PresentesPrueba{}); err != nil {
			t.Fatalf("el original quedó consumido tras fallo: %v", err)
		}
	}
}
