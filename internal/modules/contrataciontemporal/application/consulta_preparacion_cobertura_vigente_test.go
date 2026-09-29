package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type autorizadorPreparacionVigentePrueba struct {
	llamadas int
	denegar  int
	err      error
}

func (a *autorizadorPreparacionVigentePrueba) AutorizarConsultaPreparacionCoberturaVigente(
	_ context.Context,
	_ ports.SolicitudResolverContextoAutorizacionAltaV3,
	_ ports.ContextoAutorizacionAltaV3,
	_ string,
	_ time.Time,
) error {
	a.llamadas++
	if a.llamadas == a.denegar {
		return a.err
	}
	return nil
}

type fuentePreparacionVigentePrueba struct {
	catalogo domain.CatalogoViasCobertura
	llamadas int
	err      error
}

func (f *fuentePreparacionVigentePrueba) ConsultarCatalogoViasCoberturaVigente(_ context.Context, _ string) (domain.CatalogoViasCobertura, error) {
	f.llamadas++
	return f.catalogo, f.err
}

func escenarioConsultaPreparacionVigentePrueba(t *testing.T, v2 bool) (
	*ServicioConsultaPreparacionCoberturaVigente,
	SolicitudConsultarPreparacionCoberturaVigente,
	*autorizadorPreparacionVigentePrueba,
	*fuentePreparacionVigentePrueba,
) {
	t.Helper()
	vias := viasPresentacionCoberturaPrueba(2)
	if v2 {
		vias[0].Documentos = []domain.ElementoPreparacionViaCobertura{{
			Clave: "documento_bolsa", Orden: 1, ClaveI18n: "ct.cobertura.documento_bolsa",
		}}
		vias[1].Datos = []domain.ElementoPreparacionViaCobertura{{
			Clave: "dato_sae", Orden: 1, ClaveI18n: "ct.cobertura.dato_sae",
		}}
	}
	escenario := nuevoEscenarioPresentacionCobertura(t, vias)
	autorizador := &autorizadorPreparacionVigentePrueba{}
	fuente := &fuentePreparacionVigentePrueba{catalogo: escenario.global.catalogo}
	servicio, err := NuevoServicioConsultaPreparacionCoberturaVigente(
		escenario.contextos, autorizador, fuente, escenario.reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	peticion := SolicitudConsultarPreparacionCoberturaVigente{
		AutenticacionRef: escenario.solicitud.AutenticacionRef,
		SesionRef:        escenario.solicitud.SesionRef,
		PerfilRef:        escenario.solicitud.PerfilRef,
		OrganizacionRef:  escenario.solicitud.OrganizacionRef,
	}
	return servicio, peticion, autorizador, fuente
}

func TestConsultaPreparacionVigenteLeeSoloCatalogoV2ConDobleAutorizacion(t *testing.T) {
	servicio, peticion, autorizador, fuente := escenarioConsultaPreparacionVigentePrueba(t, true)
	resultado, err := servicio.ConsultarParaAdaptador(context.Background(), peticion)
	if err != nil {
		t.Fatal(err)
	}
	datos, ok := resultado.DatosParaAdaptador()
	if !ok || autorizador.llamadas != 2 || fuente.llamadas != 1 ||
		!datos.Identidad.CoincideExactamente(fuente.catalogo.Identidad()) ||
		datos.Canon != domain.CanonHuellaCatalogoCoberturaV2() ||
		len(datos.Vias) != 2 || datos.Vias[0].Documentos[0].Clave != "documento_bolsa" ||
		datos.Vias[1].Datos[0].Clave != "dato_sae" {
		t.Fatalf("proyección no corresponde al catálogo V2 vigente: %+v, %d/%d", datos, autorizador.llamadas, fuente.llamadas)
	}
	datos.Vias[0].Documentos[0].Clave = "documento_falso"
	otra, ok := resultado.DatosParaAdaptador()
	if !ok || otra.Vias[0].Documentos[0].Clave != "documento_bolsa" {
		t.Fatal("el resultado compartió metadatos mutables")
	}
}

func TestConsultaPreparacionVigenteDenegacionNoFiltraCatalogo(t *testing.T) {
	for _, llamada := range []int{1, 2} {
		t.Run(string(rune('0'+llamada)), func(t *testing.T) {
			servicio, peticion, autorizador, fuente := escenarioConsultaPreparacionVigentePrueba(t, true)
			autorizador.denegar = llamada
			autorizador.err = ErrPreparacionCatalogoCoberturaNoDisponiblePerfil
			resultado, err := servicio.ConsultarParaAdaptador(context.Background(), peticion)
			if !errors.Is(err, ErrPreparacionCatalogoCoberturaNoDisponiblePerfil) ||
				!reflect.DeepEqual(resultado, ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador{}) ||
				autorizador.llamadas != llamada || fuente.llamadas != llamada-1 {
				t.Fatalf("denegación %d filtró lectura: %+v %v (%d/%d)", llamada, resultado, err, autorizador.llamadas, fuente.llamadas)
			}
		})
	}
}

func TestConsultaPreparacionVigenteNoFingeRequisitosDesdeV1(t *testing.T) {
	servicio, peticion, autorizador, fuente := escenarioConsultaPreparacionVigentePrueba(t, false)
	resultado, err := servicio.ConsultarParaAdaptador(context.Background(), peticion)
	if !errors.Is(err, ErrPresentacionPropuestaCoberturaNoDisponible) ||
		!reflect.DeepEqual(resultado, ResultadoConsultaPreparacionCoberturaVigenteParaAdaptador{}) ||
		autorizador.llamadas != 2 || fuente.llamadas != 1 {
		t.Fatalf("V1 se presentó como preparación V2: %+v %v", resultado, err)
	}
}
