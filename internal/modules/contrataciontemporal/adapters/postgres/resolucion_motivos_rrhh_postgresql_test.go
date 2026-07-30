package postgres

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type filaResolucionMotivosRRHHPrueba struct {
	cardinalidad int64
	referencia   dominiovec.ReferenciaEntradaCatalogo
	version      int64
	err          error
	panico       bool
}

type contextoNuloResolucionMotivosRRHHPrueba struct{}

func (*contextoNuloResolucionMotivosRRHHPrueba) Deadline() (
	time.Time,
	bool,
) {
	return time.Time{}, false
}

func (*contextoNuloResolucionMotivosRRHHPrueba) Done() <-chan struct{} {
	return nil
}

func (*contextoNuloResolucionMotivosRRHHPrueba) Err() error {
	panic("un contexto nulo tipado no debe invocarse")
}

func (*contextoNuloResolucionMotivosRRHHPrueba) Value(any) any {
	return nil
}

func (f filaResolucionMotivosRRHHPrueba) Scan(destinos ...any) error {
	if f.panico {
		panic("fila hostil con DSN y SQL privados")
	}
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 5 {
		return errors.New("número de destinos inesperado")
	}
	*(destinos[0].(*int64)) = f.cardinalidad
	*(destinos[1].(*string)) = f.referencia.CatalogoID
	*(destinos[2].(*int64)) = f.version
	*(destinos[3].(*string)) = f.referencia.CatalogoHuellaSHA256
	*(destinos[4].(*string)) = f.referencia.EntradaClave
	return nil
}

type transaccionResolucionMotivosRRHHPrueba struct {
	sello                  *selloPoolResolucionMotivosRRHH
	fila                   filaResolucionMotivosRRHHPrueba
	errorCommit            error
	errorRollback          error
	antesConsulta          func()
	panicoQuery            bool
	panicoCommit           bool
	panicoRevertir         bool
	consultas              atomic.Int32
	confirmaciones         atomic.Int32
	reversiones            atomic.Int32
	mu                     sync.Mutex
	consulta               string
	argumentos             []any
	errorContextoReversion error
}

func (t *transaccionResolucionMotivosRRHHPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	if t.panicoQuery {
		panic("consulta hostil con secreto")
	}
	if t.antesConsulta != nil {
		t.antesConsulta()
	}
	t.consultas.Add(1)
	t.mu.Lock()
	t.consulta = consulta
	t.argumentos = append([]any(nil), argumentos...)
	t.mu.Unlock()
	return t.fila
}

func (t *transaccionResolucionMotivosRRHHPrueba) Commit(context.Context) error {
	if t.panicoCommit {
		panic("commit hostil con SQLSTATE")
	}
	t.confirmaciones.Add(1)
	return t.errorCommit
}

func (t *transaccionResolucionMotivosRRHHPrueba) Rollback(
	ctx context.Context,
) error {
	t.reversiones.Add(1)
	t.mu.Lock()
	t.errorContextoReversion = ctx.Err()
	t.mu.Unlock()
	if t.panicoRevertir {
		panic("rollback hostil")
	}
	return t.errorRollback
}

func (t *transaccionResolucionMotivosRRHHPrueba) Sello() *selloPoolResolucionMotivosRRHH {
	if t == nil {
		return nil
	}
	return t.sello
}

type conexionResolucionMotivosRRHHPrueba struct {
	sello         *selloPoolResolucionMotivosRRHH
	transaccion   transaccionPoolResolucionMotivosRRHH
	errorInicio   error
	panicoInicio  bool
	panicoLiberar bool
	inicios       atomic.Int32
	liberaciones  atomic.Int32
	mu            sync.Mutex
	opciones      pgx.TxOptions
}

func (*conexionResolucionMotivosRRHHPrueba) Configuracion() *pgx.ConnConfig {
	return nil
}

func (c *conexionResolucionMotivosRRHHPrueba) Sello() *selloPoolResolucionMotivosRRHH {
	if c == nil {
		return nil
	}
	return c.sello
}

func (*conexionResolucionMotivosRRHHPrueba) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	panic("la consulta debe ejecutarse dentro de la transacción")
}

func (c *conexionResolucionMotivosRRHHPrueba) BeginTx(
	_ context.Context,
	opciones pgx.TxOptions,
) (transaccionPoolResolucionMotivosRRHH, error) {
	if c.panicoInicio {
		panic("inicio hostil")
	}
	c.inicios.Add(1)
	c.mu.Lock()
	c.opciones = opciones
	c.mu.Unlock()
	return c.transaccion, c.errorInicio
}

func (c *conexionResolucionMotivosRRHHPrueba) Liberar() {
	c.liberaciones.Add(1)
	if c.panicoLiberar {
		panic("liberación hostil")
	}
}

type origenResolucionMotivosRRHHPrueba struct {
	crearConexion   func() conexionPoolResolucionMotivosRRHH
	errorAdquirir   error
	errorAcreditar  error
	panicoAdquirir  bool
	panicoAcreditar bool
	adquisiciones   atomic.Int32
	acreditaciones  atomic.Int32
}

func (o *origenResolucionMotivosRRHHPrueba) adquirirOperacion(
	context.Context,
) (conexionPoolResolucionMotivosRRHH, error) {
	if o.panicoAdquirir {
		panic("adquisición hostil con DSN")
	}
	o.adquisiciones.Add(1)
	if o.crearConexion == nil {
		return nil, o.errorAdquirir
	}
	return o.crearConexion(), o.errorAdquirir
}

func (o *origenResolucionMotivosRRHHPrueba) reacreditar(
	context.Context,
	transaccionAcreditacionResolucionMotivosRRHH,
) error {
	if o.panicoAcreditar {
		panic("acreditación hostil con rol")
	}
	o.acreditaciones.Add(1)
	return o.errorAcreditar
}

func referenciaResolucionMotivosRRHHValida() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_consultas_rrhh",
		CatalogoVersion:      7,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
}

func instanteResolucionMotivosRRHHPrueba() time.Time {
	return time.Date(2026, time.July, 30, 8, 45, 12, 123_456_000, time.UTC)
}

func prepararResolutorMotivosRRHHPrueba(
	fila filaResolucionMotivosRRHHPrueba,
) (
	*ResolutorMotivosRRHHPostgreSQL,
	*origenResolucionMotivosRRHHPrueba,
	*conexionResolucionMotivosRRHHPrueba,
	*transaccionResolucionMotivosRRHHPrueba,
) {
	sello := &selloPoolResolucionMotivosRRHH{}
	transaccion := &transaccionResolucionMotivosRRHHPrueba{
		sello: sello,
		fila:  fila,
	}
	conexion := &conexionResolucionMotivosRRHHPrueba{
		sello:       sello,
		transaccion: transaccion,
	}
	origen := &origenResolucionMotivosRRHHPrueba{
		crearConexion: func() conexionPoolResolucionMotivosRRHH {
			return conexion
		},
	}
	resolutor, err := nuevoResolutorMotivosRRHHPostgreSQL(origen)
	if err != nil {
		panic(err)
	}
	return resolutor, origen, conexion, transaccion
}

func TestResolutorMotivosRRHHPostgreSQLConsultasNominales(t *testing.T) {
	referencia := referenciaResolucionMotivosRRHHValida()
	fila := filaResolucionMotivosRRHHPrueba{
		cardinalidad: 1,
		referencia:   referencia,
		version:      int64(referencia.CatalogoVersion),
	}
	resolutor, origen, conexion, transaccion :=
		prepararResolutorMotivosRRHHPrueba(fila)

	obtenida, err := resolutor.ResolverMotivoCuadroRRHH(
		context.Background(), instanteResolucionMotivosRRHHPrueba(),
	)
	if err != nil || obtenida != referencia {
		t.Fatalf("resolución de cuadro: resultado=%+v error=%v", obtenida, err)
	}
	if origen.adquisiciones.Load() != 1 || origen.acreditaciones.Load() != 1 ||
		conexion.inicios.Load() != 1 || conexion.liberaciones.Load() != 1 ||
		transaccion.consultas.Load() != 1 ||
		transaccion.confirmaciones.Load() != 1 ||
		transaccion.reversiones.Load() != 0 {
		t.Fatal("el ciclo transaccional positivo no fue exacto")
	}
	if conexion.opciones.IsoLevel != pgx.Serializable ||
		conexion.opciones.AccessMode != pgx.ReadWrite {
		t.Fatalf("opciones inseguras: %+v", conexion.opciones)
	}
	if transaccion.consulta != consultaResolucionMotivoCuadroRRHHPostgreSQL ||
		len(transaccion.argumentos) != 1 ||
		transaccion.argumentos[0] != instanteResolucionMotivosRRHHPrueba() {
		t.Fatal("la consulta de cuadro o sus argumentos no son nominales")
	}

	resolutor, _, _, transaccion = prepararResolutorMotivosRRHHPrueba(fila)
	obtenida, err = resolutor.ResolverMotivoDetalleRRHH(
		context.Background(), instanteResolucionMotivosRRHHPrueba(),
	)
	if err != nil || obtenida != referencia ||
		transaccion.consulta != consultaResolucionMotivoDetalleRRHHPostgreSQL {
		t.Fatalf("resolución de detalle no nominal: %+v, %v", obtenida, err)
	}
	if consultaResolucionMotivoCuadroRRHHPostgreSQL ==
		consultaResolucionMotivoDetalleRRHHPostgreSQL {
		t.Fatal("las consultas nominales no pueden compartir un literal")
	}
	for _, consulta := range []string{
		consultaResolucionMotivoCuadroRRHHPostgreSQL,
		consultaResolucionMotivoDetalleRRHHPostgreSQL,
	} {
		if strings.Count(consulta, "$1::timestamptz") != 1 ||
			!strings.Contains(consulta, "LIMIT 2") ||
			!strings.Contains(consulta, "pg_catalog.count(*)::bigint") ||
			strings.Count(consulta, "catalogo_id") != 2 ||
			strings.Count(consulta, "catalogo_version") != 2 ||
			strings.Count(consulta, "catalogo_huella_sha256") != 2 ||
			strings.Count(consulta, "entrada_clave") != 2 {
			t.Fatalf("consulta literal incompleta: %q", consulta)
		}
	}
	if strings.Count(
		consultaResolucionMotivoCuadroRRHHPostgreSQL,
		"vec_autorizacion.resolver_motivo_cuadro_rrhh_v1",
	) != 1 || strings.Contains(
		consultaResolucionMotivoCuadroRRHHPostgreSQL,
		"resolver_motivo_detalle_rrhh_v1",
	) || strings.Count(
		consultaResolucionMotivoDetalleRRHHPostgreSQL,
		"vec_autorizacion.resolver_motivo_detalle_rrhh_v1",
	) != 1 || strings.Contains(
		consultaResolucionMotivoDetalleRRHHPostgreSQL,
		"resolver_motivo_cuadro_rrhh_v1",
	) {
		t.Fatal("las fachadas SQL nominales derivaron")
	}
}

func TestResolutorMotivosRRHHPostgreSQLFallaCerradoConEntradaInvalida(
	t *testing.T,
) {
	referencia := referenciaResolucionMotivosRRHHValida()
	fila := filaResolucionMotivosRRHHPrueba{
		cardinalidad: 1, referencia: referencia,
		version: int64(referencia.CatalogoVersion),
	}
	casos := []struct {
		nombre   string
		contexto context.Context
		instante time.Time
	}{
		{"contexto nulo", nil, instanteResolucionMotivosRRHHPrueba()},
		{"instante cero", context.Background(), time.Time{}},
		{"zona no canónica", context.Background(),
			instanteResolucionMotivosRRHHPrueba().In(time.FixedZone("UTC", 0))},
		{"precisión nanosegundo", context.Background(),
			instanteResolucionMotivosRRHHPrueba().Add(time.Nanosecond)},
		{"año cero", context.Background(),
			time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)},
		{"año diez mil", context.Background(),
			time.Date(10_000, time.January, 1, 0, 0, 0, 0, time.UTC)},
	}
	var contextoNuloTipado *contextoNuloResolucionMotivosRRHHPrueba
	casos = append(casos, struct {
		nombre   string
		contexto context.Context
		instante time.Time
	}{
		"contexto nulo tipado",
		contextoNuloTipado,
		instanteResolucionMotivosRRHHPrueba(),
	})
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	casos = append(casos, struct {
		nombre   string
		contexto context.Context
		instante time.Time
	}{"cancelado", cancelado, instanteResolucionMotivosRRHHPrueba()})
	vencido, cancelarVencido := context.WithDeadline(
		context.Background(), time.Now().Add(-time.Second),
	)
	defer cancelarVencido()
	casos = append(casos, struct {
		nombre   string
		contexto context.Context
		instante time.Time
	}{"vencido", vencido, instanteResolucionMotivosRRHHPrueba()})

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resolutor, origen, _, _ :=
				prepararResolutorMotivosRRHHPrueba(fila)
			obtenida, err := resolutor.ResolverMotivoCuadroRRHH(
				caso.contexto, caso.instante,
			)
			if obtenida != (dominiovec.ReferenciaEntradaCatalogo{}) ||
				!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
				origen.adquisiciones.Load() != 0 {
				t.Fatalf("entrada no cerrada: %+v, %v", obtenida, err)
			}
			if !dependenciaNula(caso.contexto) &&
				caso.contexto.Err() != nil &&
				!errors.Is(err, caso.contexto.Err()) {
				t.Fatalf("se perdió el estado del contexto: %v", err)
			}
		})
	}

	var nulo *ResolutorMotivosRRHHPostgreSQL
	if obtenida, err := nulo.ResolverMotivoCuadroRRHH(
		context.Background(), instanteResolucionMotivosRRHHPrueba(),
	); obtenida != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatal("el receptor nulo no falló cerrado")
	}
}

func TestResolutorMotivosRRHHPostgreSQLRechazaCardinalidadYReferencia(
	t *testing.T,
) {
	valida := referenciaResolucionMotivosRRHHValida()
	casos := []struct {
		nombre string
		fila   filaResolucionMotivosRRHHPrueba
	}{
		{"sin filas", filaResolucionMotivosRRHHPrueba{}},
		{"dos filas", filaResolucionMotivosRRHHPrueba{
			cardinalidad: 2, referencia: valida, version: 7,
		}},
		{"versión cero", filaResolucionMotivosRRHHPrueba{
			cardinalidad: 1, referencia: valida,
		}},
		{"versión negativa", filaResolucionMotivosRRHHPrueba{
			cardinalidad: 1, referencia: valida, version: -1,
		}},
		{"versión desbordada", filaResolucionMotivosRRHHPrueba{
			cardinalidad: 1, referencia: valida, version: math.MaxInt32 + 1,
		}},
		{"catálogo inválido", filaResolucionMotivosRRHHPrueba{
			cardinalidad: 1,
			referencia: func() dominiovec.ReferenciaEntradaCatalogo {
				r := valida
				r.CatalogoID = "../privado"
				return r
			}(),
			version: 7,
		}},
		{"huella inválida", filaResolucionMotivosRRHHPrueba{
			cardinalidad: 1,
			referencia: func() dominiovec.ReferenciaEntradaCatalogo {
				r := valida
				r.CatalogoHuellaSHA256 = strings.Repeat("0", 64)
				return r
			}(),
			version: 7,
		}},
		{"clave inválida", filaResolucionMotivosRRHHPrueba{
			cardinalidad: 1,
			referencia: func() dominiovec.ReferenciaEntradaCatalogo {
				r := valida
				r.EntradaClave = "texto_humano"
				return r
			}(),
			version: 7,
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resolutor, _, conexion, transaccion :=
				prepararResolutorMotivosRRHHPrueba(caso.fila)
			obtenida, err := resolutor.ResolverMotivoCuadroRRHH(
				context.Background(), instanteResolucionMotivosRRHHPrueba(),
			)
			if obtenida != (dominiovec.ReferenciaEntradaCatalogo{}) ||
				!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
				transaccion.confirmaciones.Load() != 0 ||
				transaccion.reversiones.Load() != 1 ||
				conexion.liberaciones.Load() != 1 {
				t.Fatalf("salida incierta no cerrada: %+v, %v", obtenida, err)
			}
		})
	}
}

func TestResolutorMotivosRRHHPostgreSQLRechazaDependenciasNulasTipadas(
	t *testing.T,
) {
	valida := referenciaResolucionMotivosRRHHValida()
	fila := filaResolucionMotivosRRHHPrueba{
		cardinalidad: 1, referencia: valida, version: 7,
	}
	resolutor, origen, _, _ := prepararResolutorMotivosRRHHPrueba(fila)
	origen.crearConexion = func() conexionPoolResolucionMotivosRRHH {
		var conexion *conexionResolucionMotivosRRHHPrueba
		return conexion
	}
	if resultado, err := resolutor.ResolverMotivoCuadroRRHH(
		context.Background(), instanteResolucionMotivosRRHHPrueba(),
	); resultado != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatal("se aceptó una conexión nula tipada")
	}

	resolutor, _, conexion, _ :=
		prepararResolutorMotivosRRHHPrueba(fila)
	var transaccion *transaccionResolucionMotivosRRHHPrueba
	conexion.transaccion = transaccion
	if resultado, err := resolutor.ResolverMotivoCuadroRRHH(
		context.Background(), instanteResolucionMotivosRRHHPrueba(),
	); resultado != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
		conexion.liberaciones.Load() != 1 {
		t.Fatal("se aceptó una transacción nula tipada")
	}
}

func TestResolutorMotivosRRHHPostgreSQLRevierteConContextoDeLimpieza(
	t *testing.T,
) {
	valida := referenciaResolucionMotivosRRHHValida()
	fila := filaResolucionMotivosRRHHPrueba{
		cardinalidad: 1, referencia: valida, version: 7,
	}
	resolutor, _, _, transaccion :=
		prepararResolutorMotivosRRHHPrueba(fila)
	ctx, cancelar := context.WithCancel(context.Background())
	transaccion.antesConsulta = cancelar
	transaccion.errorCommit = context.Canceled

	resultado, err := resolutor.ResolverMotivoCuadroRRHH(
		ctx, instanteResolucionMotivosRRHHPrueba(),
	)
	if resultado != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, context.Canceled) ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
		transaccion.reversiones.Load() != 1 {
		t.Fatalf("cancelación incierta no cerrada: %+v, %v", resultado, err)
	}
	transaccion.mu.Lock()
	errorContextoReversion := transaccion.errorContextoReversion
	transaccion.mu.Unlock()
	if errorContextoReversion != nil {
		t.Fatalf(
			"rollback heredó el contexto cancelado: %v",
			errorContextoReversion,
		)
	}
}

func TestResolutorMotivosRRHHPostgreSQLCierraFallosDelCiclo(t *testing.T) {
	errPrivado := errors.New(
		"postgres://usuario:clave@interno SQLSTATE 42501 SELECT secreto",
	)
	valida := referenciaResolucionMotivosRRHHValida()
	filaValida := filaResolucionMotivosRRHHPrueba{
		cardinalidad: 1, referencia: valida, version: 7,
	}
	casos := []struct {
		nombre  string
		alterar func(
			*origenResolucionMotivosRRHHPrueba,
			*conexionResolucionMotivosRRHHPrueba,
			*transaccionResolucionMotivosRRHHPrueba,
		)
	}{
		{"adquisición", func(o *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			_ *transaccionResolucionMotivosRRHHPrueba) {
			o.errorAdquirir = errPrivado
			o.crearConexion = nil
		}},
		{"inicio", func(_ *origenResolucionMotivosRRHHPrueba,
			c *conexionResolucionMotivosRRHHPrueba,
			_ *transaccionResolucionMotivosRRHHPrueba) {
			c.errorInicio = errPrivado
			c.transaccion = nil
		}},
		{"sello", func(_ *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			tx *transaccionResolucionMotivosRRHHPrueba) {
			tx.sello = &selloPoolResolucionMotivosRRHH{}
		}},
		{"acreditación", func(o *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			_ *transaccionResolucionMotivosRRHHPrueba) {
			o.errorAcreditar = errPrivado
		}},
		{"escaneo", func(_ *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			tx *transaccionResolucionMotivosRRHHPrueba) {
			tx.fila.err = errPrivado
		}},
		{"commit", func(_ *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			tx *transaccionResolucionMotivosRRHHPrueba) {
			tx.errorCommit = errPrivado
			tx.errorRollback = errPrivado
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resolutor, origen, conexion, transaccion :=
				prepararResolutorMotivosRRHHPrueba(filaValida)
			caso.alterar(origen, conexion, transaccion)
			obtenida, err := resolutor.ResolverMotivoCuadroRRHH(
				context.Background(), instanteResolucionMotivosRRHHPrueba(),
			)
			if obtenida != (dominiovec.ReferenciaEntradaCatalogo{}) ||
				!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
				t.Fatalf("fallo no opaco: %+v, %v", obtenida, err)
			}
			for _, secreto := range []string{
				"postgres://", "SQLSTATE", "SELECT", "clave",
			} {
				if strings.Contains(err.Error(), secreto) {
					t.Fatalf("se filtró %q en %q", secreto, err.Error())
				}
			}
			if caso.nombre != "adquisición" &&
				conexion.liberaciones.Load() != 1 {
				t.Fatal("la conexión adquirida no fue liberada")
			}
			if caso.nombre != "adquisición" && caso.nombre != "inicio" &&
				transaccion.reversiones.Load() != 1 {
				t.Fatal("la transacción incierta no se revirtió")
			}
		})
	}
}

func TestResolutorMotivosRRHHPostgreSQLRecuperaPanicos(t *testing.T) {
	valida := referenciaResolucionMotivosRRHHValida()
	fila := filaResolucionMotivosRRHHPrueba{
		cardinalidad: 1, referencia: valida, version: 7,
	}
	resolutor, origen, _, _ := prepararResolutorMotivosRRHHPrueba(fila)
	origen.panicoAdquirir = true
	if resultado, err := resolutor.ResolverMotivoCuadroRRHH(
		context.Background(), instanteResolucionMotivosRRHHPrueba(),
	); resultado != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatal("el pánico de adquisición escapó")
	}

	casos := []struct {
		nombre  string
		alterar func(
			*origenResolucionMotivosRRHHPrueba,
			*conexionResolucionMotivosRRHHPrueba,
			*transaccionResolucionMotivosRRHHPrueba,
		)
	}{
		{"inicio", func(_ *origenResolucionMotivosRRHHPrueba,
			c *conexionResolucionMotivosRRHHPrueba,
			_ *transaccionResolucionMotivosRRHHPrueba) {
			c.panicoInicio = true
		}},
		{"acreditación", func(o *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			_ *transaccionResolucionMotivosRRHHPrueba) {
			o.panicoAcreditar = true
		}},
		{"consulta", func(_ *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			tx *transaccionResolucionMotivosRRHHPrueba) {
			tx.panicoQuery = true
		}},
		{"escaneo", func(_ *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			tx *transaccionResolucionMotivosRRHHPrueba) {
			tx.fila.panico = true
		}},
		{"commit", func(_ *origenResolucionMotivosRRHHPrueba,
			_ *conexionResolucionMotivosRRHHPrueba,
			tx *transaccionResolucionMotivosRRHHPrueba) {
			tx.panicoCommit = true
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resolutor, origen, conexion, transaccion :=
				prepararResolutorMotivosRRHHPrueba(fila)
			caso.alterar(origen, conexion, transaccion)
			resultado, err := resolutor.ResolverMotivoCuadroRRHH(
				context.Background(), instanteResolucionMotivosRRHHPrueba(),
			)
			if resultado != (dominiovec.ReferenciaEntradaCatalogo{}) ||
				!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
				conexion.liberaciones.Load() != 1 {
				t.Fatalf("el pánico %s escapó: %+v, %v",
					caso.nombre, resultado, err)
			}
			if caso.nombre != "inicio" &&
				transaccion.reversiones.Load() != 1 {
				t.Fatal("la transacción incierta no se revirtió")
			}
		})
	}

	resolutor, _, conexion, transaccion :=
		prepararResolutorMotivosRRHHPrueba(fila)
	transaccion.panicoQuery = true
	transaccion.panicoRevertir = true
	conexion.panicoLiberar = true
	if resultado, err := resolutor.ResolverMotivoCuadroRRHH(
		context.Background(), instanteResolucionMotivosRRHHPrueba(),
	); resultado != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatal("la limpieza hostil no falló cerrada")
	}
	if conexion.liberaciones.Load() != 1 ||
		transaccion.reversiones.Load() != 1 {
		t.Fatal("la limpieza no se intentó exactamente una vez")
	}
}

func TestNuevoResolutorMotivoConsultaRRHHPostgreSQLCierraComposicion(
	t *testing.T,
) {
	if resolutor, err := NuevoResolutorMotivoConsultaRRHHPostgreSQL(nil); resolutor != nil ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatal("el constructor público aceptó un pool nulo")
	}
	var origenNulo *origenResolucionMotivosRRHHPrueba
	if resolutor, err := nuevoResolutorMotivosRRHHPostgreSQL(origenNulo); resolutor != nil ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatal("el constructor privado aceptó un nulo tipado")
	}

	pool := &PoolResolucionMotivosRRHHPostgreSQL{
		login:      "vec_ct_motivos_prueba",
		modoTLS:    modoTLSAcreditacionPoolO405SocketUnixPrueba,
		oidCuadro:  41,
		oidDetalle: 42,
	}
	pool.sello = &selloPoolResolucionMotivosRRHH{
		dependencia:              pool,
		login:                    pool.login,
		modo:                     pool.modoTLS,
		oidCuadro:                pool.oidCuadro,
		oidDetalle:               pool.oidDetalle,
		callbacksPredeterminados: true,
	}
	if resolutor, err := NuevoResolutorMotivoConsultaRRHHPostgreSQL(pool); err != nil || resolutor == nil {
		t.Fatalf("pool acreditado rechazado: %v", err)
	}
	pool.sello.oidDetalle = 0
	if resolutor, err := NuevoResolutorMotivoConsultaRRHHPostgreSQL(pool); resolutor != nil ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatal("el constructor aceptó un sello degradado")
	}
}

func TestResolutorMotivosRRHHPostgreSQLEsConcurrente(t *testing.T) {
	valida := referenciaResolucionMotivosRRHHValida()
	fila := filaResolucionMotivosRRHHPrueba{
		cardinalidad: 1, referencia: valida, version: 7,
	}
	sello := &selloPoolResolucionMotivosRRHH{}
	origen := &origenResolucionMotivosRRHHPrueba{}
	origen.crearConexion = func() conexionPoolResolucionMotivosRRHH {
		tx := &transaccionResolucionMotivosRRHHPrueba{
			sello: sello,
			fila:  fila,
		}
		return &conexionResolucionMotivosRRHHPrueba{
			sello: sello, transaccion: tx,
		}
	}
	resolutor, err := nuevoResolutorMotivosRRHHPostgreSQL(origen)
	if err != nil {
		t.Fatal(err)
	}
	const total = 32
	var grupo sync.WaitGroup
	errores := make(chan error, total)
	for indice := 0; indice < total; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			obtenida, err := resolutor.ResolverMotivoDetalleRRHH(
				context.Background(), instanteResolucionMotivosRRHHPrueba(),
			)
			if err != nil || obtenida != valida {
				errores <- errors.New("resolución concurrente fallida")
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatal(err)
	}
	if origen.adquisiciones.Load() != total ||
		origen.acreditaciones.Load() != total {
		t.Fatal("se perdieron operaciones concurrentes")
	}
}
