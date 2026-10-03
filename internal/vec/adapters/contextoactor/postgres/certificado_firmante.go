package postgres

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/ports"
)

const consultaResolverVinculoCertificadoFirmanteCT = `
	SELECT certificado_der_sha256, persona_ref, cuenta_ref, vinculo_ref,
	       revision::text, huella_sha256, vigente
	  FROM vec_contexto_actor_v1.resolver_vinculo_certificado_firmante_ct_v1($1)`

// FuenteVinculoCertificadoFirmantePostgreSQL lee únicamente el vínculo central
// acreditado para la huella DER exacta. La lectura no concede competencia.
type FuenteVinculoCertificadoFirmantePostgreSQL struct {
	pool iniciadorContextoActorPostgreSQL
}

// NuevoFuenteVinculoCertificadoFirmantePostgreSQL exige el LOGIN lector CA24.
// La función SQL comprueba la pertenencia nominal del runtime; no se usa SET ROLE.
func NuevoFuenteVinculoCertificadoFirmantePostgreSQL(
	ctx context.Context, pool *pgxpool.Pool,
) (*FuenteVinculoCertificadoFirmantePostgreSQL, error) {
	if ctx == nil {
		return nil, ports.ErrVinculoCertificadoFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if pool == nil {
		return nil, ports.ErrVinculoCertificadoFirmanteNoDisponible
	}
	var login string
	var acreditada bool
	if err := pool.QueryRow(ctx, `SELECT identidad_login, acreditada
	  FROM vec_contexto_actor_v1.acreditar_runtime_certificado_firmante_ct_v1()`).
		Scan(&login, &acreditada); err != nil || !acreditada || login == "" {
		return nil, errorFuenteCertificadoFirmante(ctx)
	}
	return &FuenteVinculoCertificadoFirmantePostgreSQL{pool: pool}, nil
}

func (f *FuenteVinculoCertificadoFirmantePostgreSQL) ResolverFirmantePorCertificado(
	ctx context.Context, huellaDERsha string,
) (ports.VinculoCertificadoFirmante, error) {
	var vacio ports.VinculoCertificadoFirmante
	if ctx == nil {
		return vacio, ports.ErrVinculoCertificadoFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if f == nil || valorNuloContextoActorPostgreSQL(f.pool) {
		return vacio, ports.ErrVinculoCertificadoFirmanteNoDisponible
	}
	if !huellaSHA256CertificadoFirmanteValida(huellaDERsha) {
		return vacio, ports.ErrVinculoCertificadoFirmanteNoAcreditado
	}
	tx, err := f.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errorFuenteCertificadoFirmante(ctx)
	}
	defer revertirContextoActorPostgreSQL(tx)
	if err = prepararTransaccionContextoActorPostgreSQL(ctx, tx); err != nil {
		return vacio, errorFuenteCertificadoFirmante(ctx)
	}
	filas, err := tx.Query(ctx, consultaResolverVinculoCertificadoFirmanteCT, huellaDERsha)
	if err != nil {
		return vacio, errorFuenteCertificadoFirmante(ctx)
	}
	var certificado, principal, cuenta, credencial, revisionTexto, huella *string
	var vigente *bool
	if !filas.Next() {
		err = filas.Err()
		filas.Close()
		if err != nil {
			return vacio, errorFuenteCertificadoFirmante(ctx)
		}
		return vacio, ports.ErrVinculoCertificadoFirmanteNoAcreditado
	}
	if err = filas.Scan(&certificado, &principal, &cuenta, &credencial, &revisionTexto, &huella, &vigente); err != nil {
		filas.Close()
		return vacio, errorFuenteCertificadoFirmante(ctx)
	}
	duplicada := filas.Next()
	err = filas.Err()
	filas.Close()
	if err != nil || duplicada {
		return vacio, errorFuenteCertificadoFirmante(ctx)
	}
	if certificado == nil || principal == nil || cuenta == nil || credencial == nil ||
		revisionTexto == nil || huella == nil || vigente == nil ||
		*certificado != huellaDERsha || !*vigente ||
		!referenciaCertificadoFirmanteValida(*principal, "per_") ||
		!referenciaCertificadoFirmanteValida(*cuenta, "cta_") ||
		!referenciaCertificadoFirmanteValida(*credencial, "vcc_") ||
		!huellaSHA256CertificadoFirmanteValida(*huella) {
		return vacio, ports.ErrVinculoCertificadoFirmanteNoAcreditado
	}
	revision, err := strconv.ParseUint(*revisionTexto, 10, 64)
	if err != nil || revision == 0 || strconv.FormatUint(revision, 10) != *revisionTexto {
		return vacio, ports.ErrVinculoCertificadoFirmanteNoAcreditado
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, errorFuenteCertificadoFirmante(ctx)
	}
	if err = ctx.Err(); err != nil {
		return vacio, err
	}
	return ports.VinculoCertificadoFirmante{
		CertificadoHuella: *certificado, PrincipalRef: *principal,
		CuentaRef: *cuenta, VinculoCredencialRef: *credencial,
		Revision: revision, Huella: *huella, Vigente: true,
	}, nil
}

func huellaSHA256CertificadoFirmanteValida(huella string) bool {
	if len(huella) != 64 || huella == strings.Repeat("0", 64) {
		return false
	}
	for i := range huella {
		if huella[i] < '0' || huella[i] > '9' {
			if huella[i] < 'a' || huella[i] > 'f' {
				return false
			}
		}
	}
	return true
}

func referenciaCertificadoFirmanteValida(ref, prefijo string) bool {
	if !strings.HasPrefix(ref, prefijo) || len(ref)-len(prefijo) < 22 || len(ref)-len(prefijo) > 128 {
		return false
	}
	for i := len(prefijo); i < len(ref); i++ {
		c := ref[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
			c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func errorFuenteCertificadoFirmante(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return ports.ErrVinculoCertificadoFirmanteNoDisponible
}

var _ ports.FuenteVinculoCertificadoFirmante = (*FuenteVinculoCertificadoFirmantePostgreSQL)(nil)

// RevalidadorVinculoCertificadoFirmanteTransaccion comparte la transacción
// del efecto final. No abre, confirma ni revierte una transacción propia.
type RevalidadorVinculoCertificadoFirmanteTransaccion struct{ tx pgx.Tx }

func NuevoRevalidadorVinculoCertificadoFirmanteTransaccion(tx pgx.Tx) (*RevalidadorVinculoCertificadoFirmanteTransaccion, error) {
	if valorNuloContextoActorPostgreSQL(tx) {
		return nil, ports.ErrVinculoCertificadoFirmanteNoDisponible
	}
	return &RevalidadorVinculoCertificadoFirmanteTransaccion{tx: tx}, nil
}

func (r *RevalidadorVinculoCertificadoFirmanteTransaccion) RevalidarVinculoCertificadoFirmante(ctx context.Context, v ports.VinculoCertificadoFirmante) error {
	if ctx == nil || r == nil || valorNuloContextoActorPostgreSQL(r.tx) {
		return ports.ErrVinculoCertificadoFirmanteNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !v.Vigente || !huellaSHA256CertificadoFirmanteValida(v.CertificadoHuella) || !huellaSHA256CertificadoFirmanteValida(v.Huella) ||
		v.Revision == 0 || !referenciaCertificadoFirmanteValida(v.PrincipalRef, "per_") || !referenciaCertificadoFirmanteValida(v.CuentaRef, "cta_") || !referenciaCertificadoFirmanteValida(v.VinculoCredencialRef, "vcc_") {
		return ports.ErrVinculoCertificadoFirmanteNoAcreditado
	}
	var vigente *bool
	err := r.tx.QueryRow(ctx, `SELECT vec_contexto_actor_v1.revalidar_vinculo_certificado_firmante_ct_v1($1,$2::numeric,$3,$4,$5,$6)`,
		v.VinculoCredencialRef, strconv.FormatUint(v.Revision, 10), v.Huella, v.CertificadoHuella, v.PrincipalRef, v.CuentaRef).Scan(&vigente)
	if err != nil {
		return errorFuenteCertificadoFirmante(ctx)
	}
	if vigente == nil || !*vigente {
		return ports.ErrVinculoCertificadoFirmanteNoAcreditado
	}
	return ctx.Err()
}
