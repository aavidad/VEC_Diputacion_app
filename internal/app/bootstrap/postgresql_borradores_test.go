package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
)

func TestNuevosPoolsPostgreSQLBorradoresAbreTresIdentidadesYCierraUnaVez(t *testing.T) {
	configuraciones := make([]*pgxpool.Config, 0, 3)
	poolsPrueba := []*poolPostgreSQLBorradoresPrueba{
		nuevoPoolPostgreSQLBorradoresPrueba("login-ejecutor"),
		nuevoPoolPostgreSQLBorradoresPrueba("login-proyector"),
		nuevoPoolPostgreSQLBorradoresPrueba("login-verificador"),
	}
	indice := 0
	crear := func(_ context.Context, configuracion *pgxpool.Config) (poolOperativoPostgreSQLBorradores, error) {
		configuraciones = append(configuraciones, configuracion)
		pool := poolsPrueba[indice]
		indice++
		return pool, nil
	}

	pools, err := nuevosPoolsPostgreSQLBorradores(
		context.Background(), configuracionPostgreSQLBorradoresPrueba(t), crear,
	)
	if err != nil {
		t.Fatalf("crear pools: %v", err)
	}
	if len(configuraciones) != 3 {
		t.Fatalf("pools creados = %d", len(configuraciones))
	}
	for indice, pool := range poolsPrueba {
		if pool.pings != 1 || !pool.pingConLimite || pool.consultas != 1 ||
			pool.rolConsultado != perfilesPoolPostgreSQLBorradores[indice].rolEsperado {
			t.Fatalf("pool %d no fue sondeado con su rol: %+v", indice, pool)
		}
	}
	pools.Close()
	pools.Close()
	for indice, pool := range poolsPrueba {
		if pool.cierres != 1 {
			t.Fatalf("pool %d cierres = %d", indice, pool.cierres)
		}
	}
}

func TestNuevosPoolsPostgreSQLBorradoresCierraTodoAnteFalloParcial(t *testing.T) {
	falloSecreto := errors.New("fallo con secreto-interno")
	casos := []struct {
		nombre           string
		preparar         func([]*poolPostgreSQLBorradoresPrueba)
		fallarCreacion   int
		poolConError     bool
		errorEsperado    error
		creadosEsperados int
	}{
		{
			nombre: "crear segundo", fallarCreacion: 1, poolConError: true,
			errorEsperado: ErrConexionPostgreSQLBorradoresNoDisponible, creadosEsperados: 2,
		},
		{
			nombre: "ping segundo", fallarCreacion: -1,
			preparar:      func(p []*poolPostgreSQLBorradoresPrueba) { p[1].pingErr = falloSecreto },
			errorEsperado: ErrConexionPostgreSQLBorradoresNoDisponible, creadosEsperados: 2,
		},
		{
			nombre: "after connect rechaza identidad", fallarCreacion: -1,
			preparar: func(p []*poolPostgreSQLBorradoresPrueba) {
				p[1].pingErr = errors.Join(ErrIdentidadPostgreSQLBorradoresInvalida, falloSecreto)
			},
			errorEsperado: ErrIdentidadPostgreSQLBorradoresInvalida, creadosEsperados: 2,
		},
		{
			nombre: "identidad segundo", fallarCreacion: -1,
			preparar:      func(p []*poolPostgreSQLBorradoresPrueba) { p[1].fila.err = falloSecreto },
			errorEsperado: ErrIdentidadPostgreSQLBorradoresInvalida, creadosEsperados: 2,
		},
		{
			nombre: "set role segundo", fallarCreacion: -1,
			preparar:      func(p []*poolPostgreSQLBorradoresPrueba) { p[1].fila.usuarioEfectivo = "rol-elevado" },
			errorEsperado: ErrIdentidadPostgreSQLBorradoresInvalida, creadosEsperados: 2,
		},
		{
			nombre: "usuario repetido tercero", fallarCreacion: -1,
			preparar: func(p []*poolPostgreSQLBorradoresPrueba) {
				p[2].fila.usuarioSesion = p[0].fila.usuarioSesion
				p[2].fila.usuarioEfectivo = p[0].fila.usuarioEfectivo
			},
			errorEsperado: ErrIdentidadPostgreSQLBorradoresInvalida, creadosEsperados: 3,
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			poolsPrueba := []*poolPostgreSQLBorradoresPrueba{
				nuevoPoolPostgreSQLBorradoresPrueba("login-ejecutor"),
				nuevoPoolPostgreSQLBorradoresPrueba("login-proyector"),
				nuevoPoolPostgreSQLBorradoresPrueba("login-verificador"),
			}
			if caso.preparar != nil {
				caso.preparar(poolsPrueba)
			}
			creados := 0
			crear := func(_ context.Context, _ *pgxpool.Config) (poolOperativoPostgreSQLBorradores, error) {
				pool := poolsPrueba[creados]
				indice := creados
				creados++
				if indice == caso.fallarCreacion {
					if caso.poolConError {
						return pool, falloSecreto
					}
					return nil, falloSecreto
				}
				return pool, nil
			}
			resultado, err := nuevosPoolsPostgreSQLBorradores(
				context.Background(), configuracionPostgreSQLBorradoresPrueba(t), crear,
			)
			if resultado != nil || !errors.Is(err, caso.errorEsperado) {
				t.Fatalf("resultado=%v error=%v", resultado, err)
			}
			if strings.Contains(err.Error(), "secreto") {
				t.Fatalf("detalle interno filtrado: %v", err)
			}
			if creados != caso.creadosEsperados {
				t.Fatalf("creados=%d, esperado=%d", creados, caso.creadosEsperados)
			}
			for indice, pool := range poolsPrueba {
				esperado := 0
				if indice < creados {
					esperado = 1
				}
				if pool.cierres != esperado {
					t.Fatalf("pool %d cierres=%d, esperado=%d", indice, pool.cierres, esperado)
				}
			}
		})
	}
}

func TestNuevosPoolsPostgreSQLBorradoresRechazaConfiguracionIncompletaAntesDeAbrir(t *testing.T) {
	creaciones := 0
	crear := func(context.Context, *pgxpool.Config) (poolOperativoPostgreSQLBorradores, error) {
		creaciones++
		return nil, nil
	}
	_, err := nuevosPoolsPostgreSQLBorradores(context.Background(), config.Config{}, crear)
	if !errors.Is(err, config.ErrConfiguracionPostgreSQLBorradoresIncompleta) || creaciones != 0 {
		t.Fatalf("error=%v creaciones=%d", err, creaciones)
	}
}

func TestNuevosPoolsPostgreSQLBorradoresRechazaPoolTipadoNuloSinInvocarlo(t *testing.T) {
	var poolNulo *poolPostgreSQLBorradoresPrueba
	crear := func(context.Context, *pgxpool.Config) (poolOperativoPostgreSQLBorradores, error) {
		return poolNulo, nil
	}
	resultado, err := nuevosPoolsPostgreSQLBorradores(
		context.Background(), configuracionPostgreSQLBorradoresPrueba(t), crear,
	)
	if resultado != nil || !errors.Is(err, ErrConexionPostgreSQLBorradoresNoDisponible) {
		t.Fatalf("resultado=%v error=%v", resultado, err)
	}
}

func TestNuevosPoolsPostgreSQLBorradoresNoAbreConContextoCancelado(t *testing.T) {
	creaciones := 0
	crear := func(context.Context, *pgxpool.Config) (poolOperativoPostgreSQLBorradores, error) {
		creaciones++
		return nuevoPoolPostgreSQLBorradoresPrueba("login-ejecutor"), nil
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	resultado, err := nuevosPoolsPostgreSQLBorradores(
		ctx, configuracionPostgreSQLBorradoresPrueba(t), crear,
	)
	if resultado != nil || !errors.Is(err, context.Canceled) ||
		!errors.Is(err, ErrConexionPostgreSQLBorradoresNoDisponible) || creaciones != 0 {
		t.Fatalf("resultado=%v error=%v creaciones=%d", resultado, err, creaciones)
	}
}

func configuracionPostgreSQLBorradoresPrueba(t *testing.T) config.Config {
	t.Helper()
	configuracion, err := config.NuevaConfiguracionPostgreSQLBorradores(
		"postgres://ejecutor:secreto-ejecutor@127.0.0.1:5432/vec?sslmode=disable",
		"postgres://proyector:secreto-proyector@127.0.0.1:5432/vec?sslmode=disable",
		"postgres://verificador:secreto-verificador@127.0.0.1:5432/vec?sslmode=disable",
	)
	if err != nil {
		t.Fatal(err)
	}
	return config.Config{BolsaBorradoresPostgreSQL: configuracion}
}

func nuevoPoolPostgreSQLBorradoresPrueba(usuario string) *poolPostgreSQLBorradoresPrueba {
	return &poolPostgreSQLBorradoresPrueba{
		fila: filaIdentidadPostgreSQLBorradoresPrueba{
			usuarioSesion: usuario, usuarioEfectivo: usuario, valida: true,
		},
	}
}

type poolPostgreSQLBorradoresPrueba struct {
	fila          filaIdentidadPostgreSQLBorradoresPrueba
	pingErr       error
	pings         int
	pingConLimite bool
	consultas     int
	rolConsultado string
	cierres       int
}

func (p *poolPostgreSQLBorradoresPrueba) Ping(ctx context.Context) error {
	p.pings++
	_, p.pingConLimite = ctx.Deadline()
	return p.pingErr
}

func (p *poolPostgreSQLBorradoresPrueba) QueryRow(
	_ context.Context,
	_ string,
	argumentos ...any,
) pgx.Row {
	p.consultas++
	if len(argumentos) == 1 {
		p.rolConsultado, _ = argumentos[0].(string)
	}
	return p.fila
}

func (p *poolPostgreSQLBorradoresPrueba) Close() { p.cierres++ }
