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

func TestPoolsPostgreSQLDietasDesarrolloAbreDosIdentidadesYCierra(t *testing.T) {
	poolsPrueba := make([]*poolDietasPostgreSQLPrueba, 2)
	for i := range poolsPrueba {
		poolsPrueba[i] = nuevoPoolDietasPostgreSQLPrueba("login-dietas-" + string(rune('a'+i)))
	}
	creados := 0
	pools, err := nuevosPoolsPostgreSQLDietasDesarrolloConFabrica(context.Background(), configuracionDietasPostgreSQLPrueba(t), func(_ context.Context, _ *pgxpool.Config) (poolOperativoPostgreSQLDietasDesarrollo, error) {
		p := poolsPrueba[creados]
		creados++
		return p, nil
	})
	if err != nil || creados != 2 {
		t.Fatalf("pools=%v err=%v creados=%d", pools, err, creados)
	}
	for i, p := range poolsPrueba {
		if p.pings != 1 || p.consultas != 1 || p.rol != perfilesPoolPostgreSQLDietasDesarrollo[i].rol {
			t.Fatalf("pool %d: %+v", i, p)
		}
	}
	pools.Cerrar()
	pools.Cerrar()
	for i, p := range poolsPrueba {
		if p.cierres != 1 {
			t.Fatalf("cierres pool %d = %d", i, p.cierres)
		}
	}
}

func TestPoolsPostgreSQLDietasDesarrolloFallaCerradoYNoFiltraDSN(t *testing.T) {
	secreto := errors.New("secreto-no-exponible")
	poolsPrueba := make([]*poolDietasPostgreSQLPrueba, 2)
	for i := range poolsPrueba {
		poolsPrueba[i] = nuevoPoolDietasPostgreSQLPrueba("login-" + string(rune('a'+i)))
	}
	poolsPrueba[1].err = secreto
	creados := 0
	resultado, err := nuevosPoolsPostgreSQLDietasDesarrolloConFabrica(context.Background(), configuracionDietasPostgreSQLPrueba(t), func(_ context.Context, _ *pgxpool.Config) (poolOperativoPostgreSQLDietasDesarrollo, error) {
		p := poolsPrueba[creados]
		creados++
		return p, nil
	})
	if resultado != nil || !errors.Is(err, errConexionPostgreSQLDietasDesarrolloNoDisponible) || strings.Contains(err.Error(), "secreto") {
		t.Fatalf("resultado=%v err=%v", resultado, err)
	}
	for i, p := range poolsPrueba {
		want := 0
		if i < 2 {
			want = 1
		}
		if p.cierres != want {
			t.Fatalf("cierres %d=%d", i, p.cierres)
		}
	}
}

func TestAcreditarPoolPostgreSQLDietasDesarrolloRechazaSetRoleTopologiaYLoginRepetido(t *testing.T) {
	p := nuevoPoolDietasPostgreSQLPrueba("login-a")
	p.fila.efectivo = "rol-ajeno"
	if _, _, err := acreditarPoolPostgreSQLDietasDesarrollo(context.Background(), p, rolDietasBorradoresPostgreSQLDesarrollo); !errors.Is(err, errIdentidadPostgreSQLDietasDesarrolloInvalida) {
		t.Fatalf("err=%v", err)
	}
	if !strings.Contains(consultaAcreditacionPoolPostgreSQLDietasDesarrollo, "NOT m.set_option") || strings.Contains(consultaAcreditacionPoolPostgreSQLDietasDesarrollo, "pg_control_system") {
		t.Fatal("sonda no conserva frontera de roles")
	}
}

func TestPoolsPostgreSQLDietasDesarrolloRechazaTopologiaDistintaYConfiguraLimites(t *testing.T) {
	poolsPrueba := make([]*poolDietasPostgreSQLPrueba, 2)
	for i := range poolsPrueba {
		poolsPrueba[i] = nuevoPoolDietasPostgreSQLPrueba("login-" + string(rune('a'+i)))
	}
	poolsPrueba[1].fila.base = "otra_base"
	creados := 0
	resultado, err := nuevosPoolsPostgreSQLDietasDesarrolloConFabrica(context.Background(), configuracionDietasPostgreSQLPrueba(t), func(_ context.Context, cfg *pgxpool.Config) (poolOperativoPostgreSQLDietasDesarrollo, error) {
		if cfg.MaxConns < 1 || cfg.ConnConfig.RuntimeParams["search_path"] != "pg_catalog,pg_temp" || cfg.ConnConfig.RuntimeParams["statement_timeout"] != "15s" || cfg.AfterConnect == nil {
			t.Fatal("configuracion de pool no conservadora")
		}
		p := poolsPrueba[creados]
		creados++
		return p, nil
	})
	if resultado != nil || !errors.Is(err, errIdentidadPostgreSQLDietasDesarrolloInvalida) || creados != 2 {
		t.Fatalf("resultado=%v err=%v creados=%d", resultado, err, creados)
	}
	for i, p := range poolsPrueba {
		want := 0
		if i < 2 {
			want = 1
		}
		if p.cierres != want {
			t.Fatalf("cierres %d=%d", i, p.cierres)
		}
	}
}

func configuracionDietasPostgreSQLPrueba(t *testing.T) config.Config {
	t.Helper()
	c, err := config.NuevaConfiguracionDietasBorradores(
		"postgres://d:secret@dietas.example/vec?sslmode=verify-full", "postgres://p:secret@personal.example/vec?sslmode=verify-full")
	if err != nil {
		t.Fatal(err)
	}
	return config.Config{DietasBorradoresPostgreSQL: c}
}

type filaDietasPostgreSQLPrueba struct {
	usuario, efectivo, base, direccion, puerto, inicio string
	valida                                             bool
	err                                                error
}

func (f filaDietasPostgreSQLPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*string), *destinos[1].(*string), *destinos[2].(*string), *destinos[3].(*string), *destinos[4].(*string), *destinos[5].(*string), *destinos[6].(*bool) = f.usuario, f.efectivo, f.base, f.direccion, f.puerto, f.inicio, f.valida
	return nil
}

type poolDietasPostgreSQLPrueba struct {
	fila                      filaDietasPostgreSQLPrueba
	err                       error
	pings, consultas, cierres int
	rol                       string
}

func nuevoPoolDietasPostgreSQLPrueba(usuario string) *poolDietasPostgreSQLPrueba {
	return &poolDietasPostgreSQLPrueba{fila: filaDietasPostgreSQLPrueba{usuario: usuario, efectivo: usuario, base: "vec", direccion: "127.0.0.1", puerto: "5432", inicio: "2026-09-20", valida: true}}
}
func (p *poolDietasPostgreSQLPrueba) Ping(context.Context) error { p.pings++; return p.err }
func (p *poolDietasPostgreSQLPrueba) QueryRow(_ context.Context, _ string, argumentos ...any) pgx.Row {
	p.consultas++
	p.rol, _ = argumentos[0].(string)
	return p.fila
}
func (p *poolDietasPostgreSQLPrueba) Close() { p.cierres++ }
