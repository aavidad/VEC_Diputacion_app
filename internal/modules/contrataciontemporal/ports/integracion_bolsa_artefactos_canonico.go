package ports

import (
	"bytes"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"io"
)

func cabeceraArtefactoProbatorioBolsaValida(
	esquema string,
	version uint64,
	tipo string,
	tipoEsperado string,
	claveRef string,
	selloHMAC string,
	evidencia EvidenciaDurableIntegracionBolsa,
	huellaDeclarada string,
	huellaCalculada string,
) bool {
	return esquema == esquemaArtefactoProbatorioBolsa &&
		version == versionArtefactoProbatorioBolsa &&
		tipo == tipoEsperado &&
		evidencia.TipoMaterial == tipoEsperado &&
		evidencia.Validar() == nil &&
		claveRef == evidencia.ClaveVerificacionRef &&
		hmac.Equal([]byte(selloHMAC), []byte(evidencia.SelloHMAC)) &&
		huellasBolsaIguales(huellaDeclarada, huellaCalculada)
}

func huellaArtefactoProbatorioBolsa(valor any) string {
	switch artefacto := valor.(type) {
	case ArtefactoProbatorioOrdenBolsa:
		artefacto.HuellaArtefactoSHA256 = ""
		valor = artefacto
	case ArtefactoProbatorioLlamamientoBolsa:
		artefacto.HuellaArtefactoSHA256 = ""
		valor = artefacto
	case ArtefactoProbatorioEventoBolsa:
		artefacto.HuellaArtefactoSHA256 = ""
		valor = artefacto
	default:
		return ""
	}
	contenido, err := json.Marshal(valor)
	if err != nil {
		return ""
	}
	defer borrarBytesIntegracionBolsa(contenido)
	return huellaBytesBolsa(contenido)
}

func decodificarArtefactoCerradoBolsa(contenido []byte, destino any) error {
	if len(contenido) == 0 ||
		len(contenido) > MaximoBytesArtefactoProbatorioBolsa ||
		destino == nil {
		return ErrEvidenciaBolsaNoAutenticada
	}
	if err := validarEncuadreJSONArtefactoBolsa(contenido); err != nil {
		return ErrEvidenciaBolsaNoAutenticada
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); err != io.EOF {
		return ErrEvidenciaBolsaNoAutenticada
	}
	return nil
}

func validarEncuadreJSONArtefactoBolsa(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	if err := recorrerValorJSONArtefactoBolsa(decodificador, 1); err != nil {
		return err
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return ErrEvidenciaBolsaNoAutenticada
	}
	return nil
}

func recorrerValorJSONArtefactoBolsa(
	decodificador *json.Decoder,
	profundidad uint32,
) error {
	if profundidad > MaximaProfundidadArtefactoProbatorioBolsa {
		return ErrEvidenciaBolsaNoAutenticada
	}
	token, err := decodificador.Token()
	if err != nil {
		return err
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		vistos := make(map[string]struct{})
		var elementos uint32
		for decodificador.More() {
			elementos++
			if elementos > MaximosElementosArtefactoProbatorioBolsa {
				return ErrEvidenciaBolsaNoAutenticada
			}
			tokenClave, err := decodificador.Token()
			if err != nil {
				return err
			}
			clave, valida := tokenClave.(string)
			if !valida {
				return ErrEvidenciaBolsaNoAutenticada
			}
			if _, duplicada := vistos[clave]; duplicada {
				return ErrEvidenciaBolsaNoAutenticada
			}
			vistos[clave] = struct{}{}
			if err := recorrerValorJSONArtefactoBolsa(
				decodificador,
				profundidad+1,
			); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim('}') {
			return ErrEvidenciaBolsaNoAutenticada
		}
		return nil
	case '[':
		var elementos uint32
		for decodificador.More() {
			elementos++
			if elementos > MaximosElementosArtefactoProbatorioBolsa {
				return ErrEvidenciaBolsaNoAutenticada
			}
			if err := recorrerValorJSONArtefactoBolsa(
				decodificador,
				profundidad+1,
			); err != nil {
				return err
			}
		}
		cierre, err := decodificador.Token()
		if err != nil || cierre != json.Delim(']') {
			return ErrEvidenciaBolsaNoAutenticada
		}
		return nil
	default:
		return ErrEvidenciaBolsaNoAutenticada
	}
}

func validarRecodificacionCanonicaArtefactoBolsa(
	contenido []byte,
	valor any,
) error {
	canonico, err := json.Marshal(valor)
	if err != nil {
		return err
	}
	defer borrarBytesIntegracionBolsa(canonico)
	if !bytes.Equal(contenido, canonico) {
		return ErrEvidenciaBolsaNoAutenticada
	}
	return nil
}
