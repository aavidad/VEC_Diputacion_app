package application

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const vigenciaDecisionPredeterminada = 2 * time.Minute

type ConfiguracionServicioAutorizacion struct {
	VigenciaDecision time.Duration
}

// ServicioAutorizacion evalua RBAC seguido de restricciones ABAC sin conocer
// PostgreSQL, un IdP ni el generador criptografico concreto. Un fallo en
// cualquier dependencia termina siempre en denegacion.
type ServicioAutorizacion struct {
	fuente                 ports.FuenteAutorizacion
	registroConcesiones    ports.RegistroDecisionesAutorizacion
	registroDenegaciones   ports.RegistroDenegacionesAutorizacion
	registroConcesionesV2  ports.RegistroDecisionesAutorizacionSolicitudLigadaV2
	registroDenegacionesV2 ports.RegistroDenegacionesAutorizacionSolicitudLigadaV2
	validadorMotivosV2     ports.ValidadorReferenciaMotivoAutorizacionV2
	reloj                  ports.Reloj
	generador              ports.GeneradorReferenciaDecisionAutorizacion
	vigenciaDecision       time.Duration
}

func NuevoServicioAutorizacion(
	fuente ports.FuenteAutorizacion,
	registroConcesiones ports.RegistroDecisionesAutorizacion,
	registroDenegaciones ports.RegistroDenegacionesAutorizacion,
	reloj ports.Reloj,
	generador ports.GeneradorReferenciaDecisionAutorizacion,
	configuracion ConfiguracionServicioAutorizacion,
) (*ServicioAutorizacion, error) {
	if dependenciaAutorizacionNula(fuente) || dependenciaAutorizacionNula(registroConcesiones) ||
		dependenciaAutorizacionNula(registroDenegaciones) ||
		dependenciaAutorizacionNula(reloj) || dependenciaAutorizacionNula(generador) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	vigencia := configuracion.VigenciaDecision
	if vigencia == 0 {
		vigencia = vigenciaDecisionPredeterminada
	}
	if vigencia < 0 || vigencia > domain.VigenciaMaximaDecisionAutorizacion || vigencia%time.Microsecond != 0 {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &ServicioAutorizacion{
		fuente:               fuente,
		registroConcesiones:  registroConcesiones,
		registroDenegaciones: registroDenegaciones,
		reloj:                reloj,
		generador:            generador,
		vigenciaDecision:     vigencia,
	}, nil
}

func (s *ServicioAutorizacion) Exigir(ctx context.Context, solicitud domain.SolicitudAutorizacion) (domain.DecisionAutorizacion, error) {
	return s.exigir(ctx, solicitud, nil)
}

func (s *ServicioAutorizacion) exigir(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacion,
	solicitudLigadaV2 *domain.SolicitudAutorizacionLigadaV2,
) (domain.DecisionAutorizacion, error) {
	if s == nil {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrConfiguracionAccesoInvalida,
		)
	}
	ligarSolicitudV2 := solicitudLigadaV2 != nil
	registrosInvalidos := false
	if ligarSolicitudV2 {
		registrosInvalidos = dependenciaAutorizacionNula(s.registroConcesionesV2) ||
			dependenciaAutorizacionNula(s.registroDenegacionesV2) ||
			dependenciaAutorizacionNula(s.validadorMotivosV2)
	} else {
		registrosInvalidos = dependenciaAutorizacionNula(s.registroConcesiones) ||
			dependenciaAutorizacionNula(s.registroDenegaciones)
	}
	if ctx == nil || dependenciaAutorizacionNula(s.fuente) ||
		registrosInvalidos ||
		dependenciaAutorizacionNula(s.reloj) || dependenciaAutorizacionNula(s.generador) {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrConfiguracionAccesoInvalida,
		)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if ligarSolicitudV2 {
		var err error
		solicitud, err = proyectarSolicitudAutorizacionLigadaV2(*solicitudLigadaV2)
		if err != nil {
			return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
		}
	}
	if err := solicitud.ValidarVinculoAutenticacionActor(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if ligarSolicitudV2 {
		if !domain.ReferenciaMotivoAutorizacionV2Valida(solicitud.ReferenciaMotivo) ||
			!domain.ReferenciaCorrelacionAutorizacionV2Valida(solicitud.CorrelacionRef) ||
			solicitud.Motivo != solicitud.ReferenciaMotivo.EntradaClave {
			return domain.DecisionAutorizacion{}, errors.Join(
				domain.ErrAutorizacionDenegada,
				domain.ErrSolicitudAutorizacionInvalida,
			)
		}
	} else if solicitud.TieneReferenciaMotivoAutorizacionV2() {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrSolicitudAutorizacionInvalida,
		)
	}
	// El reloj es una dependencia interna fiable: se canoniza antes de crear
	// cualquier evidencia porque timestamptz conserva microsegundos.
	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instante.IsZero() {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida)
	}
	if ligarSolicitudV2 {
		if err := s.validadorMotivosV2.ValidarReferenciaMotivoAutorizacionV2(
			ctx,
			solicitud.ReferenciaMotivo,
			instante,
		); err != nil {
			return domain.DecisionAutorizacion{}, errors.Join(
				domain.ErrAutorizacionDenegada,
				domain.ErrSolicitudAutorizacionInvalida,
				err,
			)
		}
	}
	if !solicitud.VinculoAutenticacionActor.VigenteEn(instante, solicitud.ContextoActor) {
		return domain.DecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrVinculoAutenticacionActorInvalido,
		)
	}
	huellaSolicitud, huellaMotivo := "", ""
	var err error
	if ligarSolicitudV2 {
		huellaSolicitud, err = domain.HuellaSHA256SolicitudAutorizacionV2(*solicitudLigadaV2)
		if err != nil {
			return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
		}
		huellaMotivo, err = domain.HuellaSHA256MotivoAutorizacionV2(solicitud.ReferenciaMotivo)
		if err != nil {
			return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
		}
	}
	instantanea, err := s.fuente.ObtenerInstantaneaAutorizacion(ctx, solicitud.Principal.ID, solicitud.PerfilActivoRef)
	if err != nil {
		// Sin una instantanea completa no existe base fiable para un CAS ni para
		// una evidencia de denegacion. Se falla cerrado y no se llama al registro:
		// persistir una decision inventada seria peor que no tener evidencia.
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if err := instantanea.Validar(); err != nil ||
		instantanea.AsignacionPerfil.PerfilActivoRef != solicitud.PerfilActivoRef ||
		instantanea.AsignacionPerfil.PrincipalID != solicitud.Principal.ID {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, domain.ErrConfiguracionAccesoInvalida, err)
	}
	referencia, err := s.generador.NuevaReferenciaDecisionAutorizacion()
	if err != nil || referencia == "" || referencia != strings.TrimSpace(referencia) {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	asignacion := instantanea.AsignacionPerfil
	versionRol := instantanea.VersionRol
	controlVigenciaRol := instantanea.ControlVigenciaVersionRol
	decision := domain.DecisionAutorizacion{
		DecisionRef:                       referencia,
		Codigo:                            "denegacion_por_defecto",
		PrincipalID:                       solicitud.Principal.ID,
		PerfilActivoRef:                   solicitud.PerfilActivoRef,
		Accion:                            solicitud.Accion,
		RecursoRef:                        solicitud.Recurso.Referencia,
		ModuloID:                          solicitud.Recurso.ModuloID,
		TipoRecurso:                       solicitud.Recurso.Tipo,
		Finalidad:                         solicitud.Finalidad,
		CorrelacionRef:                    solicitud.CorrelacionRef,
		VinculoAutenticacionActor:         solicitud.VinculoAutenticacionActor,
		AsignacionRef:                     asignacion.Referencia(),
		VersionRolRef:                     versionRol.Referencia(),
		ControlVigenciaVersionRolRef:      controlVigenciaRol.VersionRolRef,
		ControlVigenciaVersionRolRevision: controlVigenciaRol.Revision,
		RevisionCatalogoPoliticas:         instantanea.RevisionCatalogoPoliticas,
		CatalogoPoliticasHuellaSHA256:     instantanea.CatalogoPoliticasHuellaSHA256,
		PoliticasEvaluadasHuellasSHA256:   make(map[string]string, len(instantanea.Politicas)),
		PoliticasHuellasSHA256:            make(map[string]string),
		EmitidaEn:                         instante,
		ValidaHasta: limiteVinculoAutenticacionActorDecision(
			instante,
			instante.Add(s.vigenciaDecision),
			solicitud.VinculoAutenticacionActor,
			solicitud.ContextoActor,
		),
	}
	if ligarSolicitudV2 {
		decision.EsquemaHuellaSolicitud = domain.EsquemaHuellaSolicitudAutorizacionV2
		decision.SolicitudHuellaSHA256 = huellaSolicitud
		decision.EsquemaHuellaMotivo = domain.EsquemaHuellaMotivoAutorizacionV2
		decision.MotivoHuellaSHA256 = huellaMotivo
	}
	decision.AsignacionHuellaSHA256, err = asignacion.HuellaSHA256()
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	decision.ContextoRecursoHuellaSHA256, err = solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	decision.VersionRolHuellaSHA256, err = versionRol.HuellaSHA256()
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	decision.ControlVigenciaVersionRolHuellaSHA256, err = controlVigenciaRol.HuellaSHA256()
	if err != nil {
		return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	for _, politica := range instantanea.Politicas {
		referenciaPolitica := politica.Referencia()
		huellaPolitica, err := politica.HuellaSHA256()
		if err != nil {
			return domain.DecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
		}
		decision.PoliticasEvaluadasRefs = append(decision.PoliticasEvaluadasRefs, referenciaPolitica)
		decision.PoliticasEvaluadasHuellasSHA256[referenciaPolitica] = huellaPolitica
	}
	sort.Strings(decision.PoliticasEvaluadasRefs)

	finalizar := func(codigo string, concedida bool) (domain.DecisionAutorizacion, error) {
		decision.Codigo = codigo
		decision.Concedida = concedida
		decision.PoliticasRefs = normalizarUnicosAutorizacion(decision.PoliticasRefs)
		decision.CamposPermitidos = normalizarUnicosAutorizacion(decision.CamposPermitidos)
		decision.Obligaciones = normalizarUnicosAutorizacion(decision.Obligaciones)
		errDecision := decision.ValidarEvidenciaInstantanea()
		if ligarSolicitudV2 {
			errDecision = decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2()
		} else if decision.TieneSolicitudLigadaV2() {
			errDecision = domain.ErrDecisionAutorizacionInvalida
		}
		if errDecision != nil {
			decision.Concedida = false
			decision.Codigo = "decision_no_fiable"
			return decision, errors.Join(domain.ErrAutorizacionDenegada, errDecision)
		}
		var ordenRegistroV2 ports.OrdenRegistroDecisionAutorizacionSolicitudLigadaV2
		if ligarSolicitudV2 {
			var errOrden error
			ordenRegistroV2, errOrden = ports.NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(
				decision,
				solicitud.ReferenciaMotivo,
			)
			if errOrden != nil {
				decision.Concedida = false
				decision.Codigo = "decision_no_fiable"
				return decision, errors.Join(domain.ErrAutorizacionDenegada, errOrden)
			}
		}
		if !decision.Concedida {
			if errContexto := ctx.Err(); errContexto != nil {
				return decision, errors.Join(domain.ErrAutorizacionDenegada, errContexto)
			}
			var errRegistro error
			if ligarSolicitudV2 {
				errRegistro = s.registroDenegacionesV2.RegistrarDenegacionAutorizacionSolicitudLigadaV2(ctx, ordenRegistroV2)
			} else {
				errRegistro = s.registroDenegaciones.RegistrarDenegacionAutorizacion(ctx, decision)
			}
			if errRegistro != nil {
				return decision, errors.Join(
					domain.ErrAutorizacionDenegada,
					ports.ErrRegistroDenegacionNoDisponible,
					errRegistro,
				)
			}
			return decision, domain.ErrAutorizacionDenegada
		}
		if errContexto := ctx.Err(); errContexto != nil {
			decision.Concedida = false
			decision.Codigo = "contexto_cancelado"
			return decision, errors.Join(domain.ErrAutorizacionDenegada, errContexto)
		}
		var errRegistro error
		if ligarSolicitudV2 {
			errRegistro = s.registroConcesionesV2.RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(ctx, ordenRegistroV2)
		} else {
			errRegistro = s.registroConcesiones.RegistrarDecisionSiInstantaneaVigente(ctx, decision)
		}
		if errRegistro != nil {
			decision.Concedida = false
			if errors.Is(errRegistro, ports.ErrInstantaneaAutorizacionObsoleta) {
				decision.Codigo = "instantanea_obsoleta"
				return decision, errors.Join(domain.ErrAutorizacionDenegada, errRegistro)
			}
			decision.Codigo = "registro_no_disponible"
			return decision, errors.Join(domain.ErrAutorizacionDenegada, ports.ErrRegistroDecisionNoDisponible, errRegistro)
		}
		return decision, nil
	}

	if !asignacion.VigenteEn(instante) {
		return finalizar("perfil_no_vigente", false)
	}
	if !asignacion.Cubre(solicitud.Recurso) {
		return finalizar("ambito_no_autorizado", false)
	}

	if versionRol.Estado != domain.EstadoVersionRolPublicada ||
		versionRol.PublicadaEn.After(instante) || versionRol.Referencia() != asignacion.VersionRolRef {
		return finalizar("rol_no_publicado", false)
	}
	if controlVigenciaRol.Estado != domain.EstadoControlVigenciaVersionRolHabilitada {
		return finalizar("rol_retirado", false)
	}
	concesion, encontrada := seleccionarConcesion(
		versionRol,
		solicitud.Accion,
		solicitud.Recurso.ModuloID,
		solicitud.Recurso.Tipo,
	)
	if !encontrada {
		return finalizar("accion_no_concedida", false)
	}
	if !concesion.AdmiteFinalidad(solicitud.Finalidad) {
		return finalizar("finalidad_no_autorizada", false)
	}
	decision.GarantiaMinima = concesion.GarantiaMinima
	decision.CamposPermitidos = append([]string(nil), concesion.CamposPermitidos...)
	decision.Obligaciones = append([]string(nil), concesion.Obligaciones...)

	for _, politica := range instantanea.Politicas {
		if !politica.VigenteEn(instante) || !politica.AplicaA(solicitud) {
			continue
		}
		referenciaPolitica := politica.Referencia()
		huellaPolitica, err := politica.HuellaSHA256()
		if err != nil {
			return finalizar("politica_no_fiable", false)
		}
		decision.PoliticasRefs = append(decision.PoliticasRefs, referenciaPolitica)
		decision.PoliticasHuellasSHA256[referenciaPolitica] = huellaPolitica
		if politica.Efecto == domain.EfectoPoliticaDenegar {
			return finalizar("denegada_por_politica", false)
		}
		if !politica.Cumple(solicitud) {
			return finalizar("restriccion_abac_incumplida", false)
		}
		if politica.GarantiaMinima != "" {
			decision.GarantiaMinima, err = domain.GarantiaAutenticacionMasAlta(decision.GarantiaMinima, politica.GarantiaMinima)
			if err != nil {
				return finalizar("garantia_no_fiable", false)
			}
		}
		if politica.RestringeCampos {
			decision.CamposPermitidos = interseccionCamposAutorizacion(decision.CamposPermitidos, politica.CamposPermitidos)
		}
		decision.Obligaciones = append(decision.Obligaciones, politica.Obligaciones...)
	}
	if !domain.CumpleGarantiaAutenticacion(solicitud.Principal.AuthAssurance, decision.GarantiaMinima) {
		return finalizar("garantia_insuficiente", false)
	}
	// Una concesion no puede sobrevivir a la asignacion ni atravesar un cambio
	// temporal ya conocido del catalogo. Se consideran todas las politicas
	// publicadas, incluso las que aun no estan vigentes o no aplican a esta
	// solicitud, para que un alta futura nunca amplie por reutilizacion una
	// decision calculada con el estado anterior.
	decision.ValidaHasta = limiteTemporalDecisionAutorizacion(
		instante,
		decision.ValidaHasta,
		asignacion.VigenteHasta,
		instantanea.Politicas,
	)
	return finalizar("concedida", true)
}

func limiteVinculoAutenticacionActorDecision(
	instante time.Time,
	limite time.Time,
	vinculo domain.VinculoAutenticacionActorV1,
	actor domain.ContextoActor,
) time.Time {
	datosVinculo, err := vinculo.Datos()
	if err != nil {
		return instante
	}
	fronteras := []time.Time{
		datosVinculo.SesionValidaHasta,
		actor.Instantanea.VigenteHasta,
	}
	for _, referencia := range actor.Instantanea.Vinculos {
		fronteras = append(fronteras, referencia.VigenteHasta)
	}
	for _, frontera := range fronteras {
		if frontera.After(instante) && frontera.Before(limite) {
			limite = frontera
		}
	}
	return limite
}

func limiteTemporalDecisionAutorizacion(
	instante time.Time,
	limiteConfigurado time.Time,
	finAsignacion time.Time,
	politicas []domain.PoliticaRestrictiva,
) time.Time {
	limite := limiteConfigurado
	if finAsignacion.After(instante) && finAsignacion.Before(limite) {
		limite = finAsignacion
	}
	for _, politica := range politicas {
		if politica.Estado != domain.EstadoPoliticaRestrictivaPublicada {
			continue
		}
		for _, frontera := range []time.Time{politica.VigenteDesde, politica.VigenteHasta} {
			if frontera.After(instante) && frontera.Before(limite) {
				limite = frontera
			}
		}
	}
	return limite
}

func dependenciaAutorizacionNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

func seleccionarConcesion(version domain.VersionRol, accion, moduloID, tipoRecurso string) (domain.ConcesionRol, bool) {
	for _, concesion := range version.Concesiones {
		if concesion.Accion != accion {
			continue
		}
		if concesion.ModuloID == moduloID && concesion.TipoRecurso == tipoRecurso {
			return concesion, true
		}
	}
	return domain.ConcesionRol{}, false
}

func interseccionCamposAutorizacion(actuales, restriccion []string) []string {
	// El conjunto positivo del rol nunca contiene comodines. Solo una politica
	// restrictiva puede usar "*", y en ese caso no reduce el conjunto exacto ya
	// concedido por RBAC.
	if contieneCadenaExacta(restriccion, "*") {
		return append([]string(nil), actuales...)
	}
	permitidos := make(map[string]struct{}, len(restriccion))
	for _, campo := range restriccion {
		permitidos[campo] = struct{}{}
	}
	resultado := make([]string, 0, len(actuales))
	for _, campo := range actuales {
		if _, permitido := permitidos[campo]; permitido {
			resultado = append(resultado, campo)
		}
	}
	return resultado
}

func contieneCadenaExacta(valores []string, buscado string) bool {
	for _, valor := range valores {
		if valor == buscado {
			return true
		}
	}
	return false
}

func normalizarUnicosAutorizacion(valores []string) []string {
	vistos := make(map[string]struct{}, len(valores))
	resultado := make([]string, 0, len(valores))
	for _, valor := range valores {
		if _, existe := vistos[valor]; existe {
			continue
		}
		vistos[valor] = struct{}{}
		resultado = append(resultado, valor)
	}
	sort.Strings(resultado)
	return resultado
}

var _ ports.Autorizador = (*ServicioAutorizacion)(nil)
