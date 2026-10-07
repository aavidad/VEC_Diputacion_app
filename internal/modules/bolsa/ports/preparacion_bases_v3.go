package ports

import (
	"context"
	"errors"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	vec "vec-diputacion-granada/internal/vec/domain"
	core "vec-diputacion-granada/internal/vec/ports"
)

var ErrPreparacionBasesDenegada = errors.New("bolsa.preparacion_bases.denegada")

// Solicitudes internas: actor, correlacion y ambito proceden de autoridades
// registradas del servidor; no se reconstruyen desde el cuerpo HTTP.
type SolicitudGuardarPreparacionBasesV3 struct {
	bloqueoSerializacionGobiernoConvocatoria
	Actor          vec.ContextoActor
	Correlacion    vec.ReferenciaCorrelacionAutorizacionV2
	Ambito         bolsa.AmbitoOrganizativoConvocatoria
	Esperada       prep.Esperada
	Material       prep.Material
	ClaveOperacion string
}

type SelectorConsultaPreparacionBases struct {
	Modo   string
	Exacta prep.Esperada
}

type SolicitudConsultarPreparacionBasesV3 struct {
	bloqueoSerializacionGobiernoConvocatoria
	Actor       vec.ContextoActor
	Correlacion vec.ReferenciaCorrelacionAutorizacionV2
	Ambito      bolsa.AmbitoOrganizativoConvocatoria
	Selector    SelectorConsultaPreparacionBases
}

type OrdenGuardarPreparacionBasesV3 struct {
	bloqueoSerializacionGobiernoConvocatoria
	Solicitud    SolicitudGuardarPreparacionBasesV3
	Autorizacion core.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type OrdenConsultarPreparacionBasesV3 struct {
	bloqueoSerializacionGobiernoConvocatoria
	Solicitud    SolicitudConsultarPreparacionBasesV3
	Autorizacion core.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// El acceso actual es independiente del recibo de la version historica.
// Una ausencia o conflicto autorizado conserva esta evidencia tras COMMIT.
type EvidenciaAccesoPreparacionBasesV3 struct {
	DecisionRef         string
	ConsumoHuellaSHA256 string
	AuditoriaRef        string
	ReciboRef           string
	CorrelacionRef      string
	AccedidaEn          time.Time
}

type ResultadoPreparacionBasesV3 struct {
	Estado  string
	Version prep.Version
	Recibo  ReciboPreparacionBases
	Acceso  EvidenciaAccesoPreparacionBasesV3
}

type ProveedorAutorizacionPreparacionBasesV3 interface {
	AutorizarGuardadoPreparacionBases(context.Context, SolicitudGuardarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
	AutorizarConsultaPreparacionBases(context.Context, SolicitudConsultarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// Estas funciones consumen las diez piezas nominales V3 en la misma
// transaccion serializable que el resultado, la auditoria y los efectos.
// Nunca reciben ni convierten EvidenciaUsoDecisionAutorizacion funcional.
type RepositorioPreparacionBasesV3 interface {
	GuardarPreparacionBasesV3(context.Context, OrdenGuardarPreparacionBasesV3) (ResultadoPreparacionBasesV3, error)
	ConsultarPreparacionBasesV3(context.Context, OrdenConsultarPreparacionBasesV3) (ResultadoPreparacionBasesV3, error)
}

// PreparadorBasesDurableV3 es el puerto que consume Seleccion. Una llamada
// actual recupera cabeza y version exacta de Bolsa sin memoria del navegador.
// Recuperar no aprueba, publica, firma ni sustituye la version conservada.
type PreparadorBasesDurableV3 interface {
	Guardar(context.Context, SolicitudGuardarPreparacionBasesV3) (ResultadoPreparacionBasesV3, error)
	Consultar(context.Context, SolicitudConsultarPreparacionBasesV3) (ResultadoPreparacionBasesV3, error)
}
