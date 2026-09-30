package postgres

import (
	"context"
	"crypto/tls"
	"errors"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

const loginInboxAvisosExternos = "vec_externo_avisos_usuarios"

const acreditarInboxExternoSQL = `SELECT session_user=current_user
 AND session_user='vec_externo_avisos_usuarios'
 AND vec_usuarios_correos_externo.sesion_inbox_avisos_externos_v1()
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_stat_ssl WHERE pid=pg_catalog.pg_backend_pid() AND ssl)
 AND NOT pg_catalog.has_schema_privilege(session_user,'vec_usuarios_correos_interno','USAGE')
 AND NOT pg_catalog.has_table_privilege(session_user,'vec_usuarios_correos_externo.avisos_inbox','SELECT')`

var patronReciboAvisoExterno = regexp.MustCompile(`^aviso_recibo:[0-9a-f]{32}$`)
var patronAuditoriaInboxExterno = regexp.MustCompile(`^auditoria_tecnica_externa:[0-9a-f]{32}$`)
var patronHuellaAvisoExterno = regexp.MustCompile(`^[0-9a-f]{64}$`)

type RegistroAvisosExternosPostgreSQL struct {
	iniciar func(context.Context) (transaccionCorreos, error)
}

var _ ports.RegistroAvisosExternos = (*RegistroAvisosExternosPostgreSQL)(nil)

func NuevoRegistroAvisosExternosPostgreSQL(ctx context.Context, pool *pgxpool.Pool) (*RegistroAvisosExternosPostgreSQL, error) {
	if ctx == nil || pool == nil {
		return nil, ports.ErrAvisoExternoNoDisponible
	}
	if !configuracionInboxExternoValida(pool.Config().ConnConfig) {
		return nil, ports.ErrAvisoExternoNoDisponible
	}
	r := &RegistroAvisosExternosPostgreSQL{iniciar: func(ctx context.Context) (transaccionCorreos, error) {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return nil, err
		}
		return transaccionCorreosPGX{tx}, nil
	}}
	tx, err := r.abrir(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if err = tx.Commit(ctx); err != nil {
		return nil, errorAvisoExternoSeguro(ctx, err)
	}
	return r, nil
}

// El worker no comparte la identidad PostgreSQL del portal externo. Todos
// los destinos, incluido cualquier fallback, verifican TLS y el nombre remoto.
func configuracionInboxExternoValida(cfg *pgx.ConnConfig) bool {
	if cfg == nil || cfg.User != loginInboxAvisosExternos {
		return false
	}
	segura := func(c *tls.Config) bool {
		return c != nil && !c.InsecureSkipVerify && c.ServerName != "" && (c.MinVersion == 0 || c.MinVersion >= tls.VersionTLS12)
	}
	if !segura(cfg.TLSConfig) {
		return false
	}
	for _, f := range cfg.Fallbacks {
		if f == nil || !segura(f.TLSConfig) {
			return false
		}
	}
	return true
}
func errorAvisoExternoSeguro(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ports.ErrAvisoExternoConflicto
	}
	if errors.As(err, &pg) && pg.Code == "22023" {
		return ports.ErrAvisoExternoInvalido
	}
	return ports.ErrAvisoExternoNoDisponible
}
func (r *RegistroAvisosExternosPostgreSQL) abrir(ctx context.Context) (transaccionCorreos, error) {
	if r == nil || r.iniciar == nil || ctx == nil {
		return nil, ports.ErrAvisoExternoNoDisponible
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	tx, err := r.iniciar(ctx)
	if err != nil || tx == nil {
		return nil, errorAvisoExternoSeguro(ctx, err)
	}
	fail := func(e error) (transaccionCorreos, error) {
		_ = tx.Rollback(context.Background())
		return nil, errorAvisoExternoSeguro(ctx, e)
	}
	for _, sql := range []string{"SET LOCAL search_path=pg_catalog", "SET LOCAL row_security=on", "SET LOCAL TIME ZONE 'UTC'", "SET LOCAL lock_timeout='2s'", "SET LOCAL statement_timeout='5s'", "SET LOCAL idle_in_transaction_session_timeout='20s'"} {
		if _, err := tx.Exec(ctx, sql); err != nil {
			return fail(err)
		}
	}
	var ok bool
	if err := tx.QueryRow(ctx, acreditarInboxExternoSQL).Scan(&ok); err != nil || !ok {
		return fail(err)
	}
	return tx, nil
}
func reciboAvisoValido(r ports.ReciboAvisoExterno) bool {
	t, e := time.Parse(canonico.FormatoFechaAvisoExterno, r.AceptadoEn)
	return patronReciboAvisoExterno.MatchString(r.ReciboRef) && patronHuellaAvisoExterno.MatchString(r.Huella) && e == nil && t.Format(canonico.FormatoFechaAvisoExterno) == r.AceptadoEn
}

type denegacionInboxExternoSQL struct {
	Error        string `json:"error"`
	AuditoriaRef string `json:"auditoria_ref"`
}

// Una denegación controlada lleva auditoría real y ningún efecto. Se confirma
// ese único registro antes de devolver el error nominal. Un fallo de auditoría
// o una respuesta abierta nunca alcanza este commit.
func confirmarDenegacionInboxExterno(ctx context.Context, tx transaccionCorreos, b []byte) (bool, error) {
	var v denegacionInboxExternoSQL
	if decodificarCorreosEstricto(b, &v) != nil || v.Error == "" {
		return false, nil
	}
	if !patronAuditoriaInboxExterno.MatchString(v.AuditoriaRef) {
		return true, ports.ErrAvisoExternoNoDisponible
	}
	var nominal error
	switch v.Error {
	case "invalido":
		nominal = ports.ErrAvisoExternoInvalido
	case "conflicto":
		nominal = ports.ErrAvisoExternoConflicto
	case "no_disponible":
		nominal = ports.ErrAvisoExternoNoDisponible
	default:
		return true, ports.ErrAvisoExternoNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return true, errorAvisoExternoSeguro(ctx, err)
	}
	return true, nominal
}

type aceptacionInboxExternoSQL struct {
	Recibo       ports.ReciboAvisoExterno `json:"recibo"`
	AuditoriaRef string                   `json:"auditoria_ref"`
}

func (r *RegistroAvisosExternosPostgreSQL) AceptarAvisoExterno(ctx context.Context, material []byte) (ports.ReciboAvisoExterno, error) {
	var cero ports.ReciboAvisoExterno
	evento, err := canonico.LeerAvisoExterno(material)
	if err != nil {
		return cero, err
	}
	huella, _ := canonico.HuellaAvisoExterno(evento)
	tx, err := r.abrir(ctx)
	if err != nil {
		return cero, err
	}
	defer tx.Rollback(context.Background())
	var b []byte
	if err = tx.QueryRow(ctx, `SELECT vec_usuarios_correos_externo.aceptar_aviso_externo_v1($1::text)`, string(material)).Scan(&b); err != nil {
		return cero, errorAvisoExternoSeguro(ctx, err)
	}
	if denied, err := confirmarDenegacionInboxExterno(ctx, tx, b); denied {
		return cero, err
	}
	var v aceptacionInboxExternoSQL
	if decodificarCorreosEstricto(b, &v) != nil || !patronAuditoriaInboxExterno.MatchString(v.AuditoriaRef) || !reciboAvisoValido(v.Recibo) || v.Recibo.Huella != huella {
		return cero, ports.ErrAvisoExternoNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorAvisoExternoSeguro(ctx, err)
	}
	return v.Recibo, nil
}

type reservaAvisoExternoSQL struct {
	AuditoriaRef string                   `json:"auditoria_ref"`
	Recibo       ports.ReciboAvisoExterno `json:"recibo"`
	Evento       ports.EventoAvisoExterno `json:"evento"`
	ReservaRef   string                   `json:"reserva_ref"`
	PersonaRef   string                   `json:"persona_ref"`
	CorreoRef    string                   `json:"correo_ref"`
	Sobre        *sobreCorreoSQL          `json:"sobre"`
	Estado       string                   `json:"estado"`
	Replay       bool                     `json:"replay"`
}

func estadoAvisoValido(e string) bool {
	return e == "reservado" || e == "aceptado" || e == "no_aceptado" || e == "sin_destino"
}
func (r *RegistroAvisosExternosPostgreSQL) ReservarAvisoExterno(ctx context.Context, ref string) (ports.ReservaAvisoExterno, error) {
	var cero ports.ReservaAvisoExterno
	if !patronReciboAvisoExterno.MatchString(ref) {
		return cero, ports.ErrAvisoExternoInvalido
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return cero, err
	}
	defer tx.Rollback(context.Background())
	var b []byte
	if err = tx.QueryRow(ctx, `SELECT vec_usuarios_correos_externo.reservar_aviso_externo_v1($1::text)`, ref).Scan(&b); err != nil {
		return cero, errorAvisoExternoSeguro(ctx, err)
	}
	if denied, err := confirmarDenegacionInboxExterno(ctx, tx, b); denied {
		return cero, err
	}
	var v reservaAvisoExternoSQL
	if decodificarCorreosEstricto(b, &v) != nil || !patronAuditoriaInboxExterno.MatchString(v.AuditoriaRef) || !reciboAvisoValido(v.Recibo) || v.Recibo.ReciboRef != ref || !estadoAvisoValido(v.Estado) {
		return cero, ports.ErrAvisoExternoNoDisponible
	}
	out := ports.ReservaAvisoExterno{Recibo: v.Recibo, Estado: v.Estado, Replay: v.Replay}
	if v.Replay {
		if v.ReservaRef != "" || v.PersonaRef != "" || v.CorreoRef != "" || v.Sobre != nil || v.Evento != (ports.EventoAvisoExterno{}) {
			return cero, ports.ErrAvisoExternoNoDisponible
		}
	} else {
		h, e := canonico.HuellaAvisoExterno(v.Evento)
		if e != nil || h != v.Recibo.Huella || !patronReservaRef.MatchString(v.ReservaRef) || v.Estado != "reservado" {
			return cero, ports.ErrAvisoExternoNoDisponible
		}
		out.Evento, out.ReservaRef = v.Evento, v.ReservaRef
		if v.Sobre != nil {
			if !patronPersonaAvisos.MatchString(v.PersonaRef) || !patronCorreoRef.MatchString(v.CorreoRef) {
				return cero, ports.ErrAvisoExternoNoDisponible
			}
			out.Sobre, e = v.Sobre.decodificar(v.PersonaRef, v.CorreoRef)
			if e != nil {
				return cero, ports.ErrAvisoExternoNoDisponible
			}
			out.PersonaRef = v.PersonaRef
		} else if v.PersonaRef != "" || v.CorreoRef != "" {
			return cero, ports.ErrAvisoExternoNoDisponible
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorAvisoExternoSeguro(ctx, err)
	}
	return out, nil
}
func (r *RegistroAvisosExternosPostgreSQL) ConfirmarAvisoExterno(ctx context.Context, ref, token, estado string) error {
	if !patronReciboAvisoExterno.MatchString(ref) || !patronReservaRef.MatchString(token) || !estadoAvisoValido(estado) || estado == "reservado" {
		return ports.ErrAvisoExternoInvalido
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var b []byte
	if err = tx.QueryRow(ctx, `SELECT vec_usuarios_correos_externo.confirmar_aviso_externo_v1($1::text,$2::text,$3::text)`, ref, token, estado).Scan(&b); err != nil {
		return errorAvisoExternoSeguro(ctx, err)
	}
	if denied, err := confirmarDenegacionInboxExterno(ctx, tx, b); denied {
		return err
	}
	var v struct {
		Confirmado   bool   `json:"confirmado"`
		AuditoriaRef string `json:"auditoria_ref"`
	}
	if decodificarCorreosEstricto(b, &v) != nil || !v.Confirmado || !patronAuditoriaInboxExterno.MatchString(v.AuditoriaRef) {
		return ports.ErrAvisoExternoNoDisponible
	}
	returnError := tx.Commit(ctx)
	if returnError != nil {
		return errorAvisoExternoSeguro(ctx, returnError)
	}
	return nil
}
