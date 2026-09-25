package ports

import (
	"context"
	"errors"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	// ErrSancionesNoConfiguradas: no hay catálogo de consecuencias. Se puede
	// consultar el histórico, pero no registrar sanciones nuevas.
	ErrSancionesNoConfiguradas = errors.New("bolsa: catalogo de sanciones no configurado")
	ErrSancionNoEncontrada     = errors.New("bolsa: sancion no encontrada")
)

// ConsecuenciaSancion es una entrada del catálogo de consecuencias.
type ConsecuenciaSancion struct {
	Clave    string
	Etiqueta string
	// Efecto es una operación B8 (pausar, excluir) o «ninguna».
	Efecto   string
	Articulo string
	Ejemplo  bool
	ConPlazo bool
	// OrdenFinal: la sanción coloca a la persona al final del orden vigente
	// de su bolsa (atributo «orden»: «final» del catálogo).
	OrdenFinal bool
	// FinAutomatico: la suspensión termina sola al llegar su fecha de fin
	// (atributo «fin»: «automatico»); si no, queda no disponible hasta que
	// RRHH la reactive, como antes.
	FinAutomatico bool
	ReglaRef      string
	Huella        string
	Descripcion   string
}

// PlazoSancion es el último día de un plazo calculado por el catálogo.
type PlazoSancion struct {
	UltimoDia string
	ReglaRef  string
	Huella    string
}

// ResolucionCatalogoSancion es lo que el catálogo fija para una sanción
// concreta: su consecuencia, el fin de la suspensión (si la hay) y el
// vencimiento del recurso de reposición.
type ResolucionCatalogoSancion struct {
	Consecuencia    ConsecuenciaSancion
	SuspensionHasta string
	Recurso         PlazoSancion
}

// ReversionCatalogoSancion son los estados del recurso que revierten la
// sanción (readmisión), con la regla exacta y el motivo que queda en la
// historia de situaciones. Sin estados, ningún recurso revierte nada.
type ReversionCatalogoSancion struct {
	Estados  []string
	Motivo   string
	ReglaRef string
	Huella   string
}

// Revierte indica si el estado anotado deja sin efecto la sanción.
func (r ReversionCatalogoSancion) Revierte(estado string) bool {
	for _, e := range r.Estados {
		if e == estado {
			return true
		}
	}
	return false
}

// CatalogoSancionesParticipacion traduce el catálogo de reglas. Una
// indisponibilidad nunca se sustituye por valores supuestos.
type CatalogoSancionesParticipacion interface {
	Consecuencias(context.Context) ([]ConsecuenciaSancion, error)
	EstadosRecurso(context.Context) ([]string, error)
	ReversionRecurso(context.Context) (ReversionCatalogoSancion, error)
	ResolverSancion(ctx context.Context, clave string, notificadaEn time.Time) (ResolucionCatalogoSancion, error)
}

type SolicitudRegistrarSancion struct {
	SolicitudCambiarSituacionParticipacion
	Datos dominiobolsa.DatosSancion
}

type SolicitudRegistrarRecursoSancion struct {
	SolicitudCambiarSituacionParticipacion
	SancionRef string
	Evento     dominiobolsa.EventoRecursoSancion
	// ResueltaPor es obligatoria, y distinta de quien anota, cuando el
	// estado revierte la sanción.
	ResueltaPor string
}

// ComandoRegistrarSancion lleva la sanción completa y, si cambia la
// situación, la operación B8 que la aplica en la misma transacción.
// OrdenFinal y FinAutomatico son los efectos que declara el catálogo.
type ComandoRegistrarSancion struct {
	Sancion           dominiobolsa.SancionParticipacion
	Operacion         *ComandoOperacionSituacion
	OrdenFinal        bool
	FinAutomatico     bool
	BolsaRef          string
	ClaveIdempotencia string
	ReciboRef         string
	Material          puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type RegistroSancion struct {
	Reutilizada bool
	SancionRef  string
	ReciboRef   string
	Situacion   string
	Desde       *time.Time
	OrdenFinal  bool
}

type ComandoRegistrarRecursoSancion struct {
	ParticipacionRef  string
	SancionRef        string
	Evento            dominiobolsa.EventoRecursoSancion
	ClaveIdempotencia string
	Material          puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	// Reversion, si no es nula, revierte la sanción en la misma transacción.
	Reversion *ComandoReversionSancion
}

// ComandoReversionSancion es la readmisión que produce un recurso en estado
// revocatorio: quién la resuelve, la regla del catálogo y el recibo.
type ComandoReversionSancion struct {
	ResueltaPor string
	ReglaRef    string
	Huella      string
	Motivo      string
	ReciboRef   string
}

type RegistroRecursoSancion struct {
	Reutilizada  bool
	SancionRef   string
	Estado       string
	RegistradaEn time.Time
	// Revertida, ReciboRef, Situacion y Desde describen la reversión; la
	// situación es vacía si no hubo que cambiarla.
	Revertida bool
	ReciboRef string
	Situacion string
	Desde     *time.Time
}

type RepositorioSancionesParticipacion interface {
	RegistrarSancion(context.Context, ComandoRegistrarSancion) (RegistroSancion, error)
	RegistrarRecursoSancion(context.Context, ComandoRegistrarRecursoSancion) (RegistroRecursoSancion, error)
	ListarSanciones(ctx context.Context, participacion, actor string, material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]dominiobolsa.SancionParticipacion, error)
}

// VistaSancionesParticipacion reúne el histórico y el catálogo vigente para
// la ficha. CatalogoDisponible es falso cuando no hay catálogo compuesto: el
// histórico se muestra igualmente, pero no se pueden registrar sanciones.
type VistaSancionesParticipacion struct {
	Sanciones           []dominiobolsa.SancionParticipacion
	Consecuencias       []ConsecuenciaSancion
	EstadosRecurso      []string
	EstadosRevocatorios []string
	CatalogoDisponible  bool
}
