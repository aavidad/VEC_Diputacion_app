package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type relojPreparacionVigenteFuentePrueba struct{ instante time.Time }

func (r relojPreparacionVigenteFuentePrueba) AhoraGobiernoOperacionCobertura(context.Context) (time.Time, error) {
	return r.instante, nil
}

type resolutorPreparacionVigenteFuentePrueba struct {
	llamadas     int
	organizacion string
	expediente   string
	version      uint64
	accion       domain.ClaveCatalogo
	publicacion  cobertura.PublicacionGobiernoOperacionCobertura
	err          error
}

func (r *resolutorPreparacionVigenteFuentePrueba) ResolverGobiernoOperacionCobertura(
	_ context.Context,
	solicitud cobertura.SolicitudResolucionGobiernoOperacionCobertura,
) (cobertura.PublicacionGobiernoOperacionCobertura, error) {
	r.llamadas++
	r.organizacion, r.expediente, r.version, r.accion, _, _ = solicitud.Coordenadas()
	if r.err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, r.err
	}
	publicacion := r.publicacion
	publicacion.OrganizacionRef = r.organizacion
	publicacion.ExpedienteRef = r.expediente
	publicacion.VersionExpediente = r.version
	return publicacion, nil
}

func TestFuentePreparacionVigenteUsaResolutorDurableSinExpedienteCliente(t *testing.T) {
	resolutor := &resolutorPreparacionVigenteFuentePrueba{err: errors.New("detalle reservado del resolutor")}
	reloj := relojPreparacionVigenteFuentePrueba{instante: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)}
	fuente, err := nuevaFuentePreparacionCoberturaVigenteDesarrollo(resolutor, reloj)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := fuente.ConsultarCatalogoViasCoberturaVigente(
		context.Background(), organizacionAltaContratacionTemporalDesarrollo,
	)
	if !errors.Is(err, application.ErrPresentacionPropuestaCoberturaNoDisponible) ||
		catalogo.Validar() == nil || resolutor.llamadas != 1 ||
		resolutor.organizacion != organizacionAltaContratacionTemporalDesarrollo ||
		resolutor.expediente != expedienteSondeoGobiernoCT || resolutor.version != 1 ||
		resolutor.accion != domain.AccionDecidirCoberturaGobernada {
		t.Fatalf("la fuente usó coordenadas ajenas o filtró fallo: %q/%q/%d/%q, %v", resolutor.organizacion, resolutor.expediente, resolutor.version, resolutor.accion, err)
	}
	_, err = fuente.ConsultarCatalogoViasCoberturaVigente(context.Background(), "organizacion:ajena")
	if !errors.Is(err, application.ErrPresentacionPropuestaCoberturaNoDisponible) || resolutor.llamadas != 1 {
		t.Fatal("la fuente consultó una organización no configurada")
	}
}

func TestFuentePreparacionVigenteRestauraPublicacionV2Exacta(t *testing.T) {
	vias := viasCoberturaPredeterminadasCT()
	for indice := range vias {
		vias[indice].Ejemplo = true
	}
	vias[0].Documentos = []domain.ElementoPreparacionViaCobertura{{
		Clave: "documento_bolsa", Orden: 1, ClaveI18n: "ct.cobertura.documento_bolsa",
	}}
	vias[1].Datos = []domain.ElementoPreparacionViaCobertura{{
		Clave: "dato_sae", Orden: 1, ClaveI18n: "ct.cobertura.dato_sae",
	}}
	soporte := &soporteAltaContratacionTemporalDesarrollo{
		motivoDecisionCobertura:      referenciaMotivoAutorizacionCoberturaDesarrollo("decision"),
		motivoRectificacionCobertura: referenciaMotivoAutorizacionCoberturaDesarrollo("rectificacion"),
	}
	deseado, err := nuevoGobiernoCoberturaParaViasCT(soporte, vias, "prealta-prueba", 3)
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := domain.RestaurarCatalogoViasCobertura(deseado.catalogo)
	if err != nil {
		t.Fatal(err)
	}
	politica, err := domain.RestaurarPoliticaDecisionCobertura(deseado.politica, catalogo)
	if err != nil {
		t.Fatal(err)
	}
	resolutor := &resolutorPreparacionVigenteFuentePrueba{
		publicacion: cobertura.PublicacionGobiernoOperacionCobertura{
			Catalogo: catalogo, Politica: politica,
			PoliticaActuacion: deseado.actuaciones[0],
		},
	}
	reloj := relojPreparacionVigenteFuentePrueba{instante: time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)}
	fuente, err := nuevaFuentePreparacionCoberturaVigenteDesarrollo(resolutor, reloj)
	if err != nil {
		t.Fatal(err)
	}
	leido, err := fuente.ConsultarCatalogoViasCoberturaVigente(
		context.Background(), organizacionAltaContratacionTemporalDesarrollo,
	)
	if err != nil || !leido.Identidad().CoincideExactamente(catalogo.Identidad()) ||
		!leido.Publicacion().EsEjemplo || leido.Canon() != domain.CanonHuellaCatalogoCoberturaV2() ||
		resolutor.llamadas != 1 {
		t.Fatalf("perdió la publicación V2 exacta: %v, %+v", err, leido.Identidad())
	}
	leeVias := leido.Vias()
	if leeVias[0].Documentos[0].Clave != "documento_bolsa" ||
		leeVias[1].Datos[0].Clave != "dato_sae" {
		t.Fatal("la fuente sustituyó requisitos por valores de composición")
	}
}
