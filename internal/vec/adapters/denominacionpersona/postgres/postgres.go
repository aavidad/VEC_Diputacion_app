// Package postgres implementa CA32; no emite permisos ni descifra nombres.
package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"reflect"
	"strconv"
	"time"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrNoDisponible = errors.New("vec.persona.denominacion.postgres.no_disponible")

type conexion interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type Adaptador struct {
	pool      conexion
	protector ports.ProtectorDenominacionPersona
}

var _ ports.RegistroDenominacionPersona = (*Adaptador)(nil)
var _ ports.FuenteDenominacionPersonaAutorizada = (*Adaptador)(nil)

// Nuevo acredita el LOGIN mínimo mediante SQL. Una fachada ausente o una
// dependencia AD/AUT no publicada cierra el constructor; no se conceden ACL.
func Nuevo(ctx context.Context, pool *pgxpool.Pool, protector ports.ProtectorDenominacionPersona) (*Adaptador, error) {
	return nuevo(ctx, pool, protector)
}
func nuevo(ctx context.Context, pool conexion, protector ports.ProtectorDenominacionPersona) (*Adaptador, error) {
	if ctx == nil || nulo(pool) || nulo(protector) {
		return nil, ErrNoDisponible
	}
	a := &Adaptador{pool, protector}
	var valido bool
	if a.transaccion(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT vec_contexto_actor_v1.acreditar_runtime_denominacion_persona_v1()`).Scan(&valido)
	}) != nil || !valido {
		return nil, ErrNoDisponible
	}
	return a, nil
}
func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}
func (a *Adaptador) transaccion(ctx context.Context, usar func(pgx.Tx) error) error {
	if a == nil || ctx == nil || nulo(a.pool) || usar == nil || ctx.Err() != nil {
		return ErrNoDisponible
	}
	tx, e := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if e != nil || nulo(tx) {
		return ErrNoDisponible
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(c)
	}()
	if _, e = tx.Exec(ctx, `SET LOCAL timezone='UTC'; SET LOCAL statement_timeout='15s'; SET LOCAL idle_in_transaction_session_timeout='20s'; SET LOCAL lock_timeout='2s'`); e != nil {
		return ErrNoDisponible
	}
	e = usar(tx)
	if ctx.Err() != nil {
		return ErrNoDisponible
	}
	if e != nil {
		// Sólo el rechazo explícito del cuerpo SQL es una denegación.
		// BEGIN, preparación y COMMIT conservan resultado no disponible.
		var pg *pgconn.PgError
		if errors.As(e, &pg) && pg.Code == "42501" {
			return domain.ErrAutorizacionDenegada
		}
		return ErrNoDisponible
	}
	if e = tx.Commit(ctx); e != nil {
		return ErrNoDisponible
	}
	return nil
}

const argumentos = `($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const argumentosComprobacion = `($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea,$12::text)`

func argumentosMaterial(raw []byte, a ports.AccesoDenominacionPersona) []any {
	m := a.Material
	return []any{string(raw), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
}
func borrarArgumentos(args []any) {
	for _, v := range args {
		if b, ok := v.([]byte); ok {
			clear(b)
		}
	}
}
func validarAcceso(a ports.AccesoDenominacionPersona, r domain.RecursoAutorizable, accion string) error {
	h, e := a.Contexto.HuellaSHA256VinculadaV2()
	rh, er := r.HuellaContextoAutorizacionSHA256()
	ah, ea := a.Recurso.HuellaContextoAutorizacionSHA256()
	if e != nil || er != nil || ea != nil || rh != ah || a.ResultadoContexto.Validar() != nil || a.ResultadoContexto.HuellaSHA256 != h || a.Vinculo.ValidarPara(a.ResultadoContexto) != nil || a.Material.ValidarEstructura() != nil || a.Recurso.Referencia != r.Referencia || a.Recurso.ModuloID != modulo || a.Recurso.Tipo != tipo || a.FinalidadRef != finalidad || a.Material.PersonaVersion() != a.Contexto.Instantanea.PersonaVersion || a.Material.PerfilVersion() != a.Contexto.Instantanea.PerfilVersion || !domain.ReferenciaCorrelacionAutorizacionV2Valida(a.Auditoria.CorrelationRef) {
		return ErrNoDisponible
	}
	s := a.Material.ResumenCapacidad()
	if s.Operacion() != accion || s.EfectoRef() != r.Referencia || s.EfectoHuellaSHA256() != rh || a.Audiencia != accion+".v1" || s.AudienciaConsumo() != a.Audiencia || s.ContextoRef() != a.ResultadoContexto.RegistroContextoRef {
		return ErrNoDisponible
	}
	var d struct {
		CorrelacionRef string `json:"correlacion_ref"`
	}
	if json.Unmarshal(a.Material.DecisionCanonica(), &d) != nil || d.CorrelacionRef != a.Auditoria.CorrelationRef {
		return ErrNoDisponible
	}
	return nil
}
func materialLectura(a ports.AccesoDenominacionPersona) ([]byte, error) {
	b, r, e := MaterialLectura(a.PersonaRef, a.Version, a.Recurso.Ambitos["organizacion_ref"], a.Recurso.Ambitos["unidad_ref"])
	if e != nil || validarAcceso(a, r, ports.AccionLeerDenominacionPersona) != nil {
		return nil, ErrNoDisponible
	}
	return b, nil
}
func decodificar(raw []byte, destino any) error {
	if len(raw) < 2 || len(raw) > 65536 {
		return ErrNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil || d.Decode(new(any)) != io.EOF {
		return ErrNoDisponible
	}
	return nil
}

type reciboJSON struct {
	PersonaRef     string `json:"persona_ref"`
	ProcedenciaRef string `json:"procedencia_ref"`
	SobreSHA256    string `json:"sobre_sha256"`
	AuditoriaRef   string `json:"auditoria_ref"`
	Version        uint64 `json:"version"`
}
type acuseJSON struct {
	ConsumoRef     string `json:"consumo_ref"`
	AuditoriaRef   string `json:"auditoria_ref"`
	DecisionRef    string `json:"decision_ref"`
	CorrelacionRef string `json:"correlacion_ref"`
	RecursoRef     string `json:"recurso_ref"`
	PersonaRef     string `json:"persona_ref"`
	SobreSHA256    string `json:"sobre_sha256"`
	Version        uint64 `json:"version"`
}

func acuseWire(d domain.DatosAcuseConsumoDenominacionPersona) acuseJSON {
	return acuseJSON{d.ConsumoRef, d.AuditoriaRef, d.DecisionRef, d.CorrelacionRef, d.RecursoRef, d.PersonaRef, d.SobreSHA256, d.Version}
}
func (x acuseJSON) datos() domain.DatosAcuseConsumoDenominacionPersona {
	return domain.DatosAcuseConsumoDenominacionPersona{ConsumoRef: x.ConsumoRef, AuditoriaRef: x.AuditoriaRef, DecisionRef: x.DecisionRef, CorrelacionRef: x.CorrelacionRef, RecursoRef: x.RecursoRef, PersonaRef: x.PersonaRef, SobreSHA256: x.SobreSHA256, Version: x.Version}
}
func (a *Adaptador) PublicarDenominacionPersona(ctx context.Context, o ports.OrdenDenominacionPersona) (ports.ReciboDenominacionPersona, error) {
	p := o.Preparacion
	pv, parseErr := strconv.ParseUint(o.Acceso.Recurso.Atributos["procedencia_version"], 10, 64)
	if parseErr != nil {
		return ports.ReciboDenominacionPersona{}, ErrNoDisponible
	}
	proc := Procedencia{Ref: p.ProcedenciaRef, Version: pv, HuellaSHA256: o.Acceso.Recurso.Atributos["procedencia_sha256"], Autoridad: o.Acceso.Recurso.Atributos["procedencia_autoridad"]}
	b, r, e := MaterialPublicacion(p, proc, o.Acceso.Recurso.Ambitos["organizacion_ref"], o.Acceso.Recurso.Ambitos["unidad_ref"])
	if a == nil || ctx == nil || e != nil || o.Acceso.PersonaRef != p.PersonaRef || o.Acceso.Version != p.Sobre.Version || validarAcceso(o.Acceso, r, ports.AccionPublicarDenominacionPersona) != nil || nulo(a.protector) || a.protector.RevalidarProteccionDenominacionPersona(ctx, p.Sobre) != nil {
		return ports.ReciboDenominacionPersona{}, ErrNoDisponible
	}
	args := argumentosMaterial(b, o.Acceso)
	defer borrarArgumentos(args)
	var x reciboJSON
	e = a.transaccion(ctx, func(tx pgx.Tx) error {
		var raw []byte
		if er := tx.QueryRow(ctx, `SELECT vec_contexto_actor_v1.publicar_denominacion_persona_v1`+argumentos, args...).Scan(&raw); er != nil {
			return er
		}
		if decodificar(raw, &x) != nil || x.PersonaRef != p.PersonaRef || x.ProcedenciaRef != p.ProcedenciaRef || x.Version != p.Sobre.Version || x.SobreSHA256 != p.SobreSHA256 || !referencia(x.AuditoriaRef) {
			return ErrNoDisponible
		}
		return nil
	})
	return resultadoPublicacion(x, e)
}

// Un recibo provisional nunca sale cuando el COMMIT no quedó confirmado.
// La clasificación se limita al rechazo nominal que transaccion ya identificó.
func resultadoPublicacion(x reciboJSON, err error) (ports.ReciboDenominacionPersona, error) {
	if err != nil {
		if errors.Is(err, domain.ErrAutorizacionDenegada) {
			return ports.ReciboDenominacionPersona{}, domain.ErrAutorizacionDenegada
		}
		return ports.ReciboDenominacionPersona{}, ErrNoDisponible
	}
	return ports.ReciboDenominacionPersona{PersonaRef: x.PersonaRef, ProcedenciaRef: x.ProcedenciaRef, SobreSHA256: x.SobreSHA256, AuditoriaRef: x.AuditoriaRef, Version: x.Version}, nil
}
func (a *Adaptador) LeerDenominacionPersonaAutorizada(ctx context.Context, acceso ports.AccesoDenominacionPersona) (ports.LecturaDenominacionPersonaConfirmada, error) {
	b, e := materialLectura(acceso)
	if e != nil {
		return ports.LecturaDenominacionPersonaConfirmada{}, ErrNoDisponible
	}
	args := argumentosMaterial(b, acceso)
	defer borrarArgumentos(args)
	var x struct {
		Sobre ports.SobreDenominacionPersona `json:"sobre"`
		Acuse acuseJSON                      `json:"acuse"`
	}
	var acuse domain.AcuseConsumoDenominacionPersona
	e = a.transaccion(ctx, func(tx pgx.Tx) error {
		var raw []byte
		if er := tx.QueryRow(ctx, `SELECT vec_contexto_actor_v1.leer_denominacion_persona_v1`+argumentos, args...).Scan(&raw); er != nil {
			return er
		}
		if decodificar(raw, &x) != nil {
			return ErrNoDisponible
		}
		s, er := sobreCanonico(x.Sobre)
		if er != nil {
			return er
		}
		acuse, er = domain.NuevoAcuseConsumoDenominacionPersona(x.Acuse.datos())
		if er != nil || x.Sobre.PersonaRef != acceso.PersonaRef || x.Sobre.Version != acceso.Version || x.Acuse.PersonaRef != acceso.PersonaRef || x.Acuse.Version != acceso.Version || x.Acuse.SobreSHA256 != sha(s) || x.Acuse.DecisionRef != acceso.Material.ResumenCapacidad().DecisionRef() || x.Acuse.CorrelacionRef != acceso.Auditoria.CorrelationRef || x.Acuse.RecursoRef != acceso.Recurso.Referencia {
			return ErrNoDisponible
		}
		return nil
	})
	if e != nil {
		return ports.LecturaDenominacionPersonaConfirmada{}, ErrNoDisponible
	}
	return ports.LecturaDenominacionPersonaConfirmada{Sobre: x.Sobre, Acuse: acuse}, nil
}
func (a *Adaptador) comprobar(ctx context.Context, acceso ports.AccesoDenominacionPersona, consulta string, extra []byte) error {
	b, e := materialLectura(acceso)
	if e != nil {
		return ErrNoDisponible
	}
	args := argumentosMaterial(b, acceso)
	args = append(args, string(extra))
	defer borrarArgumentos(args)
	var valido bool
	if a.transaccion(ctx, func(tx pgx.Tx) error { return tx.QueryRow(ctx, consulta, args...).Scan(&valido) }) != nil || !valido {
		return ErrNoDisponible
	}
	return nil
}
func (a *Adaptador) ValidarAcuseDenominacionPersona(ctx context.Context, acceso ports.AccesoDenominacionPersona, l ports.LecturaDenominacionPersonaConfirmada) error {
	d, e := l.Acuse.Datos()
	raw, er := sobreCanonico(l.Sobre)
	if e != nil || er != nil || sha(raw) != d.SobreSHA256 || d.PersonaRef != acceso.PersonaRef || d.Version != acceso.Version {
		return ErrNoDisponible
	}
	b, e := json.Marshal(acuseWire(d))
	if e != nil {
		return ErrNoDisponible
	}
	return a.comprobar(ctx, acceso, `SELECT vec_contexto_actor_v1.validar_acuse_denominacion_persona_v1`+argumentosComprobacion, b)
}
func (a *Adaptador) RevalidarAccesoDenominacionPersona(ctx context.Context, acceso ports.AccesoDenominacionPersona, s ports.SobreDenominacionPersona) error {
	b, e := sobreCanonico(s)
	if e != nil || s.PersonaRef != acceso.PersonaRef || s.Version != acceso.Version {
		return ErrNoDisponible
	}
	return a.comprobar(ctx, acceso, `SELECT vec_contexto_actor_v1.revalidar_lectura_denominacion_persona_v1`+argumentosComprobacion, b)
}

// Primer corte: no ofrece búsqueda compuesta ni filtra una página descifrada.
func (*Adaptador) BuscarDenominacionPersonaAutorizada(context.Context, ports.AccesoDenominacionPersona, ports.IndiceDenominacionPersona, int) ([]ports.ReferenciaDenominacionPersona, error) {
	return nil, ErrNoDisponible
}
