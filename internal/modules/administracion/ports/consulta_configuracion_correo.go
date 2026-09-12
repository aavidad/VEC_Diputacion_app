package ports

import (
	"context"

	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// PreparacionConsultaConfiguracionCorreo ata el registro T13 al material
// canónico autorizado. No contiene secretos SMTP ni permite que el adaptador
// reconstruya la auditoría desde el transporte.
type PreparacionConsultaConfiguracionCorreo struct {
	Auditoria      vecdomain.AuditEntry
	PayloadNegocio []byte
}

type OrdenConsultaConfiguracionCorreoAutorizada struct {
	Preparacion PreparacionConsultaConfiguracionCorreo
	Material    vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// ResultadoConsultaConfiguracionCorreo sólo expone la vista redactada tras el
// commit y el recibo de auditoría que produjo la frontera T13.
type ResultadoConsultaConfiguracionCorreo struct {
	Vista           admindomain.VistaConfiguracionCorreo
	ReciboAuditoria vecdomain.AuditEntry
}

type PreparadorAuditoriaConsultaConfiguracionCorreo interface {
	PrepararAuditoriaConsultaConfiguracionCorreo(context.Context, vecdomain.Principal) (vecdomain.AuditEntry, error)
}

type AutorizadorConsultaConfiguracionCorreo interface {
	AutorizarConsultaConfiguracionCorreo(context.Context, PreparacionConsultaConfiguracionCorreo) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

// RegistroConsultaConfiguracionCorreo confirma en una única transacción la
// lectura, el consumo V3 y el registro T13 antes de devolver la vista.
type RegistroConsultaConfiguracionCorreo interface {
	ConsultarConfiguracionCorreoAuditada(context.Context, OrdenConsultaConfiguracionCorreoAutorizada) (ResultadoConsultaConfiguracionCorreo, error)
}
