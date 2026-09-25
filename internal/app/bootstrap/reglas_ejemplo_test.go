package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const (
	rutaReglasBolsaEjemploPrueba   = "../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json"
	rutaReglasCTEjemploPrueba      = "../../../data/demo/reglas/ct_reglas.ejemplo.demo.json"
	rutaMotivosCTEjemploPrueba     = "../../../data/demo/reglas/ct_motivos_rectificacion.demo.json"
	rutaInexistenteReglasEjemploPr = "../../../data/demo/reglas/no-existe.demo.json"
)

type relojReglasEjemploPrueba struct{ ahora time.Time }

func (r relojReglasEjemploPrueba) Ahora() time.Time { return r.ahora }

var relojPresentacionReglasEjemplo = relojReglasEjemploPrueba{ahora: time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)}

func configuracionDesarrolloReglasEjemplo(bolsa, ct string) config.Config {
	return config.Config{
		ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment,
		DevelopmentGuard: config.DevelopmentGuardAcknowledgement,
		ReglasEjemplo:    config.ConfiguracionReglasEjemplo{BolsaSourcePath: bolsa, CTSourcePath: ct},
	}
}

func TestReglasEjemploSeComponenSoloConCatalogoDeclarado(t *testing.T) {
	sinCatalogo, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo("", ""), nil, relojPresentacionReglasEjemplo)
	if err != nil || sinCatalogo.bolsa.Disponible() || sinCatalogo.contratacionTemporal.Disponible() {
		t.Fatalf("sin catálogo todo sigue como hoy: %+v %v", sinCatalogo, err)
	}
	compuestas, err := nuevasReglasEjemploDesarrollo(
		configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, rutaReglasCTEjemploPrueba),
		nil, relojPresentacionReglasEjemplo,
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta, err := compuestas.bolsa.Regla(t.Context(), reglas.BolsaPlazoRespuesta)
	if err != nil || respuesta.Cantidad != 1 || !respuesta.PaqueteEjemplo {
		t.Fatalf("regla de Bolsa inesperada: %+v %v", respuesta, err)
	}
	jornada, err := compuestas.contratacionTemporal.Regla(t.Context(), reglas.CTJornadaCompleta)
	if err != nil || jornada.Cantidad != 2250 || jornada.Unidad != reglas.UnidadMinutosSemanales {
		t.Fatalf("regla de CT inesperada: %+v %v", jornada, err)
	}
	// Sin Calendarios el cómputo civil sigue disponible; el administrativo no.
	if _, vencimiento, err := compuestas.bolsa.Vencimiento(t.Context(), reglas.BolsaReposicionGeneral,
		time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC), ""); err != nil || vencimiento.UltimoDia != "2026-08-10" {
		t.Fatalf("reposición civil inesperada: %+v %v", vencimiento, err)
	}
	if _, _, err := compuestas.bolsa.Vencimiento(t.Context(), reglas.BolsaPlazoRespuesta,
		time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC), ""); !errors.Is(err, reglas.ErrCalculoNoDisponible) {
		t.Fatalf("sin Calendarios no puede suponerse un plazo hábil: %v", err)
	}
	soloBolsa, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, relojPresentacionReglasEjemplo)
	if err != nil || !soloBolsa.bolsa.Disponible() || soloBolsa.contratacionTemporal.Disponible() {
		t.Fatalf("cada catálogo se compone por separado: %+v %v", soloBolsa, err)
	}
}

func TestReglasEjemploInvalidasImpidenArrancar(t *testing.T) {
	casos := []config.Config{
		configuracionDesarrolloReglasEjemplo(rutaReglasCTEjemploPrueba, ""),
		configuracionDesarrolloReglasEjemplo("", rutaReglasBolsaEjemploPrueba),
		configuracionDesarrolloReglasEjemplo(rutaInexistenteReglasEjemploPr, ""),
		configuracionDesarrolloReglasEjemplo("", rutaMotivosCTEjemploPrueba),
	}
	for indice, cfg := range casos {
		if _, err := nuevasReglasEjemploDesarrollo(cfg, nil, relojPresentacionReglasEjemplo); !errors.Is(err, errReglasEjemploNoValidas) {
			t.Errorf("caso %d aceptado: %v", indice, err)
		}
	}
	antesDePublicar := relojReglasEjemploPrueba{ahora: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)}
	if _, err := nuevasReglasEjemploDesarrollo(configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, antesDePublicar); !errors.Is(err, errReglasEjemploNoValidas) {
		t.Fatalf("un catálogo aún no vigente no puede componerse: %v", err)
	}
}

func TestReglasEjemploFueraDeDesarrolloImpidenArrancar(t *testing.T) {
	produccion := config.Config{ReglasEjemplo: config.ConfiguracionReglasEjemplo{BolsaSourcePath: rutaReglasBolsaEjemploPrueba}}
	if _, err := NewHTTPServerWithConfig(produccion); !errors.Is(err, config.ErrConfiguracionReglasEjemploFueraDesarrollo) {
		t.Fatalf("producción con reglas de ejemplo: %v", err)
	}
	motivos := config.Config{CTAnalisisMotivosSourcePath: rutaMotivosCTEjemploPrueba}
	if _, err := NewHTTPServerWithConfig(motivos); !errors.Is(err, config.ErrConfiguracionReglasEjemploFueraDesarrollo) {
		t.Fatalf("producción con motivos de ejemplo: %v", err)
	}
	if _, err := NewHTTPServerPublicoWithConfig(produccion); !errors.Is(err, config.ErrConfiguracionReglasEjemploFueraDesarrollo) {
		t.Fatalf("binario público con reglas de ejemplo: %v", err)
	}
	if _, err := nuevasReglasEjemploDesarrollo(produccion, nil, relojPresentacionReglasEjemplo); !errors.Is(err, config.ErrConfiguracionReglasEjemploFueraDesarrollo) {
		t.Fatalf("la composición también debe rechazarlas: %v", err)
	}
}

func TestMotivosRectificacionEjemploLosAceptaLaFuenteExistente(t *testing.T) {
	fuente, err := nuevaFuenteMotivosRectificacionAnalisisDesarrolloConfigurada(
		config.Config{CTAnalisisMotivosSourcePath: rutaMotivosCTEjemploPrueba},
		relojFijoAltaContratacionTemporalDesarrollo{ahora: relojPresentacionReglasEjemplo.ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	opciones := fuente.opciones(t.Context())
	if len(opciones) != 6 || opciones[0].Clave != "error_material" || opciones[5].Clave != "observacion_intervencion" {
		t.Fatalf("motivos de ejemplo inesperados: %+v", opciones)
	}
}

type consultaCalendariosReglasPrueba struct {
	recibida  calendariosports.SolicitudCalculoPlazo
	resultado calendariosports.ResultadoCalculoPlazo
	err       error
}

func (c *consultaCalendariosReglasPrueba) Centros(context.Context, int, time.Time) ([]calendariosports.CentroConCalendario, error) {
	return nil, errors.New("no usado")
}

func (c *consultaCalendariosReglasPrueba) CalendarioCentro(context.Context, calendariosports.SolicitudCalendarioCentro) (calendariosports.CalendarioCentro, error) {
	return calendariosports.CalendarioCentro{}, errors.New("no usado")
}

func (c *consultaCalendariosReglasPrueba) CalcularPlazo(_ context.Context, s calendariosports.SolicitudCalculoPlazo) (calendariosports.ResultadoCalculoPlazo, error) {
	c.recibida = s
	return c.resultado, c.err
}

func TestCalculadoraPlazosCalendariosTraduceAmbosComputos(t *testing.T) {
	vencimiento, err := calendariosdomain.ParsearFechaCivil("2026-09-29")
	if err != nil {
		t.Fatal(err)
	}
	finDia, err := vencimiento.FinEnMadrid()
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaCalendariosReglasPrueba{resultado: calendariosports.ResultadoCalculoPlazo{
		ResultadoPlazo:      calendariosdomain.ResultadoPlazo{Vencimiento: vencimiento, VenceAntesDe: finDia},
		VersionesUtilizadas: []calendariosdomain.VersionCalendario{{ID: "calendario:es:2026:1"}},
	}}
	calculadora := calculadoraPlazosCalendarios{consulta: consulta}
	contacto := time.Date(2026, 9, 28, 9, 30, 0, 0, time.UTC)
	resultado, err := calculadora.CalcularVencimiento(t.Context(), reglas.SolicitudVencimiento{
		Inicio: contacto, Unidad: reglas.UnidadDiasHabiles, Cantidad: 1,
		Computo: reglas.ComputoAdministrativo, MunicipioSede: reglas.MunicipioSedeDiputacion,
	})
	if err != nil || resultado.UltimoDia != "2026-09-29" || !resultado.VenceAntesDe.Equal(time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC)) ||
		len(resultado.Calendarios) != 1 || consulta.recibida.NotificadoEn != contacto ||
		consulta.recibida.Unidad != calendariosdomain.UnidadDiasHabiles || consulta.recibida.MunicipioSede != reglas.MunicipioSedeDiputacion {
		t.Fatalf("traducción administrativa inesperada: %+v %+v %v", resultado, consulta.recibida, err)
	}
	consulta.err = calendariosdomain.ErrCalendarioNoCubre
	if _, err := calculadora.CalcularVencimiento(t.Context(), reglas.SolicitudVencimiento{
		Inicio: contacto, Unidad: reglas.UnidadDiasHabiles, Cantidad: 1, Computo: reglas.ComputoAdministrativo,
	}); !errors.Is(err, calendariosdomain.ErrCalendarioNoCubre) {
		t.Fatalf("el error del calendario debe propagarse: %v", err)
	}
	civiles := []struct {
		inicio   time.Time
		unidad   reglas.Unidad
		cantidad int
		ultimo   string
	}{
		{time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC), reglas.UnidadMeses, 1, "2026-04-30"},
		// 23:30 UTC del 9 de marzo ya es 10 de marzo en hora peninsular.
		{time.Date(2026, 3, 9, 23, 30, 0, 0, time.UTC), reglas.UnidadMeses, 5, "2026-08-10"},
		{time.Date(2026, 1, 16, 10, 0, 0, 0, time.UTC), reglas.UnidadAnios, 5, "2031-01-16"},
		{time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), reglas.UnidadDiasNaturales, 15, "2026-10-13"},
	}
	for _, caso := range civiles {
		resultado, err := calculadoraPlazosCalendarios{}.CalcularVencimiento(t.Context(), reglas.SolicitudVencimiento{
			Inicio: caso.inicio, Unidad: caso.unidad, Cantidad: caso.cantidad, Computo: reglas.ComputoCivil,
		})
		if err != nil || resultado.UltimoDia != caso.ultimo || resultado.Prorrogado || len(resultado.Calendarios) != 0 {
			t.Errorf("cómputo civil %v %d %s: %+v %v", caso.inicio, caso.cantidad, caso.unidad, resultado, err)
		}
	}
	if _, err := (calculadoraPlazosCalendarios{}).CalcularVencimiento(t.Context(), reglas.SolicitudVencimiento{
		Inicio: contacto, Unidad: reglas.UnidadDiasHabiles, Cantidad: 1, Computo: reglas.ComputoCivil,
	}); !errors.Is(err, reglas.ErrReglaSinPlazo) {
		t.Fatalf("no hay días hábiles en el cómputo civil: %v", err)
	}
	var punteroNulo *consultaCalendariosReglasPrueba
	for nombre, sinCalendarios := range map[string]calculadoraPlazosCalendarios{
		"interfaz nula": {}, "puntero nulo": {consulta: punteroNulo},
	} {
		_, err := sinCalendarios.CalcularVencimiento(t.Context(), reglas.SolicitudVencimiento{
			Inicio: contacto, Unidad: reglas.UnidadDiasHabiles, Cantidad: 1, Computo: reglas.ComputoAdministrativo,
		})
		if !errors.Is(err, reglas.ErrCalculoNoDisponible) || !errors.Is(err, errReglasEjemploSinCalendarios) {
			t.Fatalf("%s: sin Calendarios no hay cómputo administrativo: %v", nombre, err)
		}
	}
}
