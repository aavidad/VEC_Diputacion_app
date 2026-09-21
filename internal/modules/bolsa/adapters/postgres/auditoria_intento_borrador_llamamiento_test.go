package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type txAuditoriaIntentoBorradorPrueba struct {
	pgx.Tx
	execuciones    []ejecucionAuditoriaIntentoBorradorPrueba
	confirmaciones int
	reversiones    int
}

type ejecucionAuditoriaIntentoBorradorPrueba struct {
	sql        string
	argumentos []any
}

func (t *txAuditoriaIntentoBorradorPrueba) Exec(_ context.Context, sql string, argumentos ...any) (pgconn.CommandTag, error) {
	t.execuciones = append(t.execuciones, ejecucionAuditoriaIntentoBorradorPrueba{sql, argumentos})
	return pgconn.NewCommandTag("SELECT 1"), nil
}
func (t *txAuditoriaIntentoBorradorPrueba) Commit(context.Context) error {
	t.confirmaciones++
	return nil
}
func (t *txAuditoriaIntentoBorradorPrueba) Rollback(context.Context) error {
	t.reversiones++
	return nil
}

type generadorCorrelacionIntentoBorradorPrueba struct{}

func (generadorCorrelacionIntentoBorradorPrueba) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return "correlacion_" + strings.Repeat("a", 32), nil
}

func TestAuditoriaIntentoBorradorPostgreSQLUsaTransaccionNuevaSerializableYUTC(t *testing.T) {
	tx := &txAuditoriaIntentoBorradorPrueba{}
	iniciador := &iniciadorLlamamientoPostgreSQLPrueba{tx: tx}
	auditoria, err := nuevaAuditoriaIntentoBorradorLlamamientoPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), generadorCorrelacionIntentoBorradorPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	intento := puertosbolsa.IntentoBorradorLlamamiento{Correlacion: correlacion, Accion: puertosbolsa.AccionIntentoCrearBorradorLlamamiento, ClaseRuta: puertosbolsa.ClaseRutaColeccionBorradorLlamamiento, ActorVerificado: "per_" + strings.Repeat("a", 22), Resultado: puertosbolsa.ResultadoIntentoAccesoDenegadoBorradorLlamamiento}
	if err = auditoria.RegistrarIntentoBorradorLlamamiento(context.Background(), intento); err != nil {
		t.Fatal(err)
	}
	if iniciador.inicios != 1 || iniciador.opciones.IsoLevel != pgx.Serializable || iniciador.opciones.AccessMode != pgx.ReadWrite || tx.confirmaciones != 1 {
		t.Fatalf("transaccion incorrecta: inicios=%d opciones=%#v commits=%d", iniciador.inicios, iniciador.opciones, tx.confirmaciones)
	}
	if len(tx.execuciones) != 2 || !containsBorradorLlamamiento(tx.execuciones[0].sql, "timezone") || !containsBorradorLlamamiento(tx.execuciones[1].sql, funcionRegistrarIntentoBorradorLlamamientoPostgreSQL) {
		t.Fatalf("frontera SQL incorrecta: %#v", tx.execuciones)
	}
	if len(tx.execuciones[1].argumentos) != 5 || tx.execuciones[1].argumentos[1] != "crear" || tx.execuciones[1].argumentos[2] != "coleccion" || tx.execuciones[1].argumentos[4] != "acceso_denegado" {
		t.Fatalf("parametros abiertos o incorrectos: %#v", tx.execuciones[1].argumentos)
	}
}

func TestAuditoriaIntentoBorradorPostgreSQLFallaCerradoSinEntradaValidaNiPool(t *testing.T) {
	if auditoria, err := nuevaAuditoriaIntentoBorradorLlamamientoPostgreSQL(nil); auditoria != nil || !errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) {
		t.Fatalf("pool nulo aceptado: %#v %v", auditoria, err)
	}
	auditoria := &AuditoriaIntentoBorradorLlamamientoPostgreSQL{pool: &iniciadorLlamamientoPostgreSQLPrueba{}}
	if err := auditoria.RegistrarIntentoBorradorLlamamiento(context.Background(), puertosbolsa.IntentoBorradorLlamamiento{}); !errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) {
		t.Fatalf("entrada invalida alcanzo SQL: %v", err)
	}
}
