package bootstrap

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	accionLecturaExclusionesOfrecimientoRRHH    = "bolsa.rrhh.exclusiones_llamamiento.consultar"
	finalidadLecturaExclusionesOfrecimientoRRHH = "seguimiento_contacto_rrhh"
	limiteLecturaExclusionesOfrecimientoRRHH    = 1000
)

var camposLecturaExclusionesOfrecimientoRRHH = []string{
	"bolsa_ref", "criterios_ofrecimiento", "exclusiones[].eventos_relacion",
	"exclusiones[].hechos_relacion", "exclusiones[].motivo_clave", "exclusiones[].orden_acta",
	"exclusiones[].participacion_ref", "exclusiones[].situacion_en_origen",
	"hay_mas", "llamamiento_ref", "siguiente_orden_acta", "total_exclusiones",
}

func CamposLecturaExclusionesOfrecimientoRRHH() []string {
	return slices.Clone(camposLecturaExclusionesOfrecimientoRRHH)
}

func contratoExclusionesOfrecimientoRRHH() contratoLecturaBolsa {
	return contratoLecturaBolsa{accion: accionLecturaExclusionesOfrecimientoRRHH,
		finalidad: finalidadLecturaExclusionesOfrecimientoRRHH, tipo: "llamamiento_bolsa",
		motivo: "lectura_exclusiones_llamamiento_correcta", campos: CamposLecturaExclusionesOfrecimientoRRHH()}
}

// La captura nace de la sesión interna y de una concesión específica. Bolsa
// coteja la captura al leer y registra el acceso en la misma transacción.
func CapturarLecturaExclusionesOfrecimientoRRHH(ctx context.Context,
	fuente vecports.FuenteAutorizacion, vinculo vecdomain.VinculoAutenticacionActorV2,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
	unidadRef, ambitoRef, bolsaRef, llamamientoRef string,
) (vecports.CapturaLecturaNominalBolsa, error) {
	return capturarLecturaBolsa(ctx, fuente, vinculo, resultado, unidadRef, ambitoRef,
		bolsaRef, llamamientoRef, contratoExclusionesOfrecimientoRRHH())
}

func (p *preparadorBorradorLlamamientoDesarrollo) PrepararConsultaExclusionesOfrecimientoRRHH(
	ctx context.Context, bolsaRef, llamamientoRef string, desdeOrdenActa int64,
) (puertosbolsa.ConsultaExclusionesOfrecimientoRRHH, error) {
	vacia := puertosbolsa.ConsultaExclusionesOfrecimientoRRHH{}
	if p == nil || p.soporte == nil || p.fuenteRespuestas == nil ||
		p.soporte.unidadRef == "" || p.soporte.ambitoRef == "" || bolsaRef == "" ||
		llamamientoRef == "" || desdeOrdenActa < 0 {
		return vacia, puertosbolsa.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	contexto, err := p.contextoRevalidado(ctx)
	if err != nil {
		return vacia, err
	}
	captura, err := CapturarLecturaExclusionesOfrecimientoRRHH(ctx, p.fuenteRespuestas,
		contexto.Vinculo, contexto.Resultado, p.soporte.unidadRef, p.soporte.ambitoRef,
		bolsaRef, llamamientoRef)
	if err != nil {
		return vacia, err
	}
	return puertosbolsa.ConsultaExclusionesOfrecimientoRRHH{
		BolsaRef: bolsaRef, LlamamientoRef: llamamientoRef, DesdeOrdenActa: desdeOrdenActa,
		Limite:  limiteLecturaExclusionesOfrecimientoRRHH,
		Vinculo: contexto.Vinculo, ResultadoContexto: contexto.Resultado, Captura: captura,
	}, nil
}

func (m *manejadorParticipacionBolsaDesarrollo) ConsultarExclusionesOfrecimientoRRHH(
	ctx context.Context, bolsaRef, llamamientoRef string, desdeOrdenActa int64,
) (puertosbolsa.PaginaExclusionesOfrecimientoRRHH, error) {
	vacia := puertosbolsa.PaginaExclusionesOfrecimientoRRHH{}
	if m == nil || m.preparador == nil || m.exclusiones == nil {
		return vacia, puertosbolsa.ErrExclusionesOfrecimientoRRHHNoDisponibles
	}
	q, err := m.preparador.PrepararConsultaExclusionesOfrecimientoRRHH(ctx, bolsaRef, llamamientoRef, desdeOrdenActa)
	if err != nil {
		return vacia, err
	}
	return m.exclusiones.ConsultarExclusionesOfrecimientoRRHH(ctx, q)
}

func nuevoServicioExclusionesOfrecimientoRRHH(pool *pgxpool.Pool) (*aplicacionbolsa.ServicioExclusionesOfrecimientoRRHH, error) {
	repositorio, err := postgresbolsa.NuevoRepositorioExclusionesOfrecimientoRRHHPostgreSQL(pool)
	if err != nil {
		return nil, err
	}
	return aplicacionbolsa.NuevoServicioExclusionesOfrecimientoRRHH(repositorio)
}
