package ports

import "context"

// SolicitudPerfilJustificanteBolsaRRHH usa únicamente la sesión interna recién
// registrada y su confirmación durable. Ningún campo procede del navegador.
type SolicitudPerfilJustificanteBolsaRRHH struct {
	AutenticacionRef      string
	SesionRef             string
	CuentaConfirmadaRef   string
	ControlSesionRef      string
	ControlSesionRevision uint64
	ControlSesionSHA256   string
	AutenticacionSHA256   string
	CorrelacionRef        string
}

type SeleccionPerfilJustificanteBolsaRRHH struct {
	PerfilActivoRef string
	AuditoriaRef    string
}

// SeleccionadorPerfilJustificanteBolsaRRHH devuelve una selección positiva
// única o una denegación auditada con PerfilActivoRef vacío.
type SeleccionadorPerfilJustificanteBolsaRRHH interface {
	SeleccionarPerfilJustificanteBolsaRRHH(context.Context,
		SolicitudPerfilJustificanteBolsaRRHH) (SeleccionPerfilJustificanteBolsaRRHH, error)
}
