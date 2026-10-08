package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	app "vec-diputacion-granada/internal/modules/bolsa/application/ajustesreglas"
	"vec-diputacion-granada/internal/vec/reglas"
)

const leerAjustesBolsaEnSQL = `SELECT version,huella_sha256,ajustes_canonico,vigente_desde FROM vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1($1::text,$2::timestamptz)`
const leerCabezaAjustesBolsaSQL = `SELECT version,huella_sha256,ajustes_canonico,publicada_en,vigente_desde FROM vec_bolsa_llamamientos.leer_cabeza_ajustes_reglas_v1($1::text)`

type consultaAjustesBolsaSQL interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// ConsultaAjustesReglasBolsaPostgreSQL sólo consume las funciones B88 sin
// datos personales. La conexión debe usar el ejecutor Bolsa nominal.
type ConsultaAjustesReglasBolsaPostgreSQL struct{ db consultaAjustesBolsaSQL }

func NuevaConsultaAjustesReglasBolsaPostgreSQL(db *pgxpool.Pool) (*ConsultaAjustesReglasBolsaPostgreSQL, error) {
	if db == nil {
		return nil, reglas.ErrAjustesNoDisponibles
	}
	return &ConsultaAjustesReglasBolsaPostgreSQL{db: db}, nil
}

func (c *ConsultaAjustesReglasBolsaPostgreSQL) AjustesVigentesEn(ctx context.Context, id string, instante time.Time) (reglas.VersionAjustes, bool, error) {
	if c == nil || c.db == nil || ctx == nil || id != app.CatalogoAjustes || instante.IsZero() {
		return reglas.VersionAjustes{}, false, reglas.ErrAjustesNoDisponibles
	}
	var version int64
	var huella, canon string
	var desde time.Time
	err := c.db.QueryRow(ctx, leerAjustesBolsaEnSQL, id, instante).Scan(&version, &huella, &canon, &desde)
	if errors.Is(err, pgx.ErrNoRows) {
		return reglas.VersionAjustes{}, false, nil
	}
	if err != nil {
		return reglas.VersionAjustes{}, false, reglas.ErrAjustesNoDisponibles
	}
	v, e := restaurarVersionAjustesBolsa(id, version, huella, canon, desde)
	if e != nil || desde.After(instante) {
		return reglas.VersionAjustes{}, false, reglas.ErrAjustesNoDisponibles
	}
	return v, true, nil
}

func (c *ConsultaAjustesReglasBolsaPostgreSQL) LeerCabeza(ctx context.Context) (*reglas.VersionAjustes, error) {
	if c == nil || c.db == nil || ctx == nil || ctx.Err() != nil {
		return nil, app.ErrNoDisponible
	}
	var version int64
	var huella, canon string
	var publicada, desde time.Time
	err := c.db.QueryRow(ctx, leerCabezaAjustesBolsaSQL, app.CatalogoAjustes).Scan(&version, &huella, &canon, &publicada, &desde)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil || publicada.IsZero() || desde.Before(publicada) {
		return nil, app.ErrNoDisponible
	}
	v, e := restaurarVersionAjustesBolsa(app.CatalogoAjustes, version, huella, canon, desde)
	if e != nil {
		return nil, app.ErrNoDisponible
	}
	return &v, nil
}

func restaurarVersionAjustesBolsa(id string, version int64, huella, canon string, desde time.Time) (reglas.VersionAjustes, error) {
	if id != app.CatalogoAjustes || version < 1 || version > 9_999_999 || desde.IsZero() || len(canon) < 2 || len(canon) > 16*1024 {
		return reglas.VersionAjustes{}, reglas.ErrAjustesNoDisponibles
	}
	var ajustes map[string]map[string]string
	if json.Unmarshal([]byte(canon), &ajustes) != nil || ajustes == nil {
		return reglas.VersionAjustes{}, reglas.ErrAjustesNoDisponibles
	}
	canonico, err := reglas.CanonicoAjustes(ajustes)
	if err != nil || !bytes.Equal(canonico, []byte(canon)) {
		return reglas.VersionAjustes{}, reglas.ErrAjustesNoDisponibles
	}
	h, err := reglas.HuellaAjustes(ajustes)
	if err != nil || h != huella {
		return reglas.VersionAjustes{}, reglas.ErrAjustesNoDisponibles
	}
	return reglas.VersionAjustes{CatalogoID: id, Version: int(version), HuellaSHA256: huella, VigenteDesde: desde, Ajustes: ajustes}, nil
}

var _ reglas.ConsultaAjustes = (*ConsultaAjustesReglasBolsaPostgreSQL)(nil)
