package adminperfiles

import "context"

// FuenteSeleccionAuditadaADMIN confirma la selección y su registro común en
// una sola transacción. La fuente no fabrica un perfil para listar los propios.
type FuenteSeleccionAuditadaADMIN interface {
	ListarPropiosAuditadosADMIN(context.Context, ObservacionADMIN) (LecturaPropiosAuditadaADMIN, error)
	SeleccionarPerfilAuditadoADMIN(context.Context, ObservacionADMIN, string, uint64) (SeleccionAuditadaADMIN, error)
}

type LecturaPropiosAuditadaADMIN struct {
	Propios           PerfilesPropios
	AuditoriaComunRef string
}

type SeleccionAuditadaADMIN struct {
	Seleccion         SeleccionPerfil
	AuditoriaComunRef string
}
