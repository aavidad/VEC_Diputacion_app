package ports

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
)

const longitudMaterialCapacidadReserva = 32

var errGeneracionCapacidadReserva = errors.New("vec: no se pudo generar la capacidad de reserva")

// operacionCapacidadReserva mantiene el material de autoridad dentro de un
// cierre privado e inmutable. La operacion ya nace ligada a un dominio: sin
// huella esperada devuelve solo la huella persistible y, con ella, realiza la
// comparacion autoritativa sin revelar el secreto que captura.
type operacionCapacidadReserva func(huellaEsperada *string) (huella string, coincide bool)

// nuevaOperacionCapacidadReserva crea una capacidad con 256 bits del CSPRNG
// del sistema operativo y liga de forma irreversible su dominio criptografico.
// El valor concreto no conserva slices, arrays, cadenas ni punteros con el
// material: solo el cierre, cuyo entorno no expone la API de reflexion segura.
func nuevaOperacionCapacidadReserva(dominio string) (operacionCapacidadReserva, error) {
	return nuevaOperacionCapacidadReservaConFuenteEntropia(dominio, rand.Reader)
}

// nuevaOperacionCapacidadReservaConFuenteEntropia es el punto interno de
// construccion. La fuente inyectable permite demostrar la separacion de
// dominios con la misma entropia sin publicar constructores de material.
func nuevaOperacionCapacidadReservaConFuenteEntropia(
	dominio string,
	fuente io.Reader,
) (operacionCapacidadReserva, error) {
	if dominio == "" || fuente == nil {
		return nil, errGeneracionCapacidadReserva
	}

	var material [longitudMaterialCapacidadReserva]byte
	if _, err := io.ReadFull(fuente, material[:]); err != nil {
		return nil, errGeneracionCapacidadReserva
	}
	var ceros [longitudMaterialCapacidadReserva]byte
	if subtle.ConstantTimeCompare(material[:], ceros[:]) == 1 {
		clear(material[:])
		return nil, errGeneracionCapacidadReserva
	}

	// Los parametros de la funcion inmediata son copias independientes. El
	// cierre conserva esas copias mientras que el buffer de generacion se borra.
	operacion := func(
		dominioCapturado string,
		materialCapturado [longitudMaterialCapacidadReserva]byte,
	) operacionCapacidadReserva {
		return func(huellaEsperada *string) (string, bool) {
			calculador := sha256.New()
			_, _ = calculador.Write([]byte(dominioCapturado))
			_, _ = calculador.Write([]byte{0})
			_, _ = calculador.Write(materialCapturado[:])
			huellaCalculada := calculador.Sum(nil)

			if huellaEsperada == nil {
				return hex.EncodeToString(huellaCalculada), true
			}
			if len(*huellaEsperada) != hex.EncodedLen(sha256.Size) {
				return "", false
			}
			var bytesEsperados [sha256.Size]byte
			if _, err := hex.Decode(bytesEsperados[:], []byte(*huellaEsperada)); err != nil ||
				hex.EncodeToString(bytesEsperados[:]) != *huellaEsperada {
				return "", false
			}
			return "", subtle.ConstantTimeCompare(huellaCalculada, bytesEsperados[:]) == 1
		}
	}(dominio, material)
	clear(material[:])
	return operacion, nil
}

func operacionCapacidadReservaValida(operacion operacionCapacidadReserva) bool {
	return operacion != nil
}

func huellaCapacidadReserva(operacion operacionCapacidadReserva) (string, bool) {
	if operacion == nil {
		return "", false
	}
	return operacion(nil)
}

func coincideHuellaCapacidadReserva(
	operacion operacionCapacidadReserva,
	huellaEsperada string,
) bool {
	if operacion == nil {
		return false
	}
	_, coincide := operacion(&huellaEsperada)
	return coincide
}
