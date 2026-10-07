package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

type consultaPoolPrueba struct {
	tx       pgx.Tx
	err      error
	llamadas int
	opciones pgx.TxOptions
}

func (p *consultaPoolPrueba) BeginTx(_ context.Context, o pgx.TxOptions) (pgx.Tx, error) {
	p.llamadas++
	p.opciones = o
	return p.tx, p.err
}

type consultaTxPrueba struct {
	pgx.Tx
	raw                             []byte
	errExec, errConsulta, errCommit error
	despuesConsulta, despuesCommit  func()
	commits, rollbacks              int
	ajustes, consulta               string
	argumentos                      []any
}

func (tx *consultaTxPrueba) Exec(_ context.Context, q string, _ ...any) (pgconn.CommandTag, error) {
	tx.ajustes = q
	return pgconn.CommandTag{}, tx.errExec
}
func (tx *consultaTxPrueba) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	tx.consulta, tx.argumentos = q, args
	if tx.despuesConsulta != nil {
		tx.despuesConsulta()
	}
	return filaPrueba{tx.raw, tx.errConsulta}
}
func (tx *consultaTxPrueba) Commit(context.Context) error {
	tx.commits++
	if tx.despuesCommit != nil {
		tx.despuesCommit()
	}
	return tx.errCommit
}
func (tx *consultaTxPrueba) Rollback(context.Context) error { tx.rollbacks++; return nil }

func consultaJSONPrueba(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestConsultaPropiaConfirmaFichaActualYAusenciaAuditada(t *testing.T) {
	for _, codigo := range []string{"obtenida", "no_encontrada"} {
		t.Run(codigo, func(t *testing.T) {
			o, resultado := consultaOrdenPrueba(t)
			if codigo == "no_encontrada" {
				resultado.Codigo, resultado.HechoActual, resultado.ReciboConsulta.VersionConsultada = codigo, nil, 0
			}
			tx := &consultaTxPrueba{raw: consultaJSONPrueba(t, resultado)}
			pool := &consultaPoolPrueba{tx: tx}
			repo, _ := nuevaConsulta(pool)
			out, err := repo.ConsultarActual(context.Background(), o)
			if err != nil || !reflect.DeepEqual(out, resultado) || tx.commits != 1 || tx.rollbacks != 1 ||
				pool.opciones.IsoLevel != pgx.Serializable || pool.opciones.AccessMode != pgx.ReadWrite ||
				tx.consulta != consultaHechoPropio || len(tx.argumentos) != 11 || !bytes.Equal(tx.argumentos[0].([]byte), o.SelectorCanonico) {
				t.Fatal("consulta no confirmó el resultado y contrato exactos", err)
			}
			for _, ajuste := range []string{"'search_path','pg_catalog'", "'row_security','on'", "'timezone','UTC'", "'lock_timeout','2s'", "'statement_timeout','15s'", "'idle_in_transaction_session_timeout','20s'"} {
				if !strings.Contains(tx.ajustes, ajuste) {
					t.Fatal("falta ajuste transaccional", ajuste)
				}
			}
		})
	}
}

func TestConsultaPropiaNoConfirmaRespuestaAlterada(t *testing.T) {
	for _, caso := range []string{"version", "hecho", "decision", "correlacion", "consumo", "fecha", "recibo_ausente", "ausencia_con_version", "pii", "actor_anidado", "trailing", "demasiado", "null"} {
		t.Run(caso, func(t *testing.T) {
			o, resultado := consultaOrdenPrueba(t)
			switch caso {
			case "version":
				resultado.ReciboConsulta.VersionConsultada++
			case "hecho":
				resultado.HechoActual.Referencia = "hecho:ajeno"
			case "decision":
				resultado.ReciboConsulta.DecisionRef = "decision:ajena"
			case "correlacion":
				resultado.ReciboConsulta.CorrelacionRef = "correlacion:ajena"
			case "consumo":
				resultado.ReciboConsulta.ConsumoHuellaSHA256 = "00"
			case "fecha":
				resultado.ReciboConsulta.ConsultadaEn = resultado.ReciboConsulta.ConsultadaEn.AddDate(-1, 0, 0)
			case "recibo_ausente":
				resultado.ReciboConsulta = nil
			case "ausencia_con_version":
				resultado.Codigo, resultado.HechoActual = "no_encontrada", nil
			}
			raw := consultaJSONPrueba(t, resultado)
			switch caso {
			case "pii":
				raw = append([]byte(`{"dni":"sintetico",`), raw[1:]...)
			case "actor_anidado":
				raw = bytes.Replace(raw, []byte(`"hecho_actual":{`), []byte(`"hecho_actual":{"persona_ref":"persona:ajena",`), 1)
			case "trailing":
				raw = append(raw, []byte(" {}")...)
			case "demasiado":
				raw = bytes.Repeat([]byte{' '}, 65537)
			case "null":
				raw = []byte("null")
			}
			tx := &consultaTxPrueba{raw: raw}
			repo, _ := nuevaConsulta(&consultaPoolPrueba{tx: tx})
			out, err := repo.ConsultarActual(context.Background(), o)
			if !errors.Is(err, ports.ErrConsultaNoDisponible) || !reflect.DeepEqual(out, ports.ResultadoConsultaPropia{}) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("respuesta alterada confirmada o expuesta", err)
			}
		})
	}
}

func TestConsultaPropiaOrdenAlteradaNoAlcanzaSQL(t *testing.T) {
	for _, caso := range []string{"selector", "huella", "persona", "hecho", "contexto", "material"} {
		t.Run(caso, func(t *testing.T) {
			o, _ := consultaOrdenPrueba(t)
			switch caso {
			case "selector":
				o.SelectorCanonico = append(o.SelectorCanonico, ' ')
			case "huella":
				o.HuellaConsultaSHA256 = strings.Repeat("0", 64)
			case "persona":
				o.PersonaRef = "persona:ajena"
			case "hecho":
				o.HechoRef = "hecho:ajeno"
			case "contexto":
				o.Autorizacion.Contexto.RegistroContextoRef = "contexto:ajeno"
			case "material":
				o.Autorizacion.Material = ports.AutorizacionOperacion{}.Material
			}
			pool := &consultaPoolPrueba{tx: &consultaTxPrueba{}}
			repo, _ := nuevaConsulta(pool)
			out, err := repo.ConsultarActual(context.Background(), o)
			if err == nil || pool.llamadas != 0 || !reflect.DeepEqual(out, ports.ResultadoConsultaPropia{}) {
				t.Fatal("orden alterada alcanzó SQL")
			}
		})
	}
}

func TestConsultaPropiaFallosTransaccionalesNoExponenDatos(t *testing.T) {
	for _, caso := range []string{"begin", "nil_tx", "ajustes", "consulta", "denegacion", "commit", "cancelada_inicial", "cancelada_lectura", "cancelada_commit"} {
		t.Run(caso, func(t *testing.T) {
			o, resultado := consultaOrdenPrueba(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tx := &consultaTxPrueba{raw: consultaJSONPrueba(t, resultado)}
			pool := &consultaPoolPrueba{tx: tx}
			var esperado error = ports.ErrConsultaNoDisponible
			switch caso {
			case "begin":
				pool.err = errors.New("dato privado")
			case "nil_tx":
				pool.tx = (*consultaTxPrueba)(nil)
			case "ajustes":
				tx.errExec = errors.New("dato privado")
			case "consulta":
				tx.errConsulta = errors.New("dato privado")
			case "denegacion":
				tx.errConsulta, esperado = &pgconn.PgError{Code: "42501", Message: "dato privado"}, vec.ErrAutorizacionDenegada
			case "commit":
				tx.errCommit = errors.New("dato privado")
			case "cancelada_inicial":
				cancel()
				esperado = context.Canceled
			case "cancelada_lectura":
				tx.despuesConsulta, esperado = cancel, context.Canceled
			case "cancelada_commit":
				tx.despuesCommit, esperado = cancel, context.Canceled
			}
			repo, _ := nuevaConsulta(pool)
			out, err := repo.ConsultarActual(ctx, o)
			if !errors.Is(err, esperado) || !reflect.DeepEqual(out, ports.ResultadoConsultaPropia{}) {
				t.Fatal("fallo expuso datos o error privado", err)
			}
			if caso != "begin" && caso != "nil_tx" && caso != "cancelada_inicial" && tx.rollbacks != 1 {
				t.Fatal("falta cierre de transacción")
			}
			if caso != "commit" && caso != "cancelada_commit" && tx.commits != 0 {
				t.Fatal("COMMIT después del fallo")
			}
		})
	}
}

// Esta regresión acredita el rechazo y el cierre del adaptador con material
// ya emitido. El doble no demuestra auditoría durable ni consumo V3 real.
func TestConsultaPropiaRechazoSQLRevierteAntesDeDevolverDenegacion(t *testing.T) {
	orden, resultado := consultaOrdenPrueba(t)
	privado := "diagnostico_sql_que_no_debe_salir"
	tx := &consultaTxPrueba{
		raw:         consultaJSONPrueba(t, resultado),
		errConsulta: &pgconn.PgError{Code: "42501", Message: privado},
	}
	pool := &consultaPoolPrueba{tx: tx}
	repo, err := nuevaConsulta(pool)
	if err != nil {
		t.Fatal(err)
	}
	out, err := repo.ConsultarActual(context.Background(), orden)
	if !errors.Is(err, vec.ErrAutorizacionDenegada) ||
		!reflect.DeepEqual(out, ports.ResultadoConsultaPropia{}) ||
		tx.rollbacks != 1 || tx.commits != 0 || pool.llamadas != 1 {
		t.Fatal("el rechazo SQL no terminó vacío y revertido", err)
	}
	if tx.consulta != consultaHechoPropio || len(tx.argumentos) != 11 ||
		!bytes.Equal(tx.argumentos[1].([]byte), orden.Autorizacion.Material.CapacidadCanonica()) ||
		!bytes.Equal(tx.argumentos[2].([]byte), orden.Autorizacion.Material.DecisionCanonica()) {
		t.Fatal("el rechazo no sucedió después de enviar la autorización emitida")
	}
	if strings.Contains(err.Error(), privado) {
		t.Fatal("el rechazo expone el diagnóstico SQL")
	}
}

func TestConsultaPropiaRechazaDependenciasNulas(t *testing.T) {
	if _, err := NuevaConsulta(nil); !errors.Is(err, ports.ErrConsultaNoDisponible) {
		t.Fatal("pool nulo aceptado")
	}
	o, _ := consultaOrdenPrueba(t)
	for _, repo := range []*Consulta{nil, {}} {
		if out, err := repo.ConsultarActual(context.Background(), o); err == nil || out.HechoActual != nil {
			t.Fatal("repositorio nulo expone ficha")
		}
	}
	repo, _ := nuevaConsulta(&consultaPoolPrueba{})
	if _, err := repo.ConsultarActual(nil, o); err == nil {
		t.Fatal("contexto nulo aceptado")
	}
}
