package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// LectorResultadoEmisionCTPostgreSQL usa exclusivamente B98 instalado. No
// posee pool: la composición CT aporta su transacción de detalle ya autorizada
// y auditada con LOGIN consultor RRHH. Nunca consulta tablas de Bolsa desde CT.
type LectorResultadoEmisionCTPostgreSQL struct{}

func NuevoLectorResultadoEmisionCTPostgreSQL() *LectorResultadoEmisionCTPostgreSQL {
	return &LectorResultadoEmisionCTPostgreSQL{}
}

func (r *LectorResultadoEmisionCTPostgreSQL) LeerResultadoEmisionCTEnTx(
	ctx context.Context, tx pgx.Tx, bolsaRef, llamamientoRef, reciboEmisionRef string,
) (ports.ResultadoEmisionCT, error) {
	if r == nil || ctx == nil || tx == nil || bolsaRef == "" || llamamientoRef == "" || reciboEmisionRef == "" {
		return ports.ResultadoEmisionCT{}, ports.ErrResultadoEmisionCTNoDisponible
	}
	var contenido []byte
	err := tx.QueryRow(ctx, `SELECT resultado FROM (
 SELECT vec_bolsa_llamamientos.leer_resultado_emision_ct_v1(
  $1::text,$2::text,$3::text) AS resultado
) b98 WHERE resultado IS NOT NULL`, bolsaRef, llamamientoRef, reciboEmisionRef).Scan(&contenido)
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ResultadoEmisionCT{}, ports.ErrResultadoEmisionCTNoEncontrado
	}
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42501" {
			return ports.ResultadoEmisionCT{}, ports.ErrResultadoEmisionCTDenegado
		}
		return ports.ResultadoEmisionCT{}, ports.ErrResultadoEmisionCTNoDisponible
	}
	if len(contenido) == 0 || len(contenido) > 256*1024 {
		return ports.ResultadoEmisionCT{}, ports.ErrResultadoEmisionCTDivergente
	}
	var resultado ports.ResultadoEmisionCT
	decoder := json.NewDecoder(bytes.NewReader(contenido))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&resultado) != nil || decoder.Decode(new(any)) != io.EOF ||
		validarResultadoEmisionCT(resultado, bolsaRef, llamamientoRef, reciboEmisionRef) != nil {
		return ports.ResultadoEmisionCT{}, ports.ErrResultadoEmisionCTDivergente
	}
	resultado.EmitidoEn = resultado.EmitidoEn.UTC()
	for i := range resultado.Participaciones {
		normalizarInstantesResultadoCT(&resultado.Participaciones[i])
	}
	return resultado, nil
}

func validarResultadoEmisionCT(r ports.ResultadoEmisionCT, bolsaRef, llamamientoRef, reciboRef string) error {
	if r.BolsaRef != bolsaRef || r.LlamamientoRef != llamamientoRef ||
		r.ReciboEmisionRef != reciboRef || r.EmitidoEn.IsZero() ||
		len(r.Participaciones) < 1 || len(r.Participaciones) > 100 {
		return ports.ErrResultadoEmisionCTDivergente
	}
	vistas := make(map[string]struct{}, len(r.Participaciones))
	for _, p := range r.Participaciones {
		if p.ParticipacionRef == "" {
			return ports.ErrResultadoEmisionCTDivergente
		}
		if _, repetida := vistas[p.ParticipacionRef]; repetida {
			return ports.ErrResultadoEmisionCTDivergente
		}
		vistas[p.ParticipacionRef] = struct{}{}
		switch p.Respuesta {
		case "":
			if p.Modo != "" || p.ReciboRespuestaRef != "" || p.RespondidaEn != nil || p.JustificanteRef != "" {
				return ports.ErrResultadoEmisionCTDivergente
			}
		case "acepta", "renuncia", "renuncia_justificada":
			if (p.Modo != "firme" && p.Modo != "propuesta_rrhh") || p.ReciboRespuestaRef == "" ||
				p.RespondidaEn == nil || p.RespondidaEn.IsZero() ||
				(p.Respuesta == "renuncia_justificada" && p.JustificanteRef == "") ||
				(p.Respuesta != "renuncia_justificada" && p.JustificanteRef != "") {
				return ports.ErrResultadoEmisionCTDivergente
			}
		default:
			return ports.ErrResultadoEmisionCTDivergente
		}
		switch p.ContactoResultado {
		case "":
			if p.ReciboContactoRef != "" || p.ContactoEn != nil {
				return ports.ErrResultadoEmisionCTDivergente
			}
		case "contactado", "no_contesta", "buzon", "acepta", "rechaza", "aplazado", "otro", "enviado", "no_enviado":
			if p.ReciboContactoRef == "" || p.ContactoEn == nil || p.ContactoEn.IsZero() {
				return ports.ErrResultadoEmisionCTDivergente
			}
		default:
			return ports.ErrResultadoEmisionCTDivergente
		}
		switch p.SituacionActual {
		case "":
			if p.ReciboSituacionRef != "" || p.SituacionDesde != nil {
				return ports.ErrResultadoEmisionCTDivergente
			}
		case "disponible", "no_disponible", "trabajando", "pendiente_incorporacion", "renuncia", "excluido", "disponible_desde":
			if p.ReciboSituacionRef == "" || p.SituacionDesde == nil || p.SituacionDesde.IsZero() {
				return ports.ErrResultadoEmisionCTDivergente
			}
		default:
			return ports.ErrResultadoEmisionCTDivergente
		}
	}
	return nil
}

func normalizarInstantesResultadoCT(p *ports.ResultadoParticipacionEmisionCT) {
	if p == nil {
		return
	}
	for _, fecha := range []*time.Time{p.RespondidaEn, p.ContactoEn, p.SituacionDesde} {
		if fecha != nil {
			*fecha = fecha.UTC()
		}
	}
}
