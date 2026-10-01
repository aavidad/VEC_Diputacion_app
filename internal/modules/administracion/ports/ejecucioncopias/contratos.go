// Package ejecucioncopias define las autoridades externas necesarias para copiar
// y restaurar un conjunto completo. Las referencias son opacas para aplicación.
package ejecucioncopias

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

type Peticion struct {
	OperacionRef, ActorRef, OrigenRef, DestinoRef, MotivoRef, ConjuntoRef, PoliticaRef string
}

// Lectura pertenece a una observación actual de la plataforma. Esperado procede
// de su descriptor autenticado, Observado de la instalación efectiva.
// PreimagenSHA256 incluye el estado real de datos y ficheros; no es la huella
// estructural del inventario ni una declaración del cliente.
type Lectura struct {
	Esperado, Observado         copias.Inventario
	Politica                    copias.Politica
	VersionRef, PreimagenSHA256 string
}

type Inventario interface {
	LeerActual(context.Context, string) (Lectura, error)
}

type Autorizador interface {
	// Autorizar consulta la autoridad vigente, incluso durante mantenimiento.
	Autorizar(context.Context, Peticion, string) (Concesion, error)
	// Aprobar exige dos Persona canónicas distintas, asignación nominal vigente,
	// huella de propuesta/preimagen y caducidad. Lo vuelve a comprobar al ejecutar.
	Aprobar(context.Context, Propuesta) (Aprobacion, error)
}

type Concesion struct {
	ActorRef, Accion, RecursoRef, DecisionRef string
	Vence                                     time.Time
}

type Propuesta struct {
	Peticion
	HuellaPropuesta, PreimagenSHA256, ConjuntoPreviaRef string
}

type Aprobacion struct {
	ProponentePersonaRef, AprobadorPersonaRef     string
	HuellaPropuesta, PreimagenSHA256, DecisionRef string
	Vence                                         time.Time
}

// Registro y auditoría durables residen fuera del conjunto restaurable.
// CAS valida versión/preimagen y conserva un diario de solo adición.
type Registro interface {
	Reservar(context.Context, Peticion) (Operacion, error)
	ReservarRestauracion(context.Context, Propuesta) (Operacion, error)
	Anotar(context.Context, string, string, string) error
	AplicarCopia(context.Context, EventoCopia) error
	CAS(context.Context, string, string, string, string) (Operacion, error)
	Leer(context.Context, string) (Operacion, error)
}

// RegistroConciliacion conserva una observación de plataforma ligada al CAS
// original. Su replay no añade versiones ni autoriza efectos destructivos.
type RegistroConciliacion interface {
	ConciliarRestauracion(context.Context, ObservacionRestauracion) error
}

type ObservacionRestauracion struct {
	Propuesta
	VersionRef, InstaladoRef, IndiceAutenticadoRef string
}

type EventoCopia struct {
	OperacionRef, Transicion, ConjuntoRef, ManifiestoSHA256, EjecucionRef string
	Ensayo                                                                *EnsayoRegistrado
}

type EnsayoRegistrado struct {
	Modo      ModoEnsayo
	Evidencia copias.Evidencia
	Resultado string
}

type Operacion struct {
	Ref, VersionRef, Estado, ConjuntoRef, ConjuntoPreviaPlaneadaRef, PoliticaRef, IndiceAutenticadoRef, CopiaPreviaRef, PlanRef, PreimagenSHA256, HuellaPropuesta string
}

// Ventana excluye todos los escritores, despachos y migradores; conserva la
// exclusión desde la copia previa hasta terminar sustitución o reversión.
type Ventana interface {
	Capturar(context.Context, Peticion, Lectura) (Captura, error)
	AbrirRestauracion(context.Context, Peticion) (Exclusion, error)
}

type Exclusion interface {
	CapturarPrevia(context.Context, Peticion, Lectura) (Captura, error)
	PreimagenActual(context.Context) (string, error)
	Cerrar(context.Context) error
	// Conserva el bloqueo durable de despachos/integraciones después del
	// intento de sustitución y hasta conciliación externa.
	ConservarMantenimiento(context.Context) error
}

// Captura agrupa físico frío, lógico, globals, ficheros y release de una única
// ventana observada. El capturador acredita parada limpia y cobertura completa.
type Captura struct {
	Manifiesto copias.Manifiesto
	Origen     copias.Evidencia
}

// Destino sella y publica por CS03 únicamente conjuntos completos. Recuperar
// comprueba autenticidad de índice, manifiesto y componentes antes de ensayar.
type Destino interface {
	Publicar(context.Context, Captura) (Conjunto, error)
	CerrarVerificacion(context.Context, Conjunto, copias.Verificacion) (Conjunto, error)
	Recuperar(context.Context, string) (Conjunto, error)
}

type Conjunto struct {
	Ref, IndiceAutenticadoRef, ManifiestoSHA256 string
	// Base y ejecución conservan el vínculo inicial del índice pendiente aun
	// cuando el índice final y la huella del manifiesto hayan cambiado.
	ManifiestoBaseSHA256, EjecucionVerificacionRef string
	Manifiesto                                     copias.Manifiesto
	Origen                                         copias.Evidencia
}

type ModoEnsayo string

const (
	Fisico ModoEnsayo = "fisico"
	Logico ModoEnsayo = "logico"
)

type Ensayo struct {
	Modo               ModoEnsayo
	Evidencia          copias.Evidencia
	VerificadorVersion string
	// Arranque usa el binario archivado con despachos detenidos.
	ArranqueRef string
}

type Ensayador interface {
	Ensayar(context.Context, Conjunto, ModoEnsayo) (Ensayo, error)
}

// Plataforma prepara el conjunto en ubicaciones aisladas y deja todos los
// efectos de sustitución/rollback en un diario externo, reconciliable tras fallo.
type Plataforma interface {
	Preparar(context.Context, Conjunto) (Preparado, error)
	Sustituir(context.Context, Preparado, string) error
	ArrancarAislado(context.Context, Preparado) (string, error)
	Revertir(context.Context, Conjunto, string) error
	IdentificarInstalado(context.Context, string) (string, error)
}

type Preparado struct {
	ConjuntoRef, PlanRef string
}
