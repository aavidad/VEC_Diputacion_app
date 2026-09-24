package interna

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const consultaCoincidenciaPoliticaCertificado = `
SELECT vec_identidad_sesiones_v1.coincide_politica_certificado_desarrollo_v1(
	$1::text, $2::text, $3::timestamptz
)`

type consultorPoliticaCertificado interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// La excepción temporal solo se compone si el dictamen privado, la fecha de
// retirada del listener y la fila vigente de Identidad coinciden exactamente.
// El rol lector dedicado de sesiones recibe únicamente esta sonda nominal.
func preflightPoliticaCertificado(
	ctx context.Context, q consultorPoliticaCertificado,
	material materialIdentidadCertificado, retirada time.Time,
) error {
	if ctx == nil || ctx.Err() != nil || q == nil ||
		!identificadorCertificadoPersonalValido(material.politicaRef, "pga_") ||
		!huellaCertificadoPersonalValida(material.huellaPolitica) ||
		retirada.IsZero() || retirada.Location() != time.UTC || retirada.Nanosecond() != 0 {
		return ErrCertificadoPersonalNoDisponible
	}
	sonda, cancelar := context.WithTimeout(ctx, plazoSondaPoolSeguimiento)
	defer cancelar()
	var coincide bool
	if err := q.QueryRow(sonda, consultaCoincidenciaPoliticaCertificado,
		material.politicaRef, strings.TrimPrefix(material.huellaPolitica, "sha256:"), retirada,
	).Scan(&coincide); err != nil || !coincide || sonda.Err() != nil {
		return ErrCertificadoPersonalNoDisponible
	}
	return nil
}
