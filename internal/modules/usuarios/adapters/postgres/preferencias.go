package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/shared/telemetria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	consultarCatalogoSQL = `SELECT vec_usuarios.catalogo_vigente_preferencias_v1($1::text)`
	consultarPropiasSQL  = `SELECT vec_usuarios.consultar_preferencias_propias_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	recuperarSQL         = `SELECT vec_usuarios.recuperar_preferencias_operacion_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	guardarSQL           = `SELECT vec_usuarios.guardar_preferencias_propias_v1($1::text,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
	// Los seis ajustes conservan los mismos valores y alcance de SET LOCAL.
	// Una sentencia los aplica antes de acreditar el LOGIN y de leer datos.
	ajustesTransaccionSQL = `SELECT pg_catalog.set_config('search_path','pg_catalog, pg_temp',true),
 pg_catalog.set_config('row_security','on',true),
 pg_catalog.set_config('timezone','UTC',true),
 pg_catalog.set_config('lock_timeout','3s',true),
 pg_catalog.set_config('statement_timeout','15s',true),
 pg_catalog.set_config('idle_in_transaction_session_timeout','20s',true)`
)

// El login debe heredar únicamente el ejecutor Usuarios y ninguna autoridad de
// tablas. Se vuelve a comprobar en cada transacción para cerrar revocaciones.
const acreditarEjecutorSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND l.rolinherit AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND g.rolname=$1 AND NOT g.rolcanlogin AND NOT g.rolsuper AND NOT g.rolcreatedb
 AND NOT g.rolcreaterole AND g.rolinherit AND NOT g.rolreplication AND NOT g.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,g.oid,'MEMBER') AND pg_catalog.pg_has_role(session_user,g.oid,'USAGE')
 AND EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=l.oid)=1
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid=l.oid OR m.member=g.oid)
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios.catalogo_vigente_preferencias_v1(text)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios.consultar_preferencias_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios.recuperar_preferencias_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND pg_catalog.has_function_privilege(session_user,'vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 FROM pg_catalog.pg_roles l JOIN pg_catalog.pg_roles g ON g.rolname=$1
 WHERE l.rolname=session_user`

type filaPreferencias interface{ Scan(...any) error }
type transaccionPreferencias interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) filaPreferencias
	Commit(context.Context) error
	Rollback(context.Context) error
}

type transaccionPGX struct{ pgx.Tx }

func (t transaccionPGX) QueryRow(ctx context.Context, sql string, args ...any) filaPreferencias {
	return t.Tx.QueryRow(ctx, sql, args...)
}

var _ ports.RegistroPreferencias = (*RegistroPreferenciasPostgreSQL)(nil)

// RegistroPreferenciasPostgreSQL no posee el pool: bootstrap debe cerrarlo.
type RegistroPreferenciasPostgreSQL struct {
	iniciar    func(context.Context) (transaccionPreferencias, error)
	superficie vecdomain.SuperficieAutenticacionActorV1
	rol        string
}

func rolEjecutorPreferencias(superficie vecdomain.SuperficieAutenticacionActorV1) string {
	switch superficie {
	case vecdomain.SuperficieAutenticacionInternaCorporativaV1:
		return "vec_usuarios_ejecutor_interno"
	case vecdomain.SuperficieAutenticacionExternaPersonalV1:
		return "vec_usuarios_ejecutor_externo"
	default:
		return ""
	}
}

func NuevoRegistroPreferenciasPostgreSQL(ctx context.Context, pool *pgxpool.Pool, superficie vecdomain.SuperficieAutenticacionActorV1) (*RegistroPreferenciasPostgreSQL, error) {
	rol := rolEjecutorPreferencias(superficie)
	if ctx == nil || pool == nil || ctx.Err() != nil || rol == "" {
		return nil, ports.ErrNoDisponible
	}
	r := &RegistroPreferenciasPostgreSQL{superficie: superficie, rol: rol, iniciar: func(ctx context.Context) (transaccionPreferencias, error) {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return nil, err
		}
		return transaccionPGX{tx}, nil
	}}
	// La sonda usa el mismo camino transaccional que las operaciones reales.
	tx, err := r.abrir(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if err := tx.Commit(ctx); err != nil {
		return nil, errorSeguro(ctx, err)
	}
	return r, nil
}

func (r *RegistroPreferenciasPostgreSQL) abrir(ctx context.Context) (transaccionPreferencias, error) {
	if ctx == nil || r == nil || r.iniciar == nil {
		return nil, ports.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.iniciar(ctx)
	if err != nil {
		return nil, errorSeguro(ctx, err)
	}
	if tx == nil {
		return nil, ports.ErrNoDisponible
	}
	fallar := func(err error) (transaccionPreferencias, error) {
		_ = tx.Rollback(context.Background())
		return nil, errorSeguro(ctx, err)
	}
	if _, err := tx.Exec(ctx, ajustesTransaccionSQL); err != nil {
		return fallar(err)
	}
	var valido bool
	if err := tx.QueryRow(ctx, acreditarEjecutorSQL, r.rol).Scan(&valido); err != nil {
		return fallar(err)
	}
	if !valido {
		_ = tx.Rollback(context.Background())
		return nil, ports.ErrNoDisponible
	}
	return tx, nil
}

func (r *RegistroPreferenciasPostgreSQL) CatalogoVigente(ctx context.Context, orden ports.OrdenPreferencias) (domain.CatalogoPreferencias, error) {
	var vacio domain.CatalogoPreferencias
	superficie, err := orden.Superficie()
	if err != nil || r == nil || superficie != r.superficie || rolEjecutorPreferencias(superficie) != r.rol {
		return vacio, ports.ErrProhibido
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	var datos []byte
	if err := tx.QueryRow(ctx, consultarCatalogoSQL, string(superficie)).Scan(&datos); err != nil {
		return vacio, errorSeguro(ctx, err)
	}
	c, err := decodificarCatalogo(datos)
	if err != nil {
		return vacio, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorSeguro(ctx, err)
	}
	return c, nil
}

func (r *RegistroPreferenciasPostgreSQL) ConsultarPropias(ctx context.Context, orden ports.OrdenPreferencias, material ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EstadoPreferencias, bool, error) {
	inicio := time.Now()
	estado, existe, err := r.consultarPropias(ctx, orden, material, v3)
	// La función SQL consume V3, lee y confirma el apunte nominal en la misma
	// transacción. No existe un tiempo de auditoría positiva separable aquí.
	telemetria.RegistrarFase(ctx, telemetria.FaseLectura, time.Since(inicio), err)
	return estado, existe, err
}

func (r *RegistroPreferenciasPostgreSQL) consultarPropias(ctx context.Context, orden ports.OrdenPreferencias, material ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.EstadoPreferencias, bool, error) {
	var vacio ports.EstadoPreferencias
	if err := r.validarMaterial(orden, material, v3, ports.AccionConsultarPreferencias); err != nil {
		return vacio, false, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, false, err
	}
	defer tx.Rollback(context.Background())
	argumentos, err := argumentosV3(material, v3)
	if err != nil {
		return vacio, false, err
	}
	var datos []byte
	if err := tx.QueryRow(ctx, consultarPropiasSQL, argumentos...).Scan(&datos); err != nil {
		return vacio, false, errorSeguro(ctx, err)
	}
	var estado struct {
		Existe bool `json:"existe"`
		ports.EstadoPreferencias
	}
	if decodificarEstricto(datos, &estado) != nil || estado.PersonaRef != material.PersonaRef ||
		(estado.Existe && (estado.Version == 0 || estado.CatalogoVersionRef == "" || estado.Valores.ValidarCodigos() != nil)) ||
		(!estado.Existe && estado.Version != 0) {
		return vacio, false, ports.ErrNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, false, errorSeguro(ctx, err)
	}
	return estado.EstadoPreferencias, estado.Existe, nil
}

func (r *RegistroPreferenciasPostgreSQL) RecuperarOperacion(ctx context.Context, orden ports.OrdenPreferencias, material ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboPreferencias, bool, error) {
	var vacio ports.ReciboPreferencias
	if err := r.validarMaterial(orden, material, v3, ports.AccionActualizarPreferencias); err != nil {
		return vacio, false, err
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, false, err
	}
	defer tx.Rollback(context.Background())
	argumentos, err := argumentosV3(material, v3)
	if err != nil {
		return vacio, false, err
	}
	var datos []byte
	if err := tx.QueryRow(ctx, recuperarSQL, argumentos...).Scan(&datos); err != nil {
		return vacio, false, errorSeguro(ctx, err)
	}
	if len(datos) == 0 || bytes.Equal(bytes.TrimSpace(datos), []byte("null")) {
		if err := tx.Commit(ctx); err != nil {
			return vacio, false, errorSeguro(ctx, err)
		}
		return vacio, false, nil
	}
	recibo, err := decodificarRecibo(datos, material)
	if err != nil || !recibo.Replay {
		return vacio, false, ports.ErrNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, false, errorSeguro(ctx, err)
	}
	return recibo, true, nil
}

func (r *RegistroPreferenciasPostgreSQL) Guardar(ctx context.Context, orden ports.OrdenPreferencias, peticion ports.PeticionGuardarPreferencias, material ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboPreferencias, error) {
	var vacio ports.ReciboPreferencias
	if err := r.validarMaterial(orden, material, v3, ports.AccionActualizarPreferencias); err != nil {
		return vacio, err
	}
	if peticion.VersionEsperada != material.VersionEsperada || peticion.CatalogoVersionRef != material.CatalogoVersionRef ||
		peticion.ClaveOperacion != material.ClaveOperacion || peticion.Valores != material.Valores {
		return vacio, ports.ErrPeticionInvalida
	}
	valores, err := json.Marshal(peticion.Valores)
	if err != nil {
		return vacio, ports.ErrPeticionInvalida
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	argumentos, err := argumentosV3(material, v3)
	if err != nil {
		return vacio, err
	}
	argumentos = append(argumentos[:1], append([]any{valores}, argumentos[1:]...)...)
	var datos []byte
	if err := tx.QueryRow(ctx, guardarSQL, argumentos...).Scan(&datos); err != nil {
		return vacio, errorSeguro(ctx, err)
	}
	recibo, err := decodificarRecibo(datos, material)
	if err != nil || recibo.Version != material.VersionEsperada+1 || recibo.Valores != peticion.Valores || recibo.Replay {
		return vacio, ports.ErrNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorSeguro(ctx, err)
	}
	return recibo, nil
}

func (r *RegistroPreferenciasPostgreSQL) validarMaterial(orden ports.OrdenPreferencias, material ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion string) error {
	actor, err := orden.ContextoActor()
	if err != nil {
		return ports.ErrNoAutenticado
	}
	superficie, err := orden.Superficie()
	audiencia, errAudiencia := ports.AudienciaPreferencias(accion, superficie)
	if err != nil || errAudiencia != nil || r == nil || superficie != r.superficie || material.Superficie != superficie ||
		rolEjecutorPreferencias(superficie) != r.rol {
		return ports.ErrProhibido
	}
	if material.PersonaRef == "" || material.PersonaRef != actor.PersonaRef || material.PerfilRef != actor.PerfilActivoRef ||
		material.Accion != accion || material.FinalidadRef != ports.FinalidadPreferenciasPropias ||
		material.CatalogoVersionRef == "" || v3.ValidarEstructura() != nil ||
		v3.PersonaVersion() != actor.Instantanea.PersonaVersion || v3.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		v3.ResumenCapacidad().Operacion() != accion || v3.ResumenCapacidad().EfectoRef() != actor.PersonaRef ||
		v3.ResumenCapacidad().AudienciaConsumo() != audiencia {
		return ports.ErrProhibido
	}
	return nil
}

func argumentosV3(material ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]any, error) {
	datos, err := json.Marshal(material)
	if err != nil {
		return nil, ports.ErrPeticionInvalida
	}
	return []any{string(datos), v3.CapacidadCanonica(), v3.DecisionCanonica(), v3.MotivoCanonico(),
		v3.ContextoActorCanonico(), int64(v3.PersonaVersion()), int64(v3.PerfilVersion()),
		v3.PayloadVECAD3(), v3.SobreCOSESign1(), v3.EvidenciaVerificacion(), v3.RaizPublicaSPKI()}, nil
}

func decodificarCatalogo(datos []byte) (domain.CatalogoPreferencias, error) {
	var c domain.CatalogoPreferencias
	if decodificarEstricto(datos, &c) != nil || c.Validar() != nil {
		return domain.CatalogoPreferencias{}, ports.ErrNoDisponible
	}
	return c, nil
}

func decodificarRecibo(datos []byte, material ports.MaterialPreferencias) (ports.ReciboPreferencias, error) {
	var recibo ports.ReciboPreferencias
	if decodificarEstricto(datos, &recibo) != nil || recibo.ReciboRef == "" || recibo.PersonaRef != material.PersonaRef ||
		recibo.Version == 0 || recibo.CatalogoVersionRef == "" || recibo.Valores.ValidarCodigos() != nil ||
		recibo.FechaUTC.IsZero() {
		return ports.ReciboPreferencias{}, ports.ErrNoDisponible
	}
	_, offset := recibo.FechaUTC.Zone()
	if offset != 0 {
		return ports.ReciboPreferencias{}, ports.ErrNoDisponible
	}
	recibo.FechaUTC = recibo.FechaUTC.UTC()
	return recibo, nil
}

func decodificarEstricto(datos []byte, destino any) error {
	if len(datos) == 0 || len(datos) > 64*1024 {
		return ports.ErrNoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(datos))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		return ports.ErrNoDisponible
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return ports.ErrNoDisponible
	}
	return nil
}

func errorSeguro(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P1409":
			return ports.ErrConflicto
		case "22023":
			return ports.ErrPeticionInvalida
		case "42501":
			return ports.ErrProhibido
		}
	}
	return ports.ErrNoDisponible
}
