package bootstrap

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/auditoria"
)

// Las tres identidades proceden de la autoridad de sesión mTLS. Cada lector
// nominal conserva su propio perfil activo; esta fábrica no une concesiones ni
// deriva la fuente de una referencia opaca o de una cabecera HTTP.
type dependenciasIdentidadAuditoriaConsultaRRHH struct {
	PoolCT, PoolBolsa           *pgxpool.Pool
	EmisorCT, EmisorBolsa       auditoria.EmisorMaterialV3
	IdentidadOpciones           auditoria.IdentidadConsulta
	IdentidadCT, IdentidadBolsa auditoria.IdentidadConsulta
	Opciones                    auditoria.ProveedorOpciones
	Intentos                    auditoria.ConfiguracionIntentos
}

// nuevasRutasAuditoriaConsultaConIdentidadesRRHH solo cablea autoridades ya
// compuestas. La raíz debe suministrar los perfiles nominales auditoria_ct y
// consulta_auditoria_bolsa, dos políticas V3 y sesiones revalidadas por mTLS.
// Una dependencia ausente cierra ambas rutas antes de publicarlas.
func nuevasRutasAuditoriaConsultaConIdentidadesRRHH(d dependenciasIdentidadAuditoriaConsultaRRHH) ([]vechttp.RutaExacta, error) {
	if d.PoolCT == nil || d.PoolBolsa == nil || dependenciaAuditoriaConsultaNula(d.EmisorCT) ||
		dependenciaAuditoriaConsultaNula(d.EmisorBolsa) || dependenciaAuditoriaConsultaNula(d.IdentidadOpciones) ||
		dependenciaAuditoriaConsultaNula(d.IdentidadCT) || dependenciaAuditoriaConsultaNula(d.IdentidadBolsa) ||
		dependenciaAuditoriaConsultaNula(d.Opciones) || dependenciaAuditoriaConsultaNula(d.Intentos.Registrador) {
		return nil, auditoria.ErrNoDisponible
	}
	return nuevasRutasAuditoriaConsultaRRHH(dependenciasAuditoriaConsultaRRHH{
		PoolCT: d.PoolCT, PoolBolsa: d.PoolBolsa,
		EmisorCT: d.EmisorCT, EmisorBolsa: d.EmisorBolsa,
		Identidad: identidadAuditoriaPorFuenteRRHH{
			opciones: d.IdentidadOpciones, ct: d.IdentidadCT, bolsa: d.IdentidadBolsa,
		},
		Opciones: d.Opciones,
		Intentos: d.Intentos,
	})
}

type identidadAuditoriaPorFuenteRRHH struct {
	opciones, ct, bolsa auditoria.IdentidadConsulta
}

var _ auditoria.IdentidadConsulta = identidadAuditoriaPorFuenteRRHH{}

func (i identidadAuditoriaPorFuenteRRHH) ResolverIdentidadConsulta(ctx context.Context, r *http.Request, fuente auditoria.FuenteConsulta) (auditoria.IdentidadResuelta, error) {
	if ctx == nil || ctx.Err() != nil || r == nil || r.URL == nil ||
		dependenciaAuditoriaConsultaNula(i.opciones) || dependenciaAuditoriaConsultaNula(i.ct) ||
		dependenciaAuditoriaConsultaNula(i.bolsa) {
		return auditoria.IdentidadResuelta{}, auditoria.ErrDenegada
	}
	if fuente == auditoria.FuenteConsultaGeneral && r.Method == http.MethodGet && r.URL.Path == auditoria.RutaOpciones {
		return i.opciones.ResolverIdentidadConsulta(ctx, r, fuente)
	}
	if r.Method != http.MethodPost || r.URL.Path != auditoria.RutaConsulta {
		return auditoria.IdentidadResuelta{}, auditoria.ErrDenegada
	}
	// FuenteConsulta procede del parser único del manejador. La identidad no
	// vuelve a leer el cuerpo, una cabecera ni el prefijo de un identificador.
	switch fuente {
	case auditoria.FuenteConsultaCT:
		if !dependenciaAuditoriaConsultaNula(i.ct) {
			return i.ct.ResolverIdentidadConsulta(ctx, r, fuente)
		}
	case auditoria.FuenteConsultaBolsa:
		if !dependenciaAuditoriaConsultaNula(i.bolsa) {
			return i.bolsa.ResolverIdentidadConsulta(ctx, r, fuente)
		}
	}
	return auditoria.IdentidadResuelta{}, auditoria.ErrDenegada
}
