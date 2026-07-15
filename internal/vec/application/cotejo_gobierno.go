package application

import (
	"context"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	eventoBorradorPoliticaCotejoCreado      = domain.AccionPoliticaCotejoBorradorCreada
	eventoBorradorPoliticaCotejoActualizado = domain.AccionPoliticaCotejoBorradorActualizada
	eventoPoliticaCotejoPublicada           = domain.AccionPoliticaCotejoPublicada
	eventoPoliticaCotejoRetirada            = domain.AccionPoliticaCotejoRetirada
)

type ConfiguracionPoliticaCotejo struct {
	Nombre                   string
	Descripcion              string
	Modulos                  []string
	TiposDocumentales        []string
	Clasificaciones          []string
	ClaseAcceso              domain.ClaseAccesoCotejo
	CamposPublicos           []domain.CampoPublicoCotejo
	PermiteDescargaDocumento bool
	RequiereTitularidad      bool
	RolesTitularidad         []string
	RequiereFirma            bool
	RequiereSelloTiempo      bool
	RequiereRegistro         bool
	GarantiaMinima           domain.AuthAssurance
	DiasPlazoActivacion      int
	DiasDisponibilidad       int
	FuenteRef                string
}

type OrdenCrearBorradorPoliticaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	Configuracion  ConfiguracionPoliticaCotejo
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioCotejo) CrearBorradorPoliticaCotejo(ctx context.Context, orden OrdenCrearBorradorPoliticaCotejo) (domain.PoliticaCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	if err := validarContextoCotejo(orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.Motivo, orden.CorrelacionRef, domain.AuthAssuranceHigh); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	politica := politicaCotejoDesdeConfiguracion(orden.ID, orden.Version, orden.Configuracion)
	politica.Revision = 1
	politica.Estado = domain.EstadoPoliticaCotejoBorrador
	politica.MotivoCreacion = strings.TrimSpace(orden.Motivo)
	politica.CreadaPor = strings.TrimSpace(orden.Principal.ID)
	politica.CreadaEn = ahora
	if politica.Version > 1 {
		anterior, err := s.politicas.ObtenerPoliticaCotejo(ctx, politica.ID, politica.Version-1)
		if err != nil {
			return domain.PoliticaCotejo{}, err
		}
		if anterior.Estado != domain.EstadoPoliticaCotejoPublicada && anterior.Estado != domain.EstadoPoliticaCotejoRetirada {
			return domain.PoliticaCotejo{}, ports.ErrSecuenciaPoliticaCotejoInvalida
		}
		politica.VersionAnteriorRef = anterior.Referencia()
	}
	canonico, err := politica.ClonarCanonica()
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	decision, err := exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj,
		orden.Principal, orden.PerfilActivo, AccionCrearBorradorPoliticaCotejo,
		recursoPoliticaCotejo(canonico), orden.Finalidad, orden.CorrelacionRef, orden.Motivo,
		usoCamposDecisionNoAplicables)
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	huella, err := canonico.HuellaSHA256()
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	traza := trazaGobiernoPoliticaCotejo(orden.Principal, orden.PerfilActivo, decision.DecisionRef,
		orden.Finalidad, eventoBorradorPoliticaCotejoCreado, canonico, orden.Motivo,
		orden.CorrelacionRef, "", huella, ahora)
	evento := eventoGobiernoPoliticaCotejo(eventoBorradorPoliticaCotejoCreado, canonico, huella, ahora)
	if err := s.gobiernoPoliticas.ConfirmarAltaBorradorPoliticaCotejo(ctx, canonico, traza, evento); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	return canonico, nil
}

type OrdenActualizarBorradorPoliticaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	Configuracion  ConfiguracionPoliticaCotejo
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioCotejo) ActualizarBorradorPoliticaCotejo(ctx context.Context, orden OrdenActualizarBorradorPoliticaCotejo) (domain.PoliticaCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	if err := validarContextoCotejo(orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.Motivo, orden.CorrelacionRef, domain.AuthAssuranceHigh); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	anterior, err := s.politicas.ObtenerPoliticaCotejo(ctx, strings.TrimSpace(orden.ID), orden.Version)
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	huellaAnterior, err := anterior.HuellaSHA256()
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	propuesta := politicaCotejoDesdeConfiguracion(anterior.ID, anterior.Version, orden.Configuracion)
	ahora := s.reloj.Ahora().UTC()
	actualizada, err := anterior.ActualizarBorrador(propuesta, orden.Principal.ID, orden.Motivo, ahora)
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	decision, err := exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj,
		orden.Principal, orden.PerfilActivo, AccionActualizarBorradorPoliticaCotejo,
		recursoPoliticaCotejo(actualizada), orden.Finalidad, orden.CorrelacionRef, orden.Motivo,
		usoCamposDecisionNoAplicables)
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	huellaNueva, err := actualizada.HuellaSHA256()
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	traza := trazaGobiernoPoliticaCotejo(orden.Principal, orden.PerfilActivo, decision.DecisionRef,
		orden.Finalidad, eventoBorradorPoliticaCotejoActualizado, actualizada, orden.Motivo,
		orden.CorrelacionRef, huellaAnterior, huellaNueva, ahora)
	evento := eventoGobiernoPoliticaCotejo(eventoBorradorPoliticaCotejoActualizado, actualizada, huellaNueva, ahora)
	if err := s.gobiernoPoliticas.ConfirmarActualizacionBorradorPoliticaCotejo(ctx, huellaAnterior, actualizada, traza, evento); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	return actualizada, nil
}

type OrdenPublicarPoliticaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioCotejo) PublicarPoliticaCotejo(ctx context.Context, orden OrdenPublicarPoliticaCotejo) (domain.PoliticaCotejo, error) {
	return s.transicionarPoliticaCotejo(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.ID, orden.Version, orden.AprobacionRef, orden.Motivo, orden.CorrelacionRef, true)
}

type OrdenRetirarPoliticaCotejo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioCotejo) RetirarPoliticaCotejo(ctx context.Context, orden OrdenRetirarPoliticaCotejo) (domain.PoliticaCotejo, error) {
	return s.transicionarPoliticaCotejo(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad,
		orden.ID, orden.Version, orden.AprobacionRef, orden.Motivo, orden.CorrelacionRef, false)
}

func (s *ServicioCotejo) transicionarPoliticaCotejo(
	ctx context.Context,
	principal domain.Principal,
	perfilActivo, finalidad, id string,
	version int,
	aprobacionRef, motivo, correlacionRef string,
	publicar bool,
) (domain.PoliticaCotejo, error) {
	if err := ctx.Err(); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	if err := validarContextoCotejo(principal, perfilActivo, finalidad, motivo, correlacionRef, domain.AuthAssuranceHigh); err != nil {
		return domain.PoliticaCotejo{}, err
	}
	anterior, err := s.politicas.ObtenerPoliticaCotejo(ctx, strings.TrimSpace(id), version)
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	huellaAnterior, err := anterior.HuellaSHA256()
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	accion := AccionRetirarPoliticaCotejo
	tipoEvento := eventoPoliticaCotejoRetirada
	if publicar {
		accion = AccionPublicarPoliticaCotejo
		tipoEvento = eventoPoliticaCotejoPublicada
	}
	decision, err := exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj, principal,
		perfilActivo, accion, recursoPoliticaCotejo(anterior), finalidad, correlacionRef, motivo,
		usoCamposDecisionNoAplicables)
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	ahora := s.reloj.Ahora().UTC()
	var cambiada domain.PoliticaCotejo
	if publicar {
		cambiada, err = anterior.Publicar(principal.ID, aprobacionRef, motivo, ahora)
	} else {
		cambiada, err = anterior.Retirar(principal.ID, aprobacionRef, motivo, ahora)
	}
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	huellaNueva, err := cambiada.HuellaSHA256()
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	traza := trazaGobiernoPoliticaCotejo(principal, perfilActivo, decision.DecisionRef, finalidad,
		tipoEvento, cambiada, motivo, correlacionRef, huellaAnterior, huellaNueva, ahora)
	traza.RuleRef = strings.TrimSpace(aprobacionRef)
	evento := eventoGobiernoPoliticaCotejo(tipoEvento, cambiada, huellaNueva, ahora)
	if publicar {
		err = s.gobiernoPoliticas.ConfirmarPublicacionPoliticaCotejo(ctx, huellaAnterior, cambiada, traza, evento)
	} else {
		err = s.gobiernoPoliticas.ConfirmarRetiradaPoliticaCotejo(ctx, huellaAnterior, cambiada, traza, evento)
	}
	if err != nil {
		return domain.PoliticaCotejo{}, err
	}
	return cambiada, nil
}

func politicaCotejoDesdeConfiguracion(id string, version int, configuracion ConfiguracionPoliticaCotejo) domain.PoliticaCotejo {
	return domain.PoliticaCotejo{
		ID:                       strings.TrimSpace(id),
		Version:                  version,
		Nombre:                   strings.TrimSpace(configuracion.Nombre),
		Descripcion:              strings.TrimSpace(configuracion.Descripcion),
		Modulos:                  append([]string(nil), configuracion.Modulos...),
		TiposDocumentales:        append([]string(nil), configuracion.TiposDocumentales...),
		Clasificaciones:          append([]string(nil), configuracion.Clasificaciones...),
		ClaseAcceso:              configuracion.ClaseAcceso,
		CamposPublicos:           append([]domain.CampoPublicoCotejo(nil), configuracion.CamposPublicos...),
		PermiteDescargaDocumento: configuracion.PermiteDescargaDocumento,
		RequiereTitularidad:      configuracion.RequiereTitularidad,
		RolesTitularidad:         append([]string(nil), configuracion.RolesTitularidad...),
		RequiereFirma:            configuracion.RequiereFirma,
		RequiereSelloTiempo:      configuracion.RequiereSelloTiempo,
		RequiereRegistro:         configuracion.RequiereRegistro,
		GarantiaMinima:           configuracion.GarantiaMinima,
		DiasPlazoActivacion:      configuracion.DiasPlazoActivacion,
		DiasDisponibilidad:       configuracion.DiasDisponibilidad,
		FuenteRef:                strings.TrimSpace(configuracion.FuenteRef),
	}
}

func trazaGobiernoPoliticaCotejo(
	principal domain.Principal,
	perfilActivo, autorizacionRef, finalidad, accion string,
	politica domain.PoliticaCotejo,
	motivo, correlacionRef, huellaAnterior, huellaNueva string,
	instante time.Time,
) domain.AuditEntry {
	return domain.AuditEntry{
		ActorID:          strings.TrimSpace(principal.ID),
		ActorProfile:     strings.TrimSpace(perfilActivo),
		ActorRoles:       append([]string(nil), principal.Roles...),
		AuthMethod:       principal.AuthMethod,
		AuthAssurance:    principal.AuthAssurance,
		AuthorizationRef: strings.TrimSpace(autorizacionRef),
		Purpose:          strings.TrimSpace(finalidad),
		Action:           accion,
		ModuleID:         moduloNucleoDocumental,
		SubjectRef:       politica.Referencia(),
		ObjectVersion:    politica.Revision,
		RuleRef:          politica.FuenteRef,
		Reason:           strings.TrimSpace(motivo),
		Result:           "correcto",
		BeforeHash:       huellaAnterior,
		AfterHash:        huellaNueva,
		CorrelationRef:   strings.TrimSpace(correlacionRef),
		Metadata:         metadatosPoliticaCotejo(politica),
		OccurredAt:       instante.UTC(),
	}
}

func eventoGobiernoPoliticaCotejo(tipo string, politica domain.PoliticaCotejo, huella string, instante time.Time) domain.Event {
	return domain.Event{
		Type:       tipo,
		ModuleID:   moduloNucleoDocumental,
		SubjectRef: politica.Referencia(),
		ActorID:    actorTransicionPoliticaCotejo(politica),
		OccurredAt: instante.UTC(),
		Payload: map[string]string{
			"politica_id":      politica.ID,
			"politica_version": strconv.Itoa(politica.Version),
			"revision":         strconv.Itoa(politica.Revision),
			"estado":           string(politica.Estado),
			"huella_sha256":    huella,
		},
	}
}

func actorTransicionPoliticaCotejo(politica domain.PoliticaCotejo) string {
	switch politica.Estado {
	case domain.EstadoPoliticaCotejoBorrador:
		if politica.Revision > 1 {
			return politica.ActualizadaPor
		}
		return politica.CreadaPor
	case domain.EstadoPoliticaCotejoPublicada:
		return politica.PublicadaPor
	case domain.EstadoPoliticaCotejoRetirada:
		return politica.RetiradaPor
	default:
		return ""
	}
}

func metadatosPoliticaCotejo(politica domain.PoliticaCotejo) map[string]string {
	return map[string]string{
		"clase_acceso":          string(politica.ClaseAcceso),
		"dias_activacion":       strconv.Itoa(politica.DiasPlazoActivacion),
		"dias_disponibilidad":   strconv.Itoa(politica.DiasDisponibilidad),
		"modulos":               strconv.Itoa(len(politica.Modulos)),
		"tipos_documentales":    strconv.Itoa(len(politica.TiposDocumentales)),
		"clasificaciones":       strconv.Itoa(len(politica.Clasificaciones)),
		"requiere_firma":        strconv.FormatBool(politica.RequiereFirma),
		"roles_titularidad":     strconv.Itoa(len(politica.RolesTitularidad)),
		"requiere_sello_tiempo": strconv.FormatBool(politica.RequiereSelloTiempo),
		"requiere_registro":     strconv.FormatBool(politica.RequiereRegistro),
	}
}
