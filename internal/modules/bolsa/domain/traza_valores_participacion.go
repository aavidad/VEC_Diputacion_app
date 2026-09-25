package domain

// Petición RRHH p.4: la traza de un cambio de datos de contacto dice qué
// campos cambiaron, nunca su valor en claro ni una huella de él; el valor
// anterior y el nuevo quedan en las versiones cifradas que la traza enlaza.

// Nombres de campo de contacto admitidos en la traza. Coinciden con las
// claves del agregado canónico y con la comprobación de la migración 000034.
const (
	CampoTrazaCorreo    = "correo"
	CampoTrazaTelefono1 = "telefono_1"
	CampoTrazaTelefono2 = "telefono_2"
)

// CamposContactoCambiados devuelve, en orden fijo, los campos cuyo valor
// difiere de la versión anterior. Sin versión anterior cuentan como cambiados
// los campos informados.
func CamposContactoCambiados(anterior *DatosContactoParticipacion, nuevo DatosContactoParticipacion) []string {
	var previo DatosContactoParticipacion
	if anterior != nil {
		previo = *anterior
	}
	campos := make([]string, 0, 3)
	if previo.Correo != nuevo.Correo {
		campos = append(campos, CampoTrazaCorreo)
	}
	if previo.Telefono1 != nuevo.Telefono1 {
		campos = append(campos, CampoTrazaTelefono1)
	}
	if previo.Telefono2 != nuevo.Telefono2 {
		campos = append(campos, CampoTrazaTelefono2)
	}
	return campos
}
