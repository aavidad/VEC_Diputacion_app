package calculoexperienciaoficial

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestRectificacionSoloPuedeEjecutarseEnServicioInternoAlto(t *testing.T) {
	externo := nuevoEscenarioServicioPrueba(t, perfilExternoOrdinario, false)
	ordenExterna := ordenRectificacionPrueba(t, externo.datosOrden)
	if _, err := externo.servicio.Ejecutar(context.Background(), ordenExterna); !errors.Is(
		err, ErrOrdenInvalida,
	) || len(externo.exigidor.llamadas) != 0 || externo.fuente.llamadas != 0 {
		t.Fatalf("la rectificacion externa alcanzo infraestructura: %v", err)
	}

	interno := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	ordenInterna := ordenRectificacionPrueba(t, interno.datosOrden)
	if _, err := interno.servicio.Ejecutar(context.Background(), ordenInterna); err != nil {
		t.Fatal(err)
	}
	if len(interno.exigidor.llamadas) != 2 ||
		interno.exigidor.llamadas[1].recurso.Tipo != tipoRecursoRectificacion ||
		interno.exigidor.llamadas[1].recurso.Atributos["predecesor_ref"] !=
			"recibo:calculo:predecesor:1" || interno.confirmador.llamadas != 1 {
		t.Fatal("la rectificacion interna no uso su recurso y predecesor separados")
	}
}

func TestGarantiaExternaBajaSeRechazaAntesDeLeerFuente(t *testing.T) {
	t.Run("actor_y_vinculo", func(t *testing.T) {
		escenario := nuevoEscenarioServicioPrueba(t, perfilExternoOrdinario, false)
		actor, vinculo := sesionPrueba(
			t, escenario.ahora, dominiovec.SuperficieAutenticacionExternaPersonalV1,
			dominiovec.AuthAssuranceLow,
		)
		datos := escenario.datosOrden
		datos.ContextoActor, datos.VinculoAutenticacionActor = actor, vinculo
		orden := debePrueba(NuevaOrdenConfiable(datos))
		_, err := escenario.servicio.Ejecutar(context.Background(), orden)
		if !errors.Is(err, ErrOrdenInvalida) || len(escenario.exigidor.llamadas) != 0 ||
			escenario.fuente.llamadas != 0 {
			t.Fatalf("sesion low alcanzo infraestructura: %v", err)
		}
	})
	t.Run("decision_pdp", func(t *testing.T) {
		escenario := nuevoEscenarioServicioPrueba(t, perfilExternoOrdinario, false)
		escenario.exigidor.garantiaDecision = dominiovec.AuthAssuranceLow
		_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
		if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
			escenario.fuente.llamadas != 0 || escenario.confirmador.llamadas != 0 {
			t.Fatalf("decision low alcanzo la fuente: %v", err)
		}
	})
}

func TestEvidenciaDebeSerPosteriorALaFronteraQueAutoriza(t *testing.T) {
	for _, fase := range []int{1, 2} {
		t.Run(string(rune('0'+fase)), func(t *testing.T) {
			escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
			escenario.exigidor.antiguaEn = fase
			_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
				escenario.confirmador.llamadas != 0 {
				t.Fatalf("evidencia antigua produjo efecto: %v", err)
			}
			if fase == 1 && escenario.fuente.llamadas != 0 {
				t.Fatal("una evidencia de lectura antigua alcanzo la fuente")
			}
		})
	}
}

func TestFuenteRechazaRolesReutilizadosYPruebaCaducada(t *testing.T) {
	t.Run("auditoria_como_sujeto", func(t *testing.T) {
		escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
		referencia := escenario.datosOrden.Selector.SujetoPseudonimo.Referencia()
		escenario.fuente.resultado.Auditoria = referenciaPrueba(t, referencia, 2)
		_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
		if !errors.Is(err, ErrFuenteNoConfiable) ||
			len(escenario.exigidor.llamadas) != 1 || escenario.confirmador.llamadas != 0 {
			t.Fatalf("se acepto confusion de roles: %v", err)
		}
	})
	t.Run("prueba_caducada", func(t *testing.T) {
		escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
		escenario.fuente.resultado.Prueba.ValidaHasta = escenario.ahora
		_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
		if !errors.Is(err, ErrFuenteNoConfiable) || escenario.confirmador.llamadas != 0 {
			t.Fatalf("se acepto prueba caducada: %v", err)
		}
	})
	t.Run("sujeto_mixto", func(t *testing.T) {
		escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
		escenario.fuente.resultado.Prueba.SujetoPseudonimo =
			referenciaPrueba(
				t, "hmac-sha256:seudonimo_oficial_v1:"+strings.Repeat("1", 64), 1,
			)
		_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
		if !errors.Is(err, ErrFuenteNoConfiable) || escenario.confirmador.llamadas != 0 {
			t.Fatalf("se acepto una fuente mixta: %v", err)
		}
	})
}

func TestOrdenRechazaConfusionDeRolesDelSelectorAntesDeInfraestructura(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	datos := escenario.datosOrden
	datos.Selector.Convocatoria = datos.Selector.SujetoPseudonimo
	if _, err := NuevaOrdenConfiable(datos); !errors.Is(err, ErrOrdenInvalida) {
		t.Fatalf("la orden acepto sujeto como convocatoria: %v", err)
	}
	if len(escenario.exigidor.llamadas) != 0 || escenario.fuente.llamadas != 0 ||
		escenario.confirmador.llamadas != 0 {
		t.Fatal("validar la orden alcanzo infraestructura")
	}
}

func TestFalloTecnicoOReciboAdulteradoNuncaSeConfirmanComoExito(t *testing.T) {
	t.Run("motor_distinto", func(t *testing.T) {
		escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
		datos := escenario.datosOrden
		datos.MotorEsperado.HuellaContratoSHA256 = strings.Repeat("e", 64)
		orden := debePrueba(NuevaOrdenConfiable(datos))
		_, err := escenario.servicio.Ejecutar(context.Background(), orden)
		if !errors.Is(err, ErrMotorNoCoincide) || escenario.confirmador.llamadas != 0 {
			t.Fatalf("motor distinto produjo confirmacion: %v", err)
		}
	})
	for _, intento := range []bool{false, true} {
		nombre := "intencion"
		if intento {
			nombre = "intento"
		}
		t.Run(nombre, func(t *testing.T) {
			escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
			escenario.confirmador.adulterar = !intento
			escenario.confirmador.adulterarIntento = intento
			_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
			if !errors.Is(err, ErrReciboNoConfiable) {
				t.Fatalf("se acepto recibo adulterado: %v", err)
			}
		})
	}
}

func TestCancelacionEnFronterasNoDejaConfirmacionParcial(t *testing.T) {
	t.Run("despues_fuente", func(t *testing.T) {
		escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.fuente.cancelar = cancelar
		_, err := escenario.servicio.Ejecutar(ctx, escenario.orden)
		if !errors.Is(err, context.Canceled) || len(escenario.exigidor.llamadas) != 1 ||
			escenario.confirmador.llamadas != 0 {
			t.Fatalf("cancelacion tras fuente avanzo: %v", err)
		}
	})
	t.Run("despues_autorizacion_escritura", func(t *testing.T) {
		escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
		ctx, cancelar := context.WithCancel(context.Background())
		escenario.exigidor.cancelar, escenario.exigidor.cancelarEn = cancelar, 2
		_, err := escenario.servicio.Ejecutar(ctx, escenario.orden)
		if !errors.Is(err, context.Canceled) || escenario.confirmador.llamadas != 0 {
			t.Fatalf("cancelacion antes de confirmar produjo efecto: %v", err)
		}
	})
}

func TestConfirmacionDuraderaPuedeReutilizarElMismoEfecto(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
	escenario.confirmador.desenlace = ConfirmacionReutilizada
	salida, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
	if err != nil {
		t.Fatal(err)
	}
	desenlace, err := salida.Desenlace()
	if err != nil || desenlace != ConfirmacionReutilizada {
		t.Fatalf("no se conservo la idempotencia durable: %s, %v", desenlace, err)
	}
}

func ordenRectificacionPrueba(
	t *testing.T, base DatosOrdenConfiable,
) OrdenCalculoExperienciaOficial {
	t.Helper()
	base.TipoEfecto = oficial.EfectoRectificacion
	base.Predecesor = &oficial.VinculoPredecesorV1{
		ReferenciaRecibo:   "recibo:calculo:predecesor:1",
		HuellaReciboSHA256: strings.Repeat("7", 64),
	}
	return debePrueba(NuevaOrdenConfiable(base))
}

var _ = time.Time{}
