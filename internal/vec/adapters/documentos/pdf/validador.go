package pdf

import (
	"bytes"
	"context"
	"errors"
)

var ErrSalidaPDFInvalida = errors.New("pdf: salida generada invalida")

var nombresPDFActivos = [][]byte{
	[]byte("/JavaScript"),
	[]byte("/JS"),
	[]byte("/Launch"),
	[]byte("/AA"),
	[]byte("/Type /EmbeddedFile"),
	[]byte("/RichMedia"),
	[]byte("/AcroForm"),
	[]byte("/XFA"),
	[]byte("/SubmitForm"),
	[]byte("/URI"),
}

// ValidarSalida aplica una comprobacion estructural independiente antes de
// custodiar el artefacto. No sustituye la futura validacion PDF/A y PDF/UA.
func (Renderizador) ValidarSalida(ctx context.Context, contenido []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(contenido) < len("%PDF-1.0\n%%EOF") || !bytes.HasPrefix(contenido, []byte("%PDF-")) ||
		!bytes.HasSuffix(bytes.TrimSpace(contenido), []byte("%%EOF")) {
		return ErrSalidaPDFInvalida
	}
	estructura, correcto := estructuraSinFlujos(contenido)
	if !correcto {
		return ErrSalidaPDFInvalida
	}
	for _, nombre := range nombresPDFActivos {
		if bytes.Contains(estructura, nombre) {
			return ErrSalidaPDFInvalida
		}
	}
	return nil
}

// estructuraSinFlujos retira los bytes comprimidos de cada stream. Las
// acciones PDF se declaran en diccionarios estructurales, no dentro del flujo.
func estructuraSinFlujos(contenido []byte) ([]byte, bool) {
	restante := contenido
	resultado := make([]byte, 0, len(contenido)/4)
	for {
		indice := bytes.Index(restante, []byte("stream"))
		if indice < 0 {
			resultado = append(resultado, restante...)
			return resultado, true
		}
		finCabecera := indice + len("stream")
		if finCabecera >= len(restante) {
			return nil, false
		}
		salto := 0
		switch {
		case restante[finCabecera] == '\n':
			salto = 1
		case restante[finCabecera] == '\r' && finCabecera+1 < len(restante) && restante[finCabecera+1] == '\n':
			salto = 2
		default:
			// Era texto ordinario que contenia la palabra, no un operador.
			resultado = append(resultado, restante[:finCabecera]...)
			restante = restante[finCabecera:]
			continue
		}
		resultado = append(resultado, restante[:finCabecera+salto]...)
		restante = restante[finCabecera+salto:]
		finFlujo := bytes.Index(restante, []byte("endstream"))
		if finFlujo < 0 {
			return nil, false
		}
		resultado = append(resultado, []byte("endstream")...)
		restante = restante[finFlujo+len("endstream"):]
	}
}
