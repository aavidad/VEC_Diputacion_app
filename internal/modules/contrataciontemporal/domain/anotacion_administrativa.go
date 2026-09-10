package domain

import "errors"

const AccionRegistrarAnotacionAdministrativa ClaveCatalogo = "contratacion_temporal.anotacion_administrativa.registrar"

var ErrAnotacionAdministrativaInvalida = errors.New(
	"contratacion temporal: anotacion administrativa invalida",
)

// VinculoSeguimientoOriginal identifica la versión inmutable que acredita la
// incorporación antecedente. La aplicación lo resuelve desde la historia
// durable; nunca se acepta desde el canal HTTP.
type VinculoSeguimientoOriginal struct {
	SeguimientoRef              string `json:"seguimiento_ref"`
	VersionSeguimiento          uint64 `json:"version_seguimiento"`
	HuellaRaizSeguimientoSHA256 string `json:"huella_raiz_seguimiento_sha256"`
}

func (v VinculoSeguimientoOriginal) Validar() error {
	if !referenciaValida(v.SeguimientoRef) || v.VersionSeguimiento != 1 ||
		!huellaValida(v.HuellaRaizSeguimientoSHA256) {
		return ErrAnotacionAdministrativaInvalida
	}
	return nil
}
