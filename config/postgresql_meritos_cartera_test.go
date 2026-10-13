package config

import (
	"errors"
	"strings"
	"testing"
)

func TestMeritosCarteraConexionesSeparadasYFalloSoloDelModulo(t *testing.T) {
	const datos = "postgres://vec_externo_meritos_cartera@localhost:5432/vec?sslmode=disable"
	const frontera = "postgres://vec_externo_meritos_frontera@localhost:5432/vec?sslmode=disable"
	const fuente = "postgres://vec_externo_meritos_fuente_autorizacion@localhost:5432/vec?sslmode=disable"
	const registro = "postgres://vec_externo_meritos_registro_autorizacion@localhost:5432/vec?sslmode=disable"
	const motivos = "postgres://vec_externo_meritos_motivos_autorizacion@localhost:5432/vec?sslmode=disable"
	const contexto = "postgres://vec_externo_meritos_contexto@localhost:5432/vec?sslmode=disable"
	base := Config{ExternoMeritosCarteraPostgreSQL: ConfiguracionPostgreSQLExterna{dsn: datos},
		ExternoMeritosFronteraPostgreSQL:             ConfiguracionPostgreSQLExterna{dsn: frontera},
		ExternoMeritosFuenteAutorizacionPostgreSQL:   ConfiguracionPostgreSQLExterna{dsn: fuente},
		ExternoMeritosRegistroAutorizacionPostgreSQL: ConfiguracionPostgreSQLExterna{dsn: registro},
		ExternoMeritosMotivosAutorizacionPostgreSQL:  ConfiguracionPostgreSQLExterna{dsn: motivos},
		ExternoMeritosContextoPostgreSQL:             ConfiguracionPostgreSQLExterna{dsn: contexto}}
	if d, f, s, r, m, c, err := base.DSNMeritosCarteraSeparados(); err != nil || d != datos || f != frontera || s != fuente || r != registro || m != motivos || c != contexto {
		t.Fatalf("seis login nominales M4: %v", err)
	}
	base.ExternoMeritosFronteraPostgreSQL = ConfiguracionPostgreSQLExterna{}
	if _, _, _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLIncompleto) {
		t.Fatalf("ausencia no detectada: %v", err)
	}
	base.ExternoMeritosFronteraPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: datos}
	if _, _, _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLNoSeparado) {
		t.Fatalf("mismo login en datos y frontera: %v", err)
	}
	base.ExternoMeritosFronteraPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: frontera}
	base.ExternoBolsaPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: datos}
	if _, _, _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLNoSeparado) {
		t.Fatalf("login Bolsa reutilizado: %v", err)
	}
	base.ExternoBolsaPostgreSQL = ConfiguracionPostgreSQLExterna{}
	base.ExternoMeritosRegistroAutorizacionPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: frontera}
	if _, _, _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLNoSeparado) {
		t.Fatalf("login V3 reutiliza frontera: %v", err)
	}
	base.ExternoMeritosRegistroAutorizacionPostgreSQL = ConfiguracionPostgreSQLExterna{
		dsn: "postgres://vec_externo_otro@localhost:5432/vec?sslmode=disable"}
	if _, _, _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLNoSeparado) ||
		!strings.Contains(err.Error(), "clave=login_m4_3 esperado=vec_externo_meritos_registro_autorizacion actual=vec_externo_otro") {
		t.Fatalf("login nominal distinto no identificado: %v", err)
	}
	base.ExternoMeritosRegistroAutorizacionPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: registro}
	base.ExternoMeritosContextoPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: frontera}
	if _, _, _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLNoSeparado) {
		t.Fatalf("contexto CA15 no segregado: %v", err)
	}
}
