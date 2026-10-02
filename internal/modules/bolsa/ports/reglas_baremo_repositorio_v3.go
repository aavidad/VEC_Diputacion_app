package ports

import (
	"context"
	"time"

	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// SolicitudMaterialGobiernoReglasV3 solo recibe operaciones preparadas por
// application. El broker nominal obtiene una decisión positiva actual; no
// provisiona perfiles, permisos ni concesiones durante esta petición.
type SolicitudMaterialGobiernoReglasV3 struct {
	Operacion        string
	Accion           string
	Finalidad        string
	Campos           []string
	Audiencia        string
	Recurso          vecdomain.RecursoAutorizable
	Motivo           vecdomain.ReferenciaEntradaCatalogo
	MaterialCanonico []byte
}

type ProveedorMaterialGobiernoReglasV3 interface {
	ProveerMaterialGobiernoReglasV3(context.Context, vecdomain.VinculoAutenticacionActorV2, SolicitudMaterialGobiernoReglasV3) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// OrdenAltaBorradorReglasV3 contiene la propuesta canónica del servidor y la
// intención semántica estable. La propuesta no es un estado ya confirmado.
type OrdenAltaBorradorReglasV3 struct {
	MaterialCanonico      []byte
	HuellaMaterialSHA256  string
	VersionCanonica       []byte
	HuellaVersionSHA256   string
	EstadoPropuesto       reglas.VinculoEstadoReglasBaremo
	ClaveOperacion        string
	HuellaSolicitudSHA256 string
	Autorizacion          vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// EvidenciaAccesoGobiernoReglasV3 identifica el consumo de ESTA llamada. En
// replay permanece separado de la evidencia que creó el recibo histórico.
type EvidenciaAccesoGobiernoReglasV3 struct {
	DecisionRef          string
	DecisionHuellaSHA256 string
	EfectoRef            string
	EfectoHuellaSHA256   string
	ConsumoHuellaSHA256  string
	AuditoriaRef         string
	ConsumidaEn          time.Time
}

type ReciboAltaBorradorReglasV3 struct {
	ReciboRef             string
	ClaveOperacion        string
	HuellaSolicitudSHA256 string
	Estado                reglas.VinculoEstadoReglasBaremo
	VersionCanonica       []byte
	TransaccionRef        string
	AuditoriaRef          string
	OutboxRef             string
	ConsumoOriginal       EvidenciaAccesoGobiernoReglasV3
	ConfirmadaEn          time.Time
}

type ResultadoAltaBorradorReglasV3 struct {
	Recibo ReciboAltaBorradorReglasV3
	Acceso EvidenciaAccesoGobiernoReglasV3
	Replay bool
}

// ConfirmarAltaBorrador consume V3 fresco, comprueba intención ligada al actor
// y al contenido y reconcilia ANTES de crear otro estado. Una repetición con
// el mismo material devuelve canon/fecha/recibo originales. Otra huella para
// la clave produce conflicto. Reintentar el alta resuelve también una primera
// respuesta perdida, sin conocer todavía el selector original. El material
// V3 fresco no sustituye ni redefine esa intención estable.
// Un alta nueva exige ausencia de ese contenido,
// añade revisión 1 inmutable, CAS, historia, auditoría, outbox y recibo en el
// mismo COMMIT SERIALIZABLE. Nunca usa las funciones históricas V2/AD2.
// El adaptador coteja los bytes Go originales (no jsonb::text), proyecciones,
// motivo, actor y ámbitos; PostgreSQL vuelve a verificar todo el material V3.
type RepositorioGobiernoReglasBaremoV3 interface {
	ConfirmarAltaBorrador(context.Context, OrdenAltaBorradorReglasV3) (ResultadoAltaBorradorReglasV3, error)
}
