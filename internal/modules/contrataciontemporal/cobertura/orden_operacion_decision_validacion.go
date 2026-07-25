package cobertura

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func coordenadasOrdenOperacionDecisionCoberturaCoinciden(
	identidad DatosIdentidadOperacionDecisionCobertura,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
	solicitudGobierno SolicitudGobiernoOperacionCobertura,
	gobierno DatosGobiernoOperacionCobertura,
) bool {
	return identidad.Validar() == nil &&
		solicitudGobierno.validar() == nil &&
		solicitudGobierno.organizacionRef == identidad.organizacionRef &&
		solicitudGobierno.expedienteRef == identidad.expedienteRef &&
		solicitudGobierno.versionExpediente == identidad.versionExpediente &&
		solicitudGobierno.tipo == identidad.tipo &&
		solicitudGobierno.accion == identidad.accion &&
		reserva.AgregadoAnterior != nil &&
		reserva.AgregadoAnterior.OrganizacionRef == identidad.organizacionRef &&
		reserva.AgregadoAnterior.Referencia == identidad.expedienteRef &&
		reserva.AgregadoAnterior.Version == identidad.versionExpediente &&
		gobierno.Accion == identidad.accion
}

func propuestaExactaOperacionDecisionCobertura(
	preparacion PreparacionConjuntosViasCobertura,
	instante time.Time,
) (domain.PropuestaDecisionCobertura, error) {
	datos, err := preparacion.DatosCrearPropuestaEn(instante)
	if err != nil {
		return domain.PropuestaDecisionCobertura{}, err
	}
	return domain.CrearPropuestaDecisionCobertura(datos)
}

func propuestasOperacionDecisionCoberturaIguales(
	actual domain.PropuestaDecisionCobertura,
	esperada domain.PropuestaDecisionCobertura,
) bool {
	return referenciasOperacionDecisionCoberturaIguales(
		actual.Referencia(),
		esperada.Referencia(),
	) && referenciasOperacionDecisionCoberturaIguales(
		actual.HuellaSHA256(),
		esperada.HuellaSHA256(),
	) && reflect.DeepEqual(actual.Publicacion(), esperada.Publicacion())
}

func ventanaPreparacionOrdenOperacionDecisionCoberturaValida(
	instantePreparacion time.Time,
	generadaEn time.Time,
	preparadaC1En time.Time,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
	gobierno DatosGobiernoOperacionCobertura,
	validaHastaPropuesta time.Time,
	validaHastaC1 time.Time,
) bool {
	return instanteOperacionDecisionCoberturaValido(instantePreparacion) &&
		instanteOperacionDecisionCoberturaValido(generadaEn) &&
		instanteOperacionDecisionCoberturaValido(preparadaC1En) &&
		!preparadaC1En.Before(reserva.ObservadaEnDB) &&
		!generadaEn.Before(reserva.ObservadaEnDB) &&
		!generadaEn.Before(preparadaC1En) &&
		!gobierno.EvaluadaEn.Before(reserva.ObservadaEnDB) &&
		!instantePreparacion.Before(generadaEn) &&
		!instantePreparacion.Before(reserva.ObservadaEnDB) &&
		instantePreparacion.Before(reserva.PropiedadHasta) &&
		!instantePreparacion.Before(gobierno.EvaluadaEn) &&
		instantePreparacion.Before(gobierno.ValidaHasta) &&
		instantePreparacion.Before(validaHastaPropuesta) &&
		instantePreparacion.Before(validaHastaC1)
}

func gobiernoC1OperacionDecisionCoberturaCoincide(
	gobierno DatosGobiernoOperacionCobertura,
	propuesta domain.PublicacionPropuestaDecisionCobertura,
	identidad DatosIdentidadOperacionDecisionCobertura,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
) bool {
	return gobierno.Catalogo.Identidad() == propuesta.Catalogo &&
		gobierno.Politica.Identidad() == propuesta.Politica &&
		gobierno.PoliticaActuacion.Catalogo == propuesta.Catalogo &&
		gobierno.PoliticaActuacion.Politica == propuesta.Politica &&
		gobierno.FinalidadCTClave == propuesta.FinalidadClave &&
		gobierno.FinalidadCTRef == propuesta.FinalidadRef &&
		propuesta.OrganizacionRef == identidad.organizacionRef &&
		propuesta.ExpedienteRef == identidad.expedienteRef &&
		propuesta.VersionExpediente == identidad.versionExpediente &&
		referenciasOperacionDecisionCoberturaIguales(
			propuesta.AnalisisRef,
			reserva.AnalisisRef,
		) &&
		referenciasOperacionDecisionCoberturaIguales(
			propuesta.AnalisisHuellaSHA256,
			reserva.AnalisisHuellaSHA256,
		)
}

func motivoFuncionalOperacionDecisionCobertura(
	identidad DatosIdentidadOperacionDecisionCobertura,
	resolucion ResolucionMotivoDecisionCobertura,
	generadaEn time.Time,
	instanteLimite time.Time,
) (domain.MotivoGobernadoDecisionCobertura, error) {
	vacio := domain.MotivoGobernadoDecisionCobertura{}
	if identidad.motivo == vacio {
		if resolucion != (ResolucionMotivoDecisionCobertura{}) {
			return vacio, ErrOrdenOperacionDecisionCoberturaInvalida
		}
		return vacio, nil
	}
	motivo, err := resolucion.Motivo()
	resueltaEn, errInstante := resolucion.ResueltaEn()
	if err != nil || errInstante != nil || motivo != identidad.motivo ||
		resueltaEn.Before(generadaEn) ||
		resueltaEn.After(instanteLimite) {
		return vacio, ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return motivo, nil
}

func transicionPuraOperacionDecisionCobertura(
	identidad DatosIdentidadOperacionDecisionCobertura,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
	gobierno DatosGobiernoOperacionCobertura,
	propuesta domain.PropuestaDecisionCobertura,
	motivo domain.MotivoGobernadoDecisionCobertura,
	instante time.Time,
) (domain.Expediente, error) {
	if reserva.AgregadoAnterior == nil {
		return domain.Expediente{}, ErrOrdenOperacionDecisionCoberturaInvalida
	}
	actuacion := domain.DatosActuacion{
		AccionClave:   gobierno.Accion,
		ActorRef:      identidad.actorRef,
		UnidadRef:     gobierno.UnidadEjecutoraRef,
		ReciboRef:     reserva.ReciboRef,
		RealizadaEn:   instante,
		FaseDestino:   gobierno.FaseDestino,
		EstadoDestino: gobierno.EstadoDestino,
	}
	switch identidad.tipo {
	case domain.DecisionCoberturaInicial:
		return reserva.AgregadoAnterior.RegistrarDecisionCoberturaGobernada(
			identidad.versionExpediente,
			domain.DatosAdoptarDecisionCobertura{
				PerfilRef: identidad.perfilRef, ViaElegida: identidad.viaElegida,
				Motivo: motivo,
			},
			propuesta,
			actuacion,
		)
	case domain.DecisionCoberturaRectificacion:
		return reserva.AgregadoAnterior.RectificarDecisionCoberturaGobernada(
			identidad.versionExpediente,
			domain.DatosRectificarDecisionCobertura{
				PerfilRef: identidad.perfilRef, ViaElegida: identidad.viaElegida,
				Motivo: motivo, PredecesoraRef: identidad.predecesoraRef,
				PredecesoraHuellaSHA256: identidad.predecesoraHuella,
			},
			propuesta,
			actuacion,
		)
	default:
		return domain.Expediente{}, ErrOrdenOperacionDecisionCoberturaInvalida
	}
}

func recursoVECOperacionDecisionCobertura(
	identidad DatosIdentidadOperacionDecisionCobertura,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
	gobierno DatosGobiernoOperacionCobertura,
	semantica domain.IdentidadSemanticaPropuestaDecisionCobertura,
	propuesta domain.PublicacionPropuestaDecisionCobertura,
) (dominiovec.RecursoAutorizable, error) {
	atributos := map[string]string{
		"tipo_operacion": string(identidad.tipo), "expediente_ref": identidad.expedienteRef,
		"version_expediente_esperada": strconv.FormatUint(identidad.versionExpediente, 10),
		"accion":                      string(identidad.accion), "via_elegida": string(identidad.viaElegida),
		"propuesta_ref": propuesta.Referencia, "propuesta_huella_sha256": propuesta.HuellaSHA256,
		"propuesta_semantica_ref":              semantica.Referencia,
		"propuesta_semantica_huella_sha256":    semantica.HuellaSHA256,
		"preparacion_evidencias_ref":           propuesta.PreparacionEvidenciasRef,
		"preparacion_evidencias_huella_sha256": propuesta.PreparacionEvidenciasHuellaSHA256,
		"analisis_ref":                         propuesta.AnalisisRef, "analisis_huella_sha256": propuesta.AnalisisHuellaSHA256,
		"catalogo_ref":                     propuesta.Catalogo.Referencia,
		"catalogo_version":                 strconv.FormatUint(propuesta.Catalogo.Version, 10),
		"catalogo_huella_sha256":           propuesta.Catalogo.HuellaSHA256,
		"politica_ref":                     propuesta.Politica.Referencia,
		"politica_version":                 strconv.FormatUint(propuesta.Politica.Version, 10),
		"politica_huella_sha256":           propuesta.Politica.HuellaSHA256,
		"politica_actuacion_ref":           gobierno.PoliticaActuacion.Referencia,
		"politica_actuacion_version":       strconv.FormatUint(gobierno.PoliticaActuacion.Version, 10),
		"politica_actuacion_huella_sha256": gobierno.PoliticaActuacion.HuellaSHA256,
		"reserva_ref":                      reserva.ReservaRef,
		"revision_cercado":                 strconv.FormatUint(reserva.RevisionCercado, 10),
	}
	if identidad.predecesoraRef != "" {
		atributos["predecesora_ref"] = identidad.predecesoraRef
		atributos["predecesora_huella_sha256"] = identidad.predecesoraHuella
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: reserva.ReservaRef,
		ModuloID:   moduloRecursoOperacionDecisionCobertura,
		Tipo:       tipoRecursoOperacionDecisionCobertura,
		Ambitos: map[string]string{
			"organizacion_ref":     identidad.organizacionRef,
			"unidad_ejecutora_ref": gobierno.UnidadEjecutoraRef,
		},
		Atributos: atributos,
	}
	if recurso.Validar() != nil {
		return dominiovec.RecursoAutorizable{}, ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return recurso, nil
}

func validarCandidataVECOperacionDecisionCobertura(
	preparacion *datosPreparacionOrdenOperacionDecisionCobertura,
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	resumen puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3,
) error {
	if preparacion == nil {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	datosResumen, err := resumen.Datos()
	concedida, concesion, denegacion, errResultado := candidata.Resultado()
	var datosOrden puertosvec.DatosOrdenRegistroAutorizacionLigadaV3
	if errResultado == nil && concedida {
		datosOrden, errResultado = concesion.Datos()
	} else if errResultado == nil {
		datosOrden, errResultado = denegacion.Datos()
	}
	datosSolicitud, errSolicitud := datosOrden.Solicitud.Datos()
	datosVinculo, errVinculo := datosSolicitud.VinculoAutenticacionActor.Datos()
	correlacion, errCorrelacion := datosSolicitud.Correlacion.ValorCanonico()
	identidad, errIdentidad := preparacion.solicitudReserva.consulta.identidadInterna()
	if err != nil || errResultado != nil || errSolicitud != nil ||
		errVinculo != nil || errCorrelacion != nil || errIdentidad != nil ||
		datosResumen.Concedida != concedida ||
		datosResumen.EmitidaEn.Before(preparacion.preparadaEn) ||
		!datosResumen.EmitidaEn.Before(preparacion.validaHasta) ||
		!datosResumen.ValidaHasta.After(datosResumen.EmitidaEn) ||
		!referenciasOperacionDecisionCoberturaIguales(
			datosResumen.DecisionRef, preparacion.reserva.DecisionVECRef,
		) ||
		datosVinculo.PrincipalID != identidad.actorRef ||
		datosVinculo.PerfilActivoRef != identidad.perfilRef ||
		datosSolicitud.Accion != string(preparacion.datosGobierno.Accion) ||
		datosSolicitud.Finalidad != string(preparacion.datosGobierno.FinalidadVEC) ||
		datosSolicitud.ReferenciaMotivo != preparacion.datosGobierno.MotivoAutorizacion ||
		datosOrden.ReferenciaMotivo != preparacion.datosGobierno.MotivoAutorizacion ||
		!referenciasOperacionDecisionCoberturaIguales(
			correlacion, preparacion.reserva.CorrelacionVECRef,
		) ||
		!recursosOperacionDecisionCoberturaIguales(
			datosSolicitud.Recurso, preparacion.recursoVEC,
		) {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	limite := minimoInstanteOperacionDecisionCobertura(
		preparacion.validaHasta, datosResumen.ValidaHasta,
	)
	if !limite.After(datosResumen.EmitidaEn) {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return nil
}

func nuevaReservaMinimaOperacionDecisionCobertura(
	solicitud SolicitudReservarOperacionDecisionCobertura,
	reserva DatosReservaPropietariaOperacionDecisionCobertura,
) (reservaMinimaOperacionDecisionCobertura, error) {
	solicitudClonada, errClon :=
		clonarSolicitudReservaOperacionDecisionCobertura(solicitud)
	datos, err := solicitudClonada.Datos()
	if errClon != nil || err != nil || reserva.validarPara(solicitud) != nil {
		return reservaMinimaOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	minima := reservaMinimaOperacionDecisionCobertura{
		solicitud:               solicitudClonada,
		organizacionRef:         datos.OrganizacionRef,
		expedienteRef:           datos.ExpedienteRef,
		versionExpediente:       datos.VersionExpediente,
		ambitoHMAC:              reserva.AmbitoIdempotenciaHMAC,
		semanticaHMAC:           reserva.HuellaSemanticaHMAC,
		tokenSHA256:             reserva.TokenPropietarioSHA256,
		reservaRef:              reserva.ReservaRef,
		reciboRef:               reserva.ReciboRef,
		actuacionRef:            reserva.ActuacionRef,
		auditoriaRef:            reserva.AuditoriaRef,
		eventoRef:               reserva.EventoRef,
		correlacionVECRef:       reserva.CorrelacionVECRef,
		decisionVECRef:          reserva.DecisionVECRef,
		revisionCercadoAnterior: reserva.RevisionCercadoAnterior,
		revisionCercado:         reserva.RevisionCercado,
		observadaEnDB:           reserva.ObservadaEnDB,
		propiedadHasta:          reserva.PropiedadHasta,
	}
	minima.huellaSHA256, err =
		calcularHuellaReservaMinimaOperacionDecisionCobertura(minima)
	if minima.validar() != nil {
		return reservaMinimaOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return minima, nil
}

func (r reservaMinimaOperacionDecisionCobertura) validar() error {
	datos, err := r.solicitud.Datos()
	huella, errHuella := calcularHuellaReservaMinimaOperacionDecisionCobertura(r)
	referencias := []string{
		r.reservaRef, r.reciboRef, r.actuacionRef, r.auditoriaRef,
		r.eventoRef, r.correlacionVECRef, r.decisionVECRef,
	}
	if err != nil || errHuella != nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(r.huellaSHA256) ||
		!referenciasOperacionDecisionCoberturaIguales(
			huella,
			r.huellaSHA256,
		) ||
		datos.OrganizacionRef != r.organizacionRef ||
		datos.ExpedienteRef != r.expedienteRef ||
		datos.VersionExpediente != r.versionExpediente ||
		!r.solicitud.tokenCoincide(r.tokenSHA256) ||
		!r.solicitud.CoincideParPersistido(r.ambitoHMAC, r.semanticaHMAC) ||
		r.revisionCercadoAnterior >= MaximoEnteroSeguroOperacionDecisionCobertura ||
		r.revisionCercado != r.revisionCercadoAnterior+1 ||
		r.revisionCercado > MaximoEnteroSeguroOperacionDecisionCobertura ||
		!instanteOperacionDecisionCoberturaValido(r.observadaEnDB) ||
		!instanteOperacionDecisionCoberturaValido(r.propiedadHasta) ||
		!r.propiedadHasta.After(r.observadaEnDB) ||
		r.propiedadHasta.Sub(r.observadaEnDB) >
			MaximoLeaseOperacionDecisionCobertura {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	for _, referencia := range referencias {
		if !domain.ReferenciaOpacaValida(referencia) {
			return ErrOrdenOperacionDecisionCoberturaInvalida
		}
	}
	return nil
}

func calcularHuellaReservaMinimaOperacionDecisionCobertura(
	r reservaMinimaOperacionDecisionCobertura,
) (string, error) {
	canon := nuevoCanonOperacionDecisionCobertura()
	canon.texto("VEC-CT-RESERVA-MINIMA-DECISION-COBERTURA-C3-V1")
	canon.texto(r.organizacionRef)
	canon.texto(r.expedienteRef)
	canon.entero(r.versionExpediente)
	canon.texto(r.ambitoHMAC)
	canon.texto(r.semanticaHMAC)
	canon.texto(r.tokenSHA256)
	canon.texto(r.reservaRef)
	canon.texto(r.reciboRef)
	canon.texto(r.actuacionRef)
	canon.texto(r.auditoriaRef)
	canon.texto(r.eventoRef)
	canon.texto(r.correlacionVECRef)
	canon.texto(r.decisionVECRef)
	canon.entero(r.revisionCercadoAnterior)
	canon.entero(r.revisionCercado)
	canon.texto(r.observadaEnDB.Format(time.RFC3339Nano))
	canon.texto(r.propiedadHasta.Format(time.RFC3339Nano))
	material, err := canon.resultado()
	if err != nil {
		return "", ErrOrdenOperacionDecisionCoberturaInvalida
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func resumenCorrelacionCandidataOperacionDecisionCobertura(
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
) string {
	concedida, concesion, denegacion, err := candidata.Resultado()
	var datos puertosvec.DatosOrdenRegistroAutorizacionLigadaV3
	if err == nil && concedida {
		datos, err = concesion.Datos()
	} else if err == nil {
		datos, err = denegacion.Datos()
	}
	if err != nil {
		return ""
	}
	solicitud, err := datos.Solicitud.Datos()
	if err != nil {
		return ""
	}
	correlacion, _ := solicitud.Correlacion.ValorCanonico()
	return correlacion
}

func recursosOperacionDecisionCoberturaIguales(
	actual dominiovec.RecursoAutorizable,
	esperado dominiovec.RecursoAutorizable,
) bool {
	return actual.Validar() == nil && esperado.Validar() == nil &&
		referenciasOperacionDecisionCoberturaIguales(actual.Referencia, esperado.Referencia) &&
		actual.ModuloID == esperado.ModuloID &&
		actual.Tipo == esperado.Tipo &&
		reflect.DeepEqual(actual.Ambitos, esperado.Ambitos) &&
		reflect.DeepEqual(actual.Atributos, esperado.Atributos)
}

func clonarDatosPreparacionOrdenOperacionDecisionCobertura(
	origen *datosPreparacionOrdenOperacionDecisionCobertura,
) *datosPreparacionOrdenOperacionDecisionCobertura {
	if origen == nil {
		return nil
	}
	clon := *origen
	clon.reserva = clonarReservaPropietariaOperacionDecisionCobertura(origen.reserva)
	clon.recursoVEC = clonarRecursoOperacionDecisionCobertura(origen.recursoVEC)
	clon.preparacionC1.conjuntos = make(
		[]ConjuntoEvidenciasCobertura,
		len(origen.preparacionC1.conjuntos),
	)
	for indice := range origen.preparacionC1.conjuntos {
		conjunto := origen.preparacionC1.conjuntos[indice]
		conjunto.evidencias = append(
			[]EvidenciaConsultaCobertura(nil),
			conjunto.evidencias...,
		)
		clon.preparacionC1.conjuntos[indice] = conjunto
	}
	return &clon
}

func clonarRecursoOperacionDecisionCobertura(
	recurso dominiovec.RecursoAutorizable,
) dominiovec.RecursoAutorizable {
	clon := recurso
	clon.Ambitos = clonarMapaOperacionDecisionCobertura(recurso.Ambitos)
	clon.Atributos = clonarMapaOperacionDecisionCobertura(recurso.Atributos)
	return clon
}

func clonarMapaOperacionDecisionCobertura(origen map[string]string) map[string]string {
	if origen == nil {
		return nil
	}
	clon := make(map[string]string, len(origen))
	for clave, valor := range origen {
		clon[clave] = valor
	}
	return clon
}

func minimoInstanteOperacionDecisionCobertura(instantes ...time.Time) time.Time {
	if len(instantes) == 0 {
		return time.Time{}
	}
	minimo := instantes[0]
	for _, instante := range instantes[1:] {
		if instante.Before(minimo) {
			minimo = instante
		}
	}
	return minimo
}
