package postgres

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgconn/ctxwatch"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type trazadorAcreditacionO405Hostil struct{}

type observadorContextoAcreditacionO405Hostil struct{}

func (observadorContextoAcreditacionO405Hostil) HandleCancel(context.Context) {}
func (observadorContextoAcreditacionO405Hostil) HandleUnwatchAfterCancel()    {}

type transaccionFinalizacionConcurrenteO405Prueba struct {
	pgx.Tx
	commitIniciado chan struct{}
	continuar      chan struct{}
	commits        int
	rollbacks      int
}

func (t *transaccionFinalizacionConcurrenteO405Prueba) Commit(
	context.Context,
) error {
	t.commits++
	close(t.commitIniciado)
	<-t.continuar
	return nil
}

func (t *transaccionFinalizacionConcurrenteO405Prueba) Rollback(
	context.Context,
) error {
	t.rollbacks++
	return nil
}

func (trazadorAcreditacionO405Hostil) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	_ pgx.TraceQueryStartData,
) context.Context {
	return ctx
}

func (trazadorAcreditacionO405Hostil) TraceQueryEnd(
	context.Context,
	*pgx.Conn,
	pgx.TraceQueryEndData,
) {
}

func nuevaConexionAcreditacionO405Valida(
	tx pgx.Tx,
) *conexionAcreditacionPoolO405Prueba {
	return &conexionAcreditacionPoolO405Prueba{
		fila: filaAcreditacionPoolO405Prueba{
			valores: valoresAcreditacionPoolO405Prueba(true),
		},
		tx: tx,
	}
}

func TestAcreditacionPoolO405RechazaHooksDelPoolAntesDeAdquirir(
	t *testing.T,
) {
	casos := []struct {
		nombre     string
		cambiar    func(*pgxpool.Config)
		rompeSello bool
	}{
		{"BeforeConnect", func(c *pgxpool.Config) {
			c.BeforeConnect = func(context.Context, *pgx.ConnConfig) error {
				return nil
			}
		}, false},
		{"AfterConnect", func(c *pgxpool.Config) {
			c.AfterConnect = func(context.Context, *pgx.Conn) error {
				return nil
			}
		}, false},
		{"BeforeAcquire", func(c *pgxpool.Config) {
			c.BeforeAcquire = func(context.Context, *pgx.Conn) bool {
				return true
			}
		}, false},
		{"PrepareConn", func(c *pgxpool.Config) {
			c.PrepareConn = func(context.Context, *pgx.Conn) (bool, error) {
				return true, nil
			}
		}, false},
		{"AfterRelease", func(c *pgxpool.Config) {
			c.AfterRelease = func(*pgx.Conn) bool { return true }
		}, false},
		{"BeforeClose", func(c *pgxpool.Config) {
			c.BeforeClose = func(*pgx.Conn) {}
		}, false},
		{"ShouldPing", func(c *pgxpool.Config) {
			c.ShouldPing = func(
				context.Context,
				pgxpool.ShouldPingParams,
			) bool {
				return false
			}
		}, false},
		{"OnNotification original", func(c *pgxpool.Config) {
			c.ConnConfig.OnNotification = func(
				*pgconn.PgConn,
				*pgconn.Notification,
			) {
			}
		}, false},
		{"OnPgError original custom", func(c *pgxpool.Config) {
			c.ConnConfig.OnPgError = func(
				*pgconn.PgConn,
				*pgconn.PgError,
			) bool {
				return true
			}
		}, false},
		{"DialFunc original ausente", func(c *pgxpool.Config) {
			c.ConnConfig.DialFunc = nil
		}, false},
		{"LookupFunc original ausente", func(c *pgxpool.Config) {
			c.ConnConfig.LookupFunc = nil
		}, false},
		{"BuildFrontend original ausente", func(c *pgxpool.Config) {
			c.ConnConfig.BuildFrontend = nil
		}, false},
		{"BuildContextWatcherHandler original ausente", func(c *pgxpool.Config) {
			c.ConnConfig.BuildContextWatcherHandler = nil
		}, false},
		{"DialFunc original custom", func(c *pgxpool.Config) {
			c.ConnConfig.DialFunc = func(
				context.Context,
				string,
				string,
			) (net.Conn, error) {
				return nil, errors.New("dial observado")
			}
		}, true},
		{"LookupFunc original custom", func(c *pgxpool.Config) {
			c.ConnConfig.LookupFunc = func(
				context.Context,
				string,
			) ([]string, error) {
				return []string{"203.0.113.1"}, nil
			}
		}, true},
		{"BuildFrontend original observa protocolo", func(c *pgxpool.Config) {
			c.ConnConfig.BuildFrontend = func(
				lector io.Reader,
				escritor io.Writer,
			) *pgproto3.Frontend {
				return pgproto3.NewFrontend(lector, escritor)
			}
		}, true},
		{"BuildContextWatcherHandler original custom", func(c *pgxpool.Config) {
			c.ConnConfig.BuildContextWatcherHandler = func(
				*pgconn.PgConn,
			) ctxwatch.Handler {
				return observadorContextoAcreditacionO405Hostil{}
			}
		}, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			configuracion := configuracionPoolAcreditacionO405Prueba()
			caso.cambiar(configuracion)
			origen := &origenAcreditacionPoolO405Prueba{
				configuracion: configuracion,
				conexion:      nuevaConexionAcreditacionO405Valida(nil),
				sinSello:      caso.rompeSello,
			}
			err := acreditarPoolRecuperacionCoberturaO405(
				context.Background(),
				origen,
				modoTLSAcreditacionPoolO405Produccion,
			)
			if !errors.Is(
				err,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
			) || origen.adquisiciones != 0 {
				t.Fatalf(
					"hook aceptado/adquirido: adquirir=%d err=%v",
					origen.adquisiciones,
					err,
				)
			}
		})
	}
}

func TestAcreditacionPoolO405ToleraSoloOnPgErrorSeguroDePgconn(
	t *testing.T,
) {
	predeterminada, err := pgconn.ParseConfig(
		"host=localhost user=vec_o405 dbname=postgres " +
			"password='' sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}
	configuracion := configuracionPoolAcreditacionO405Prueba()
	configuracion.ConnConfig.OnPgError = predeterminada.OnPgError
	if !configuracionPoolAcreditacionO405Valida(
		configuracion,
		modoTLSAcreditacionPoolO405Produccion,
	) {
		t.Fatal("handler fatal seguro predeterminado rechazado")
	}
}

func TestAcreditacionPoolO405RechazaMutadoresDeConexionEfectiva(
	t *testing.T,
) {
	casos := []struct {
		nombre     string
		cambiar    func(*pgx.ConnConfig)
		rompeSello bool
	}{
		{"AfterNetConnect", func(c *pgx.ConnConfig) {
			c.AfterNetConnect = func(
				context.Context,
				*pgconn.Config,
				net.Conn,
			) (net.Conn, error) {
				return nil, nil
			}
		}, false},
		{"ValidateConnect", func(c *pgx.ConnConfig) {
			c.ValidateConnect = func(context.Context, *pgconn.PgConn) error {
				return nil
			}
		}, false},
		{"AfterConnect pgconn", func(c *pgx.ConnConfig) {
			c.Config.AfterConnect = func(
				context.Context,
				*pgconn.PgConn,
			) error {
				return nil
			}
		}, false},
		{"trazador", func(c *pgx.ConnConfig) {
			c.Tracer = trazadorAcreditacionO405Hostil{}
		}, false},
		{"parámetros de sesión", func(c *pgx.ConnConfig) {
			c.RuntimeParams = map[string]string{
				"search_path": "public",
			}
		}, false},
		{"OnPgError efectivo custom", func(c *pgx.ConnConfig) {
			c.OnPgError = func(
				*pgconn.PgConn,
				*pgconn.PgError,
			) bool {
				return true
			}
		}, false},
		{"DialFunc efectivo ausente", func(c *pgx.ConnConfig) {
			c.DialFunc = nil
		}, false},
		{"LookupFunc efectivo ausente", func(c *pgx.ConnConfig) {
			c.LookupFunc = nil
		}, false},
		{"BuildFrontend efectivo ausente", func(c *pgx.ConnConfig) {
			c.BuildFrontend = nil
		}, false},
		{"BuildContextWatcherHandler efectivo ausente", func(c *pgx.ConnConfig) {
			c.BuildContextWatcherHandler = nil
		}, false},
		{"DialFunc efectivo custom", func(c *pgx.ConnConfig) {
			c.DialFunc = func(
				context.Context,
				string,
				string,
			) (net.Conn, error) {
				return nil, errors.New("dial observado")
			}
		}, true},
		{"LookupFunc efectivo custom", func(c *pgx.ConnConfig) {
			c.LookupFunc = func(
				context.Context,
				string,
			) ([]string, error) {
				return []string{"203.0.113.1"}, nil
			}
		}, true},
		{"BuildFrontend efectivo muta protocolo", func(c *pgx.ConnConfig) {
			c.BuildFrontend = func(
				lector io.Reader,
				escritor io.Writer,
			) *pgproto3.Frontend {
				return pgproto3.NewFrontend(lector, escritor)
			}
		}, true},
		{"BuildContextWatcherHandler efectivo custom", func(c *pgx.ConnConfig) {
			c.BuildContextWatcherHandler = func(
				*pgconn.PgConn,
			) ctxwatch.Handler {
				return observadorContextoAcreditacionO405Hostil{}
			}
		}, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			conexion := nuevaConexionAcreditacionO405Valida(nil)
			conexion.configuracion =
				configuracionConexionAcreditacionPoolO405Prueba()
			caso.cambiar(conexion.configuracion)
			conexion.sinSello = caso.rompeSello
			err := acreditarConexionRecuperacionCoberturaO405(
				context.Background(),
				conexion,
				modoTLSAcreditacionPoolO405Produccion,
			)
			if !errors.Is(
				err,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
			) || conexion.consulta != "" {
				t.Fatalf("mutador efectivo aceptado/consultado: %v", err)
			}
		})
	}
}

func TestIniciadorO405AcreditaYComienzaEnLaMismaConexion(
	t *testing.T,
) {
	txBase := &transaccionPreparacionPrueba{}
	conexion := nuevaConexionAcreditacionO405Valida(txBase)
	origen := &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
		conexion:      conexion,
	}
	iniciador := &iniciadorTransaccionesAcreditadoO405{
		origen: origen, modo: modoTLSAcreditacionPoolO405Produccion,
		oidFuncion: 405,
	}
	tx, err := iniciador.BeginTx(context.Background(), pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		t.Fatal(err)
	}
	if origen.adquisiciones != 1 || conexion.inicios != 1 ||
		conexion.acreditadaAntesDeBegin || conexion.liberaciones != 0 {
		t.Fatalf(
			"ciclo previo incorrecto: adquirir=%d acreditar=%t begin=%d liberar=%d",
			origen.adquisiciones,
			conexion.acreditadaAntesDeBegin,
			conexion.inicios,
			conexion.liberaciones,
		)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(context.Background()); !errors.Is(
		err,
		pgx.ErrTxClosed,
	) {
		t.Fatalf("segunda finalización=%v; quiere ErrTxClosed", err)
	}
	if txBase.confirmaciones != 1 || txBase.reversiones != 0 ||
		conexion.liberaciones != 1 {
		t.Fatalf(
			"finalización incorrecta: commit=%d rollback=%d liberar=%d",
			txBase.confirmaciones,
			txBase.reversiones,
			conexion.liberaciones,
		)
	}
}

func TestEjecutorO405NoConfiaEnConexionDeReadinessParaLaOperacion(
	t *testing.T,
) {
	readiness := nuevaConexionAcreditacionO405Valida(nil)
	operacion := nuevaConexionAcreditacionO405Valida(
		&transaccionPreparacionPrueba{},
	)
	operacion.configuracion = &pgx.ConnConfig{
		Config: pgconn.Config{Host: "db-o405.example"},
	}
	origen := &origenAcreditacionPoolO405Prueba{
		configuracion: configuracionPoolAcreditacionO405Prueba(),
		conexiones: []conexionAcreditacionPoolO405{
			readiness,
			operacion,
		},
	}
	ejecutor, err := nuevoEjecutorRecuperacionCoberturaO405Acreditado(
		context.Background(),
		origen,
		modoTLSAcreditacionPoolO405Produccion,
	)
	if err != nil {
		t.Fatalf("readiness válido rechazado: %v", err)
	}
	callbackInvocado := false
	err = ejecutor.EjecutarLecturaResultadoHistoricoTCB(
		context.Background(),
		func(
			cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
		) error {
			callbackInvocado = true
			return nil
		},
	)
	if !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) || callbackInvocado || readiness.liberaciones != 1 ||
		operacion.liberaciones != 1 || operacion.inicios != 0 ||
		operacion.consulta != "" {
		t.Fatalf(
			"segunda conexión no cerró: err=%v callback=%t readiness=%d "+
				"operacion_liberar=%d begin=%d consulta=%q",
			err,
			callbackInvocado,
			readiness.liberaciones,
			operacion.liberaciones,
			operacion.inicios,
			operacion.consulta,
		)
	}
}

func TestIniciadorO405LiberaUnaVezAnteFalloOPanico(
	t *testing.T,
) {
	casos := []struct {
		nombre   string
		conexion *conexionAcreditacionPoolO405Prueba
	}{
		{
			"transacción nula sin error",
			nuevaConexionAcreditacionO405Valida(nil),
		},
		{
			"error de catálogo",
			func() *conexionAcreditacionPoolO405Prueba {
				c := nuevaConexionAcreditacionO405Valida(nil)
				c.fila = filaAcreditacionPoolO405Prueba{
					err: errors.New("detalle privado catálogo"),
				}
				return c
			}(),
		},
		{
			"error begin",
			func() *conexionAcreditacionPoolO405Prueba {
				c := nuevaConexionAcreditacionO405Valida(nil)
				c.errBegin = errors.New("detalle privado begin")
				return c
			}(),
		},
		{
			"pánico begin",
			func() *conexionAcreditacionPoolO405Prueba {
				c := nuevaConexionAcreditacionO405Valida(nil)
				c.panicoBegin = true
				return c
			}(),
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			iniciador := &iniciadorTransaccionesAcreditadoO405{
				origen: &origenAcreditacionPoolO405Prueba{
					configuracion: configuracionPoolAcreditacionO405Prueba(),
					conexion:      caso.conexion,
				},
				modo:       modoTLSAcreditacionPoolO405Produccion,
				oidFuncion: 405,
			}
			tx, err := iniciador.BeginTx(
				context.Background(),
				pgx.TxOptions{},
			)
			if tx != nil || !errors.Is(
				err,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
			) || caso.conexion.liberaciones != 1 {
				t.Fatalf(
					"fallo no saneado/liberado: tx=%v liberar=%d err=%v",
					tx,
					caso.conexion.liberaciones,
					err,
				)
			}
		})
	}
}

func TestIniciadorO405LiberaConexionDevueltaJuntoConError(
	t *testing.T,
) {
	for _, panicoLiberar := range []bool{false, true} {
		nombre := map[bool]string{
			false: "liberación normal",
			true:  "pánico al liberar",
		}[panicoLiberar]
		t.Run(nombre, func(t *testing.T) {
			conexion := nuevaConexionAcreditacionO405Valida(
				&transaccionPreparacionPrueba{},
			)
			conexion.panicoLiberar = panicoLiberar
			iniciador := &iniciadorTransaccionesAcreditadoO405{
				origen: &origenAcreditacionPoolO405Prueba{
					configuracion: configuracionPoolAcreditacionO405Prueba(),
					conexion:      conexion,
					err:           errors.New("detalle privado acquire"),
				},
				modo:       modoTLSAcreditacionPoolO405Produccion,
				oidFuncion: 405,
			}
			tx, err := iniciador.BeginTx(
				context.Background(),
				pgx.TxOptions{},
			)
			if tx != nil || !errors.Is(
				err,
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
			) || strings.Contains(err.Error(), "detalle") ||
				conexion.liberaciones != 1 || conexion.consulta != "" ||
				conexion.inicios != 0 {
				t.Fatalf(
					"conn+error fugó/avanzó: tx=%v liberar=%d consulta=%q "+
						"begin=%d err=%v",
					tx,
					conexion.liberaciones,
					conexion.consulta,
					conexion.inicios,
					err,
				)
			}
		})
	}
}

func TestTransaccionAcreditadaO405SaneaPanicoAlLiberarUnaVez(
	t *testing.T,
) {
	conexion := nuevaConexionAcreditacionO405Valida(nil)
	conexion.panicoLiberar = true
	tx := &transaccionAcreditadaO405{
		Tx:       &transaccionPreparacionPrueba{},
		conexion: conexion,
	}
	if err := tx.Commit(context.Background()); !errors.Is(
		err,
		cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	) {
		t.Fatalf("pánico de liberación no saneado: %v", err)
	}
	if err := tx.Rollback(context.Background()); !errors.Is(
		err,
		pgx.ErrTxClosed,
	) {
		t.Fatalf("segunda finalización no sellada: %v", err)
	}
	if conexion.liberaciones != 1 {
		t.Fatalf("liberaciones=%d; quiere 1", conexion.liberaciones)
	}
}

func TestTransaccionAcreditadaO405SellaFinalizacionConcurrente(
	t *testing.T,
) {
	base := &transaccionFinalizacionConcurrenteO405Prueba{
		commitIniciado: make(chan struct{}),
		continuar:      make(chan struct{}),
	}
	conexion := nuevaConexionAcreditacionO405Valida(nil)
	tx := &transaccionAcreditadaO405{Tx: base, conexion: conexion}
	errCommit := make(chan error, 1)
	go func() {
		errCommit <- tx.Commit(context.Background())
	}()
	<-base.commitIniciado
	if err := tx.Rollback(context.Background()); !errors.Is(
		err,
		pgx.ErrTxClosed,
	) {
		t.Fatalf("finalización concurrente=%v; quiere ErrTxClosed", err)
	}
	if conexion.liberaciones != 0 {
		t.Fatal("la finalización rechazada liberó una conexión aún en commit")
	}
	close(base.continuar)
	if err := <-errCommit; err != nil {
		t.Fatal(err)
	}
	if base.commits != 1 || base.rollbacks != 0 ||
		conexion.liberaciones != 1 {
		t.Fatalf(
			"sellado incorrecto: commit=%d rollback=%d liberar=%d",
			base.commits,
			base.rollbacks,
			conexion.liberaciones,
		)
	}
}
