package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

// ConfirmarAltaExternaEnTransaccion usa la transacción SERIALIZABLE de Cronos.
// La fachada SQL de Documentos comprueba su fila y consume AD3; si Cronos
// falla después, ambos consumos y el enlace retroceden juntos.
func ConfirmarAltaExternaEnTransaccion(
	ctx context.Context, tx pgx.Tx,
	s ports.SolicitudConfirmacionAltaExternaEnlace,
	a ports.AutorizacionConfirmacionAltaExternaEnlace,
) (domain.ConstanciaAltaExternaEnlace, error) {
	if ctx == nil || ctx.Err() != nil || tx == nil {
		return domain.ConstanciaAltaExternaEnlace{}, ErrRepositorioNoDisponible
	}
	preimagen, err := s.Preimagen()
	if err != nil || a.Validar(s, time.Now().UTC()) != nil {
		return domain.ConstanciaAltaExternaEnlace{}, ports.ErrSolicitudInvalida
	}
	auth, err := a.AuthJSON(s)
	if err != nil {
		return domain.ConstanciaAltaExternaEnlace{}, ports.ErrSolicitudInvalida
	}
	material := a.Material
	var raw []byte
	err = tx.QueryRow(ctx,
		"SELECT vec_documentos.confirmar_alta_externa_para_enlace_v1($1,$2::jsonb,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)",
		preimagen, string(auth), material.CapacidadCanonica(), material.DecisionCanonica(),
		material.MotivoCanonico(), material.ContextoActorCanonico(), int64(material.PersonaVersion()),
		int64(material.PerfilVersion()), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI(),
	).Scan(&raw)
	if err != nil {
		return domain.ConstanciaAltaExternaEnlace{}, clasificarErrorSQL(err)
	}
	var c domain.ConstanciaAltaExternaEnlace
	if json.Unmarshal(raw, &c) != nil || c.Validar() != nil ||
		c.Registro.Documento.ID != s.DocumentoID || c.Registro.Documento.Version != s.Version ||
		c.Registro.Documento.SHA256 != s.ContenidoSHA256 ||
		c.Registro.Documento.CustodioID != s.CustodioID || c.Registro.Documento.CustodiaRef != s.CustodiaRef ||
		c.Registro.ExpedienteRef != s.ExpedienteRef || c.Registro.TipoRef != s.TipoRef {
		return domain.ConstanciaAltaExternaEnlace{}, ErrRepositorioNoDisponible
	}
	return c, nil
}
