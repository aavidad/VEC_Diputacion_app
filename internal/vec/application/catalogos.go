package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrDependenciaCatalogosRequerida = errors.New("vec: dependencia de catalogos requerida")
	ErrOrdenCatalogoInvalida         = errors.New("vec: orden de catalogo invalida")
	ErrSerializacionOrdenCatalogo    = errors.New("vec: serializacion de orden interna de catalogo prohibida")
)

const (
	AccionCrearCatalogoConfigurable      = ports.AccionCrearCatalogoConfigurable
	AccionActualizarCatalogoConfigurable = ports.AccionActualizarCatalogoConfigurable
	AccionPublicarCatalogoConfigurable   = ports.AccionPublicarCatalogoConfigurable
	AccionRetirarCatalogoConfigurable    = ports.AccionRetirarCatalogoConfigurable
)

type ServicioCatalogos struct {
	consulta    ports.ConsultaCatalogosConfigurables
	gobierno    ports.RepositorioGobiernoCatalogos
	autorizador ports.Autorizador
	reloj       ports.Reloj
}

func NuevoServicioCatalogos(
	consulta ports.ConsultaCatalogosConfigurables,
	gobierno ports.RepositorioGobiernoCatalogos,
	autorizador ports.Autorizador,
	reloj ports.Reloj,
) (*ServicioCatalogos, error) {
	if dependenciaCatalogoNula(consulta) || dependenciaCatalogoNula(gobierno) ||
		dependenciaCatalogoNula(autorizador) || dependenciaCatalogoNula(reloj) {
		return nil, ErrDependenciaCatalogosRequerida
	}
	return &ServicioCatalogos{consulta: consulta, gobierno: gobierno, autorizador: autorizador, reloj: reloj}, nil
}

// CredencialesGobiernoCatalogo agrupa las capacidades opacas resueltas por el
// middleware interno. No contiene roles ni permisos declarados por el cliente.
type CredencialesGobiernoCatalogo struct {
	ContextoActor             domain.ContextoActor
	VinculoAutenticacionActor domain.VinculoAutenticacionActorV1
}

type OrdenCrearBorradorCatalogo struct {
	Credenciales   CredencialesGobiernoCatalogo
	Finalidad      string
	ID             string
	Version        int
	ModuloID       string
	Nombre         string
	Descripcion    string
	FuenteRef      string
	Entradas       []domain.EntradaCatalogoConfigurable
	Motivo         string
	CorrelacionRef string
}

func (OrdenCrearBorradorCatalogo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOrdenCatalogo
}
func (*OrdenCrearBorradorCatalogo) UnmarshalJSON([]byte) error {
	return ErrSerializacionOrdenCatalogo
}
func (OrdenCrearBorradorCatalogo) String() string { return "[ORDEN-CATALOGO-INTERNA]" }

func (s *ServicioCatalogos) CrearBorrador(ctx context.Context, orden OrdenCrearBorradorCatalogo) (domain.CatalogoConfigurable, error) {
	actor, err := s.validarContextoGobiernoCatalogo(
		ctx, orden.Credenciales, orden.Finalidad, orden.Motivo, orden.CorrelacionRef,
	)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if !actor.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.CatalogoConfigurable{}, domain.ErrGarantiaInsuficiente
	}
	if orden.Version < 1 {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	if orden.ID != strings.TrimSpace(orden.ID) || orden.ModuloID != strings.TrimSpace(orden.ModuloID) {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if instante.IsZero() {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	catalogo := domain.CatalogoConfigurable{
		ID:             strings.TrimSpace(orden.ID),
		Version:        orden.Version,
		Revision:       1,
		ModuloID:       strings.TrimSpace(orden.ModuloID),
		Nombre:         strings.TrimSpace(orden.Nombre),
		Descripcion:    strings.TrimSpace(orden.Descripcion),
		FuenteRef:      strings.TrimSpace(orden.FuenteRef),
		MotivoCreacion: strings.TrimSpace(orden.Motivo),
		Entradas:       append([]domain.EntradaCatalogoConfigurable(nil), orden.Entradas...),
		Estado:         domain.EstadoCatalogoBorrador,
		CreadoPor:      actor.Principal.ID,
		CreadoEn:       instante,
	}
	if orden.Version > 1 {
		anterior, err := s.consulta.ObtenerCatalogo(ctx, catalogo.ID, orden.Version-1)
		if err != nil {
			return domain.CatalogoConfigurable{}, err
		}
		if strings.TrimSpace(orden.ModuloID) != anterior.ModuloID {
			return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
		}
		base, err := anterior.NuevaVersion(orden.Version, actor.Principal.ID, orden.FuenteRef, orden.Motivo, instante)
		if err != nil {
			return domain.CatalogoConfigurable{}, err
		}
		base.Nombre = catalogo.Nombre
		base.Descripcion = catalogo.Descripcion
		base.Entradas = catalogo.Entradas
		catalogo = base
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	decision, err := s.autorizarCatalogo(ctx, actor, orden.Credenciales.VinculoAutenticacionActor, AccionCrearCatalogoConfigurable,
		canonico, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	huella, err := canonico.HuellaSHA256()
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	traza, evento := evidenciaGobiernoCatalogo(canonico, actor.Principal, actor.PerfilActivoRef, decision.DecisionRef,
		orden.Finalidad, domain.AccionCatalogoBorradorCreado, "", huella, orden.CorrelacionRef)
	evidencia, err := s.evidenciaUsoDecisionCatalogo(ctx, decision)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if err := s.gobierno.ConfirmarAltaBorradorCatalogo(ctx, canonico, traza, evento, evidencia); err != nil {
		return domain.CatalogoConfigurable{}, fmt.Errorf("confirmar borrador de catalogo: %w", err)
	}
	return canonico, nil
}

type OrdenActualizarBorradorCatalogo struct {
	Credenciales     CredencialesGobiernoCatalogo
	Finalidad        string
	ID               string
	Version          int
	RevisionEsperada int
	Nombre           string
	Descripcion      string
	FuenteRef        string
	Entradas         []domain.EntradaCatalogoConfigurable
	Motivo           string
	CorrelacionRef   string
}

func (OrdenActualizarBorradorCatalogo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOrdenCatalogo
}
func (*OrdenActualizarBorradorCatalogo) UnmarshalJSON([]byte) error {
	return ErrSerializacionOrdenCatalogo
}
func (OrdenActualizarBorradorCatalogo) String() string { return "[ORDEN-CATALOGO-INTERNA]" }

func (s *ServicioCatalogos) ActualizarBorrador(ctx context.Context, orden OrdenActualizarBorradorCatalogo) (domain.CatalogoConfigurable, error) {
	actor, err := s.validarContextoGobiernoCatalogo(
		ctx, orden.Credenciales, orden.Finalidad, orden.Motivo, orden.CorrelacionRef,
	)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if !actor.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.CatalogoConfigurable{}, domain.ErrGarantiaInsuficiente
	}
	if orden.Version < 1 || orden.RevisionEsperada < 1 {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	if orden.ID != strings.TrimSpace(orden.ID) {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	actual, err := s.consulta.ObtenerCatalogo(ctx, orden.ID, orden.Version)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	huellaAnterior, err := actual.HuellaSHA256()
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	actualizado, err := actual.ActualizarBorrador(orden.RevisionEsperada, actor.Principal.ID, orden.Nombre,
		orden.Descripcion, orden.FuenteRef, orden.Motivo, orden.Entradas, s.reloj.Ahora().UTC().Truncate(time.Microsecond))
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	decision, err := s.autorizarCatalogo(ctx, actor, orden.Credenciales.VinculoAutenticacionActor, AccionActualizarCatalogoConfigurable,
		actualizado, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	huellaPosterior, err := actualizado.HuellaSHA256()
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	traza, evento := evidenciaGobiernoCatalogo(actualizado, actor.Principal, actor.PerfilActivoRef, decision.DecisionRef,
		orden.Finalidad, domain.AccionCatalogoBorradorActualizado, huellaAnterior, huellaPosterior, orden.CorrelacionRef)
	evidencia, err := s.evidenciaUsoDecisionCatalogo(ctx, decision)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if err := s.gobierno.ConfirmarActualizacionBorradorCatalogo(ctx, huellaAnterior, actualizado, traza, evento, evidencia); err != nil {
		return domain.CatalogoConfigurable{}, fmt.Errorf("confirmar actualizacion de catalogo: %w", err)
	}
	return actualizado, nil
}

type OrdenPublicarCatalogo struct {
	Credenciales   CredencialesGobiernoCatalogo
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (OrdenPublicarCatalogo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOrdenCatalogo
}
func (*OrdenPublicarCatalogo) UnmarshalJSON([]byte) error { return ErrSerializacionOrdenCatalogo }
func (OrdenPublicarCatalogo) String() string              { return "[ORDEN-CATALOGO-INTERNA]" }

func (s *ServicioCatalogos) Publicar(ctx context.Context, orden OrdenPublicarCatalogo) (domain.CatalogoConfigurable, error) {
	actor, err := s.validarContextoGobiernoCatalogo(
		ctx, orden.Credenciales, orden.Finalidad, orden.Motivo, orden.CorrelacionRef,
	)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if !actor.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.CatalogoConfigurable{}, domain.ErrGarantiaInsuficiente
	}
	if strings.TrimSpace(orden.AprobacionRef) == "" || orden.Version < 1 {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	if orden.ID != strings.TrimSpace(orden.ID) {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	borrador, err := s.consulta.ObtenerCatalogo(ctx, orden.ID, orden.Version)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	huellaAnterior, err := borrador.HuellaSHA256()
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	publicado, err := borrador.Publicar(actor.Principal.ID, orden.AprobacionRef, orden.Motivo, s.reloj.Ahora().UTC().Truncate(time.Microsecond))
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	decision, err := s.autorizarCatalogo(ctx, actor, orden.Credenciales.VinculoAutenticacionActor, AccionPublicarCatalogoConfigurable,
		publicado, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	huellaPosterior, err := publicado.HuellaSHA256()
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	traza, evento := evidenciaGobiernoCatalogo(publicado, actor.Principal, actor.PerfilActivoRef, decision.DecisionRef,
		orden.Finalidad, domain.AccionCatalogoPublicado, huellaAnterior, huellaPosterior, orden.CorrelacionRef)
	evidencia, err := s.evidenciaUsoDecisionCatalogo(ctx, decision)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if err := s.gobierno.ConfirmarPublicacionCatalogo(ctx, huellaAnterior, publicado, traza, evento, evidencia); err != nil {
		return domain.CatalogoConfigurable{}, fmt.Errorf("confirmar publicacion de catalogo: %w", err)
	}
	return publicado, nil
}

type OrdenRetirarCatalogo struct {
	Credenciales   CredencialesGobiernoCatalogo
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (OrdenRetirarCatalogo) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionOrdenCatalogo
}
func (*OrdenRetirarCatalogo) UnmarshalJSON([]byte) error { return ErrSerializacionOrdenCatalogo }
func (OrdenRetirarCatalogo) String() string              { return "[ORDEN-CATALOGO-INTERNA]" }

func (s *ServicioCatalogos) Retirar(ctx context.Context, orden OrdenRetirarCatalogo) (domain.CatalogoConfigurable, error) {
	actor, err := s.validarContextoGobiernoCatalogo(
		ctx, orden.Credenciales, orden.Finalidad, orden.Motivo, orden.CorrelacionRef,
	)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if !actor.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.CatalogoConfigurable{}, domain.ErrGarantiaInsuficiente
	}
	if strings.TrimSpace(orden.AprobacionRef) == "" || orden.Version < 1 {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	if orden.ID != strings.TrimSpace(orden.ID) {
		return domain.CatalogoConfigurable{}, ErrOrdenCatalogoInvalida
	}
	publicado, err := s.consulta.ObtenerCatalogo(ctx, orden.ID, orden.Version)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	huellaAnterior, err := publicado.HuellaSHA256()
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	retirado, err := publicado.Retirar(actor.Principal.ID, orden.AprobacionRef, orden.Motivo, s.reloj.Ahora().UTC().Truncate(time.Microsecond))
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	decision, err := s.autorizarCatalogo(ctx, actor, orden.Credenciales.VinculoAutenticacionActor, AccionRetirarCatalogoConfigurable,
		retirado, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	huellaPosterior, err := retirado.HuellaSHA256()
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	traza, evento := evidenciaGobiernoCatalogo(retirado, actor.Principal, actor.PerfilActivoRef, decision.DecisionRef,
		orden.Finalidad, domain.AccionCatalogoRetirado, huellaAnterior, huellaPosterior, orden.CorrelacionRef)
	evidencia, err := s.evidenciaUsoDecisionCatalogo(ctx, decision)
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if err := s.gobierno.ConfirmarRetiradaCatalogo(ctx, huellaAnterior, retirado, traza, evento, evidencia); err != nil {
		return domain.CatalogoConfigurable{}, fmt.Errorf("confirmar retirada de catalogo: %w", err)
	}
	return retirado, nil
}

func (s *ServicioCatalogos) evidenciaUsoDecisionCatalogo(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) (ports.EvidenciaUsoDecisionAutorizacion, error) {
	if ctx == nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errors.Join(
			domain.ErrAutorizacionDenegada,
			domain.ErrConfiguracionAccesoInvalida,
		)
	}
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	instante := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(decision, instante)
	if err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaUsoDecisionAutorizacion{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	return evidencia, nil
}

func (s *ServicioCatalogos) autorizarCatalogo(
	ctx context.Context,
	actor domain.ContextoActor,
	vinculo domain.VinculoAutenticacionActorV1,
	accion string,
	catalogo domain.CatalogoConfigurable,
	finalidad, correlacionRef, motivo string,
) (domain.DecisionAutorizacion, error) {
	return exigirDecisionAutorizacionVinculada(ctx, s.autorizador, s.reloj, actor, vinculo, accion,
		domain.RecursoAutorizable{
			Referencia: catalogo.Referencia(),
			ModuloID:   catalogo.ModuloID,
			Tipo:       "catalogo_configurable",
			Atributos: map[string]string{
				"estado":   string(catalogo.Estado),
				"revision": strconv.Itoa(catalogo.Revision),
			},
		}, finalidad, correlacionRef, motivo, usoCamposDecisionNoAplicables)
}

func (s *ServicioCatalogos) validarContextoGobiernoCatalogo(
	ctx context.Context,
	credenciales CredencialesGobiernoCatalogo,
	finalidad, motivo, correlacionRef string,
) (domain.ContextoActor, error) {
	if s == nil || dependenciaCatalogoNula(s.consulta) || dependenciaCatalogoNula(s.gobierno) ||
		dependenciaCatalogoNula(s.autorizador) || dependenciaCatalogoNula(s.reloj) || ctx == nil {
		return domain.ContextoActor{}, ErrDependenciaCatalogosRequerida
	}
	if err := ctx.Err(); err != nil {
		return domain.ContextoActor{}, err
	}
	actor, err := credenciales.ContextoActor.Clonar()
	if err != nil || credenciales.VinculoAutenticacionActor.ValidarPara(actor) != nil {
		return domain.ContextoActor{}, errors.Join(ErrOrdenCatalogoInvalida, domain.ErrVinculoAutenticacionActorInvalido, err)
	}
	if finalidad != strings.TrimSpace(finalidad) || motivo != strings.TrimSpace(motivo) ||
		correlacionRef != strings.TrimSpace(correlacionRef) || finalidad == "" || motivo == "" || correlacionRef == "" {
		return domain.ContextoActor{}, ErrOrdenCatalogoInvalida
	}
	return actor, nil
}

func evidenciaGobiernoCatalogo(
	catalogo domain.CatalogoConfigurable,
	principal domain.Principal,
	perfilActivo, autorizacionRef, finalidad, accion, huellaAnterior, huellaPosterior, correlacionRef string,
) (domain.AuditEntry, domain.Event) {
	actor, fecha, regla, motivo := catalogo.CreadoPor, catalogo.CreadoEn, catalogo.FuenteRef, catalogo.MotivoCreacion
	switch accion {
	case domain.AccionCatalogoBorradorActualizado:
		actor, fecha, motivo = catalogo.UltimaModificacionPor, catalogo.UltimaModificacionEn, catalogo.MotivoModificacion
	case domain.AccionCatalogoPublicado:
		actor, fecha, regla, motivo = catalogo.PublicadoPor, catalogo.PublicadoEn, catalogo.AprobacionRef, catalogo.MotivoPublicacion
	case domain.AccionCatalogoRetirado:
		actor, fecha, regla, motivo = catalogo.RetiradoPor, catalogo.RetiradoEn, catalogo.RetiradaAprobacionRef, catalogo.MotivoRetirada
	}
	referencia := catalogo.Referencia()
	traza := domain.AuditEntry{
		ActorID:          actor,
		ActorProfile:     strings.TrimSpace(perfilActivo),
		ActorRoles:       append([]string(nil), principal.Roles...),
		AuthMethod:       principal.AuthMethod,
		AuthAssurance:    principal.AuthAssurance,
		AuthorizationRef: strings.TrimSpace(autorizacionRef),
		Purpose:          strings.TrimSpace(finalidad),
		Action:           accion,
		ModuleID:         catalogo.ModuloID,
		SubjectRef:       referencia,
		ObjectVersion:    catalogo.Version,
		RuleRef:          regla,
		Reason:           motivo,
		Result:           "correcto",
		BeforeHash:       huellaAnterior,
		AfterHash:        huellaPosterior,
		CorrelationRef:   strings.TrimSpace(correlacionRef),
		OccurredAt:       fecha.UTC(),
		Metadata: map[string]string{
			"catalogo_id":      catalogo.ID,
			"catalogo_version": strconv.Itoa(catalogo.Version),
			"estado":           string(catalogo.Estado),
			"revision":         strconv.Itoa(catalogo.Revision),
		},
	}
	evento := domain.Event{
		Type:       accion,
		ModuleID:   catalogo.ModuloID,
		SubjectRef: referencia,
		ActorID:    actor,
		OccurredAt: fecha.UTC(),
		Payload: map[string]string{
			"catalogo_id":       catalogo.ID,
			"catalogo_version":  strconv.Itoa(catalogo.Version),
			"catalogo_revision": strconv.Itoa(catalogo.Revision),
			"estado":            string(catalogo.Estado),
			"huella_sha256":     huellaPosterior,
		},
	}
	return traza, evento
}

func dependenciaCatalogoNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
