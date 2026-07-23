package postgresimportacionconvoca

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

type incidenciaPostgreSQL struct {
	Fila   int    `json:"fila"`
	Campo  string `json:"campo"`
	Codigo string `json:"codigo"`
}

type filaProtegidaPostgreSQL struct {
	Numero                        int    `json:"numero"`
	EsquemaProteccion             string `json:"esquema_proteccion"`
	ClaveRef                      string `json:"clave_ref"`
	NonceHex                      string `json:"nonce_hex"`
	ContenidoCifradoHex           string `json:"contenido_cifrado_hex"`
	HuellaContenidoCifradoSHA256  string `json:"huella_contenido_cifrado_sha256"`
	DerivacionDocumentoHMACSHA256 string `json:"derivacion_documento_hmac_sha256"`
}

func serializarIncidencias(incidencias []dominio.Incidencia) ([]byte, error) {
	datos := make([]incidenciaPostgreSQL, len(incidencias))
	for i := range incidencias {
		datos[i] = incidenciaPostgreSQL(incidencias[i])
	}
	return json.Marshal(datos)
}

func deserializarIncidencias(contenido []byte) ([]dominio.Incidencia, error) {
	var datos []incidenciaPostgreSQL
	if err := decodificarJSONExacto(contenido, &datos); err != nil {
		return nil, ErrLoteNoConfiable
	}
	resultado := make([]dominio.Incidencia, len(datos))
	for i := range datos {
		resultado[i] = dominio.Incidencia(datos[i])
	}
	return resultado, nil
}

func serializarFilasProtegidas(filas []FilaStagingProtegida) ([]byte, error) {
	datos := make([]filaProtegidaPostgreSQL, len(filas))
	for i := range filas {
		huella := sha256.Sum256(filas[i].ContenidoCifrado)
		datos[i] = filaProtegidaPostgreSQL{
			Numero: filas[i].Numero, EsquemaProteccion: filas[i].EsquemaProteccion,
			ClaveRef:                     filas[i].ClaveRef,
			NonceHex:                     hex.EncodeToString(filas[i].Nonce),
			ContenidoCifradoHex:          hex.EncodeToString(filas[i].ContenidoCifrado),
			HuellaContenidoCifradoSHA256: hex.EncodeToString(huella[:]),
			DerivacionDocumentoHMACSHA256: hex.EncodeToString(
				filas[i].DerivacionDocumentoHMACSHA256,
			),
		}
	}
	return json.Marshal(datos)
}

func deserializarFilasProtegidas(contenido []byte) ([]FilaStagingProtegida, error) {
	var datos []filaProtegidaPostgreSQL
	if err := decodificarJSONExacto(contenido, &datos); err != nil {
		return nil, ErrLoteNoConfiable
	}
	filas := make([]FilaStagingProtegida, len(datos))
	for i := range datos {
		nonce, errNonce := hex.DecodeString(datos[i].NonceHex)
		cifrado, errCifrado := hex.DecodeString(datos[i].ContenidoCifradoHex)
		derivacion, errDerivacion := hex.DecodeString(datos[i].DerivacionDocumentoHMACSHA256)
		huellaEsperada, errHuella := hex.DecodeString(datos[i].HuellaContenidoCifradoSHA256)
		huella := sha256.Sum256(cifrado)
		if errNonce != nil || errCifrado != nil || errDerivacion != nil ||
			errHuella != nil || len(huellaEsperada) != sha256.Size ||
			!bytesIgualesConstantes(huella[:], huellaEsperada) {
			return nil, ErrLoteNoConfiable
		}
		filas[i] = FilaStagingProtegida{
			Numero: datos[i].Numero, EsquemaProteccion: datos[i].EsquemaProteccion,
			ClaveRef: datos[i].ClaveRef, Nonce: nonce, ContenidoCifrado: cifrado,
			DerivacionDocumentoHMACSHA256: derivacion,
		}
	}
	return filas, nil
}

func decodificarJSONExacto(contenido []byte, destino any) error {
	if len(contenido) < 2 {
		return errors.New("JSON vacio")
	}
	decodificador := json.NewDecoder(newLectorBytes(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if decodificador.More() {
		return errors.New("JSON con contenido adicional")
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return errors.New("JSON con contenido adicional")
	}
	return nil
}
