// Package postgresimportacionconvoca implementa la persistencia durable del
// staging validado de Convoca. Los datos de cada fila permanecen cifrados y
// el repositorio no conoce claves maestras.
package postgresimportacionconvoca

import (
	"context"
	"crypto/subtle"
	"errors"
	"regexp"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

const (
	// EsquemaProteccionStagingV1 identifica el contrato entre este adaptador y
	// un protector conectado a KMS/HSM. La clave y el algoritmo concretos no
	// forman parte del repositorio PostgreSQL.
	EsquemaProteccionStagingV1 = "vec.bolsa.importacion-convoca.proteccion-staging.v1"

	maximoFilasStaging         = 100_001
	maximoBytesCifradosPorFila = 128 * 1024
)

var (
	ErrProtectorRequerido      = errors.New("bolsa importacion Convoca: protector de staging requerido")
	ErrProteccionNoDisponible  = errors.New("bolsa importacion Convoca: proteccion de staging no disponible")
	ErrMaterialNoConfiable     = errors.New("bolsa importacion Convoca: material protegido no confiable")
	patronReferenciaProteccion = regexp.MustCompile(`^[a-z][a-z0-9._:/-]{2,255}$`)
)

// SolicitudProteccionStaging entrega una copia de las filas ya validadas. El
// protector debe usar ImportacionRef, HuellaFicheroSHA256, Esquema y Numero
// como AAD canonico; debe emplear AEAD, una clave versionada y un DEK nuevo
// por lote o fila. La derivacion ciega es HMAC-SHA-256 del documento
// enmascarado bajo una clave distinta de la de cifrado.
type SolicitudProteccionStaging struct {
	ImportacionRef      string
	HuellaFicheroSHA256 string
	Esquema             dominio.EsquemaExportacion
	Filas               []dominio.FilaAceptada
}

// FilaStagingProtegida es un sobre opaco. No incluye nombres, documento,
// turno, meritos ni puntuaciones en claro.
type FilaStagingProtegida struct {
	Numero                        int
	EsquemaProteccion             string
	ClaveRef                      string
	Nonce                         []byte
	ContenidoCifrado              []byte
	DerivacionDocumentoHMACSHA256 []byte
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

// ProtectorStagingConvoca representa un adaptador criptografico sustituible.
// Una implementacion productiva debe respaldarse en KMS/HSM, admitir el
// descifrado con claves historicas y no registrar nunca los claros.
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
	}
	return destino
}

func validarCorrespondenciaProteccion(
	aceptadas []dominio.FilaAceptada,
	protegidas []FilaStagingProtegida,
) error {
	if len(aceptadas) != len(protegidas) || len(protegidas) > maximoFilasStaging {
		return ErrMaterialNoConfiable
	}
	numeros := make(map[int]struct{}, len(aceptadas))
	for i := range aceptadas {
		fila := protegidas[i]
		if aceptadas[i].Numero < 2 || aceptadas[i].Numero != fila.Numero ||
			fila.EsquemaProteccion != EsquemaProteccionStagingV1 ||
			!patronReferenciaProteccion.MatchString(fila.ClaveRef) ||
			len(fila.Nonce) < 12 || len(fila.Nonce) > 64 ||
			len(fila.ContenidoCifrado) < 16 ||
			len(fila.ContenidoCifrado) > maximoBytesCifradosPorFila ||
			len(fila.DerivacionDocumentoHMACSHA256) != 32 ||
			todosCeros(fila.DerivacionDocumentoHMACSHA256) {
			return ErrMaterialNoConfiable
		}
		if _, repetido := numeros[fila.Numero]; repetido {
			return ErrMaterialNoConfiable
		}
		numeros[fila.Numero] = struct{}{}
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
	ceros := make([]byte, len(valor))
	return subtle.ConstantTimeCompare(valor, ceros) == 1
}
