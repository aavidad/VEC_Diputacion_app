package application

import (
	"context"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

// controlIntentosContacto aplica las reglas de intentos telefónicos del
// catálogo (b02, b03 y b04) al registro de contactos. Nulo: sin catálogo, el
// registro conserva su conducta de siempre.
type controlIntentosContacto struct {
	politica   puertosbolsa.PoliticaIntentosContacto
	calendario puertosbolsa.CalendarioDiasHabiles
	reloj      func() time.Time
}

type intentoPreparado struct {
	politica      dominiobolsa.PoliticaIntentosTelefonicos
	diaHabil      bool
	fechaDiaHabil time.Time
}

// EstablecerControlIntentos compone el control de intentos. Se llama solo
// durante la composición, antes de atender peticiones.
func (s *ServicioContactoParticipacion) EstablecerControlIntentos(politica puertosbolsa.PoliticaIntentosContacto, calendario puertosbolsa.CalendarioDiasHabiles, reloj func() time.Time) error {
	if s == nil || politica == nil || calendario == nil || reloj == nil {
		return puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	s.intentos = &controlIntentosContacto{politica: politica, calendario: calendario, reloj: reloj}
	return nil
}

// prepararIntento resuelve la política para un intento telefónico ligado a
// un llamamiento y aplica la franja, que no depende del histórico. Devuelve
// nil si no procede control.
func (s *ServicioContactoParticipacion) prepararIntento(ctx context.Context, c dominiobolsa.ContactoParticipacion) (*intentoPreparado, error) {
	if s.intentos == nil || c.Canal != dominiobolsa.CanalContactoTelefono || c.LlamamientoRef == "" {
		return nil, nil
	}
	politica, _, configurada, err := s.intentos.politica.PoliticaIntentosTelefonicos(ctx)
	if err != nil || (configurada && politica.Validar() != nil) {
		return nil, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	if !configurada {
		return nil, nil
	}
	preparado := &intentoPreparado{politica: politica, diaHabil: true}
	instanteEvaluado := c.Instante
	if c.InstanteServidor {
		// Este reloj sólo prepara la consulta de calendario. La fecha final
		// sale de PostgreSQL y se compara allí antes del INSERT.
		instanteEvaluado = s.intentos.reloj().UTC()
		if instanteEvaluado.IsZero() {
			return nil, puertosbolsa.ErrContactoParticipacionNoDisponible
		}
	}
	if politica.Franja.Zona != nil && politica.Franja.SoloDiasHabiles {
		fechaLocal := instanteEvaluado.In(politica.Franja.Zona)
		preparado.fechaDiaHabil = time.Date(fechaLocal.Year(), fechaLocal.Month(), fechaLocal.Day(), 0, 0, 0, 0, time.UTC)
		instanteCalendario := instanteEvaluado
		if c.InstanteServidor {
			instanteCalendario = fechaLocal
		}
		if preparado.diaHabil, err = s.intentos.calendario.EsDiaHabil(ctx, instanteCalendario); err != nil {
			return nil, puertosbolsa.ErrContactoParticipacionNoDisponible
		}
	}
	if !c.InstanteServidor && politica.Franja.Control == dominiobolsa.ControlReglaImpedir && len(politica.Franja.AvisosFranja(c.Instante, preparado.diaHabil)) > 0 {
		return nil, dominiobolsa.ErrIntentoFueraDeFranja
	}
	return preparado, nil
}

// completarIntento calcula, con el resumen previo que devolvió el
// repositorio, los avisos del intento y el estado del llamamiento tras él.
func completarIntento(p *intentoPreparado, c dominiobolsa.ContactoParticipacion, registro *puertosbolsa.RegistroContactoParticipacion) error {
	if p == nil {
		return nil
	}
	if registro.ResumenPrevio == nil {
		return puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	previo := *registro.ResumenPrevio
	avisos, err := dominiobolsa.EvaluarIntentoTelefonico(p.politica, previo, c.Instante, p.diaHabil)
	if err != nil && !registro.Reutilizado {
		// El repositorio ya ha aplicado estas reglas bajo cerrojo: una
		// discrepancia no se oculta.
		return puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	estado := dominiobolsa.EstadoIntentos(p.politica, previo.Sumar(p.politica, c.Resultado, c.Instante))
	estado.Avisos = avisos
	registro.Intentos = &estado
	return nil
}

// EstadoIntentosTelefonicos evalúa el control de intentos de un llamamiento
// sobre el histórico ya leído con la autorización de consulta de contactos.
func (s *ServicioContactoParticipacion) EstadoIntentosTelefonicos(ctx context.Context, llamamientoRef string, contactos []dominiobolsa.ContactoParticipacion, completo bool) (puertosbolsa.EstadoIntentosContacto, error) {
	if ctx == nil || s == nil || llamamientoRef == "" {
		return puertosbolsa.EstadoIntentosContacto{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	if s.intentos == nil {
		return puertosbolsa.EstadoIntentosContacto{LlamamientoRef: llamamientoRef}, nil
	}
	politica, reglas, configurada, err := s.intentos.politica.PoliticaIntentosTelefonicos(ctx)
	if err != nil || (configurada && politica.Validar() != nil) {
		return puertosbolsa.EstadoIntentosContacto{}, puertosbolsa.ErrContactoParticipacionNoDisponible
	}
	if !configurada {
		return puertosbolsa.EstadoIntentosContacto{LlamamientoRef: llamamientoRef}, nil
	}
	ahora := s.intentos.reloj().UTC()
	estado := dominiobolsa.EstadoIntentos(politica, dominiobolsa.ResumirIntentosTelefonicos(politica, llamamientoRef, contactos))
	if !estado.BajaPropuesta && !estado.Contactado && politica.Franja.Zona != nil {
		diaHabil := true
		if politica.Franja.SoloDiasHabiles {
			if diaHabil, err = s.intentos.calendario.EsDiaHabil(ctx, ahora); err != nil {
				return puertosbolsa.EstadoIntentosContacto{}, puertosbolsa.ErrContactoParticipacionNoDisponible
			}
		}
		estado.Avisos = politica.Franja.AvisosFranja(ahora, diaHabil)
	}
	if !estado.SiguientePermitidoDesde.IsZero() && ahora.Before(estado.SiguientePermitidoDesde) {
		estado.Avisos = append(estado.Avisos, dominiobolsa.AvisoIntentoAntesDeSeparacion)
	}
	return puertosbolsa.EstadoIntentosContacto{
		Configurada: true, LlamamientoRef: llamamientoRef, Estado: estado, Politica: politica,
		Reglas: append([]puertosbolsa.ReglaIntentosContacto(nil), reglas...), Completo: completo, Ahora: ahora,
	}, nil
}
