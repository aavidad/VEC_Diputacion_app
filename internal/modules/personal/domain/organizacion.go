package domain

import "errors"

const (
	prefijoReferenciaOrganizacion = "org_"
	minimoSufijoOrganizacion      = 16
	maximoSufijoOrganizacion      = 80
)

var ErrReferenciaOrganizacionInvalida error = errors.New("personal: referencia de organizacion invalida")

// ReferenciaOrganizacion conserva el identificador nominal canonico de una
// organizacion. Su forma no acredita existencia, actividad, vigencia,
// procedencia ni autorizacion.
type ReferenciaOrganizacion struct {
	referencia string
}

// NuevaReferenciaOrganizacion admite exclusivamente la representacion
// canonica recibida, sin normalizarla ni completarla.
func NuevaReferenciaOrganizacion(referencia string) (ReferenciaOrganizacion, error) {
	valor := ReferenciaOrganizacion{referencia: referencia}
	if err := valor.Validar(); err != nil {
		return ReferenciaOrganizacion{}, err
	}
	return valor, nil
}

// Validar reacredita la forma canonica del valor nominal.
func (r ReferenciaOrganizacion) Validar() error {
	if !referenciaOrganizacionValida(r.referencia) {
		return ErrReferenciaOrganizacionInvalida
	}
	return nil
}

// Referencia devuelve los bytes exactos recibidos al construir un valor
// valido.
func (r ReferenciaOrganizacion) Referencia() (string, error) {
	if err := r.Validar(); err != nil {
		return "", err
	}
	return r.referencia, nil
}

func referenciaOrganizacionValida(referencia string) bool {
	longitudSufijo := len(referencia) - len(prefijoReferenciaOrganizacion)
	if longitudSufijo < minimoSufijoOrganizacion || longitudSufijo > maximoSufijoOrganizacion {
		return false
	}
	if referencia[:len(prefijoReferenciaOrganizacion)] != prefijoReferenciaOrganizacion {
		return false
	}
	for indice := len(prefijoReferenciaOrganizacion); indice < len(referencia); indice++ {
		caracter := referencia[indice]
		if (caracter < 'a' || caracter > 'z') && (caracter < '0' || caracter > '9') {
			return false
		}
	}
	return true
}
