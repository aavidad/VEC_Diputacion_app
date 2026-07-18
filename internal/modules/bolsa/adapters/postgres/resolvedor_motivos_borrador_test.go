package postgres

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const catalogoMotivosBorradorPostgreSQLPrueba = "motivos_autorizacion_rrhh"

type validadorMotivoBorradorPostgreSQLPrueba struct {
	validar  func(context.Context, dominiovec.ReferenciaEntradaCatalogo, time.Time) error
	llamadas atomic.Int64
}

func (v *validadorMotivoBorradorPostgreSQLPrueba) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context,
	referencia dominiovec.ReferenciaEntradaCatalogo,
	instante time.Time,
) error {
	v.llamadas.Add(1)
	return v.validar(ctx, referencia, instante)
}

type validadorMotivoBorradorPostgreSQLNulo struct{}

func (*validadorMotivoBorradorPostgreSQLNulo) ValidarReferenciaMotivoAutorizacionV2(
	context.Context,
	dominiovec.ReferenciaEntradaCatalogo,
	time.Time,
) error {
	panic("no debe ejecutarse")
}

type contextoMotivoBorradorPostgreSQLNulo struct{}

func (*contextoMotivoBorradorPostgreSQLNulo) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*contextoMotivoBorradorPostgreSQLNulo) Done() <-chan struct{}       { return nil }
func (*contextoMotivoBorradorPostgreSQLNulo) Err() error                  { return nil }
func (*contextoMotivoBorradorPostgreSQLNulo) Value(any) any               { return nil }

func TestResolvedorMotivoBorradorPostgreSQLConservaReferenciaExacta(t *testing.T) {
	t.Parallel()
	referencia := referenciaMotivoBorradorPostgreSQLPrueba()
	instante := instanteMotivoBorradorPostgreSQLPrueba()
	validador := &validadorMotivoBorradorPostgreSQLPrueba{
		validar: func(
			ctx context.Context,
			recibida dominiovec.ReferenciaEntradaCatalogo,
			recibidoEn time.Time,
		) error {
			if ctx == nil || recibida != referencia || !recibidoEn.Equal(instante) ||
				recibidoEn.Location() != time.UTC {
				t.Fatalf("consulta historica alterada: referencia=%#v instante=%v", recibida, recibidoEn)
			}
			return nil
		},
	}
	resolvedor := nuevoResolvedorMotivoBorradorPostgreSQLPrueba(t, validador)

	resuelta, err := resolvedor.ResolverMotivoBorrador(
		context.Background(), referencia, instante,
	)
	if err != nil {
		t.Fatalf("resolver motivo exacto: %v", err)
	}
	if resuelta != referencia {
		t.Fatalf("referencia sustituida: resuelta=%#v solicitada=%#v", resuelta, referencia)
	}
	if llamadas := validador.llamadas.Load(); llamadas != 1 {
		t.Fatalf("validaciones ejecutadas: %d", llamadas)
	}
}

func TestResolvedorMotivoBorradorPostgreSQLRechazaNoVigenciaYNoCoincidencia(t *testing.T) {
	t.Parallel()
	referencia := referenciaMotivoBorradorPostgreSQLPrueba()
	catalogoDistinto := referencia
	catalogoDistinto.CatalogoID = "motivos_autorizacion_ajenos"
	versionDistinta := referencia
	versionDistinta.CatalogoVersion++
	huellaDistinta := referencia
	huellaDistinta.CatalogoHuellaSHA256 = strings.Repeat("b", 64)
	entradaDistinta := referencia
	entradaDistinta.EntradaClave = "motivo_fedcba9876543210fedcba9876543210"
	for _, caso := range []struct {
		nombre     string
		referencia dominiovec.ReferenciaEntradaCatalogo
	}{
		{"catalogo_no_publicado_en_instante", referencia},
		{"entrada_no_vigente_en_instante", referencia},
		{"catalogo_retirado_en_instante", referencia},
		{"catalogo_no_coincide", catalogoDistinto},
		{"version_no_coincide", versionDistinta},
		{"huella_no_coincide", huellaDistinta},
		{"entrada_no_coincide", entradaDistinta},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			validador := &validadorMotivoBorradorPostgreSQLPrueba{
				validar: func(
					_ context.Context,
					recibida dominiovec.ReferenciaEntradaCatalogo,
					recibidoEn time.Time,
				) error {
					if recibida != caso.referencia || !recibidoEn.Equal(instanteMotivoBorradorPostgreSQLPrueba()) {
						t.Fatalf("consulta alterada: referencia=%#v instante=%v", recibida, recibidoEn)
					}
					return dominiovec.ErrSolicitudAutorizacionInvalida
				},
			}
			resolvedor := nuevoResolvedorMotivoBorradorPostgreSQLPrueba(t, validador)
			resuelta, err := resolvedor.ResolverMotivoBorrador(
				context.Background(),
				caso.referencia,
				instanteMotivoBorradorPostgreSQLPrueba(),
			)
			if resuelta != (dominiovec.ReferenciaEntradaCatalogo{}) ||
				!errors.Is(err, gobiernoconvocatorias.ErrOrdenBorradorInvalida) ||
				!errors.Is(err, dominiovec.ErrSolicitudAutorizacionInvalida) {
				t.Fatalf("fallo cerrado no conservado: resuelta=%#v err=%v", resuelta, err)
			}
			if llamadas := validador.llamadas.Load(); llamadas != 1 {
				t.Fatalf("validaciones ejecutadas: %d", llamadas)
			}
		})
	}
}

func TestResolvedorMotivoBorradorPostgreSQLRechazaEntradaSinConsultar(t *testing.T) {
	t.Parallel()
	referenciaValida := referenciaMotivoBorradorPostgreSQLPrueba()
	instanteValido := instanteMotivoBorradorPostgreSQLPrueba()
	contextoCancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	var contextoNulo *contextoMotivoBorradorPostgreSQLNulo
	versionCambiada := referenciaValida
	versionCambiada.CatalogoVersion = 0
	huellaCambiada := referenciaValida
	huellaCambiada.CatalogoHuellaSHA256 = strings.Repeat("0", 64)
	entradaCambiada := referenciaValida
	entradaCambiada.EntradaClave = "motivo_humano"

	casos := []struct {
		nombre     string
		ctx        context.Context
		referencia dominiovec.ReferenciaEntradaCatalogo
		instante   time.Time
		centinela  error
	}{
		{"contexto_nulo", nil, referenciaValida, instanteValido, nil},
		{"contexto_nulo_tipado", contextoNulo, referenciaValida, instanteValido, nil},
		{"contexto_cancelado", contextoCancelado, referenciaValida, instanteValido, context.Canceled},
		{"version_invalida", context.Background(), versionCambiada, instanteValido, nil},
		{"huella_nula", context.Background(), huellaCambiada, instanteValido, nil},
		{"entrada_no_opaca", context.Background(), entradaCambiada, instanteValido, nil},
		{"instante_cero", context.Background(), referenciaValida, time.Time{}, nil},
		{"instante_no_utc", context.Background(), referenciaValida, instanteValido.In(time.FixedZone("UTC+1", 3600)), nil},
		{"instante_submicrosegundo", context.Background(), referenciaValida, instanteValido.Add(time.Nanosecond), nil},
		{"instante_anio_cero", context.Background(), referenciaValida, time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC), nil},
		{"instante_anio_10000", context.Background(), referenciaValida, time.Date(10_000, 1, 1, 0, 0, 0, 0, time.UTC), nil},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			validador := &validadorMotivoBorradorPostgreSQLPrueba{
				validar: func(
					context.Context,
					dominiovec.ReferenciaEntradaCatalogo,
					time.Time,
				) error {
					t.Fatal("una entrada invalida alcanzo PostgreSQL")
					return nil
				},
			}
			resolvedor := nuevoResolvedorMotivoBorradorPostgreSQLPrueba(t, validador)
			resuelta, err := resolvedor.ResolverMotivoBorrador(
				caso.ctx, caso.referencia, caso.instante,
			)
			if resuelta != (dominiovec.ReferenciaEntradaCatalogo{}) ||
				!errors.Is(err, gobiernoconvocatorias.ErrOrdenBorradorInvalida) {
				t.Fatalf("entrada invalida aceptada: resuelta=%#v err=%v", resuelta, err)
			}
			if caso.centinela != nil && !errors.Is(err, caso.centinela) {
				t.Fatalf("centinela de contexto perdido: %v", err)
			}
			if llamadas := validador.llamadas.Load(); llamadas != 0 {
				t.Fatalf("validaciones ejecutadas: %d", llamadas)
			}
		})
	}
}

func TestResolvedorMotivoBorradorPostgreSQLSaneaErroresDeFuente(t *testing.T) {
	t.Parallel()
	const secreto = "postgresql://rrhh:secreto@servidor/vec SELECT dni_privado"
	for _, caso := range []struct {
		nombre   string
		causa    error
		fuente   bool
		contexto error
	}{
		{"error_desconocido", errors.New(secreto), false, nil},
		{"fuente_no_disponible", errors.Join(puertosvec.ErrFuenteAutorizacionNoDisponible, errors.New(secreto)), true, nil},
		{"cancelacion_de_fuente", errors.Join(puertosvec.ErrFuenteAutorizacionNoDisponible, context.Canceled, errors.New(secreto)), true, context.Canceled},
		{"plazo_de_fuente", errors.Join(puertosvec.ErrFuenteAutorizacionNoDisponible, context.DeadlineExceeded, errors.New(secreto)), true, context.DeadlineExceeded},
	} {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			validador := &validadorMotivoBorradorPostgreSQLPrueba{
				validar: func(
					context.Context,
					dominiovec.ReferenciaEntradaCatalogo,
					time.Time,
				) error {
					return caso.causa
				},
			}
			resolvedor := nuevoResolvedorMotivoBorradorPostgreSQLPrueba(t, validador)
			resuelta, err := resolvedor.ResolverMotivoBorrador(
				context.Background(),
				referenciaMotivoBorradorPostgreSQLPrueba(),
				instanteMotivoBorradorPostgreSQLPrueba(),
			)
			if resuelta != (dominiovec.ReferenciaEntradaCatalogo{}) ||
				!errors.Is(err, gobiernoconvocatorias.ErrOrdenBorradorInvalida) {
				t.Fatalf("fallo cerrado no conservado: resuelta=%#v err=%v", resuelta, err)
			}
			if obtenido := errors.Is(err, puertosvec.ErrFuenteAutorizacionNoDisponible); obtenido != caso.fuente {
				t.Fatalf("clasificacion de fuente: obtenida=%v esperada=%v err=%v", obtenido, caso.fuente, err)
			}
			if caso.contexto != nil && !errors.Is(err, caso.contexto) {
				t.Fatalf("centinela de contexto perdido: %v", err)
			}
			for _, prohibido := range []string{"secreto", "SELECT", "dni_privado", "servidor"} {
				if strings.Contains(err.Error(), prohibido) {
					t.Fatalf("el error filtra %q: %q", prohibido, err)
				}
			}
		})
	}
}

func TestResolvedorMotivoBorradorPostgreSQLCompruebaCancelacionPosterior(t *testing.T) {
	t.Parallel()
	ctx, cancelar := context.WithCancel(context.Background())
	validador := &validadorMotivoBorradorPostgreSQLPrueba{
		validar: func(
			context.Context,
			dominiovec.ReferenciaEntradaCatalogo,
			time.Time,
		) error {
			cancelar()
			return nil
		},
	}
	resolvedor := nuevoResolvedorMotivoBorradorPostgreSQLPrueba(t, validador)
	resuelta, err := resolvedor.ResolverMotivoBorrador(
		ctx,
		referenciaMotivoBorradorPostgreSQLPrueba(),
		instanteMotivoBorradorPostgreSQLPrueba(),
	)
	if resuelta != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, gobiernoconvocatorias.ErrOrdenBorradorInvalida) ||
		!errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion posterior aceptada: resuelta=%#v err=%v", resuelta, err)
	}
}

func TestNuevoResolvedorMotivoBorradorPostgreSQLRechazaNulos(t *testing.T) {
	t.Parallel()
	if resolvedor, err := NuevoResolvedorMotivoBorradorPostgreSQL(nil); resolvedor != nil ||
		!errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
		t.Fatalf("validador PostgreSQL nulo aceptado: resolvedor=%v err=%v", resolvedor, err)
	}
	var validadorNulo *validadorMotivoBorradorPostgreSQLNulo
	if resolvedor, err := nuevoResolvedorMotivoBorradorPostgreSQL(validadorNulo); resolvedor != nil ||
		!errors.Is(err, gobiernoconvocatorias.ErrServicioBorradoresInvalido) {
		t.Fatalf("validador nulo tipado aceptado: resolvedor=%v err=%v", resolvedor, err)
	}

	var receptor *ResolvedorMotivoBorradorPostgreSQL
	if resuelta, err := receptor.ResolverMotivoBorrador(
		context.Background(),
		referenciaMotivoBorradorPostgreSQLPrueba(),
		instanteMotivoBorradorPostgreSQLPrueba(),
	); resuelta != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, gobiernoconvocatorias.ErrOrdenBorradorInvalida) {
		t.Fatalf("receptor nulo aceptado: resuelta=%#v err=%v", resuelta, err)
	}

	receptor = &ResolvedorMotivoBorradorPostgreSQL{validador: validadorNulo}
	if resuelta, err := receptor.ResolverMotivoBorrador(
		context.Background(),
		referenciaMotivoBorradorPostgreSQLPrueba(),
		instanteMotivoBorradorPostgreSQLPrueba(),
	); resuelta != (dominiovec.ReferenciaEntradaCatalogo{}) ||
		!errors.Is(err, gobiernoconvocatorias.ErrOrdenBorradorInvalida) {
		t.Fatalf("dependencia nula tipada aceptada: resuelta=%#v err=%v", resuelta, err)
	}
}

func nuevoResolvedorMotivoBorradorPostgreSQLPrueba(
	t *testing.T,
	validador *validadorMotivoBorradorPostgreSQLPrueba,
) *ResolvedorMotivoBorradorPostgreSQL {
	t.Helper()
	resolvedor, err := nuevoResolvedorMotivoBorradorPostgreSQL(validador)
	if err != nil {
		t.Fatalf("crear resolvedor: %v", err)
	}
	return resolvedor
}

func referenciaMotivoBorradorPostgreSQLPrueba() dominiovec.ReferenciaEntradaCatalogo {
	return dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogoMotivosBorradorPostgreSQLPrueba,
		CatalogoVersion:      7,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
}

func instanteMotivoBorradorPostgreSQLPrueba() time.Time {
	return time.Date(2026, time.July, 18, 9, 30, 0, 123_456_000, time.UTC)
}
