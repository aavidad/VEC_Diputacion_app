package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const loginResolutorMotivosRRHHPrueba = "vec_ct_motivos_prueba"

type filaAcreditacionResolutorPrueba struct {
	valores []any
	err     error
}

func (f filaAcreditacionResolutorPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("destinos no coincidentes")
	}
	for i, destino := range destinos {
		switch puntero := destino.(type) {
		case *string:
			*puntero = f.valores[i].(string)
		case *uint32:
			*puntero = f.valores[i].(uint32)
		case *bool:
			*puntero = f.valores[i].(bool)
		default:
			return errors.New("destino inválido")
		}
	}
	return nil
}

type conexionPoolResolucionMotivosRRHHPrueba struct {
	configuracion *pgx.ConnConfig
	sello         *selloPoolResolucionMotivosRRHH
	fila          filaAcreditacionResolutorPrueba
	consulta      string
	argumentos    []any
	liberaciones  atomic.Int32
	panicoQuery   bool
	panicoLiberar bool
}

func (c *conexionPoolResolucionMotivosRRHHPrueba) Configuracion() *pgx.ConnConfig {
	if c == nil {
		return nil
	}
	return c.configuracion
}

func (c *conexionPoolResolucionMotivosRRHHPrueba) Sello() *selloPoolResolucionMotivosRRHH {
	if c == nil {
		return nil
	}
	return c.sello
}

func (c *conexionPoolResolucionMotivosRRHHPrueba) QueryRow(
	_ context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	if c.panicoQuery {
		panic("detalle que no debe escapar")
	}
	c.consulta = consulta
	c.argumentos = append([]any(nil), argumentos...)
	return c.fila
}

func (*conexionPoolResolucionMotivosRRHHPrueba) BeginTx(
	context.Context,
	pgx.TxOptions,
) (pgx.Tx, error) {
	return nil, errors.New("no usado en M2.1")
}

func (c *conexionPoolResolucionMotivosRRHHPrueba) Liberar() {
	c.liberaciones.Add(1)
	if c.panicoLiberar {
		panic("liberación hostil")
	}
}

type origenPoolResolucionMotivosRRHHPrueba struct {
	configuracion *pgxpool.Config
	sello         *selloPoolResolucionMotivosRRHH
	conexion      *conexionPoolResolucionMotivosRRHHPrueba
	errorAdquirir error
	panico        bool
	vinculable    bool
	adquisiciones atomic.Int32
	cierres       atomic.Int32
}

func (o *origenPoolResolucionMotivosRRHHPrueba) Configuracion() *pgxpool.Config {
	if o == nil {
		return nil
	}
	return o.configuracion
}

func (o *origenPoolResolucionMotivosRRHHPrueba) Sello() *selloPoolResolucionMotivosRRHH {
	if o == nil {
		return nil
	}
	return o.sello
}

func (o *origenPoolResolucionMotivosRRHHPrueba) VincularSello(
	sello *selloPoolResolucionMotivosRRHH,
) bool {
	if o == nil || !o.vinculable || o.sello != nil || sello == nil {
		return false
	}
	o.sello = sello
	return true
}

func (o *origenPoolResolucionMotivosRRHHPrueba) Adquirir(
	context.Context,
) (conexionPoolResolucionMotivosRRHH, error) {
	if o.panico {
		panic("adquisición hostil")
	}
	o.adquisiciones.Add(1)
	if o.conexion != nil && o.conexion.sello == nil {
		o.conexion.sello = o.sello
	}
	return o.conexion, o.errorAdquirir
}

func (o *origenPoolResolucionMotivosRRHHPrueba) Cerrar() {
	o.cierres.Add(1)
}

func valoresAcreditacionResolutorValidos(login string) []any {
	valores := []any{login, login, uint32(41001), uint32(41002)}
	for range 11 {
		valores = append(valores, true)
	}
	return valores
}

func configuracionPoolResolucionMotivosRRHHPrueba(
	login string,
) *pgxpool.Config {
	configuracion := configuracionPoolAcreditacionO405Prueba()
	configuracion.ConnConfig.User = login
	return configuracion
}

func nuevoOrigenPoolResolucionMotivosRRHHPrueba(
	login string,
) *origenPoolResolucionMotivosRRHHPrueba {
	configuracion := configuracionPoolResolucionMotivosRRHHPrueba(login)
	return &origenPoolResolucionMotivosRRHHPrueba{
		configuracion: configuracion,
		vinculable:    true,
		conexion: &conexionPoolResolucionMotivosRRHHPrueba{
			configuracion: configuracion.ConnConfig,
			fila: filaAcreditacionResolutorPrueba{
				valores: valoresAcreditacionResolutorValidos(login),
			},
		},
	}
}

func construirPoolResolucionMotivosRRHHPrueba(
	ctx context.Context,
	origen *origenPoolResolucionMotivosRRHHPrueba,
) (*PoolResolucionMotivosRRHHPostgreSQL, error) {
	return construirPoolResolucionMotivosRRHH(
		ctx,
		origen.configuracion,
		loginResolutorMotivosRRHHPrueba,
		modoTLSAcreditacionPoolO405Produccion,
		func(context.Context, *pgxpool.Config) (
			origenPoolResolucionMotivosRRHH,
			error,
		) {
			return origen, nil
		},
	)
}

func TestAcreditacionResolucionMotivosRRHHFallaCerradoAnteCadaDeriva(
	t *testing.T,
) {
	t.Parallel()
	for indice := 4; indice < 15; indice++ {
		indice := indice
		t.Run("deriva", func(t *testing.T) {
			t.Parallel()
			valores := valoresAcreditacionResolutorValidos(
				loginResolutorMotivosRRHHPrueba,
			)
			valores[indice] = false
			_, _, err := acreditarConexionResolucionMotivosRRHH(
				context.Background(),
				&conexionPoolResolucionMotivosRRHHPrueba{
					fila: filaAcreditacionResolutorPrueba{valores: valores},
				},
				loginResolutorMotivosRRHHPrueba,
				modoTLSAcreditacionPoolO405Produccion,
				0,
				0,
			)
			if !errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
				t.Fatalf("deriva %d admitida: %v", indice, err)
			}
		})
	}
}

func TestAcreditacionResolucionMotivosRRHHExigeIdentidadOIDYConsultaCompleta(
	t *testing.T,
) {
	t.Parallel()
	conexion := &conexionPoolResolucionMotivosRRHHPrueba{
		fila: filaAcreditacionResolutorPrueba{
			valores: valoresAcreditacionResolutorValidos(
				loginResolutorMotivosRRHHPrueba,
			),
		},
	}
	oidCuadro, oidDetalle, err := acreditarConexionResolucionMotivosRRHH(
		context.Background(),
		conexion,
		loginResolutorMotivosRRHHPrueba,
		modoTLSAcreditacionPoolO405Produccion,
		41001,
		41002,
	)
	if err != nil || oidCuadro != 41001 || oidDetalle != 41002 {
		t.Fatalf("acreditación válida: (%d,%d,%v)", oidCuadro, oidDetalle, err)
	}
	fragmentos := []string{
		"pg_advisory_xact_lock_shared",
		"vinculaciones-motivo-rrhh:000008",
		"vinculaciones-motivo-rrhh:000009",
		"vinculaciones-motivo-rrhh:000010",
		"pg_stat_ssl",
		"TLSv1.3",
		"pg_auth_members",
		"pg_has_role",
		"oid=10 AND rolsuper",
		"aclexplode",
		"EXCEPT ALL",
		"has_database_privilege",
		"has_schema_privilege",
		"has_function_privilege",
		"has_table_privilege",
		"has_any_column_privilege",
		"has_sequence_privilege",
		"pg_default_acl",
		"pg_policy",
		"pg_shdepend",
		"d92704658d0af8acea83cd765e02976561c787a95906b9a10ee8a43ac0be16ef",
		"ec662cc7118eb25eb2ebe79107c1ad1f16e5f5197fab8ff5e2051b3ddbc9fc7a",
		"6699",
		"6702",
	}
	for _, fragmento := range fragmentos {
		if !strings.Contains(conexion.consulta, fragmento) {
			t.Fatalf("falta acreditación %q", fragmento)
		}
	}
	if len(conexion.argumentos) != 2 ||
		conexion.argumentos[0] != loginResolutorMotivosRRHHPrueba ||
		conexion.argumentos[1] != true {
		t.Fatalf("argumentos inesperados: %#v", conexion.argumentos)
	}
}

func TestAcreditacionResolucionMotivosRRHHRechazaOIDRecreadoYFilasHostiles(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre    string
		mutar     func([]any)
		errorFila error
		panico    bool
	}{
		{"cuadro recreado", func(v []any) { v[2] = uint32(42001) }, nil, false},
		{"detalle recreado", func(v []any) { v[3] = uint32(42002) }, nil, false},
		{"oid cero", func(v []any) { v[2] = uint32(0) }, nil, false},
		{"oid repetido", func(v []any) { v[3] = uint32(41001) }, nil, false},
		{"fallo scan", func([]any) {}, errors.New("detalle SQL"), false},
		{"pánico scan", func(v []any) { v[0] = 9 }, nil, false},
		{"pánico query", func([]any) {}, nil, true},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			valores := valoresAcreditacionResolutorValidos(
				loginResolutorMotivosRRHHPrueba,
			)
			caso.mutar(valores)
			conexion := &conexionPoolResolucionMotivosRRHHPrueba{
				fila: filaAcreditacionResolutorPrueba{
					valores: valores,
					err:     caso.errorFila,
				},
				panicoQuery: caso.panico,
			}
			oidCuadro, oidDetalle, err := acreditarConexionResolucionMotivosRRHH(
				context.Background(),
				conexion,
				loginResolutorMotivosRRHHPrueba,
				modoTLSAcreditacionPoolO405Produccion,
				41001,
				41002,
			)
			if oidCuadro != 0 || oidDetalle != 0 ||
				!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
				strings.Contains(err.Error(), "detalle SQL") {
				t.Fatalf("fallo no opaco: (%d,%d,%v)", oidCuadro, oidDetalle, err)
			}
		})
	}
}

func TestAcreditacionResolucionMotivosRRHHConservaContextoYAdmiteSocketPrivado(
	t *testing.T,
) {
	t.Parallel()
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	ctxVencido, terminar := context.WithDeadline(
		context.Background(), time.Now().Add(-time.Second),
	)
	defer terminar()
	for _, caso := range []struct {
		ctx      context.Context
		esperado error
	}{
		{ctxCancelado, context.Canceled},
		{ctxVencido, context.DeadlineExceeded},
	} {
		_, _, err := acreditarConexionResolucionMotivosRRHH(
			caso.ctx,
			(*conexionPoolResolucionMotivosRRHHPrueba)(nil),
			loginResolutorMotivosRRHHPrueba,
			modoTLSAcreditacionPoolO405Produccion,
			0,
			0,
		)
		if !errors.Is(err, caso.esperado) ||
			!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
			t.Fatalf("contexto sin ambos errores: %v", err)
		}
	}

	conexion := &conexionPoolResolucionMotivosRRHHPrueba{
		fila: filaAcreditacionResolutorPrueba{
			valores: valoresAcreditacionResolutorValidos(
				loginResolutorMotivosRRHHPrueba,
			),
		},
	}
	if _, _, err := acreditarConexionResolucionMotivosRRHH(
		context.Background(),
		conexion,
		loginResolutorMotivosRRHHPrueba,
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
		0,
		0,
	); err != nil {
		t.Fatalf("socket privado rechazado: %v", err)
	}
	if len(conexion.argumentos) != 2 || conexion.argumentos[1] != false {
		t.Fatalf("socket pidió TLS: %#v", conexion.argumentos)
	}
}

func TestConstruccionPoolResolucionMotivosRRHHSellaOIDYCierreConcurrente(
	t *testing.T,
) {
	t.Parallel()
	origen := nuevoOrigenPoolResolucionMotivosRRHHPrueba(
		loginResolutorMotivosRRHHPrueba,
	)
	pool, err := construirPoolResolucionMotivosRRHHPrueba(
		context.Background(), origen,
	)
	if err != nil {
		t.Fatalf("construcción válida: %v", err)
	}
	if pool.oidCuadro != 41001 || pool.oidDetalle != 41002 ||
		!selloPoolResolucionMotivosRRHHValido(pool.sello, pool, true) ||
		origen.conexion.liberaciones.Load() != 1 {
		t.Fatal("pool no quedó sellado o readiness no liberó la conexión")
	}
	var grupo sync.WaitGroup
	for range 32 {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			pool.Cerrar()
		}()
	}
	grupo.Wait()
	if origen.cierres.Load() != 1 {
		t.Fatalf("cierre no idempotente: %d", origen.cierres.Load())
	}
}

func TestConstruccionPoolResolucionMotivosRRHHFallaCerradoYLimpiaParcial(
	t *testing.T,
) {
	t.Parallel()
	casos := []struct {
		nombre string
		mutar  func(*origenPoolResolucionMotivosRRHHPrueba)
	}{
		{"sello rechazado", func(o *origenPoolResolucionMotivosRRHHPrueba) {
			o.vinculable = false
		}},
		{"adquisición falla", func(o *origenPoolResolucionMotivosRRHHPrueba) {
			o.errorAdquirir = errors.New("dsn sensible")
			o.conexion = nil
		}},
		{"conexión y error", func(o *origenPoolResolucionMotivosRRHHPrueba) {
			o.errorAdquirir = errors.New("resultado incierto")
		}},
		{"pánico adquisición", func(o *origenPoolResolucionMotivosRRHHPrueba) {
			o.panico = true
		}},
		{"sello conexión distinto", func(o *origenPoolResolucionMotivosRRHHPrueba) {
			o.conexion.sello = &selloPoolResolucionMotivosRRHH{}
		}},
		{"config conexión distinta", func(o *origenPoolResolucionMotivosRRHHPrueba) {
			configuracion := o.conexion.configuracion.Copy()
			configuracion.User = "otro_login"
			o.conexion.configuracion = configuracion
		}},
		{"manifiesto degradado", func(o *origenPoolResolucionMotivosRRHHPrueba) {
			o.conexion.fila.valores[13] = false
		}},
		{"pánico liberación", func(o *origenPoolResolucionMotivosRRHHPrueba) {
			o.conexion.panicoLiberar = true
		}},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			origen := nuevoOrigenPoolResolucionMotivosRRHHPrueba(
				loginResolutorMotivosRRHHPrueba,
			)
			caso.mutar(origen)
			pool, err := construirPoolResolucionMotivosRRHHPrueba(
				context.Background(), origen,
			)
			if pool != nil ||
				!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
				strings.Contains(err.Error(), "dsn sensible") {
				t.Fatalf("fallo parcial no saneado: (%v,%v)", pool, err)
			}
			if origen.cierres.Load() != 1 {
				t.Fatalf("origen parcial no cerrado: %d", origen.cierres.Load())
			}
			if origen.conexion != nil && origen.adquisiciones.Load() > 0 &&
				origen.conexion.liberaciones.Load() != 1 {
				t.Fatalf(
					"conexión parcial no liberada: %d",
					origen.conexion.liberaciones.Load(),
				)
			}
		})
	}
}

func TestConstruccionPoolResolucionMotivosRRHHRechazaFronterasEIncertidumbre(
	t *testing.T,
) {
	t.Parallel()
	configuracion := configuracionPoolResolucionMotivosRRHHPrueba(
		loginResolutorMotivosRRHHPrueba,
	)
	origen := nuevoOrigenPoolResolucionMotivosRRHHPrueba(
		loginResolutorMotivosRRHHPrueba,
	)
	var llamadas atomic.Int32
	crearValido := func(context.Context, *pgxpool.Config) (
		origenPoolResolucionMotivosRRHH,
		error,
	) {
		llamadas.Add(1)
		return origen, nil
	}
	ctxCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	for _, caso := range []struct {
		nombre string
		ctx    context.Context
		config *pgxpool.Config
		crear  creadorOrigenPoolResolucionMotivosRRHH
	}{
		{"contexto nulo", nil, configuracion, crearValido},
		{"contexto cancelado", ctxCancelado, configuracion, crearValido},
		{"configuración nula", context.Background(), nil, crearValido},
		{"creador nulo", context.Background(), configuracion, nil},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			pool, err := construirPoolResolucionMotivosRRHH(
				caso.ctx,
				caso.config,
				loginResolutorMotivosRRHHPrueba,
				modoTLSAcreditacionPoolO405Produccion,
				caso.crear,
			)
			if pool != nil ||
				!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
				t.Fatalf("frontera insegura admitida: (%v,%v)", pool, err)
			}
		})
	}
	if llamadas.Load() != 0 {
		t.Fatalf("se invocó el creador tras una frontera inválida: %d", llamadas.Load())
	}

	for _, caso := range []struct {
		nombre string
		crear  creadorOrigenPoolResolucionMotivosRRHH
		cerrar bool
	}{
		{
			"origen y error",
			func(context.Context, *pgxpool.Config) (
				origenPoolResolucionMotivosRRHH,
				error,
			) {
				return origen, errors.New("detalle de conexión reservado")
			},
			true,
		},
		{
			"origen nulo tipado",
			func(context.Context, *pgxpool.Config) (
				origenPoolResolucionMotivosRRHH,
				error,
			) {
				return (*origenPoolResolucionMotivosRRHHPrueba)(nil), nil
			},
			false,
		},
		{
			"pánico del creador",
			func(context.Context, *pgxpool.Config) (
				origenPoolResolucionMotivosRRHH,
				error,
			) {
				panic("detalle reservado")
			},
			false,
		},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			pool, err := construirPoolResolucionMotivosRRHH(
				context.Background(),
				configuracion,
				loginResolutorMotivosRRHHPrueba,
				modoTLSAcreditacionPoolO405Produccion,
				caso.crear,
			)
			if pool != nil ||
				!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) ||
				strings.Contains(err.Error(), "reservado") {
				t.Fatalf("incertidumbre no saneada: (%v,%v)", pool, err)
			}
			if caso.cerrar != (origen.cierres.Load() == 1) {
				t.Fatalf("cierre parcial inesperado: %d", origen.cierres.Load())
			}
			origen.cierres.Store(0)
		})
	}
}

func TestPoolResolucionMotivosRRHHRechazaCopiaYConfiguracionMutada(
	t *testing.T,
) {
	t.Parallel()
	origen := nuevoOrigenPoolResolucionMotivosRRHHPrueba(
		loginResolutorMotivosRRHHPrueba,
	)
	pool, err := construirPoolResolucionMotivosRRHHPrueba(
		context.Background(), origen,
	)
	if err != nil {
		t.Fatalf("construcción válida: %v", err)
	}
	adquisiciones := origen.adquisiciones.Load()
	copia := *pool
	if conexion, err := copia.adquirirOperacion(context.Background()); conexion != nil ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatalf("copia admitida: (%v,%v)", conexion, err)
	}
	copia.Cerrar()
	if origen.cierres.Load() != 0 ||
		origen.adquisiciones.Load() != adquisiciones {
		t.Fatal("la copia adquirió o cerró recursos")
	}
	origen.configuracion.ConnConfig.User = "otro_login"
	if conexion, err := pool.adquirirOperacion(context.Background()); conexion != nil ||
		!errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
		t.Fatalf("configuración mutada admitida: (%v,%v)", conexion, err)
	}
	pool.Cerrar()
}

func TestFabricaPoolResolucionMotivosRRHHRechazaDSNLoginYTLS(
	t *testing.T,
) {
	t.Parallel()
	dsn := "postgres:///?host=/tmp/vec-m2-inexistente&port=5432&user=" +
		loginResolutorMotivosRRHHPrueba + "&sslmode=disable"
	for _, caso := range []struct {
		login string
		modo  modoTLSAcreditacionPoolO405
	}{
		{"otro_login", modoTLSAcreditacionPoolO405SocketUnixPrueba},
		{rolResolutorMotivosRRHHPostgreSQL, modoTLSAcreditacionPoolO405SocketUnixPrueba},
		{" " + loginResolutorMotivosRRHHPrueba + " ", modoTLSAcreditacionPoolO405SocketUnixPrueba},
		{loginResolutorMotivosRRHHPrueba, modoTLSAcreditacionPoolO405Produccion},
	} {
		if _, err := nuevoPoolResolucionMotivosRRHHPostgreSQL(
			context.Background(), dsn, caso.login, caso.modo,
		); !errors.Is(err, ports.ErrMotivoConsultaRRHHNoDisponible) {
			t.Fatalf("entrada insegura admitida: %v", err)
		}
	}
}

func TestLoginResolutorMotivosRRHHValido(t *testing.T) {
	t.Parallel()
	for _, login := range []string{
		"",
		rolResolutorMotivosRRHHPostgreSQL,
		" Vec",
		"vec-admin",
		"vec.ct",
	} {
		if loginResolutorMotivosRRHHValido(login) {
			t.Fatalf("LOGIN inseguro admitido: %q", login)
		}
	}
}

func TestIntegracionPoolResolucionMotivosRRHHPostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_POSTGRES_TEST_MOTIVOS_RRHH_RESOLUTOR_DSN")
	login := os.Getenv("VEC_POSTGRES_TEST_MOTIVOS_RRHH_RESOLUTOR_LOGIN")
	if dsn == "" || login == "" {
		t.Skip("integración M2.1 no configurada")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	pool, err := nuevoPoolResolucionMotivosRRHHPostgreSQL(
		ctx,
		dsn,
		login,
		modoTLSAcreditacionPoolO405SocketUnixPrueba,
	)
	if err != nil {
		conexionDiagnostico, errorConexion := pgx.Connect(ctx, dsn)
		if errorConexion != nil {
			t.Fatalf("acreditar pool real M2.1: %v", err)
		}
		defer conexionDiagnostico.Close(ctx)
		var sesion, efectivo string
		var oidCuadro, oidDetalle uint32
		var valores [11]bool
		errorDiagnostico := conexionDiagnostico.QueryRow(
			ctx,
			consultaAcreditacionPoolResolucionMotivosRRHH,
			login,
			false,
		).Scan(
			&sesion,
			&efectivo,
			&oidCuadro,
			&oidDetalle,
			&valores[0],
			&valores[1],
			&valores[2],
			&valores[3],
			&valores[4],
			&valores[5],
			&valores[6],
			&valores[7],
			&valores[8],
			&valores[9],
			&valores[10],
		)
		t.Fatalf(
			"acreditar pool real M2.1: %v; diagnóstico=%v identidad=%t "+
				"oid=(%d,%d) controles=%v",
			err,
			errorDiagnostico,
			sesion == login && efectivo == login,
			oidCuadro,
			oidDetalle,
			valores,
		)
	}
	if pool.oidCuadro == 0 || pool.oidDetalle == 0 ||
		pool.oidCuadro == pool.oidDetalle {
		t.Fatal("el pool real no conservó los dos OID nominales")
	}
	conexion, err := pool.adquirirOperacion(ctx)
	if err != nil {
		t.Fatalf("adquirir conexión real sellada: %v", err)
	}
	tx, err := conexion.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		liberarConexionResolucionMotivosRRHH(conexion)
		t.Fatalf("abrir transacción real: %v", err)
	}
	if err := pool.reacreditar(ctx, tx); err != nil {
		_ = tx.Rollback(ctx)
		liberarConexionResolucionMotivosRRHH(conexion)
		t.Fatalf("reacreditar en la misma transacción: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		liberarConexionResolucionMotivosRRHH(conexion)
		t.Fatalf("revertir transacción de prueba: %v", err)
	}
	if liberarConexionResolucionMotivosRRHH(conexion) {
		t.Fatal("falló la liberación de la conexión real")
	}
	pool.Cerrar()
}
