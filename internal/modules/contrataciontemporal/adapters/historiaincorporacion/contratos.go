// Package historiaincorporacion restaura historia CT mediante lecturas
// propietarias. Ninguna implementación de este paquete registra concesiones,
// firma, consume, consulta tablas cruzadas ni concede permiso vigente.
package historiaincorporacion

import (
	"context"
	"errors"
	"time"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaDocumento = "vec.contratacion-temporal.incorporacion.ejercicio.historia-original.v2"
const MaximoBytesDocumento = 32 << 20

var ErrHistoria = errors.New("contratacion temporal: historia original no recuperable")

// Selector procede de la fila CT conocida por el propietario, nunca browser.
type Selector struct{ ReciboRef, MaterialSHA256, IntencionSHA256 string }

// Documento es transporte ordinario cerrado. No contiene tipos nominales
// opacos y NO acredita procedencia por su forma/hash. Su lector es parte de TCB.
type Documento struct {
	Esquema             string
	Evidencia           ct.EvidenciaOrdenOriginalIncorporacionV2
	Recibo              ct.ReciboRegistroIncorporacionV2
	Publicacion         dom.PublicacionDefinicionSeguimiento
	Anterior, Posterior dom.EstadoPersistidoSeguimiento
}
type LectorRegistro interface {
	LeerRegistroOriginal(context.Context, Selector) ([]byte, error)
}

// Las autoridades siguientes deben releer versiones ORIGINALES, no reevaluar
// con instantánea actual. Las referencias/hash delimitan una lectura exacta.
type ReferenciaAutenticacion struct{ AutenticacionRef, SesionRef, HuellaSHA256 string }
type ReferenciaContexto struct{ RegistroRef, HuellaSHA256, ProcedenciaSHA256 string }
type ReferenciaEvaluacion struct{ DecisionRef, DecisionSHA256, SolicitudSHA256 string }
type LectorAutenticacion interface {
	LeerAutenticacionOriginal(context.Context, ReferenciaAutenticacion) (core.AutenticacionRevalidadaV1, error)
}
type LectorContexto interface {
	LeerContextoOriginal(context.Context, ReferenciaContexto) (core.ResultadoContextoActorRegistradoV2, error)
}
type LectorEvaluacion interface {
	LeerEvaluacionOriginal(context.Context, ReferenciaEvaluacion) (core.InstantaneaAutorizacion, error)
}

type Restauracion struct {
	OrdenOriginal ct.OrdenConfirmacionIncorporacionV2
	Historia      ct.HistoriaRegistroIncorporacionV2
}
type Restaurador struct {
	registros       LectorRegistro
	autenticaciones LectorAutenticacion
	contextos       LectorContexto
	evaluaciones    LectorEvaluacion
	concesiones     vp.LectorConcesionHistoricaAutorizacionLigadaV3
	reloj           ct.Reloj
}

// Validación typed-nil sin reflexión ni conversiones de tipos nominales.
// Las interfaces se verifican además por recuperación acotada de panic al entrar
// en Restaurar; una dependencia nil concreta nunca produce historia aceptada.
func NuevoRestaurador(r LectorRegistro, a LectorAutenticacion, c LectorContexto, e LectorEvaluacion, g vp.LectorConcesionHistoricaAutorizacionLigadaV3, reloj ct.Reloj) (*Restaurador, error) {
	if r == nil || a == nil || c == nil || e == nil || g == nil || reloj == nil {
		return nil, ErrHistoria
	}
	return &Restaurador{r, a, c, e, g, reloj}, nil
}

type relojOriginal struct{ t time.Time }

func (r relojOriginal) Ahora() time.Time { return r.t }

type autenticacionHistorica struct {
	l   LectorAutenticacion
	ref ReferenciaAutenticacion
}

func (a autenticacionHistorica) RevalidarAutenticacionActorV1(ctx context.Context, _ core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	return a.l.LeerAutenticacionOriginal(ctx, a.ref)
}

type contextoHistorico struct {
	l   LectorContexto
	ref ReferenciaContexto
}

func (c contextoHistorico) ResolverContextoActorRegistradoV2(ctx context.Context, _ core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	return c.l.LeerContextoOriginal(ctx, c.ref)
}

// Sólo conserva la correlación del registro releído: no emite una nueva.
type correlacionHistorica string

func (c correlacionHistorica) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return string(c), nil
}
