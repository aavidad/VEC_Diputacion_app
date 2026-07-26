// Package postgresimportacionconvoca implementa persistencia durable del
// staging validado de Convoca sin almacenar datos personales en claro.
package postgresimportacionconvoca

import (
	"context"
	"crypto/subtle"
	"errors"
	"regexp"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const (
	// EsquemaProteccionStagingV1 identifica el contrato con un protector
	// respaldado por KMS/HSM. Clave y algoritmo no pertenecen a PostgreSQL.
	EsquemaProteccionStagingV1 = "vec.bolsa.importacion-convoca.proteccion-staging.v1"

	maximoFilasStaging         = 100_001
	maximoBytesCifradosPorFila = 128 * 1024
	maximoBytesProtegidosLote  = 24 * 1024 * 1024
	maximoBytesJSONFilas       = 72 * 1024 * 1024
	maximoBytesJSONActa        = 24 * 1024 * 1024
)

var (
	ErrProtectorRequerido      = errors.New("bolsa importacion Convoca: protector de staging requerido")
	ErrProteccionNoDisponible  = errors.New("bolsa importacion Convoca: proteccion de staging no disponible")
	ErrMaterialNoConfiable     = errors.New("bolsa importacion Convoca: material protegido no confiable")
	patronReferenciaProteccion = regexp.MustCompile(`^[a-z][a-z0-9._:/-]{2,255}$`)
)

// SolicitudProteccionStaging exige usar ImportacionRef, huella, esquema y
// número de fila como AAD canónico. La derivación ciega usa una clave distinta
// de la de cifrado.
type SolicitudProteccionStaging struct {
	ImportacionRef      string
	HuellaFicheroSHA256 string
	Esquema             dominio.EsquemaExportacion
	Filas               []dominio.FilaAceptada
}

type FilaStagingProtegida struct {
	Numero                        int
	EsquemaProteccion             string
	ClaveRef                      string
	ClaveDerivacionRef            string
	ClaveAtestacionRef            string
	Nonce                         []byte
	ContenidoCifrado              []byte
	DerivacionDocumentoHMACSHA256 []byte
	AtestacionFilaHMACSHA256      []byte
}

type ResultadoProteccionStaging struct {
	Filas []FilaStagingProtegida
}

type SolicitudRecuperacionStaging struct {
	ImportacionRef      string
	HuellaFicheroSHA256 string
	Esquema             dominio.EsquemaExportacion
	Filas               []FilaStagingProtegida
}

// ProtectorStagingConvoca es sustituible. Una implementación productiva debe
// usar AEAD, claves versionadas en KMS/HSM y descifrado de versiones históricas.
type ProtectorStagingConvoca interface {
	ProtegerStaging(context.Context, SolicitudProteccionStaging) (ResultadoProteccionStaging, error)
	RecuperarStaging(context.Context, SolicitudRecuperacionStaging) ([]dominio.FilaAceptada, error)
}

func clonarFilasDominio(origen []dominio.FilaAceptada) []dominio.FilaAceptada {
	lote := dominio.ClonarLote(dominio.LoteValidado{Aceptadas: origen})
	return lote.Aceptadas
}

func clonarFilasProtegidas(origen []FilaStagingProtegida) []FilaStagingProtegida {
	destino := make([]FilaStagingProtegida, len(origen))
	for i := range origen {
		destino[i] = origen[i]
		destino[i].Nonce = append([]byte(nil), origen[i].Nonce...)
		destino[i].ContenidoCifrado = append([]byte(nil), origen[i].ContenidoCifrado...)
		destino[i].DerivacionDocumentoHMACSHA256 = append(
			[]byte(nil), origen[i].DerivacionDocumentoHMACSHA256...,
		)
		destino[i].AtestacionFilaHMACSHA256 = append(
			[]byte(nil), origen[i].AtestacionFilaHMACSHA256...,
		)
	}
	return destino
}

func validarCorrespondenciaProteccion(
	aceptadas []dominio.FilaAceptada,
	protegidas []FilaStagingProtegida,
) error {
	if len(aceptadas) != len(protegidas) || validarFilasProtegidas(protegidas) != nil {
		return ErrMaterialNoConfiable
	}
	for i := range aceptadas {
		if aceptadas[i].Numero < 2 || aceptadas[i].Numero != protegidas[i].Numero {
			return ErrMaterialNoConfiable
		}
	}
	return nil
}

func validarFilasProtegidas(protegidas []FilaStagingProtegida) error {
	if len(protegidas) > maximoFilasStaging {
		return ErrMaterialNoConfiable
	}
	numeroAnterior := 1
	totalBytes := 0
	for i := range protegidas {
		fila := protegidas[i]
		bytesFila := len(fila.Nonce) + len(fila.ContenidoCifrado) +
			len(fila.DerivacionDocumentoHMACSHA256) +
			len(fila.AtestacionFilaHMACSHA256)
		if fila.Numero <= numeroAnterior ||
			fila.EsquemaProteccion != EsquemaProteccionStagingV1 ||
			!patronReferenciaProteccion.MatchString(fila.ClaveRef) ||
			!patronReferenciaProteccion.MatchString(fila.ClaveDerivacionRef) ||
			!patronReferenciaProteccion.MatchString(fila.ClaveAtestacionRef) ||
			fila.ClaveDerivacionRef == fila.ClaveRef ||
			fila.ClaveAtestacionRef == fila.ClaveRef ||
			fila.ClaveAtestacionRef == fila.ClaveDerivacionRef ||
			len(fila.Nonce) < 12 || len(fila.Nonce) > 64 ||
			len(fila.ContenidoCifrado) < 16 ||
			len(fila.ContenidoCifrado) > maximoBytesCifradosPorFila ||
			len(fila.DerivacionDocumentoHMACSHA256) != 32 ||
			todosCeros(fila.DerivacionDocumentoHMACSHA256) ||
			len(fila.AtestacionFilaHMACSHA256) != 32 ||
			todosCeros(fila.AtestacionFilaHMACSHA256) ||
			bytesFila > maximoBytesProtegidosLote-totalBytes {
			return ErrMaterialNoConfiable
		}
		numeroAnterior = fila.Numero
		totalBytes += bytesFila
	}
	return nil
}

func validarCorrespondenciaRecuperada(
	protegidas []FilaStagingProtegida,
	recuperadas []dominio.FilaAceptada,
	esquema dominio.EsquemaExportacion,
) error {
	if len(protegidas) != len(recuperadas) {
		return ErrMaterialNoConfiable
	}
	for i := range recuperadas {
		if recuperadas[i].Numero != protegidas[i].Numero ||
			recuperadas[i].Esquema != esquema {
			return ErrMaterialNoConfiable
		}
	}
	return nil
}

func todosCeros(valor []byte) bool {
	if len(valor) == 0 {
		return true
	}
	return subtle.ConstantTimeCompare(valor, make([]byte, len(valor))) == 1
}
