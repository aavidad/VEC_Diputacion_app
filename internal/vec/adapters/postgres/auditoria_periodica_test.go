package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type poolPeriodicoPrueba struct {
	txs    []*txPeriodicoPrueba
	usados int
}

func TestPeriodicoConfirmacionYConfiguracionNoEntreganDatosConCommitIncierto(t *testing.T) {
	ctx, _ := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	for _, modo := range []string{"confirmar", "configurar"} {
		t.Run(modo, func(t *testing.T) {
			tx := &txPeriodicoPrueba{raw: []byte(`{"auditoria_ref":"aud_v3_per_sintetico","secuencia":2,"configuracion_version":1,"configuracion_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`), commitErr: errors.New("respuesta_perdida")}
			p := &poolPeriodicoPrueba{txs: []*txPeriodicoPrueba{tx}}
			f := &FuenteCheckpointPeriodicoPostgreSQL{pool: p, maxRegistros: 10}
			if modo == "configurar" {
				v, h, err := f.ConfigurarCheckpointPeriodico(ctx, []byte(`{}`), strings.Repeat("0", 64))
				if !errors.Is(err, ErrCheckpointPeriodicoCommitIndeterminado) || v != 0 || h != "" {
					t.Fatal("configuración indeterminada no puede entregar datos de éxito")
				}
				return
			}
			c := domain.CheckpointDesarrollo{Esquema: domain.EsquemaCheckpointDesarrollo, Politica: domain.PoliticaCheckpoint{Version: 1, PoliticaRef: "politica:sintetica", PoliticaVersion: 1, ClaveRef: "clave:sintetica", ClaveVersion: 1, ProveedorKMS: "kms:sintetico", ProveedorKMSVersion: 1, ProveedorTSA: "tsa:sintetico", ProveedorTSAVersion: 1, OperacionTSA: "operacion:sintetica", Modo: "DESARROLLO"}, Cobertura: domain.CoberturaCheckpoint{CadenaID: "cadena:sintetica", PrimeraSecuencia: 1, UltimaSecuencia: 1, Registros: 1, AnteriorSHA256: strings.Repeat("0", 64), CabezaSHA256: strings.Repeat("a", 64)}}
			b, _ := c.Canonico()
			r := domain.ReciboCheckpointDesarrollo{Checkpoint: c, PinSPKISHA256: strings.Repeat("b", 64), TSA: domain.ReciboTSACheckpoint{Referencia: "tsa-desarrollo:hmac-sha256:" + strings.Repeat("c", 64), HuellaPreimagenSHA256: strings.Repeat("d", 64), HuellaCheckpointSHA256: domain.HuellaCheckpoint(b), Autoridad: "no_autoritativo", Esquema: "vec.tsa.desarrollo.v1"}, FirmaBase64: "fixture-unitaria"}
			a, err := f.ConfirmarCheckpoint(ctx, "captura_sintetica", r)
			if !errors.Is(err, ErrCheckpointPeriodicoCommitIndeterminado) || a != (ports.AcuseCheckpointPeriodico{}) {
				t.Fatal("confirmación indeterminada no puede entregar acuse")
			}
		})
	}
}

func (p *poolPeriodicoPrueba) BeginTx(ctx context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	if ctx.Err() != nil || o.IsoLevel != pgx.Serializable || o.AccessMode != pgx.ReadWrite {
		return nil, errors.New("transaccion_invalida")
	}
	if p.usados >= len(p.txs) {
		return nil, errors.New("transaccion_extra")
	}
	tx := p.txs[p.usados]
	p.usados++
	return tx, nil
}

type txPeriodicoPrueba struct {
	pgx.Tx
	queryErr, commitErr error
	raw                 []byte
	cerrada             bool
	consulta            string
	contextoCancelado   bool
}

func (tx *txPeriodicoPrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (tx *txPeriodicoPrueba) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	tx.consulta = q
	tx.contextoCancelado = ctx.Err() != nil
	return filaPeriodicaPrueba{raw: tx.raw, err: tx.queryErr}
}
func (tx *txPeriodicoPrueba) Rollback(context.Context) error {
	if tx.cerrada {
		return pgx.ErrTxClosed
	}
	tx.cerrada = true
	return nil
}
func (tx *txPeriodicoPrueba) Commit(context.Context) error { tx.cerrada = true; return tx.commitErr }

type filaPeriodicaPrueba struct {
	raw []byte
	err error
}

func (f filaPeriodicaPrueba) Scan(args ...any) error {
	if f.err != nil {
		return f.err
	}
	*args[0].(*[]byte) = append([]byte(nil), f.raw...)
	return nil
}
func TestPeriodicoNoEntregaAcuseSiCommitEsIncierto(t *testing.T) {
	tx := &txPeriodicoPrueba{raw: []byte(`{"estado":"no_vencido","acuse":{}}`), commitErr: errors.New("respuesta_perdida")}
	p := &poolPeriodicoPrueba{txs: []*txPeriodicoPrueba{tx}}
	f := &FuenteCheckpointPeriodicoPostgreSQL{pool: p, maxRegistros: 10}
	ctx, _ := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	c, err := f.CapturarCheckpointPendiente(ctx)
	if !errors.Is(err, ErrCheckpointPeriodicoCommitIndeterminado) || c.Estado != "" || p.usados != 1 {
		t.Fatal("un COMMIT incierto no debe entregar captura ni registrar falso rollback")
	}
}
func TestPeriodicoAuditaErrorDespuesDeRollback(t *testing.T) {
	original := &txPeriodicoPrueba{queryErr: &pgconn.PgError{Code: "42501", Message: "dato_que_no_debe_salir"}}
	registro := &txPeriodicoPrueba{raw: []byte(`{"auditoria_ref":"aud_v3_per_sintetico","secuencia":1}`)}
	p := &poolPeriodicoPrueba{txs: []*txPeriodicoPrueba{original, registro}}
	f := &FuenteCheckpointPeriodicoPostgreSQL{pool: p, maxRegistros: 10}
	ctx, _ := ports.ConCorrelacionIncidenciasPeticion(context.Background())
	_, err := f.CapturarCheckpointPendiente(ctx)
	if err == nil || strings.Contains(err.Error(), "dato_que_no_debe_salir") || p.usados != 2 || !original.cerrada ||
		!registro.cerrada || !strings.Contains(registro.consulta, "registrar_intento_periodico_v1") || registro.contextoCancelado {
		t.Fatal("debe conservar el fallo en otra transacción sin exponer mensajes SQL")
	}
}
