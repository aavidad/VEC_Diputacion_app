package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// AlmacenAutorizacionUsuariosExterno ejecuta únicamente las fachadas AUT-17.
// Cada instancia recibe un pool de fuente o registro exclusivo; PostgreSQL
// comprueba el LOGIN real y deniega cualquier papel de la autorización interna.
type AlmacenAutorizacionUsuariosExterno struct {
	pool iniciadorTransacciones
}

func NuevoAlmacenAutorizacionUsuariosExterno(pool *pgxpool.Pool) (*AlmacenAutorizacionUsuariosExterno, error) {
	return nuevoAlmacenAutorizacionUsuariosExterno(pool)
}

func nuevoAlmacenAutorizacionUsuariosExterno(pool iniciadorTransacciones) (*AlmacenAutorizacionUsuariosExterno, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &AlmacenAutorizacionUsuariosExterno{pool: pool}, nil
}

func (a *AlmacenAutorizacionUsuariosExterno) ObtenerInstantaneaAutorizacion(ctx context.Context, principalID, perfilActivoRef string) (domain.InstantaneaAutorizacion, error) {
	if a == nil || valorNuloPostgreSQL(a.pool) || ctx == nil {
		return domain.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.InstantaneaAutorizacion{}, err
	}
	if !identificadorPostgreSQLSeguro(principalID, 512) || !identificadorPostgreSQLSeguro(perfilActivoRef, 512) {
		return domain.InstantaneaAutorizacion{}, ports.ErrAsignacionPerfilNoEncontrada
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil || valorNuloPostgreSQL(tx) {
		return domain.InstantaneaAutorizacion{}, errorFuenteAutorizacion(ctx)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if configurarTransaccionAutorizacion(ctx, tx) != nil {
		return domain.InstantaneaAutorizacion{}, errorFuenteAutorizacion(ctx)
	}
	var asignacionJSON, rolJSON, controlJSON, politicasJSON []byte
	var revisionTexto, huella string
	err = tx.QueryRow(ctx, `SELECT documento_asignacion, documento_rol, documento_control_rol,
		revision_catalogo, huella_catalogo, documentos_politicas
		FROM vec_autorizacion.obtener_instantanea_usuarios_externo_v1($1,$2)`, principalID, perfilActivoRef).
		Scan(&asignacionJSON, &rolJSON, &controlJSON, &revisionTexto, &huella, &politicasJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.InstantaneaAutorizacion{}, ports.ErrAsignacionPerfilNoEncontrada
	}
	if err != nil {
		return domain.InstantaneaAutorizacion{}, errorFuenteAutorizacion(ctx)
	}
	var asignacion domain.AsignacionPerfil
	var rol domain.VersionRol
	var control domain.ControlVigenciaVersionRol
	var politicas []domain.PoliticaRestrictiva
	if decodificarDocumentoPostgreSQL(asignacionJSON, &asignacion) != nil ||
		decodificarDocumentoPostgreSQL(rolJSON, &rol) != nil ||
		decodificarDocumentoPostgreSQL(controlJSON, &control) != nil ||
		decodificarDocumentoPostgreSQL(politicasJSON, &politicas) != nil ||
		asignacion.PrincipalID != principalID || asignacion.PerfilActivoRef != perfilActivoRef {
		return domain.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	revision, err := parsearRevisionAutorizacion(revisionTexto)
	if err != nil {
		return domain.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	instantanea := domain.InstantaneaAutorizacion{AsignacionPerfil: asignacion, VersionRol: rol,
		ControlVigenciaVersionRol: control, Politicas: politicas,
		RevisionCatalogoPoliticas: revision, CatalogoPoliticasHuellaSHA256: huella}
	if instantanea.Validar() != nil || tx.Commit(ctx) != nil {
		return domain.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	return instantanea, nil
}

func (a *AlmacenAutorizacionUsuariosExterno) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx context.Context, orden ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	if a == nil || valorNuloPostgreSQL(a.pool) || ctx == nil {
		return time.Time{}, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
	}
	datos, err := orden.Datos()
	if err != nil {
		return time.Time{}, err
	}
	return a.registrarConReintento(ctx, datos, true, ports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible)
}

func (a *AlmacenAutorizacionUsuariosExterno) RegistrarDenegacionAutorizacionLigadaV3(ctx context.Context, orden ports.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	if a == nil || valorNuloPostgreSQL(a.pool) || ctx == nil {
		return ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
	}
	datos, err := orden.Datos()
	if err != nil {
		return err
	}
	_, err = a.registrarConReintento(ctx, datos, false, ports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible)
	return err
}

func (a *AlmacenAutorizacionUsuariosExterno) registrarConReintento(ctx context.Context, datos ports.DatosOrdenRegistroAutorizacionLigadaV3, concedida bool, noDisponible error) (time.Time, error) {
	var registrada time.Time
	err := postgresqlcomun.RepetirTrasCarreraSerializable(ctx, func() error {
		var e error
		registrada, e = a.registrar(ctx, datos, concedida, noDisponible)
		return e
	})
	var carrera carreraSerializacionRegistroV3
	if errors.As(err, &carrera) {
		return time.Time{}, carrera.traducido
	}
	if err != nil {
		return time.Time{}, err
	}
	return registrada, nil
}

func (a *AlmacenAutorizacionUsuariosExterno) registrar(ctx context.Context, datos ports.DatosOrdenRegistroAutorizacionLigadaV3, concedidaEsperada bool, noDisponible error) (time.Time, error) {
	decision, motivo, huella, emitida, vence, codigo, err := serializarDecisionContextoActorV3PostgreSQL(datos, concedidaEsperada)
	if err != nil {
		return time.Time{}, noDisponible
	}
	defer borrarBytesAutorizacionPostgreSQL(decision, motivo)
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, err, noDisponible)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionAutorizacion(ctx, tx); err != nil {
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, err, noDisponible)
	}
	filas, err := tx.Query(ctx, `SELECT concedida,codigo,decision_huella_sha256,registrada_en
		FROM vec_autorizacion.registrar_decision_usuarios_externo_v3($1::bytea,$2::bytea,$3::numeric,$4::numeric)`,
		decision, motivo, strconv.FormatUint(datos.ResultadoContexto.Contexto.Instantanea.PersonaVersion, 10),
		strconv.FormatUint(datos.ResultadoContexto.Contexto.Instantanea.PerfilVersion, 10))
	if err != nil {
		if !valorNuloPostgreSQL(filas) {
			filas.Close()
		}
		return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, err, noDisponible)
	}
	if valorNuloPostgreSQL(filas) {
		return time.Time{}, noDisponible
	}
	defer filas.Close()
	var registrada time.Time
	var concedida bool
	var codigoReal, huellaReal string
	if !filas.Next() {
		if filas.Err() != nil {
			return time.Time{}, errorRegistroAutorizacionLigadaV3(ctx, filas.Err(), noDisponible)
		}
		return time.Time{}, ports.ErrInstantaneaAutorizacionObsoleta
	}
	if err = filas.Scan(&concedida, &codigoReal, &huellaReal, &registrada); err != nil || filas.Next() || filas.Err() != nil {
		return time.Time{}, noDisponible
	}
	registrada = registrada.UTC()
	if concedida != concedidaEsperada || codigoReal != codigo || huellaReal != huella ||
		!instanteRegistroContextoActorV3PostgreSQLValido(registrada) || registrada.Before(emitida) || !registrada.Before(vence) {
		return time.Time{}, noDisponible
	}
	filas.Close()
	if err = ctx.Err(); err != nil {
		return time.Time{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return time.Time{}, errorCommitRegistroAutorizacionLigadaV3(err, noDisponible)
	}
	return registrada, nil
}

var _ ports.FuenteAutorizacion = (*AlmacenAutorizacionUsuariosExterno)(nil)
var _ ports.RegistroConcesionesCandidatasAutorizacionLigadaV3 = (*AlmacenAutorizacionUsuariosExterno)(nil)
var _ ports.RegistroDenegacionesAutorizacionLigadaV3 = (*AlmacenAutorizacionUsuariosExterno)(nil)
