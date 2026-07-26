//go:build integracion_postgresql_o405

package postgres

import (
	"context"
)

// NuevoPoolRecuperacionCoberturaO405PostgreSQLParaIntegracionSocketUnix crea,
// solo bajo el tag O4-05, el pool sellado para el socket Unix local sin TLS.
func NuevoPoolRecuperacionCoberturaO405PostgreSQLParaIntegracionSocketUnix(
	ctx context.Context,
	cadenaConexion string,
) (*PoolRecuperacionCoberturaO405PostgreSQL, error) {
	return nuevoPoolRecuperacionCoberturaO405PostgreSQL(
		ctx,
		cadenaConexion,
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	)
}

// NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQLParaIntegracionSocketUnix
// solo existe bajo el tag explícito de integración O4-05. La excepción cubre
// exclusivamente sockets Unix locales sin TLS; conserva toda la acreditación
// de identidad y privilegios.
func NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQLParaIntegracionSocketUnix(
	ctx context.Context,
	dependencia *PoolRecuperacionCoberturaO405PostgreSQL,
) (*EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL, error) {
	return nuevoEjecutorRecuperacionCoberturaO405Acreditado(
		ctx,
		&origenAcreditacionPoolO405PostgreSQL{
			dependencia: dependencia,
		},
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	)
}
