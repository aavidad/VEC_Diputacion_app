package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrCheckpointPeriodicoPostgreSQL = errors.New("checkpoint_periodico_postgresql_no_disponible")
var ErrCheckpointPeriodicoCommitIndeterminado = errors.New("checkpoint_periodico_commit_indeterminado")

type poolCheckpointPeriodico interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// FuenteCheckpointPeriodicoPostgreSQL sólo ejecuta las fachadas técnicas AD186.
// No obtiene registros personales, acepta cobertura ni usa tablas de módulos.
type FuenteCheckpointPeriodicoPostgreSQL struct {
	pool         poolCheckpointPeriodico
	maxRegistros uint64
	rol          string
}

var _ ports.FuenteCheckpointPeriodico = (*FuenteCheckpointPeriodicoPostgreSQL)(nil)

func NuevaFuenteCheckpointPeriodicoPostgreSQL(pool *pgxpool.Pool, maxRegistros uint64) (*FuenteCheckpointPeriodicoPostgreSQL, error) {
	if pool == nil || maxRegistros == 0 || maxRegistros > 9007199254740991 {
		return nil, ErrCheckpointPeriodicoPostgreSQL
	}
	return &FuenteCheckpointPeriodicoPostgreSQL{pool: pool, maxRegistros: maxRegistros, rol: "vec_auditoria_periodica_sellador"}, nil
}

func (f *FuenteCheckpointPeriodicoPostgreSQL) ConfigurarCheckpointPeriodico(ctx context.Context, configuracion []byte, preimagen string) (uint64, string, error) {
	if f == nil {
		return 0, "", ErrCheckpointPeriodicoPostgreSQL
	}
	configurador := *f
	configurador.rol = "vec_auditoria_periodica_configurador"
	var r struct {
		Version uint64            `json:"configuracion_version"`
		Huella  string            `json:"configuracion_sha256"`
		Acuse   acusePeriodicoSQL `json:"acuse"`
	}
	err := configurador.operar(ctx, "configurar_sello_periodico_v1", `SELECT vec_autorizacion_atestada_v3.configurar_sello_periodico_v1($2::jsonb,$3::text,$1::text)`, &r, string(configuracion), preimagen)
	if err != nil {
		return 0, "", err
	}
	return r.Version, r.Huella, err
}

type acusePeriodicoSQL struct {
	AuditoriaRef       string    `json:"auditoria_ref"`
	Secuencia          uint64    `json:"secuencia"`
	HuellaSHA256       string    `json:"huella_sha256"`
	RegistradaEn       time.Time `json:"registrada_en"`
	CorrelacionRef     string    `json:"correlacion_ref"`
	CapturaRef         string    `json:"captura_ref"`
	ReciboHuellaSHA256 string    `json:"recibo_huella_sha256"`
}

func (a acusePeriodicoSQL) puerto() ports.AcuseCheckpointPeriodico {
	return ports.AcuseCheckpointPeriodico{AuditoriaRef: a.AuditoriaRef, Secuencia: a.Secuencia, HuellaSHA256: a.HuellaSHA256,
		RegistradaEn: a.RegistradaEn.UTC(), CorrelacionRef: a.CorrelacionRef, CapturaRef: a.CapturaRef, ReciboHuellaSHA256: a.ReciboHuellaSHA256}
}

type capturaPeriodicaSQL struct {
	ReciboTexto          string                             `json:"recibo_texto"`
	Estado               string                             `json:"estado"`
	CapturaRef           string                             `json:"captura_ref"`
	ConfiguracionVersion uint64                             `json:"configuracion_version"`
	ConfiguracionSHA256  string                             `json:"configuracion_sha256"`
	PinSPKISHA256        string                             `json:"pin_spki_sha256"`
	Checkpoint           domain.CheckpointDesarrollo        `json:"checkpoint"`
	Acuse                acusePeriodicoSQL                  `json:"acuse"`
	Recibo               *domain.ReciboCheckpointDesarrollo `json:"recibo"`
	AcuseConfirmacion    acusePeriodicoSQL                  `json:"acuse_confirmacion"`
}

func (c capturaPeriodicaSQL) puerto() ports.CapturaCheckpointPeriodico {
	return ports.CapturaCheckpointPeriodico{Estado: c.Estado, CapturaRef: c.CapturaRef, ConfiguracionVersion: c.ConfiguracionVersion,
		ConfiguracionSHA256: c.ConfiguracionSHA256, PinSPKISHA256: c.PinSPKISHA256, Checkpoint: c.Checkpoint, Acuse: c.Acuse.puerto()}
}

func (f *FuenteCheckpointPeriodicoPostgreSQL) CapturarCheckpointPendiente(ctx context.Context) (ports.CapturaCheckpointPeriodico, error) {
	if f == nil {
		return ports.CapturaCheckpointPeriodico{}, ErrCheckpointPeriodicoPostgreSQL
	}
	var r capturaPeriodicaSQL
	err := f.operar(ctx, "capturar_sello_periodico_v1", `SELECT vec_autorizacion_atestada_v3.capturar_sello_periodico_v1($1::text,$2::numeric)`, &r, strconv.FormatUint(f.maxRegistros, 10))
	if err != nil {
		return ports.CapturaCheckpointPeriodico{}, err
	}
	if r.Estado == "denegado" {
		return ports.CapturaCheckpointPeriodico{}, ErrCheckpointPeriodicoPostgreSQL
	}
	return r.puerto(), nil
}
func (f *FuenteCheckpointPeriodicoPostgreSQL) ConfirmarCheckpoint(ctx context.Context, ref string, r domain.ReciboCheckpointDesarrollo) (ports.AcuseCheckpointPeriodico, error) {
	if _, err := r.CanonicoParaFirma(); err != nil {
		return ports.AcuseCheckpointPeriodico{}, ErrCheckpointPeriodicoPostgreSQL
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return ports.AcuseCheckpointPeriodico{}, ErrCheckpointPeriodicoPostgreSQL
	}
	var a acusePeriodicoSQL
	// $1 siempre es la correlación emitida por el contexto privado del proceso.
	err = f.operar(ctx, "confirmar_sello_periodico_v1", `SELECT vec_autorizacion_atestada_v3.confirmar_sello_periodico_v1($2::text,$3::text,$4::text,$1::text)`, &a, ref, string(raw), domain.HuellaCheckpoint(raw))
	if err != nil {
		return ports.AcuseCheckpointPeriodico{}, err
	}
	return a.puerto(), err
}

// Recuperar permite resolver COMMIT incierto con la referencia original,
// revalidando la autoridad actual y auditando la lectura en su misma TX.
func (f *FuenteCheckpointPeriodicoPostgreSQL) RecuperarCheckpoint(ctx context.Context, ref string) (ports.CapturaCheckpointPeriodico, *domain.ReciboCheckpointDesarrollo, ports.AcuseCheckpointPeriodico, string, error) {
	var r capturaPeriodicaSQL
	err := f.operar(ctx, "capturar_sello_periodico_v1", `SELECT vec_autorizacion_atestada_v3.recuperar_sello_periodico_v1($2::text,$1::text)`, &r, ref)
	if err != nil {
		return ports.CapturaCheckpointPeriodico{}, nil, ports.AcuseCheckpointPeriodico{}, "", err
	}
	c := r.puerto()
	if r.Estado == "confirmado" {
		c.Estado = "pendiente"
	}
	return c, r.Recibo, r.AcuseConfirmacion.puerto(), r.ReciboTexto, nil
}

func (f *FuenteCheckpointPeriodicoPostgreSQL) operar(ctx context.Context, accion, consulta string, destino any, args ...any) error {
	correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if f == nil || valorNuloPostgreSQL(f.pool) || ctx == nil || !ok {
		return ErrCheckpointPeriodicoPostgreSQL
	}
	params := append([]any{"correlacion_" + correlacion}, args...)
	err := f.transaccion(ctx, consulta, destino, params...)
	if err == nil || errors.Is(err, ErrCheckpointPeriodicoCommitIndeterminado) {
		return err
	}
	resultado := "error"
	var p *pgconn.PgError
	if errors.As(err, &p) && p.Code == "42501" {
		resultado = "denegado"
	}
	if f.RegistrarFalloCheckpoint(ctx, accion, resultado) != nil {
		return ErrCheckpointPeriodicoPostgreSQL
	}
	return ErrCheckpointPeriodicoPostgreSQL
}

// Se invoca una vez cerrado el intento original. No incluye mensajes del
// proveedor ni marca un COMMIT incierto como una operación sin efecto.
func (f *FuenteCheckpointPeriodicoPostgreSQL) RegistrarFalloCheckpoint(ctx context.Context, accion, resultado string) error {
	corr, ok := ports.CorrelacionIncidenciasPeticion(ctx)
	if ctx == nil || !ok || f == nil || valorNuloPostgreSQL(f.pool) {
		return ErrCheckpointPeriodicoPostgreSQL
	}
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(10*time.Second))
	defer cancel()
	var a acusePeriodicoSQL
	return f.transaccion(c, `SELECT vec_autorizacion_atestada_v3.registrar_intento_periodico_v1($1::text,$2::text,$3::text)`, &a, accion, resultado, "correlacion_"+corr)
}

func (f *FuenteCheckpointPeriodicoPostgreSQL) transaccion(ctx context.Context, consulta string, destino any, args ...any) error {
	tx, err := f.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	rollback := func() error {
		c, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(3*time.Second))
		defer cancel()
		return tx.Rollback(c)
	}
	defer func() { _ = rollback() }()
	// LOGIN propio NOINHERIT: activa únicamente la autoridad fija de esta
	// fachada, nunca un rol enviado por el cliente ni membresía de owner.
	rol := `SET LOCAL ROLE vec_auditoria_periodica_sellador`
	if f.rol == "vec_auditoria_periodica_configurador" {
		rol = `SET LOCAL ROLE vec_auditoria_periodica_configurador`
	}
	if _, err = tx.Exec(ctx, rol); err != nil {
		if rollback() != nil {
			return ErrCheckpointPeriodicoCommitIndeterminado
		}
		return err
	}
	if _, err = tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		if rollback() != nil {
			return ErrCheckpointPeriodicoCommitIndeterminado
		}
		return err
	}
	var raw []byte
	if err = tx.QueryRow(ctx, consulta, args...).Scan(&raw); err != nil {
		if rollback() != nil {
			return ErrCheckpointPeriodicoCommitIndeterminado
		}
		return err
	}
	if len(raw) > 32768 || json.Unmarshal(raw, destino) != nil {
		if rollback() != nil {
			return ErrCheckpointPeriodicoCommitIndeterminado
		}
		return ErrCheckpointPeriodicoPostgreSQL
	}
	if tx.Commit(ctx) != nil {
		return ErrCheckpointPeriodicoCommitIndeterminado
	}
	return nil
}
