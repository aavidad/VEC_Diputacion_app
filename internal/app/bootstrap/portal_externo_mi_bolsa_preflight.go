package bootstrap

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// La fachada exterior conserva el contrato de B29/B30/B40, pero usa los
// destinos propios B63. Esta sonda comprueba esas firmas, nunca las generales.
var consultaMigracionesPortalCandidatoExterno = nuevaConsultaMigracionesPortalCandidatoExterno()

func nuevaConsultaMigracionesPortalCandidatoExterno() string {
	consulta := consultaMigracionesPortalCandidato
	for _, par := range [][2]string{
		{"manifestar_disposicion_oferta_v1", "manifestar_disposicion_oferta_externo_v1"},
		{"listar_ofertas_candidato_v1", "listar_ofertas_candidato_externo_v1"},
		{"solicitar_portal_candidato_v1", "solicitar_portal_candidato_externo_v1"},
		{"responder_llamamiento_portal_v1", "responder_llamamiento_portal_externo_v1"},
		{"leer_portal_candidato_v1", "leer_portal_candidato_externo_v1"},
		{"confirmar_contacto_propio_v1", "confirmar_contacto_propio_externo_v1"},
		{"leer_contacto_candidato_v1", "leer_contacto_candidato_externo_v1"},
	} {
		consulta = strings.ReplaceAll(consulta, par[0], par[1])
	}
	return consulta
}

func comprobarMigracionesPortalCandidatoExterno(ctx context.Context, bolsa *pgxpool.Pool) error {
	if ctx == nil || bolsa == nil {
		return errPortalCandidatoComprobacionRota
	}
	var estado estadoMigracionesPortalCandidato
	if err := bolsa.QueryRow(ctx, consultaMigracionesPortalCandidatoExterno).Scan(&estado.ad384,
		&estado.ad386, &estado.bolsa29, &estado.bolsa30, &estado.bolsa40); err != nil {
		return errPortalCandidatoComprobacionRota
	}
	return estado.diagnostico()
}
