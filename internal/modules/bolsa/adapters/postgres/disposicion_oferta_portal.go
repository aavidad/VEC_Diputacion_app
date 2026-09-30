package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	funcionManifestarDisposicionV1 = "vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1"
	funcionListarOfertasCandidato  = "vec_bolsa_llamamientos.listar_ofertas_candidato_v1"
)

var _ puertosbolsa.RegistroDisposicionOferta = (*RegistroDisposicionOfertaPostgreSQL)(nil)

// RegistroDisposicionOfertaPostgreSQL escribe la disposición de la persona a
// una oferta (Bolsa 000029): consume la decisión AD3-84 en la misma
// transacción que la fila.
type RegistroDisposicionOfertaPostgreSQL struct {
	portal *RegistroPortalCandidatoPostgreSQL
}

func NuevoRegistroDisposicionOfertaPostgreSQL(pool *pgxpool.Pool) (*RegistroDisposicionOfertaPostgreSQL, error) {
	portal, err := NuevoRegistroPortalCandidatoPostgreSQL(pool)
	if err != nil {
		return nil, err
	}
	return &RegistroDisposicionOfertaPostgreSQL{portal: portal}, nil
}

func (r *RegistroDisposicionOfertaPostgreSQL) ManifestarDisposicion(ctx context.Context, s puertosbolsa.DisposicionPortalCandidato) (puertosbolsa.ReciboDisposicionPortal, error) {
	var vacio puertosbolsa.ReciboDisposicionPortal
	if ctx == nil || r == nil || r.portal == nil || s.OfertaRef == "" || s.CandidatoRef == "" || s.ReciboRef == "" || s.ManifestadaEn.IsZero() || s.Material.ValidarEstructura() != nil {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	tx, err := r.portal.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertir(tx)
	m := s.Material
	var recibo puertosbolsa.ReciboDisposicionPortal
	err = tx.QueryRow(ctx, `SELECT reutilizada, recibo_ref, oferta_ref, manifestada_en FROM `+funcionManifestarDisposicionV1+`($1::text,$2::text,$3::text,$4::text,$5::timestamptz,$6::bytea,$7::bytea,$8::bytea,$9::bytea,$10::numeric,$11::numeric,$12::bytea,$13::bytea,$14::bytea,$15::bytea)`,
		s.OfertaRef, s.ReciboRef, s.CandidatoRef, s.Clave, s.ManifestadaEn.UTC(),
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&recibo.Reutilizada, &recibo.ReciboRef, &recibo.OfertaRef, &recibo.ManifestadaEn)
	if err != nil {
		return vacio, errorDisposicionOferta(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorDisposicionOferta(ctx, err)
	}
	recibo.ManifestadaEn = recibo.ManifestadaEn.UTC()
	return recibo, nil
}

func errorDisposicionOferta(ctx context.Context, err error) error {
	var pgErr *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "VBO05":
			return puertosbolsa.ErrPortalDisposicionYaManifestada
		case "VBO06":
			return puertosbolsa.ErrPortalOfertaNoAbierta
		}
	}
	return errorPortalCandidato(ctx, err)
}

type ofertaCandidatoPostgreSQL struct {
	OfertaRef    string                   `json:"oferta_ref"`
	Bolsa        string                   `json:"bolsa"`
	Datos        dominiobolsa.DatosOferta `json:"datos"`
	PublicadaEn  time.Time                `json:"publicada_en"`
	VenceAntesDe time.Time                `json:"vence_antes_de"`
	Estado       string                   `json:"estado"`
	Disposicion  *struct {
		Recibo        string    `json:"recibo"`
		ManifestadaEn time.Time `json:"manifestada_en"`
	} `json:"disposicion"`
}

// leerOfertasCandidato solo se usa dentro de la transacción que ya consumió
// la consulta propia del mismo candidato.
func leerOfertasCandidato(ctx context.Context, tx consultorPortal, candidato string, corte time.Time) ([]puertosbolsa.OfertaPortalCandidato, error) {
	var contenido []byte
	if err := tx.QueryRow(ctx, `SELECT `+funcionListarOfertasCandidato+`($1::text,$2::timestamptz)`, candidato, corte.UTC()).Scan(&contenido); err != nil {
		return nil, errorPortalCandidato(ctx, err)
	}
	var filas []ofertaCandidatoPostgreSQL
	if json.Unmarshal(contenido, &filas) != nil {
		return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	ofertas := make([]puertosbolsa.OfertaPortalCandidato, 0, len(filas))
	for _, f := range filas {
		switch f.Estado {
		case puertosbolsa.EstadoOfertaPortalAbierta, puertosbolsa.EstadoOfertaPortalPendienteResolucion,
			puertosbolsa.EstadoOfertaPortalResuelta, puertosbolsa.EstadoOfertaPortalAdjudicadaPropia:
		default:
			return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
		}
		if f.OfertaRef == "" || f.Bolsa == "" || f.Datos.Validar() != nil || f.PublicadaEn.IsZero() || !f.VenceAntesDe.After(f.PublicadaEn) || f.PublicadaEn.After(corte) {
			return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
		}
		o := puertosbolsa.OfertaPortalCandidato{OfertaRef: f.OfertaRef, Bolsa: f.Bolsa, Datos: f.Datos, PublicadaEn: f.PublicadaEn.UTC(), VenceAntesDe: f.VenceAntesDe.UTC(), Estado: f.Estado}
		if d := f.Disposicion; d != nil {
			if d.Recibo == "" || d.ManifestadaEn.IsZero() || d.ManifestadaEn.After(corte) {
				return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
			}
			o.Disposicion = &puertosbolsa.DisposicionPropiaPortal{Recibo: d.Recibo, ManifestadaEn: d.ManifestadaEn.UTC()}
		}
		ofertas = append(ofertas, o)
	}
	return ofertas, nil
}
