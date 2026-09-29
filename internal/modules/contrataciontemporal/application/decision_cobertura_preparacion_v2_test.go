package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPropuestaCoberturaProyectaPreparacionDelMismoCatalogoV2(t *testing.T) {
	vias := viasPresentacionCoberturaPrueba(2)
	vias[0].Documentos = []domain.ElementoPreparacionViaCobertura{{
		Clave: "documento_bolsa", Orden: 1, ClaveI18n: "ct.cobertura.documento_bolsa",
	}}
	vias[1].Datos = []domain.ElementoPreparacionViaCobertura{{
		Clave: "dato_sae", Orden: 1, ClaveI18n: "ct.cobertura.dato_sae",
	}}
	escenario := nuevoEscenarioPresentacionCobertura(t, vias)
	base := escenario.global.catalogo.Publicacion()
	catalogo, err := domain.PublicarCatalogoViasCobertura(domain.BorradorCatalogoViasCobertura{
		Referencia: base.Referencia, Version: base.Version,
		PublicadoEn: base.PublicadoEn, Vigencia: base.Vigencia,
		ProcedenciaRef: base.ProcedenciaRef, EsEjemplo: true, Vias: base.Vias,
	})
	if err != nil {
		t.Fatal(err)
	}
	escenario.global.catalogo = catalogo
	escenario.global.entorno.catalogo = catalogo
	escenario.gobierno.catalogo = catalogo
	escenario.global.politica = politicaCoberturaPrueba(t, catalogo, escenario.global.entorno.inicio)
	escenario.gobierno.politica = escenario.global.politica
	escenario.global.entorno.publicador.publicar = func(_ context.Context, _ ports.SolicitudConsultarCobertura) (ports.ConfirmacionPublicacionCobertura, error) {
		return ports.NuevaConfirmacionPublicacionCobertura(
			escenario.global.entorno.publicador.identidad.AutoridadRef(),
			catalogo.Publicacion(), escenario.global.entorno.reloj.Ahora(),
		)
	}

	presentacion, err := escenario.servicio.Proponer(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if escenario.accesos.total() != 2 || presentacion.PreparacionCatalogo == nil ||
		!presentacion.PreparacionCatalogo.Identidad.CoincideExactamente(catalogo.Identidad()) ||
		!presentacion.PreparacionCatalogo.EsEjemplo ||
		len(presentacion.PreparacionCatalogo.Vias) != 2 {
		t.Fatalf("preparación V2 no ligada al catálogo autorizado: %+v", presentacion.PreparacionCatalogo)
	}
	if presentacion.PreparacionCatalogo.Vias[0].Documentos[0].Clave != "documento_bolsa" ||
		presentacion.PreparacionCatalogo.Vias[1].Datos[0].Clave != "dato_sae" {
		t.Fatalf("listas por vía alteradas: %+v", presentacion.PreparacionCatalogo.Vias)
	}
	resultado, err := nuevaResultadoPropuestaCoberturaParaAdaptador(presentacion)
	if err != nil {
		t.Fatal(err)
	}
	presentacion.PreparacionCatalogo.Vias[0].Documentos[0].Clave = "documento_falso"
	datos, ok := resultado.DatosParaAdaptador()
	if !ok || datos.PreparacionCatalogo.Vias[0].Documentos[0].Clave != "documento_bolsa" {
		t.Fatal("el adaptador compartió la lista de aplicación")
	}
	datos.PreparacionCatalogo.Vias[1].Datos[0].ClaveI18n = "ct.cobertura.falsa"
	otra, ok := resultado.DatosParaAdaptador()
	if !ok || otra.PreparacionCatalogo.Vias[1].Datos[0].ClaveI18n != "ct.cobertura.dato_sae" {
		t.Fatal("el adaptador compartió la lista entregada")
	}
	falsa := presentacion
	falsa.PreparacionCatalogo = &PreparacionCatalogoPropuestaCobertura{
		Identidad: catalogo.Identidad(), Canon: catalogo.Canon(),
		Vias: []PreparacionViaPropuestaCobertura{
			{Clave: "via_global_01", Orden: 1},
			{Clave: "via_global_02", Orden: 2},
		},
	}
	if _, err := nuevaResultadoPropuestaCoberturaParaAdaptador(falsa); !errors.Is(err, ErrPresentacionPropuestaCoberturaNoConfiable) {
		t.Fatalf("aceptó V2 sin elementos publicados: %v", err)
	}
}

func TestPropuestaCoberturaV1NoExponePreparacionV2(t *testing.T) {
	escenario := nuevoEscenarioPresentacionCobertura(t, viasPresentacionCoberturaPrueba(1))
	presentacion, err := escenario.servicio.Proponer(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if presentacion.PreparacionCatalogo != nil {
		t.Fatal("la publicación V1 aparentó tener preparación V2")
	}
}
