package auditoria

import (
	"context"
	"errors"
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaConsulta              = "/api/vec/auditoria/consultas"
	AccionConsultar           = "vec.auditoria.consultar"
	ModuloAutorizacion        = "auditoria"
	TipoRecurso               = "historial_auditoria"
	AudienciaConsumo          = "vec_auditoria.consulta_rrhh.v1"
	MaximoRegistros    uint16 = 100
	MaximoIntervalo           = 31 * 24 * time.Hour
)

var (
	ErrDenegada       = errors.New("auditoria: consulta denegada")
	ErrNoDisponible   = errors.New("auditoria: fuente no disponible")
	ErrFuenteInvalida = errors.New("auditoria: respuesta de fuente invalida")
)

// Posicion es una clave de orden interno. El HTTP expone solo un cursor
// autenticado; los identificadores de esta estructura no son un permiso.
type Posicion struct {
	OcurridoEn time.Time
	Fuente     string
	ID         string
}

// Filtro se liga por V3 a la consulta. Hasta es exclusivo y Antes, si existe,
// excluye esa fila de la pagina siguiente.
type Filtro struct {
	Fuente        string
	ExpedienteRef string
	ActorRef      string
	Desde         time.Time
	Hasta         time.Time
	Limite        uint16
	Antes         Posicion
	FinalidadRef  string
	MotivoRef     string
}

// ConsultaAutorizada transporta el material nominal V3 hasta el adaptador del
// propietario. Ningun adaptador debe tomar un permiso de escritura por esta via.
type ConsultaAutorizada struct {
	Filtro            Filtro
	Material          vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Solicitud         vecdomain.SolicitudAutorizacionLigadaV3
	Decision          vecdomain.DecisionAutorizacionLigadaV3
	Confirmacion      vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	ResultadoContexto vecdomain.ResultadoContextoActorRegistradoV2
}

// Registro contiene solo una proyeccion autorizada. Cuando el propietario no
// conserva una preimagen legible, los mapas son vacios y solo hay huellas.
type Registro struct {
	ID               string            `json:"id"`
	Fuente           string            `json:"fuente"`
	ModuloID         string            `json:"modulo_id"`
	Accion           string            `json:"accion"`
	ActorRef         string            `json:"actor_ref"`
	OcurridoEn       time.Time         `json:"ocurrido_en"`
	Resultado        string            `json:"resultado"`
	ExpedienteRef    string            `json:"expediente_ref"`
	ReciboRef        string            `json:"recibo_ref"`
	AntesSHA256      string            `json:"antes_sha256"`
	DespuesSHA256    string            `json:"despues_sha256"`
	Motivo           string            `json:"motivo"`
	Antes            map[string]string `json:"antes"`
	Despues          map[string]string `json:"despues"`
	DatosDisponibles bool              `json:"datos_disponibles"`
}

// Cada fuente pertenece a su modulo y consulta solo su propio almacen. El
// resultado viene ordenado por (ocurrido_en,fuente,id) descendente.
type PaginaFuente struct{ Registros []Registro }

type FuenteAuditoria interface {
	ConsultarAuditoria(context.Context, ConsultaAutorizada) (PaginaFuente, error)
}
