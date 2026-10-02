package politicacopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"
	d "vec-diputacion-granada/internal/modules/administracion/domain/politicacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/politicacopias"
)

type Servicio struct {
	Repositorio p.Repositorio
	Autoridad   p.Autoridad
	Reloj       p.Reloj
	Reservador  p.Reservador
	Ejecutor    p.Ejecutor
	Notificador p.NotificadorFallo
}

func (s Servicio) preparado() bool {
	return s.Repositorio != nil && s.Autoridad != nil && s.Reloj != nil
}
func atribucion(a p.Atribucion, doble bool) error {
	if !d.Referencia(a.Actor) || !d.Referencia(a.Correlacion) || (a.Revisor != "" && !d.Referencia(a.Revisor)) || (doble && (a.Revisor == "" || a.Revisor == a.Actor)) {
		return d.ErrDenegado
	}
	return nil
}
func (s Servicio) Configurar(ctx context.Context, esperada uint64, policy d.Politica) (p.Registro, error) {
	if !s.preparado() {
		return p.Registro{}, d.ErrDenegado
	}
	if policy.Validar() != nil {
		return p.Registro{}, d.ErrEntrada
	}
	policy = policy.Normalizar()
	// Resolve permission before reading policy history; require double control for
	// a new policy and for retention changes under either the old or new policy.
	intent := p.Intencion{Accion: "configurar_politica_copias", Politica: policy.Referencia, Destino: policy.Destino, PoliticaSHA256: policy.SHA256(), VersionEsperada: esperada}
	a, e := s.Autoridad.Autorizar(ctx, intent)
	if e != nil {
		return p.Registro{}, d.ErrDenegado
	}
	originalActor := a.Actor
	current, e := s.Repositorio.Actual(ctx)
	if e != nil {
		return p.Registro{}, e
	}
	if current.Version != esperada {
		return p.Registro{}, d.ErrVersion
	}
	if current.Version > 0 && current.Politica.Referencia != policy.Referencia {
		return p.Registro{}, d.ErrEntrada
	}
	var oldIntent *p.Intencion
	if current.Version > 0 {
		i := p.Intencion{Accion: "configurar_politica_copias", Politica: current.Politica.Referencia, Destino: current.Politica.Destino, PoliticaSHA256: current.SHA256, VersionEsperada: current.Version}
		prior, err := s.Autoridad.Autorizar(ctx, i)
		if err != nil || prior.Actor != a.Actor || atribucion(prior, false) != nil {
			return p.Registro{}, d.ErrDenegado
		}
		oldIntent = &i
	}
	intent.DobleControl = (current.Version == 0 || !reflect.DeepEqual(current.Politica.Normalizar().Retencion, policy.Retencion)) && (policy.Retencion.ExigeDobleControl() || current.Version > 0 && current.Politica.Retencion.ExigeDobleControl())
	if intent.DobleControl {
		a, e = s.Autoridad.Autorizar(ctx, intent)
		if e != nil {
			return p.Registro{}, d.ErrDenegado
		}
	}
	if a.Actor != originalActor {
		return p.Registro{}, d.ErrDenegado
	}
	if e = atribucion(a, intent.DobleControl); e != nil {
		return p.Registro{}, e
	}
	return s.Repositorio.Guardar(ctx, esperada, policy, a, s.Reloj.Ahora().UTC(), func(ctx context.Context) error {
		if oldIntent != nil && s.Autoridad.Revalidar(ctx, *oldIntent, a) != nil {
			return d.ErrDenegado
		}
		if s.Autoridad.Revalidar(ctx, intent, a) != nil {
			return d.ErrDenegado
		}
		return nil
	})
}
func (s Servicio) Consultar(ctx context.Context) ([]p.Registro, error) {
	if !s.preparado() {
		return nil, d.ErrDenegado
	}
	history, e := s.Repositorio.Historia(ctx)
	if e != nil {
		return nil, e
	}
	if len(history) == 0 {
		return nil, d.ErrDenegado
	}
	for _, r := range history {
		intent := p.Intencion{Accion: "consultar_politica_copias", Politica: r.Politica.Referencia, Destino: r.Politica.Destino, PoliticaSHA256: r.SHA256, VersionEsperada: r.Version}
		a, e := s.Autoridad.Autorizar(ctx, intent)
		if e != nil || atribucion(a, false) != nil {
			return nil, d.ErrDenegado
		}
		if s.Autoridad.Revalidar(ctx, intent, a) != nil {
			return nil, d.ErrDenegado
		}
	}
	return history, nil
}
func (s Servicio) Planificar(ctx context.Context, copias []d.Copia) (d.PlanRetencion, error) {
	history, e := s.Consultar(ctx)
	if e != nil {
		return d.PlanRetencion{}, e
	}
	if len(history) == 0 {
		return d.PlanRetencion{}, d.ErrEntrada
	}
	return history[len(history)-1].Politica.PlanificarRetencion(copias, s.Reloj.Ahora())
}
func (s Servicio) Proximos(ctx context.Context, desde time.Time, cantidad int) ([]d.EventoAgenda, error) {
	history, e := s.Consultar(ctx)
	if e != nil {
		return nil, e
	}
	if len(history) == 0 {
		return nil, d.ErrEntrada
	}
	return history[len(history)-1].Politica.Proximos(desde, cantidad)
}

// Ejecutar schedules only the current policy, during its exact civil window.
// The register and executor retain ownership of operation state and retries.
func (s Servicio) Ejecutar(ctx context.Context, version uint64, fecha time.Time) (p.ReciboReserva, error) {
	if !s.preparado() || s.Reservador == nil || s.Ejecutor == nil {
		return p.ReciboReserva{}, d.ErrDependencia
	}
	current, e := s.Repositorio.Actual(ctx)
	if e != nil {
		return p.ReciboReserva{}, e
	}
	if current.Version != version || version == 0 {
		return p.ReciboReserva{}, d.ErrVersion
	}
	events, e := current.Politica.Proximos(fecha, 1)
	if e != nil || !events[0].Fecha.Equal(fecha) {
		return p.ReciboReserva{}, d.ErrVentana
	}
	now := s.Reloj.Ahora()
	event := events[0]
	if now.Before(event.Fecha) || !now.Before(event.FinVentana) {
		return p.ReciboReserva{}, d.ErrVentana
	}
	reserve := p.Reserva{Politica: current.Politica.Referencia, Version: version, PoliticaSHA256: current.SHA256, Destino: current.Politica.Destino, Evento: event}
	b, _ := json.Marshal(struct{ Politica, Destino, FechaCivil string }{reserve.Politica, reserve.Destino, reserve.Evento.FechaCivil})
	h := sha256.Sum256(b)
	reserve.Clave = "agenda:" + hex.EncodeToString(h[:])
	intent := p.Intencion{Accion: "ejecutar_copia_programada", Politica: reserve.Politica, Destino: reserve.Destino, PoliticaSHA256: reserve.PoliticaSHA256, VersionEsperada: version, ClaveEjecucion: reserve.Clave}
	a, e := s.Autoridad.Autorizar(ctx, intent)
	if e != nil || atribucion(a, false) != nil {
		return p.ReciboReserva{}, d.ErrDenegado
	}
	// Serialize policy CAS and scheduling authorization through Guardar's sibling
	// lock operation when supported, preventing a revoked policy version race.
	guard, ok := s.Repositorio.(p.Versionador)
	if !ok {
		return p.ReciboReserva{}, d.ErrDependencia
	}
	var receipt p.ReciboReserva
	e = guard.ConVersion(ctx, version, func(ctx context.Context) error {
		if s.Autoridad.Revalidar(ctx, intent, a) != nil {
			return d.ErrDenegado
		}
		var err error
		receipt, err = s.Reservador.Reservar(ctx, reserve, a)
		if err != nil {
			return err
		}
		if !d.Referencia(receipt.Operacion) {
			return d.ErrEntrada
		}
		if s.Autoridad.Revalidar(ctx, intent, a) != nil {
			return d.ErrDenegado
		}
		n := s.Reloj.Ahora()
		if n.Before(event.Fecha) || !n.Before(event.FinVentana) {
			return d.ErrVentana
		}
		return s.Ejecutor.Ejecutar(ctx, reserve, receipt, a)
	})
	if e != nil {
		receipt.AvisoFallo = "no_configurado"
		if s.Notificador != nil {
			if notifyErr := s.Notificador.NotificarFallo(ctx, p.FalloAgenda{Clave: reserve.Clave, Operacion: receipt.Operacion, Politica: reserve.Politica, Destino: reserve.Destino, Version: version, Motivo: "ejecucion_programada_fallida"}); notifyErr != nil {
				receipt.AvisoFallo = "fallido"
				e = errors.Join(e, d.ErrAviso)
			} else {
				receipt.AvisoFallo = "emitido_sin_acuse"
			}
		}
	}
	return receipt, e
}
