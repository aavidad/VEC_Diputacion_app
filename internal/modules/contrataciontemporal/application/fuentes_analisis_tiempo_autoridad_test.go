package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type relojAvanzableFuentesAplicacionPrueba struct {
	mu       sync.Mutex
	instante time.Time
}

func (r *relojAvanzableFuentesAplicacionPrueba) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.instante
}

func (r *relojAvanzableFuentesAplicacionPrueba) fijar(instante time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.instante = instante
}

type fuenteRCAvancePresentacionAplicacionPrueba struct {
	base      fuenteRCAplicacionPrueba
	reloj     *relojAvanzableFuentesAplicacionPrueba
	posterior time.Time
	consultas int
}

func (f *fuenteRCAvancePresentacionAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	f.reloj.fijar(f.posterior)
	return f.base.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (f *fuenteRCAvancePresentacionAplicacionPrueba) ValidarRC(
	ctx context.Context,
	solicitud ports.SolicitudValidarRC,
) (ports.ResultadoValidacionRC, error) {
	f.consultas++
	return f.base.ValidarRC(ctx, solicitud)
}

type calculadorCosteAvancePresentacionAplicacionPrueba struct {
	base      calculadorCosteAplicacionPrueba
	reloj     *relojAvanzableFuentesAplicacionPrueba
	posterior time.Time
	consultas int
}

func (c *calculadorCosteAvancePresentacionAplicacionPrueba) PresentarAutoridadFuenteAnalisis(
	ctx context.Context,
	desafio ports.DesafioAutoridadFuenteAnalisis,
) (ports.PresentacionAutoridadFuenteAnalisis, error) {
	c.reloj.fijar(c.posterior)
	return c.base.PresentarAutoridadFuenteAnalisis(ctx, desafio)
}

func (c *calculadorCosteAvancePresentacionAplicacionPrueba) CalcularCoste(
	ctx context.Context,
	solicitud ports.SolicitudCalcularCoste,
) (ports.ResultadoCalculoCoste, error) {
	c.consultas++
	return c.base.CalcularCoste(ctx, solicitud)
}

type consumidorNoInvocableFuentesAplicacionPrueba struct {
	consumos int
}

func (c *consumidorNoInvocableFuentesAplicacionPrueba) ConsumirRespuestaFuenteAnalisis(
	context.Context,
	ports.OrdenConsumoRespuestaFuenteAnalisis,
) (ports.ReciboConsumoRespuestaFuenteAnalisis, error) {
	c.consumos++
	return ports.ReciboConsumoRespuestaFuenteAnalisis{},
		errors.New("consumo-sintetico-inesperado")
}

func TestFuentesAnalisisCompruebanHorizonteTrasCadaPresentacion(
	t *testing.T,
) {
	const (
		avancePresentacion = 2 * time.Second
		vigenciaCredencial = 6 * time.Second
	)
	t.Run("validacion_rc", func(t *testing.T) {
		capacidad, solicitudes, instante :=
			entornoFuentesAnalisisTemporalAplicacionPrueba(t)
		reloj := &relojAvanzableFuentesAplicacionPrueba{
			instante: instante,
		}
		base := capacidad.fuenteRC.(fuenteRCAplicacionPrueba)
		base.autoridad.datos.ValidaHasta =
			instante.Add(vigenciaCredencial)
		fuente := &fuenteRCAvancePresentacionAplicacionPrueba{
			base:      base,
			reloj:     reloj,
			posterior: instante.Add(avancePresentacion),
		}
		consumidor := &consumidorNoInvocableFuentesAplicacionPrueba{}

		_, err := ValidarRCConFuente(
			context.Background(),
			fuente,
			capacidad.verificador,
			capacidad.publicador,
			consumidor,
			capacidad.confianza,
			reloj,
			solicitudes.ValidacionRC,
		)

		if !errors.Is(err, ports.ErrArtefactoAnalisisNoConfiable) {
			t.Fatalf("credencial sin horizonte posterior aceptada: %v", err)
		}
		if fuente.consultas != 0 || consumidor.consumos != 0 {
			t.Fatalf(
				"hubo efectos tras rechazar autoridad: consultas=%d consumos=%d",
				fuente.consultas,
				consumidor.consumos,
			)
		}
		if obtenida := reloj.Ahora(); !obtenida.Equal(
			instante.Add(avancePresentacion),
		) {
			t.Fatalf("no se conservó el instante posterior: %s", obtenida)
		}
	})

	t.Run("calculo_coste", func(t *testing.T) {
		capacidad, solicitudes, instante :=
			entornoFuentesAnalisisTemporalAplicacionPrueba(t)
		reloj := &relojAvanzableFuentesAplicacionPrueba{
			instante: instante,
		}
		base := capacidad.calculador.(calculadorCosteAplicacionPrueba)
		base.autoridad.datos.ValidaHasta =
			instante.Add(vigenciaCredencial)
		calculador := &calculadorCosteAvancePresentacionAplicacionPrueba{
			base:      base,
			reloj:     reloj,
			posterior: instante.Add(avancePresentacion),
		}
		consumidor := &consumidorNoInvocableFuentesAplicacionPrueba{}

		_, err := CalcularCosteConFuente(
			context.Background(),
			calculador,
			capacidad.verificador,
			consumidor,
			capacidad.confianza,
			reloj,
			*solicitudes.CalculoCoste,
		)

		if !errors.Is(err, ports.ErrArtefactoAnalisisNoConfiable) {
			t.Fatalf("credencial sin horizonte posterior aceptada: %v", err)
		}
		if calculador.consultas != 0 || consumidor.consumos != 0 {
			t.Fatalf(
				"hubo efectos tras rechazar autoridad: consultas=%d consumos=%d",
				calculador.consultas,
				consumidor.consumos,
			)
		}
		if obtenida := reloj.Ahora(); !obtenida.Equal(
			instante.Add(avancePresentacion),
		) {
			t.Fatalf("no se conservó el instante posterior: %s", obtenida)
		}
	})
}

func entornoFuentesAnalisisTemporalAplicacionPrueba(
	t *testing.T,
) (
	*CapacidadPrepararArtefactoAnalisisO3,
	ports.SolicitudesFuentesAnalisisO3,
	time.Time,
) {
	t.Helper()
	escenario := nuevoEscenarioOperacionAnalisisSaneado(
		t,
		ports.OperacionRegistrarAnalisis,
		"-horizonte-posterior-sintetico",
	)
	preparador := nuevoPreparadorArtefactoAnalisisO3AplicacionPrueba(
		t,
		escenario.instante,
	)
	capacidad, correcta :=
		preparador.(*CapacidadPrepararArtefactoAnalisisO3)
	if !correcta {
		t.Fatal("la composición sintética no devolvió la capacidad real")
	}
	solicitud := ports.SolicitudPrepararArtefactoAnalisis{
		ArtefactoRef:      escenario.registrar.ArtefactoRef,
		OrganizacionRef:   escenario.registrar.OrganizacionRef,
		ExpedienteRef:     escenario.registrar.ExpedienteRef,
		VersionExpediente: escenario.registrar.VersionEsperada,
		DatosFuncionales:  escenario.registrar.DatosFuncionales,
		SolicitadaEn:      escenario.instante,
	}
	solicitudes, err := capacidad.solicitudes.
		PrepararSolicitudesFuentesAnalisisO3(
			context.Background(),
			solicitud,
		)
	if err != nil {
		t.Fatal(err)
	}
	if solicitudes.CalculoCoste == nil {
		t.Fatal("el escenario sintético no produjo cálculo de coste")
	}
	return capacidad, solicitudes, escenario.instante
}
