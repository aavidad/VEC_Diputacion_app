package ports

import "context"

// SolicitudSeleccionFirmanteV2 son los datos del paso del plan y el
// certificado verificado. Ninguno concede acceso: AUT56 resuelve con las
// autoridades propietarias quién es el firmante y qué ejerce.
type SolicitudSeleccionFirmanteV2 struct {
	CertificadoHuella, CargoRef, RolID, Accion, TipoRecurso, Finalidad string
	OrganizacionRef, UnidadRef                                         string
}

type ReferenciaVersionadaFirmanteV2 struct {
	Referencia   string
	Version      uint64
	HuellaSHA256 string
}

// SeleccionFirmanteV2 es la respuesta de seleccionar_firmante_plan_ct_v1:
// persona y cuenta del certificado, su vínculo, el perfil activo con el rol
// del paso, el enlace de ejercicio del cargo y las versiones y huellas de
// asignación, rol y control. Es una lectura previa al PDP; AUT32/AUT35 y el
// consumo V3 lo vuelven a comprobar en la transacción de la firma.
type SeleccionFirmanteV2 struct {
	PersonaRef, CuentaRef, PerfilActivoRef, RolID, CargoRef, EnlaceEjercicioRef string
	VinculoCertificado                                                          ReferenciaVersionadaFirmanteV2
	Asignacion                                                                  ReferenciaVersionadaFirmanteV2
	AsignacionVigenteDesde, AsignacionVigenteHasta                              string
	RolRef, RolHuellaSHA256                                                     string
	ControlRol                                                                  ReferenciaVersionadaFirmanteV2
}

// FuenteSeleccionFirmanteV2 la implementa el adaptador PostgreSQL de AUT56.
// ErrCompetenciaFirmanteNoAcreditada: no hay selección única para ese paso;
// ErrCompetenciaFirmanteNoDisponible: cualquier otro fallo.
type FuenteSeleccionFirmanteV2 interface {
	SeleccionarFirmanteV2(context.Context, SolicitudSeleccionFirmanteV2) (SeleccionFirmanteV2, error)
}
