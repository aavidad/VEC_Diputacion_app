package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Puertos internos del lector de servicios para certificados. Conectan su
// autorización y persistencia; no crean otro dato laboral ni otro permiso.
type ProveedorAutorizacionLectorServiciosCertificados interface {
	AutorizarServiciosParaCertificados(context.Context, domain.MaterialLectorServiciosCertificados) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenLectorServiciosCertificados struct {
	Material     domain.MaterialLectorServiciosCertificados
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// RepositorioLectorServiciosCertificados revalida en la fuente que el empleado
// es el canónico de la persona actual, consume la autorización propia y lee al
// corte en una transacción con la auditoría común. Valida la respuesta antes de
// confirmar; un error deja la transacción revertida y no devuelve datos.
type RepositorioLectorServiciosCertificados interface {
	ConsultarServiciosParaCertificados(context.Context, OrdenLectorServiciosCertificados) (ResultadoServiciosParaCertificadosV2, error)
}

// IntentoLectorServiciosCertificados lleva sólo un motivo cerrado. La identidad
// la conserva el registrador de composición desde la frontera de la petición.
type IntentoLectorServiciosCertificados struct {
	Motivo string
}

type RegistroIntentosLectorServiciosCertificados interface {
	VerificarRegistroServiciosCertificados(context.Context) error
	RegistrarIntentoServiciosCertificados(context.Context, IntentoLectorServiciosCertificados) error
}
