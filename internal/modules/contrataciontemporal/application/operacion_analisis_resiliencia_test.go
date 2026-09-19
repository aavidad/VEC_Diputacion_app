package application

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"gopkg.in/yaml.v3"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestOperacionAnalisisTienePresupuestoGlobalMaximoDeCincoSegundos(
	t *testing.T,
) {
	if ports.TiempoMaximoOperacionAnalisis != 5*time.Second {
		t.Fatalf(
			"presupuesto global inesperado: %s",
			ports.TiempoMaximoOperacionAnalisis,
		)
	}
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-presupuesto-global-sintetico",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	if _, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	); err != nil {
		t.Fatal(err)
	}
	if d.contextos.margen <= 0 ||
		d.contextos.margen > ports.TiempoMaximoOperacionAnalisis {
		t.Fatalf(
			"las dependencias no recibieron el presupuesto global: %s",
			d.contextos.margen,
		)
	}
}

func TestOperacionAnalisisRespetaCancelacionAntesYDespuesDelCommit(
	t *testing.T,
) {
	t.Run("antes", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-cancelacion-previa-sintetica",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		ctx, cancelar := context.WithCancel(context.Background())
		d.sellador.antes = cancelar
		_, err := servicio.Registrar(ctx, escenario.registrar)
		if !errors.Is(err, context.Canceled) ||
			d.preparaciones.llamadas != 0 ||
			d.transaccion.llamadas != 0 ||
			d.transaccion.consumosFuentes != 0 ||
			d.transaccion.consumosV3 != 0 {
			t.Fatalf("cancelación previa ambigua: %v", err)
		}
	})
	t.Run("despues", func(t *testing.T) {
		escenario := nuevoEscenarioOperacionAnalisisSaneado(
			t,
			ports.OperacionRegistrarAnalisis,
			"-cancelacion-posterior-sintetica",
		)
		servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
		ctx, cancelar := context.WithCancel(context.Background())
		d.transaccion.despues = cancelar
		recibo, err := servicio.Registrar(ctx, escenario.registrar)
		if err != nil || recibo.VersionResultante != 2 ||
			d.transaccion.llamadas != 1 ||
			d.transaccion.consumosFuentes != 1 ||
			d.transaccion.consumosV3 != 1 ||
			d.transaccion.commits != 1 {
			t.Fatalf("commit confirmado quedó ambiguo: %#v, %v", recibo, err)
		}
	})
}

func TestOperacionAnalisisRechazaVersionSinMargenYDependenciaTypedNil(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-limites-sinteticos",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	solicitud := escenario.registrar
	solicitud.VersionEsperada = ports.MaximoEnteroSeguroOperacionAnalisis
	_, err := servicio.Registrar(context.Background(), solicitud)
	if !errors.Is(err, ErrSolicitudOperacionAnalisisInvalida) ||
		d.contextos.llamadas != 0 {
		t.Fatalf("versión sin margen aceptada: %v", err)
	}
	var artefactos *preparadorArtefactoAnalisisDoble
	if _, err = NuevoServicioOperacionAnalisis(
		d.contextos,
		artefactos,
		d.sellador,
		d.preparaciones,
		d.politicas,
		d.correlaciones,
		d.autorizador,
		d.reloj,
		d.transaccion,
	); !errors.Is(err, ErrServicioOperacionAnalisisInvalido) {
		t.Fatalf("typed nil aceptado: %v", err)
	}
}

func TestOperacionAnalisisNoRectificaConCoberturaYaMaterializada(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRectificarAnalisis,
		"-cobertura-sintetica",
	)
	instanteCobertura := escenario.instante.Add(-5 * time.Minute)
	conCobertura, err := escenario.expediente.RegistrarViaCobertura(
		escenario.expediente.Version,
		domain.DecisionViaCobertura{
			ViaClave:         "via.cobertura_sintetica",
			ProcedimientoRef: "procedimiento:sintetico-001",
			BolsaRef:         "bolsa:sintetica-001",
			Comprobaciones: []domain.ComprobacionCobertura{{
				Clave:      "comprobacion.sintetica",
				Resultado:  domain.ComprobacionAfirmativa,
				FuenteRef:  "fuente:cobertura-sintetica-001",
				ReciboRef:  "recibo:cobertura-sintetica-001",
				EvaluadaEn: instanteCobertura,
			}},
			Motivacion: "Motivación enteramente sintética.",
		},
		domain.DatosActuacion{
			AccionClave:   "cobertura.sintetica_registrada",
			ActorRef:      "actor:cobertura-sintetico-001",
			UnidadRef:     "unidad:cobertura-sintetica-001",
			ReciboRef:     "recibo:cobertura-accion-sintetico-001",
			RealizadaEn:   instanteCobertura,
			FaseDestino:   escenario.expediente.FaseActual,
			EstadoDestino: escenario.expediente.EstadoActual,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	escenario.expediente = conCobertura
	escenario.rectificar.VersionEsperada = conCobertura.Version
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)

	_, err = servicio.Rectificar(context.Background(), escenario.rectificar)
	if !errors.Is(err, ErrSolicitudOperacionAnalisisInvalida) ||
		d.transaccion.llamadas != 0 {
		t.Fatalf("se rectificó una cobertura materializada: %v", err)
	}
}

func TestOrdenOperacionAnalisisRechazaCambioExtraYTodosLosCodecs(
	t *testing.T,
) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-orden-sintetica",
	)
	servicio, d := construirServicioOperacionAnalisisSaneado(t, escenario)
	if _, err := servicio.Registrar(
		context.Background(),
		escenario.registrar,
	); err != nil {
		t.Fatal(err)
	}
	evidencia, err := d.transaccion.orden.Datos()
	if err != nil {
		t.Fatal(err)
	}
	alterado := evidencia.ExpedienteSiguiente.Clonar()
	alterado.NumeroVisible = "2026/SINT-9999"
	_, err = ports.NuevaOrdenConfirmarOperacionAnalisis(
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
			SolicitudV3:          evidencia.SolicitudV3,
			DecisionV3:           evidencia.DecisionV3,
			ConfirmacionV3:       evidencia.ConfirmacionV3,
			InstanteEfecto:       evidencia.InstanteEfecto,
			ExpedienteSiguiente:  alterado,
		},
	)
	if !errors.Is(err, ports.ErrOrdenOperacionAnalisisInvalida) {
		t.Fatal("la fábrica aceptó un cambio adicional del expediente")
	}
	_, err = ports.NuevaOrdenConfirmarOperacionAnalisis(
		ports.DatosOrdenConfirmarOperacionAnalisis{
			SolicitudContexto:    evidencia.SolicitudContexto,
			ContextoAutorizacion: evidencia.ContextoAutorizacion,
			SolicitudArtefacto:   evidencia.SolicitudArtefacto,
			Artefacto:            evidencia.Artefacto,
			SolicitudPreparacion: evidencia.SolicitudPreparacion,
			Preparacion:          evidencia.Preparacion,
			SolicitudPolitica:    evidencia.SolicitudPolitica,
			Politica:             evidencia.Politica,
			SolicitudV3:          evidencia.SolicitudV3,
			DecisionV3:           evidencia.DecisionV3,
			ConfirmacionV3:       evidencia.ConfirmacionV3,
			InstanteEfecto:       evidencia.InstanteEfecto,
			ExpedienteSiguiente:  evidencia.ExpedienteSiguiente,
		},
	)
	if !errors.Is(err, ports.ErrOrdenOperacionAnalisisInvalida) {
		t.Fatal("la fábrica aceptó una orden sin consumo pendiente")
	}
	comprobarError := func(nombre string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s serializó una orden opaca", nombre)
		}
	}
	orden := d.transaccion.orden
	_, err = json.Marshal(orden)
	comprobarError("json", err)
	_, err = xml.Marshal(orden)
	comprobarError("xml", err)
	_, err = orden.MarshalText()
	comprobarError("texto", err)
	_, err = orden.MarshalBinary()
	comprobarError("binario", err)
	var destino bytes.Buffer
	comprobarError("gob", gob.NewEncoder(&destino).Encode(orden))
	_, err = cbor.Marshal(orden)
	comprobarError("cbor", err)
	_, err = yaml.Marshal(orden)
	comprobarError("yaml", err)
	if strings.Contains(fmt.Sprint(orden), escenario.registrar.ExpedienteRef) ||
		strings.Contains(fmt.Sprintf("%#v", orden), escenario.registrar.ArtefactoRef) {
		t.Fatal("la representación textual expone coordenadas internas")
	}
}
