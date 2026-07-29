package ports

import "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"

func (d DetalleExpedienteRRHH) validarEstructura() error {
	if d.validarContenidoEstructura() != nil ||
		d.validarEvidenciaEstructura() != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (d DetalleExpedienteRRHH) validarEvidenciaEstructura() error {
	if d.Lectura.validar() != nil ||
		d.Lectura.expedienteRef != d.Resumen.ExpedienteRef ||
		d.Lectura.version != d.Resumen.Version ||
		d.Lectura.totalPublicado != 1 ||
		d.Lectura.registradaEn.Before(d.Resumen.ActualizadoEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

// validarContenidoEstructura comprueba la proyección antes de que exista el
// recibo durable. No autoriza ni publica el detalle: solo permite calcular su
// huella dentro de la misma transacción que registrará después la lectura.
func (d DetalleExpedienteRRHH) validarContenidoEstructura() error {
	if d.validarContenidoPublicable() != nil ||
		d.validarIntegridadContenidoEstructura() != nil {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (d DetalleExpedienteRRHH) validarIntegridadContenidoEstructura() error {
	if !d.huellaCoincide() ||
		d.bloques != 0 &&
			d.bloques != bloqueAnalisisRRHH &&
			d.bloques != bloqueAnalisisRRHH|bloqueCoberturaRRHH &&
			d.bloques != bloqueAnalisisRRHH|bloqueCoberturaRRHH|bloqueAsignacionRRHH {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	debeTenerAnalisis := d.bloques&bloqueAnalisisRRHH != 0
	debeTenerCobertura := d.bloques&bloqueCoberturaRRHH != 0
	debeTenerAsignacion := d.bloques&bloqueAsignacionRRHH != 0
	if (d.Analisis != nil) != debeTenerAnalisis ||
		(d.Cobertura != nil) != debeTenerCobertura ||
		(d.Asignacion != nil) != debeTenerAsignacion {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if d.Analisis != nil && !d.Analisis.vinculo.coincide(d.Hitos) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if d.Cobertura != nil {
		if !d.Cobertura.vinculo.coincide(d.Hitos) ||
			d.Cobertura.vinculo.secuencia <= d.Analisis.vinculo.secuencia {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	}
	if d.Asignacion != nil &&
		(!d.Asignacion.vinculo.coincide(d.Hitos) ||
			!d.Asignacion.AsignadaEn.Equal(d.Asignacion.vinculo.realizadaEn) ||
			d.Asignacion.vinculo.secuencia <= d.Cobertura.vinculo.secuencia) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (d DetalleExpedienteRRHH) validarContenidoPublicable() error {
	if d.Resumen.Validar() != nil ||
		d.Solicitud.validar() != nil ||
		len(d.Hitos) < 1 ||
		uint64(len(d.Hitos)) != d.Resumen.Version {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if d.Analisis == nil {
		if d.Resumen.ModalidadClave != "" {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	} else if d.Analisis.validar() != nil ||
		d.Resumen.ModalidadClave != d.Analisis.ModalidadClave ||
		d.Resumen.CategoriaRef != d.Analisis.CategoriaRef {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if d.Cobertura != nil &&
		(d.Analisis == nil || d.Cobertura.validar() != nil) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	if d.Asignacion == nil {
		if d.Resumen.UnidadRef != "" {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	} else if d.Cobertura == nil || d.Asignacion.validar() != nil ||
		d.Resumen.UnidadRef != d.Asignacion.UnidadRef ||
		d.Asignacion.AsignadaEn.Before(d.Resumen.CreadoEn) ||
		d.Asignacion.AsignadaEn.After(d.Resumen.ActualizadoEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return d.validarHitos()
}

// ValidarContenidoPublicablePara comprueba el detalle neutral contra la
// intención de lectura. No valida ni expone autoridad, ámbito, vínculos
// probatorios privados o recibos.
func (d DetalleExpedienteRRHH) ValidarContenidoPublicablePara(
	solicitud SolicitudDetalleRRHH,
) error {
	if solicitud.validar() != nil ||
		d.validarContenidoPublicable() != nil ||
		d.Resumen.ExpedienteRef != solicitud.expedienteRef ||
		(solicitud.versionObservada != 0 &&
			solicitud.versionObservada != d.Resumen.Version) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (d DetalleExpedienteRRHH) validarHitos() error {
	for i, hito := range d.Hitos {
		if hito.validar() != nil ||
			hito.Secuencia != uint64(i+1) ||
			hito.RealizadaEn.After(d.Resumen.ActualizadoEn) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
		if i == 0 {
			if hito.FaseOrigen != "" ||
				hito.EstadoOrigen != domain.EstadoPendiente {
				return ErrResultadoConsultaRRHHNoConfiable
			}
			continue
		}
		anterior := d.Hitos[i-1]
		if hito.FaseOrigen != anterior.FaseDestino ||
			hito.EstadoOrigen != anterior.EstadoDestino ||
			hito.RealizadaEn.Before(anterior.RealizadaEn) {
			return ErrResultadoConsultaRRHHNoConfiable
		}
	}
	ultimo := d.Hitos[len(d.Hitos)-1]
	if ultimo.FaseDestino != d.Resumen.FaseClave ||
		ultimo.EstadoDestino != d.Resumen.EstadoClave ||
		!ultimo.RealizadaEn.Equal(d.Resumen.ActualizadoEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (d DetalleExpedienteRRHH) ValidarPara(
	orden OrdenConsultaDetalleRRHH,
) error {
	solicitud := orden.solicitud
	if d.ValidarContenidoPublicablePara(solicitud) != nil ||
		orden.capacidad.validaPara(
			orden.contexto, DominioHuellaConsultaDetalleRRHH,
			orden.consultaHuella, AccionConsultarDetalleRRHH,
			FinalidadConsultarDetalleRRHH,
			solicitud.expedienteRef, orden.instante,
		) != nil ||
		d.validarIntegridadContenidoEstructura() != nil ||
		d.validarEvidenciaEstructura() != nil ||
		!d.Resumen.cumpleAmbito(orden.capacidad) ||
		!d.Lectura.coincideCon(
			orden.contexto, orden.capacidad,
			solicitud.expedienteRef, d.Resumen.Version,
		) ||
		d.Lectura.registradaEn.Before(orden.instante) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (d DetalleExpedienteRRHH) Clonar() DetalleExpedienteRRHH {
	d.Hitos = append([]HitoExpedienteRRHH(nil), d.Hitos...)
	if d.Analisis != nil {
		analisis := *d.Analisis
		if analisis.CostePrevisto != nil {
			coste := *analisis.CostePrevisto
			analisis.CostePrevisto = &coste
		}
		d.Analisis = &analisis
	}
	if d.Cobertura != nil {
		cobertura := *d.Cobertura
		cobertura.Comprobaciones = append(
			[]ComprobacionOperativaRRHH(nil),
			cobertura.Comprobaciones...,
		)
		d.Cobertura = &cobertura
	}
	if d.Asignacion != nil {
		asignacion := *d.Asignacion
		d.Asignacion = &asignacion
	}
	return d
}

func (DetalleExpedienteRRHH) String() string {
	return "[detalle-expediente-rrhh-redactado]"
}

func (DetalleExpedienteRRHH) GoString() string {
	return "[detalle-expediente-rrhh-redactado]"
}
