package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

func calendariosPlazoFasePrueba(t *testing.T) *consultaCalendariosReglasPrueba {
	t.Helper()
	ultimo, err := calendariosdomain.ParsearFechaCivil("2026-09-29")
	if err != nil {
		t.Fatal(err)
	}
	finDia, err := ultimo.FinEnMadrid()
	if err != nil {
		t.Fatal(err)
	}
	return &consultaCalendariosReglasPrueba{resultado: calendariosports.ResultadoCalculoPlazo{
		ResultadoPlazo:      calendariosdomain.ResultadoPlazo{Vencimiento: ultimo, VenceAntesDe: finDia},
		VersionesUtilizadas: []calendariosdomain.VersionCalendario{{ID: "calendario:es:2026:1"}},
	}}
}

func calculadoraPlazoFaseCTPrueba(t *testing.T, ruta string, calendarios *consultaCalendariosReglasPrueba) ports.CalculadoraPlazoFaseRRHH {
	t.Helper()
	resolutor, err := nuevoResolutorReglasEjemplo(ruta, reglas.CatalogoContratacionTemporal, reglas.ModuloContratacionTemporal,
		calculadoraPlazosCalendarios{consulta: calendarios}, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	return nuevaCalculadoraPlazoFaseCT(resolutor)
}

func TestPlazoFaseCTSaleDelAtributoFasesDelCatalogo(t *testing.T) {
	calendarios := calendariosPlazoFasePrueba(t)
	calculadora := calculadoraPlazoFaseCTPrueba(t, rutaReglasCTEjemploPrueba, calendarios)
	desde := time.Date(2026, 9, 15, 9, 30, 0, 0, time.UTC)
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	casos := map[domain.ClaveFase]string{
		"asignacion_unidad":  reglas.CTPlazoInformes,
		"informe_juridico":   reglas.CTPlazoInformes,
		"fiscalizacion":      reglas.CTPlazoFiscalizacion,
		"subsanacion_unidad": reglas.CTPlazoSubsanacion,
	}
	for fase, clave := range casos {
		plazo, aplicable, err := calculadora.CalcularPlazoFase(t.Context(), ports.SolicitudPlazoFaseRRHH{Fase: fase, Desde: desde, Ahora: ahora})
		if err != nil || !aplicable || !plazo.Valido() || plazo.UltimoDia != "2026-09-29" ||
			plazo.Estado != ports.PlazoFaseEnPlazo || !plazo.ReglaEjemplo ||
			!strings.HasSuffix(plazo.ReglaRef, ":"+clave) {
			t.Fatalf("fase %s: %+v %v %v", fase, plazo, aplicable, err)
		}
		// Diez días hábiles, sin variante urgente, desde la entrada en fase.
		if calendarios.recibida.Cantidad != 10 || !calendarios.recibida.NotificadoEn.Equal(desde) {
			t.Fatalf("fase %s pidió %+v", fase, calendarios.recibida)
		}
	}
	for _, fase := range []domain.ClaveFase{"solicitud", "nombramiento", "llamamiento"} {
		if plazo, aplicable, err := calculadora.CalcularPlazoFase(t.Context(), ports.SolicitudPlazoFaseRRHH{Fase: fase, Desde: desde, Ahora: ahora}); err != nil || aplicable {
			t.Fatalf("la fase %s no tiene regla en el catálogo: %+v %v %v", fase, plazo, aplicable, err)
		}
	}
}

func TestPlazoFaseCTDistingueEnPlazoVenceHoyYVencido(t *testing.T) {
	calculadora := calculadoraPlazoFaseCTPrueba(t, rutaReglasCTEjemploPrueba, calendariosPlazoFasePrueba(t))
	desde := time.Date(2026, 9, 15, 9, 30, 0, 0, time.UTC)
	casos := []struct {
		ahora  time.Time
		estado ports.EstadoPlazoFaseRRHH
	}{
		{time.Date(2026, 9, 28, 21, 59, 0, 0, time.UTC), ports.PlazoFaseEnPlazo},
		// 22:00 UTC es ya el día 29 en hora peninsular (CEST).
		{time.Date(2026, 9, 28, 22, 0, 0, 0, time.UTC), ports.PlazoFaseVenceHoy},
		{time.Date(2026, 9, 29, 21, 59, 59, 0, time.UTC), ports.PlazoFaseVenceHoy},
		{time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC), ports.PlazoFaseVencido},
	}
	for _, caso := range casos {
		plazo, aplicable, err := calculadora.CalcularPlazoFase(t.Context(), ports.SolicitudPlazoFaseRRHH{Fase: "fiscalizacion", Desde: desde, Ahora: caso.ahora})
		if err != nil || !aplicable || plazo.Estado != caso.estado {
			t.Fatalf("%s: %+v %v", caso.ahora, plazo, err)
		}
	}
}

func TestPlazoFaseCTSinCatalogoOAmbiguoNoSuponePlazo(t *testing.T) {
	if nuevaCalculadoraPlazoFaseCT(nil) != nil {
		t.Fatal("sin catálogo no hay calculadora")
	}
	contenido, err := os.ReadFile(rutaReglasCTEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	// Dos reglas vigentes para la misma fase: el catálogo es incoherente.
	ambiguo := strings.Replace(string(contenido), `"inicio": "solicitud",`, `"inicio": "solicitud", "fases": "fiscalizacion",`, 1)
	if ambiguo == string(contenido) {
		t.Fatal("no se pudo preparar el catálogo ambiguo")
	}
	ruta := filepath.Join(t.TempDir(), "ct_reglas_ambiguo.demo.json")
	if err := os.WriteFile(ruta, []byte(ambiguo), 0o600); err != nil {
		t.Fatal(err)
	}
	calculadora := calculadoraPlazoFaseCTPrueba(t, ruta, calendariosPlazoFasePrueba(t))
	solicitud := ports.SolicitudPlazoFaseRRHH{Fase: "fiscalizacion", Desde: time.Date(2026, 9, 15, 9, 30, 0, 0, time.UTC), Ahora: relojPresentacionReglasEjemplo.ahora}
	if _, aplicable, err := calculadora.CalcularPlazoFase(t.Context(), solicitud); !errors.Is(err, errPlazoFaseAmbiguo) || aplicable {
		t.Fatalf("regla ambigua aceptada: %v %v", aplicable, err)
	}
	// Sin Calendarios el cómputo administrativo no se supone.
	sinCalendarios := calculadoraPlazoFaseCTPrueba(t, rutaReglasCTEjemploPrueba, nil)
	if _, aplicable, err := sinCalendarios.CalcularPlazoFase(t.Context(), solicitud); !errors.Is(err, reglas.ErrCalculoNoDisponible) || aplicable {
		t.Fatalf("plazo hábil sin calendario: %v %v", aplicable, err)
	}
}
