package domain

func (e Expediente) Validar() error {
	if !referenciaValida(e.Referencia) ||
		!referenciaValida(e.OrganizacionRef) ||
		!patronNumero.MatchString(e.NumeroVisible) ||
		e.Version == 0 || e.Flujo.Validar() != nil || !e.FaseActual.Valida() ||
		!e.EstadoActual.Valido() || e.Solicitud.Validar() != nil ||
		!instanteCanonico(e.CreadoEn) || !instanteCanonico(e.ActualizadoEn) ||
		e.ActualizadoEn.Before(e.CreadoEn) || len(e.Actuaciones) == 0 ||
		uint64(len(e.Actuaciones)) != e.Version {
		return ErrExpedienteInvalido
	}
	if e.Analisis != nil && e.Analisis.Validar() != nil {
		return ErrExpedienteInvalido
	}
	if e.ViaCobertura != nil && (e.Analisis == nil || !e.Analisis.HabilitaAvance() ||
		e.ViaCobertura.Validar() != nil) {
		return ErrExpedienteInvalido
	}
	if e.Asignacion != nil && (e.ViaCobertura == nil || e.Asignacion.Validar() != nil) {
		return ErrExpedienteInvalido
	}
	for indice, actuacion := range e.Actuaciones {
		if !actuacionValida(actuacion, uint64(indice+1)) {
			return ErrExpedienteInvalido
		}
		if indice == 0 {
			if actuacion.FaseOrigen != "" || actuacion.EstadoOrigen != EstadoPendiente ||
				!actuacion.RealizadaEn.Equal(e.CreadoEn) {
				return ErrExpedienteInvalido
			}
			continue
		}
		anterior := e.Actuaciones[indice-1]
		if actuacion.FaseOrigen != anterior.FaseDestino ||
			actuacion.EstadoOrigen != anterior.EstadoDestino ||
			actuacion.RealizadaEn.Before(anterior.RealizadaEn) {
			return ErrExpedienteInvalido
		}
	}
	if e.Analisis != nil && !analisisLigadoAActuacion(e.Analisis, e.Actuaciones) {
		return ErrExpedienteInvalido
	}
	ultima := e.Actuaciones[len(e.Actuaciones)-1]
	if ultima.FaseDestino != e.FaseActual || ultima.EstadoDestino != e.EstadoActual ||
		!ultima.RealizadaEn.Equal(e.ActualizadoEn) {
		return ErrExpedienteInvalido
	}
	return nil
}

func analisisLigadoAActuacion(analisis *AnalisisRRHH, actuaciones []Actuacion) bool {
	if analisis == nil || analisis.ActuacionRegistro == nil ||
		analisis.ActuacionRegistro.validar() != nil ||
		analisis.ActuacionRegistro.Secuencia > uint64(len(actuaciones)) {
		return false
	}
	actuacion := actuaciones[analisis.ActuacionRegistro.Secuencia-1]
	return analisis.ActuacionRegistro.correspondeA(actuacion) &&
		!analisis.ValidacionRC.ValidadaEn.After(actuacion.RealizadaEn)
}

func actuacionValida(a Actuacion, secuencia uint64) bool {
	return a.Secuencia == secuencia && a.VersionExpediente == secuencia &&
		a.AccionClave.Valida() && referenciaValida(a.ActorRef) &&
		referenciaValida(a.UnidadRef) && referenciaValida(a.ReciboRef) &&
		instanteCanonico(a.RealizadaEn) && a.FaseDestino.Valida() &&
		a.EstadoOrigen.Valido() && a.EstadoDestino.Valido() &&
		!(!a.FaseOrigen.Valida() && a.FaseOrigen != "") &&
		textoValido(a.Observaciones, 2000, true) &&
		referenciasUnicasValidas(a.DocumentosRef, 64)
}

func (e Expediente) Clonar() Expediente {
	e.Solicitud = e.Solicitud.clonar()
	if e.Analisis != nil {
		clon := e.Analisis.clonar()
		e.Analisis = &clon
	}
	if e.ViaCobertura != nil {
		clon := e.ViaCobertura.clonar()
		e.ViaCobertura = &clon
	}
	if e.Asignacion != nil {
		clon := *e.Asignacion
		e.Asignacion = &clon
	}
	e.Actuaciones = append([]Actuacion(nil), e.Actuaciones...)
	for indice := range e.Actuaciones {
		e.Actuaciones[indice].DocumentosRef = append(
			[]string(nil), e.Actuaciones[indice].DocumentosRef...,
		)
	}
	return e
}
