package bootstrap

import (
	"context"
	"errors"
	"testing"

	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// El adaptador PostgreSQL real ofrece la consulta CT172 y el registro CT176.
var _ registroFirmaV2DurableDesarrollo = (*postgrescontratacion.RegistroFirmasVerificadasPostgreSQL)(nil)

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
	return dependenciasFirmaR5Desarrollo{original: p, registro: p, descriptorPlan: p, emisorPlan: p, consulta: p,
		autorizar: p, verificador: p, pdfAnterior: p, competencia: p, politicaFirmantes: p}
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

func (dependenciasR5PresentesPrueba) RegistrarFirmaVerificadaV2(context.Context, ports.MaterialFirmaVerificadaV2, ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	return ports.ReciboFirmaDocumento{}, nil
}
func (dependenciasR5PresentesPrueba) RegistrarFirmaConPlanV2(context.Context, ports.MaterialFirmaVerificadaV2, ports.CapacidadFirmaConPlanV2) (ports.ReciboFirmaDocumento, error) {
	return ports.ReciboFirmaDocumento{}, nil
}
func (dependenciasR5PresentesPrueba) DescriptorPlanFijadoFirmaV2(context.Context, ports.MaterialFirmaVerificadaV2) (ports.DescriptorPlanFijadoFirmaV2, error) {
	return ports.DescriptorPlanFijadoFirmaV2{}, nil
}
func (dependenciasR5PresentesPrueba) AutorizarMaterialPlanFirmaV2(context.Context, ports.MaterialFirmaVerificadaV2, vd.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}
func (dependenciasR5PresentesPrueba) ConsultarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2, ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	return ports.LecturaFirmasR5V2{}, nil
}
func (dependenciasR5PresentesPrueba) AutorizarConsultaFirmasR5V2(context.Context, ports.MaterialConsultaFirmasR5V2) (ports.CapacidadConsultaFirmasR5V2, error) {
	return ports.CapacidadConsultaFirmasR5V2{}, nil
}
func (dependenciasR5PresentesPrueba) AutorizarFirmaVerificadaV2(context.Context, ports.MaterialFirmaVerificadaV2) (ports.CapacidadFirmaVerificadaV2, error) {
	return ports.CapacidadFirmaVerificadaV2{}, nil
}
func (dependenciasR5PresentesPrueba) ObtenerPerfilActivoOperadorFirmaV2(context.Context) (string, error) {
	return "prf_operador_prueba", nil
}
func (dependenciasR5PresentesPrueba) ObtenerPDFFirmaAnterior(context.Context, ports.SolicitudPDFFirmaAnterior) (ports.PDFFirmaAnterior, error) {
	return ports.PDFFirmaAnterior{}, nil
}
func (dependenciasR5PresentesPrueba) VerificarFirmas(context.Context, docports.SolicitudVerificacionFirma) (docports.VerificacionFirmasDocumento, error) {
	return docports.VerificacionFirmasDocumento{}, nil
}

type baseR5ComposicionPrueba struct {
	ports.FuenteCircuitoFirma
	ports.RegistroFirmasDocumento
	ports.AutorizadorFirmaDocumento
}

func (dependenciasR5PresentesPrueba) CustodiarFirmado(context.Context, ports.OrdenCustodiaFirmado) (ports.DocumentoCustodiado, error) {
	return ports.DocumentoCustodiado{}, nil
}

func baseR5MontajePrueba(t *testing.T) *ctapplication.ServicioFirmaDocumento {
	t.Helper()
	p := baseR5ComposicionPrueba{}
	base, err := ctapplication.NuevoServicioFirmaDocumento(p, p, p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := base.ComponerCustodia(dependenciasR5PresentesPrueba{}, map[string]string{"informe_definitivo": "informe_firmado"}); err != nil {
		t.Fatal(err)
	}
	return base
}

func TestComposicionFirmasR5PublicaAmbasViasV2SinOcuparBase(t *testing.T) {
	base := baseR5MontajePrueba(t)
	f := &firmaDocumentoCTDesarrollo{servicio: base, custodiaR5Compuesta: true}
	if err := f.componerFirmasR5(dependenciasCompletasR5Prueba()); err != nil || f.firmaExterna == nil || f.firmaVec == nil {
		t.Fatalf("montaje V2: %v", err)
	}
	if err := base.ComponerOriginalAutorizado(dependenciasR5PresentesPrueba{}); err != nil {
		t.Fatalf("el montaje mutó el servicio legado: %v", err)
	}
	if err := f.componerFirmasR5(dependenciasCompletasR5Prueba()); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) {
		t.Fatalf("segundo montaje: %v", err)
	}
}

func TestComposicionFirmasR5CadaDependenciaV2EsObligatoria(t *testing.T) {
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
	} {
		t.Run(nombre, func(t *testing.T) {
			base := baseR5MontajePrueba(t)
			f := &firmaDocumentoCTDesarrollo{servicio: base, custodiaR5Compuesta: true}
			d := dependenciasCompletasR5Prueba()
			omitir(&d)
			if err := f.componerFirmasR5(d); !errors.Is(err, errFirmaDocumentoCTDesarrolloNoDisponible) || f.firmaExterna != nil || f.firmaVec != nil {
				t.Fatalf("dependencia %s: %v", nombre, err)
			}
			if err := f.componerFirmasR5(dependenciasCompletasR5Prueba()); err != nil {
				t.Fatalf("fallo ocupó original/base: %v", err)
			}
		})
	}
}

// registroDirectoContado tiene también el registro directo de CT172, como el
// adaptador PostgreSQL real; el montaje no debe poder alcanzarlo.
type registroDirectoContado struct {
	dependenciasR5PresentesPrueba
	directas, consultas int
}

func (r *registroDirectoContado) RegistrarFirmaVerificadaV2(context.Context, ports.MaterialFirmaVerificadaV2, ports.CapacidadFirmaVerificadaV2) (ports.ReciboFirmaDocumento, error) {
	r.directas++
	return ports.ReciboFirmaDocumento{}, nil
}
func (r *registroDirectoContado) ConsultarFirmasAutorizadasV2(context.Context, ports.MaterialConsultaFirmasR5V2, ports.CapacidadConsultaFirmasR5V2) (ports.LecturaFirmasR5V2, error) {
	r.consultas++
	return ports.LecturaFirmasR5V2{}, nil
}

// La garantía principal es de compilación: registroFirmaV2DurableDesarrollo
// no ofrece RegistrarFirmaVerificadaV2, así que d.registro no puede pasarse a
// las vías R5. Esta prueba fija además el comportamiento en ejecución.
func TestComposicionFirmasR5RegistraSiempreConPlanCT176(t *testing.T) {
	f := &firmaDocumentoCTDesarrollo{servicio: baseR5MontajePrueba(t), custodiaR5Compuesta: true}
	d := dependenciasCompletasR5Prueba()
	durable := &registroDirectoContado{}
	d.registro = durable
	if err := f.componerFirmasR5(d); err != nil {
		t.Fatal(err)
	}
	if _, ok := f.registroR5.(*firmaautorizacionv2.RegistroConPlanV2); !ok {
		t.Fatalf("las vías R5 no reciben el registro con plan: %T", f.registroR5)
	}
	if _, err := f.registroR5.RegistrarFirmaVerificadaV2(t.Context(), ports.MaterialFirmaVerificadaV2{}, ports.CapacidadFirmaVerificadaV2{}); err == nil {
		t.Fatal("registro sin material aceptado")
	}
	if _, err := f.registroR5.ConsultarFirmasAutorizadasV2(t.Context(), ports.MaterialConsultaFirmasR5V2{}, ports.CapacidadConsultaFirmasR5V2{}); err != nil || durable.consultas != 1 {
		t.Fatalf("consulta no delegada: %d, %v", durable.consultas, err)
	}
	// El adaptador de consulta que recibe el decorador nunca escribe por CT172.
	c := consultaFirmasV2SinRegistroDirecto{durable}
	if _, err := c.RegistrarFirmaVerificadaV2(t.Context(), ports.MaterialFirmaVerificadaV2{}, ports.CapacidadFirmaVerificadaV2{}); !errors.Is(err, ports.ErrRegistroFirmaDocumentoNoDisponible) {
		t.Fatalf("registro directo no rechazado: %v", err)
	}
	if durable.directas != 0 {
		t.Fatalf("el montaje llamó %d veces al registro directo CT172", durable.directas)
	}
}
