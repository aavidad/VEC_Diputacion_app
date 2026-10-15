package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ResolutorAmbitoJustificanteRenunciaRRHHPostgreSQL struct {
	pool    *pgxpool.Pool
	captura bolsaports.CapturadorAmbitoJustificanteRenunciaRRHH
}

func NuevoResolutorAmbitoJustificanteRenunciaRRHHConPool(pool *pgxpool.Pool,
	captura bolsaports.CapturadorAmbitoJustificanteRenunciaRRHH) (*ResolutorAmbitoJustificanteRenunciaRRHHPostgreSQL, error) {
	if pool == nil || captura == nil {
		return nil, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	return &ResolutorAmbitoJustificanteRenunciaRRHHPostgreSQL{pool: pool, captura: captura}, nil
}

func (r *ResolutorAmbitoJustificanteRenunciaRRHHPostgreSQL) ResolverAmbitoJustificanteRenunciaRRHH(ctx context.Context,
	q bolsaports.SolicitudConsultaJustificanteRenunciaRRHH) (bolsaports.AmbitoJustificanteRenunciaRRHH, error) {
	var vacio bolsaports.AmbitoJustificanteRenunciaRRHH
	if r == nil || r.pool == nil || r.captura == nil || ctx == nil {
		return vacio, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !bolsaJustificanteValida.MatchString(q.BolsaRef) || !participacionJustificanteValida.MatchString(q.ParticipacionRef) ||
		q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil {
		return vacio, bolsaports.ErrJustificanteRenunciaDenegado
	}
	// Esta primera respuesta solo prepara la captura; nunca se entrega como
	// permiso ni como dato de ficha sin superar la lectura auditada siguiente.
	raw, err := ejecutarPreparacionJustificante(ctx, r.pool,
		`SELECT vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1($1,$2)`, q.BolsaRef, q.ParticipacionRef)
	if err != nil {
		return vacio, err
	}
	var metadatos bolsaports.AmbitoJustificanteRenunciaRRHH
	if err := decodificarPreparacionJustificante(raw, &metadatos); err != nil {
		return vacio, err
	}
	if !ambitoJustificanteRRHHValido(metadatos, q) || metadatos.AuditoriaRef != "" {
		return vacio, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	captura, err := r.captura.CapturarAmbitoJustificanteRenunciaRRHH(ctx, q, metadatos)
	if err != nil {
		return vacio, errorAutoridadJustificante(err)
	}
	if captura.BolsaRef != q.BolsaRef || captura.LlamamientoRef != metadatos.LlamamientoRef ||
		len(captura.ContextoCanonico) == 0 || len(captura.VinculoCanonico) == 0 || len(captura.Orden) == 0 ||
		!captura.ValidaHasta.After(time.Now().UTC()) {
		return vacio, bolsaports.ErrJustificanteRenunciaDenegado
	}
	raw, err = ejecutarPreparacionJustificante(ctx, r.pool,
		`SELECT vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1($1,$2,$3,$4,$5::jsonb)`,
		q.BolsaRef, q.ParticipacionRef, captura.ContextoCanonico, captura.VinculoCanonico, captura.Orden)
	if err != nil {
		return vacio, err
	}
	var actual bolsaports.AmbitoJustificanteRenunciaRRHH
	if err := decodificarPreparacionJustificante(raw, &actual); err != nil {
		return vacio, err
	}
	if !ambitoJustificanteRRHHValido(actual, q) ||
		actual.AuditoriaRef == "" || actual.UnidadRef != metadatos.UnidadRef || actual.AmbitoRef != metadatos.AmbitoRef ||
		actual.LlamamientoRef != metadatos.LlamamientoRef || !captura.ValidaHasta.After(time.Now().UTC()) {
		return vacio, bolsaports.ErrJustificanteRenunciaDenegado
	}
	return actual, nil
}

func ambitoJustificanteRRHHValido(a bolsaports.AmbitoJustificanteRenunciaRRHH, q bolsaports.SolicitudConsultaJustificanteRenunciaRRHH) bool {
	return a.BolsaRef == q.BolsaRef && a.ParticipacionRef == q.ParticipacionRef &&
		a.LlamamientoRef != "" && len(a.LlamamientoRef) <= 256 && a.UnidadRef != "" && len(a.UnidadRef) <= 256 &&
		a.AmbitoRef != "" && len(a.AmbitoRef) <= 256
}
