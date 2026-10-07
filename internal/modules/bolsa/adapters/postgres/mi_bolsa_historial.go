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

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const funcionConsultarHistorialMiBolsaV1 = "vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1"

var _ puertosbolsa.ConsultaHistorialMiBolsa = (*ConsultaMiBolsaPostgreSQL)(nil)

func (r *ConsultaMiBolsaPostgreSQL) ConsultarHistorialMiBolsa(ctx context.Context, s puertosbolsa.SolicitudConsultaHistorialMiBolsa) (puertosbolsa.PaginaHistorialMiBolsa, error) {
	var resultado puertosbolsa.PaginaHistorialMiBolsa
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var err error
		resultado, err = r.consultarHistorialMiBolsaIntento(ctx, s)
		return err
	})
	if err != nil {
		var carrera errorCarreraLecturaMiBolsa
		if errors.As(err, &carrera) {
			err = carrera.error
		}
		if ctx != nil && ctx.Err() != nil {
			err = ctx.Err()
		}
		return puertosbolsa.PaginaHistorialMiBolsa{}, err
	}
	return resultado, nil
}

// Cada intento consume la misma autorización y audita la lectura dentro de una
// transacción nueva. El resultado solo se publica después de confirmar el COMMIT.
func (r *ConsultaMiBolsaPostgreSQL) consultarHistorialMiBolsaIntento(ctx context.Context, s puertosbolsa.SolicitudConsultaHistorialMiBolsa) (puertosbolsa.PaginaHistorialMiBolsa, error) {
	var vacia puertosbolsa.PaginaHistorialMiBolsa
	if ctx == nil || r == nil || valorNulo(r.pool) || s.CandidatoRef == "" || s.ConsultadaEn.IsZero() || s.Pagina < 1 || s.Pagina > puertosbolsa.MaximaPaginaHistorialMiBolsa || s.Material.ValidarEstructura() != nil {
		return vacia, puertosbolsa.ErrConsultaHistorialMiBolsaInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, errorIntentoHistorialMiBolsa(ctx, err)
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacia, errorIntentoHistorialMiBolsa(ctx, err)
	}
	m := s.Material
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT `+funcionConsultarHistorialMiBolsaV1+`($1::text,$2::timestamptz,$3::integer,$4::bytea,$5::bytea,$6::bytea,$7::bytea,$8::numeric,$9::numeric,$10::bytea,$11::bytea,$12::bytea,$13::bytea)`,
		s.CandidatoRef, s.ConsultadaEn.UTC(), s.Pagina,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&contenido)
	if err != nil {
		return vacia, errorIntentoHistorialMiBolsa(ctx, err)
	}
	defer borrarBytesPostgreSQL(contenido)
	resultado, err := decodificarHistorialMiBolsa(contenido, s.ConsultadaEn, s.Pagina)
	if err != nil {
		return vacia, err
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacia, errorIntentoHistorialMiBolsa(ctx, err)
	}
	return resultado, nil
}

// Los campos de cada variante se leen con un DTO explícito; se rechazan
// propiedades inesperadas, incluidos referencias internas o datos personales.
type hechoHistorialMiBolsaJSON struct {
	Clase          string     `json:"clase"`
	Bolsa          string     `json:"bolsa"`
	Categoria      string     `json:"categoria"`
	OcurridoEn     time.Time  `json:"ocurrido_en"`
	Tipo           string     `json:"tipo,omitempty"`
	Inicio         *time.Time `json:"inicio,omitempty"`
	FinPrevisto    *time.Time `json:"fin_previsto,omitempty"`
	ModalidadClave *string    `json:"modalidad_clave,omitempty"`
	Procedencia    string     `json:"procedencia,omitempty"`
	Canal          string     `json:"canal,omitempty"`
	Resultado      string     `json:"resultado,omitempty"`
	Respuesta      string     `json:"respuesta,omitempty"`
	Modo           string     `json:"modo,omitempty"`
	Estado         string     `json:"estado,omitempty"`
}

type paginaHistorialMiBolsaJSON struct {
	ConsultadaEn time.Time                   `json:"consultada_en"`
	Pagina       int                         `json:"pagina"`
	Tamano       int                         `json:"tamano"`
	HayMas       bool                        `json:"hay_mas"`
	Items        []hechoHistorialMiBolsaJSON `json:"items"`
}

func decodificarHistorialMiBolsa(contenido []byte, esperada time.Time, pagina int) (puertosbolsa.PaginaHistorialMiBolsa, error) {
	var vacia puertosbolsa.PaginaHistorialMiBolsa
	if len(contenido) == 0 || len(contenido) > 131072 {
		return vacia, puertosbolsa.ErrResultadoHistorialMiBolsaInvalido
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var p paginaHistorialMiBolsaJSON
	if dec.Decode(&p) != nil || dec.Decode(&struct{}{}) != io.EOF || !p.ConsultadaEn.Equal(esperada.UTC()) || p.Pagina != pagina || p.Tamano != puertosbolsa.TamanoPaginaHistorialMiBolsa || len(p.Items) > p.Tamano || p.HayMas && len(p.Items) != p.Tamano {
		return vacia, puertosbolsa.ErrResultadoHistorialMiBolsaInvalido
	}
	r := puertosbolsa.PaginaHistorialMiBolsa{ConsultadaEn: p.ConsultadaEn.UTC(), Pagina: p.Pagina, Tamano: p.Tamano, HayMas: p.HayMas, Items: make([]puertosbolsa.HechoHistorialMiBolsa, 0, len(p.Items))}
	for _, x := range p.Items {
		h := puertosbolsa.HechoHistorialMiBolsa{Clase: x.Clase, Bolsa: x.Bolsa, Categoria: x.Categoria, OcurridoEn: x.OcurridoEn.UTC(), Tipo: x.Tipo, ModalidadClave: x.ModalidadClave, Procedencia: x.Procedencia, Canal: x.Canal, Resultado: x.Resultado, Respuesta: x.Respuesta, Modo: x.Modo, Estado: x.Estado}
		if x.Inicio != nil {
			v := x.Inicio.UTC()
			h.Inicio = &v
		}
		if x.FinPrevisto != nil {
			v := x.FinPrevisto.UTC()
			h.FinPrevisto = &v
		}
		r.Items = append(r.Items, h)
	}
	return r, nil
}

func errorHistorialMiBolsa(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "22000", "22023":
			return puertosbolsa.ErrConsultaHistorialMiBolsaInvalida
		}
	}
	return puertosbolsa.ErrHistorialMiBolsaNoDisponible
}

func errorIntentoHistorialMiBolsa(ctx context.Context, err error) error {
	nominal := errorHistorialMiBolsa(ctx, err)
	if postgresql.EsCarreraSerializable(err) {
		return errorCarreraLecturaMiBolsa{nominal}
	}
	return nominal
}
