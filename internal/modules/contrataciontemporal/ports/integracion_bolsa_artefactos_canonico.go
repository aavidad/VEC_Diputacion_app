package ports

import (
	"bytes"
	"crypto/hmac"
	"encoding/json"
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
