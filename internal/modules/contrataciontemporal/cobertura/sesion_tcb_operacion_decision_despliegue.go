package cobertura

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func desplegarOrdenOperacionDecisionCoberturaEnSesionTCB(
	ctx context.Context,
	orden OrdenOperacionDecisionCobertura,
	destino SesionTCBOperacionDecisionCobertura,
	guardia *guardiaCicloSesionTCBOperacionDecisionCobertura,
) (ReciboOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(destino) ||
		guardia == nil ||
		orden.validar() != nil {
		return ReciboOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ReciboOperacionDecisionCobertura{}, err
	}
	cabecera, err := nuevaCabeceraSesionTCBOperacionDecisionCobertura(orden)
	if err != nil {
		return ReciboOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	sesion := nuevaSesionControladaOperacionDecisionCobertura(
		destino,
		guardia,
	)
	if err = sesion.aplicarApertura(cabecera); err != nil {
		return ReciboOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}

	if orden.datos.concesion != nil {
		datos := orden.datos.concesion
		gobierno, errGobierno :=
			nuevoGobiernoSesionTCBOperacionDecisionCobertura(datos)
		decisionVEC, errVEC :=
			nuevaDecisionVECSesionTCBOperacionDecisionCobertura(
				datos.candidata,
				datos.resumen,
			)
		ordenes, errOrdenes := datos.preparacion.preparacionC1.
			OrdenesPendientesEn(datos.efectoEn)
		if errGobierno != nil || errVEC != nil || errOrdenes != nil ||
			len(ordenes) == 0 ||
			uint64(len(ordenes)) >
				MaximoConsumosC1SesionTCBOperacionDecisionCobertura {
			return ReciboOperacionDecisionCobertura{},
				ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		if err = sesion.aplicarGobierno(gobierno); err != nil ||
			sesion.aplicarDecisionVEC(decisionVEC) != nil {
			return ReciboOperacionDecisionCobertura{},
				ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		total := uint64(len(ordenes))
		for indice := range ordenes {
			consumo, errConsumo :=
				nuevoConsumoC1SesionTCBOperacionDecisionCobertura(
					uint64(indice)+1,
					total,
					ordenes[indice],
					datos.efectoEn,
				)
			if errConsumo != nil || sesion.aplicarConsumoC1(consumo) != nil {
				return ReciboOperacionDecisionCobertura{},
					ErrSesionTCBOperacionDecisionCoberturaInvalida
			}
		}
		efecto, errEfecto :=
			nuevoEfectoConcedidoSesionTCBOperacionDecisionCobertura(
				datos,
			)
		if errEfecto != nil || sesion.aplicarConcesion(efecto) != nil {
			return ReciboOperacionDecisionCobertura{},
				ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
	} else {
		datos := orden.datos.denegacion
		decisionVEC, errVEC :=
			nuevaDecisionVECSesionTCBOperacionDecisionCobertura(
				datos.candidata,
				datos.resumen,
			)
		terminal, errTerminal :=
			nuevoTerminalDenegadoSesionTCBOperacionDecisionCobertura(
				datos,
			)
		if errVEC != nil || errTerminal != nil ||
			sesion.aplicarDecisionVEC(decisionVEC) != nil ||
			sesion.aplicarDenegacion(terminal) != nil {
			return ReciboOperacionDecisionCobertura{},
				ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
	}

	crudo, err := sesion.aplicarConfirmacion(ctx)
	if err != nil {
		return ReciboOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	recibo, err := crudo.recibo()
	if err != nil ||
		validarReciboParaOrdenOperacionDecisionCobertura(orden, recibo) != nil {
		return ReciboOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return recibo, nil
}

func nuevaCabeceraSesionTCBOperacionDecisionCobertura(
	orden OrdenOperacionDecisionCobertura,
) (CabeceraSesionTCBOperacionDecisionCobertura, error) {
	huellaOrden, err := huellaConfirmacionOperacionDecisionCobertura(orden)
	if err != nil {
		return CabeceraSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	datos := DatosCabeceraSesionTCBOperacionDecisionCobertura{
		Esquema:           esquemaSesionTCBOperacionDecisionCobertura,
		HuellaOrdenSHA256: huellaOrden,
	}
	if orden.datos.concesion != nil {
		concesion := orden.datos.concesion
		preparacion := concesion.preparacion
		solicitud, errSolicitud := preparacion.solicitudReserva.Datos()
		referenciaC1, errReferencia :=
			preparacion.preparacionC1.Referencia()
		huellaC1, errHuella := preparacion.preparacionC1.HuellaSHA256()
		validaC1, errValida := preparacion.preparacionC1.ValidaHasta()
		numeroC1, huellaOrdenesC1, errOrdenes :=
			identidadOrdenesC1ConfirmacionOperacionDecisionCobertura(
				preparacion.preparacionC1,
				concesion.efectoEn,
			)
		if errSolicitud != nil || errReferencia != nil || errHuella != nil ||
			errValida != nil || errOrdenes != nil {
			return CabeceraSesionTCBOperacionDecisionCobertura{},
				ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		reserva := preparacion.reserva
		datos.Rama = RamaSesionTCBOperacionDecisionCoberturaConcedida
		datos.OrganizacionRef = solicitud.OrganizacionRef
		datos.ExpedienteRef = solicitud.ExpedienteRef
		datos.VersionExpediente = solicitud.VersionExpediente
		datos.ReservaRef = reserva.ReservaRef
		datos.ReciboRef = reserva.ReciboRef
		datos.ActuacionRef = reserva.ActuacionRef
		datos.AuditoriaRef = reserva.AuditoriaRef
		datos.EventoRef = reserva.EventoRef
		datos.CorrelacionVECRef = reserva.CorrelacionVECRef
		datos.DecisionVECRef = reserva.DecisionVECRef
		datos.AnalisisRef = reserva.AnalisisRef
		datos.AnalisisHuellaSHA256 = reserva.AnalisisHuellaSHA256
		datos.TokenPropietarioSHA256 = solicitud.TokenPropietarioSHA256
		datos.AmbitoIdempotenciaHMAC = reserva.AmbitoIdempotenciaHMAC
		datos.HuellaSemanticaHMAC = reserva.HuellaSemanticaHMAC
		datos.RevisionCercadoAnterior = reserva.RevisionCercadoAnterior
		datos.RevisionCercado = reserva.RevisionCercado
		datos.ObservadaEnDB = reserva.ObservadaEnDB
		datos.PropiedadHasta = reserva.PropiedadHasta
		datos.ValidaHastaOrden = concesion.validaHasta
		datos.PreparacionC1Ref = referenciaC1
		datos.PreparacionC1HuellaSHA256 = huellaC1
		datos.PreparacionC1PreparadaEn = preparacion.preparacionC1.preparadaEn
		datos.PreparacionC1ValidaHasta = validaC1
		datos.NumeroConsumosC1 = numeroC1
		datos.HuellaOrdenesConsumoC1SHA256 = huellaOrdenesC1
	} else {
		denegacion := orden.datos.denegacion
		reserva := denegacion.prueba.reserva
		datos.Rama = RamaSesionTCBOperacionDecisionCoberturaDenegada
		datos.OrganizacionRef = reserva.organizacionRef
		datos.ExpedienteRef = reserva.expedienteRef
		datos.VersionExpediente = reserva.versionExpediente
		datos.ReservaRef = reserva.reservaRef
		datos.ReciboRef = reserva.reciboRef
		datos.ActuacionRef = reserva.actuacionRef
		datos.AuditoriaRef = reserva.auditoriaRef
		datos.EventoRef = reserva.eventoRef
		datos.CorrelacionVECRef = reserva.correlacionVECRef
		datos.DecisionVECRef = reserva.decisionVECRef
		solicitud, _ := reserva.solicitud.Datos()
		datos.TokenPropietarioSHA256 = solicitud.TokenPropietarioSHA256
		datos.AmbitoIdempotenciaHMAC = reserva.ambitoHMAC
		datos.HuellaSemanticaHMAC = reserva.semanticaHMAC
		datos.RevisionCercadoAnterior = reserva.revisionCercadoAnterior
		datos.RevisionCercado = reserva.revisionCercado
		datos.ObservadaEnDB = reserva.observadaEnDB
		datos.PropiedadHasta = reserva.propiedadHasta
		datos.ValidaHastaOrden = denegacion.validaHasta
	}
	cabecera := CabeceraSesionTCBOperacionDecisionCobertura{
		datos: &datosCabeceraSesionTCBOperacionDecisionCobertura{
			DatosCabeceraSesionTCBOperacionDecisionCobertura: datos,
		},
	}
	if cabecera.validar() != nil {
		return CabeceraSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return cabecera, nil
}

func nuevoGobiernoSesionTCBOperacionDecisionCobertura(
	orden *datosOrdenConcedidaOperacionDecisionCobertura,
) (GobiernoSesionTCBOperacionDecisionCobertura, error) {
	if orden == nil || orden.preparacion == nil {
		return GobiernoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	d := orden.preparacion.datosGobierno
	gobierno := GobiernoSesionTCBOperacionDecisionCobertura{
		datos: &datosGobiernoSesionTCBOperacionDecisionCobertura{
			catalogo: d.Catalogo, politica: d.Politica,
			politicaActuacion: d.PoliticaActuacion,
			accion:            d.Accion, finalidadCTClave: d.FinalidadCTClave,
			finalidadCTRef: d.FinalidadCTRef, finalidadVEC: d.FinalidadVEC,
			unidadEjecutoraRef: d.UnidadEjecutoraRef,
			faseDestino:        d.FaseDestino, estadoDestino: d.EstadoDestino,
			motivoAutorizacion: d.MotivoAutorizacion,
			evaluadaEn:         d.EvaluadaEn, validaHasta: d.ValidaHasta,
		},
	}
	if gobierno.validar() != nil {
		return GobiernoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return gobierno, nil
}

func nuevaDecisionVECSesionTCBOperacionDecisionCobertura(
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	resumen puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3,
) (DecisionVECSesionTCBOperacionDecisionCobertura, error) {
	decision := DecisionVECSesionTCBOperacionDecisionCobertura{
		datos: &datosDecisionVECSesionTCBOperacionDecisionCobertura{
			candidata: candidata,
			resumen:   resumen,
		},
	}
	if decision.validar() != nil {
		return DecisionVECSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return decision, nil
}

func nuevoConsumoC1SesionTCBOperacionDecisionCobertura(
	posicion uint64,
	total uint64,
	orden puertosct.OrdenConsumoCobertura,
	instante time.Time,
) (ConsumoC1SesionTCBOperacionDecisionCobertura, error) {
	consumo := ConsumoC1SesionTCBOperacionDecisionCobertura{
		datos: &datosConsumoC1SesionTCBOperacionDecisionCobertura{
			posicion: posicion,
			total:    total,
			orden:    orden,
			instante: instante,
		},
	}
	if consumo.validar() != nil {
		return ConsumoC1SesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return consumo, nil
}

func nuevoEfectoConcedidoSesionTCBOperacionDecisionCobertura(
	orden *datosOrdenConcedidaOperacionDecisionCobertura,
) (EfectoConcedidoSesionTCBOperacionDecisionCobertura, error) {
	if orden == nil || orden.preparacion == nil {
		return EfectoConcedidoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	identidad, err :=
		orden.preparacion.solicitudReserva.consulta.identidadInterna()
	motivo, errMotivo := motivoFuncionalOperacionDecisionCobertura(
		identidad,
		orden.preparacion.motivo,
		orden.preparacion.propuesta.Publicacion().GeneradaEn,
		orden.efectoEn,
	)
	if err != nil || errMotivo != nil ||
		orden.preparacion.reserva.AgregadoAnterior == nil {
		return EfectoConcedidoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	efecto := EfectoConcedidoSesionTCBOperacionDecisionCobertura{
		datos: &datosEfectoConcedidoSesionTCBOperacionDecisionCobertura{
			agregadoAnterior:  orden.preparacion.reserva.AgregadoAnterior.Clonar(),
			agregadoSiguiente: orden.agregadoSiguiente.Clonar(),
			propuesta:         orden.preparacion.propuesta,
			motivoFuncional:   motivo,
			efectoEn:          orden.efectoEn,
			validaHasta:       orden.validaHasta,
		},
	}
	if efecto.validar() != nil {
		return EfectoConcedidoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return efecto, nil
}

func nuevoTerminalDenegadoSesionTCBOperacionDecisionCobertura(
	orden *datosOrdenDenegadaOperacionDecisionCobertura,
) (TerminalDenegadoSesionTCBOperacionDecisionCobertura, error) {
	if orden == nil {
		return TerminalDenegadoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	terminal := TerminalDenegadoSesionTCBOperacionDecisionCobertura{
		datos: &datosTerminalDenegadoSesionTCBOperacionDecisionCobertura{
			prueba:      orden.prueba,
			validaHasta: orden.validaHasta,
		},
	}
	if terminal.validar() != nil {
		return TerminalDenegadoSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return terminal, nil
}

func (c CabeceraSesionTCBOperacionDecisionCobertura) validar() error {
	if c.datos == nil {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	d := c.datos.DatosCabeceraSesionTCBOperacionDecisionCobertura
	referencias := []string{
		d.OrganizacionRef, d.ExpedienteRef, d.ReservaRef, d.ReciboRef,
		d.ActuacionRef, d.AuditoriaRef, d.EventoRef, d.CorrelacionVECRef,
		d.DecisionVECRef,
	}
	if d.Esquema != esquemaSesionTCBOperacionDecisionCobertura ||
		!d.Rama.valida() ||
		!huellaSHA256OperacionDecisionCoberturaValida(d.HuellaOrdenSHA256) ||
		d.VersionExpediente == 0 ||
		d.VersionExpediente >= MaximoEnteroSeguroOperacionDecisionCobertura ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			d.TokenPropietarioSHA256,
		) ||
		!puertosct.SelloHMACSHA256Valido(d.AmbitoIdempotenciaHMAC) ||
		!puertosct.SelloHMACSHA256Valido(d.HuellaSemanticaHMAC) ||
		d.RevisionCercadoAnterior >=
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		d.RevisionCercado != d.RevisionCercadoAnterior+1 ||
		d.RevisionCercado > MaximoEnteroSeguroOperacionDecisionCobertura ||
		!instanteOperacionDecisionCoberturaValido(d.ObservadaEnDB) ||
		!instanteOperacionDecisionCoberturaValido(d.PropiedadHasta) ||
		!instanteOperacionDecisionCoberturaValido(d.ValidaHastaOrden) ||
		!d.PropiedadHasta.After(d.ObservadaEnDB) ||
		!d.ValidaHastaOrden.After(d.ObservadaEnDB) {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	for _, referencia := range referencias {
		if !domain.ReferenciaOpacaValida(referencia) {
			return ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
	}
	if d.Rama == RamaSesionTCBOperacionDecisionCoberturaConcedida {
		if !domain.ReferenciaOpacaValida(d.AnalisisRef) ||
			!huellaSHA256OperacionDecisionCoberturaValida(
				d.AnalisisHuellaSHA256,
			) ||
			!domain.ReferenciaOpacaValida(d.PreparacionC1Ref) ||
			!huellaSHA256OperacionDecisionCoberturaValida(
				d.PreparacionC1HuellaSHA256,
			) ||
			!instanteOperacionDecisionCoberturaValido(
				d.PreparacionC1PreparadaEn,
			) ||
			!instanteOperacionDecisionCoberturaValido(
				d.PreparacionC1ValidaHasta,
			) ||
			!d.PreparacionC1ValidaHasta.After(
				d.PreparacionC1PreparadaEn,
			) ||
			d.NumeroConsumosC1 == 0 ||
			d.NumeroConsumosC1 >
				MaximoConsumosC1SesionTCBOperacionDecisionCobertura ||
			!huellaSHA256OperacionDecisionCoberturaValida(
				d.HuellaOrdenesConsumoC1SHA256,
			) {
			return ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		return nil
	}
	if d.PreparacionC1Ref != "" || d.PreparacionC1HuellaSHA256 != "" ||
		d.AnalisisRef != "" || d.AnalisisHuellaSHA256 != "" ||
		!d.PreparacionC1PreparadaEn.IsZero() ||
		!d.PreparacionC1ValidaHasta.IsZero() ||
		d.NumeroConsumosC1 != 0 ||
		d.HuellaOrdenesConsumoC1SHA256 != "" {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return nil
}

func (g GobiernoSesionTCBOperacionDecisionCobertura) validar() error {
	if g.datos == nil || g.datos.catalogo.Validar() != nil ||
		g.datos.politica.ValidarPara(
			g.datos.catalogo,
			g.datos.politicaActuacion.OrganizacionRef,
			g.datos.finalidadCTClave,
			g.datos.finalidadCTRef,
			g.datos.evaluadaEn,
		) != nil ||
		g.datos.politicaActuacion.Validar() != nil ||
		g.datos.politicaActuacion.Catalogo !=
			g.datos.catalogo.Identidad() ||
		g.datos.politicaActuacion.Politica !=
			g.datos.politica.Identidad() ||
		g.datos.politicaActuacion.Accion != g.datos.accion ||
		g.datos.politicaActuacion.FinalidadContratacionClave !=
			g.datos.finalidadCTClave ||
		g.datos.politicaActuacion.FinalidadContratacionRef !=
			g.datos.finalidadCTRef ||
		g.datos.politicaActuacion.FinalidadAutorizacionVEC !=
			g.datos.finalidadVEC ||
		g.datos.politicaActuacion.UnidadEjecutoraRef !=
			g.datos.unidadEjecutoraRef ||
		g.datos.politicaActuacion.FaseDestino != g.datos.faseDestino ||
		g.datos.politicaActuacion.EstadoDestino != g.datos.estadoDestino ||
		!instanteOperacionDecisionCoberturaValido(g.datos.evaluadaEn) ||
		!instanteOperacionDecisionCoberturaValido(g.datos.validaHasta) ||
		!g.datos.validaHasta.After(g.datos.evaluadaEn) {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return nil
}

func datosDecisionVECSesionTCB(
	d DecisionVECSesionTCBOperacionDecisionCobertura,
) (
	bool,
	puertosvec.DatosOrdenRegistroAutorizacionLigadaV3,
	puertosvec.DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	if d.datos == nil ||
		d.datos.resumen.ValidarPara(d.datos.candidata) != nil {
		return false, puertosvec.DatosOrdenRegistroAutorizacionLigadaV3{},
			puertosvec.DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	concedida, concesion, denegacion, err := d.datos.candidata.Resultado()
	var orden puertosvec.DatosOrdenRegistroAutorizacionLigadaV3
	if err == nil && concedida {
		orden, err = concesion.Datos()
	} else if err == nil {
		orden, err = denegacion.Datos()
	}
	resumen, errResumen := d.datos.resumen.Datos()
	if err != nil || errResumen != nil || resumen.Concedida != concedida {
		return false, puertosvec.DatosOrdenRegistroAutorizacionLigadaV3{},
			puertosvec.DatosResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return concedida, orden, resumen, nil
}

func (d DecisionVECSesionTCBOperacionDecisionCobertura) validar() error {
	_, _, _, err := datosDecisionVECSesionTCB(d)
	return err
}

func (c ConsumoC1SesionTCBOperacionDecisionCobertura) validar() error {
	if c.datos == nil || c.datos.posicion == 0 ||
		c.datos.posicion > c.datos.total ||
		c.datos.total == 0 ||
		c.datos.total >
			MaximoConsumosC1SesionTCBOperacionDecisionCobertura ||
		!instanteOperacionDecisionCoberturaValido(c.datos.instante) {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	orden, errOrden := c.datos.orden.Datos()
	resumen, errResumen :=
		c.datos.orden.ResumenPendienteEn(c.datos.instante)
	if errOrden != nil || errResumen != nil ||
		orden.PeticionRef != resumen.PeticionRef ||
		orden.OrganizacionRef != resumen.OrganizacionRef ||
		orden.ExpedienteRef != resumen.ExpedienteRef ||
		orden.VersionExpediente != resumen.VersionExpediente ||
		orden.HuellaPeticionSHA256 != resumen.HuellaPeticionSHA256 ||
		orden.HuellaResultadoSHA256 != resumen.HuellaResultadoSHA256 ||
		orden.AutoridadRef != resumen.AutoridadRef ||
		orden.Generacion != resumen.Generacion ||
		orden.ReciboRespuestaRef != resumen.ReciboRespuestaRef ||
		orden.HuellaRespuestaSHA256 != resumen.HuellaRespuestaSHA256 {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return nil
}

func (e EfectoConcedidoSesionTCBOperacionDecisionCobertura) validar() error {
	if e.datos == nil ||
		e.datos.agregadoAnterior.Validar() != nil ||
		e.datos.agregadoSiguiente.Validar() != nil ||
		e.datos.agregadoAnterior.OrganizacionRef !=
			e.datos.agregadoSiguiente.OrganizacionRef ||
		e.datos.agregadoAnterior.Referencia !=
			e.datos.agregadoSiguiente.Referencia ||
		e.datos.agregadoAnterior.Version == 0 ||
		e.datos.agregadoAnterior.Version >=
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		e.datos.agregadoSiguiente.Version !=
			e.datos.agregadoAnterior.Version+1 ||
		e.datos.agregadoSiguiente.Version >
			MaximoEnteroSeguroOperacionDecisionCobertura ||
		len(e.datos.agregadoSiguiente.DecisionesCobertura) == 0 ||
		!instanteOperacionDecisionCoberturaValido(e.datos.efectoEn) ||
		!instanteOperacionDecisionCoberturaValido(e.datos.validaHasta) ||
		!e.datos.validaHasta.After(e.datos.efectoEn) {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	publicacion := e.datos.propuesta.Publicacion()
	if publicacion.OrganizacionRef !=
		e.datos.agregadoAnterior.OrganizacionRef ||
		publicacion.ExpedienteRef != e.datos.agregadoAnterior.Referencia ||
		publicacion.VersionExpediente != e.datos.agregadoAnterior.Version {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return nil
}

func (d TerminalDenegadoSesionTCBOperacionDecisionCobertura) validar() error {
	if d.datos == nil || d.datos.prueba.validar() != nil ||
		!instanteOperacionDecisionCoberturaValido(d.datos.validaHasta) ||
		!d.datos.validaHasta.After(d.datos.prueba.reserva.observadaEnDB) {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return nil
}

func (d DatosReciboSesionTCBOperacionDecisionCobertura) recibo() (
	ReciboOperacionDecisionCobertura,
	error,
) {
	if d.Aplicada == d.DenegadaVEC {
		return ReciboOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	recibo := ReciboOperacionDecisionCobertura{
		ReciboRef: d.ReciboRef, ReservaRef: d.ReservaRef,
		AuditoriaRef:            d.AuditoriaRef,
		CorrelacionVECRef:       d.CorrelacionVECRef,
		DecisionVECRef:          d.DecisionVECRef,
		DecisionVECHuellaSHA256: d.DecisionVECHuellaSHA256,
		CodigoProbatorioVEC:     d.CodigoProbatorioVEC,
		ConcedidaVEC:            d.ConcedidaVEC,
		RevisionCercado:         d.RevisionCercado,
		AmbitoIdempotenciaHMAC:  d.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     d.HuellaSemanticaHMAC,
		ConfirmadaEn:            d.ConfirmadaEn,
	}
	if d.Aplicada {
		recibo.Aplicada = &ResultadoAplicadoOperacionDecisionCobertura{
			DecisionCoberturaRef:    d.DecisionCoberturaRef,
			DecisionCoberturaHuella: d.DecisionCoberturaHuella,
			VersionResultante:       d.VersionResultante,
			EventoRef:               d.EventoRef,
			ActuacionRef:            d.ActuacionRef,
		}
	} else {
		if d.DecisionCoberturaRef != "" ||
			d.DecisionCoberturaHuella != "" ||
			d.VersionResultante != 0 ||
			d.EventoRef != "" ||
			d.ActuacionRef != "" {
			return ReciboOperacionDecisionCobertura{},
				ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		recibo.DenegadaVEC =
			&ResultadoDenegadoVECOperacionDecisionCobertura{}
	}
	return recibo, nil
}
