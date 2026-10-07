package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/reglas"
)

const consultaAjustesReglasEnV1 = `SELECT version, huella_sha256, ajustes_canonico, vigente_desde
FROM vec_contratacion_temporal.leer_ajustes_reglas_en_v1($1::text, $2::timestamptz)`

const (
	catalogoAjustesCT       = reglas.CatalogoContratacionTemporal + reglas.SufijoCatalogoAjustes
	limiteConsultaAjustesCT = 5 * time.Second
	maximoCanonicoAjustesCT = 16 * 1024
	maximoVersionAjustesCT  = 9999999 // CT-148: regla_ajuste_version_v1.version
)

type consultadorAjustesReglasCT interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// ConsultaAjustesReglasPostgreSQL lee únicamente la versión que selecciona
// CT-148 para el instante solicitado. El rol de conexión debe ser el ejecutor
// de Contratación temporal, al que CT-148 concede EXECUTE sobre esa función.
type ConsultaAjustesReglasPostgreSQL struct {
	consultador consultadorAjustesReglasCT
}

var _ reglas.ConsultaAjustes = (*ConsultaAjustesReglasPostgreSQL)(nil)

func NuevaConsultaAjustesReglasPostgreSQL(pool *pgxpool.Pool) (*ConsultaAjustesReglasPostgreSQL, error) {
	return nuevaConsultaAjustesReglasPostgreSQL(pool)
}

func nuevaConsultaAjustesReglasPostgreSQL(consultador consultadorAjustesReglasCT) (*ConsultaAjustesReglasPostgreSQL, error) {
	if dependenciaNula(consultador) {
		return nil, reglas.ErrAjustesNoDisponibles
	}
	return &ConsultaAjustesReglasPostgreSQL{consultador: consultador}, nil
}

func (c *ConsultaAjustesReglasPostgreSQL) AjustesVigentesEn(ctx context.Context, catalogoAjustesID string, instante time.Time) (reglas.VersionAjustes, bool, error) {
	vacio := reglas.VersionAjustes{}
	if ctx == nil || c == nil || dependenciaNula(c.consultador) ||
		catalogoAjustesID != catalogoAjustesCT || instante.IsZero() {
		return vacio, false, reglas.ErrAjustesNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return vacio, false, err
	}
	consultaCtx, cancelar := context.WithTimeout(ctx, plazoarranque.Ampliar(limiteConsultaAjustesCT))
	defer cancelar()

	var version int64
	var huella, canonico string
	var desde time.Time
	err := c.consultador.QueryRow(consultaCtx, consultaAjustesReglasEnV1, catalogoAjustesID, instante).Scan(
		&version, &huella, &canonico, &desde,
	)
	if errContexto := consultaCtx.Err(); errContexto != nil {
		return vacio, false, errContexto
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return vacio, false, nil
	}
	if err != nil {
		return vacio, false, normalizarErrorConsultaAjustesReglasCT(err)
	}
	if version < 1 || version > maximoVersionAjustesCT || desde.IsZero() || desde.After(instante) ||
		len(canonico) == 0 || len(canonico) > maximoCanonicoAjustesCT || len(huella) != 64 {
		return vacio, false, reglas.ErrAjustesNoDisponibles
	}
	var ajustes map[string]map[string]string
	if err := json.Unmarshal([]byte(canonico), &ajustes); err != nil || ajustes == nil {
		return vacio, false, reglas.ErrAjustesNoDisponibles
	}
	canonicoComprobado, err := reglas.CanonicoAjustes(ajustes)
	if err != nil || !bytes.Equal(canonicoComprobado, []byte(canonico)) {
		return vacio, false, reglas.ErrAjustesNoDisponibles
	}
	suma := sha256.Sum256(canonicoComprobado)
	if hex.EncodeToString(suma[:]) != huella {
		return vacio, false, reglas.ErrAjustesNoDisponibles
	}
	if err := consultaCtx.Err(); err != nil {
		return vacio, false, err
	}
	return reglas.VersionAjustes{
		CatalogoID:   catalogoAjustesID,
		Version:      int(version),
		HuellaSHA256: huella,
		VigenteDesde: desde,
		Ajustes:      ajustes,
	}, true, nil
}

func normalizarErrorConsultaAjustesReglasCT(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && (pg.Code == "55P03" || pg.Code == "40001") {
		return reglas.ErrAjustesConflicto
	}
	return reglas.ErrAjustesNoDisponibles
}
