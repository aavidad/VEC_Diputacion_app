package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// RepositorioMarcajesRemotos implementa la consulta de teletrabajo y el
// repositorio del fichaje remoto sobre cronos_v1 000007. Cada lectura del
// estado consume su propia decisión V3 de disponibilidad; el registro usa la
// decisión de marcaje propio y la recuperación la de recibo. El canal es el
// acreditado por la política configurada en la composición, nunca del cliente.
type RepositorioMarcajesRemotos struct {
	db        iniciadorMarcaje
	marcajes  *RepositorioMarcajes
	proveedor ports.ProveedorMaterialDisponibilidadMarcajeRemoto
	canal     domain.AcreditacionCanalMarcaje
}

func NuevoRepositorioMarcajesRemotos(pool *pgxpool.Pool, auditoria ports.RegistroResultadoEjecucionMarcaje, proveedor ports.ProveedorMaterialDisponibilidadMarcajeRemoto, canal domain.AcreditacionCanalMarcaje) (*RepositorioMarcajesRemotos, error) {
	if pool == nil || auditoria == nil || dependenciaPostgresNula(proveedor) || canal.Validar() != nil || canal.OrigenRef() != domain.OrigenMarcajeRemoto {
		return nil, ports.ErrDependenciaNoDisponible
	}
	return &RepositorioMarcajesRemotos{
		db:        pool,
		marcajes:  &RepositorioMarcajes{db: pool, auditoria: auditoria, consulta: consultaMarcajeRemoto},
		proveedor: proveedor, canal: canal,
	}, nil
}

type claveEstadoRemoto struct{}

// estadoRemotoPeticion guarda, sólo durante una petición, el estado leído al
// consultar el teletrabajo. La continuidad sin clave se sirve de él para no
// consumir una segunda decisión; con clave siempre se relee bajo bloqueo.
type estadoRemotoPeticion struct {
	mu       sync.Mutex
	listo    bool
	persona  string
	perfil   string
	empleado string
	instante time.Time
	estado   estadoRemotoSQL
}

// ContextoConEstadoRemoto prepara el ámbito de una única petición HTTP. La
// composición lo instala antes del manejador; sin él la continuidad falla
// cerrada.
func ContextoConEstadoRemoto(ctx context.Context) context.Context {
	return context.WithValue(ctx, claveEstadoRemoto{}, &estadoRemotoPeticion{})
}

type estadoRemotoSQL struct {
	Autorizado            *bool              `json:"autorizado"`
	Desde                 *time.Time         `json:"desde"`
	Hasta                 *time.Time         `json:"hasta"`
	ContinuidadConfirmada *bool              `json:"continuidad_confirmada"`
	MovimientosPermitidos []domain.PunchKind `json:"movimientos_permitidos"`
}

func (r *RepositorioMarcajesRemotos) ConsultarTeletrabajoPropio(ctx context.Context, actor vecdomain.ContextoActor, empleado string, instante time.Time) (domain.PeriodoTeletrabajo, bool, error) {
	cache, ok := cacheEstadoRemoto(ctx)
	if !ok {
		return domain.PeriodoTeletrabajo{}, false, ports.ErrDependenciaNoDisponible
	}
	estado, err := r.consultarEstado(ctx, actor, empleado, instante, "")
	if err != nil {
		return domain.PeriodoTeletrabajo{}, false, err
	}
	cache.mu.Lock()
	cache.listo, cache.persona, cache.perfil, cache.empleado, cache.instante, cache.estado = true, actor.PersonaRef, actor.PerfilActivoRef, empleado, instante, estado
	cache.mu.Unlock()
	if !*estado.Autorizado {
		return domain.PeriodoTeletrabajo{}, false, nil
	}
	return domain.PeriodoTeletrabajo{DesdeUTC: estado.Desde.UTC(), HastaUTC: estado.Hasta.UTC()}, true, nil
}

func (r *RepositorioMarcajesRemotos) ConfirmarContinuidadMarcajeRemoto(ctx context.Context, actor vecdomain.ContextoActor, empleado string, periodo domain.PeriodoTeletrabajo, clave string) (ports.EstadoSecuenciaMarcajeRemoto, error) {
	cache, ok := cacheEstadoRemoto(ctx)
	if !ok {
		return ports.EstadoSecuenciaMarcajeRemoto{}, ports.ErrDependenciaNoDisponible
	}
	cache.mu.Lock()
	listo, persona, perfil, previo, instante, estado := cache.listo, cache.persona, cache.perfil, cache.empleado, cache.instante, cache.estado
	cache.mu.Unlock()
	if !listo || persona != actor.PersonaRef || perfil != actor.PerfilActivoRef || previo != empleado || !*estado.Autorizado ||
		!estado.Desde.Equal(periodo.DesdeUTC) || !estado.Hasta.Equal(periodo.HastaUTC) {
		return ports.EstadoSecuenciaMarcajeRemoto{}, ports.ErrDependenciaNoDisponible
	}
	if clave != "" {
		var err error
		estado, err = r.consultarEstado(ctx, actor, empleado, instante, clave)
		if err != nil {
			return ports.EstadoSecuenciaMarcajeRemoto{}, err
		}
		if !*estado.Autorizado || !estado.Desde.Equal(periodo.DesdeUTC) || !estado.Hasta.Equal(periodo.HastaUTC) {
			return ports.EstadoSecuenciaMarcajeRemoto{}, ports.ErrDependenciaNoDisponible
		}
	}
	return ports.EstadoSecuenciaMarcajeRemoto{ContinuidadConfirmada: *estado.ContinuidadConfirmada, MovimientosPermitidos: append([]domain.PunchKind{}, estado.MovimientosPermitidos...)}, nil
}

func (r *RepositorioMarcajesRemotos) consultarEstado(ctx context.Context, actor vecdomain.ContextoActor, empleado string, instante time.Time, clave string) (estadoRemotoSQL, error) {
	if r == nil || r.db == nil || dependenciaPostgresNula(r.proveedor) || ctx == nil {
		return estadoRemotoSQL{}, ports.ErrDependenciaNoDisponible
	}
	material := domain.MaterialDisponibilidadMarcajeRemoto{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ClaveOperacion: clave, InstanteUTC: instante.UTC().Truncate(time.Microsecond), Canal: r.canal}
	canonico, err := material.Canonico()
	if err != nil {
		return estadoRemotoSQL{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoDisponibilidadMarcajeRemoto(material)
	if err != nil {
		return estadoRemotoSQL{}, ports.ErrDependenciaNoDisponible
	}
	v3, err := r.proveedor.ProveerMaterialDisponibilidadMarcajeRemoto(ctx, material)
	if err != nil {
		return estadoRemotoSQL{}, errorProveedorV3(ctx, err)
	}
	if !resumenV3Ligado(v3, application.AudienciaDisponibilidadMarcajeRemoto, application.AccionConsultarDisponibilidadRemota, recurso) {
		return estadoRemotoSQL{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarLecturaV3(ctx, r.db, consultaEstadoRemotoPropio, canonico, v3)
	if err != nil {
		return estadoRemotoSQL{}, err
	}
	defer clear(bruto)
	return decodificarEstadoRemoto(bruto)
}

func decodificarEstadoRemoto(bruto []byte) (estadoRemotoSQL, error) {
	var e estadoRemotoSQL
	if len(bruto) == 0 || len(bruto) > 4096 {
		return estadoRemotoSQL{}, ports.ErrDependenciaNoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()
	if dec.Decode(&e) != nil || dec.Decode(&struct{}{}) != io.EOF || e.Autorizado == nil || e.ContinuidadConfirmada == nil ||
		e.MovimientosPermitidos == nil || domain.ValidarMovimientosRemotosPermitidos(e.MovimientosPermitidos) != nil {
		return estadoRemotoSQL{}, ports.ErrDependenciaNoDisponible
	}
	if !*e.Autorizado {
		if e.Desde != nil || e.Hasta != nil || *e.ContinuidadConfirmada || len(e.MovimientosPermitidos) != 0 {
			return estadoRemotoSQL{}, ports.ErrDependenciaNoDisponible
		}
		return e, nil
	}
	if e.Desde == nil || e.Hasta == nil || !e.Desde.Before(*e.Hasta) || (!*e.ContinuidadConfirmada && len(e.MovimientosPermitidos) != 0) {
		return estadoRemotoSQL{}, ports.ErrDependenciaNoDisponible
	}
	desde, hasta := e.Desde.UTC(), e.Hasta.UTC()
	e.Desde, e.Hasta = &desde, &hasta
	return e, nil
}

func (r *RepositorioMarcajesRemotos) RegistrarOriginalRemotoAutorizado(ctx context.Context, m domain.MarcajeOriginal, material domain.MaterialAutorizacionMarcajePropio, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMarcajePropio, error) {
	if r == nil || r.marcajes == nil || m.CanalAcreditado != r.canal || material.Canal != r.canal {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	return r.marcajes.RegistrarOriginalAutorizado(ctx, m, material, v3)
}

// RecuperarOriginalRemotoAutorizado sólo afirma ausencia tras el COMMIT de su
// evidencia; la función SQL espera antes al escritor de la misma clave.
func (r *RepositorioMarcajesRemotos) RecuperarOriginalRemotoAutorizado(ctx context.Context, material domain.MaterialRecuperacionMarcajeRemoto, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboMarcajePropio, error) {
	if r == nil || r.db == nil || ctx == nil || material.Canal != r.canal {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	canonico, err := material.Canonico()
	if err != nil {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	recurso, err := application.RecursoRecuperacionMarcajeRemoto(material)
	if err != nil || !resumenV3Ligado(v3, application.AudienciaRecuperacionMarcajeRemoto, application.AccionRecuperarMarcajeRemoto, recurso) {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	bruto, err := ejecutarLecturaV3(ctx, r.db, consultaRecuperarRemoto, canonico, v3)
	if err != nil {
		return ports.ReciboMarcajePropio{}, err
	}
	defer clear(bruto)
	if ausenciaConfirmada(bruto) {
		return ports.ReciboMarcajePropio{}, ports.ErrMarcajeRemotoNoEncontrado
	}
	var recibido reciboSQL
	if decodificarRecibo(bruto, &recibido) != nil || !referenciaRecibo.MatchString(recibido.Referencia) || recibido.Replay == nil || !*recibido.Replay ||
		recibido.MarcajeOriginalRef != "marcaje:cronos:"+material.ClaveOperacion || recibido.InstanteUTC.IsZero() {
		return ports.ReciboMarcajePropio{}, ports.ErrDependenciaNoDisponible
	}
	return ports.ReciboMarcajePropio{Referencia: recibido.Referencia, InstanteUTC: recibido.InstanteUTC.UTC(), MarcajeOriginalRef: recibido.MarcajeOriginalRef, Replay: true}, nil
}

func ausenciaConfirmada(bruto []byte) bool {
	var a struct {
		Ausente *bool `json:"ausente"`
	}
	dec := json.NewDecoder(bytes.NewReader(bruto))
	dec.DisallowUnknownFields()
	return len(bruto) <= 64 && dec.Decode(&a) == nil && dec.Decode(&struct{}{}) == io.EOF && a.Ausente != nil && *a.Ausente
}

func dependenciaPostgresNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

func cacheEstadoRemoto(ctx context.Context) (*estadoRemotoPeticion, bool) {
	if ctx == nil {
		return nil, false
	}
	c, ok := ctx.Value(claveEstadoRemoto{}).(*estadoRemotoPeticion)
	return c, ok && c != nil
}

var (
	_ ports.RepositorioMarcajesRemotos      = (*RepositorioMarcajesRemotos)(nil)
	_ ports.ConsultaAutorizacionTeletrabajo = (*RepositorioMarcajesRemotos)(nil)
)
