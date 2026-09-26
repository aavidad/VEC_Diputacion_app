package bootstrap

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	cthttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type fuenteDescriptoresCTPrueba struct{}

func (fuenteDescriptoresCTPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (vecdomain.InstantaneaAutorizacion, error) {
	return vecdomain.InstantaneaAutorizacion{}, nil
}

type concesionesDescriptoresCTPrueba struct{}

func (concesionesDescriptoresCTPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return time.Time{}, nil
}

type denegacionesDescriptoresCTPrueba struct{}

func (denegacionesDescriptoresCTPrueba) RegistrarDenegacionAutorizacionLigadaV3(context.Context, vecports.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	return nil
}

type motivosDescriptoresCTPrueba struct{}

func (motivosDescriptoresCTPrueba) ValidarReferenciaMotivoAutorizacionV2(context.Context, vecdomain.ReferenciaEntradaCatalogo, time.Time) error {
	return nil
}

func politicaDescriptoresCTPrueba(t *testing.T) politicaAutorizacionSolicitudLigadaV3Desarrollo {
	t.Helper()
	politica, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(
		fuenteDescriptoresCTPrueba{}, concesionesDescriptoresCTPrueba{}, denegacionesDescriptoresCTPrueba{}, motivosDescriptoresCTPrueba{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return politica
}

func TestDescriptoresContratacionTemporalDeclaranLosParesExactos(t *testing.T) {
	pares := []struct{ clave, accion, ruta string }{
		{"ct-analisis-registrar", ctports.AccionRegistrarAnalisis, cthttp.RutaRegistroAnalisisRRHH},
		{"ct-analisis-rectificar", ctports.AccionRectificarAnalisis, cthttp.RutaRectificacionAnalisisRRHH},
		{"ct-solicitud-crear", ctports.AccionCrearSolicitud, cthttp.RutaAltaSolicitudes},
		{"ct-asignacion-registrar", ctports.AccionRegistrarAsignacion, cthttp.RutaAsignaciones},
		{"ct-informe-juridico-emitir", ctports.AccionEmitirInformeJuridico, cthttp.RutaPreparacionesInformeJuridico},
		{"ct-subsanacion-reparo-registrar", string(ctdomain.AccionRegistrarSubsanacionReparo), cthttp.RutaSubsanacionReparos},
		{"ct-cobertura-decidir", string(ctdomain.AccionDecidirCoberturaGobernada), cthttp.RutaDecisionCobertura},
		{"ct-cobertura-rectificar", string(ctdomain.AccionRectificarCoberturaGobernada), cthttp.RutaRectificacionCobertura},
		{"ct-cuadro-consultar", ctports.AccionConsultarCuadroRRHH, cthttp.RutaConsultaCuadroRRHH},
		{"ct-expediente-consultar", ctports.AccionConsultarDetalleRRHH, cthttp.RutaConsultaDetalleRRHH},
		{"ct-cobertura-proponer", accionPropuestaCoberturaDesarrollo, cthttp.RutaPropuestaCobertura},
		{"ct-cobertura-resultado-consultar", string(ctports.AccionConsultarResultadoCobertura), cthttp.RutaResultadoCobertura},
		{"ct-cese-registrar", string(ctdomain.AccionCesarNombramiento), cthttp.RutaCesesNombramiento},
		{"ct-expediente-cerrar", string(ctdomain.AccionCerrarExpediente), cthttp.RutaCierresExpediente},
		{"ct-nombramiento-modificar", string(ctdomain.AccionModificarTrasNombramiento), cthttp.RutaModificacionesNombramiento},
		{"ct-seguimiento-cese-consultar", accionConsultarSeguimientoCeseDesarrollo, cthttp.RutaSeguimientoCese},
		{"ct-expediente-cancelar", string(ctdomain.AccionCancelarExpediente), cthttp.RutaCancelacionesExpediente},
		{"ct-cancelacion-consultar", accionConsultarCancelacionCTDesarrollo, cthttp.RutaCancelacionExpediente},
		{"ct-ginpix-confirmar", string(ctdomain.AccionConfirmarGINPIX), cthttp.RutaConfirmacionesGINPIX},
	}
	lectores := []string{"prf_ct_lector_uno", "prf_ct_lector_dos"}
	fronteras := descriptoresFronterasContratacionTemporalDesarrollo("prf_ct_prueba", lectores)
	lectores[0] = "prf_ct_alterado"
	if len(fronteras) != len(pares) {
		t.Fatalf("fronteras=%d, want %d", len(fronteras), len(pares))
	}
	for indice, par := range pares {
		frontera := fronteras[indice]
		perfilesEsperados := []string{"prf_ct_prueba"}
		if indice == 8 || indice == 9 {
			perfilesEsperados = []string{"prf_ct_lector_uno", "prf_ct_lector_dos"}
		}
		if frontera.Clave != par.clave || frontera.Metodo != http.MethodPost || frontera.Ruta != par.ruta || frontera.ClaveCapacidad != par.accion || !reflect.DeepEqual(frontera.PerfilesActivosRef, perfilesEsperados) {
			t.Fatalf("par %d = %#v", indice, frontera)
		}
		if frontera.Metodo == http.MethodHead {
			t.Fatalf("par %s declaró HEAD", par.clave)
		}
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalogo.resolver(http.MethodHead, cthttp.RutaConsultaCuadroRRHH); ok {
		t.Fatal("HEAD derivado para cuadro")
	}
	if _, ok := catalogo.resolver(http.MethodPost, cthttp.RutaConsultaCuadroRRHH); !ok {
		t.Fatal("cursor no conserva la frontera cuadro")
	}
}

func TestPerfilesConsultaContratacionTemporalSonUnConjuntoOrdenadoYValido(t *testing.T) {
	lectores := []string{"prf_ct_base", "prf_ct_externo", "prf_ct_externo"}
	perfiles := perfilesConsultaContratacionTemporalDesarrollo("prf_ct_base", lectores)
	lectores[1] = "prf_ct_alterado"
	if esperados := []string{"prf_ct_base", "prf_ct_externo"}; !reflect.DeepEqual(perfiles, esperados) {
		t.Fatalf("perfiles=%q, want %q", perfiles, esperados)
	}
	if _, err := nuevoCatalogoFronterasComunDesarrollo(
		descriptoresFronterasContratacionTemporalDesarrollo("prf_ct_base", perfiles),
	); err != nil {
		t.Fatalf("descriptores válidos rechazados: %v", err)
	}
	if perfiles := perfilesConsultaContratacionTemporalDesarrollo("", []string{"prf_ct_externo"}); perfiles != nil {
		t.Fatalf("base vacía produjo perfiles: %q", perfiles)
	}
}

func TestDescriptoresContratacionTemporalAutorizacionRechazanCruces(t *testing.T) {
	fronteras := descriptoresFronterasContratacionTemporalDesarrollo("prf_ct_prueba", []string{"prf_ct_prueba"})
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	descriptores := descriptoresAutorizacionContratacionTemporalDesarrollo(politicaDescriptoresCTPrueba(t))
	if len(descriptores) != len(fronteras) {
		t.Fatalf("autorizaciones=%d, want %d", len(descriptores), len(fronteras))
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, descriptores)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalogo.politicaPara(ctports.AccionRegistrarAnalisis, "ct-analisis-rectificar", clavePoliticaContratacionTemporalDesarrollo, ctports.AccionRegistrarAnalisis); ok {
		t.Fatal("cruce acción/frontera admitido")
	}
	if _, ok := catalogo.politicaPara(ctports.AccionRegistrarAnalisis, "ct-analisis-registrar", clavePoliticaContratacionTemporalDesarrollo, ctports.AccionRectificarAnalisis); ok {
		t.Fatal("cruce capacidad/frontera admitido")
	}
}

func TestDescriptoresMaterialContratacionTemporalSonNominales(t *testing.T) {
	descriptores := descriptoresMaterialAutorizacionContratacionTemporalDesarrollo()
	if len(descriptores) != 4 {
		t.Fatalf("materiales=%d, want 4", len(descriptores))
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores)
	if err != nil {
		t.Fatal(err)
	}
	for _, audiencia := range []string{
		ctports.AudienciaConsumoConsultaCuadroRRHHV3,
		ctports.AudienciaConsumoConsultaDetalleRRHHV3,
		ctapplication.AudienciaDespachoCorreoLlamamientoV3,
		ctapplication.AudienciaResultadoCorreoLlamamientoV3,
	} {
		if _, ok := catalogo.descriptorPara(audiencia); !ok {
			t.Fatalf("audiencia CT ausente: %s", audiencia)
		}
	}
}

// La propuesta y el resultado de cobertura pasan por el mismo PDP común que
// la decisión. Sin frontera y política propias, el perfil RRHH que registró el
// análisis recibía 403 al pedir la propuesta en cuanto se componía Bolsa.
func TestDescriptoresContratacionTemporalAutorizanLecturasDeCobertura(t *testing.T) {
	fronteras := descriptoresFronterasContratacionTemporalDesarrollo("prf_ct_prueba", []string{"prf_ct_prueba"})
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, descriptoresAutorizacionContratacionTemporalDesarrollo(politicaDescriptoresCTPrueba(t)))
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct{ frontera, accion, ruta string }{
		{"ct-cobertura-proponer", accionPropuestaCoberturaDesarrollo, cthttp.RutaPropuestaCobertura},
		{"ct-cobertura-resultado-consultar", string(ctports.AccionConsultarResultadoCobertura), cthttp.RutaResultadoCobertura},
	} {
		descriptor, ok := catalogoFronteras.resolver(http.MethodPost, caso.ruta)
		if !ok || descriptor.Clave != caso.frontera || !descriptor.admitePerfil("prf_ct_prueba") || descriptor.admitePerfil("prf_ct_ajeno") {
			t.Fatalf("%s sin frontera propia: %#v", caso.ruta, descriptor)
		}
		if _, ok := catalogo.politicaPara(caso.accion, caso.frontera, clavePoliticaContratacionTemporalDesarrollo, caso.accion); !ok {
			t.Fatalf("%s sin política en el PDP común", caso.accion)
		}
		if _, ok := catalogo.politicaPara(caso.accion, "ct-cobertura-decidir", clavePoliticaContratacionTemporalDesarrollo, caso.accion); ok {
			t.Fatalf("%s admitida por la frontera de la decisión", caso.accion)
		}
	}
}
