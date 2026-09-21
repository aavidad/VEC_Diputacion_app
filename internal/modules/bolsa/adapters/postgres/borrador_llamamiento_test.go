package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type txBorradorLlamamientoPrueba struct {
	pgx.Tx
	set string
}

func (t *txBorradorLlamamientoPrueba) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	t.set = sql
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (t *txBorradorLlamamientoPrueba) Rollback(context.Context) error { return nil }

func TestNuevoRepositorioBorradorLlamamientoPostgreSQLFallaCerradoSinPool(t *testing.T) {
	repo, err := nuevoRepositorioBorradorLlamamientoPostgreSQL(nil)
	if repo != nil || !errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) {
		t.Fatalf("pool nulo aceptado: %#v %v", repo, err)
	}
}

func TestInicioBorradorLlamamientoPostgreSQLFijaTransaccionYEntorno(t *testing.T) {
	tx := &txBorradorLlamamientoPrueba{}
	iniciador := &iniciadorLlamamientoPostgreSQLPrueba{tx: tx}
	repo, err := nuevoRepositorioBorradorLlamamientoPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	obtenida, err := repo.iniciar(context.Background())
	if err != nil || obtenida != tx {
		t.Fatalf("inicio no devolvio tx: %v", err)
	}
	if iniciador.opciones.IsoLevel != pgx.Serializable || iniciador.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("opciones inseguras: %#v", iniciador.opciones)
	}
	for _, ajuste := range []string{"search_path", "row_security", "timezone", "lock_timeout", "statement_timeout", "idle_in_transaction_session_timeout"} {
		if !containsBorradorLlamamiento(tx.set, ajuste) {
			t.Fatalf("falta SET %s: %s", ajuste, tx.set)
		}
	}
}

func TestErroresBorradorLlamamientoPostgreSQLNoFiltranNiConfundenClave(t *testing.T) {
	if err := errorCrearBorradorLlamamientoPostgreSQL(context.Background(), &pgconn.PgError{Code: "VBL01"}); !errors.Is(err, puertosbolsa.ErrClaveBorradorLlamamientoReutilizada) {
		t.Fatalf("colision semantica no preservada: %v", err)
	}
	if err := errorCrearBorradorLlamamientoPostgreSQL(context.Background(), &pgconn.PgError{Code: "23505"}); !errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) {
		t.Fatalf("unique tecnico expuesto como conflicto semantico: %v", err)
	}
	if err := errorCrearBorradorLlamamientoPostgreSQL(context.Background(), &pgconn.PgError{Code: "42501"}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("creacion revocada no denegada: %v", err)
	}
	if err := errorConsultarBorradorLlamamientoPostgreSQL(context.Background(), &pgconn.PgError{Code: "42501"}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		t.Fatalf("lectura ajena no denegada: %v", err)
	}
}

func containsBorradorLlamamiento(s, parte string) bool {
	for i := 0; i+len(parte) <= len(s); i++ {
		if s[i:i+len(parte)] == parte {
			return true
		}
	}
	return false
}

func TestRepositorioBorradorLlamamientoPostgreSQLNoAbreTransaccionConComandoInvalido(t *testing.T) {
	tx := &transaccionLlamamientoPostgreSQLPrueba{}
	iniciador := &iniciadorLlamamientoPostgreSQLPrueba{tx: tx}
	repo, err := nuevoRepositorioBorradorLlamamientoPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CrearBorradorLlamamiento(context.Background(), puertosbolsa.ComandoCrearBorradorLlamamiento{})
	if !errors.Is(err, puertosbolsa.ErrFuenteBorradorLlamamientoNoDisponible) || iniciador.inicios != 0 || tx.confirmaciones != 0 {
		t.Fatalf("comando invalido alcanzo PostgreSQL: err=%v inicios=%d commits=%d", err, iniciador.inicios, tx.confirmaciones)
	}
}

func TestCanonCrearBorradorLlamamientoPostgreSQLConservaOrdenYUTF8(t *testing.T) {
	// Este vector protege la misma representación que PostgreSQL recompone
	// antes de calcular la huella: no se permite una variante HTML de Go.
	obtenido, err := dominiobolsa.RepresentacionCanonicaComandoCrearBorradorLlamamiento(
		"per_0123456789abcdefghijkl", "unidad:rrhh", "ambito:rrhh", "clave-prueba-001",
		dominiobolsa.ContenidoBorradorLlamamiento{Resumen: "Ámbito <>&"},
	)
	esperado := `{"esquema":"vec.bolsa.llamamiento.borrador-interno.crear.v1","propietario_ref":"per_0123456789abcdefghijkl","unidad_ref":"unidad:rrhh","ambito_ref":"ambito:rrhh","clave_idempotencia":"clave-prueba-001","contenido":{"resumen":"Ámbito <>&"}}`
	if err != nil || string(obtenido) != esperado {
		t.Fatalf("canon distinto:\n%s\n%s\n%v", obtenido, esperado, err)
	}
}

func TestReferenciaReciboBorradorLlamamientoValidaAlfabetoTraducido(t *testing.T) {
	if !referenciaReciboBorradorLlamamientoValida("recibo:" + "abcdefghijklmnopabcdefghijklmnopabcdefghijklmnopabcdefghijklmnop") {
		t.Fatal("recibo SQL valido rechazado")
	}
	if referenciaReciboBorradorLlamamientoValida("recibo:" + "qbcdefghijklmnopabcdefghijklmnopabcdefghijklmnopabcdefghijklmnop") {
		t.Fatal("recibo con caracter fuera de translate aceptado")
	}
}

func TestJSONExactoBorradorLlamamientoPostgreSQLAceptaResumenValidoEscapadoSobre4096(t *testing.T) {
	contenido := struct {
		Resumen string `json:"resumen"`
	}{Resumen: strings.Repeat("<", 1000)}
	serializado, err := json.Marshal(contenido)
	if err != nil {
		t.Fatal(err)
	}
	if len(serializado) <= 4096 || len(serializado) > 16384 {
		t.Fatalf("vector de regresion fuera de rango: %d", len(serializado))
	}
	var obtenido struct {
		Resumen string `json:"resumen"`
	}
	if !jsonExactoBorradorLlamamientoPostgreSQL(serializado, &obtenido) {
		t.Fatalf("resumen valido rechazado: bytes=%d", len(serializado))
	}
	if obtenido.Resumen != contenido.Resumen {
		t.Fatalf("resumen alterado tras la lectura: %q", obtenido.Resumen)
	}
}
