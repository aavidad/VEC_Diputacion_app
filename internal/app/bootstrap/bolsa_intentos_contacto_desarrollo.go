package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"time"

	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	calendariosdomain "vec-diputacion-granada/internal/modules/calendarios/domain"
	calendariosports "vec-diputacion-granada/internal/modules/calendarios/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

var errCalendarioLlamadasNoDisponible = errors.New("bootstrap: calendario de llamadas no disponible")

// componerIntentosContactoBolsaDesarrollo engancha las reglas b02, b03 y b04
// del catálogo de Bolsa al registro de contactos ya compuesto. Sin catálogo,
// o sin Bolsa compuesta, el registro conserva su conducta de siempre. Sin
// calendarios, una franja limitada a días hábiles no se puede comprobar y el
// intento se rechaza como no disponible en lugar de suponer el día hábil.
func componerIntentosContactoBolsaDesarrollo(resolutor *reglas.Resolutor, calendarios calendariosports.ConsultaCalendarios, mutador http.Handler) error {
	politica := reglasbolsa.NuevosIntentosContacto(resolutor)
	participacion, ok := mutador.(*manejadorParticipacionBolsaDesarrollo)
	if !politica.Configurada() || !ok || participacion == nil || participacion.servicio == nil {
		return nil
	}
	// Un catálogo con las reglas de intentos incompletas impide arrancar,
	// igual que un paquete de reglas ilegible.
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if _, _, _, err := politica.PoliticaIntentosTelefonicos(ctx); err != nil {
		return errors.Join(errReglasEjemploNoValidas, err)
	}
	return participacion.servicio.EstablecerControlIntentos(politica, calendarioLlamadasDesarrollo{consulta: calendarios, sede: reglas.MunicipioSedeDiputacion}, relojCalendariosDesarrollo{}.Ahora)
}

// calendarioLlamadasDesarrollo dice si un día es hábil en la sede con el
// calendario oficial: el primer día hábil contado desde la víspera es el
// propio día solo si este es hábil.
type calendarioLlamadasDesarrollo struct {
	consulta calendariosports.ConsultaCalendarios
	sede     string
}

func (c calendarioLlamadasDesarrollo) EsDiaHabil(ctx context.Context, instante time.Time) (bool, error) {
	if dependenciaMotivosRectificacionAnalisisNula(c.consulta) {
		return false, errCalendarioLlamadasNoDisponible
	}
	dia, err := calendariosdomain.FechaCivilDe(instante)
	if err != nil {
		return false, errCalendarioLlamadasNoDisponible
	}
	vispera, err := dia.SumarDias(-1)
	if err != nil {
		return false, errCalendarioLlamadasNoDisponible
	}
	resultado, err := c.consulta.CalcularPlazo(ctx, calendariosports.SolicitudCalculoPlazo{
		Inicio: vispera, Unidad: calendariosdomain.UnidadDiasHabiles, Cantidad: 1, MunicipioSede: c.sede,
	})
	if err != nil || !resultado.Vencimiento.EsValida() {
		return false, errCalendarioLlamadasNoDisponible
	}
	return resultado.Vencimiento == dia, nil
}
