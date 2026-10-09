package bootstrap

import (
	"context"
	"strconv"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func (s *soporteAltaContratacionTemporalDesarrollo) instantaneaParaContexto(
	ctx context.Context,
	ruta string,
) (dominiovec.InstantaneaAutorizacion, bool) {
	// Las rutas con perfil fijo nunca preparan ni publican: consumen.
	if fijo := s.perfilFijoParaContexto(ctx, ruta); fijo != nil {
		return s.instantaneaPerfilFijoParaContexto(ctx, ruta, fijo)
	}
	if ruta == rutaEntregaPeticionCentro {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	instantanea, valida := s.instantaneaParaRuta(ruta)
	dinamica := ruta == rutaCambiosOrganizacionContratacionTemporalDesarrollo ||
		ruta == httpinterno.RutaAltaSolicitudes ||
		ruta == httpinterno.RutaPropuestaCobertura ||
		rutaMutacionDurableContratacionTemporalDesarrollo(ruta) ||
		rutaConsultaRRHHContratacionTemporalDesarrollo(ruta) ||
		ruta == httpinterno.RutaDecisionCobertura ||
		ruta == httpinterno.RutaRectificacionCobertura
	if ruta == httpinterno.RutaSubsanacionReparos || rutaSeguimientoCeseDesarrollo(ruta) || rutaCancelacionCTDesarrollo(ruta) {
		dinamica = true
	}
	if !valida || !dinamica {
		return instantanea, valida
	}
	if ctx == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	datos, existe := ctx.Value(
		claveSolicitudAutorizacionContratacionTemporalDesarrollo{},
	).(dominiovec.DatosSolicitudAutorizacionLigadaV3)
	if !existe {
		if ruta == rutaEntregaPeticionCentro {
			if _, ok := altaDePeticionConfiable(ctx); ok {
				return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantanea), true
			}
		}
		if ruta == httpinterno.RutaAltaSolicitudes || ruta == httpinterno.RutaPropuestaCobertura {
			return instantanea, true
		}
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	if rutaPeticionCentroDesarrollo(ruta) {
		if !s.peticionesCentro || !solicitudAutorizacionPeticionCentroDesarrolloValida(ctx, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		// El perfil general del centro nunca se prepara ni se publica por
		// petición: se consume la asignación publicada o se deniega.
		if s.soloConsumePublicada {
			return s.instantaneaConsumidaPublicada(ctx)
		}
		m, _ := ctx.Value(claveMaterialPeticionCentroDesarrollo{}).(materialAutorizacionPeticionCentroDesarrollo)
		if s.perfilCancelacionCentro && (m.cancelacion == nil || m.cancelacion.accion != string(domain.AccionCancelarExpediente)) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		// La cancelación por el centro liga el expediente exacto: los
		// ámbitos son los del recurso, con el centro ya cotejado.
		if m.cancelacion != nil {
			instantanea.AsignacionPerfil.Ambitos = m.cancelacion.ambitos()
		}
	} else if ruta == rutaCambiosOrganizacionContratacionTemporalDesarrollo {
		if !solicitudAutorizacionOrganizacionDesarrolloValida(ctx, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
	} else if rutaConsultaRRHHContratacionTemporalDesarrollo(ruta) {
		if !s.solicitudAutorizacionConsultaRRHHDesarrolloValida(ruta, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		// El ámbito de organización es fijo; nunca se amplía desde la petición.
	} else if rutaAnalisisContratacionTemporalDesarrollo(ruta) ||
		rutaAsignacionContratacionTemporalDesarrollo(ruta) ||
		rutaInformeJuridicoContratacionTemporalDesarrollo(ruta) {
		// El análisis, la asignación y el informe jurídico solo se autorizan
		// con su perfil fijo (arriba); nunca se prepara ni se publica su
		// permiso por petición.
		return dominiovec.InstantaneaAutorizacion{}, false
	} else if rutaSeguimientoCeseDesarrollo(ruta) {
		ambitos, valida := s.ambitosSeguimientoCese(ruta, datos)
		if !valida {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		instantanea.AsignacionPerfil.Ambitos = ambitos
	} else if rutaCancelacionCTDesarrollo(ruta) {
		ambitos, valida := s.ambitosCancelacionCT(ruta, datos)
		if !valida {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		instantanea.AsignacionPerfil.Ambitos = ambitos
	} else if ruta == httpinterno.RutaSubsanacionReparos {
		// La subsanación solo se autoriza con su perfil fijo (arriba).
		return dominiovec.InstantaneaAutorizacion{}, false
	} else if ruta == httpinterno.RutaFirmaDocumento {
		if !solicitudAutorizacionFirmaDocumentoCTDesarrolloValida(ctx, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
	} else if rutaLlamamientoContratacionTemporalDesarrollo(ruta) {
		if !solicitudAutorizacionLlamamientoDesarrolloValida(ctx, ruta, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		if ruta == httpinterno.RutaRegistroComunicacionLlamamiento && accionCorreoLlamamientoDesarrollo(datos.Accion) {
			s.mu.Lock()
			instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaCorreo)
			s.mu.Unlock()
			if instantanea.Validar() != nil {
				return dominiovec.InstantaneaAutorizacion{}, false
			}
		}
		if ruta == httpinterno.RutaResolucionComunicacionLlamamiento || ruta == httpinterno.RutaContinuacionLlamamiento ||
			ruta == httpinterno.RutaEventoPlazoLlamamiento {
			s.mu.Lock()
			switch datos.Accion {
			case postgrescontratacion.AccionConsultaJustificanteRespuestaRecibida:
				instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaConsultaJustificante)
			case postgrescontratacion.AccionContinuacionLlamamiento:
				instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaContinuacionCT)
			case "bolsa.llamamiento.siguiente.abrir":
				instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaSiguienteBolsa)
			case postgrescontratacion.AccionResolucionManualLlamamiento:
				instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaResolucionManual)
			case "bolsa.llamamiento.aceptacion_rrhh.registrar":
				instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaAceptacionBolsa)
			case "bolsa.llamamiento.renuncia_rrhh.registrar":
				instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaRenunciaBolsa)
			}
			s.mu.Unlock()
			if instantanea.Validar() != nil {
				return dominiovec.InstantaneaAutorizacion{}, false
			}
		}
		if datos.Accion == ports.AccionReanudacionSeleccionLlamamiento {
			s.mu.Lock()
			instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaReanudacionLlamamiento)
			s.mu.Unlock()
			if instantanea.Validar() != nil {
				return dominiovec.InstantaneaAutorizacion{}, false
			}
		}
		if datos.Accion == ports.AccionReanudacionSolicitudLlamamiento {
			s.mu.Lock()
			instantanea = clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(s.instantaneaReanudacionSolicitudLlamamiento)
			s.mu.Unlock()
			if instantanea.Validar() != nil {
				return dominiovec.InstantaneaAutorizacion{}, false
			}
		}
		// Correo conserva la asignación publicada. El recurso no amplía sus ámbitos.
		if !accionCorreoLlamamientoDesarrollo(datos.Accion) {
			instantanea.AsignacionPerfil.Ambitos = ambitosLlamamientoDesarrollo(datos.Recurso)
		}
	} else if ruta == httpinterno.RutaDecisionCobertura ||
		ruta == httpinterno.RutaRectificacionCobertura {
		if !solicitudAutorizacionDecisionCoberturaDesarrolloValida(ruta, datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
	} else if ruta == httpinterno.RutaPropuestaCobertura {
		if !solicitudAutorizacionPropuestaCoberturaDesarrolloValida(datos) {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
	} else if !s.solicitudAutorizacionAltaContratacionTemporalDesarrolloValida(ruta, datos) {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	clave, claveValida := claveInstantaneaContratacionTemporalDesarrolloDesdeDatos(
		datos,
	)
	if !claveValida {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	preparada, existe := s.instantaneasPorSolicitud[clave]
	autoridad := s.autoridadAsignaciones
	s.mu.Unlock()
	if existe {
		return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(preparada),
			preparada.Validar() == nil
	}
	if autoridad == nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	var err error
	if ruta == httpinterno.RutaReincorporacionesTitular || ruta == httpinterno.RutaCapacidadReincorporacionTitular {
		ct130, ok := autoridad.(autoridadReincorporacionTitularPostgreSQL)
		if !ok {
			return dominiovec.InstantaneaAutorizacion{}, false
		}
		preparada, err = ct130.PrepararInstantaneaReincorporacionTitular(ctx, instantanea)
	} else {
		preparada, err = autoridad.PrepararInstantanea(ctx, instantanea)
	}
	if err != nil || preparada.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, false
	}
	s.mu.Lock()
	if existente, yaExiste := s.instantaneasPorSolicitud[clave]; yaExiste {
		preparada = existente
	} else {
		s.instantaneasPorSolicitud[clave] = preparada
	}
	s.mu.Unlock()
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(preparada),
		preparada.Validar() == nil
}

func (s *soporteAltaContratacionTemporalDesarrollo) publicarInstantaneaDecisionCobertura(
	ctx context.Context,
	ruta string,
) error {
	capacidad, valida := s.capacidadValida(ctx)
	if !valida || capacidad.ruta != ruta ||
		(ruta != httpinterno.RutaDecisionCobertura &&
			ruta != httpinterno.RutaRectificacionCobertura) {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	instantanea, valida := s.instantaneaParaContexto(ctx, ruta)
	if !valida || instantanea.Validar() != nil {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	// Con perfil fijo basta con que la asignación publicada sea consumible:
	// la confirmación la vuelve a comprobar bajo bloqueo; no se publica nada.
	if s.perfilFijoParaContexto(ctx, ruta) != nil {
		return nil
	}
	s.mu.Lock()
	autoridad := s.autoridadAsignaciones
	s.mu.Unlock()
	if autoridad == nil || autoridad.PublicarInstantanea(ctx, instantanea) != nil {
		return errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

func claveInstantaneaContratacionTemporalDesarrollo(
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
) (string, bool) {
	huella, err := dominiovec.HuellaSHA256SolicitudAutorizacionV3(solicitud)
	return huella, err == nil && huella != ""
}

func claveInstantaneaContratacionTemporalDesarrolloDesdeDatos(
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) (string, bool) {
	solicitud, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		return "", false
	}
	return claveInstantaneaContratacionTemporalDesarrollo(solicitud)
}

func (s *soporteAltaContratacionTemporalDesarrollo) solicitudAutorizacionAltaContratacionTemporalDesarrolloValida(
	ruta string,
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	return ruta == httpinterno.RutaAltaSolicitudes &&
		datos.Accion == ports.AccionCrearSolicitud &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == ports.TipoRecursoExpediente &&
		datos.Finalidad == ports.FinalidadCrearSolicitud &&
		len(datos.Recurso.Ambitos) == 3 &&
		datos.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		s.centroDeCatalogo(datos.Recurso.Ambitos["centro_ref"]) &&
		s.categoriaDeCatalogo(datos.Recurso.Ambitos["categoria_ref"])
}

// solicitudAutorizacionAnalisisContratacionTemporalDesarrolloValida: el
// recurso lleva el expediente en su referencia y en los ámbitos solo la
// organización y la fase y el estado previos, que el catálogo debe admitir.
func solicitudAutorizacionAnalisisContratacionTemporalDesarrolloValida(
	ruta string,
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
	fase faseOperacionCT,
) bool {
	accionValida := (ruta == httpinterno.RutaRegistroAnalisisRRHH && datos.Accion == ports.AccionRegistrarAnalisis) ||
		(ruta == httpinterno.RutaRectificacionAnalisisRRHH && datos.Accion == ports.AccionRectificarAnalisis)
	return accionValida && datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == ports.TipoRecursoAnalisis &&
		datos.Finalidad == finalidadAnalisisContratacionTemporalDesarrollo &&
		datos.Recurso.Referencia != "" &&
		len(datos.Recurso.Ambitos) == 3 &&
		datos.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		fase.admite(domain.ClaveFase(datos.Recurso.Ambitos["fase_previa"]),
			domain.EstadoOperativo(datos.Recurso.Ambitos["estado_previo"])) &&
		datos.Recurso.Atributos[ports.AtributoUnidadPoliticaRef] == unidadCoberturaContratacionTemporalDesarrollo
}

func solicitudAutorizacionPropuestaCoberturaDesarrolloValida(
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	versionTexto := datos.Recurso.Atributos["version_esperada"]
	version, err := strconv.ParseUint(versionTexto, 10, 64)
	return datos.Accion == accionPropuestaCoberturaDesarrollo &&
		datos.Finalidad == finalidadPropuestaCoberturaDesarrollo &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == ports.TipoRecursoExpediente &&
		len(datos.Recurso.Ambitos) == 2 &&
		datos.Recurso.Ambitos["organizacion_ref"] == organizacionAltaContratacionTemporalDesarrollo &&
		datos.Recurso.Ambitos["unidad_ejecutora_ref"] == unidadCoberturaContratacionTemporalDesarrollo &&
		len(datos.Recurso.Atributos) == 1 && err == nil && version != 0 &&
		strconv.FormatUint(version, 10) == versionTexto
}

func solicitudAutorizacionDecisionCoberturaDesarrolloValida(
	ruta string,
	datos dominiovec.DatosSolicitudAutorizacionLigadaV3,
) bool {
	accion := string(domain.AccionDecidirCoberturaGobernada)
	if ruta == httpinterno.RutaRectificacionCobertura {
		accion = string(domain.AccionRectificarCoberturaGobernada)
	} else if ruta != httpinterno.RutaDecisionCobertura {
		return false
	}
	return datos.Accion == accion &&
		datos.Recurso.ModuloID == ports.ModuloContratacion &&
		datos.Recurso.Tipo == tipoRecursoDecisionCoberturaDesarrollo &&
		datos.Finalidad == finalidadDecisionCoberturaDesarrollo &&
		len(datos.Recurso.Ambitos) == 2 &&
		datos.Recurso.Ambitos["organizacion_ref"] ==
			organizacionAltaContratacionTemporalDesarrollo &&
		datos.Recurso.Ambitos["unidad_ejecutora_ref"] ==
			unidadCoberturaContratacionTemporalDesarrollo
}

func nuevaInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
	origenOpcional ...*origenConsultasContratacionTemporalDesarrollo,
) (dominiovec.InstantaneaAutorizacion, error) {
	centros := []string{centroAltaContratacionTemporalDesarrollo}
	categorias := []string{categoriaAltaContratacionTemporalDesarrollo}
	var o *origenConsultasContratacionTemporalDesarrollo
	if len(origenOpcional) > 0 {
		o = origenOpcional[0]
	}
	if o != nil {
		centros = o.referenciasCentros()
		categorias = o.referenciasCategorias()
	}
	return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"tecnico_rrhh_desarrollo",
		"Tecnico RRHH de desarrollo",
		"asignacion-rrhh-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{{
			Accion:         ports.AccionCrearSolicitud,
			ModuloID:       ports.ModuloContratacion,
			TipoRecurso:    ports.TipoRecursoExpediente,
			Finalidades:    []string{ports.FinalidadCrearSolicitud},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		[]dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "centro_ref", Valores: centros},
			{Clave: "categoria_ref", Valores: categorias},
		},
	)
}

func nuevaInstantaneaAutorizacionCoberturaContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
) (dominiovec.InstantaneaAutorizacion, error) {
	concesion := func(accion, finalidad, tipoRecurso string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{
			Accion: accion, ModuloID: ports.ModuloContratacion,
			TipoRecurso:    tipoRecurso,
			Finalidades:    []string{finalidad},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"tecnico_rrhh_cobertura_desarrollo",
		"Tecnico RRHH de cobertura de desarrollo",
		"asignacion-rrhh-cobertura-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{
			{
				Accion: accionPropuestaCoberturaDesarrollo, ModuloID: ports.ModuloContratacion,
				TipoRecurso:      ports.TipoRecursoExpediente,
				Finalidades:      []string{finalidadPropuestaCoberturaDesarrollo},
				GarantiaMinima:   dominiovec.AuthAssuranceHigh,
				CamposPermitidos: append([]string(nil), camposPreparacionPropuestaCoberturaDesarrollo...),
				Obligaciones:     []string{"registrar_acceso"},
			},
			concesion(string(domain.AccionDecidirCoberturaGobernada), finalidadDecisionCoberturaDesarrollo, tipoRecursoDecisionCoberturaDesarrollo),
			concesion(string(domain.AccionRectificarCoberturaGobernada), finalidadDecisionCoberturaDesarrollo, tipoRecursoDecisionCoberturaDesarrollo),
			concesion(
				string(ports.AccionConsultarResultadoCobertura),
				string(ports.FinalidadRecuperarResultadoCobertura),
				ports.TipoRecursoExpediente,
			),
		},
		[]dominiovec.AmbitoPerfil{
			{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}},
			{Clave: "unidad_ejecutora_ref", Valores: []string{unidadCoberturaContratacionTemporalDesarrollo}},
		},
	)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	// La version 1 ya se publico sin campos de preparacion.
	instantanea.VersionRol.Version = 2
	instantanea.AsignacionPerfil.VersionRolRef = instantanea.VersionRol.Referencia()
	instantanea.ControlVigenciaVersionRol.VersionRolRef = instantanea.VersionRol.Referencia()
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return instantanea, nil
}

// nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo es la
// plantilla del perfil fijo del análisis: la organización y las fases y el
// estado previos del catálogo (c23), sin expediente.
func nuevaInstantaneaAutorizacionAnalisisContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
	fase faseOperacionCT,
) (dominiovec.InstantaneaAutorizacion, error) {
	if !fase.valida() {
		return dominiovec.InstantaneaAutorizacion{}, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	concesion := func(accion string) dominiovec.ConcesionRol {
		return dominiovec.ConcesionRol{
			Accion: accion, ModuloID: ports.ModuloContratacion,
			TipoRecurso:    ports.TipoRecursoAnalisis,
			Finalidades:    []string{finalidadAnalisisContratacionTemporalDesarrollo},
			GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID,
		perfilRef,
		ahora,
		"tecnico_rrhh_analisis_desarrollo",
		"Tecnico RRHH de analisis de desarrollo",
		"asignacion-rrhh-analisis-desarrollo-no-autoritativa",
		[]dominiovec.ConcesionRol{
			concesion(ports.AccionRegistrarAnalisis),
			concesion(ports.AccionRectificarAnalisis),
		},
		fase.ambitosPerfil(organizacionAltaContratacionTemporalDesarrollo),
	)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	// La versión 1 publicada solo permite registrar. La rectificación añade
	// una concesión y exige otra versión, sin alterar el rol histórico.
	instantanea.VersionRol.Version = 2
	instantanea.AsignacionPerfil.VersionRolRef = instantanea.VersionRol.Referencia()
	instantanea.ControlVigenciaVersionRol.VersionRolRef = instantanea.VersionRol.Referencia()
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return instantanea, nil
}

func nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
	principalID string,
	perfilRef string,
	ahora time.Time,
	rolID string,
	nombreRol string,
	asignacionID string,
	concesiones []dominiovec.ConcesionRol,
	ambitos []dominiovec.AmbitoPerfil,
) (dominiovec.InstantaneaAutorizacion, error) {
	desde, hasta, vigente := ventanaAutoridadSinteticaContratacionTemporalDesarrollo(ahora)
	if !vigente {
		return dominiovec.InstantaneaAutorizacion{},
			errAltaContratacionTemporalDesarrolloNoDisponible
	}
	publicadaEn := desde
	asignacionID = referenciaAltaContratacionTemporalDesarrollo(
		"asg_", principalID+"\x00"+perfilRef+"\x00"+asignacionID,
	)
	version := dominiovec.VersionRol{
		RolID: rolID, Version: 1, Nombre: nombreRol,
		Estado:       dominiovec.EstadoVersionRolPublicada,
		Concesiones:  concesiones,
		PublicadaPor: "seguridad:desarrollo:no-autoritativa",
		PublicadaEn:  publicadaEn,
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: asignacionID,
		Version:      1, PerfilActivoRef: perfilRef, PrincipalID: principalID,
		VersionRolRef: version.Referencia(),
		Estado:        dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos:       ambitos,
		VigenteDesde:  desde,
		VigenteHasta:  hasta,
		EmitidaPor:    "identidad:desarrollo:no-autoritativa",
		EmitidaEn:     desde,
	}
	huellaPoliticas, err := dominiovec.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, err
	}
	instantanea := dominiovec.InstantaneaAutorizacion{
		AsignacionPerfil: asignacion,
		VersionRol:       version,
		ControlVigenciaVersionRol: dominiovec.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: publicadaEn,
		},
		RevisionCatalogoPoliticas:     1,
		CatalogoPoliticasHuellaSHA256: huellaPoliticas,
	}
	if instantanea.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errAltaContratacionTemporalDesarrolloNoDisponible
	}
	return instantanea, nil
}

func referenciaMotivoAutorizacionCoberturaDesarrollo(
	operacion string,
) dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion_cobertura",
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo("catalogo-motivos-cobertura"),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_",
			"cobertura-"+operacion,
		),
	}
}

func referenciaMotivoAutorizacionAnalisisDesarrollo(
	operacion string,
) dominiovec.ReferenciaEntradaCatalogo {
	catalogoID := "motivos_autorizacion_analisis"
	materialHuella := "catalogo-motivos-analisis"
	if operacion == "rectificacion" {
		// El catálogo de registro v1 ya está publicado y es inmutable.
		catalogoID = "motivos_autorizacion_rectificacion_analisis"
		materialHuella = "catalogo-motivos-rectificacion-analisis"
	}
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogoID,
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: huellaAltaContratacionTemporalDesarrollo(materialHuella),
		EntradaClave: referenciaAltaContratacionTemporalDesarrollo(
			"motivo_",
			"analisis-"+operacion,
		),
	}
}

func clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(
	instantanea dominiovec.InstantaneaAutorizacion,
) dominiovec.InstantaneaAutorizacion {
	copia := instantanea
	copia.VersionRol.Concesiones = append(
		[]dominiovec.ConcesionRol(nil), instantanea.VersionRol.Concesiones...,
	)
	for indice := range copia.VersionRol.Concesiones {
		copia.VersionRol.Concesiones[indice].Finalidades = append(
			[]string(nil), instantanea.VersionRol.Concesiones[indice].Finalidades...,
		)
		copia.VersionRol.Concesiones[indice].CamposPermitidos = append(
			[]string(nil), instantanea.VersionRol.Concesiones[indice].CamposPermitidos...,
		)
		copia.VersionRol.Concesiones[indice].Obligaciones = append(
			[]string(nil), instantanea.VersionRol.Concesiones[indice].Obligaciones...,
		)
	}
	copia.AsignacionPerfil.Ambitos = append(
		[]dominiovec.AmbitoPerfil(nil), instantanea.AsignacionPerfil.Ambitos...,
	)
	for indice := range copia.AsignacionPerfil.Ambitos {
		copia.AsignacionPerfil.Ambitos[indice].Valores = append(
			[]string(nil), instantanea.AsignacionPerfil.Ambitos[indice].Valores...,
		)
	}
	copia.Politicas = append([]dominiovec.PoliticaRestrictiva(nil), instantanea.Politicas...)
	for indice := range copia.Politicas {
		politicaOrigen := instantanea.Politicas[indice]
		politica := &copia.Politicas[indice]
		politica.Acciones = append([]string(nil), politicaOrigen.Acciones...)
		politica.Modulos = append([]string(nil), politicaOrigen.Modulos...)
		politica.TiposRecurso = append([]string(nil), politicaOrigen.TiposRecurso...)
		politica.FinalidadesPermitidas = append(
			[]string(nil), politicaOrigen.FinalidadesPermitidas...,
		)
		politica.Restricciones = append(
			[]dominiovec.RestriccionAtributoRecurso(nil), politicaOrigen.Restricciones...,
		)
		for indiceRestriccion := range politica.Restricciones {
			politica.Restricciones[indiceRestriccion].ValoresPermitidos = append(
				[]string(nil), politicaOrigen.Restricciones[indiceRestriccion].ValoresPermitidos...,
			)
		}
	}
	return copia
}
