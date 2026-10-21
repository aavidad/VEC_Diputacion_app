package postgres

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/shared/postgresql"
	dochttp "vec-diputacion-granada/internal/vec/documentos/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

type filaCarreraDocumentos struct{ err error }

func (f filaCarreraDocumentos) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	switch v := destinos[0].(type) {
	case *[]byte:
		*v = []byte(`{"items":[]}`)
	case *string:
		*v = "denegacion:documentos:uno"
	}
	return nil
}

type transaccionCarreraDocumentos struct {
	transaccionDocumentosPrueba
	etapa string
	err   error
}

func (tx *transaccionCarreraDocumentos) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx.etapa == "ajustes" {
		return pgconn.CommandTag{}, tx.err
	}
	return tx.transaccionDocumentosPrueba.Exec(ctx, sql, args...)
}

func (tx *transaccionCarreraDocumentos) QueryRow(ctx context.Context, sql string, args ...any) filaTransaccion {
	tx.transaccionDocumentosPrueba.QueryRow(ctx, sql, args...)
	if tx.etapa == "consulta" {
		return filaCarreraDocumentos{tx.err}
	}
	return filaCarreraDocumentos{}
}

func (tx *transaccionCarreraDocumentos) Commit(ctx context.Context) error {
	if tx.etapa == "commit" {
		return tx.err
	}
	return tx.transaccionDocumentosPrueba.Commit(ctx)
}

func TestConsultaDocumentosRepiteTransaccionCompleta(t *testing.T) {
	for _, etapa := range []string{"inicio", "ajustes", "consulta", "commit"} {
		t.Run(etapa, func(t *testing.T) {
			var intentos []*transaccionCarreraDocumentos
			args := []any{[]byte("preimagen"), "autoridad", "correlacion"}
			inicios := 0
			raw, err := consultarConReintento(t.Context(), func() (transaccionRepositorio, error) {
				inicios++
				for _, anterior := range intentos {
					if anterior.llamadas[len(anterior.llamadas)-1] != "rollback" {
						t.Fatal("se abrió otro intento antes de cerrar el anterior")
					}
				}
				tx := &transaccionCarreraDocumentos{}
				if inicios == 1 {
					tx.etapa, tx.err = etapa, &pgconn.PgError{Code: "40001", Message: "detalle_privado"}
					if etapa == "inicio" {
						return nil, tx.err
					}
				}
				intentos = append(intentos, tx)
				return tx, nil
			}, "SELECT listado($1,$2,$3)", args...)
			if err != nil || string(raw) != `{"items":[]}` || inicios != 2 {
				t.Fatalf("resultado=%s error=%v intentos=%d", raw, err, inicios)
			}
			for _, tx := range intentos {
				if tx.etapa != "ajustes" && (tx.consulta != "SELECT listado($1,$2,$3)" || !reflect.DeepEqual(tx.argumentos, args)) {
					t.Fatal("el reintento cambió la consulta o su autoridad")
				}
			}
		})
	}
}

func TestConsultaDocumentosAgotamientoYErroresDefinitivos(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		causa    error
		want     error
		intentos int
	}{
		{"40001 persistente", &pgconn.PgError{Code: "40001", Message: "detalle_privado"}, ErrRepositorioNoDisponible, postgresql.IntentosMaximosCarreraSerializable},
		{"denegación", &pgconn.PgError{Code: "42501"}, ports.ErrAccesoDenegado, 1},
		{"bloqueo", &pgconn.PgError{Code: "55P03"}, ErrRepositorioNoDisponible, 1},
		{"commit ambiguo", io.ErrUnexpectedEOF, ErrRepositorioNoDisponible, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			inicios := 0
			raw, err := consultarConReintento(t.Context(), func() (transaccionRepositorio, error) {
				inicios++
				return &transaccionCarreraDocumentos{etapa: "commit", err: caso.causa}, nil
			}, "SELECT listado()")
			var pg *pgconn.PgError
			if raw != nil || !errors.Is(err, caso.want) || inicios != caso.intentos || errors.As(err, &pg) {
				t.Fatalf("resultado=%s error=%v intentos=%d", raw, err, inicios)
			}
			if errors.Is(err, ports.ErrCapacidadNoDisponible) {
				comprobarConsultaHTTP503(t, err)
			}
		})
	}
}

type consultaDocumentosError struct{ err error }

func (s consultaDocumentosError) ListarExpediente(context.Context, ports.ConsultaExpediente) (ports.PaginaDocumentos, error) {
	return ports.PaginaDocumentos{}, s.err
}
func (consultaDocumentosError) DescargarOriginal(context.Context, ports.ConsultaDocumento) (ports.Original, error) {
	return ports.Original{}, ports.ErrCapacidadNoDisponible
}
func (consultaDocumentosError) ResolverConsultaExpediente(context.Context, ports.ConsultaExpediente) (ports.AutorizacionV3, error) {
	return ports.AutorizacionV3{}, nil
}
func (consultaDocumentosError) ResolverDescargaOriginal(context.Context, ports.ConsultaDocumento, string) (ports.AutorizacionV3, error) {
	return ports.AutorizacionV3{}, ports.ErrCapacidadNoDisponible
}

func comprobarConsultaHTTP503(t *testing.T, err error) {
	t.Helper()
	s := consultaDocumentosError{err}
	rutas, fallo := dochttp.NuevasRutasExactas(s, s)
	if fallo != nil {
		t.Fatal(fallo)
	}
	r := httptest.NewRequest(http.MethodPost, dochttp.RutaConsultaExpediente,
		strings.NewReader(`{"expediente_ref":"ref:`+strings.Repeat("a", 64)+`","limite":10}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable || w.Body.String() != `{"error":{"clave_i18n":"api.documentos.error.servicio_no_disponible","codigo":"servicio_no_disponible"}}` {
		t.Fatalf("respuesta inesperada: %d %s", w.Code, w.Body.String())
	}
}

type consultaFronteraCarrera struct {
	llamadas   int
	fallos     int
	argumentos []any
	t          *testing.T
}

func (c *consultaFronteraCarrera) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	c.llamadas++
	if c.argumentos != nil && !reflect.DeepEqual(c.argumentos, args) {
		c.t.Fatal("cambió la correlación o la orden de denegación")
	}
	c.argumentos = append([]any(nil), args...)
	if c.fallos < 0 || c.llamadas <= c.fallos {
		return filaCarreraDocumentos{&pgconn.PgError{Code: "40001", Message: "detalle_privado"}}
	}
	return filaCarreraDocumentos{}
}

func TestDenegacionFronteraDocumentosRepiteSoloCarreras(t *testing.T) {
	for _, fallos := range []int{1, -1} {
		c := &consultaFronteraCarrera{fallos: fallos, t: t}
		r := &RegistradorFrontera{db: c}
		err := r.RegistrarDenegacion(t.Context(), OrdenDenegacionFrontera{
			CorrelacionRef: "corr_0123456789abcdef0123456789abcdef", Motivo: MotivoFronteraDenegado,
			Ruta: RutaFronteraConsulta, Metodo: "POST"})
		if fallos == 1 && (err != nil || c.llamadas != 2) {
			t.Fatalf("no recuperó la denegación: %v intentos=%d", err, c.llamadas)
		}
		if fallos < 0 && (err != ErrFronteraNoDisponible || c.llamadas != postgresql.IntentosMaximosCarreraSerializable) {
			t.Fatalf("agotamiento sin centinela: %v intentos=%d", err, c.llamadas)
		}
	}
}
