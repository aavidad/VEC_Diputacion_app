package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type transaccionBorradorPrueba struct {
	pgx.Tx
	consultaErr, commitErr, rollbackErr error
	salida                              []byte
	commits, rollbacks                  int
	rollbackCancelado                   bool
}

func (t *transaccionBorradorPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	return filaBorradorPrueba{err: t.consultaErr, salida: t.salida}
}
func (t *transaccionBorradorPrueba) Commit(context.Context) error { t.commits++; return t.commitErr }
func (t *transaccionBorradorPrueba) Rollback(ctx context.Context) error {
	t.rollbacks++
	t.rollbackCancelado = ctx.Err() != nil
	return t.rollbackErr
}

type filaBorradorPrueba struct {
	err    error
	salida []byte
}

func (f filaBorradorPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	if f.salida != nil {
		*dest[0].(*[]byte) = append([]byte(nil), f.salida...)
	} else {
		*dest[0].(*[]byte) = []byte(`{}`)
	}
	return nil
}

type iniciadorBorradorPrueba struct {
	tx  *transaccionBorradorPrueba
	err error
}

func (p iniciadorBorradorPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return p.tx, p.err
}

// Se prueban resultados del transporte; estos dobles no acreditan V3 ni PG.
func TestResultadoBorradorDistingueRollbackDeCommitAmbiguo(t *testing.T) {
	for _, caso := range []struct {
		nombre, resultado, causa               string
		consulta, commit, rollback, validacion error
		incierto                               bool
	}{
		{"confirmado", "", "", nil, nil, nil, nil, false},
		{"conflicto", "fallo_confirmado", "conflicto", &pgconn.PgError{Code: "PDI01"}, nil, nil, nil, false},
		{"sql", "fallo_confirmado", "persistencia", errors.New("fallo SQL sintético"), nil, nil, nil, false},
		{"recibo", "fallo_confirmado", "recibo_invalido", nil, nil, nil, ports.ErrBorradorNoDisponible, false},
		{"rollback", "resultado_indeterminado", "rollback", errors.New("fallo"), nil, errors.New("sin respuesta"), nil, true},
		{"commit", "resultado_indeterminado", "commit", nil, errors.New("sin respuesta"), nil, nil, true},
		{"commit rollback", "fallo_confirmado", "commit_revertido", nil, pgx.ErrTxCommitRollback, pgx.ErrTxClosed, nil, false},
		{"commit rollback envuelto", "fallo_confirmado", "commit_revertido", nil, fmt.Errorf("transacción: %w", pgx.ErrTxCommitRollback), nil, nil, false},
		{"commit serializable", "fallo_confirmado", "commit_revertido", nil, &pgconn.PgError{Code: "40001"}, pgx.ErrTxClosed, nil, false},
		{"commit serializable envuelto", "fallo_confirmado", "commit_revertido", nil, fmt.Errorf("transacción: %w", &pgconn.PgError{Code: "40001"}), nil, nil, false},
		{"commit conexion cerrada", "resultado_indeterminado", "commit", nil, pgx.ErrTxClosed, pgx.ErrTxClosed, nil, true},
		{"commit cancelacion", "resultado_indeterminado", "commit", nil, context.Canceled, pgx.ErrTxClosed, nil, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &transaccionBorradorPrueba{consultaErr: caso.consulta, commitErr: caso.commit, rollbackErr: caso.rollback}
			r := &RepositorioBorradorComision{pool: iniciadorBorradorPrueba{tx: tx}}
			resultado, causa, err := r.transaccion(context.Background(), "SELECT prueba", AccionCrearBorradorPropio, []byte(`{}`), vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, func([]byte) error { return caso.validacion })
			if resultado != caso.resultado || causa != caso.causa || errors.Is(err, ports.ErrResultadoBorradorIncierto) != caso.incierto {
				t.Fatalf("clasificación incorrecta: %s/%s/%v", resultado, causa, err)
			}
			if (err != nil) != (caso.resultado != "") {
				t.Fatal("resultado y error incoherentes")
			}
			if err != nil && (tx.rollbacks != 1 || tx.rollbackCancelado) {
				t.Fatal("rollback omitido o cancelado")
			}
			if caso.consulta != nil || caso.validacion != nil {
				if tx.commits != 0 {
					t.Fatal("confirmó salida fallida")
				}
			}
		})
	}
}

func TestRollbackBorradorSobreviveCancelacionPeticion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tx := &transaccionBorradorPrueba{consultaErr: context.Canceled}
	r := &RepositorioBorradorComision{pool: iniciadorBorradorPrueba{tx: tx}}
	resultado, _, err := r.transaccion(ctx, "SELECT prueba", AccionCrearBorradorPropio, nil, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, func([]byte) error { return nil })
	if resultado != "fallo_confirmado" || !errors.Is(err, context.Canceled) || tx.rollbacks != 1 || tx.rollbackCancelado {
		t.Fatal("cancelación impidió cerrar transacción")
	}
}

// Comprueba el límite de transporte con dobles, no acredita persistencia PG.
func TestLimitesPorOperacionYResumenVeinteRutasMaximas(t *testing.T) {
	x := solicitudPrueba(t)
	x.Borrador.Ruta.Trazado = make([][2]float64, 2000)
	for i := range x.Borrador.Ruta.Trazado {
		x.Borrador.Ruta.Trazado[i] = [2]float64{37.123456789012, -3.123456789012}
	}
	material, _, e := materialCreacion(x)
	if e != nil || len(material) <= 64<<10 || len(material) > ports.MaxBytesMaterialBorrador {
		t.Fatal("material máximo razonable no admitido", e, len(material))
	}
	recibo := ports.ReciboBorradorComision{ComisionRef: "dietas:borrador:Operacion_AAAAAAA", ReciboRef: "recibo:prueba", CorrelacionRef: "corr:prueba", Version: 1, RegistradoEn: time.Now().UTC().Truncate(time.Microsecond)}
	detalle := struct {
		Borrador any `json:"borrador"`
		Recibo   any `json:"recibo"`
	}{x.Borrador, recibo}
	salida, e := json.MarshalIndent(detalle, "", " ")
	if e != nil || len(salida) <= 64<<10 || len(salida) > ports.MaxBytesDetalleBorrador {
		t.Fatal("frontera detalle no ejercitada", len(salida))
	}
	tx := &transaccionBorradorPrueba{salida: salida}
	r := &RepositorioBorradorComision{pool: iniciadorBorradorPrueba{tx: tx}}
	_, _, e = r.transaccion(context.Background(), "prueba", AccionRecuperarBorradorPropio, nil, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, func(b []byte) error { return decodificar(b, &detalle) })
	if e != nil || tx.commits != 1 {
		t.Fatal("detalle grande no recuperable", e)
	}
	pagina := ports.PaginaBorradoresPropios{Borradores: make([]ports.BorradorConRecibo, 20)}
	for i := range pagina.Borradores {
		pagina.Borradores[i] = ports.BorradorConRecibo{Borrador: x.Borrador.Resumen(), Recibo: recibo}
	}
	salida, _ = json.Marshal(pagina)
	if len(salida) > ports.MaxBytesListadoBorrador || strings.Contains(string(salida), `"trazado"`) || strings.Contains(string(salida), `"tramos"`) || !strings.Contains(string(salida), `"ruta_etiquetas"`) {
		t.Fatal("listado pesado o sin ruta")
	}
	for accion, limite := range map[string]int{AccionCrearBorradorPropio: ports.MaxBytesReciboBorrador, AccionRecuperarBorradorPropio: ports.MaxBytesDetalleBorrador, AccionListarBorradoresPropios: ports.MaxBytesListadoBorrador} {
		for _, exceso := range []int{0, 1} {
			tx := &transaccionBorradorPrueba{salida: []byte("{}" + strings.Repeat(" ", limite-2+exceso))}
			r := &RepositorioBorradorComision{pool: iniciadorBorradorPrueba{tx: tx}}
			_, causa, e := r.transaccion(context.Background(), "prueba", accion, nil, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, func(b []byte) error { var v map[string]any; return decodificar(b, &v) })
			if exceso == 0 && (e != nil || tx.commits != 1) {
				t.Fatal("límite exacto rechazado", accion)
			}
			if exceso == 1 && (e == nil || causa != "recibo_invalido" || tx.commits != 0 || tx.rollbacks != 1) {
				t.Fatal("exceso confirmado", accion)
			}
		}
	}
}

type proveedorFalloMaterial struct {
	base EnlaceAutorizacionBorrador
	err  error
}

func (p proveedorFalloMaterial) AutorizarBorradorPropio(context.Context, core.ContextoActor, string, core.RecursoAutorizable) (AutorizacionBorrador, error) {
	return AutorizacionBorrador{ResultadoBase: p.base}, p.err
}
func TestMaterialInvalidoNoIniciaTransaccionNiFalseaDenegacion(t *testing.T) {
	x := solicitudPrueba(t)
	pool := &poolObservado{}
	base := EnlaceAutorizacionBorrador{DecisionRef: "dec_sintetica", ContextoRef: "rca_sintetico", DecisionHuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: "correlacion_sintetica"}
	r := &RepositorioBorradorComision{pool: pool, proveedor: proveedorFalloMaterial{base: base}}
	_, e := r.CrearBorradorPropio(context.Background(), x)
	if !errors.Is(e, ports.ErrBorradorNoDisponible) || errors.Is(e, ports.ErrAccesoBorradorDenegado) || pool.llamadas != 0 {
		t.Fatal("material inválido consumido o fallo etiquetado como decisión PDP")
	}
}
