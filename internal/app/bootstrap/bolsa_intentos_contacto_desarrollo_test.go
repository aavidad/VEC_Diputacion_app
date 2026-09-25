package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
)

// consultaCalendariosHabiles responde el primer día hábil tras la víspera
// según una lista fija de días inhábiles.
type consultaCalendariosHabiles struct {
	calendariosports.ConsultaCalendarios
	inhabiles map[string]bool
	err       error
	pedido    calendariosports.SolicitudCalculoPlazo
}

func (c *consultaCalendariosHabiles) CalcularPlazo(_ context.Context, s calendariosports.SolicitudCalculoPlazo) (calendariosports.ResultadoCalculoPlazo, error) {
	c.pedido = s
	if c.err != nil {
		return calendariosports.ResultadoCalculoPlazo{}, c.err
	}
	dia, _ := s.Inicio.SumarDias(1)
	for c.inhabiles[dia.String()] {
		dia, _ = dia.SumarDias(1)
	}
	var r calendariosports.ResultadoCalculoPlazo
	r.Vencimiento = dia
	return r, nil
}

func TestCalendarioLlamadasDistingueDiasHabiles(t *testing.T) {
	consulta := &consultaCalendariosHabiles{inhabiles: map[string]bool{"2026-10-03": true, "2026-10-04": true}}
	calendario := calendarioLlamadasDesarrollo{consulta: consulta, sede: "municipio:ine:18087"}
	// 22:30 UTC del viernes 2 son ya las 00:30 del sábado 3 en Madrid.
	for instante, esperado := range map[time.Time]bool{
		time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC):   true,
		time.Date(2026, 10, 2, 22, 30, 0, 0, time.UTC): false,
		time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC):   false,
		time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC):   true,
	} {
		habil, err := calendario.EsDiaHabil(t.Context(), instante)
		if err != nil || habil != esperado {
			t.Fatalf("%s: hábil=%v err=%v", instante, habil, err)
		}
	}
	if consulta.pedido.Unidad != calendariosdomain.UnidadDiasHabiles || consulta.pedido.Cantidad != 1 || consulta.pedido.MunicipioSede != "municipio:ine:18087" {
		t.Fatalf("solicitud al calendario: %+v", consulta.pedido)
	}
	consulta.err = errors.New("sin calendario")
	if _, err := calendario.EsDiaHabil(t.Context(), time.Date(2026, 10, 5, 9, 0, 0, 0, time.UTC)); err == nil {
		t.Fatal("un calendario caído no puede dar el día por hábil")
	}
	if _, err := (calendarioLlamadasDesarrollo{}).EsDiaHabil(t.Context(), time.Now()); err == nil {
		t.Fatal("sin calendarios no hay día hábil")
	}
}

func TestComponerIntentosSinCatalogoNoCambiaNada(t *testing.T) {
	if err := componerIntentosContactoBolsaDesarrollo(nil, nil, nil); err != nil {
		t.Fatal(err)
	}
}
