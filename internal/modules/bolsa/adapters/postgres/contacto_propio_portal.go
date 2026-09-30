package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	funcionConfirmarContactoPropioV1 = "vec_bolsa_llamamientos.confirmar_contacto_propio_v1"
	funcionLeerContactoCandidatoV1   = "vec_bolsa_llamamientos.leer_contacto_candidato_v1"
)

var _ puertosbolsa.RegistroConfirmacionContacto = (*RegistroConfirmacionContactoPostgreSQL)(nil)

// RegistroConfirmacionContactoPostgreSQL escribe la confirmación del contacto
// propio (Bolsa 000040): consume la decisión AD3-86 en la misma transacción.
type RegistroConfirmacionContactoPostgreSQL struct {
	portal *RegistroPortalCandidatoPostgreSQL
}

func NuevoRegistroConfirmacionContactoPostgreSQL(pool *pgxpool.Pool) (*RegistroConfirmacionContactoPostgreSQL, error) {
	portal, err := NuevoRegistroPortalCandidatoPostgreSQL(pool)
	if err != nil {
		return nil, err
	}
	return &RegistroConfirmacionContactoPostgreSQL{portal: portal}, nil
}

func (r *RegistroConfirmacionContactoPostgreSQL) ConfirmarContacto(ctx context.Context, s puertosbolsa.ConfirmacionContactoPortal) (puertosbolsa.ReciboConfirmacionContacto, error) {
	var vacio puertosbolsa.ReciboConfirmacionContacto
	if ctx == nil || r == nil || r.portal == nil || s.CandidatoRef == "" || s.Bolsa == "" || s.Version < 1 || s.ReciboRef == "" || s.ConfirmadaEn.IsZero() || s.Material.ValidarEstructura() != nil {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	tx, err := r.portal.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertir(tx)
	m := s.Material
	var recibo puertosbolsa.ReciboConfirmacionContacto
	err = tx.QueryRow(ctx, `SELECT reutilizada, recibo_ref, version, confirmada_en FROM `+funcionConfirmarContactoPropioV1+`($1::text,$2::text,$3::bigint,$4::text,$5::text,$6::timestamptz,$7::bytea,$8::bytea,$9::bytea,$10::bytea,$11::numeric,$12::numeric,$13::bytea,$14::bytea,$15::bytea,$16::bytea)`,
		s.CandidatoRef, s.Bolsa, s.Version, s.Clave, s.ReciboRef, s.ConfirmadaEn.UTC(),
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&recibo.Reutilizada, &recibo.ReciboRef, &recibo.Version, &recibo.ConfirmadaEn)
	if err != nil {
		return vacio, errorConfirmacionContacto(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorConfirmacionContacto(ctx, err)
	}
	recibo.ConfirmadaEn = recibo.ConfirmadaEn.UTC()
	return recibo, nil
}

func errorConfirmacionContacto(ctx context.Context, err error) error {
	var pgErr *pgconn.PgError
	if (ctx == nil || ctx.Err() == nil) && errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "VBC01":
			return puertosbolsa.ErrPortalClaveReutilizada
		case "VBC02":
			return puertosbolsa.ErrPortalContactoCambiado
		case "VBC03":
			return puertosbolsa.ErrPortalSinContacto
		case "VBC04":
			return puertosbolsa.ErrPortalContactoYaConfirmado
		}
	}
	return errorPortalCandidato(ctx, err)
}

type contactoCandidatoPostgreSQL struct {
	Bolsa   string `json:"bolsa"`
	Version int64  `json:"version"`
	Origen  *struct {
		Origen       string    `json:"origen"`
		VigenteHasta time.Time `json:"vigente_hasta"`
		UltimoDia    string    `json:"ultimo_dia"`
	} `json:"origen"`
	ConfirmadaEn *time.Time `json:"confirmada_en"`
}

// leerContactosCandidato solo se usa dentro de la transacción que ya consumió
// la consulta propia del mismo candidato.
func leerContactosCandidato(ctx context.Context, tx consultorPortal, candidato string, corte time.Time) ([]puertosbolsa.ContactoPortalCandidato, error) {
	var contenido []byte
	if err := tx.QueryRow(ctx, `SELECT `+funcionLeerContactoCandidatoV1+`($1::text,$2::timestamptz)`, candidato, corte.UTC()).Scan(&contenido); err != nil {
		return nil, errorPortalCandidato(ctx, err)
	}
	var filas []contactoCandidatoPostgreSQL
	if json.Unmarshal(contenido, &filas) != nil {
		return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	contactos := make([]puertosbolsa.ContactoPortalCandidato, 0, len(filas))
	for _, f := range filas {
		if f.Bolsa == "" || f.Version < 1 || (f.ConfirmadaEn != nil && f.ConfirmadaEn.After(corte)) {
			return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
		}
		c := puertosbolsa.ContactoPortalCandidato{Bolsa: f.Bolsa, Version: f.Version, ConfirmadaEn: utcOpcional(f.ConfirmadaEn)}
		if o := f.Origen; o != nil {
			if _, err := time.Parse(time.DateOnly, o.UltimoDia); err != nil || o.Origen != "convoca" || o.VigenteHasta.IsZero() {
				return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
			}
			c.Origen = &puertosbolsa.OrigenContactoPortal{VigenteHasta: o.VigenteHasta.UTC(), UltimoDia: o.UltimoDia}
		}
		contactos = append(contactos, c)
	}
	return contactos, nil
}
