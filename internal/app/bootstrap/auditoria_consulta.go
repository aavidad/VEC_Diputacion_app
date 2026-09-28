package bootstrap

import (
	"context"
	"reflect"

	"github.com/jackc/pgx/v5/pgxpool"

	bolsaauditoria "vec-diputacion-granada/internal/modules/bolsa/adapters/auditoriaconsulta"
	ctauditoria "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/auditoriaconsulta"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La raíz aporta pools nominales, emisores independientes y fronteras ya
// compuestas. Esta fábrica no activa la capacidad ni publica concesiones.
type dependenciasAuditoriaConsultaRRHH struct {
	PoolCT, PoolBolsa     *pgxpool.Pool
	EmisorCT, EmisorBolsa auditoria.EmisorMaterialV3
	Identidad             auditoria.IdentidadConsulta
	Opciones              auditoria.ProveedorOpciones
}

func nuevasRutasAuditoriaConsultaRRHH(d dependenciasAuditoriaConsultaRRHH) ([]vechttp.RutaExacta, error) {
	if d.PoolCT == nil || d.PoolBolsa == nil || dependenciaAuditoriaConsultaNula(d.EmisorCT) ||
		dependenciaAuditoriaConsultaNula(d.EmisorBolsa) || dependenciaAuditoriaConsultaNula(d.Identidad) ||
		dependenciaAuditoriaConsultaNula(d.Opciones) {
		return nil, auditoria.ErrNoDisponible
	}
	ct, err := ctauditoria.NuevaFuente(d.PoolCT)
	if err != nil {
		return nil, err
	}
	bolsa, err := bolsaauditoria.NuevaFuente(d.PoolBolsa)
	if err != nil {
		return nil, err
	}
	servicio, err := auditoria.NuevoServicio(emisorAuditoriaConsultaRRHH{ct: d.EmisorCT, bolsa: d.EmisorBolsa}, ct, bolsa)
	if err != nil {
		return nil, err
	}
	manejador, err := auditoria.NuevoManejador(servicio, d.Opciones, d.Identidad)
	if err != nil {
		return nil, err
	}
	return []vechttp.RutaExacta{
		{Ruta: auditoria.RutaOpciones, Manejador: manejador},
		{Ruta: auditoria.RutaConsulta, Manejador: manejador},
	}, nil
}

func dependenciaAuditoriaConsultaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return r.IsNil()
	}
	return false
}

// No se infiere la fuente de la referencia, del perfil ni de una cabecera.
// La propia solicitud V3, creada por el servicio tras validar el filtro,
// conserva el ámbito exacto y determina un único emisor.
type emisorAuditoriaConsultaRRHH struct {
	ct, bolsa auditoria.EmisorMaterialV3
}

var _ auditoria.EmisorMaterialV3 = emisorAuditoriaConsultaRRHH{}

func (e emisorAuditoriaConsultaRRHH) EmitirMaterialAutorizacionAtestadaV3(
	ctx context.Context,
	s vecdomain.SolicitudAutorizacionLigadaV3,
	c vecdomain.ResultadoContextoActorRegistradoV2,
) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	var decision vecdomain.DecisionAutorizacionLigadaV3
	var confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	if ctx == nil || ctx.Err() != nil || dependenciaAuditoriaConsultaNula(e.ct) || dependenciaAuditoriaConsultaNula(e.bolsa) {
		return decision, confirmacion, nil, auditoria.ErrDenegada
	}
	datos, err := s.Datos()
	if err != nil || datos.Accion != auditoria.AccionConsultar ||
		datos.Recurso.ModuloID != auditoria.ModuloAutorizacion ||
		datos.Recurso.Tipo != auditoria.TipoRecurso || len(datos.Recurso.Ambitos) != 2 ||
		datos.Recurso.Ambitos["expediente_ref"] != datos.Recurso.Referencia ||
		len(datos.Recurso.Atributos) != 1 || !huellaAuditoriaConsultaValida(datos.Recurso.Atributos["filtro_sha256"]) {
		return decision, confirmacion, nil, auditoria.ErrDenegada
	}
	switch datos.Recurso.Ambitos["fuente"] {
	case "ct":
		return e.ct.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	case "bolsa":
		return e.bolsa.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	default:
		return decision, confirmacion, nil, auditoria.ErrDenegada
	}
}

func huellaAuditoriaConsultaValida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, caracter := range valor {
		if !((caracter >= '0' && caracter <= '9') || (caracter >= 'a' && caracter <= 'f')) {
			return false
		}
	}
	return true
}
