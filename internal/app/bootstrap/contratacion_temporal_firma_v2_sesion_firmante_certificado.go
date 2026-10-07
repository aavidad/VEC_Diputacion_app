package bootstrap

import (
	"fmt"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadordinaria"
	"vec-diputacion-granada/internal/app/composicion/interna/contrataciontemporal/firmavec"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// La composición local conserva su constructor anterior. La única
// implementación de la fuente por certificado vive en firmavec.
type fuenteCertificadoFirmaVecV2 = firmavec.FuenteCertificadoTemporal

func nuevaAutoridadSesionFirmanteV2Certificado(
	comun *identidadordinaria.FuenteCertificadoTemporal, reloj vp.Reloj,
) (*autoridadSesionFirmanteV2, error) {
	fuente, err := firmavec.NuevaFuenteCertificadoTemporal(comun, reloj)
	if err != nil {
		return nil, errSesionFirmanteV2NoDisponible
	}
	a, err := nuevaAutoridadSesionFirmanteV2ConFuente(fuente, reloj, nil)
	if err != nil {
		return nil, err
	}
	a.garantia = core.AuthAssuranceSubstantial
	a.exigirCanalTLS = true
	return a, nil
}

func solicitudCertificadoFirmaVecV2Valida(q ports.SolicitudSesionFirmanteV2, ahora time.Time) bool {
	return firmavec.SolicitudValida(q, ahora)
}

type errorSesionFirmanteV2Opaco struct{ causa error }

func (e *errorSesionFirmanteV2Opaco) Error() string    { return errSesionFirmanteV2Denegada.Error() }
func (e *errorSesionFirmanteV2Opaco) String() string   { return e.Error() }
func (e *errorSesionFirmanteV2Opaco) GoString() string { return e.Error() }
func (e *errorSesionFirmanteV2Opaco) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(e.Error()))
}
func (e *errorSesionFirmanteV2Opaco) Unwrap() []error {
	return []error{errSesionFirmanteV2Denegada, e.causa}
}

func falloSesionFirmanteV2(causa error) error {
	if causa == nil {
		return errSesionFirmanteV2Denegada
	}
	return &errorSesionFirmanteV2Opaco{causa: causa}
}
