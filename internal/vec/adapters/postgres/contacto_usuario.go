package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	vecapp "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// RegistroContactoUsuarioPostgreSQL guarda exclusivamente el sobre cifrado.
// writer y reader se construyen por separado: ninguna instancia reutiliza un
// pool ni una identidad para ambas superficies.
type RegistroContactoUsuarioPostgreSQL struct{ pool iniciadorTransacciones }
type ResolutorContactoUsuarioPostgreSQL struct {
	pool      iniciadorTransacciones
	protector ports.ProtectorContactoUsuario
}

func NuevoRegistroContactoUsuarioPostgreSQL(pool *pgxpool.Pool) (*RegistroContactoUsuarioPostgreSQL, error) {
	return nuevoRegistroContactoUsuarioPostgreSQL(pool)
}
func nuevoRegistroContactoUsuarioPostgreSQL(pool iniciadorTransacciones) (*RegistroContactoUsuarioPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, vecapp.ErrContactoUsuarioNoDisponible
	}
	return &RegistroContactoUsuarioPostgreSQL{pool: pool}, nil
}
func NuevoResolutorContactoUsuarioPostgreSQL(pool *pgxpool.Pool, protector ports.ProtectorContactoUsuario) (*ResolutorContactoUsuarioPostgreSQL, error) {
	return nuevoResolutorContactoUsuarioPostgreSQL(pool, protector)
}
func nuevoResolutorContactoUsuarioPostgreSQL(pool iniciadorTransacciones, protector ports.ProtectorContactoUsuario) (*ResolutorContactoUsuarioPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) || valorNuloPostgreSQL(protector) {
		return nil, vecapp.ErrContactoUsuarioNoDisponible
	}
	return &ResolutorContactoUsuarioPostgreSQL{pool: pool, protector: protector}, nil
}

func (r *RegistroContactoUsuarioPostgreSQL) GuardarContactoUsuario(ctx context.Context, orden ports.OrdenRegistroContactoUsuario) (ports.ReciboContactoUsuario, error) {
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil || ctx.Err() != nil {
		return ports.ReciboContactoUsuario{}, vecapp.ErrContactoUsuarioNoDisponible
	}
	p := orden.Preparacion
	payload, err := vecapp.PayloadNegocioContactoUsuario(p)
	if err != nil || !bytes.Equal(payload, p.PayloadNegocio) || orden.Material.ValidarEstructura() != nil {
		return ports.ReciboContactoUsuario{}, vecapp.ErrContactoUsuarioNoDisponible
	}
	recurso, auditoria, err := contactoRecursoAuditoria(p.Recurso, p.Auditoria)
	if err != nil {
		return ports.ReciboContactoUsuario{}, vecapp.ErrContactoUsuarioNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return ports.ReciboContactoUsuario{}, contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return ports.ReciboContactoUsuario{}, contactoError(ctx, err)
	}
	var sujeto string
	var version uint64
	var consumoRef, consumoHuella string
	var evidencia []byte
	err = tx.QueryRow(ctx, `SELECT sujeto_ref, version, consumo_ref, consumo_huella_sha256, auditoria_central FROM vec_contacto_usuario_v1.registrar_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(accionRegistroContacto(p.VersionEsperada), orden.Material, payload, recurso, auditoria)...).Scan(&sujeto, &version, &consumoRef, &consumoHuella, &evidencia)
	if err != nil {
		return ports.ReciboContactoUsuario{}, contactoError(ctx, err)
	}
	var recibo ports.ReciboContactoUsuario
	evidenciaCentral, err := vecapp.ValidarEvidenciaCentralRegistroContactoUsuario(evidencia, p, orden.Material.ResumenCapacidad().DecisionRef(), consumoRef, consumoHuella)
	if sujeto != p.SujetoRef || version != p.VersionNueva || err != nil {
		return ports.ReciboContactoUsuario{}, vecapp.ErrContactoUsuarioNoDisponible
	}
	// La fuente central conserva sujeto/versión en la auditoría; nunca se infiere
	// de una respuesta SQL parcial ni se fabrica una firma local.
	recibo.SujetoRef, recibo.Version = sujeto, version
	recibo.ConsumoRef, recibo.ConsumoHuellaSHA256, recibo.EvidenciaCentral = consumoRef, consumoHuella, evidenciaCentral
	if err = ctx.Err(); err != nil {
		return ports.ReciboContactoUsuario{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ReciboContactoUsuario{}, contactoError(ctx, err)
	}
	return recibo, nil
}

func (r *ResolutorContactoUsuarioPostgreSQL) ConContactoUsuario(ctx context.Context, solicitud ports.SolicitudAccesoContactoUsuario, ejecutar func(domain.ContactoUsuario) error) error {
	if r == nil || valorNuloPostgreSQL(r.pool) || valorNuloPostgreSQL(r.protector) || ctx == nil || ctx.Err() != nil || ejecutar == nil || solicitud.Material.ValidarEstructura() != nil {
		return vecapp.ErrContactoUsuarioNoDisponible
	}
	recurso, auditoria, err := contactoRecursoAuditoria(solicitud.Recurso, solicitud.Auditoria)
	if err != nil {
		return vecapp.ErrContactoUsuarioNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return contactoError(ctx, err)
	}
	var sobre ports.SobreContactoUsuario
	var evidencia []byte
	var consumoRef, consumoHuella string
	err = tx.QueryRow(ctx, `SELECT version, clave_ref, nonce, cifrado, consumo_ref, consumo_huella_sha256, auditoria_central FROM vec_contacto_usuario_v1.consultar_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(vecapp.AccionConsultarContactoUsuario, solicitud.Material, solicitud.PayloadNegocio, recurso, auditoria)...).Scan(&sobre.Version, &sobre.ClaveRef, &sobre.Nonce, &sobre.Cifrado, &consumoRef, &consumoHuella, &evidencia)
	if err != nil {
		return contactoError(ctx, err)
	}
	_, err = vecapp.ValidarEvidenciaCentralConsultaContactoUsuario(evidencia, solicitud, consumoRef, consumoHuella)
	if sobre.Version != solicitud.Version || sobre.ClaveRef == "" || len(sobre.Nonce) < 12 || len(sobre.Cifrado) < 16 || err != nil {
		return vecapp.ErrContactoUsuarioNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return contactoError(ctx, err)
	}
	// No se descifra antes de confirmar el consumo y la auditoría de acceso.
	return r.protector.ConContactoUsuarioDescifrado(ctx, solicitud.SujetoRef, sobre, func(claro []byte) error {
		defer borrarBytesAutorizacionPostgreSQL(claro)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// El segundo borde durable vuelve a comprobar gobierno, sesión, contexto,
		// revocación y que la versión sigue siendo la misma después de descifrar.
		// El material no se consume otra vez: la función SQL sólo revalida su uso.
		if e := r.revalidarTrasDescifrado(ctx, solicitud, recurso, auditoria); e != nil {
			return e
		}
		contacto, e := domain.NuevoContactoUsuario(solicitud.SujetoRef, string(claro), solicitud.Version)
		if e != nil {
			return vecapp.ErrContactoUsuarioNoDisponible
		}
		return ejecutar(contacto)
	})
}

func (r *ResolutorContactoUsuarioPostgreSQL) revalidarTrasDescifrado(ctx context.Context, s ports.SolicitudAccesoContactoUsuario, recurso, auditoria []byte) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		return contactoError(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionContacto(ctx, tx); err != nil {
		return contactoError(ctx, err)
	}
	var vigente bool
	err = tx.QueryRow(ctx, `SELECT vec_contacto_usuario_v1.revalidar_consulta_contacto_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11,$12,$13,$14)`, argumentosContacto(vecapp.AccionConsultarContactoUsuario, s.Material, s.PayloadNegocio, recurso, auditoria)...).Scan(&vigente)
	if err != nil || !vigente {
		return contactoError(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return contactoError(ctx, err)
	}
	return nil
}

func accionRegistroContacto(version uint64) string {
	if version == 0 {
		return vecapp.AccionAltaContactoUsuario
	}
	return vecapp.AccionActualizarContactoUsuario
}
func argumentosContacto(accion string, m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, negocio, recurso, auditoria []byte) []any {
	return []any{accion, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), strconv.FormatUint(m.PersonaVersion(), 10), strconv.FormatUint(m.PerfilVersion(), 10), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(), negocio, recurso, auditoria}
}
func contactoRecursoAuditoria(recurso domain.RecursoAutorizable, auditoria domain.AuditEntry) ([]byte, []byte, error) {
	if recurso.Validar() != nil || recurso.Ambitos == nil || recurso.Atributos == nil {
		return nil, nil, errors.New("recurso")
	}
	r, err := json.Marshal(struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}{recurso.Ambitos, recurso.Atributos})
	if err != nil {
		return nil, nil, err
	}
	a, err := json.Marshal(auditoria)
	return r, a, err
}
func configurarTransaccionContacto(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`)
	return err
}
func contactoError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var p *pgconn.PgError
	if errors.As(err, &p) && (p.Code == "40001" || p.Code == "40P01" || p.Code == "P1102") {
		return vecapp.ErrContactoUsuarioNoDisponible
	}
	return vecapp.ErrContactoUsuarioNoDisponible
}

var _ ports.RegistroContactoUsuario = (*RegistroContactoUsuarioPostgreSQL)(nil)
var _ ports.ResolutorContactoUsuarioAutorizado = (*ResolutorContactoUsuarioPostgreSQL)(nil)
