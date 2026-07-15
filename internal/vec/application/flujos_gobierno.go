package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrDependenciaGobiernoFlujosRequerida = errors.New("vec: dependencia de gobierno de flujos requerida")
	ErrOrdenGobiernoFlujoInvalida         = errors.New("vec: orden de gobierno de flujo invalida")
	ErrReferenciaCatalogoFlujoInvalida    = errors.New("vec: referencia de catalogo de flujo invalida")
)

const (
	AccionCrearDefinicionFlujo      = "vec.flujos.definicion.crear"
	AccionActualizarDefinicionFlujo = "vec.flujos.definicion.actualizar"
	AccionPublicarDefinicionFlujo   = "vec.flujos.definicion.publicar"
	AccionRetirarDefinicionFlujo    = "vec.flujos.definicion.retirar"
)

type ServicioGobiernoFlujos struct {
	consulta    ports.ConsultaDefinicionesFlujo
	gobierno    ports.RepositorioGobiernoFlujos
	catalogos   ports.ConsultaCatalogosConfigurables
	autorizador ports.Autorizador
	reloj       ports.Reloj
}

func NuevoServicioGobiernoFlujos(
	consulta ports.ConsultaDefinicionesFlujo,
	gobierno ports.RepositorioGobiernoFlujos,
	catalogos ports.ConsultaCatalogosConfigurables,
	autorizador ports.Autorizador,
	reloj ports.Reloj,
) (*ServicioGobiernoFlujos, error) {
	if consulta == nil || gobierno == nil || catalogos == nil || autorizador == nil || reloj == nil {
		return nil, ErrDependenciaGobiernoFlujosRequerida
	}
	return &ServicioGobiernoFlujos{
		consulta: consulta, gobierno: gobierno, catalogos: catalogos, autorizador: autorizador, reloj: reloj,
	}, nil
}

type OrdenCrearBorradorFlujo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	ModuloID       string
	TipoEntidad    string
	Configuracion  domain.ConfiguracionBorradorFlujo
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioGobiernoFlujos) CrearBorrador(
	ctx context.Context,
	orden OrdenCrearBorradorFlujo,
) (domain.DefinicionFlujo, error) {
	if err := validarContextoGobiernoFlujo(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad, orden.Motivo, orden.CorrelacionRef); err != nil {
		return domain.DefinicionFlujo{}, err
	}
	if !orden.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.DefinicionFlujo{}, domain.ErrGarantiaInsuficiente
	}
	if orden.Version < 1 || orden.ID != strings.TrimSpace(orden.ID) ||
		orden.ModuloID != strings.TrimSpace(orden.ModuloID) || orden.TipoEntidad != strings.TrimSpace(orden.TipoEntidad) {
		return domain.DefinicionFlujo{}, ErrOrdenGobiernoFlujoInvalida
	}
	instante := s.reloj.Ahora().UTC()
	if instante.IsZero() {
		return domain.DefinicionFlujo{}, ErrOrdenGobiernoFlujoInvalida
	}
	definicion := domain.DefinicionFlujo{
		ID:                              orden.ID,
		Version:                         orden.Version,
		Revision:                        1,
		ModuloID:                        orden.ModuloID,
		TipoEntidad:                     orden.TipoEntidad,
		Nombre:                          orden.Configuracion.Nombre,
		Descripcion:                     orden.Configuracion.Descripcion,
		FuenteRef:                       orden.Configuracion.FuenteRef,
		MotivoCreacion:                  strings.TrimSpace(orden.Motivo),
		EstadoInicial:                   orden.Configuracion.EstadoInicial,
		AccionInicio:                    orden.Configuracion.AccionInicio,
		GarantiaInicio:                  orden.Configuracion.GarantiaInicio,
		PermiteFinalizacionTrasRetirada: orden.Configuracion.PermiteFinalizacionTrasRetirada,
		Estados:                         append([]domain.EstadoFlujoConfigurable(nil), orden.Configuracion.Estados...),
		Transiciones:                    append([]domain.TransicionFlujoConfigurable(nil), orden.Configuracion.Transiciones...),
		Estado:                          domain.EstadoDefinicionFlujoBorrador,
		CreadaPor:                       strings.TrimSpace(orden.Principal.ID),
		CreadaEn:                        instante,
	}
	if orden.Version > 1 {
		anterior, err := s.consulta.ObtenerDefinicionFlujo(ctx, orden.ID, orden.Version-1)
		if err != nil {
			return domain.DefinicionFlujo{}, err
		}
		if anterior.ModuloID != orden.ModuloID || anterior.TipoEntidad != orden.TipoEntidad {
			return domain.DefinicionFlujo{}, ErrOrdenGobiernoFlujoInvalida
		}
		base, err := anterior.NuevaVersion(orden.Version, orden.Principal.ID, orden.Configuracion.FuenteRef, orden.Motivo, instante)
		if err != nil {
			return domain.DefinicionFlujo{}, err
		}
		base.Nombre = orden.Configuracion.Nombre
		base.Descripcion = orden.Configuracion.Descripcion
		base.EstadoInicial = orden.Configuracion.EstadoInicial
		base.AccionInicio = orden.Configuracion.AccionInicio
		base.GarantiaInicio = orden.Configuracion.GarantiaInicio
		base.PermiteFinalizacionTrasRetirada = orden.Configuracion.PermiteFinalizacionTrasRetirada
		base.Estados = append([]domain.EstadoFlujoConfigurable(nil), orden.Configuracion.Estados...)
		base.Transiciones = append([]domain.TransicionFlujoConfigurable(nil), orden.Configuracion.Transiciones...)
		definicion = base
	}
	canonica, err := definicion.ClonarCanonico()
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	decision, err := s.autorizarDefinicion(ctx, orden.Principal, orden.PerfilActivo, AccionCrearDefinicionFlujo,
		canonica, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	huella, err := canonica.HuellaSHA256()
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	traza, evento := evidenciaGobiernoFlujo(canonica, orden.Principal, orden.PerfilActivo, decision.DecisionRef,
		orden.Finalidad, domain.AccionDefinicionFlujoBorradorCreada, "", huella, orden.CorrelacionRef)
	if err := s.gobierno.ConfirmarAltaBorradorFlujo(ctx, canonica, traza, evento); err != nil {
		return domain.DefinicionFlujo{}, fmt.Errorf("confirmar borrador de flujo: %w", err)
	}
	return canonica, nil
}

type OrdenActualizarBorradorFlujo struct {
	Principal        domain.Principal
	PerfilActivo     string
	Finalidad        string
	ID               string
	Version          int
	RevisionEsperada int
	Configuracion    domain.ConfiguracionBorradorFlujo
	Motivo           string
	CorrelacionRef   string
}

func (s *ServicioGobiernoFlujos) ActualizarBorrador(
	ctx context.Context,
	orden OrdenActualizarBorradorFlujo,
) (domain.DefinicionFlujo, error) {
	if err := validarContextoGobiernoFlujo(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad, orden.Motivo, orden.CorrelacionRef); err != nil {
		return domain.DefinicionFlujo{}, err
	}
	if !orden.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.DefinicionFlujo{}, domain.ErrGarantiaInsuficiente
	}
	if orden.Version < 1 || orden.RevisionEsperada < 1 || orden.ID != strings.TrimSpace(orden.ID) {
		return domain.DefinicionFlujo{}, ErrOrdenGobiernoFlujoInvalida
	}
	actual, err := s.consulta.ObtenerDefinicionFlujo(ctx, orden.ID, orden.Version)
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	huellaAnterior, err := actual.HuellaSHA256()
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	actualizada, err := actual.ActualizarBorrador(
		orden.RevisionEsperada,
		orden.Principal.ID,
		orden.Motivo,
		orden.Configuracion,
		s.reloj.Ahora().UTC(),
	)
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	decision, err := s.autorizarDefinicion(ctx, orden.Principal, orden.PerfilActivo, AccionActualizarDefinicionFlujo,
		actualizada, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	huellaPosterior, err := actualizada.HuellaSHA256()
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	traza, evento := evidenciaGobiernoFlujo(actualizada, orden.Principal, orden.PerfilActivo, decision.DecisionRef,
		orden.Finalidad, domain.AccionDefinicionFlujoBorradorActualizada, huellaAnterior, huellaPosterior, orden.CorrelacionRef)
	if err := s.gobierno.ConfirmarActualizacionBorradorFlujo(ctx, huellaAnterior, actualizada, traza, evento); err != nil {
		return domain.DefinicionFlujo{}, fmt.Errorf("confirmar actualizacion de flujo: %w", err)
	}
	return actualizada, nil
}

type OrdenPublicarFlujo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioGobiernoFlujos) Publicar(ctx context.Context, orden OrdenPublicarFlujo) (domain.DefinicionFlujo, error) {
	if err := validarContextoGobiernoFlujo(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad, orden.Motivo, orden.CorrelacionRef); err != nil {
		return domain.DefinicionFlujo{}, err
	}
	if !orden.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.DefinicionFlujo{}, domain.ErrGarantiaInsuficiente
	}
	if orden.Version < 1 || orden.ID != strings.TrimSpace(orden.ID) || strings.TrimSpace(orden.AprobacionRef) == "" {
		return domain.DefinicionFlujo{}, ErrOrdenGobiernoFlujoInvalida
	}
	borrador, err := s.consulta.ObtenerDefinicionFlujo(ctx, orden.ID, orden.Version)
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	decision, err := s.autorizarDefinicion(ctx, orden.Principal, orden.PerfilActivo, AccionPublicarDefinicionFlujo,
		borrador, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	if err := s.validarReferenciasCatalogo(ctx, borrador); err != nil {
		return domain.DefinicionFlujo{}, err
	}
	huellaAnterior, err := borrador.HuellaSHA256()
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	publicada, err := borrador.Publicar(orden.Principal.ID, orden.AprobacionRef, orden.Motivo, s.reloj.Ahora().UTC())
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	huellaPosterior, err := publicada.HuellaSHA256()
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	traza, evento := evidenciaGobiernoFlujo(publicada, orden.Principal, orden.PerfilActivo, decision.DecisionRef,
		orden.Finalidad, domain.AccionDefinicionFlujoPublicada, huellaAnterior, huellaPosterior, orden.CorrelacionRef)
	if err := s.gobierno.ConfirmarPublicacionFlujo(ctx, huellaAnterior, publicada, traza, evento); err != nil {
		return domain.DefinicionFlujo{}, fmt.Errorf("confirmar publicacion de flujo: %w", err)
	}
	return publicada, nil
}

type OrdenRetirarFlujo struct {
	Principal      domain.Principal
	PerfilActivo   string
	Finalidad      string
	ID             string
	Version        int
	AprobacionRef  string
	Motivo         string
	CorrelacionRef string
}

func (s *ServicioGobiernoFlujos) Retirar(ctx context.Context, orden OrdenRetirarFlujo) (domain.DefinicionFlujo, error) {
	if err := validarContextoGobiernoFlujo(ctx, orden.Principal, orden.PerfilActivo, orden.Finalidad, orden.Motivo, orden.CorrelacionRef); err != nil {
		return domain.DefinicionFlujo{}, err
	}
	if !orden.Principal.AuthAssurance.Cumple(domain.AuthAssuranceHigh) {
		return domain.DefinicionFlujo{}, domain.ErrGarantiaInsuficiente
	}
	if orden.Version < 1 || orden.ID != strings.TrimSpace(orden.ID) || strings.TrimSpace(orden.AprobacionRef) == "" {
		return domain.DefinicionFlujo{}, ErrOrdenGobiernoFlujoInvalida
	}
	publicada, err := s.consulta.ObtenerDefinicionFlujo(ctx, orden.ID, orden.Version)
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	decision, err := s.autorizarDefinicion(ctx, orden.Principal, orden.PerfilActivo, AccionRetirarDefinicionFlujo,
		publicada, orden.Finalidad, orden.CorrelacionRef, orden.Motivo)
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	huellaAnterior, err := publicada.HuellaSHA256()
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	retirada, err := publicada.Retirar(orden.Principal.ID, orden.AprobacionRef, orden.Motivo, s.reloj.Ahora().UTC())
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	huellaPosterior, err := retirada.HuellaSHA256()
	if err != nil {
		return domain.DefinicionFlujo{}, err
	}
	traza, evento := evidenciaGobiernoFlujo(retirada, orden.Principal, orden.PerfilActivo, decision.DecisionRef,
		orden.Finalidad, domain.AccionDefinicionFlujoRetirada, huellaAnterior, huellaPosterior, orden.CorrelacionRef)
	if err := s.gobierno.ConfirmarRetiradaFlujo(ctx, huellaAnterior, retirada, traza, evento); err != nil {
		return domain.DefinicionFlujo{}, fmt.Errorf("confirmar retirada de flujo: %w", err)
	}
	return retirada, nil
}

func (s *ServicioGobiernoFlujos) validarReferenciasCatalogo(ctx context.Context, definicion domain.DefinicionFlujo) error {
	catalogos := make(map[string]domain.CatalogoConfigurable)
	for _, estado := range definicion.Estados {
		clave := estado.Catalogo.CatalogoID + ":" + strconv.Itoa(estado.Catalogo.CatalogoVersion)
		catalogo, existe := catalogos[clave]
		if !existe {
			var err error
			catalogo, err = s.catalogos.ObtenerCatalogo(ctx, estado.Catalogo.CatalogoID, estado.Catalogo.CatalogoVersion)
			if err != nil {
				return errors.Join(ErrReferenciaCatalogoFlujoInvalida, err)
			}
			if catalogo.Estado != domain.EstadoCatalogoPublicado {
				return ErrReferenciaCatalogoFlujoInvalida
			}
			huella, err := catalogo.HuellaContenidoSHA256()
			if err != nil || huella != estado.Catalogo.CatalogoHuellaSHA256 {
				return ErrReferenciaCatalogoFlujoInvalida
			}
			catalogos[clave] = catalogo
		}
		encontrada := false
		for _, entrada := range catalogo.Entradas {
			if entrada.Clave == estado.Catalogo.EntradaClave {
				encontrada = true
				break
			}
		}
		if !encontrada || estado.Clave != estado.Catalogo.EntradaClave {
			return ErrReferenciaCatalogoFlujoInvalida
		}
	}
	return nil
}

func (s *ServicioGobiernoFlujos) autorizarDefinicion(
	ctx context.Context,
	principal domain.Principal,
	perfilActivo, accion string,
	definicion domain.DefinicionFlujo,
	finalidad, correlacionRef, motivo string,
) (domain.DecisionAutorizacion, error) {
	return exigirDecisionAutorizacion(ctx, s.autorizador, s.reloj, principal, perfilActivo, accion,
		domain.RecursoAutorizable{
			Referencia: definicion.Referencia(),
			ModuloID:   definicion.ModuloID,
			Tipo:       "definicion_flujo",
			Atributos: map[string]string{
				"estado":       string(definicion.Estado),
				"revision":     strconv.Itoa(definicion.Revision),
				"tipo_entidad": definicion.TipoEntidad,
			},
		}, finalidad, correlacionRef, motivo, usoCamposDecisionNoAplicables)
}

func validarContextoGobiernoFlujo(
	ctx context.Context,
	principal domain.Principal,
	perfilActivo, finalidad, motivo, correlacionRef string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := principal.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(perfilActivo) == "" || strings.TrimSpace(finalidad) == "" ||
		strings.TrimSpace(motivo) == "" || strings.TrimSpace(correlacionRef) == "" {
		return ErrOrdenGobiernoFlujoInvalida
	}
	return nil
}

func evidenciaGobiernoFlujo(
	definicion domain.DefinicionFlujo,
	principal domain.Principal,
	perfilActivo, autorizacionRef, finalidad, accion, huellaAnterior, huellaPosterior, correlacionRef string,
) (domain.AuditEntry, domain.Event) {
	actor, fecha, regla, motivo := definicion.CreadaPor, definicion.CreadaEn, definicion.FuenteRef, definicion.MotivoCreacion
	switch accion {
	case domain.AccionDefinicionFlujoBorradorActualizada:
		actor, fecha, motivo = definicion.UltimaModificacionPor, definicion.UltimaModificacionEn, definicion.MotivoModificacion
	case domain.AccionDefinicionFlujoPublicada:
		actor, fecha, regla, motivo = definicion.PublicadaPor, definicion.PublicadaEn, definicion.AprobacionRef, definicion.MotivoPublicacion
	case domain.AccionDefinicionFlujoRetirada:
		actor, fecha, regla, motivo = definicion.RetiradaPor, definicion.RetiradaEn, definicion.RetiradaAprobacionRef, definicion.MotivoRetirada
	}
	huellaContenido, _ := definicion.HuellaContenidoSHA256()
	referencia := definicion.Referencia()
	traza := domain.AuditEntry{
		ActorID:          actor,
		ActorProfile:     strings.TrimSpace(perfilActivo),
		ActorRoles:       append([]string(nil), principal.Roles...),
		AuthMethod:       principal.AuthMethod,
		AuthAssurance:    principal.AuthAssurance,
		AuthorizationRef: strings.TrimSpace(autorizacionRef),
		Purpose:          strings.TrimSpace(finalidad),
		Action:           accion,
		ModuleID:         definicion.ModuloID,
		SubjectRef:       referencia,
		ObjectVersion:    definicion.Version,
		RuleRef:          regla,
		Reason:           motivo,
		Result:           "correcto",
		BeforeHash:       huellaAnterior,
		AfterHash:        huellaPosterior,
		CorrelationRef:   strings.TrimSpace(correlacionRef),
		OccurredAt:       fecha.UTC(),
		Metadata: map[string]string{
			"definicion_id":           definicion.ID,
			"definicion_version":      strconv.Itoa(definicion.Version),
			"revision":                strconv.Itoa(definicion.Revision),
			"tipo_entidad":            definicion.TipoEntidad,
			"huella_contenido_sha256": huellaContenido,
		},
	}
	evento := domain.Event{
		Type:       accion,
		ModuleID:   definicion.ModuloID,
		SubjectRef: referencia,
		ActorID:    actor,
		OccurredAt: fecha.UTC(),
		Payload: map[string]string{
			"definicion_id":           definicion.ID,
			"definicion_version":      strconv.Itoa(definicion.Version),
			"definicion_revision":     strconv.Itoa(definicion.Revision),
			"estado":                  string(definicion.Estado),
			"huella_sha256":           huellaPosterior,
			"huella_contenido_sha256": huellaContenido,
		},
	}
	return traza, evento
}
