package domain

import "regexp"

// Las referencias que emite el propio sistema llevan un espacio de nombres
// solo con letras y una huella SHA-256 en hexadecimal en minusculas. Esa
// forma no puede contener un documento de identidad escrito por una persona,
// pero sus cifras hexadecimales si casan por azar con el patron de DNI; por
// eso quedan fuera de ese filtro.
var patronReferenciaPropiaSistema = regexp.MustCompile(`^[a-z_]+(:[a-z_]+)*:[0-9a-f]{64}$`)

// ReferenciaPropiaSistema indica si la referencia tiene la forma exacta que
// genera el sistema: espacio de nombres alfabetico y huella SHA-256 en
// hexadecimal. Solo esas referencias quedan exentas del filtro de documentos
// de identidad, que en otro caso rechazaria por azar muchas huellas validas.
// Los UUID conservan su excepcion acotada a campos concretos (por ejemplo,
// la evaluacion del plazo en la integracion de llamamientos).
func ReferenciaPropiaSistema(valor string) bool {
	return patronReferenciaPropiaSistema.MatchString(valor)
}
