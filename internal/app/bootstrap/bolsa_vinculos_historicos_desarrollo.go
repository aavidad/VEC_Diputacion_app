package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	importacionpg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgresimportacionconvoca"
	protector "vec-diputacion-granada/internal/modules/bolsa/adapters/protectorstagingdesarrollo"
	"vec-diputacion-granada/internal/modules/bolsa/application/constitucion"
)

var huellaBolsaHistorica = regexp.MustCompile(`^[0-9a-f]{64}$`)

type ResultadoRellenoVinculos struct {
	Nuevos     int `json:"nuevos"`
	Existentes int `json:"existentes"`
}

// EjecutarRellenoVinculosBolsa recupera el acta desde el staging cifrado,
// calcula las mismas referencias que constituir-bolsa y pide a Bolsa SQL que
// las vincule por fila_numero. Aplicar=false deja todo en ROLLBACK.
func EjecutarRellenoVinculosBolsa(ctx context.Context, cfg config.Config, huella, categoria string, aplicar bool) (ResultadoRellenoVinculos, error) {
	var vacio ResultadoRellenoVinculos
	if ctx == nil || !cfg.DevelopmentEnabledByDoubleKey() || !huellaBolsaHistorica.MatchString(huella) {
		return vacio, ErrConstitucionBolsaNoDisponible
	}
	categoria, err := validarCategoriaImportacionConvoca(cfg, categoria)
	if err != nil {
		return vacio, err
	}
	material, err := cargarMaterialSeguridadDesarrollo(cfg)
	if err != nil {
		return vacio, err
	}
	defer borrarMaterialImportacionConvoca(material)
	p, err := protector.Nuevo(material.claveKMS)
	if err != nil {
		return vacio, err
	}
	derivador, err := protector.NuevoDerivadorCandidato(material.claveKMS)
	if err != nil {
		return vacio, err
	}
	poolImportacion, err := abrirPoolRecuperacionConvocaRelleno(ctx, cfg)
	if err != nil {
		return vacio, err
	}
	defer poolImportacion.Close()
	recuperador, err := importacionpg.NuevoRepositorioRecuperacionPostgreSQL(poolImportacion, p)
	if err != nil {
		return vacio, err
	}
	lote, _, existe, err := recuperador.RecuperarLote(ctx, huella, categoria)
	if err != nil {
		return vacio, err
	}
	if !existe {
		return vacio, constitucion.ErrActaNoEncontrada
	}
	filas, err := constitucion.DerivarFilasVinculo(lote, derivador)
	if err != nil {
		return vacio, err
	}
	contenido, err := json.Marshal(filas)
	if err != nil {
		return vacio, err
	}
	adminDSN := os.Getenv("VEC_PRINCIPAL_ADMIN_DATABASE_URL")
	if adminDSN == "" {
		return vacio, errors.New("falta conexion administrativa para relleno Bolsa")
	}
	adminConfig, err := pgx.ParseConfig(adminDSN)
	if err != nil {
		return vacio, errors.New("conexion administrativa invalida")
	}
	if err = validarTLSPostgreSQLBorradores(&adminConfig.Config, true); err != nil {
		return vacio, errors.New("conexion administrativa TLS invalida")
	}
	admin, err := pgx.ConnectConfig(ctx, adminConfig)
	if err != nil {
		return vacio, errors.New("conexion administrativa no disponible")
	}
	defer admin.Close(context.Background())
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return vacio, err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SET LOCAL ROLE vec_bolsa_llamamientos_propietario"); err != nil {
		return vacio, errors.New("autoridad propietaria de Bolsa no disponible")
	}
	var respuesta []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.rellenar_vinculos_candidato_v1($1::text,$2::jsonb,$3::timestamptz)`,
		lote.Acta.ActaRef, contenido, time.Now().UTC().Truncate(time.Microsecond)).Scan(&respuesta)
	if err != nil {
		return vacio, err
	}
	var resultado ResultadoRellenoVinculos
	if err = json.Unmarshal(respuesta, &resultado); err != nil || resultado.Nuevos < 0 || resultado.Existentes < 0 || resultado.Nuevos+resultado.Existentes != len(filas) {
		return vacio, errors.New("respuesta de relleno no confiable")
	}
	if aplicar {
		if err = tx.Commit(ctx); err != nil {
			return vacio, err
		}
	}
	return resultado, nil
}

func abrirPoolRecuperacionConvocaRelleno(ctx context.Context, cfg config.Config) (*pgxpool.Pool, error) {
	dsn, err := cfg.BolsaImportacionConvocaPostgreSQL.DSN()
	if err != nil {
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil || pc == nil || pc.ConnConfig == nil || validarTLSPostgreSQLBorradores(&pc.ConnConfig.Config, true) != nil {
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	pc.MaxConns = 2
	pc.MinConns = 0
	pc.ConnConfig.ConnectTimeout = 5 * time.Second
	if pc.ConnConfig.RuntimeParams == nil {
		pc.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-rellenar-vinculos-bolsa", "timezone": "UTC", "search_path": "pg_catalog,pg_temp", "default_transaction_read_only": "on", "statement_timeout": "15s", "lock_timeout": "3s", "idle_in_transaction_session_timeout": "15s"} {
		pc.ConnConfig.RuntimeParams[k] = v
	}
	pc.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		var autorizado bool
		err := c.QueryRow(ctx, `SELECT session_user=current_user
			AND pg_has_role(session_user,'vec_bolsa_importacion_convoca_recuperador','MEMBER')
			AND NOT pg_has_role(session_user,'vec_bolsa_importacion_convoca_ejecutor','MEMBER')`).Scan(&autorizado)
		if err != nil || !autorizado {
			return ErrPoolImportacionConvocaNoDisponible
		}
		return nil
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, ErrPoolImportacionConvocaNoDisponible
	}
	return pool, nil
}
