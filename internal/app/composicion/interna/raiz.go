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
	DependenciaPerfilActivo          Dependencia = "perfil_activo_institucional"
	DependenciaSeudonimizacionHSM    Dependencia = "seudonimizacion_hsm"
	DependenciaMaterialCOSEConsultas Dependencia = "material_cose_consultas_ct"
	DependenciaConsultaRRHHNominal   Dependencia = "postgres_consulta_rrhh_nominal"
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

// El primer montaje se limita a cuadro y detalle CT. No consume TSA ni
// firma documental; ambos pertenecen a efectos distintos.
var dependenciasLecturaCT = [...]Dependencia{
	DependenciaTLSMutuo,
	DependenciaTransporteAsercion,
	DependenciaVerificadorAsercion,
	DependenciaEvaluadorGarantia,
	DependenciaPerfilActivo,
	DependenciaSesionesDurables,
	DependenciaSeudonimizacionHSM,
	DependenciaRevalidacionActor,
	DependenciaContextoActor,
	DependenciaPDPV3,
	DependenciaMaterialCOSEConsultas,
	DependenciaConsultaRRHHNominal,
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

// NuevaAplicacion es la única entrada del binario interno. El contrato del
// proveedor institucional de aserciones y el HSM no están aún disponibles:
// esta raíz devuelve un inventario verificable antes de construir o escuchar.
// Los adaptadores de prueba no se conectan al arranque productivo.
func NuevaAplicacion(ctx context.Context, cfg Configuracion) (*AplicacionInterna, error) {
	if ctx == nil {
		return nil, ErrDependenciasProductivasNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := cfg.Validar(); err != nil {
		return nil, err
	}
	return nil, &ErrorDependenciasFaltantes{
		faltantes: append([]Dependencia(nil), dependenciasLecturaCT[:]...),
	}
}
