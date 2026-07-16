package memory

import (
	"context"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func (r *RepositorioBaremaciones) AbandonarReserva(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAbandonarReservaBaremacion,
) error {
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahora, err := r.ahora()
	if err != nil {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	accion, _ := accionAbandono(solicitud.Clase)
	if solicitud.Contexto.ValidarVigentePara(
		accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	ahora, err = r.ahora()
	if err != nil {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	if solicitud.Contexto.ValidarVigentePara(
		accion, puertosbolsa.ClaseRecursoBaremacion, solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	huellaToken := huellaTokenReserva(solicitud.Token)
	huellaEfecto := huellaCanonica(
		"abandono-reserva-baremacion-v1", huellaToken, string(solicitud.Clase), solicitud.BaremacionMeritoRef,
	)
	uso, err := nuevoUsoAutorizacionBaremacion(solicitud.Contexto, ahora, huellaEfecto)
	if err != nil {
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	claveAmbito, existe := r.ambitoPorHuellaToken[huellaToken]
	if !existe {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	reserva, existe := r.reservasPorAmbito[claveAmbito]
	if !existe || !cadenasConstantesIguales(reserva.HuellaTokenSHA256, huellaToken) ||
		!mismoVinculoOperacion(reserva.SolicitudReserva.Contexto, solicitud.Contexto) ||
		reserva.SolicitudReserva.Clase != solicitud.Clase ||
		reserva.SolicitudReserva.BaremacionMeritoRef != solicitud.BaremacionMeritoRef {
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
	consumida, err := r.comprobarUsoAutorizacionBloqueado(uso)
	if err != nil {
		return err
	}
	switch reserva.Estado {
	case estadoReservaAbandonada:
		if !consumida {
			return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		return nil
	case estadoReservaActiva:
		if consumida {
			return puertosbolsa.ErrEvidenciaBaremacionNoConfiable
		}
		if !ahora.Before(reserva.SolicitudReserva.ExpiraEn.UTC()) {
			r.cambiarEstadoReservaBloqueado(claveAmbito, reserva, estadoReservaExpirada)
			return puertosbolsa.ErrReservaBaremacionNoValida
		}
		r.cambiarEstadoReservaBloqueado(claveAmbito, reserva, estadoReservaAbandonada)
		r.usosAutorizacion[uso.DecisionRef] = uso
		return nil
	default:
		return puertosbolsa.ErrReservaBaremacionNoValida
	}
}

func (r *RepositorioBaremaciones) ObtenerVersionVigente(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerBaremacionVigente,
) (puertosbolsa.VersionBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if r == nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrBaremacionNoEncontrada
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarBaremacionVigente, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.mu.RLock()
	bloqueoActivo := true
	defer func() {
		if bloqueoActivo {
			r.mu.RUnlock()
		}
	}()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	ahora, err = r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarBaremacionVigente, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	versiones := r.versionesPorBaremacion[solicitud.BaremacionMeritoRef]
	if len(versiones) == 0 || versiones[len(versiones)-1].Agregado.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrBaremacionNoEncontrada
	}
	instantaneas, err := r.instantaneasManifiestosVersionBloqueada(versiones[len(versiones)-1])
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	version, err := versiones[len(versiones)-1].Clonar()
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	r.mu.RUnlock()
	bloqueoActivo = false
	if err := r.verificarInstantaneasManifiestos(ctx, instantaneas); err != nil {
		return puertosbolsa.VersionBaremacion{}, errorVerificacionConContexto(
			ctx, puertosbolsa.ErrEvidenciaBaremacionNoConfiable,
		)
	}
	return version, nil
}

func (r *RepositorioBaremaciones) ObtenerVersion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerVersionBaremacion,
) (puertosbolsa.VersionBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if r == nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarVersionBaremacion, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.mu.RLock()
	bloqueoActivo := true
	defer func() {
		if bloqueoActivo {
			r.mu.RUnlock()
		}
	}()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.VersionBaremacion{}, err
	}
	ahora, err = r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarVersionBaremacion, puertosbolsa.ClaseRecursoBaremacion,
		solicitud.BaremacionMeritoRef, ahora,
	) != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	versiones := r.versionesPorBaremacion[solicitud.BaremacionMeritoRef]
	if solicitud.Numero > uint64(len(versiones)) ||
		versiones[solicitud.Numero-1].Agregado.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	instantaneas, err := r.instantaneasManifiestosVersionBloqueada(versiones[solicitud.Numero-1])
	if err != nil {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	version, err := versiones[solicitud.Numero-1].Clonar()
	if err != nil || version.Referencia.Numero != solicitud.Numero {
		return puertosbolsa.VersionBaremacion{}, puertosbolsa.ErrVersionBaremacionNoEncontrada
	}
	r.mu.RUnlock()
	bloqueoActivo = false
	if err := r.verificarInstantaneasManifiestos(ctx, instantaneas); err != nil {
		return puertosbolsa.VersionBaremacion{}, errorVerificacionConContexto(
			ctx, puertosbolsa.ErrEvidenciaBaremacionNoConfiable,
		)
	}
	return version, nil
}

func (r *RepositorioBaremaciones) ObtenerEvidenciaTransaccion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion,
) (puertosbolsa.EvidenciaTransaccionBaremacionRecuperada, error) {
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	if solicitud.Validar() != nil || r == nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	ahora, err := r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion, puertosbolsa.ClaseRecursoTransaccion,
		solicitud.AuditoriaRef, ahora,
	) != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	r.mu.RLock()
	bloqueoActivo := true
	defer func() {
		if bloqueoActivo {
			r.mu.RUnlock()
		}
	}()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, err
	}
	ahora, err = r.ahora()
	if err != nil || solicitud.Contexto.ValidarVigentePara(
		puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion, puertosbolsa.ClaseRecursoTransaccion,
		solicitud.AuditoriaRef, ahora,
	) != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
	if !r.cadenasIntegrasBloqueadas() {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	indice := -1
	for actual := range r.auditorias {
		if r.auditorias[actual].Referencia == solicitud.AuditoriaRef {
			indice = actual
			break
		}
	}
	if indice < 0 || indice >= len(r.eventosOutbox) ||
		r.eventosOutbox[indice].Referencia != solicitud.EventoOutboxRef {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoEncontrada
	}
	auditoria, evento := r.auditorias[indice], r.eventosOutbox[indice]
	versiones := r.versionesPorBaremacion[solicitud.BaremacionMeritoRef]
	if solicitud.NumeroVersion > uint64(len(versiones)) || auditoria.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoEncontrada
	}
	instantaneas, err := r.instantaneasManifiestosVersionBloqueada(versiones[solicitud.NumeroVersion-1])
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	version, err := versiones[solicitud.NumeroVersion-1].Clonar()
	if err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	auditoria.CamposPermitidos = append([]string(nil), auditoria.CamposPermitidos...)
	var manifiesto *puertosbolsa.ManifiestoProbatorioBaremacion
	if len(instantaneas) != 0 {
		actual := instantaneas[len(instantaneas)-1].Clonar()
		manifiesto = &actual
	}
	resultado := puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{
		Version: version, Auditoria: auditoria, Evento: evento, Manifiesto: manifiesto,
		Evidencia: puertosbolsa.EvidenciaTransaccionBaremacion{
			AuditoriaRef: auditoria.Referencia, HuellaAuditoriaSHA256: auditoria.HuellaRegistroSHA256,
			EventoOutboxRef: evento.Referencia, HuellaEventoOutboxSHA256: evento.HuellaRegistroSHA256,
			ConfirmadaEn: auditoria.RegistradaEn,
		},
	}
	if resultado.ValidarPara(solicitud) != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, puertosbolsa.ErrEvidenciaBaremacionNoConfiable
	}
	r.mu.RUnlock()
	bloqueoActivo = false
	if err := r.verificarInstantaneasManifiestos(ctx, instantaneas); err != nil {
		return puertosbolsa.EvidenciaTransaccionBaremacionRecuperada{}, errorVerificacionConContexto(
			ctx, puertosbolsa.ErrEvidenciaBaremacionNoConfiable,
		)
	}
	return resultado, nil
}

func (r *RepositorioBaremaciones) comprobarVersionEsperadaBloqueado(
	solicitud puertosbolsa.SolicitudReservarCambioBaremacion,
	ahora time.Time,
) error {
	versiones := r.versionesPorBaremacion[solicitud.BaremacionMeritoRef]
	switch solicitud.Clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		if len(versiones) != 0 {
			if versiones[len(versiones)-1].Agregado.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
				return puertosbolsa.ErrBaremacionNoEncontrada
			}
			if ahora.Before(versiones[len(versiones)-1].ConfirmadaEn) {
				return puertosbolsa.ErrSolicitudBaremacionInvalida
			}
			return puertosbolsa.ErrBaremacionYaExiste
		}
		return nil
	case puertosbolsa.ClaseCambioIncorporarDecision:
		if len(versiones) == 0 {
			return puertosbolsa.ErrBaremacionNoEncontrada
		}
		if versiones[len(versiones)-1].Agregado.SujetoRef != solicitud.Contexto.Proyeccion().SujetoRef {
			return puertosbolsa.ErrBaremacionNoEncontrada
		}
		if ahora.Before(versiones[len(versiones)-1].ConfirmadaEn) {
			return puertosbolsa.ErrSolicitudBaremacionInvalida
		}
		actual := versiones[len(versiones)-1].Referencia
		if !referenciasVersionIguales(&actual, solicitud.VersionEsperada) {
			return puertosbolsa.ErrVersionBaremacionConflicto
		}
		return nil
	default:
		return puertosbolsa.ErrSolicitudBaremacionInvalida
	}
}

func (r *RepositorioBaremaciones) validarCambioBloqueado(
	reserva puertosbolsa.SolicitudReservarCambioBaremacion,
	agregado dominiobolsa.BaremacionMerito,
	huellaNueva string,
	confirmadaEn time.Time,
) (*puertosbolsa.VersionBaremacion, uint64, error) {
	versiones := r.versionesPorBaremacion[agregado.ID]
	switch reserva.Clase {
	case puertosbolsa.ClaseCambioAltaBaremacion:
		if reserva.VersionEsperada != nil || len(versiones) != 0 {
			return nil, 0, puertosbolsa.ErrBaremacionYaExiste
		}
		if len(agregado.Decisiones) != 0 {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		return nil, 1, nil
	case puertosbolsa.ClaseCambioIncorporarDecision:
		if reserva.VersionEsperada == nil || len(versiones) == 0 {
			return nil, 0, puertosbolsa.ErrBaremacionNoEncontrada
		}
		actual, err := versiones[len(versiones)-1].Clonar()
		if err != nil || !referenciasVersionIguales(&actual.Referencia, reserva.VersionEsperada) {
			return nil, 0, puertosbolsa.ErrVersionBaremacionConflicto
		}
		if confirmadaEn.Before(actual.ConfirmadaEn) {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		if len(agregado.Decisiones) != len(actual.Agregado.Decisiones)+1 {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		esperado, err := actual.Agregado.IncorporarDecision(agregado.Decisiones[len(agregado.Decisiones)-1])
		if err != nil {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		huellaEsperada, err := esperado.HuellaEstadoSHA256()
		if err != nil || !cadenasConstantesIguales(huellaEsperada, huellaNueva) {
			return nil, 0, puertosbolsa.ErrHistorialBaremacionNoAnexable
		}
		return &actual, actual.Referencia.Numero + 1, nil
	default:
		return nil, 0, puertosbolsa.ErrSolicitudBaremacionInvalida
	}
}

func (r *RepositorioBaremaciones) cambiarEstadoReservaBloqueado(
	claveAmbito string,
	reserva reservaBaremacion,
	estado estadoReserva,
) {
	reserva.Estado = estado
	r.reservasPorAmbito[claveAmbito] = reserva
	if r.ambitoActivoPorBaremacion[reserva.SolicitudReserva.BaremacionMeritoRef] == claveAmbito {
		delete(r.ambitoActivoPorBaremacion, reserva.SolicitudReserva.BaremacionMeritoRef)
	}
}
