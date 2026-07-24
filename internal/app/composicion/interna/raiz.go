package interna

import (
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
