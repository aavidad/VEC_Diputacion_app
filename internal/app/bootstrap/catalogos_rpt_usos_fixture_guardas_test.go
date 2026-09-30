package bootstrap

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
)

func admisionRPTUsosFixturePrueba() (admisionRPTUsosFixture, config.Config, [3]*pgxpool.Config) {
	a := admisionRPTUsosFixture{
		activado: true, host: "127.0.0.1", puerto: 55531, base: "fixture_clon",
		huellaClonEsperada: strings.Repeat("a", 64), huellaClonReady: strings.Repeat("a", 64),
		readyH6: "ensayo:h6:fixture", readyAD132: "ensayo:ad132:fixture", readySeisRPT: "ensayo:rpt:seis",
		aprobacionRef: "aprobacion:rpt:fixture", preimagenAsignacionHuellaSHA256: strings.Repeat("b", 64),
	}
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment,
		AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	var pools [3]*pgxpool.Config
	for i, usuario := range []string{"fixture_gobierno", "fixture_registro", "fixture_ejecutor"} {
		pools[i] = &pgxpool.Config{ConnConfig: &pgx.ConnConfig{Config: pgconn.Config{
			Host: a.host, Port: a.puerto, Database: a.base, User: usuario, RuntimeParams: map[string]string{},
		}}}
	}
	return a, cfg, pools
}

func TestRPTUsosFixtureAdmisionDestinoPrivado(t *testing.T) {
	a, cfg, pools := admisionRPTUsosFixturePrueba()
	if err := a.validar(cfg, pools); err != nil {
		t.Fatal(err)
	}
	// Un puerto local distinto se admite sólo si el destino privado y todos
	// los pools lo declaran. No se fija el puerto del MCP de consultas.
	a.puerto = 55532
	for _, pool := range pools {
		pool.ConnConfig.Port = a.puerto
	}
	if err := a.validar(cfg, pools); err != nil {
		t.Fatal(err)
	}
}

func TestRPTUsosFixtureDeniegaAntesConexion(t *testing.T) {
	casos := map[string]func(*admisionRPTUsosFixture, *config.Config, [3]*pgxpool.Config){
		"sin_opt_in": func(a *admisionRPTUsosFixture, _ *config.Config, _ [3]*pgxpool.Config) { a.activado = false },
		"produccion": func(_ *admisionRPTUsosFixture, c *config.Config, _ [3]*pgxpool.Config) {
			c.ExecutionProfile = config.ExecutionProfileProduction
		},
		"sin_doble_llave": func(_ *admisionRPTUsosFixture, c *config.Config, _ [3]*pgxpool.Config) { c.DevelopmentGuard = "" },
		"otro_clon": func(a *admisionRPTUsosFixture, _ *config.Config, _ [3]*pgxpool.Config) {
			a.huellaClonReady = strings.Repeat("c", 64)
		},
		"h6_pendiente":    func(a *admisionRPTUsosFixture, _ *config.Config, _ [3]*pgxpool.Config) { a.readyH6 = "" },
		"ad132_pendiente": func(a *admisionRPTUsosFixture, _ *config.Config, _ [3]*pgxpool.Config) { a.readyAD132 = "" },
		"rpt_pendiente":   func(a *admisionRPTUsosFixture, _ *config.Config, _ [3]*pgxpool.Config) { a.readySeisRPT = "" },
		"sin_aprobacion":  func(a *admisionRPTUsosFixture, _ *config.Config, _ [3]*pgxpool.Config) { a.aprobacionRef = "" },
		"sin_huella": func(a *admisionRPTUsosFixture, _ *config.Config, _ [3]*pgxpool.Config) {
			a.preimagenAsignacionHuellaSHA256 = ""
		},
		"remoto": func(_ *admisionRPTUsosFixture, _ *config.Config, p [3]*pgxpool.Config) {
			p[2].ConnConfig.Host = "192.0.2.1"
		},
		"otra_base": func(_ *admisionRPTUsosFixture, _ *config.Config, p [3]*pgxpool.Config) {
			p[2].ConnConfig.Database = "otra"
		},
		"otro_puerto": func(_ *admisionRPTUsosFixture, _ *config.Config, p [3]*pgxpool.Config) { p[2].ConnConfig.Port++ },
		"usuario_compartido": func(_ *admisionRPTUsosFixture, _ *config.Config, p [3]*pgxpool.Config) {
			p[2].ConnConfig.User = p[0].ConnConfig.User
		},
		"role_por_dsn": func(_ *admisionRPTUsosFixture, _ *config.Config, p [3]*pgxpool.Config) {
			p[2].ConnConfig.RuntimeParams["role"] = "propietario"
		},
		"fallback_remoto": func(_ *admisionRPTUsosFixture, _ *config.Config, p [3]*pgxpool.Config) {
			p[2].ConnConfig.Fallbacks = []*pgconn.FallbackConfig{{Host: "192.0.2.1", Port: 55531}}
		},
		"pool_ausente": func(_ *admisionRPTUsosFixture, _ *config.Config, p [3]*pgxpool.Config) { p[2].ConnConfig = nil },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			a, cfg, pools := admisionRPTUsosFixturePrueba()
			cambiar(&a, &cfg, pools)
			if err := a.validar(cfg, pools); !errors.Is(err, ErrSeguridadComunDesarrolloDenegada) {
				t.Fatalf("admisión inesperada: %v", err)
			}
		})
	}
}
