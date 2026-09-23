package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type fuenteAutorizacionDescriptoresBolsaPrueba struct{}

func (fuenteAutorizacionDescriptoresBolsaPrueba) ObtenerInstantaneaAutorizacion(
	context.Context, string, string,
) (dominiovec.InstantaneaAutorizacion, error) {
	return dominiovec.InstantaneaAutorizacion{}, nil
}

type registroConcesionesDescriptoresBolsaPrueba struct{}

func (registroConcesionesDescriptoresBolsaPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(
	context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3,
) (time.Time, error) {
	return time.Time{}, nil
}

type registroDenegacionesDescriptoresBolsaPrueba struct{}

func (registroDenegacionesDescriptoresBolsaPrueba) RegistrarDenegacionAutorizacionLigadaV3(
	context.Context, puertosvec.OrdenRegistroDenegacionAutorizacionLigadaV3,
) error {
	return nil
}

type validadorMotivosDescriptoresBolsaPrueba struct{}

func (validadorMotivosDescriptoresBolsaPrueba) ValidarReferenciaMotivoAutorizacionV2(
	context.Context, dominiovec.ReferenciaEntradaCatalogo, time.Time,
) error {
	return nil
}

func politicaDescriptoresBolsaPrueba(t *testing.T) politicaAutorizacionSolicitudLigadaV3Desarrollo {
	t.Helper()
	politica, err := nuevaPoliticaAutorizacionSolicitudLigadaV3Desarrollo(
		fuenteAutorizacionDescriptoresBolsaPrueba{},
		registroConcesionesDescriptoresBolsaPrueba{},
		registroDenegacionesDescriptoresBolsaPrueba{},
		validadorMotivosDescriptoresBolsaPrueba{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return politica
}

func TestDescriptoresBorradorLlamamientoBolsaFronterasExactas(t *testing.T) {
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo("prf_bolsa_bback")
	if err != nil {
		t.Fatal(err)
	}
	if len(fronteras) != 9 {
		t.Fatalf("fronteras = %d, se esperan 9", len(fronteras))
	}
	for _, frontera := range fronteras {
		if len(frontera.PerfilesActivosRef) != 1 || frontera.PerfilesActivosRef[0] != "prf_bolsa_bback" {
			t.Fatalf("perfiles de frontera %q = %#v", frontera.Clave, frontera.PerfilesActivosRef)
		}
	}
	catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaBorradoresLlamamiento); !ok {
		t.Fatal("POST crear no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaBorradoresLlamamiento+"/borrador-llamamiento:alta:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"); !ok {
		t.Fatal("GET detalle de un segmento no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/situacion"); !ok {
		t.Fatal("POST B2 no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/operaciones"); !ok {
		t.Fatal("POST B8 no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/operaciones"); !ok {
		t.Fatal("GET B8 no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/contactos"); !ok {
		t.Fatal("POST B3 no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/contactos"); !ok {
		t.Fatal("GET B3 no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaBolsasGestion+"/bolsa:01/candidatos"); !ok {
		t.Fatal("GET B5 no quedó declarado para consultar contactos")
	}
	if _, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/datos-contacto"); !ok {
		t.Fatal("GET B4 no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaEmisionesLlamamiento); !ok {
		t.Fatal("POST B7 no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaEmisionesLlamamiento); !ok {
		t.Fatal("GET de recuperación B7 no quedó declarado")
	}
	if _, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaBolsasGestion+"/bolsa:01/candidatos/participacion:01/situacion"); ok {
		t.Fatal("GET B3 abrió un subrecurso ajeno")
	}
	for _, caso := range []struct{ metodo, ruta string }{
		{http.MethodHead, bolsahttp.RutaBorradoresLlamamiento + "/borrador"},
		{http.MethodGet, bolsahttp.RutaBorradoresLlamamiento},
		{http.MethodGet, bolsahttp.RutaBorradoresLlamamiento + "/uno/dos"},
	} {
		if _, ok := catalogo.resolver(caso.metodo, caso.ruta); ok {
			t.Fatalf("se acreditó frontera indebida: %s %s", caso.metodo, caso.ruta)
		}
	}
	if _, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(""); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
		t.Fatalf("perfil ausente = %v", err)
	}
}

func TestDescriptoresBorradorLlamamientoBolsaAutorizacionExacta(t *testing.T) {
	politica := politicaDescriptoresBolsaPrueba(t)
	fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo("prf_bolsa_bback")
	if err != nil {
		t.Fatal(err)
	}
	catalogoFronteras, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
	if err != nil {
		t.Fatal(err)
	}
	descriptores, err := descriptoresAutorizacionBorradorLlamamientoBolsaDesarrollo(politica)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := nuevoCatalogoAutorizacionComunDesarrollo(catalogoFronteras, descriptores)
	if err != nil {
		t.Fatal(err)
	}
	if p, ok := catalogo.politicaPara(puertosbolsa.AccionCrearBorradorLlamamientoInterno, claveFronteraCrearBorradorLlamamientoBolsa, clavePoliticaBorradorLlamamientoBolsaDesarrollo, claveCapacidadCrearBorradorLlamamientoBolsa); !ok || !p.valida() {
		t.Fatal("crear no conservó la política Bolsa completa")
	}
	if p, ok := catalogo.politicaPara(puertosbolsa.AccionConsultarBorradorLlamamientoInterno, claveFronteraConsultarBorradorLlamamientoBolsa, clavePoliticaBorradorLlamamientoBolsaDesarrollo, claveCapacidadConsultarBorradorLlamamientoBolsa); !ok || !p.valida() {
		t.Fatal("consultar no conservó la política Bolsa completa")
	}
	if p, ok := catalogo.politicaPara(puertosbolsa.AccionCambiarSituacionParticipacion, claveFronteraSituacionParticipacionBolsa, clavePoliticaBorradorLlamamientoBolsaDesarrollo, claveCapacidadSituacionParticipacionBolsa); !ok || !p.valida() {
		t.Fatal("cambio B2 no conservó la política Bolsa completa")
	}
	if p, ok := catalogo.politicaPara(puertosbolsa.AccionRegistrarContactoParticipacion, claveFronteraSituacionParticipacionBolsa, clavePoliticaBorradorLlamamientoBolsaDesarrollo, claveCapacidadSituacionParticipacionBolsa); !ok || !p.valida() {
		t.Fatal("registro B3 no conservó la política Bolsa completa")
	}
	if p, ok := catalogo.politicaPara(puertosbolsa.AccionRegistrarDatosContactoParticipacion, claveFronteraSituacionParticipacionBolsa, clavePoliticaBorradorLlamamientoBolsaDesarrollo, claveCapacidadSituacionParticipacionBolsa); !ok || !p.valida() {
		t.Fatal("registro B4 no conservó la política Bolsa completa")
	}
	for _, caso := range []struct{ accion, frontera, capacidad string }{
		{puertosbolsa.AccionCrearBorradorLlamamientoInterno, claveFronteraConsultarBorradorLlamamientoBolsa, claveCapacidadConsultarBorradorLlamamientoBolsa},
		{puertosbolsa.AccionConsultarBorradorLlamamientoInterno, claveFronteraCrearBorradorLlamamientoBolsa, claveCapacidadCrearBorradorLlamamientoBolsa},
		{puertosbolsa.AccionCrearBorradorLlamamientoInterno, claveFronteraCrearBorradorLlamamientoBolsa, claveCapacidadConsultarBorradorLlamamientoBolsa},
		{puertosbolsa.AccionConsultarBorradorLlamamientoInterno, claveFronteraConsultarBorradorLlamamientoBolsa, claveCapacidadCrearBorradorLlamamientoBolsa},
	} {
		if _, ok := catalogo.politicaPara(caso.accion, caso.frontera, clavePoliticaBorradorLlamamientoBolsaDesarrollo, caso.capacidad); ok {
			t.Fatalf("acción %q se admitió en frontera/capacidad %q/%q", caso.accion, caso.frontera, caso.capacidad)
		}
	}
}

func TestDescriptoresBorradorLlamamientoBolsaMaterialExacto(t *testing.T) {
	descriptores := descriptoresMaterialBorradorLlamamientoBolsaDesarrollo()
	if len(descriptores) != 7 {
		t.Fatalf("materiales = %d, se esperan 7", len(descriptores))
	}
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores)
	if err != nil {
		t.Fatal(err)
	}
	for _, esperado := range []descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: puertosbolsa.AudienciaCrearBorradorLlamamientoInterno, Dominio: dominioMaterialCrearBorradorLlamamientoBolsa, Prefijo: prefijoMaterialCrearBorradorLlamamientoBolsa, ProveedorNominal: "proveedor-material-borrador-llamamiento-bolsa-crear"},
		{Audiencia: puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno, Dominio: dominioMaterialConsultarBorradorLlamamientoBolsa, Prefijo: prefijoMaterialConsultarBorradorLlamamientoBolsa, ProveedorNominal: "proveedor-material-borrador-llamamiento-bolsa-consultar"},
		{Audiencia: puertosbolsa.AudienciaCambiarSituacionParticipacion, Dominio: dominioMaterialSituacionParticipacionBolsa, Prefijo: prefijoMaterialSituacionParticipacionBolsa, ProveedorNominal: "proveedor-material-situacion-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaRegistrarContactoParticipacion, Dominio: dominioMaterialContactoParticipacionBolsa, Prefijo: prefijoMaterialContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaConsultarContactoParticipacion, Dominio: dominioMaterialConsultaContactoParticipacionBolsa, Prefijo: prefijoMaterialConsultaContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-consulta-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaRegistrarDatosContactoParticipacion, Dominio: dominioMaterialDatosContactoParticipacionBolsa, Prefijo: prefijoMaterialDatosContactoParticipacionBolsa, ProveedorNominal: "proveedor-material-datos-contacto-participacion-bolsa"},
		{Audiencia: puertosbolsa.AudienciaEmitirLlamamiento, Dominio: dominioMaterialEmisionLlamamientoBolsa, Prefijo: prefijoMaterialEmisionLlamamientoBolsa, ProveedorNominal: "proveedor-material-emision-llamamiento-bolsa"},
	} {
		actual, ok := catalogo.descriptorPara(esperado.Audiencia)
		if !ok || actual != esperado {
			t.Fatalf("descriptor de %q = %#v", esperado.Audiencia, actual)
		}
		if actual.Prefijo == puertosbolsa.AccionCrearBorradorLlamamientoInterno || actual.Prefijo == puertosbolsa.AccionConsultarBorradorLlamamientoInterno {
			t.Fatalf("prefijo de %q deriva de una acción", esperado.Audiencia)
		}
	}
	if _, ok := catalogo.descriptorPara("vec_contratacion_temporal.confirmar_alta_atestada.v1"); ok {
		t.Fatal("el catálogo B-BACK declaró material CT")
	}
}
