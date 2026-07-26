package application

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestOperacionAnalisisAutorizacionLigaAtributosDurablesExactos(
	t *testing.T,
) {
	casos := []struct {
		nombre    string
		operacion ports.TipoOperacionAnalisis
	}{
		{"registro", ports.OperacionRegistrarAnalisis},
		{"rectificacion", ports.OperacionRectificarAnalisis},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioOperacionAnalisisSaneado(
				t, caso.operacion, "-atributos-v3-"+caso.nombre,
			)
			servicio, dependencias :=
				construirServicioOperacionAnalisisSaneado(t, escenario)
			if caso.operacion == ports.OperacionRegistrarAnalisis {
				_, err := servicio.Registrar(
					context.Background(), escenario.registrar,
				)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				_, err := servicio.Rectificar(
					context.Background(), escenario.rectificar,
				)
				if err != nil {
					t.Fatal(err)
				}
			}
			evidencia, err := dependencias.transaccion.orden.Datos()
			if err != nil {
				t.Fatal(err)
			}
			fuentes, err := evidencia.OrdenConsumoFuentes.Datos()
			if err != nil {
				t.Fatal(err)
			}
			huellaAnalisis, err := ports.HuellaAnalisisDerivadoO3(
				evidencia.SolicitudArtefacto, evidencia.Artefacto,
			)
			if err != nil {
				t.Fatal(err)
			}
			solicitud, err := evidencia.SolicitudV3.Datos()
			if err != nil {
				t.Fatal(err)
			}
			motivoEsperado := ports.ValorMotivoRectificacionNoAplica
			if caso.operacion == ports.OperacionRectificarAnalisis {
				motivoEsperado = string(escenario.motivoRectificacion)
			}
			esperados := map[string]string{
				ports.AtributoOperacionAnalisis:       string(caso.operacion),
				ports.AtributoVersionAnalisis:         strconv.FormatUint(evidencia.SolicitudPolitica.VersionExpediente, 10),
				ports.AtributoPoliticaAnalisisRef:     evidencia.Politica.DefinicionRef,
				ports.AtributoPoliticaAnalisisVersion: strconv.FormatUint(evidencia.Politica.Version, 10),
				ports.AtributoPoliticaAnalisisHuella:  evidencia.Politica.HuellaSHA256,
				ports.AtributoArtefactoAnalisisRef:    evidencia.Politica.ArtefactoRef,
				ports.AtributoArtefactoAnalisisHuella: evidencia.Politica.ArtefactoHuellaSHA256,
				ports.AtributoAnalisisDerivadoHuella:  huellaAnalisis,
				ports.AtributoConjuntoFuentesHuella:   fuentes.HuellaSHA256,
				ports.AtributoUnidadPoliticaRef:       evidencia.Politica.UnidadRef,
				ports.AtributoMotivoRectificacion:     motivoEsperado,
				ports.AtributoHuellaSemanticaAnalisis: solicitud.Recurso.Atributos[ports.AtributoHuellaSemanticaAnalisis],
				ports.AtributoSegregacionAnalisis:     strconv.FormatBool(evidencia.Politica.ExigeActorDistinto),
			}
			if !reflect.DeepEqual(solicitud.Recurso.Atributos, esperados) {
				t.Fatalf(
					"atributos V3 no exactos: obtenidos=%#v esperados=%#v",
					solicitud.Recurso.Atributos, esperados,
				)
			}
		})
	}
}

func TestOrdenOperacionAnalisisRechazaAtributosDurablesAlterados(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t, ports.OperacionRegistrarAnalisis, "-ataque-atributos-v3",
	)
	servicio, dependencias :=
		construirServicioOperacionAnalisisSaneado(t, escenario)
	if _, err := servicio.Registrar(
		context.Background(), escenario.registrar,
	); err != nil {
		t.Fatal(err)
	}
	evidencia, err := dependencias.transaccion.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		mutar  func(map[string]string)
	}{
		{
			"sin_huella_fuentes",
			func(a map[string]string) {
				delete(a, ports.AtributoConjuntoFuentesHuella)
			},
		},
		{
			"huella_analisis_ajena",
			func(a map[string]string) {
				a[ports.AtributoAnalisisDerivadoHuella] =
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
						"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			},
		},
		{
			"unidad_ajena",
			func(a map[string]string) {
				a[ports.AtributoUnidadPoliticaRef] =
					"unidad:ajena-sintetica-001"
			},
		},
		{
			"motivo_inesperado",
			func(a map[string]string) {
				a[ports.AtributoMotivoRectificacion] =
					"contratacion_temporal.analisis.rectificacion.ajena"
			},
		},
		{
			"atributo_extra",
			func(a map[string]string) { a["atributo_ajeno"] = "valor" },
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datosSolicitud, errDatos := evidencia.SolicitudV3.Datos()
			if errDatos != nil {
				t.Fatal(errDatos)
			}
			caso.mutar(datosSolicitud.Recurso.Atributos)
			solicitudAlterada, errNueva :=
				dominiovec.NuevaSolicitudAutorizacionLigadaV3(
					datosSolicitud,
				)
			if errNueva != nil {
				t.Fatal(errNueva)
			}
			decision, confirmacion, errConcesion :=
				concesionAutorizacionV3Prueba(
					t,
					solicitudAlterada,
					evidencia.ContextoAutorizacion.Resultado,
					evidencia.Politica.MotivoAutorizacion,
					evidencia.InstanteEfecto,
					"dec_abcdef0123456789abcdef0123456789",
					true,
				)
			if errConcesion != nil {
				t.Fatal(errConcesion)
			}
			_, errOrden := ports.NuevaOrdenConfirmarOperacionAnalisis(
				ports.DatosOrdenConfirmarOperacionAnalisis{
					SolicitudContexto:    evidencia.SolicitudContexto,
					ContextoAutorizacion: evidencia.ContextoAutorizacion,
					SolicitudArtefacto:   evidencia.SolicitudArtefacto,
					Artefacto:            evidencia.Artefacto,
					OrdenConsumoFuentes:  evidencia.OrdenConsumoFuentes,
					SolicitudPreparacion: evidencia.SolicitudPreparacion,
					Preparacion:          evidencia.Preparacion,
					SolicitudPolitica:    evidencia.SolicitudPolitica,
					Politica:             evidencia.Politica,
					SolicitudV3:          solicitudAlterada,
					DecisionV3:           decision,
					ConfirmacionV3:       confirmacion,
					InstanteEfecto:       evidencia.InstanteEfecto,
					ExpedienteSiguiente:  evidencia.ExpedienteSiguiente,
				},
			)
			if !errors.Is(errOrden, ports.ErrOrdenOperacionAnalisisInvalida) {
				t.Fatal("la orden aceptó atributos V3 alterados")
			}
		})
	}
}
