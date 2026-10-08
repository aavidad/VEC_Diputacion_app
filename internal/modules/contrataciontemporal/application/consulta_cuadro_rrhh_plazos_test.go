package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type calculadoraPlazoFasePrueba struct {
	solicitudes []ports.SolicitudPlazoFaseRRHH
	plazo       ports.PlazoFaseRRHH
	aplicable   bool
	err         error
}

type calculadoraCapturaPrueba struct {
	actual   int
	capturas []string
}

func (c *calculadoraCapturaPrueba) CalcularPlazoFase(context.Context, ports.SolicitudPlazoFaseRRHH) (ports.PlazoFaseRRHH, bool, error) {
	c.actual++
	return ports.PlazoFaseRRHH{}, false, nil
}

func (c *calculadoraCapturaPrueba) CalcularPlazoConCaptura(_ context.Context, _ ports.SolicitudPlazoFaseRRHH, captura ports.CapturaPlazoFaseRRHH) (ports.PlazoFaseRRHH, bool, error) {
	c.capturas = append(c.capturas, captura.BaseHuella)
	p := plazoFaseValidoPrueba()
	p.ReglaRef = captura.BaseHuella
	return p, true, nil
}

func TestPlazoFaseUsaCapturaYNoCabezaActual(t *testing.T) {
	t.Parallel()
	desde := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	calculadora := &calculadoraCapturaPrueba{}
	clave := clavePlazoFaseCuadro{fase: "fiscalizacion", desde: desde}
	legado := ports.CapturaPlazoFaseRRHH{Estado: "legado_sin_instantanea", Fase: clave.fase, FaseDesde: desde}
	if p := calcularPlazoFase(t.Context(), calculadora, clave, desde.Add(time.Hour), &legado); p == nil || p.Estado != ports.PlazoFaseNoCalculado || calculadora.actual != 0 {
		t.Fatalf("legado sin baseline: %+v", p)
	}
	for _, huella := range []string{"base-anterior", "base-posterior"} {
		captura := ports.CapturaPlazoFaseRRHH{Estado: "legado_base_transicion", Fase: clave.fase, FaseDesde: desde, BaseHuella: huella}
		p := calcularPlazoFase(t.Context(), calculadora, clave, desde.Add(time.Hour), &captura)
		if p == nil || p.ReglaRef != huella {
			t.Fatalf("captura %s: %+v", huella, p)
		}
	}
	if calculadora.actual != 0 || len(calculadora.capturas) != 2 {
		t.Fatalf("se consultó cabeza actual: %+v", calculadora)
	}
}

func (c *calculadoraPlazoFasePrueba) CalcularPlazoFase(
	_ context.Context,
	solicitud ports.SolicitudPlazoFaseRRHH,
) (ports.PlazoFaseRRHH, bool, error) {
	c.solicitudes = append(c.solicitudes, solicitud)
	return c.plazo, c.aplicable, c.err
}

func plazoFaseValidoPrueba() ports.PlazoFaseRRHH {
	return ports.PlazoFaseRRHH{
		UltimoDia: "2026-08-07", VenceAntesDe: time.Date(2026, 8, 7, 22, 0, 0, 0, time.UTC),
		Estado: ports.PlazoFaseEnPlazo, ReglaRef: "vec.contratacion_temporal.reglas:1:c03.plazo_fiscalizacion",
		ReglaEjemplo: true,
	}
}

func TestConsultaCuadroRRHHCalculaPlazoDesdeEntradaEnFase(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	desde := entorno.sesion.pagina.Expedientes[0].CreadoEn
	entorno.sesion.pagina.FasesDesde = []time.Time{desde}
	servicio, err := NuevoServicioConsultaCuadroRRHH(entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj)
	if err != nil {
		t.Fatal(err)
	}
	calculadora := &calculadoraPlazoFasePrueba{plazo: plazoFaseValidoPrueba(), aplicable: true}
	servicio.ConfigurarPlazosFase(calculadora)
	pagina, err := servicio.Consultar(context.Background(), entorno.cuadro)
	if err != nil {
		t.Fatal(err)
	}
	if len(pagina.Plazos) != 1 || pagina.Plazos[0] == nil || *pagina.Plazos[0] != calculadora.plazo {
		t.Fatalf("plazo no proyectado: %+v", pagina.Plazos)
	}
	if len(calculadora.solicitudes) != 1 || !calculadora.solicitudes[0].Desde.Equal(desde) ||
		calculadora.solicitudes[0].Fase != entorno.sesion.pagina.Expedientes[0].FaseClave ||
		!calculadora.solicitudes[0].Ahora.Equal(entorno.ahora) {
		t.Fatalf("solicitud de plazo inesperada: %+v", calculadora.solicitudes)
	}
}

func TestConsultaCuadroRRHHSinPlazoConservaElCuadro(t *testing.T) {
	t.Parallel()
	casos := map[string]struct {
		calculadora *calculadoraPlazoFasePrueba
		fasesDesde  bool
	}{
		"sin_calculadora":   {fasesDesde: true},
		"sin_fecha_de_fase": {calculadora: &calculadoraPlazoFasePrueba{plazo: plazoFaseValidoPrueba(), aplicable: true}},
		"fase_sin_regla":    {calculadora: &calculadoraPlazoFasePrueba{}, fasesDesde: true},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			if caso.fasesDesde {
				entorno.sesion.pagina.FasesDesde = []time.Time{entorno.sesion.pagina.Expedientes[0].CreadoEn}
			}
			servicio, err := NuevoServicioConsultaCuadroRRHH(entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj)
			if err != nil {
				t.Fatal(err)
			}
			if caso.calculadora != nil {
				servicio.ConfigurarPlazosFase(caso.calculadora)
			}
			pagina, err := servicio.Consultar(context.Background(), entorno.cuadro)
			if err != nil || len(pagina.Expedientes) != 1 || pagina.Plazos != nil {
				t.Fatalf("el cuadro debe salir igual y sin plazos: %+v %v", pagina.Plazos, err)
			}
		})
	}
}

func TestConsultaCuadroRRHHPlazoNoCalculadoSeMuestraComoTal(t *testing.T) {
	t.Parallel()
	for nombre, calculadora := range map[string]*calculadoraPlazoFasePrueba{
		"calculo_no_posible":  {err: errors.New("sin calendario")},
		"resultado_no_valido": {aplicable: true},
	} {
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			entorno.sesion.pagina.FasesDesde = []time.Time{entorno.sesion.pagina.Expedientes[0].CreadoEn}
			servicio, err := NuevoServicioConsultaCuadroRRHH(entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj)
			if err != nil {
				t.Fatal(err)
			}
			servicio.ConfigurarPlazosFase(calculadora)
			pagina, err := servicio.Consultar(context.Background(), entorno.cuadro)
			if err != nil || len(pagina.Plazos) != 1 || pagina.Plazos[0] == nil ||
				*pagina.Plazos[0] != (ports.PlazoFaseRRHH{Estado: ports.PlazoFaseNoCalculado}) {
				t.Fatalf("el fallo debe verse como «no calculado»: %+v %v", pagina.Plazos, err)
			}
		})
	}
}

func TestConsultaCuadroRRHHRechazaFaseDesdeIncoherente(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	resumen := entorno.sesion.pagina.Expedientes[0]
	for _, fases := range [][]time.Time{
		{resumen.CreadoEn.Add(-time.Microsecond)},
		{resumen.ActualizadoEn.Add(time.Microsecond)},
		{resumen.CreadoEn, resumen.CreadoEn},
	} {
		entorno.sesion.pagina.FasesDesde = fases
		servicio, err := NuevoServicioConsultaCuadroRRHH(entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := servicio.Consultar(context.Background(), entorno.cuadro); !errors.Is(err, ErrResultadoConsultaRRHHNoConfiable) {
			t.Fatalf("fase de entrada incoherente aceptada %v: %v", fases, err)
		}
	}
}

func TestConsultaCuadroRRHHPideElPlazoUrgenteDeLosExpedientesUrgentes(t *testing.T) {
	t.Parallel()
	for _, urgentes := range [][]bool{nil, {false}, {true}} {
		entorno := nuevoEntornoConsultaRRHH(t)
		entorno.sesion.pagina.FasesDesde = []time.Time{entorno.sesion.pagina.Expedientes[0].CreadoEn}
		entorno.sesion.pagina.Urgentes = urgentes
		servicio, err := NuevoServicioConsultaCuadroRRHH(entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj)
		if err != nil {
			t.Fatal(err)
		}
		calculadora := &calculadoraPlazoFasePrueba{plazo: plazoFaseValidoPrueba(), aplicable: true}
		servicio.ConfigurarPlazosFase(calculadora)
		pagina, err := servicio.Consultar(context.Background(), entorno.cuadro)
		if err != nil {
			t.Fatal(err)
		}
		urgente := len(urgentes) == 1 && urgentes[0]
		if len(calculadora.solicitudes) != 1 || calculadora.solicitudes[0].Urgente != urgente ||
			len(pagina.Urgentes) != len(urgentes) {
			t.Fatalf("urgentes %v: solicitudes %+v, página %v", urgentes, calculadora.solicitudes, pagina.Urgentes)
		}
	}
}

// preparadorPlazoFasePrueba lee sus reglas una vez por consulta y entrega la
// calculadora preparada, o falla.
type preparadorPlazoFasePrueba struct {
	calculadoraPlazoFasePrueba
	preparaciones int
	preparada     *calculadoraPlazoFasePrueba
	errPreparar   error
}

func (p *preparadorPlazoFasePrueba) PrepararPlazosFase(context.Context) (ports.CalculadoraPlazoFaseRRHH, error) {
	p.preparaciones++
	if p.errPreparar != nil {
		return nil, p.errPreparar
	}
	return p.preparada, nil
}

// Con una calculadora que admite prepararse, la página calcula sus plazos con
// una sola preparación; si la preparación falla, calcula fila a fila como antes.
func TestConsultaCuadroRRHHPreparaLosPlazosUnaVezPorConsulta(t *testing.T) {
	t.Parallel()
	for _, caso := range []struct {
		nombre      string
		errPreparar error
	}{{"preparada", nil}, {"falla_y_calcula_fila_a_fila", errors.New("catálogo no disponible")}} {
		t.Run(caso.nombre, func(t *testing.T) {
			entorno := nuevoEntornoConsultaRRHH(t)
			entorno.sesion.pagina.FasesDesde = []time.Time{entorno.sesion.pagina.Expedientes[0].CreadoEn}
			servicio, err := NuevoServicioConsultaCuadroRRHH(entorno.autoridad, entorno.emisor, entorno.sesion, entorno.reloj)
			if err != nil {
				t.Fatal(err)
			}
			preparador := &preparadorPlazoFasePrueba{
				calculadoraPlazoFasePrueba: calculadoraPlazoFasePrueba{plazo: plazoFaseValidoPrueba(), aplicable: true},
				preparada:                  &calculadoraPlazoFasePrueba{plazo: plazoFaseValidoPrueba(), aplicable: true},
				errPreparar:                caso.errPreparar,
			}
			servicio.ConfigurarPlazosFase(preparador)
			pagina, err := servicio.Consultar(context.Background(), entorno.cuadro)
			if err != nil {
				t.Fatal(err)
			}
			if len(pagina.Plazos) != 1 || pagina.Plazos[0] == nil || *pagina.Plazos[0] != plazoFaseValidoPrueba() {
				t.Fatalf("plazo no proyectado: %+v", pagina.Plazos)
			}
			usadas, sinUsar := len(preparador.preparada.solicitudes), len(preparador.solicitudes)
			if caso.errPreparar != nil {
				usadas, sinUsar = sinUsar, usadas
			}
			if preparador.preparaciones != 1 || usadas != 1 || sinUsar != 0 {
				t.Fatalf("preparaciones %d, preparada %d, fila a fila %d", preparador.preparaciones,
					len(preparador.preparada.solicitudes), len(preparador.solicitudes))
			}
		})
	}
}
