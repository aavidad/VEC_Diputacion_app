package application

import (
	"context"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

const rutaReglasBolsaPlazoPrueba = "../../../../data/demo/reglas/bolsa_reglas.ejemplo.demo.json"

type relojPlazoPrueba time.Time

func (r relojPlazoPrueba) Ahora() time.Time { return time.Time(r) }

type calculadoraPlazoPrueba struct {
	recibida reglas.SolicitudVencimiento
	err      error
}

func (c *calculadoraPlazoPrueba) CalcularVencimiento(_ context.Context, s reglas.SolicitudVencimiento) (reglas.Vencimiento, error) {
	c.recibida = s
	if c.err != nil {
		return reglas.Vencimiento{}, c.err
	}
	madrid, _ := time.LoadLocation("Europe/Madrid")
	return reglas.Vencimiento{UltimoDia: "2026-09-29", VenceAntesDe: time.Date(2026, 9, 30, 0, 0, 0, 0, madrid), Calendarios: []string{"calendario:v1"}}, nil
}

func resolutorPlazoPrueba(t *testing.T, instante time.Time, calculadora reglas.CalculadoraPlazos) *reglas.Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(rutaReglasBolsaPlazoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: reglas.CatalogoBolsa, ModuloID: reglas.ModuloBolsa,
		Reloj: relojPlazoPrueba(instante), Calculadora: calculadora, MunicipioSede: reglas.MunicipioSedeDiputacion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func TestPlazoRespuestaConCatalogoDevuelveReglaYVencimiento(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 8, 30, 15, 0, time.UTC)
	calculadora := &calculadoraPlazoPrueba{}
	servicio, err := NuevoServicioPlazoRespuestaLlamamiento(resolutorPlazoPrueba(t, ahora, calculadora), func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	plazo, err := servicio.ConsultarPlazoRespuesta(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !plazo.Configurada || plazo.Regla.Referencia != "vec.bolsa.reglas:1:b05.plazo_respuesta" ||
		plazo.Regla.Origen != string(reglas.OrigenEjemplo) || !plazo.Regla.Ejemplo || plazo.Regla.Articulo != "" ||
		plazo.Regla.Texto == "" || plazo.Regla.Etiqueta == "" {
		t.Fatalf("regla inesperada: %+v", plazo.Regla)
	}
	if calculadora.recibida.Inicio != ahora || calculadora.recibida.Unidad != reglas.UnidadDiasHabiles ||
		calculadora.recibida.Cantidad != 1 || calculadora.recibida.Computo != reglas.ComputoAdministrativo ||
		calculadora.recibida.MunicipioSede != reglas.MunicipioSedeDiputacion {
		t.Fatalf("solicitud de cálculo inesperada: %+v", calculadora.recibida)
	}
	if plazo.UltimoDia != "2026-09-29" || !plazo.CalculadoEn.Equal(ahora) ||
		!plazo.VenceEn.Equal(time.Date(2026, 9, 29, 21, 59, 59, 0, time.UTC)) ||
		!plazo.VenceAntesDe.Equal(time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC)) {
		t.Fatalf("vencimiento inesperado: %+v", plazo)
	}
}

func TestPlazoRespuestaSinCatalogoMantieneTextoLibre(t *testing.T) {
	var sinCatalogo *reglas.Resolutor
	servicio, err := NuevoServicioPlazoRespuestaLlamamiento(sinCatalogo, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	plazo, err := servicio.ConsultarPlazoRespuesta(t.Context())
	if err != nil || plazo != (puertosbolsa.PlazoRespuestaLlamamiento{}) {
		t.Fatalf("sin catálogo debe responder no configurado: %+v %v", plazo, err)
	}
}

func TestPlazoRespuestaCalculoNoDisponibleNoInventaPlazo(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 8, 0, 0, 0, time.UTC)
	servicio, err := NuevoServicioPlazoRespuestaLlamamiento(resolutorPlazoPrueba(t, ahora, &calculadoraPlazoPrueba{err: errors.New("calendario caído")}), func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	if plazo, err := servicio.ConsultarPlazoRespuesta(t.Context()); !errors.Is(err, puertosbolsa.ErrPlazoRespuestaNoDisponible) || plazo.Configurada {
		t.Fatalf("un fallo de cálculo no puede proponer plazo: %+v %v", plazo, err)
	}
	sinCalculadora, err := NuevoServicioPlazoRespuestaLlamamiento(resolutorPlazoPrueba(t, ahora, nil), func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sinCalculadora.ConsultarPlazoRespuesta(t.Context()); !errors.Is(err, puertosbolsa.ErrPlazoRespuestaNoDisponible) {
		t.Fatalf("sin calculadora: %v", err)
	}
	if _, err := NuevoServicioPlazoRespuestaLlamamiento(nil, time.Now); !errors.Is(err, puertosbolsa.ErrPlazoRespuestaNoDisponible) {
		t.Fatalf("sin resolutor: %v", err)
	}
}
