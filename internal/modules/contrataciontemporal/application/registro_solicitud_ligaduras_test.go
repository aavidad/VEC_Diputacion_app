package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestRegistroSolicitudMatrizAdversarialDeLigadurasV3(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	actorAlternativo := contextoAutorizacionAltaV3PruebaConMarcas(
		t,
		escenario.instante,
		"b",
		"a",
	)
	perfilAlternativo := contextoAutorizacionAltaV3PruebaConMarcas(
		t,
		escenario.instante,
		"a",
		"b",
	)
	motivoAlternativo := escenario.motivo
	motivoAlternativo.EntradaClave = "motivo_22222222222222222222222222222222"

	type mutacion func(
		*dominiovec.DatosSolicitudAutorizacionLigadaV3,
	) (
		dominiovec.ResultadoContextoActorRegistradoV2,
		dominiovec.ReferenciaEntradaCatalogo,
	)
	casos := map[string]mutacion{
		"actor": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			datos.VinculoAutenticacionActor = actorAlternativo.Vinculo
			return actorAlternativo.Resultado, escenario.motivo
		},
		"perfil": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			datos.VinculoAutenticacionActor = perfilAlternativo.Vinculo
			return perfilAlternativo.Resultado, escenario.motivo
		},
		"organizacion": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			datos.Recurso.Ambitos["organizacion_ref"] = "organizacion:ajena"
			return escenario.contexto.Resultado, escenario.motivo
		},
		"accion": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			datos.Accion = "contratacion_temporal.solicitud.otra"
			return escenario.contexto.Resultado, escenario.motivo
		},
		"finalidad": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			datos.Finalidad = "finalidad_ajena"
			return escenario.contexto.Resultado, escenario.motivo
		},
		"recurso": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			datos.Recurso.Referencia = selloHMACRegistroPrueba(
				claveAmbitoRegistroPrueba,
				"e",
			)
			return escenario.contexto.Resultado, escenario.motivo
		},
		"motivo": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			datos.ReferenciaMotivo = motivoAlternativo
			return escenario.contexto.Resultado, motivoAlternativo
		},
		"flujo": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			datos.Recurso.Atributos["flujo_ref"] = "flujo:ajeno"
			return escenario.contexto.Resultado, escenario.motivo
		},
		"correlacion": func(datos *dominiovec.DatosSolicitudAutorizacionLigadaV3) (
			dominiovec.ResultadoContextoActorRegistradoV2,
			dominiovec.ReferenciaEntradaCatalogo,
		) {
			correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
				context.Background(),
				&generadorReferenciasDoble{
					correlacion: "correlacion_22222222222222222222222222222222",
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			datos.Correlacion = correlacion
			return escenario.contexto.Resultado, escenario.motivo
		},
	}

	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			servicio, d := construirServicioRegistro(t, escenario)
			d.autorizador.transformar = func(
				original dominiovec.SolicitudAutorizacionLigadaV3,
				_ dominiovec.ResultadoContextoActorRegistradoV2,
			) (
				dominiovec.SolicitudAutorizacionLigadaV3,
				dominiovec.ResultadoContextoActorRegistradoV2,
				dominiovec.ReferenciaEntradaCatalogo,
			) {
				datos, err := original.Datos()
				if err != nil {
					t.Fatal(err)
				}
				resultado, motivo := mutar(&datos)
				alterada, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
				if err != nil {
					t.Fatal(err)
				}
				return alterada, resultado, motivo
			}

			_, err := servicio.Registrar(context.Background(), escenario.solicitud)
			if !errors.Is(err, ports.ErrAutorizacionDenegada) ||
				d.transaccion.llamadas != 0 {
				t.Fatalf("ligadura %s cruzada produjo efecto: %v", nombre, err)
			}
		})
	}
}
