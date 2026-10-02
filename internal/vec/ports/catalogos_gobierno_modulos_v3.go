package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// IdentidadGobiernoModulosV3 procede de la frontera ADMIN registrada. Los
// campos editables de un catálogo nunca son fuente de identidad o competencia.
type IdentidadGobiernoModulosV3 struct {
	Resultado domain.ResultadoContextoActorRegistradoV2
	Vinculo   domain.VinculoAutenticacionActorV2
}

// ConfiguracionGobiernoModulosAprobada es la preimagen de la configuración
// institucional. El repositorio coteja versión, huella, perfil y registro
// compuesto con CAT6 dentro de la transacción; esta prelectura no autoriza.
type ConfiguracionGobiernoModulosAprobada struct {
	Version          int64
	HuellaSHA256     string
	RegistroSHA256   string
	CatalogoID       string
	PerfilFijoRef    string
	FinalidadRef     string
	Registrados      []string
	Gobernados       []string
	MotivoCrear      domain.ReferenciaEntradaCatalogo
	MotivoActualizar domain.ReferenciaEntradaCatalogo
	MotivoPublicar   domain.ReferenciaEntradaCatalogo
	MotivoRetirar    domain.ReferenciaEntradaCatalogo
}

// FuenteGobiernoModulosV3 obtiene identidad y configuración de proveedores
// confiables de composición, nunca de una petición HTTP ni del catálogo C3.
type FuenteGobiernoModulosV3 interface {
	IdentidadRegistradaGobiernoModulos(context.Context, EvidenciaUsoDecisionAutorizacion) (IdentidadGobiernoModulosV3, error)
	ConfiguracionAprobadaGobiernoModulos(context.Context) (ConfiguracionGobiernoModulosAprobada, error)
}

// EmisorGobiernoModulosV3 entrega material de la autoridad central V3. Su
// consumo y el cambio de catálogo se confirman juntos en PostgreSQL.
type EmisorGobiernoModulosV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ConfirmacionRegistroConcesionAutorizacionLigadaV3, ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}
