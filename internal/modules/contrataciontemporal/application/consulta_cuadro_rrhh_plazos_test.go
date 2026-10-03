package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

func instantaneaCuadroPrueba(fase string, desde time.Time, huella string) *reglas.InstantaneaPersistidaRegla {
	return &reglas.InstantaneaPersistidaRegla{
		CatalogoBaseID: reglas.CatalogoContratacionTemporal, CatalogoBaseHuella: huella,
		CatalogoBaseCanonico: []byte("base"), CanonicoAjustes: []byte("{}"),
		CatalogoAjustesID: reglas.CatalogoAjustesDe(reglas.CatalogoContratacionTemporal),
		Fase:              fase, FaseDesde: desde, PreparadaEn: desde,
	}
}

type calculadoraPlazoFasePrueba struct {
	solicitudes []ports.SolicitudPlazoFaseRRHH
	plazo       ports.PlazoFaseRRHH
	aplicable   bool
	err         error
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
	entorno.sesion.pagina.InstantaneasPlazo = []*reglas.InstantaneaPersistidaRegla{
		instantaneaCuadroPrueba(string(entorno.sesion.pagina.Expedientes[0].FaseClave), desde, "huella-a"),
	}
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
		!calculadora.solicitudes[0].Ahora.Equal(entorno.ahora) ||
		calculadora.solicitudes[0].Instantanea == nil ||
		calculadora.solicitudes[0].Instantanea.CatalogoBaseHuella != "huella-a" {
		t.Fatalf("solicitud de plazo inesperada: %+v", calculadora.solicitudes)
	}
	original := entorno.sesion.pagina.InstantaneasPlazo[0]
	original.CatalogoBaseCanonico[0] = 'X'
	original.CanonicoAjustes[0] = 'X'
	if pagina.InstantaneasPlazo[0] == original ||
		string(pagina.InstantaneasPlazo[0].CatalogoBaseCanonico) != "base" ||
		string(pagina.InstantaneasPlazo[0].CanonicoAjustes) != "{}" ||
		string(calculadora.solicitudes[0].Instantanea.CatalogoBaseCanonico) != "base" {
		t.Fatal("la consulta expuso alias mutables de la instantánea")
	}
}

func TestConsultaCuadroRRHHSinPlazoConservaElCuadro(t *testing.T) {
	t.Parallel()
	casos := map[string]struct {
		calculadora *calculadoraPlazoFasePrueba
		fasesDesde  bool
		instantanea bool
	}{
		"sin_calculadora":   {fasesDesde: true},
		"sin_fecha_de_fase": {calculadora: &calculadoraPlazoFasePrueba{plazo: plazoFaseValidoPrueba(), aplicable: true}},
		"fase_sin_regla":    {calculadora: &calculadoraPlazoFasePrueba{}, fasesDesde: true, instantanea: true},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			t.Parallel()
			entorno := nuevoEntornoConsultaRRHH(t)
			if caso.fasesDesde {
				entorno.sesion.pagina.FasesDesde = []time.Time{entorno.sesion.pagina.Expedientes[0].CreadoEn}
			}
			if caso.instantanea {
				desde := entorno.sesion.pagina.FasesDesde[0]
				entorno.sesion.pagina.InstantaneasPlazo = []*reglas.InstantaneaPersistidaRegla{
					instantaneaCuadroPrueba(string(entorno.sesion.pagina.Expedientes[0].FaseClave), desde, "huella-a"),
				}
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

func TestConsultaCuadroRRHHCacheDistingueInstantaneasAunqueDeclarenLaMismaHuella(t *testing.T) {
	t.Parallel()
	entorno := nuevoEntornoConsultaRRHH(t)
	resumen := entorno.sesion.pagina.Expedientes[0]
	desde := resumen.CreadoEn
	primera := instantaneaCuadroPrueba(string(resumen.FaseClave), desde, "misma-huella")
	segunda := clonarInstantaneaPlazo(primera)
	segunda.CatalogoBaseCanonico[0] = 'X'
	calculadora := &calculadoraPlazoFasePrueba{plazo: plazoFaseValidoPrueba(), aplicable: true}
	servicio := &ServicioConsultaCuadroRRHH{reloj: entorno.reloj, plazos: calculadora}
	pagina := ports.PaginaCuadroRRHH{
		Expedientes:       []ports.ResumenExpedienteRRHH{resumen, resumen, resumen},
		FasesDesde:        []time.Time{desde, desde, desde},
		InstantaneasPlazo: []*reglas.InstantaneaPersistidaRegla{primera, segunda, segunda},
	}
	plazos := servicio.calcularPlazosFase(t.Context(), pagina)
	if len(plazos) != 3 || len(calculadora.solicitudes) != 2 ||
		string(calculadora.solicitudes[0].Instantanea.CatalogoBaseCanonico) != "base" ||
		string(calculadora.solicitudes[1].Instantanea.CatalogoBaseCanonico) != "Xase" {
		t.Fatalf("la caché mezcló instantáneas: %d plazos, %d llamadas", len(plazos), len(calculadora.solicitudes))
	}
}
