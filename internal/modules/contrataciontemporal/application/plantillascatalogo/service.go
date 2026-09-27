package plantillascatalogo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	CatalogoID = "vec.contratacion_temporal.plantillas_documentos"
	ModuloID   = "contratacion_temporal"
)

var (
	ErrNoDisponible    = errors.New("contratacion temporal: catalogo de plantillas no disponible")
	ErrEntradaInvalida = errors.New("contratacion temporal: entrada de plantilla invalida")
	ErrConflicto       = errors.New("contratacion temporal: version o clave de plantilla en conflicto")
)

// El validador lo aporta el adaptador documental; application no importa PDF ni DOCX.
type Validador func(vecdomain.CatalogoConfigurable) error

type Reloj interface{ Ahora() time.Time }

// Repositorio consume autorización V3 nominal incluso para lectura y replay.
// La implementación PostgreSQL ejecuta el consumo dentro de la operación SQL.
type Repositorio interface {
	Consultar(context.Context, vecdomain.ContextoActor) (Lectura, error)
	Cambiar(context.Context, vecdomain.ContextoActor, MaterialCambio) (ResultadoCambio, error)
	ComprobarAccion(context.Context, vecdomain.ContextoActor, string, Lectura) (bool, error)
}

type Lectura struct {
	Borrador            *vecdomain.CatalogoConfigurable `json:"borrador"`
	Publicado           *vecdomain.CatalogoConfigurable `json:"publicado"`
	PuedeEditar         bool                            `json:"puede_editar"`
	PuedePublicar       bool                            `json:"puede_publicar"`
	EditorDeEstaVersion bool                            `json:"editor_de_esta_version"`
}

type Recibo struct {
	ReciboRef    string    `json:"recibo_ref"`
	RegistradoEn time.Time `json:"registrado_en"`
	EstadoReplay string    `json:"estado_replay"`
}

type ResultadoCambio struct {
	Catalogo vecdomain.CatalogoConfigurable `json:"catalogo"`
	Recibo   Recibo                         `json:"recibo"`
}

type SolicitudEditar struct {
	ClaveIdempotencia string                                `json:"clave_idempotencia"`
	VersionEsperada   int                                   `json:"version_esperada"`
	RevisionEsperada  int                                   `json:"revision_esperada"`
	Motivo            string                                `json:"motivo"`
	FuenteRef         string                                `json:"fuente_ref"`
	Entrada           vecdomain.EntradaCatalogoConfigurable `json:"entrada"`
}

type SolicitudPublicar struct {
	ClaveIdempotencia string `json:"clave_idempotencia"`
	VersionEsperada   int    `json:"version_esperada"`
	RevisionEsperada  int    `json:"revision_esperada"`
	Motivo            string `json:"motivo"`
	AprobacionRef     string `json:"aprobacion_ref"`
}

// MaterialCambio conserva la solicitud original para verificar un replay
// aunque la cabeza del catálogo ya haya avanzado. El catálogo propuesto puede
// ser nil únicamente cuando la versión esperada ya está en conflicto; SQL
// resolverá la clave idempotente o devolverá 409 sin efecto.
type MaterialCambio struct {
	Operacion            string                          `json:"operacion"`
	ClaveIdempotencia    string                          `json:"clave_idempotencia"`
	VersionEsperada      int                             `json:"version_esperada"`
	RevisionEsperada     int                             `json:"revision_esperada"`
	Solicitud            json.RawMessage                 `json:"solicitud"`
	Catalogo             *vecdomain.CatalogoConfigurable `json:"catalogo"`
	CatalogoHuellaSHA256 string                          `json:"catalogo_huella_sha256,omitempty"`
	BaseHuellaSHA256     string                          `json:"catalogo_base_huella_sha256,omitempty"`
}

type Servicio struct {
	repo    Repositorio
	reloj   Reloj
	validar Validador
	inicial *vecdomain.CatalogoConfigurable
}

func NuevoServicio(repo Repositorio, reloj Reloj, validar Validador, inicial *vecdomain.CatalogoConfigurable) (*Servicio, error) {
	if repo == nil || reloj == nil || validar == nil || inicial == nil {
		return nil, ErrNoDisponible
	}
	c, err := inicial.ClonarCanonico()
	if err != nil || c.ID != CatalogoID || c.ModuloID != ModuloID || c.Estado != vecdomain.EstadoCatalogoPublicado || validar(c) != nil {
		return nil, ErrNoDisponible
	}
	return &Servicio{repo: repo, reloj: reloj, validar: validar, inicial: &c}, nil
}

func (s *Servicio) Consultar(ctx context.Context, actor vecdomain.ContextoActor) (Lectura, error) {
	if s == nil || ctx == nil || actor.Validar() != nil {
		return Lectura{}, ErrNoDisponible
	}
	lectura, err := s.repo.Consultar(ctx, actor)
	if err != nil {
		return Lectura{}, err
	}
	if err = validarLectura(lectura, s.validar); err != nil {
		return Lectura{}, err
	}
	if lectura.Publicado == nil {
		return Lectura{}, ErrNoDisponible
	}
	if lectura.Publicado.Version == s.inicial.Version {
		actual, _ := lectura.Publicado.HuellaSHA256()
		esperada, _ := s.inicial.HuellaSHA256()
		if actual != esperada {
			return Lectura{}, ErrNoDisponible
		}
	}
	// Estos indicadores son proyecciones positivas del PDP para la interfaz.
	// La decisión de escritura se emite de nuevo para el material exacto y se
	// consume dentro de PostgreSQL; la consulta no concede permiso por sí sola.
	lectura.PuedeEditar, _ = s.repo.ComprobarAccion(ctx, actor, "editar", lectura)
	if lectura.Borrador != nil && !lectura.EditorDeEstaVersion && actor.Principal.ID != lectura.Borrador.CreadoPor &&
		actor.Principal.ID != lectura.Borrador.UltimaModificacionPor {
		lectura.PuedePublicar, _ = s.repo.ComprobarAccion(ctx, actor, "publicar", lectura)
	}
	return lectura, nil
}

func (s *Servicio) Editar(ctx context.Context, actor vecdomain.ContextoActor, solicitud SolicitudEditar) (ResultadoCambio, error) {
	if s == nil || ctx == nil || actor.Validar() != nil || !claveUUID(solicitud.ClaveIdempotencia) || solicitud.VersionEsperada < 0 || solicitud.RevisionEsperada < 0 ||
		!textoMotivo(solicitud.Motivo) || !textoFuente(solicitud.FuenteRef) || solicitud.Entrada.Validar() != nil || solicitud.Entrada.Clave == "etiquetas" {
		return ResultadoCambio{}, ErrEntradaInvalida
	}
	// El dominio normaliza las fechas antes de persistir el catálogo. La
	// solicitud idempotente usa la misma forma para que SQL compare exactamente
	// la entrada autorizada y el replay conserve su huella.
	solicitud.Entrada.VigenteDesde = solicitud.Entrada.VigenteDesde.UTC()
	if !solicitud.Entrada.VigenteHasta.IsZero() {
		solicitud.Entrada.VigenteHasta = solicitud.Entrada.VigenteHasta.UTC()
	}
	lectura, err := s.Consultar(ctx, actor)
	if err != nil {
		return ResultadoCambio{}, err
	}
	material, err := nuevoMaterial("editar", solicitud.ClaveIdempotencia, solicitud.VersionEsperada, solicitud.RevisionEsperada, solicitud)
	if err != nil {
		return ResultadoCambio{}, err
	}
	actual := lectura.Borrador
	if actual != nil && (actual.Version != solicitud.VersionEsperada || actual.Revision != solicitud.RevisionEsperada) ||
		actual == nil && lectura.Publicado != nil && (lectura.Publicado.Version != solicitud.VersionEsperada || solicitud.RevisionEsperada != 0) ||
		actual == nil && lectura.Publicado == nil && (solicitud.VersionEsperada != 0 || solicitud.RevisionEsperada != 0) {
		return s.repo.Cambiar(ctx, actor, material)
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() {
		return ResultadoCambio{}, ErrNoDisponible
	}
	var siguiente vecdomain.CatalogoConfigurable
	if actual != nil {
		entradas := append([]vecdomain.EntradaCatalogoConfigurable(nil), actual.Entradas...)
		encontrada := false
		for i := range entradas {
			if entradas[i].Clave == solicitud.Entrada.Clave {
				entradas[i] = solicitud.Entrada
				encontrada = true
				break
			}
		}
		if !encontrada {
			entradas = append(entradas, solicitud.Entrada)
		}
		siguiente, err = actual.ActualizarBorrador(actual.Revision, actor.Principal.ID, actual.Nombre, actual.Descripcion, solicitud.FuenteRef, solicitud.Motivo, entradas, ahora)
	} else if lectura.Publicado != nil {
		siguiente, err = lectura.Publicado.NuevaVersion(lectura.Publicado.Version+1, actor.Principal.ID, solicitud.FuenteRef, solicitud.Motivo, ahora)
		if lectura.Publicado.Version == s.inicial.Version {
			material.BaseHuellaSHA256, _ = lectura.Publicado.HuellaSHA256()
		}
		if err == nil {
			encontrada := false
			for i := range siguiente.Entradas {
				if siguiente.Entradas[i].Clave == solicitud.Entrada.Clave {
					siguiente.Entradas[i] = solicitud.Entrada
					encontrada = true
					break
				}
			}
			if !encontrada {
				siguiente.Entradas = append(siguiente.Entradas, solicitud.Entrada)
			}
		}
	} else {
		// El catálogo publicado se provisiona por el canal migrador antes de HTTP.
		return ResultadoCambio{}, ErrNoDisponible
	}
	if err != nil {
		return ResultadoCambio{}, ErrEntradaInvalida
	}
	if err = s.validar(siguiente); err != nil {
		return ResultadoCambio{}, ErrEntradaInvalida
	}
	material.Catalogo = &siguiente
	material.CatalogoHuellaSHA256, _ = siguiente.HuellaSHA256()
	return s.repo.Cambiar(ctx, actor, material)
}

func (s *Servicio) Publicar(ctx context.Context, actor vecdomain.ContextoActor, solicitud SolicitudPublicar) (ResultadoCambio, error) {
	if s == nil || ctx == nil || actor.Validar() != nil || !claveUUID(solicitud.ClaveIdempotencia) || solicitud.VersionEsperada < 1 || solicitud.RevisionEsperada < 1 ||
		!textoMotivo(solicitud.Motivo) || !textoFuente(solicitud.AprobacionRef) {
		return ResultadoCambio{}, ErrEntradaInvalida
	}
	lectura, err := s.Consultar(ctx, actor)
	if err != nil {
		return ResultadoCambio{}, err
	}
	material, err := nuevoMaterial("publicar", solicitud.ClaveIdempotencia, solicitud.VersionEsperada, solicitud.RevisionEsperada, solicitud)
	if err != nil {
		return ResultadoCambio{}, err
	}
	if lectura.Borrador == nil || lectura.Borrador.Version != solicitud.VersionEsperada || lectura.Borrador.Revision != solicitud.RevisionEsperada {
		return s.repo.Cambiar(ctx, actor, material)
	}
	if lectura.EditorDeEstaVersion || actor.Principal.ID == lectura.Borrador.CreadoPor || actor.Principal.ID == lectura.Borrador.UltimaModificacionPor {
		return ResultadoCambio{}, vecdomain.ErrAutorizacionDenegada
	}
	publicado, err := lectura.Borrador.Publicar(actor.Principal.ID, solicitud.AprobacionRef, solicitud.Motivo, s.reloj.Ahora().UTC().Truncate(time.Microsecond))
	if err != nil || s.validar(publicado) != nil {
		return ResultadoCambio{}, ErrEntradaInvalida
	}
	material.Catalogo = &publicado
	material.CatalogoHuellaSHA256, _ = publicado.HuellaSHA256()
	return s.repo.Cambiar(ctx, actor, material)
}

func nuevoMaterial(operacion, clave string, version, revision int, solicitud any) (MaterialCambio, error) {
	b, err := json.Marshal(solicitud)
	if err != nil || len(b) > 256<<10 {
		return MaterialCambio{}, ErrEntradaInvalida
	}
	return MaterialCambio{Operacion: operacion, ClaveIdempotencia: clave, VersionEsperada: version, RevisionEsperada: revision, Solicitud: b}, nil
}

func validarLectura(l Lectura, v Validador) error {
	for _, c := range []*vecdomain.CatalogoConfigurable{l.Borrador, l.Publicado} {
		if c == nil {
			continue
		}
		if c.ID != CatalogoID || c.ModuloID != ModuloID || c.Validar() != nil || v(*c) != nil {
			return ErrNoDisponible
		}
	}
	if l.Borrador != nil && l.Borrador.Estado != vecdomain.EstadoCatalogoBorrador {
		return ErrNoDisponible
	}
	if l.Publicado != nil && l.Publicado.Estado != vecdomain.EstadoCatalogoPublicado {
		return ErrNoDisponible
	}
	if l.Borrador != nil && l.Publicado != nil && l.Borrador.Version != l.Publicado.Version+1 {
		return ErrNoDisponible
	}
	return nil
}

func claveUUID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, c := range v {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func textoMotivo(v string) bool { return v != "" && v == strings.TrimSpace(v) && len(v) <= 1024 }
func textoFuente(v string) bool { return v != "" && v == strings.TrimSpace(v) && len(v) <= 512 }
