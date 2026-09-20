package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

var ErrRegistroPeticionesPostgreSQLNoDisponible = errors.New("registro PostgreSQL de peticiones de sesion no disponible")

const consultaConsumirPeticionSesion = `
	SELECT sesion_ref, autenticacion_ref, asercion_ref, cuenta_ref,
	       control_sesion_ref, control_sesion_revision, sesion_valida_hasta
	  FROM vec_identidad_sesiones_v1.consumir_asercion_peticion_sesion_v1(
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13
	)`

const consultaRevocarSesionExacta = `
	SELECT vec_identidad_sesiones_v1.revocar_sesion_v1($1,$2,$3,$4)`

// RegistroPeticionesSesionPostgreSQL mantiene separados el consumidor de
// aserciones (revalidador) y la revocación (revocador). La composición debe
// crear dos pools con LOGIN distintos y aplicar 000006 antes de este binario.
type RegistroPeticionesSesionPostgreSQL struct {
	revalidacion iniciadorTransacciones
	revocacion   iniciadorTransacciones
}

var _ httpseguridad.RegistroPeticionesSesion = (*RegistroPeticionesSesionPostgreSQL)(nil)
var _ httpseguridad.RevocadorSesionExacta = (*RegistroPeticionesSesionPostgreSQL)(nil)

func NuevoRegistroPeticionesSesionPostgreSQL(
	ctx context.Context,
	poolRevalidacion, poolRevocacion *pgxpool.Pool,
) (*RegistroPeticionesSesionPostgreSQL, error) {
	if valorNulo(ctx) || poolRevalidacion == nil || poolRevocacion == nil ||
		poolRevalidacion == poolRevocacion || ctx.Err() != nil {
		return nil, ErrRegistroPeticionesPostgreSQLNoDisponible
	}
	usuarioRevalidacion, err := acreditarCapacidadPool(ctx, poolRevalidacion, capacidadRevalidar)
	if err != nil {
		return nil, ErrRegistroPeticionesPostgreSQLNoDisponible
	}
	usuarioRevocacion, err := acreditarCapacidadPool(ctx, poolRevocacion, capacidadRevocar)
	if err != nil || usuarioRevalidacion == usuarioRevocacion {
		return nil, ErrRegistroPeticionesPostgreSQLNoDisponible
	}
	return nuevoRegistroPeticionesSesionPostgreSQL(poolRevalidacion, poolRevocacion)
}

func nuevoRegistroPeticionesSesionPostgreSQL(
	revalidacion, revocacion iniciadorTransacciones,
) (*RegistroPeticionesSesionPostgreSQL, error) {
	if valorNulo(revalidacion) || valorNulo(revocacion) || mismaInstancia(revalidacion, revocacion) {
		return nil, ErrRegistroPeticionesPostgreSQLNoDisponible
	}
	return &RegistroPeticionesSesionPostgreSQL{revalidacion: revalidacion, revocacion: revocacion}, nil
}

func (r *RegistroPeticionesSesionPostgreSQL) ConsumirYRevalidar(
	ctx context.Context,
	solicitud httpseguridad.SolicitudConsumoPeticionSesion,
) (httpseguridad.ConfirmacionPeticionSesion, error) {
	var cero httpseguridad.ConfirmacionPeticionSesion
	if r == nil || valorNulo(ctx) || valorNulo(r.revalidacion) || !solicitudPeticionSesionValida(solicitud) {
		return cero, errorRegistroPeticiones(ctx)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	tx, err := r.revalidacion.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return cero, errorRegistroPeticiones(ctx)
	}
	defer revertir(tx)
	if err = prepararTransaccion(ctx, tx); err != nil {
		return cero, errorRegistroPeticiones(ctx)
	}
	var resultado httpseguridad.ConfirmacionPeticionSesion
	var controlSesionRevisionTexto string
	err = tx.QueryRow(ctx, consultaConsumirPeticionSesion,
		solicitud.EsquemaHMAC, solicitud.DominioHMACRef, solicitud.ClaveHMACID,
		int64(solicitud.ClaveHMACVersion), solicitud.SesionIDHMAC[:], solicitud.NonceSHA256,
		solicitud.CanalVinculadoRef, string(solicitud.Superficie), solicitud.Metodo,
		solicitud.Destino, solicitud.CuerpoSHA256, solicitud.EmitidaEn, solicitud.ExpiraEn,
	).Scan(&resultado.SesionRef, &resultado.AutenticacionRef, &resultado.AsercionRef,
		&resultado.CuentaRef, &resultado.ControlSesionRef,
		&controlSesionRevisionTexto, &resultado.SesionValidaHasta)
	if err != nil {
		return cero, errorRegistroPeticiones(ctx)
	}
	resultado.ControlSesionRevision, err = strconv.ParseUint(controlSesionRevisionTexto, 10, 64)
	resultado.SesionValidaHasta = resultado.SesionValidaHasta.UTC().Truncate(time.Microsecond)
	if err != nil || !confirmacionPeticionSesionValida(resultado) {
		return cero, errorRegistroPeticiones(ctx)
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, errorRegistroPeticiones(ctx)
	}
	return resultado, nil
}

func (r *RegistroPeticionesSesionPostgreSQL) RevocarSesionExacta(
	ctx context.Context,
	solicitud httpseguridad.SolicitudRevocarSesionExacta,
) error {
	if r == nil || valorNulo(ctx) || valorNulo(r.revocacion) || !solicitudRevocacionExactaValida(solicitud) {
		return errorRegistroPeticiones(ctx)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := r.revocacion.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return errorRegistroPeticiones(ctx)
	}
	defer revertir(tx)
	if err = prepararTransaccion(ctx, tx); err != nil {
		return errorRegistroPeticiones(ctx)
	}
	operacionRef, err := nuevaReferenciaOperacion(rand.Reader)
	if err != nil {
		return errorRegistroPeticiones(ctx)
	}
	var revision string
	err = tx.QueryRow(ctx, consultaRevocarSesionExacta, solicitud.SesionRef,
		solicitud.ControlSesionRef, uint64Texto(solicitud.ControlSesionRevision), operacionRef,
	).Scan(&revision)
	if err != nil || revision == "" || revision == uint64Texto(solicitud.ControlSesionRevision) {
		return errorRegistroPeticiones(ctx)
	}
	if err = tx.Commit(ctx); err != nil {
		return errorRegistroPeticiones(ctx)
	}
	return nil
}

var referenciaPeticionSesion = regexp.MustCompile(`^(?:aut|ase|ses|cse|cta|opr|idh)_[A-Za-z0-9_-]{22,128}$`)

func solicitudPeticionSesionValida(s httpseguridad.SolicitudConsumoPeticionSesion) bool {
	return s.EsquemaHMAC == "vec.identidad.hmac-sha256.v1" &&
		referenciaConPrefijoPeticion(s.DominioHMACRef, "idh_") && textoTecnicoPostgreSQLValido(s.ClaveHMACID, 128) &&
		s.ClaveHMACVersion > 0 && s.ClaveHMACVersion <= uint64(1<<63-1) && s.SesionIDHMAC != [32]byte{} &&
		hexSHA256PostgreSQLValido(s.NonceSHA256) && textoTecnicoPostgreSQLValido(s.CanalVinculadoRef, 256) &&
		s.Superficie.Valida() && s.Superficie != httpseguridad.SuperficiePublicaAnonima && metodoPeticionPostgreSQLValido(s.Metodo) &&
		textoTecnicoPostgreSQLValido(s.Destino, 2048) && hexSHA256PostgreSQLValido(s.CuerpoSHA256) &&
		instantePeticionPostgreSQLValido(s.EmitidaEn) && instantePeticionPostgreSQLValido(s.ExpiraEn) &&
		s.ExpiraEn.After(s.EmitidaEn)
}

func confirmacionPeticionSesionValida(c httpseguridad.ConfirmacionPeticionSesion) bool {
	return referenciaConPrefijoPeticion(c.SesionRef, "ses_") &&
		referenciaConPrefijoPeticion(c.AutenticacionRef, "aut_") &&
		referenciaConPrefijoPeticion(c.AsercionRef, "ase_") &&
		referenciaConPrefijoPeticion(c.CuentaRef, "cta_") &&
		referenciaConPrefijoPeticion(c.ControlSesionRef, "cse_") &&
		c.ControlSesionRevision > 0 && instantePeticionPostgreSQLValido(c.SesionValidaHasta)
}

func solicitudRevocacionExactaValida(s httpseguridad.SolicitudRevocarSesionExacta) bool {
	return referenciaConPrefijoPeticion(s.SesionRef, "ses_") &&
		referenciaConPrefijoPeticion(s.ControlSesionRef, "cse_") && s.ControlSesionRevision > 0
}

func referenciaConPrefijoPeticion(valor, prefijo string) bool {
	return strings.HasPrefix(valor, prefijo) && referenciaPeticionSesion.MatchString(valor)
}

func textoTecnicoPostgreSQLValido(valor string, maximo int) bool {
	return len(valor) > 0 && len(valor) <= maximo && valor == strings.TrimSpace(valor) &&
		!strings.ContainsAny(valor, " \t\r\n")
}

func hexSHA256PostgreSQLValido(valor string) bool {
	if len(valor) != 64 || valor != strings.ToLower(valor) || strings.Trim(valor, "0") == "" {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}

func metodoPeticionPostgreSQLValido(metodo string) bool {
	switch metodo {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func instantePeticionPostgreSQLValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Nanosecond()%1_000 == 0
}

func uint64Texto(valor uint64) string {
	return strconv.FormatUint(valor, 10)
}

func errorRegistroPeticiones(ctx context.Context) error {
	if !valorNulo(ctx) && ctx.Err() != nil {
		return ctx.Err()
	}
	return httpseguridad.ErrRegistroPeticionesAusente
}
