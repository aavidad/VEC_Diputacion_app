package memory

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func (s *Store) ConfirmarAltaBorradorCatalogo(ctx context.Context, catalogo domain.CatalogoConfigurable, traza domain.AuditEntry, evento domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return err
	}
	if canonico.Estado != domain.EstadoCatalogoBorrador || canonico.Revision != 1 ||
		!evidenciaCatalogoValida(canonico, traza, evento, domain.AccionCatalogoBorradorCreado, "") {
		return domain.ErrCatalogoConfigurableInvalido
	}
	clave := claveCatalogo(canonico.ID, canonico.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, existe := s.catalogos[clave]; existe {
		return ports.ErrVersionCatalogoYaExiste
	}
	if canonico.Version > 1 {
		anterior, existe := s.catalogos[claveCatalogo(canonico.ID, canonico.Version-1)]
		if !existe || (anterior.Estado != domain.EstadoCatalogoPublicado && anterior.Estado != domain.EstadoCatalogoRetirado) ||
			canonico.VersionAnteriorRef != anterior.Referencia() {
			return ports.ErrSecuenciaCatalogoInvalida
		}
	}
	s.catalogos[clave] = canonico
	s.confirmarEvidenciaCatalogoBloqueado(traza, evento)
	return nil
}

func (s *Store) ConfirmarActualizacionBorradorCatalogo(ctx context.Context, huellaAnterior string, catalogo domain.CatalogoConfigurable, traza domain.AuditEntry, evento domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return err
	}
	if canonico.Estado != domain.EstadoCatalogoBorrador || canonico.Revision < 2 || strings.TrimSpace(huellaAnterior) == "" ||
		!evidenciaCatalogoValida(canonico, traza, evento, domain.AccionCatalogoBorradorActualizado, huellaAnterior) {
		return domain.ErrCatalogoConfigurableInvalido
	}
	clave := claveCatalogo(canonico.ID, canonico.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.catalogos[clave]
	if !existe {
		return ports.ErrCatalogoNoEncontrado
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || huellaActual != huellaAnterior || actual.Estado != domain.EstadoCatalogoBorrador ||
		canonico.Revision != actual.Revision+1 || !identidadVersionCatalogoIgual(actual, canonico) {
		return ports.ErrRevisionCatalogoEnConflicto
	}
	s.catalogos[clave] = canonico
	s.confirmarEvidenciaCatalogoBloqueado(traza, evento)
	return nil
}

func (s *Store) ConfirmarPublicacionCatalogo(ctx context.Context, huellaBorrador string, catalogo domain.CatalogoConfigurable, traza domain.AuditEntry, evento domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return err
	}
	if canonico.Estado != domain.EstadoCatalogoPublicado || strings.TrimSpace(huellaBorrador) == "" ||
		!evidenciaCatalogoValida(canonico, traza, evento, domain.AccionCatalogoPublicado, huellaBorrador) {
		return domain.ErrCatalogoConfigurableInvalido
	}
	base := canonico
	base.Estado = domain.EstadoCatalogoBorrador
	base.PublicadoPor = ""
	base.PublicadoEn = time.Time{}
	base.AprobacionRef = ""
	base.MotivoPublicacion = ""
	huellaBase, err := base.HuellaSHA256()
	if err != nil || huellaBase != huellaBorrador {
		return ports.ErrRevisionCatalogoEnConflicto
	}

	clave := claveCatalogo(canonico.ID, canonico.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.catalogos[clave]
	if !existe {
		return ports.ErrCatalogoNoEncontrado
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || actual.Estado != domain.EstadoCatalogoBorrador || huellaActual != huellaBorrador {
		return ports.ErrRevisionCatalogoEnConflicto
	}
	s.catalogos[clave] = canonico
	s.confirmarEvidenciaCatalogoBloqueado(traza, evento)
	return nil
}

func (s *Store) ConfirmarRetiradaCatalogo(ctx context.Context, huellaPublicada string, catalogo domain.CatalogoConfigurable, traza domain.AuditEntry, evento domain.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	canonico, err := catalogo.ClonarCanonico()
	if err != nil {
		return err
	}
	if canonico.Estado != domain.EstadoCatalogoRetirado || strings.TrimSpace(huellaPublicada) == "" ||
		!evidenciaCatalogoValida(canonico, traza, evento, domain.AccionCatalogoRetirado, huellaPublicada) {
		return domain.ErrCatalogoConfigurableInvalido
	}
	base := canonico
	base.Estado = domain.EstadoCatalogoPublicado
	base.RetiradoPor = ""
	base.RetiradoEn = time.Time{}
	base.RetiradaAprobacionRef = ""
	base.MotivoRetirada = ""
	huellaBase, err := base.HuellaSHA256()
	if err != nil || huellaBase != huellaPublicada {
		return ports.ErrRevisionCatalogoEnConflicto
	}

	clave := claveCatalogo(canonico.ID, canonico.Version)
	s.mu.Lock()
	defer s.mu.Unlock()
	actual, existe := s.catalogos[clave]
	if !existe {
		return ports.ErrCatalogoNoEncontrado
	}
	huellaActual, err := actual.HuellaSHA256()
	if err != nil || actual.Estado != domain.EstadoCatalogoPublicado || huellaActual != huellaPublicada {
		return ports.ErrRevisionCatalogoEnConflicto
	}
	s.catalogos[clave] = canonico
	s.confirmarEvidenciaCatalogoBloqueado(traza, evento)
	return nil
}

func (s *Store) ObtenerCatalogo(ctx context.Context, id string, version int) (domain.CatalogoConfigurable, error) {
	if err := ctx.Err(); err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	catalogo, existe := s.catalogos[claveCatalogo(id, version)]
	if !existe {
		return domain.CatalogoConfigurable{}, ports.ErrCatalogoNoEncontrado
	}
	return catalogo.ClonarCanonico()
}

func (s *Store) ListarVersionesCatalogo(ctx context.Context, id string) ([]domain.CatalogoConfigurable, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ports.ErrCatalogoNoEncontrado
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	resultado := make([]domain.CatalogoConfigurable, 0)
	for _, catalogo := range s.catalogos {
		if catalogo.ID == id {
			clon, err := catalogo.ClonarCanonico()
			if err != nil {
				return nil, err
			}
			resultado = append(resultado, clon)
		}
	}
	if len(resultado) == 0 {
		return nil, ports.ErrCatalogoNoEncontrado
	}
	sort.Slice(resultado, func(i, j int) bool { return resultado[i].Version < resultado[j].Version })
	return resultado, nil
}

func (s *Store) confirmarEvidenciaCatalogoBloqueado(traza domain.AuditEntry, evento domain.Event) {
	trazaConfirmada := s.appendAuditLocked(traza)
	evento.Payload = cloneStringMap(evento.Payload)
	if evento.Payload == nil {
		evento.Payload = map[string]string{}
	}
	evento.Payload["auditoria_ref"] = trazaConfirmada.ID
	s.appendEventLocked(evento)
}

func evidenciaCatalogoValida(catalogo domain.CatalogoConfigurable, traza domain.AuditEntry, evento domain.Event, accion, huellaAnterior string) bool {
	huella, err := catalogo.HuellaSHA256()
	if err != nil {
		return false
	}
	actor, fecha, regla, motivo := catalogo.CreadoPor, catalogo.CreadoEn, catalogo.FuenteRef, catalogo.MotivoCreacion
	switch accion {
	case domain.AccionCatalogoBorradorActualizado:
		actor, fecha, motivo = catalogo.UltimaModificacionPor, catalogo.UltimaModificacionEn, catalogo.MotivoModificacion
	case domain.AccionCatalogoPublicado:
		actor, fecha, regla, motivo = catalogo.PublicadoPor, catalogo.PublicadoEn, catalogo.AprobacionRef, catalogo.MotivoPublicacion
	case domain.AccionCatalogoRetirado:
		actor, fecha, regla, motivo = catalogo.RetiradoPor, catalogo.RetiradoEn, catalogo.RetiradaAprobacionRef, catalogo.MotivoRetirada
	case domain.AccionCatalogoBorradorCreado:
	default:
		return false
	}
	return traza.ActorID == actor && strings.TrimSpace(traza.ActorProfile) != "" &&
		traza.AuthMethod.Valido() && traza.AuthAssurance.Valida() && strings.TrimSpace(traza.AuthorizationRef) != "" &&
		strings.TrimSpace(traza.Purpose) != "" && traza.Action == accion && traza.ModuleID == catalogo.ModuloID &&
		traza.SubjectRef == catalogo.Referencia() && traza.ObjectVersion == catalogo.Version && traza.RuleRef == regla &&
		traza.Result == "correcto" && traza.BeforeHash == huellaAnterior && traza.AfterHash == huella &&
		traza.Reason == motivo && traza.Metadata["revision"] == strconv.Itoa(catalogo.Revision) &&
		strings.TrimSpace(traza.CorrelationRef) != "" && traza.OccurredAt.Equal(fecha) &&
		evento.Type == accion && evento.ModuleID == catalogo.ModuloID && evento.SubjectRef == catalogo.Referencia() &&
		evento.ActorID == actor && evento.OccurredAt.Equal(fecha) && evento.Payload["catalogo_id"] == catalogo.ID &&
		evento.Payload["catalogo_version"] == strconv.Itoa(catalogo.Version) &&
		evento.Payload["catalogo_revision"] == strconv.Itoa(catalogo.Revision) &&
		evento.Payload["estado"] == string(catalogo.Estado) && evento.Payload["huella_sha256"] == huella
}

func identidadVersionCatalogoIgual(anterior, posterior domain.CatalogoConfigurable) bool {
	return anterior.ID == posterior.ID && anterior.Version == posterior.Version &&
		anterior.VersionAnteriorRef == posterior.VersionAnteriorRef && anterior.ModuloID == posterior.ModuloID &&
		anterior.CreadoPor == posterior.CreadoPor && anterior.CreadoEn.Equal(posterior.CreadoEn) &&
		anterior.MotivoCreacion == posterior.MotivoCreacion
}

func claveCatalogo(id string, version int) string {
	return strings.TrimSpace(id) + ":" + strconv.Itoa(version)
}

var _ ports.ConsultaCatalogosConfigurables = (*Store)(nil)
var _ ports.RepositorioGobiernoCatalogos = (*Store)(nil)
