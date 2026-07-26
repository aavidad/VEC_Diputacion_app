package cobertura

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const esquemaPruebaDenegacionOperacionDecisionCobertura = "" +
	"VEC-CT-PRUEBA-DENEGACION-DECISION-COBERTURA-C3-V1"

func nuevaPruebaDenegacionOperacionDecisionCobertura(
	preparacion *datosPreparacionOrdenOperacionDecisionCobertura,
) (pruebaDenegacionOperacionDecisionCobertura, error) {
	if preparacion == nil ||
		(PreparacionOrdenOperacionDecisionCobertura{
			datos: preparacion,
		}).validar() != nil {
		return pruebaDenegacionOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	identidad, errIdentidad :=
		preparacion.solicitudReserva.consulta.identidadInterna()
	reserva, errReserva := nuevaReservaMinimaOperacionDecisionCobertura(
		preparacion.solicitudReserva,
		preparacion.reserva,
	)
	recurso := clonarRecursoOperacionDecisionCobertura(
		preparacion.recursoVEC,
	)
	if errIdentidad != nil || errReserva != nil {
		return pruebaDenegacionOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	prueba := pruebaDenegacionOperacionDecisionCobertura{
		reserva: reserva, recursoVEC: recurso,
		actorRef: identidad.actorRef, perfilRef: identidad.perfilRef,
		accionVEC:         preparacion.datosGobierno.Accion,
		finalidadVEC:      preparacion.datosGobierno.FinalidadVEC,
		motivoVEC:         preparacion.datosGobierno.MotivoAutorizacion,
		limitePreparacion: preparacion.validaHasta,
	}
	prueba.huellaSHA256, _ =
		calcularHuellaPruebaDenegacionOperacionDecisionCobertura(prueba)
	if prueba.validar() != nil {
		return pruebaDenegacionOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return prueba, nil
}

func (p pruebaDenegacionOperacionDecisionCobertura) validar() error {
	identidad, errIdentidad := p.reserva.solicitud.consulta.identidadInterna()
	huella, errHuella :=
		calcularHuellaPruebaDenegacionOperacionDecisionCobertura(p)
	if p.reserva.validar() != nil || errIdentidad != nil ||
		errHuella != nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(p.huellaSHA256) ||
		!referenciasOperacionDecisionCoberturaIguales(
			huella,
			p.huellaSHA256,
		) ||
		p.recursoVEC.Validar() != nil ||
		!domain.ReferenciaOpacaValida(p.actorRef) ||
		!domain.ReferenciaOpacaValida(p.perfilRef) ||
		p.actorRef != identidad.actorRef ||
		p.perfilRef != identidad.perfilRef ||
		p.accionVEC != identidad.accion ||
		!p.finalidadVEC.Valida() ||
		!dominiovec.ReferenciaMotivoAutorizacionV2Valida(p.motivoVEC) ||
		!instanteOperacionDecisionCoberturaValido(p.limitePreparacion) ||
		!p.limitePreparacion.After(p.reserva.observadaEnDB) ||
		p.limitePreparacion.After(p.reserva.propiedadHasta) ||
		!recursoPruebaDenegacionOperacionDecisionCoberturaCoherente(
			p.recursoVEC,
			identidad,
			p.reserva,
		) {
		return ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return nil
}

func calcularHuellaPruebaDenegacionOperacionDecisionCobertura(
	p pruebaDenegacionOperacionDecisionCobertura,
) (string, error) {
	canon := nuevoCanonOperacionDecisionCobertura()
	canon.texto(esquemaPruebaDenegacionOperacionDecisionCobertura)
	escribirReservaMinimaPruebaDenegacionOperacionDecisionCobertura(
		canon,
		p.reserva,
	)
	escribirRecursoPruebaDenegacionOperacionDecisionCobertura(
		canon,
		p.recursoVEC,
	)
	canon.texto(p.actorRef)
	canon.texto(p.perfilRef)
	canon.texto(string(p.accionVEC))
	canon.texto(string(p.finalidadVEC))
	escribirReferenciaMotivoPruebaDenegacionOperacionDecisionCobertura(
		canon,
		p.motivoVEC,
	)
	canon.texto(p.limitePreparacion.Format(time.RFC3339Nano))
	material, err := canon.resultado()
	if err != nil {
		return "", ErrOrdenOperacionDecisionCoberturaInvalida
	}
	huella := sha256.Sum256(material)
	return hex.EncodeToString(huella[:]), nil
}

func escribirReservaMinimaPruebaDenegacionOperacionDecisionCobertura(
	canon *canonOperacionDecisionCobertura,
	reserva reservaMinimaOperacionDecisionCobertura,
) {
	canon.texto(reserva.organizacionRef)
	canon.texto(reserva.expedienteRef)
	canon.entero(reserva.versionExpediente)
	canon.texto(reserva.ambitoHMAC)
	canon.texto(reserva.semanticaHMAC)
	canon.texto(reserva.tokenSHA256)
	canon.texto(reserva.reservaRef)
	canon.texto(reserva.reciboRef)
	canon.texto(reserva.actuacionRef)
	canon.texto(reserva.auditoriaRef)
	canon.texto(reserva.eventoRef)
	canon.texto(reserva.correlacionVECRef)
	canon.texto(reserva.decisionVECRef)
	canon.entero(reserva.revisionCercadoAnterior)
	canon.entero(reserva.revisionCercado)
	canon.texto(reserva.observadaEnDB.Format(time.RFC3339Nano))
	canon.texto(reserva.propiedadHasta.Format(time.RFC3339Nano))
	canon.texto(reserva.huellaSHA256)
}

func escribirRecursoPruebaDenegacionOperacionDecisionCobertura(
	canon *canonOperacionDecisionCobertura,
	recurso dominiovec.RecursoAutorizable,
) {
	canon.texto(recurso.Referencia)
	canon.texto(recurso.ModuloID)
	canon.texto(recurso.Tipo)
	escribirMapaPruebaDenegacionOperacionDecisionCobertura(
		canon,
		recurso.Ambitos,
	)
	escribirMapaPruebaDenegacionOperacionDecisionCobertura(
		canon,
		recurso.Atributos,
	)
}

func escribirMapaPruebaDenegacionOperacionDecisionCobertura(
	canon *canonOperacionDecisionCobertura,
	valores map[string]string,
) {
	claves := make([]string, 0, len(valores))
	for clave := range valores {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	canon.entero(uint64(len(claves)))
	for _, clave := range claves {
		canon.texto(clave)
		canon.texto(valores[clave])
	}
}

func escribirReferenciaMotivoPruebaDenegacionOperacionDecisionCobertura(
	canon *canonOperacionDecisionCobertura,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) {
	canon.texto(motivo.CatalogoID)
	canon.entero(uint64(motivo.CatalogoVersion))
	canon.texto(motivo.CatalogoHuellaSHA256)
	canon.texto(motivo.EntradaClave)
}

func recursoPruebaDenegacionOperacionDecisionCoberturaCoherente(
	recurso dominiovec.RecursoAutorizable,
	identidad DatosIdentidadOperacionDecisionCobertura,
	reserva reservaMinimaOperacionDecisionCobertura,
) bool {
	semantica := identidad.identidadSemantica
	atributos := recurso.Atributos
	ambitos := recurso.Ambitos
	if len(ambitos) != 2 ||
		recurso.ModuloID != moduloRecursoOperacionDecisionCobertura ||
		recurso.Tipo != tipoRecursoOperacionDecisionCobertura ||
		!referenciasOperacionDecisionCoberturaIguales(
			recurso.Referencia,
			reserva.reservaRef,
		) ||
		ambitos["organizacion_ref"] != identidad.organizacionRef ||
		!domain.ReferenciaOpacaValida(ambitos["unidad_ejecutora_ref"]) ||
		atributos["tipo_operacion"] != string(identidad.tipo) ||
		atributos["expediente_ref"] != identidad.expedienteRef ||
		atributos["version_expediente_esperada"] !=
			strconv.FormatUint(identidad.versionExpediente, 10) ||
		atributos["accion"] != string(identidad.accion) ||
		atributos["via_elegida"] != string(identidad.viaElegida) ||
		atributos["propuesta_semantica_ref"] != semantica.Referencia ||
		atributos["propuesta_semantica_huella_sha256"] !=
			semantica.HuellaSHA256 ||
		atributos["reserva_ref"] != reserva.reservaRef ||
		atributos["revision_cercado"] !=
			strconv.FormatUint(reserva.revisionCercado, 10) {
		return false
	}
	requeridosReferencia := []string{
		"propuesta_ref",
		"preparacion_evidencias_ref",
		"analisis_ref",
		"catalogo_ref",
		"politica_ref",
		"politica_actuacion_ref",
	}
	for _, clave := range requeridosReferencia {
		if !domain.ReferenciaOpacaValida(atributos[clave]) {
			return false
		}
	}
	requeridosHuella := []string{
		"propuesta_huella_sha256",
		"propuesta_semantica_huella_sha256",
		"preparacion_evidencias_huella_sha256",
		"analisis_huella_sha256",
		"catalogo_huella_sha256",
		"politica_huella_sha256",
		"politica_actuacion_huella_sha256",
	}
	for _, clave := range requeridosHuella {
		if !huellaSHA256OperacionDecisionCoberturaValida(atributos[clave]) {
			return false
		}
	}
	requeridosVersion := []string{
		"catalogo_version",
		"politica_version",
		"politica_actuacion_version",
	}
	for _, clave := range requeridosVersion {
		version, err := strconv.ParseUint(atributos[clave], 10, 64)
		if err != nil || version == 0 ||
			version > MaximoEnteroSeguroOperacionDecisionCobertura {
			return false
		}
	}
	esperados := 24
	if identidad.tipo == domain.DecisionCoberturaRectificacion {
		esperados += 2
		if atributos["predecesora_ref"] != identidad.predecesoraRef ||
			atributos["predecesora_huella_sha256"] !=
				identidad.predecesoraHuella {
			return false
		}
	}
	return len(atributos) == esperados
}

func candidataLigaPruebaDenegacionOperacionDecisionCobertura(
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	prueba pruebaDenegacionOperacionDecisionCobertura,
) bool {
	concedida, _, denegacion, err := candidata.Resultado()
	if err != nil || concedida {
		return false
	}
	orden, err := denegacion.Datos()
	if err != nil {
		return false
	}
	solicitud, errSolicitud := orden.Solicitud.Datos()
	vinculo, errVinculo := solicitud.VinculoAutenticacionActor.Datos()
	correlacion, errCorrelacion := solicitud.Correlacion.ValorCanonico()
	return errSolicitud == nil && errVinculo == nil && errCorrelacion == nil &&
		recursosOperacionDecisionCoberturaIguales(
			solicitud.Recurso,
			prueba.recursoVEC,
		) &&
		vinculo.PrincipalID == prueba.actorRef &&
		vinculo.PerfilActivoRef == prueba.perfilRef &&
		solicitud.Accion == string(prueba.accionVEC) &&
		solicitud.Finalidad == string(prueba.finalidadVEC) &&
		solicitud.ReferenciaMotivo == prueba.motivoVEC &&
		orden.ReferenciaMotivo == prueba.motivoVEC &&
		referenciasOperacionDecisionCoberturaIguales(
			correlacion,
			prueba.reserva.correlacionVECRef,
		)
}

func clonarSolicitudReservaOperacionDecisionCobertura(
	solicitud SolicitudReservarOperacionDecisionCobertura,
) (SolicitudReservarOperacionDecisionCobertura, error) {
	if solicitud.Validar() != nil {
		return SolicitudReservarOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	identidad, errIdentidad := solicitud.consulta.identidadInterna()
	ambitos, errAmbitos := clonarColeccionHMACOperacionDecisionCobertura(
		solicitud.consulta.sellos.AmbitosIdempotenciaHMAC,
	)
	semanticas, errSemanticas := clonarColeccionHMACOperacionDecisionCobertura(
		solicitud.consulta.sellos.HuellasSemanticasHMAC,
	)
	if errIdentidad != nil || errAmbitos != nil || errSemanticas != nil {
		return SolicitudReservarOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	consulta, errConsulta :=
		NuevaSolicitudConsultarOperacionDecisionCoberturaConfirmada(
			identidad,
			SellosOperacionDecisionCobertura{
				AmbitosIdempotenciaHMAC: ambitos,
				HuellasSemanticasHMAC:   semanticas,
			},
		)
	if errConsulta != nil {
		return SolicitudReservarOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	clon, err := NuevaSolicitudReservarOperacionDecisionCobertura(
		consulta,
		solicitud.token,
	)
	if err != nil {
		return SolicitudReservarOperacionDecisionCobertura{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return clon, nil
}

func clonarColeccionHMACOperacionDecisionCobertura(
	coleccion ports.ColeccionSellosHMAC,
) (ports.ColeccionSellosHMAC, error) {
	datos, err := coleccion.Datos()
	if err != nil {
		return ports.ColeccionSellosHMAC{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	retenidos := make([]string, len(datos.Retenidos))
	for indice := range datos.Retenidos {
		retenidos[indice] = datos.Retenidos[indice].Valor
	}
	clon, err := ports.NuevaColeccionSellosHMAC(
		datos.Activo.Valor,
		retenidos,
	)
	if err != nil {
		return ports.ColeccionSellosHMAC{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return clon, nil
}

func clonarCandidataOperacionDecisionCobertura(
	candidata puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
) (
	puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	concedida, concesion, denegacion, err := candidata.Resultado()
	var datos puertosvec.DatosOrdenRegistroAutorizacionLigadaV3
	if err == nil && concedida {
		datos, err = concesion.Datos()
	} else if err == nil {
		datos, err = denegacion.Datos()
	}
	if err != nil {
		return puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	clon, err := puertosvec.NuevaCandidataRegistroDecisionAutorizacionLigadaV3(
		datos.Solicitud,
		datos.Decision,
		datos.ReferenciaMotivo,
		datos.ResultadoContexto,
	)
	if err != nil {
		return puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	resumen, err := clon.Resumen()
	if err != nil {
		return puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			puertosvec.ResumenCandidataRegistroDecisionAutorizacionLigadaV3{},
			ErrOrdenOperacionDecisionCoberturaInvalida
	}
	return clon, resumen, nil
}
