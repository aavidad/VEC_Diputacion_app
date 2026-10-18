package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

type RegistroOriginalBolsaRRHH struct{ base *Repositorio }

func NuevoRegistroOriginalBolsaRRHHConPool(pool *pgxpool.Pool) (*RegistroOriginalBolsaRRHH, error) {
	if pool == nil {
		return nil, ErrRepositorioNoDisponible
	}
	return &RegistroOriginalBolsaRRHH{base: &Repositorio{db: pool}}, nil
}

// Documentos coteja el permiso documental del revisor y la relación de Bolsa
// antes de entregar la instantánea. La persona titular no identifica al revisor.
func (r *RegistroOriginalBolsaRRHH) ConsultarAdmitidoRRHH(ctx context.Context, q docports.ConsultaPersistenciaSubidaLocalRRHH) (docports.InstantaneaSubidaLocal, error) {
	var vacia docports.InstantaneaSubidaLocal
	if r == nil || r.base == nil || ctx == nil || ctx.Err() != nil {
		return vacia, docports.ErrSubidaLocalNoDisponible
	}
	pre, err := docports.PreimagenConsultaSubidaLocalRRHH(q.SolicitudOriginalSubidaLocalRRHH)
	if err != nil {
		return vacia, err
	}
	a := q.AutorizacionV3
	if a.Accion != docports.AccionBolsaJustificanteRenunciaDescargarRRHH || a.CargaID != q.DocumentoID ||
		a.RecursoRef != q.RecursoRef || a.CorrelacionRef != q.CorrelacionRef {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	auth, err := authSubidaLocalJSON(a, q.ActorRegistradoV2, q.RecursoRef)
	if err != nil {
		return vacia, err
	}
	m, err := materialSubidaLocal(a)
	if err != nil {
		return vacia, err
	}
	raw, err := r.base.transaccion(ctx,
		"SELECT vec_documentos.consultar_justificante_renuncia_rrhh_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		pre, string(auth), m.capacidad, m.decision, m.motivo, m.contexto, m.persona, m.perfil,
		m.payload, m.sobre, m.evidencia, m.raiz)
	if err != nil {
		return vacia, err
	}
	var i docports.InstantaneaSubidaLocal
	if json.Unmarshal(raw, &i) != nil || !instantaneaSubidaLocalValida(i) ||
		i.Resultado.Estado != docports.EstadoSubidaLocalAdmitida || i.Resultado.ComprobanteRef != q.DocumentoID ||
		(q.CargaID != "" && i.Resultado.CargaID != q.CargaID) || i.RecursoRef != q.RecursoRef ||
		i.PersonaRef != q.PersonaTitularRef || i.HuellaPDFSHA256 != q.HuellaPDFSHA256 ||
		i.ModuloID != "bolsa" || i.TipoRecurso != "justificante_renuncia" ||
		i.Finalidad != "custodiar_justificante_renuncia_bolsa" {
		return vacia, docports.ErrSubidaLocalDenegada
	}
	i.Resultado.RecursoRef = i.RecursoRef
	return i, nil
}

var _ docports.RegistroOriginalSubidaLocalRRHH = (*RegistroOriginalBolsaRRHH)(nil)
