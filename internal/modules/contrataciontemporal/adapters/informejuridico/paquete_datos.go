// Package informejuridico prepara datos canónicos para la capacidad documental
// común. No redacta, genera, firma, aprueba ni publica informes jurídicos.
package informejuridico

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	EsquemaPaqueteDatosInformeJuridicoV1        = "vec.dipgra.contratacion-temporal.informe-juridico.paquete-datos"
	VersionEsquemaPaqueteDatosInformeJuridicoV1 = uint16(1)
	MaximoBytesPaqueteDatosInformeJuridico      = 160 * 1024

	maximoReferenciasNormativasPaquete = 64
	maximoAnexosPaquete                = 64
	maximoBytesReferenciasPaquete      = 20_544
	reservaFijaJSONPaquete             = 16 * 1024
	expansionMaximaJSONPaquete         = 6
)

var ErrPaqueteDatosInformeJuridicoInvalido = errors.New(
	"contratacion temporal: paquete de datos juridicos invalido",
)

type paqueteDatosJSON struct {
	Esquema               string                    `json:"esquema"`
	VersionEsquema        uint16                    `json:"version_esquema"`
	ExpedienteRef         string                    `json:"expediente_ref"`
	VersionExpediente     uint64                    `json:"version_expediente"`
	Plantilla             plantillaJSON             `json:"plantilla"`
	ReferenciasNormativas []referenciaNormativaJSON `json:"referencias_normativas"`
	Anexos                []anexoJSON               `json:"anexos"`
	HuellaBorradorSHA256  string                    `json:"huella_borrador_sha256"`
}

type plantillaJSON struct {
	PlantillaRef string `json:"plantilla_ref"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type referenciaNormativaJSON struct {
	NormaRef     string `json:"norma_ref"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type anexoJSON struct {
	DocumentoRef     string `json:"documento_ref"`
	VersionDocumento uint64 `json:"version_documento"`
	HuellaSHA256     string `json:"huella_sha256"`
}

type datosPaqueteInformeJuridico struct {
	contenidoJSON []byte
	huellaSHA256  string
}

// PaqueteDatosInformeJuridico es un sobre inmutable de datos para el generador
// documental. Su existencia no acredita que se haya generado un informe.
type PaqueteDatosInformeJuridico struct {
	datos *datosPaqueteInformeJuridico
}

// GenerarPaqueteDatos transforma un borrador de referencias validado en JSON
// cerrado y determinista. No consulta anexos ni incorpora su contenido.
func GenerarPaqueteDatos(
	borrador domain.BorradorInformeJuridico,
) (PaqueteDatosInformeJuridico, error) {
	estado := borrador.Estado()
	presupuesto, valido := presupuestoPaqueteDatos(estado)
	if !valido {
		return PaqueteDatosInformeJuridico{}, ErrPaqueteDatosInformeJuridicoInvalido
	}
	restaurado, err := domain.RestaurarBorradorInformeJuridico(estado)
	if err != nil {
		return PaqueteDatosInformeJuridico{}, ErrPaqueteDatosInformeJuridicoInvalido
	}
	estado = restaurado.Estado()

	normativas := make([]referenciaNormativaJSON, len(estado.ReferenciasNormativas))
	for indice, referencia := range estado.ReferenciasNormativas {
		normativas[indice] = referenciaNormativaJSON{
			NormaRef: referencia.NormaRef, Version: referencia.Version,
			HuellaSHA256: referencia.HuellaSHA256,
		}
	}
	sort.Slice(normativas, func(i, j int) bool {
		return normativas[i].NormaRef < normativas[j].NormaRef
	})
	anexos := make([]anexoJSON, len(estado.Anexos))
	for indice, anexo := range estado.Anexos {
		anexos[indice] = anexoJSON{
			DocumentoRef: anexo.DocumentoRef, VersionDocumento: anexo.VersionDocumento,
			HuellaSHA256: anexo.HuellaSHA256,
		}
	}
	sort.Slice(anexos, func(i, j int) bool {
		if anexos[i].DocumentoRef != anexos[j].DocumentoRef {
			return anexos[i].DocumentoRef < anexos[j].DocumentoRef
		}
		return anexos[i].VersionDocumento < anexos[j].VersionDocumento
	})

	contenido, err := codificarJSONCanonico(paqueteDatosJSON{
		Esquema:           EsquemaPaqueteDatosInformeJuridicoV1,
		VersionEsquema:    VersionEsquemaPaqueteDatosInformeJuridicoV1,
		ExpedienteRef:     estado.ExpedienteRef,
		VersionExpediente: estado.VersionEsperadaExpediente,
		Plantilla: plantillaJSON{
			PlantillaRef: estado.Plantilla.PlantillaRef, Version: estado.Plantilla.Version,
			HuellaSHA256: estado.Plantilla.HuellaSHA256,
		},
		ReferenciasNormativas: normativas,
		Anexos:                anexos,
		HuellaBorradorSHA256:  estado.HuellaSHA256,
	})
	if err != nil || len(contenido) == 0 || len(contenido) > presupuesto ||
		len(contenido) > MaximoBytesPaqueteDatosInformeJuridico ||
		!utf8.Valid(contenido) || !json.Valid(contenido) {
		return PaqueteDatosInformeJuridico{}, ErrPaqueteDatosInformeJuridicoInvalido
	}
	suma := sha256.Sum256(contenido)
	paquete := PaqueteDatosInformeJuridico{datos: &datosPaqueteInformeJuridico{
		contenidoJSON: append([]byte(nil), contenido...),
		huellaSHA256:  hex.EncodeToString(suma[:]),
	}}
	if paquete.Validar() != nil {
		return PaqueteDatosInformeJuridico{}, ErrPaqueteDatosInformeJuridicoInvalido
	}
	return paquete, nil
}

func codificarJSONCanonico(datos paqueteDatosJSON) ([]byte, error) {
	return json.Marshal(datos)
}

// Validar comprueba de nuevo la integridad exacta de los bytes retenidos.
func (p PaqueteDatosInformeJuridico) Validar() error {
	if p.datos == nil || len(p.datos.contenidoJSON) == 0 ||
		len(p.datos.contenidoJSON) > MaximoBytesPaqueteDatosInformeJuridico ||
		!utf8.Valid(p.datos.contenidoJSON) || !json.Valid(p.datos.contenidoJSON) {
		return ErrPaqueteDatosInformeJuridicoInvalido
	}
	suma := sha256.Sum256(p.datos.contenidoJSON)
	if p.datos.huellaSHA256 != hex.EncodeToString(suma[:]) {
		return ErrPaqueteDatosInformeJuridicoInvalido
	}
	return nil
}

// ContenidoJSON devuelve bytes de propiedad exclusiva del llamador.
func (p PaqueteDatosInformeJuridico) ContenidoJSON() ([]byte, error) {
	if p.Validar() != nil {
		return nil, ErrPaqueteDatosInformeJuridicoInvalido
	}
	return append([]byte(nil), p.datos.contenidoJSON...), nil
}

// HuellaSHA256 devuelve el SHA-256 hexadecimal de ContenidoJSON.
func (p PaqueteDatosInformeJuridico) HuellaSHA256() (string, error) {
	if p.Validar() != nil {
		return "", ErrPaqueteDatosInformeJuridicoInvalido
	}
	return p.datos.huellaSHA256, nil
}

// presupuestoPaqueteDatos aplica cardinalidad y bytes antes de construir las
// proyecciones o solicitar al codificador su búfer.
func presupuestoPaqueteDatos(
	estado domain.EstadoBorradorInformeJuridico,
) (int, bool) {
	if len(estado.ReferenciasNormativas) == 0 ||
		len(estado.ReferenciasNormativas) > maximoReferenciasNormativasPaquete ||
		len(estado.Anexos) > maximoAnexosPaquete {
		return 0, false
	}
	restante := maximoBytesReferenciasPaquete
	consumir := func(texto string) bool {
		if len(texto) > restante {
			return false
		}
		restante -= len(texto)
		return true
	}
	if !consumir(estado.ExpedienteRef) || !consumir(estado.Plantilla.PlantillaRef) ||
		!consumir(estado.Plantilla.HuellaSHA256) || !consumir(estado.HuellaSHA256) {
		return 0, false
	}
	for _, referencia := range estado.ReferenciasNormativas {
		if !consumir(referencia.NormaRef) || !consumir(referencia.HuellaSHA256) {
			return 0, false
		}
	}
	for _, anexo := range estado.Anexos {
		if !consumir(anexo.DocumentoRef) || !consumir(anexo.HuellaSHA256) {
			return 0, false
		}
	}
	bytesTexto := maximoBytesReferenciasPaquete - restante
	presupuesto := reservaFijaJSONPaquete + bytesTexto*expansionMaximaJSONPaquete
	return presupuesto, presupuesto <= MaximoBytesPaqueteDatosInformeJuridico
}
