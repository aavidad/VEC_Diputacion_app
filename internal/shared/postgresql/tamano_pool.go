package postgresql

import (
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FijarTamanoPool deja explícito el máximo de conexiones de un pool. Manda
// la configuración del despliegue: si el DSN trae pool_max_conns, pgx ya lo
// ha leído y se respeta. Si no lo trae, se usa porDefecto y nunca el valor
// implícito de pgx, que depende de los núcleos de la máquina y puede agotar
// max_connections de PostgreSQL.
func FijarTamanoPool(cfg *pgxpool.Config, dsn string, porDefecto int32) {
	if cfg == nil || dsnFijaTamanoPool(dsn) {
		return
	}
	cfg.MaxConns = max(porDefecto, 1)
}

func dsnFijaTamanoPool(dsn string) bool {
	dsn = strings.TrimSpace(dsn)
	if u, err := url.Parse(dsn); err == nil && (u.Scheme == "postgres" || u.Scheme == "postgresql") {
		return u.Query().Has("pool_max_conns")
	}
	for _, campo := range strings.Fields(dsn) {
		if strings.HasPrefix(campo, "pool_max_conns=") {
			return true
		}
	}
	return false
}
