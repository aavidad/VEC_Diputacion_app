package postgres

import (
	"context"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const funcionSolicitarDocumentalPortalV1 = "vec_bolsa_llamamientos.solicitar_documental_portal_v1"

// SolicitarDocumentalPortal consume la decisión propia y registra la petición
// dentro de la misma transacción serializable; solo devuelve el recibo durable.
func (r *RegistroPortalCandidatoPostgreSQL) SolicitarDocumentalPortal(ctx context.Context, s puertosbolsa.SolicitudDocumentalPortal) (puertosbolsa.ReciboSolicitudDocumentalPortal, error) {
	var vacio puertosbolsa.ReciboSolicitudDocumentalPortal
	if ctx == nil || r == nil || valorNulo(r.pool) || s.CandidatoRef == "" || s.Bolsa == "" ||
		s.RegistradaEn.IsZero() || s.Material.ValidarEstructura() != nil {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertir(tx)
	m := s.Material
	var recibo puertosbolsa.ReciboSolicitudDocumentalPortal
	err = tx.QueryRow(ctx, `SELECT reutilizada, solicitud_ref, recibo_ref, contenido_sha256, registrada_en, version, estado FROM `+funcionSolicitarDocumentalPortalV1+`($1::text,$2::text,$3::text,$4::text,$5::text,$6::text,$7::text,$8::date,$9::text,$10::timestamptz,$11::bytea,$12::bytea,$13::bytea,$14::bytea,$15::numeric,$16::numeric,$17::bytea,$18::bytea,$19::bytea,$20::bytea)`,
		s.SolicitudRef, s.ReciboRef, s.ContenidoSHA256, s.CandidatoRef, s.Bolsa, s.DocumentoRef, s.DocumentoSHA256, textoOpcional(s.FechaFinCausa), s.Clave, s.RegistradaEn.UTC(),
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&recibo.Reutilizada, &recibo.SolicitudRef, &recibo.ReciboRef, &recibo.ContenidoSHA256, &recibo.RegistradaEn, &recibo.Version, &recibo.Estado)
	if err != nil {
		return vacio, errorPortalCandidato(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorPortalCandidato(ctx, err)
	}
	recibo.RegistradaEn = recibo.RegistradaEn.UTC()
	return recibo, nil
}
