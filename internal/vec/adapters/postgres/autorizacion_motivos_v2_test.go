package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const catalogoMotivosAutorizacionV2PostgreSQLPrueba = "motivos_autorizacion_rrhh"

type filaMotivoAutorizacionV2PostgreSQLPrueba struct {
	resuelta bool
	err      error
}

func (f filaMotivoAutorizacionV2PostgreSQLPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != 1 {
		return errors.New("numero de destinos inesperado")
	}
	destino, correcto := destinos[0].(*bool)
	if !correcto || destino == nil {
		return errors.New("destino inesperado")
	}
	*destino = f.resuelta
	return nil
}

type consultorMotivoAutorizacionV2PostgreSQLPrueba struct {
	consultar func(context.Context, string, ...any) pgx.Row
	llamadas  atomic.Int32
}

func (c *consultorMotivoAutorizacionV2PostgreSQLPrueba) QueryRow(
	ctx context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	c.llamadas.Add(1)
	return c.consultar(ctx, consulta, argumentos...)
}

type consultorMotivoAutorizacionV2PostgreSQLNulo struct{}

func (*consultorMotivoAutorizacionV2PostgreSQLNulo) QueryRow(
	context.Context,
	string,
	...any,
) pgx.Row {
	panic("una dependencia nula tipada no debe consultarse")
}

type contextoMotivoAutorizacionV2PostgreSQLNulo struct{}

func (*contextoMotivoAutorizacionV2PostgreSQLNulo) Deadline() (time.Time, bool) {
	panic("un contexto nulo tipado no debe inspeccionarse")
}

func (*contextoMotivoAutorizacionV2PostgreSQLNulo) Done() <-chan struct{} {
	panic("un contexto nulo tipado no debe inspeccionarse")
}

func (*contextoMotivoAutorizacionV2PostgreSQLNulo) Err() error {
	panic("un contexto nulo tipado no debe inspeccionarse")
}

func (*contextoMotivoAutorizacionV2PostgreSQLNulo) Value(any) any {
	panic("un contexto nulo tipado no debe inspeccionarse")
}

func TestValidadorMotivoAutorizacionV2PostgreSQLConsultaHistoricaExacta(t *testing.T) {
	t.Parallel()
	referencia := referenciaMotivoAutorizacionV2PostgreSQLPrueba()
	instante := time.Date(2026, time.July, 17, 9, 8, 7, 654_321_000, time.UTC)
	ctx := context.WithValue(context.Background(), claveContextoMotivoPostgreSQLPrueba{}, "misma-instancia")

	consulta := &consultorMotivoAutorizacionV2PostgreSQLPrueba{
		consultar: func(
			recibido context.Context,
			sentencia string,
			argumentos ...any,
		) pgx.Row {
			if recibido != ctx {
				t.Fatal("se sustituyo el contexto recibido")
			}
			if sentencia != consultaResolverMotivoAutorizacionV2Historico {
				t.Fatalf("consulta distinta:\n%s", sentencia)
			}
			esperados := []any{
				referencia.CatalogoID,
				referencia.CatalogoVersion,
				referencia.CatalogoHuellaSHA256,
				referencia.EntradaClave,
				instante,
			}
			if !reflect.DeepEqual(argumentos, esperados) {
				t.Fatalf("argumentos: %#v; esperados: %#v", argumentos, esperados)
			}
			return filaMotivoAutorizacionV2PostgreSQLPrueba{resuelta: true}
		},
	}
	validador, err := nuevoValidadorReferenciaMotivoPostgreSQLV2(
		consulta,
		catalogoMotivosAutorizacionV2PostgreSQLPrueba,
	)
	if err != nil {
		t.Fatalf("crear validador: %v", err)
	}
	if err = validador.ValidarReferenciaMotivoAutorizacionV2(ctx, referencia, instante); err != nil {
		t.Fatalf("resolver referencia historica: %v", err)
	}
	if llamadas := consulta.llamadas.Load(); llamadas != 1 {
		t.Fatalf("consultas ejecutadas: %d", llamadas)
	}
	if strings.Contains(consultaResolverMotivoAutorizacionV2Historico, "_actual") {
		t.Fatal("el adaptador historico contiene una ruta al resolvedor actual")
	}
}

func TestValidadorMotivoAutorizacionV2PostgreSQLFallaCerradoAnteFalseYNoRows(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre          string
		resuelta        bool
		err             error
		falloDeLaFuente bool
	}{
		{nombre: "resultado_negativo"},
		{nombre: "sin_fila", err: pgx.ErrNoRows, falloDeLaFuente: true},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			consulta := &consultorMotivoAutorizacionV2PostgreSQLPrueba{
				consultar: func(context.Context, string, ...any) pgx.Row {
					return filaMotivoAutorizacionV2PostgreSQLPrueba{
						resuelta: caso.resuelta,
						err:      caso.err,
					}
				},
			}
			validador := nuevoValidadorMotivoAutorizacionV2PostgreSQLPrueba(t, consulta)
			err := validador.ValidarReferenciaMotivoAutorizacionV2(
				context.Background(),
				referenciaMotivoAutorizacionV2PostgreSQLPrueba(),
				instanteMotivoAutorizacionV2PostgreSQLPrueba(),
			)
			if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("fallo cerrado esperado: %v", err)
			}
			if obtenido := errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible); obtenido != caso.falloDeLaFuente {
				t.Fatalf("clasificacion de fuente: obtenida=%v esperada=%v err=%v", obtenido, caso.falloDeLaFuente, err)
			}
			if llamadas := consulta.llamadas.Load(); llamadas != 1 {
				t.Fatalf("consultas ejecutadas: %d", llamadas)
			}
		})
	}
}

func TestValidadorMotivoAutorizacionV2PostgreSQLNoFiltraErrores(t *testing.T) {
	t.Parallel()
	const detalle = "postgresql://usuario:secreto@servidor/vec SELECT dni_privado"
	casos := []struct {
		nombre string
		err    error
	}{
		{"error_sql", &pgconn.PgError{Code: "XX000", Message: detalle, Detail: "dni_privado"}},
		{"error_scan", errors.New(detalle)},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			consulta := &consultorMotivoAutorizacionV2PostgreSQLPrueba{
				consultar: func(context.Context, string, ...any) pgx.Row {
					return filaMotivoAutorizacionV2PostgreSQLPrueba{err: caso.err}
				},
			}
			validador := nuevoValidadorMotivoAutorizacionV2PostgreSQLPrueba(t, consulta)
			err := validador.ValidarReferenciaMotivoAutorizacionV2(
				context.Background(),
				referenciaMotivoAutorizacionV2PostgreSQLPrueba(),
				instanteMotivoAutorizacionV2PostgreSQLPrueba(),
			)
			if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("fallo cerrado esperado: %v", err)
			}
			if !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
				t.Fatalf("no se distinguio el fallo de infraestructura: %v", err)
			}
			for _, prohibido := range []string{"secreto", "SELECT", "dni_privado", "servidor"} {
				if strings.Contains(err.Error(), prohibido) {
					t.Fatalf("el error filtra %q: %q", prohibido, err)
				}
			}
		})
	}
}

func TestValidadorMotivoAutorizacionV2PostgreSQLConservaCancelacionInternaYMarcaFuente(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre   string
		causa    error
		esperado error
	}{
		{"cancelada", errors.Join(errors.New("postgresql://secreto"), context.Canceled), context.Canceled},
		{"plazo_agotado", errors.Join(errors.New("SELECT dato_privado"), context.DeadlineExceeded), context.DeadlineExceeded},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			consulta := &consultorMotivoAutorizacionV2PostgreSQLPrueba{
				consultar: func(context.Context, string, ...any) pgx.Row {
					return filaMotivoAutorizacionV2PostgreSQLPrueba{err: caso.causa}
				},
			}
			validador := nuevoValidadorMotivoAutorizacionV2PostgreSQLPrueba(t, consulta)
			err := validador.ValidarReferenciaMotivoAutorizacionV2(
				context.Background(),
				referenciaMotivoAutorizacionV2PostgreSQLPrueba(),
				instanteMotivoAutorizacionV2PostgreSQLPrueba(),
			)
			if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) ||
				!errors.Is(err, caso.esperado) {
				t.Fatalf("centinelas no conservados: %v", err)
			}
			if !errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
				t.Fatalf("el fallo interno no marco la fuente indisponible: %v", err)
			}
			if strings.Contains(err.Error(), "secreto") || strings.Contains(err.Error(), "dato_privado") {
				t.Fatalf("detalle filtrado: %q", err)
			}
		})
	}
}

func TestValidadorMotivoAutorizacionV2PostgreSQLRechazaContextoCanceladoAntesDeConsultar(t *testing.T) {
	t.Parallel()
	consulta := consultorMotivoAutorizacionV2PostgreSQLQueNoDebeEjecutarse(t)
	validador := nuevoValidadorMotivoAutorizacionV2PostgreSQLPrueba(t, consulta)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	err := validador.ValidarReferenciaMotivoAutorizacionV2(
		ctx,
		referenciaMotivoAutorizacionV2PostgreSQLPrueba(),
		instanteMotivoAutorizacionV2PostgreSQLPrueba(),
	)
	if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion no conservada: %v", err)
	}
	if errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("la cancelacion del llamador se atribuyo a la fuente: %v", err)
	}
	if llamadas := consulta.llamadas.Load(); llamadas != 0 {
		t.Fatalf("se consulto con contexto cancelado: %d", llamadas)
	}
}

func TestValidadorMotivoAutorizacionV2PostgreSQLRechazaPlazoAgotadoAntesDeConsultar(t *testing.T) {
	t.Parallel()
	consulta := consultorMotivoAutorizacionV2PostgreSQLQueNoDebeEjecutarse(t)
	validador := nuevoValidadorMotivoAutorizacionV2PostgreSQLPrueba(t, consulta)
	ctx, cancelar := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelar()
	err := validador.ValidarReferenciaMotivoAutorizacionV2(
		ctx,
		referenciaMotivoAutorizacionV2PostgreSQLPrueba(),
		instanteMotivoAutorizacionV2PostgreSQLPrueba(),
	)
	if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) ||
		!errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("plazo agotado no conservado: %v", err)
	}
	if errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("el plazo del llamador se atribuyo a la fuente: %v", err)
	}
	if llamadas := consulta.llamadas.Load(); llamadas != 0 {
		t.Fatalf("se consulto con plazo agotado: %d", llamadas)
	}
}

func TestValidadorMotivoAutorizacionV2PostgreSQLRechazaCancelacionDuranteLaConsulta(t *testing.T) {
	t.Parallel()
	ctx, cancelar := context.WithCancel(context.Background())
	consulta := &consultorMotivoAutorizacionV2PostgreSQLPrueba{
		consultar: func(context.Context, string, ...any) pgx.Row {
			cancelar()
			return filaMotivoAutorizacionV2PostgreSQLPrueba{resuelta: true}
		},
	}
	validador := nuevoValidadorMotivoAutorizacionV2PostgreSQLPrueba(t, consulta)
	err := validador.ValidarReferenciaMotivoAutorizacionV2(
		ctx,
		referenciaMotivoAutorizacionV2PostgreSQLPrueba(),
		instanteMotivoAutorizacionV2PostgreSQLPrueba(),
	)
	if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion durante consulta no conservada: %v", err)
	}
	if errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
		t.Fatalf("la cancelacion del llamador se atribuyo a la fuente: %v", err)
	}
	if llamadas := consulta.llamadas.Load(); llamadas != 1 {
		t.Fatalf("consultas ejecutadas: %d", llamadas)
	}
}

func TestValidadorMotivoAutorizacionV2PostgreSQLRechazaEntradasAntesDeConsultar(t *testing.T) {
	t.Parallel()
	referenciaValida := referenciaMotivoAutorizacionV2PostgreSQLPrueba()
	instanteValido := instanteMotivoAutorizacionV2PostgreSQLPrueba()
	var contextoNuloTipado *contextoMotivoAutorizacionV2PostgreSQLNulo
	casos := []struct {
		nombre     string
		ctx        context.Context
		referencia domain.ReferenciaEntradaCatalogo
		instante   time.Time
	}{
		{"contexto_nulo", nil, referenciaValida, instanteValido},
		{"contexto_nulo_tipado", contextoNuloTipado, referenciaValida, instanteValido},
		{"referencia_cero", context.Background(), domain.ReferenciaEntradaCatalogo{}, instanteValido},
		{"otro_catalogo", context.Background(), func() domain.ReferenciaEntradaCatalogo {
			r := referenciaValida
			r.CatalogoID = "otro_catalogo"
			return r
		}(), instanteValido},
		{"clave_no_opaca", context.Background(), func() domain.ReferenciaEntradaCatalogo {
			r := referenciaValida
			r.EntradaClave = "consulta_expediente_persona"
			return r
		}(), instanteValido},
		{"instante_cero", context.Background(), referenciaValida, time.Time{}},
		{"instante_no_utc", context.Background(), referenciaValida, instanteValido.In(time.FixedZone("UTC+1", 3600))},
		{"instante_submicrosegundo", context.Background(), referenciaValida, instanteValido.Add(time.Nanosecond)},
		{"anio_cero", context.Background(), referenciaValida, time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"anio_10000", context.Background(), referenciaValida, time.Date(10_000, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			consulta := consultorMotivoAutorizacionV2PostgreSQLQueNoDebeEjecutarse(t)
			validador := nuevoValidadorMotivoAutorizacionV2PostgreSQLPrueba(t, consulta)
			err := validador.ValidarReferenciaMotivoAutorizacionV2(
				caso.ctx,
				caso.referencia,
				caso.instante,
			)
			if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("entrada invalida aceptada: %v", err)
			}
			if errors.Is(err, ports.ErrFuenteAutorizacionNoDisponible) {
				t.Fatalf("una entrada invalida se clasifico como averia de fuente: %v", err)
			}
			if llamadas := consulta.llamadas.Load(); llamadas != 0 {
				t.Fatalf("consultas ejecutadas: %d", llamadas)
			}
		})
	}
}

func TestNuevoValidadorMotivoAutorizacionV2PostgreSQLRechazaDependenciaYCatalogoInvalidos(t *testing.T) {
	t.Parallel()
	if validador, err := NuevoValidadorReferenciaMotivoPostgreSQLV2(
		nil,
		catalogoMotivosAutorizacionV2PostgreSQLPrueba,
	); validador != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("pool nulo aceptado: validador=%v err=%v", validador, err)
	}
	var consultaNula *consultorMotivoAutorizacionV2PostgreSQLNulo
	if validador, err := nuevoValidadorReferenciaMotivoPostgreSQLV2(
		consultaNula,
		catalogoMotivosAutorizacionV2PostgreSQLPrueba,
	); validador != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
		t.Fatalf("dependencia nula tipada aceptada: validador=%v err=%v", validador, err)
	}
	consulta := consultorMotivoAutorizacionV2PostgreSQLQueNoDebeEjecutarse(t)
	for _, catalogoID := range []string{"", " catalogo", "catalogo con espacios", "catalogo\ninyectado"} {
		if validador, err := nuevoValidadorReferenciaMotivoPostgreSQLV2(
			consulta,
			catalogoID,
		); validador != nil || !errors.Is(err, domain.ErrConfiguracionAccesoInvalida) {
			t.Fatalf("catalogo invalido %q aceptado: validador=%v err=%v", catalogoID, validador, err)
		}
	}
}

func TestValidadorMotivoAutorizacionV2PostgreSQLRechazaReceptorODependenciaNulos(t *testing.T) {
	t.Parallel()
	var validador *ValidadorReferenciaMotivoPostgreSQLV2
	err := validador.ValidarReferenciaMotivoAutorizacionV2(
		context.Background(),
		referenciaMotivoAutorizacionV2PostgreSQLPrueba(),
		instanteMotivoAutorizacionV2PostgreSQLPrueba(),
	)
	if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("receptor nulo: %v", err)
	}
	var consultaNula *consultorMotivoAutorizacionV2PostgreSQLNulo
	validador = &ValidadorReferenciaMotivoPostgreSQLV2{
		consulta: consultaNula, catalogoID: catalogoMotivosAutorizacionV2PostgreSQLPrueba,
	}
	err = validador.ValidarReferenciaMotivoAutorizacionV2(
		context.Background(),
		referenciaMotivoAutorizacionV2PostgreSQLPrueba(),
		instanteMotivoAutorizacionV2PostgreSQLPrueba(),
	)
	if !errors.Is(err, domain.ErrSolicitudAutorizacionInvalida) {
		t.Fatalf("dependencia nula tipada: %v", err)
	}
}

func nuevoValidadorMotivoAutorizacionV2PostgreSQLPrueba(
	t *testing.T,
	consulta consultorFilaMotivoAutorizacionV2,
) *ValidadorReferenciaMotivoPostgreSQLV2 {
	t.Helper()
	validador, err := nuevoValidadorReferenciaMotivoPostgreSQLV2(
		consulta,
		catalogoMotivosAutorizacionV2PostgreSQLPrueba,
	)
	if err != nil {
		t.Fatalf("crear validador: %v", err)
	}
	return validador
}

func consultorMotivoAutorizacionV2PostgreSQLQueNoDebeEjecutarse(
	t *testing.T,
) *consultorMotivoAutorizacionV2PostgreSQLPrueba {
	t.Helper()
	return &consultorMotivoAutorizacionV2PostgreSQLPrueba{
		consultar: func(context.Context, string, ...any) pgx.Row {
			t.Error("se ejecuto una consulta inesperada")
			return filaMotivoAutorizacionV2PostgreSQLPrueba{err: errors.New("consulta inesperada")}
		},
	}
}

func referenciaMotivoAutorizacionV2PostgreSQLPrueba() domain.ReferenciaEntradaCatalogo {
	return domain.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogoMotivosAutorizacionV2PostgreSQLPrueba,
		CatalogoVersion:      17,
		CatalogoHuellaSHA256: strings.Repeat("7", 64),
		EntradaClave:         "motivo_11111111111111111111111111111111",
	}
}

func instanteMotivoAutorizacionV2PostgreSQLPrueba() time.Time {
	return time.Date(2026, time.July, 17, 10, 11, 12, 345_678_000, time.UTC)
}

type claveContextoMotivoPostgreSQLPrueba struct{}

var _ ports.ValidadorReferenciaMotivoAutorizacionV2 = (*ValidadorReferenciaMotivoPostgreSQLV2)(nil)
