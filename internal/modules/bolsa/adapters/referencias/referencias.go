// Package referencias contiene emisores productivos de identificadores opacos
// del modulo de Bolsa. No incorpora identidad, datos del expediente ni claves
// de negocio a las referencias.
package referencias

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const (
	prefijoInstantaneaOrdenBolsa   = "llamamiento_instantanea_v1_"
	prefijoPropuestaLlamamiento    = "llamamiento_propuesta_v1_"
	bytesAleatoriosReferencia      = 32
	maximoIntentosReferenciaValida = 8
)

// GeneradorCriptograficoLlamamientos implementa el puerto de referencias con
// el CSPRNG del sistema. El lector queda privado para impedir que la
// composicion productiva inyecte identificadores previsibles.
//
// La codificacion base64url sin relleno conserva 256 bits de entropia. La
// validacion posterior evita devolver, incluso por coincidencia accidental,
// texto que el dominio considere un documento personal evidente.
type GeneradorCriptograficoLlamamientos struct {
	lector io.Reader
}

// NuevoGeneradorCriptograficoLlamamientos fija crypto/rand.Reader como unica
// fuente productiva de entropia. El valor cero falla cerrado.
func NuevoGeneradorCriptograficoLlamamientos() GeneradorCriptograficoLlamamientos {
	return GeneradorCriptograficoLlamamientos{lector: rand.Reader}
}

func (g GeneradorCriptograficoLlamamientos) NuevaReferenciaInstantaneaOrdenBolsa() (string, error) {
	return g.nuevaReferencia(prefijoInstantaneaOrdenBolsa)
}

func (g GeneradorCriptograficoLlamamientos) NuevaReferenciaPropuestaLlamamiento() (string, error) {
	return g.nuevaReferencia(prefijoPropuestaLlamamiento)
}

func (g GeneradorCriptograficoLlamamientos) nuevaReferencia(prefijo string) (string, error) {
	if g.lector == nil {
		return "", puertosbolsa.ErrGeneracionReferenciaLlamamiento
	}

	aleatorio := make([]byte, bytesAleatoriosReferencia)
	for intento := 0; intento < maximoIntentosReferenciaValida; intento++ {
		if _, err := io.ReadFull(g.lector, aleatorio); err != nil {
			return "", errors.Join(puertosbolsa.ErrGeneracionReferenciaLlamamiento, err)
		}
		referencia := prefijo + base64.RawURLEncoding.EncodeToString(aleatorio)
		if puertosbolsa.ReferenciaOpacaLlamamientoValida(referencia) {
			return referencia, nil
		}
	}
	return "", puertosbolsa.ErrGeneracionReferenciaLlamamiento
}

var _ puertosbolsa.GeneradorReferenciasLlamamiento = GeneradorCriptograficoLlamamientos{}
