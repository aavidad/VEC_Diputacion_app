package interna

import (
	"context"
	"errors"
)

var ErrDependenciasProductivasNoDisponibles = errors.New(
	"composicion interna: dependencias productivas obligatorias no disponibles",
)

// Dependencia identifica una capacidad de seguridad o persistencia que debe
// existir antes de construir el listener. Es un conjunto cerrado en C4.
type Dependencia string

const (
	DependenciaTLSMutuo              Dependencia = "tls_mutuo"
	DependenciaIdentidadCorporativa  Dependencia = "identidad_corporativa"
	DependenciaSesionesDurables      Dependencia = "sesiones_durables"
	DependenciaRevalidacionActor     Dependencia = "revalidacion_autenticacion_actor"
	DependenciaContextoActor         Dependencia = "contexto_actor"
	DependenciaPDPV3                 Dependencia = "pdp_v3"
	DependenciaKMSCifrado            Dependencia = "kms_cifrado"
	DependenciaKMSRevalidacion       Dependencia = "kms_revalidacion"
	DependenciaKMSVerificacionFirmas Dependencia = "kms_verificacion_firmas"
	DependenciaTSACualificada        Dependencia = "tsa_cualificada"
	DependenciaPostgreSQLEjecutor    Dependencia = "postgres_ejecutor_consulta"
	DependenciaPostgreSQLProyector   Dependencia = "postgres_proyector_gobierno"
	DependenciaPostgreSQLVerificador Dependencia = "postgres_verificador_recibo"
	DependenciaAPIInterna            Dependencia = "api_interna"
	DependenciaTransporteAsercion    Dependencia = "transporte_asercion_institucional"
	DependenciaVerificadorAsercion   Dependencia = "verificador_asercion_institucional"
	DependenciaEvaluadorGarantia     Dependencia = "evaluador_garantia_institucional"
	DependenciaF1Registrado          Dependencia = "contexto_actor_f1_registrado"
	DependenciaPDPV3Seguimiento      Dependencia = "pdp_v3_seguimiento"
	DependenciaPostgreSQLSeguimiento Dependencia = "postgres_seguimiento_v2"
	DependenciaAuditoriaLectura      Dependencia = "auditoria_lectura"
)

var dependenciasC4 = [...]Dependencia{
	DependenciaTLSMutuo,
	DependenciaIdentidadCorporativa,
	DependenciaSesionesDurables,
	DependenciaRevalidacionActor,
	DependenciaContextoActor,
	DependenciaPDPV3,
	DependenciaKMSCifrado,
	DependenciaKMSRevalidacion,
	DependenciaKMSVerificacionFirmas,
	DependenciaTSACualificada,
	DependenciaPostgreSQLEjecutor,
	DependenciaPostgreSQLProyector,
	DependenciaPostgreSQLVerificador,
	DependenciaAPIInterna,
}

// El primer corte solo consulta el seguimiento original. No necesita TSA ni
// firmar nuevos documentos; exige identidad por petición, F1, V3 y los once
// pools PostgreSQL que valida ServidorV2PostgreSQL.
var dependenciasConsultaSeguimiento = [...]Dependencia{
	DependenciaTLSMutuo,
	DependenciaTransporteAsercion,
	DependenciaVerificadorAsercion,
	DependenciaEvaluadorGarantia,
	DependenciaSesionesDurables,
	DependenciaRevalidacionActor,
	DependenciaF1Registrado,
	DependenciaPDPV3Seguimiento,
	DependenciaPostgreSQLSeguimiento,
	DependenciaAuditoriaLectura,
	DependenciaAPIInterna,
}

// ErrorDependenciasFaltantes conserva el inventario sin incluir rutas,
// credenciales ni valores ambientales en Error().
type ErrorDependenciasFaltantes struct {
	faltantes []Dependencia
}

func (e *ErrorDependenciasFaltantes) Error() string {
	return ErrDependenciasProductivasNoDisponibles.Error()
}

func (e *ErrorDependenciasFaltantes) Unwrap() error {
	return ErrDependenciasProductivasNoDisponibles
}

// Faltantes entrega una copia defensiva para diagnóstico y pruebas de
// preparación; no debe serializarse como respuesta HTTP.
func (e *ErrorDependenciasFaltantes) Faltantes() []Dependencia {
	if e == nil {
		return nil
	}
	return append([]Dependencia(nil), e.faltantes...)
}

// Falta permite comprobar una capacidad concreta sin depender del orden del
// inventario.
func (e *ErrorDependenciasFaltantes) Falta(dependencia Dependencia) bool {
	if e == nil {
		return false
	}
	for _, faltante := range e.faltantes {
		if faltante == dependencia {
			return true
		}
	}
	return false
}

// NuevoServidor valida primero el limite de red y las referencias TLS. C4 no
// admite indicadores booleanos que simulen proveedores: hasta que C5/C6
// inyecten implementaciones productivas, devuelve siempre nil y nunca llama a
// net.Listen ni construye un healthcheck vacio. Al completar las dependencias,
// esta misma raiz debera usar exclusivamente construirServidorInterno.
func NuevoServidor(cfg Configuracion) (*ServidorInterno, error) {
	if err := cfg.Validar(); err != nil {
		return nil, err
	}
	faltantes := append([]Dependencia(nil), dependenciasC4[:]...)
	return nil, &ErrorDependenciasFaltantes{faltantes: faltantes}
}

// NuevaAplicacion es la entrada del binario. El cargador institucional no
// existe aún: su ausencia devuelve un inventario antes de construir TLS o
// abrir el listener. La composición inyectada queda preparada para consumir
// exclusivamente proveedores productivos cuando se acredite su contrato.
func NuevaAplicacion(ctx context.Context, cfg Configuracion) (*AplicacionInterna, error) {
	return nuevaAplicacionConsultaSeguimiento(ctx, cfg, obtenerProveedoresConsultaSeguimiento)
}

type cargadorConsultaSeguimiento func(context.Context, Configuracion) (proveedoresConsultaSeguimiento, error)

func nuevaAplicacionConsultaSeguimiento(ctx context.Context, cfg Configuracion, cargar cargadorConsultaSeguimiento) (*AplicacionInterna, error) {
	if ctx == nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validar(); err != nil {
		return nil, err
	}
	if cargar == nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	p, err := cargar(ctx, cfg)
	if err != nil {
		cerrarProveedoresConsultaSeguimiento(p)
		return nil, err
	}
	return componerConsultaSeguimiento(ctx, cfg, p)
}
