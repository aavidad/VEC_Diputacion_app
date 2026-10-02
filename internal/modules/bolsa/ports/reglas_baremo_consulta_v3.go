package ports

import (
	"context"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// SelectorGobiernoReglasV3 no acredita existencia ni permiso. Las coordenadas
// completas se cotejan contra una misma fila después de consumir V3.
type SelectorGobiernoReglasV3 struct {
	Identidad reglas.IdentidadConjuntoReglasBaremo
	Estado    reglas.VinculoEstadoReglasBaremo
}

type OrdenConsultaGobiernoReglasV3 struct {
	Selector              SelectorGobiernoReglasV3
	MaterialCanonico      []byte
	HuellaMaterialSHA256  string
	ClaveOperacion        string
	HuellaSolicitudSHA256 string
	Autorizacion          vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ResultadoConsultaGobiernoReglasV3 struct {
	VersionCanonica []byte
	Acceso          EvidenciaAccesoGobiernoReglasV3
}

type ResultadoRecuperacionGobiernoReglasV3 struct {
	Recibo ReciboAltaBorradorReglasV3
	Acceso EvidenciaAccesoGobiernoReglasV3
	Existe bool
}

// Ambas lecturas consumen decisión V3 actual y añaden auditoría en la misma
// transacción que obtiene los datos. Recuperar no crea borradores ni altera
// historia/recibo; también autoriza y audita la respuesta ausente. No sustituir
// selector por "última versión", ni reutilizar una concesión histórica.
// RecuperarRecibo exige un selector original conocido. Una primera respuesta
// perdida se reconcilia reintentando el alta con la misma intención estable.
type ConsultaGobiernoReglasBaremoV3 interface {
	ObtenerExacta(context.Context, OrdenConsultaGobiernoReglasV3) (ResultadoConsultaGobiernoReglasV3, error)
	RecuperarRecibo(context.Context, OrdenConsultaGobiernoReglasV3) (ResultadoRecuperacionGobiernoReglasV3, error)
}
