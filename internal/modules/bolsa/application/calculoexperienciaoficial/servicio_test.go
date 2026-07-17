package calculoexperienciaoficial

import (
	"context"
	"errors"
	"testing"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestServicioEjecutaDosAutorizacionesYConfirmaResultadoExacto(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilExternoOrdinario, false)
	resultado, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
	if err != nil {
		t.Fatal(err)
	}
	motor, errResultado := resultado.Resultado()
	recibo, errRecibo := resultado.Recibo()
	desenlace, errDesenlace := resultado.Desenlace()
	if errResultado != nil || errRecibo != nil || errDesenlace != nil ||
		motor.Estado() != calculo.ResultadoExperienciaCompletado ||
		recibo.Validar() != nil || desenlace != ConfirmacionCreada {
		t.Fatal("resultado oficial o recibo no validos")
	}
	if escenario.fuente.llamadas != 1 || escenario.confirmador.llamadas != 1 ||
		len(escenario.exigidor.llamadas) != 2 {
		t.Fatalf("secuencia incompleta: fuente=%d pdp=%d confirmar=%d",
			escenario.fuente.llamadas, len(escenario.exigidor.llamadas),
			escenario.confirmador.llamadas)
	}
	lectura := escenario.exigidor.llamadas[0]
	escritura := escenario.exigidor.llamadas[1]
	if lectura.recurso.Tipo != tipoRecursoFuenteCalculo ||
		escritura.recurso.Tipo != tipoRecursoCalculoOficial ||
		lectura.recurso.Ambitos["sujeto_ref"] !=
			escenario.datosOrden.Selector.SujetoPseudonimo.Referencia() ||
		lectura.recurso.Ambitos["convocatoria_ref"] !=
			escenario.datosOrden.Selector.Convocatoria.Referencia() ||
		escritura.recurso.Ambitos["sujeto_ref"] !=
			escenario.datosOrden.Selector.SujetoPseudonimo.Referencia() ||
		escritura.recurso.Ambitos["convocatoria_ref"] !=
			escenario.datosOrden.Selector.Convocatoria.Referencia() ||
		!correlacionesDistintas(lectura.correlacion, escritura.correlacion) {
		t.Fatal("las autorizaciones no quedaron separadas o ligadas contra IDOR")
	}
}

func TestServicioConfirmaUnBloqueoExplicable(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, true)
	salida, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
	if err != nil {
		t.Fatal(err)
	}
	resultado, _ := salida.Resultado()
	if resultado.Estado() != calculo.ResultadoExperienciaBloqueado ||
		resultado.Fase() != calculo.FaseResultadoSeleccion ||
		escenario.confirmador.llamadas != 1 {
		t.Fatal("el bloqueo de negocio no fue confirmado de forma durable")
	}
}

func TestDenegacionesNuncaAlcanzanConfirmacion(t *testing.T) {
	for _, fase := range []int{1, 2} {
		t.Run(string(rune('0'+fase)), func(t *testing.T) {
			escenario := nuevoEscenarioServicioPrueba(t, perfilInternoAlto, false)
			escenario.exigidor.fallarEn = fase
			_, err := escenario.servicio.Ejecutar(context.Background(), escenario.orden)
			if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) ||
				escenario.confirmador.llamadas != 0 {
				t.Fatalf("denegacion %d no cerro el flujo: %v", fase, err)
			}
			if fase == 1 && escenario.fuente.llamadas != 0 {
				t.Fatal("una lectura denegada alcanzo la fuente")
			}
		})
	}
}

func TestCancelacionPreviaNoProduceEfectos(t *testing.T) {
	escenario := nuevoEscenarioServicioPrueba(t, perfilExternoOrdinario, false)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err := escenario.servicio.Ejecutar(ctx, escenario.orden)
	if !errors.Is(err, context.Canceled) || len(escenario.exigidor.llamadas) != 0 ||
		escenario.fuente.llamadas != 0 || escenario.confirmador.llamadas != 0 {
		t.Fatalf("la cancelacion produjo trabajo o efecto: %v", err)
	}
}
