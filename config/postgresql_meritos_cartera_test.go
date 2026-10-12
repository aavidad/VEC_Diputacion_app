package config

import (
	"errors"
	"testing"
)

func TestMeritosCarteraConexionesSeparadasYFalloSoloDelModulo(t *testing.T) {
	const datos = "postgres://vec_externo_meritos_cartera@localhost:5432/vec?sslmode=disable"
	const frontera = "postgres://vec_externo_meritos_frontera@localhost:5432/vec?sslmode=disable"
	const registro = "postgres://vec_externo_meritos_registro_autorizacion@localhost:5432/vec?sslmode=disable"
	const motivos = "postgres://vec_externo_meritos_motivos_autorizacion@localhost:5432/vec?sslmode=disable"
	base := Config{ExternoMeritosCarteraPostgreSQL: ConfiguracionPostgreSQLExterna{dsn: datos},
		ExternoMeritosFronteraPostgreSQL:             ConfiguracionPostgreSQLExterna{dsn: frontera},
		ExternoMeritosRegistroAutorizacionPostgreSQL: ConfiguracionPostgreSQLExterna{dsn: registro},
		ExternoMeritosMotivosAutorizacionPostgreSQL:  ConfiguracionPostgreSQLExterna{dsn: motivos}}
	if d, f, r, m, err := base.DSNMeritosCarteraSeparados(); err != nil || d != datos || f != frontera || r != registro || m != motivos {
		t.Fatalf("cuatro login nominales M4: %v", err)
	}
	base.ExternoMeritosFronteraPostgreSQL = ConfiguracionPostgreSQLExterna{}
	if _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLIncompleto) {
		t.Fatalf("ausencia no detectada: %v", err)
	}
	base.ExternoMeritosFronteraPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: datos}
	if _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLNoSeparado) {
		t.Fatalf("mismo login en datos y frontera: %v", err)
	}
	base.ExternoMeritosFronteraPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: frontera}
	base.ExternoBolsaPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: datos}
	if _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLNoSeparado) {
		t.Fatalf("login Bolsa reutilizado: %v", err)
	}
	base.ExternoBolsaPostgreSQL = ConfiguracionPostgreSQLExterna{}
	base.ExternoMeritosRegistroAutorizacionPostgreSQL = ConfiguracionPostgreSQLExterna{dsn: frontera}
	if _, _, _, _, err := base.DSNMeritosCarteraSeparados(); !errors.Is(err, ErrMeritosCarteraPostgreSQLNoSeparado) {
		t.Fatalf("login V3 reutiliza frontera: %v", err)
	}
}
