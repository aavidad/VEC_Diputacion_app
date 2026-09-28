package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	consultarCatalogoSQL = `SELECT vec_usuarios.catalogo_vigente_preferencias_v1()`
	consultarPropiasSQL  = `SELECT vec_usuarios.consultar_preferencias_propias_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	recuperarSQL         = `SELECT vec_usuarios.recuperar_preferencias_operacion_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
	guardarSQL           = `SELECT vec_usuarios.guardar_preferencias_propias_v1($1::text,$2::jsonb,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`
)

// El login debe heredar únicamente el ejecutor Usuarios y ninguna autoridad de
// tablas. Se vuelve a comprobar en cada transacción para cerrar revocaciones.
const acreditarEjecutorSQL = `SELECT session_user=current_user
 AND l.rolcanlogin AND NOT l.rolsuper AND NOT l.rolcreatedb
 AND NOT l.rolcreaterole AND NOT l.rolreplication AND NOT l.rolbypassrls
 AND g.rolname='vec_usuarios_ejecutor' AND NOT g.rolcanlogin
 AND g.rolinherit AND NOT g.rolbypassrls
 AND pg_has_role(session_user,g.oid,'MEMBER') AND pg_has_role(session_user,g.oid,'USAGE')
 AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=l.oid AND m.roleid=g.oid
   AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=l.oid)=1
 AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=g.oid)
 AND has_function_privilege(session_user,'vec_usuarios.catalogo_vigente_preferencias_v1()','EXECUTE')
 AND has_function_privilege(session_user,'vec_usuarios.consultar_preferencias_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND has_function_privilege(session_user,'vec_usuarios.recuperar_preferencias_operacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND has_function_privilege(session_user,'vec_usuarios.guardar_preferencias_propias_v1(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 FROM pg_roles l JOIN pg_roles g ON g.rolname='vec_usuarios_ejecutor'
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
	iniciar func(context.Context) (transaccionPreferencias, error)
}

func NuevoRegistroPreferenciasPostgreSQL(ctx context.Context, pool *pgxpool.Pool) (*RegistroPreferenciasPostgreSQL, error) {
	if ctx == nil || pool == nil || ctx.Err() != nil {
		return nil, ports.ErrNoDisponible
	}
	r := &RegistroPreferenciasPostgreSQL{iniciar: func(ctx context.Context) (transaccionPreferencias, error) {
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
	for _, ajuste := range [...]string{
		"SET LOCAL search_path = pg_catalog",
		"SET LOCAL row_security = on",
		"SET LOCAL TIME ZONE 'UTC'",
		"SET LOCAL lock_timeout = '3s'",
		"SET LOCAL statement_timeout = '15s'",
		"SET LOCAL idle_in_transaction_session_timeout = '20s'",
	} {
		if _, err := tx.Exec(ctx, ajuste); err != nil {
			return fallar(err)
		}
	}
	var valido bool
	if err := tx.QueryRow(ctx, acreditarEjecutorSQL).Scan(&valido); err != nil {
		return fallar(err)
	}
	if !valido {
		_ = tx.Rollback(context.Background())
		return nil, ports.ErrNoDisponible
	}
	return tx, nil
}

func (r *RegistroPreferenciasPostgreSQL) CatalogoVigente(ctx context.Context) (domain.CatalogoPreferencias, error) {
	var vacio domain.CatalogoPreferencias
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	var datos []byte
	if err := tx.QueryRow(ctx, consultarCatalogoSQL).Scan(&datos); err != nil {
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
	var vacio ports.EstadoPreferencias
	if err := validarMaterial(orden, material, v3, ports.AccionConsultarPreferencias); err != nil {
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
	if err := validarMaterial(orden, material, v3, ports.AccionActualizarPreferencias); err != nil {
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
	if err := validarMaterial(orden, material, v3, ports.AccionActualizarPreferencias); err != nil {
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

func validarMaterial(orden ports.OrdenPreferencias, material ports.MaterialPreferencias, v3 vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, accion string) error {
	actor, err := orden.ContextoActor()
	if err != nil {
		return ports.ErrNoAutenticado
	}
	if material.PersonaRef == "" || material.PersonaRef != actor.PersonaRef || material.PerfilRef != actor.PerfilActivoRef ||
		material.Accion != accion || material.FinalidadRef != ports.FinalidadPreferenciasPropias ||
		material.CatalogoVersionRef == "" || v3.ValidarEstructura() != nil ||
		v3.PersonaVersion() != actor.Instantanea.PersonaVersion || v3.PerfilVersion() != actor.Instantanea.PerfilVersion ||
		v3.ResumenCapacidad().Operacion() != accion {
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

type catalogoSQL struct {
	VersionRef      string                     `json:"version_ref"`
	Idiomas         []opcionSQL                `json:"idiomas"`
	TamanosTexto    []opcionSQL                `json:"tamano_textos"`
	Temas           []opcionSQL                `json:"temas"`
	Inicios         []opcionSQL                `json:"inicios"`
	Filas           []int                      `json:"filas"`
	Predeterminados domain.ValoresPreferencias `json:"predeterminados"`
}

type opcionSQL struct {
	Codigo    string `json:"codigo"`
	NombreKey string `json:"nombre_key"`
}

func opcionesDominio(opciones []opcionSQL) []domain.OpcionPreferencia {
	salida := make([]domain.OpcionPreferencia, 0, len(opciones))
	for _, opcion := range opciones {
		salida = append(salida, domain.OpcionPreferencia{Codigo: opcion.Codigo, NombreKey: opcion.NombreKey})
	}
	return salida
}

func decodificarCatalogo(datos []byte) (domain.CatalogoPreferencias, error) {
	var sql catalogoSQL
	if decodificarEstricto(datos, &sql) != nil {
		return domain.CatalogoPreferencias{}, ports.ErrNoDisponible
	}
	c := domain.CatalogoPreferencias{VersionRef: sql.VersionRef, Idiomas: opcionesDominio(sql.Idiomas), TamanosTexto: opcionesDominio(sql.TamanosTexto),
		Temas: opcionesDominio(sql.Temas), Inicios: opcionesDominio(sql.Inicios), Filas: sql.Filas, Predeterminados: sql.Predeterminados}
	if c.Validar() != nil {
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
