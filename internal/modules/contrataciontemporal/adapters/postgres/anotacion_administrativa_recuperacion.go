package postgres

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Este lector no satisface RecuperadorMaterial...Autorizado. La composición
// exige primero una lectura RRHH nominal actual sobre el expediente exacto.
type LectorMaterialAnotacionAdministrativaPostgreSQL struct{ pool iniciadorTransacciones }

func NuevoLectorMaterialAnotacionAdministrativaPostgreSQL(p *pgxpool.Pool) (*LectorMaterialAnotacionAdministrativaPostgreSQL, error) {
	if dependenciaNula(p) {
		return nil, ct.ErrPersistenciaAnotacionAdministrativaNoDisponible
	}
	return &LectorMaterialAnotacionAdministrativaPostgreSQL{p}, nil
}
func (a *LectorMaterialAnotacionAdministrativaPostgreSQL) RecuperarMaterialAnotacionAdministrativa(ctx context.Context, s ct.SolicitudRecuperarMaterialAnotacionAdministrativa) (ct.MaterialAnotacionAdministrativa, error) {
	var cero ct.MaterialAnotacionAdministrativa
	if ctx == nil || a == nil || dependenciaNula(a.pool) || s.AmbitosHMAC.ValidarDominio(ct.DominioAmbitoIdempotenciaAnotacionAdministrativa) != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	datos, e := s.AmbitosHMAC.Datos()
	if e != nil {
		return cero, ct.ErrPreparacionAnotacionAdministrativaInvalida
	}
	ambitos := []string{datos.Activo.Valor}
	for _, r := range datos.Retenidos {
		ambitos = append(ambitos, r.Valor)
	}
	b, e := json.Marshal(map[string]any{"organizacion_ref": s.OrganizacionRef, "expediente_ref": s.ExpedienteRef, "actor_ref": s.ActorRef, "perfil_ref": s.PerfilRef, "ambitos_hmac": ambitos})
	if e != nil {
		return cero, e
	}
	defer clear(b)
	tx, e := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	defer revertirTransaccion(tx)
	if _, e = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	var raw []byte
	defer func() { clear(raw) }()
	if e = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.recuperar_material_anotacion_administrativa_v1($1::jsonb)::text`, b).Scan(&raw); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	var m materialAnotacionWire
	if len(raw) > 16384 || decodificarJSONEstricto(raw, &m) != nil {
		return cero, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	r := ct.MaterialAnotacionAdministrativa{OrganizacionRef: m.Organizacion, ExpedienteRef: m.Expediente, SolicitudPersonalRef: m.Solicitud, VersionEsperada: m.Version, ClaveIdempotencia: s.ClaveIdempotencia, ActorRef: m.Actor, PerfilRef: m.Perfil, Observaciones: m.Observaciones}
	if r.Validar() != nil || r.OrganizacionRef != s.OrganizacionRef || r.ExpedienteRef != s.ExpedienteRef || r.ActorRef != s.ActorRef || r.PerfilRef != s.PerfilRef || (s.VersionEsperada != 0 && s.VersionEsperada != r.VersionEsperada) {
		return cero, ct.ErrResultadoAnotacionAdministrativaNoConfiable
	}
	if e = tx.Commit(ctx); e != nil {
		return cero, errorAnotacion(ctx, e)
	}
	return r, nil
}
