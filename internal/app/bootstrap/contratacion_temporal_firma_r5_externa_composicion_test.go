package bootstrap

import (
	"context"
	"errors"
	"testing"

	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type contextoPreflightR5ExternoPrueba struct{}

func (contextoPreflightR5ExternoPrueba) ResolverContextoAutorizacionAltaV3(context.Context,
	ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	return ports.ContextoAutorizacionAltaV3{}, ports.ErrFirmaDocumentoDenegada
}

func TestComposicionR5ExternaPublicaSoloServicioConRegistroConPlan(t *testing.T) {
	base := baseR5MontajePrueba(t)
	f := &firmaDocumentoCTDesarrollo{servicio: base, custodiaR5Compuesta: true}
	d := dependenciasCompletasR5Prueba()
	durable := &registroDirectoContado{}
	d.registro = durable
	if err := f.componerFirmasR5Externa(d); err != nil {
		t.Fatal(err)
	}
	if f.firmaExterna == nil || f.firmaVec != nil {
		t.Fatal("la composición externa publicó una vía incorrecta")
	}
	if _, ok := f.registroR5.(*firmaautorizacionv2.RegistroConPlanV2); !ok {
		t.Fatalf("registro R5 sin plan CT176: %T", f.registroR5)
	}
	if _, err := ctapplication.NuevoServicioPreflightFirmaR5(f.firmaExterna, contextoPreflightR5ExternoPrueba{}, nil); err != nil {
		t.Fatalf("preflight sin servicio externo válido: %v", err)
	}
	if _, err := f.registroR5.ConsultarFirmasAutorizadasV2(t.Context(), ports.MaterialConsultaFirmasR5V2{}, ports.CapacidadConsultaFirmasR5V2{}); err != nil || durable.consultas != 1 {
		t.Fatalf("consulta de historia R5 no delegada: %v, %d", err, durable.consultas)
	}
	if durable.directas != 0 {
		t.Fatalf("se usó el registro directo CT172: %d", durable.directas)
	}
	if err := base.ComponerOriginalAutorizado(dependenciasR5PresentesPrueba{}); err != nil {
		t.Fatalf("servicio anterior alterado: %v", err)
	}
	if err := f.componerFirmasR5Externa(d); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) {
		t.Fatalf("segundo montaje externo: %v", err)
	}
	if err := f.componerFirmasR5(d); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) {
		t.Fatalf("montaje conjunto tras externo: %v", err)
	}
}

func TestComposicionR5ExternaDeniegaSinBaseOCustodiaSinMutar(t *testing.T) {
	for _, tc := range []struct {
		nombre string
		firma  *firmaDocumentoCTDesarrollo
	}{
		{nombre: "receptor_nulo"},
		{nombre: "sin_servicio", firma: &firmaDocumentoCTDesarrollo{custodiaR5Compuesta: true}},
		{nombre: "sin_custodia", firma: &firmaDocumentoCTDesarrollo{servicio: baseR5MontajePrueba(t)}},
		{nombre: "servicio_incompleto", firma: &firmaDocumentoCTDesarrollo{servicio: &ctapplication.ServicioFirmaDocumento{}, custodiaR5Compuesta: true}},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			if err := tc.firma.componerFirmasR5Externa(dependenciasCompletasR5Prueba()); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) {
				t.Fatalf("montaje incompleto: %v", err)
			}
			if tc.firma != nil && (tc.firma.firmaExterna != nil || tc.firma.firmaVec != nil || tc.firma.registroR5 != nil) {
				t.Fatal("un fallo publicó estado R5")
			}
		})
	}
}

func TestComposicionR5ExternaExigeDiezPuertosYSinNulosTipados(t *testing.T) {
	for nombre, omitir := range map[string]func(*dependenciasFirmaR5Desarrollo){
		"original":     func(d *dependenciasFirmaR5Desarrollo) { d.original = nil },
		"registro":     func(d *dependenciasFirmaR5Desarrollo) { d.registro = nil },
		"plan":         func(d *dependenciasFirmaR5Desarrollo) { d.descriptorPlan = nil },
		"emisor_plan":  func(d *dependenciasFirmaR5Desarrollo) { d.emisorPlan = nil },
		"consulta":     func(d *dependenciasFirmaR5Desarrollo) { d.consulta = nil },
		"autorizador":  func(d *dependenciasFirmaR5Desarrollo) { d.autorizar = nil },
		"verificador":  func(d *dependenciasFirmaR5Desarrollo) { d.verificador = nil },
		"pdf_anterior": func(d *dependenciasFirmaR5Desarrollo) { d.pdfAnterior = nil },
		"competencia":  func(d *dependenciasFirmaR5Desarrollo) { d.competencia = nil },
		"politica":     func(d *dependenciasFirmaR5Desarrollo) { d.politicaFirmantes = nil },
		"registro_nulo_tipado": func(d *dependenciasFirmaR5Desarrollo) {
			d.registro = (*postgrescontratacion.RegistroFirmasVerificadasPostgreSQL)(nil)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			base := baseR5MontajePrueba(t)
			f := &firmaDocumentoCTDesarrollo{servicio: base, custodiaR5Compuesta: true}
			d := dependenciasCompletasR5Prueba()
			omitir(&d)
			if err := f.componerFirmasR5Externa(d); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) {
				t.Fatalf("dependencia %s: %v", nombre, err)
			}
			if f.firmaExterna != nil || f.firmaVec != nil || f.registroR5 != nil || f.servicio != base {
				t.Fatal("un fallo cambió el estado de composición")
			}
			if err := base.ComponerOriginalAutorizado(dependenciasR5PresentesPrueba{}); err != nil {
				t.Fatalf("fallo ocupó el servicio anterior: %v", err)
			}
		})
	}
}
