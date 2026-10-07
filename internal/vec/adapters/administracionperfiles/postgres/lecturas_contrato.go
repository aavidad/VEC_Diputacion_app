package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// NuevaFuenteLecturas recibe componentes centrales, sin construir una
// Autoridad de actos ni requerir sus ACL. Las siete fachadas heredadas no
// acreditan por su existencia el protocolo de filtros o el acuse de auditoría.
// Hasta que se admita ese protocolo y el contrato técnico L, el constructor
// permanece cerrado. No crea una fuente alternativa ni prueba PostgreSQL.
func NuevaFuenteLecturas(ctx context.Context, pool *pgxpool.Pool, emisor Emisor,
	fuente ports.FuenteAutorizacion, catalogo ports.CatalogoRolesAdministrables, reloj ports.Reloj,
) (*FuenteLecturas, error) {
	if ctx == nil || pool == nil || ausente(emisor) || ausente(fuente) || ausente(catalogo) || ausente(reloj) {
		return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return nil, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

// El protocolo SQL heredado solo define texto de 2..80 bytes y cursor.
// Los filtros nuevos no se eliminan ni se aplican sobre una página recuperada:
// requieren una fachada sucesora que los autorice antes de paginar.
func consultaLecturaHeredada(x api.ConsultaPersonas) error {
	if !textoLectura(x.Texto, 132, true) || x.Texto != strings.TrimSpace(x.Texto) ||
		(x.Texto != "" && len(x.Texto) < 2) || !textoLectura(x.Cursor, 256, true) ||
		(x.PerfilRef != "" && !domain.RolVersionAdministracionPerfilesValido(x.PerfilRef)) ||
		(x.UnidadRef != "" && !unidadLectura(x.UnidadRef)) ||
		(x.Estado != "" && x.Estado != "vigente" && x.Estado != "caducado") {
		return domain.ErrActoAdministracionPerfilesInvalido
	}
	if x.Texto == "" || len(x.Texto) > 80 || x.PerfilRef != "" || x.UnidadRef != "" || x.Estado != "" {
		return ports.ErrAutoridadAdministracionPerfilesNoDisponible
	}
	return nil
}

func unidadLectura(s string) bool {
	if !strings.HasPrefix(s, "unidad:") || len(s) <= len("unidad:") || len(s) > 256 {
		return false
	}
	for _, c := range s[len("unidad:"):] {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' || c == ':' || c == '.') {
			return false
		}
	}
	return true
}
