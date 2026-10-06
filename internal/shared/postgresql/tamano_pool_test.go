package postgresql

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestFijarTamanoPoolRespetaElDSNYSiNoUsaElPorDefecto(t *testing.T) {
	for _, caso := range []struct {
		dsn  string
		pide int32
		max  int32
	}{
		{"postgres://u@localhost/db", 6, 6},
		{"postgres://u@localhost/db?pool_max_conns=20", 6, 20},
		{"host=localhost user=u dbname=db", 2, 2},
		{"host=localhost user=u dbname=db pool_max_conns=12", 2, 12},
		{"postgres://u@localhost/db", 0, 1},
	} {
		cfg, err := pgxpool.ParseConfig(caso.dsn)
		if err != nil {
			t.Fatal(err)
		}
		FijarTamanoPool(cfg, caso.dsn, caso.pide)
		if cfg.MaxConns != caso.max {
			t.Fatalf("%s con %d: MaxConns=%d; se esperaba %d", caso.dsn, caso.pide, cfg.MaxConns, caso.max)
		}
	}
	FijarTamanoPool(nil, "", 3)
}
