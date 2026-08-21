package domain

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sort"
)

const (
	EsquemaBorradorInformeJuridicoV1        = "vec.dipgra.contratacion-temporal.informe-juridico.borrador"
	VersionEsquemaBorradorInformeJuridicoV1 = uint16(1)

	maximoReferenciasNormativasInformeJuridico = 64
	maximoAnexosInformeJuridico                = 64
	maximoBytesReferenciasInformeJuridico      = 20_480
	maximoEnteroSeguroInformeJuridico          = uint64(9_007_199_254_740_991)
)

var ErrBorradorInformeJuridicoInvalido = errors.New(
	"contratacion temporal: borrador de informe juridico invalido",
)

// CanonInformeJuridico fija el esquema binario y evita aceptar versiones por
// aproximacion. Una nueva version requiere otro constructor explicito.
type CanonInformeJuridico struct {
	Esquema        string `json:"esquema"`
	VersionEsquema uint16 `json:"version_esquema"`
}

func CanonBorradorInformeJuridicoV1() CanonInformeJuridico {
	return CanonInformeJuridico{
		Esquema:        EsquemaBorradorInformeJuridicoV1,
		VersionEsquema: VersionEsquemaBorradorInformeJuridicoV1,
	}
}

type ReferenciaPlantillaInformeJuridico struct {
	PlantillaRef string `json:"plantilla_ref"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type ReferenciaNormativaInformeJuridico struct {
	NormaRef     string `json:"norma_ref"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type AnexoDocumentalInformeJuridico struct {
	DocumentoRef     string `json:"documento_ref"`
	VersionDocumento uint64 `json:"version_documento"`
	HuellaSHA256     string `json:"huella_sha256"`
}

// DatosBorradorInformeJuridico solo liga identidades gobernadas. No contiene
// texto juridico, conclusiones, aprobaciones ni estados de firma.
type DatosBorradorInformeJuridico struct {
	Canon                     CanonInformeJuridico                 `json:"canon"`
	ExpedienteRef             string                               `json:"expediente_ref"`
	VersionEsperadaExpediente uint64                               `json:"version_esperada_expediente"`
	Plantilla                 ReferenciaPlantillaInformeJuridico   `json:"plantilla"`
	ReferenciasNormativas     []ReferenciaNormativaInformeJuridico `json:"referencias_normativas"`
	Anexos                    []AnexoDocumentalInformeJuridico     `json:"anexos"`
}

type EstadoBorradorInformeJuridico struct {
	DatosBorradorInformeJuridico
	HuellaSHA256 string `json:"huella_sha256"`
}

// BorradorInformeJuridico es inmutable desde fuera del paquete.
type BorradorInformeJuridico struct{ estado EstadoBorradorInformeJuridico }

func NuevoBorradorInformeJuridico(
	datos DatosBorradorInformeJuridico,
) (BorradorInformeJuridico, error) {
	normalizados, err := normalizarDatosBorradorInformeJuridico(datos)
	if err != nil {
		return BorradorInformeJuridico{}, err
	}
	estado := EstadoBorradorInformeJuridico{DatosBorradorInformeJuridico: normalizados}
	estado.HuellaSHA256 = huellaInformeJuridico(materialCanonicoInformeJuridico(normalizados))
	return BorradorInformeJuridico{estado: estado}, nil
}

func RestaurarBorradorInformeJuridico(
	estado EstadoBorradorInformeJuridico,
) (BorradorInformeJuridico, error) {
	restaurado, err := NuevoBorradorInformeJuridico(estado.DatosBorradorInformeJuridico)
	if err != nil || !huellasInformeJuridicoIguales(
		estado.HuellaSHA256,
		restaurado.estado.HuellaSHA256,
	) {
		return BorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
	}
	return restaurado, nil
}

func (b BorradorInformeJuridico) Validar() error {
	_, err := RestaurarBorradorInformeJuridico(b.Estado())
	return err
}

func (b BorradorInformeJuridico) Estado() EstadoBorradorInformeJuridico {
	estado := b.estado
	estado.ReferenciasNormativas = append(
		[]ReferenciaNormativaInformeJuridico(nil), estado.ReferenciasNormativas...,
	)
	estado.Anexos = append([]AnexoDocumentalInformeJuridico(nil), estado.Anexos...)
	return estado
}

func (b BorradorInformeJuridico) SerializarCanonico() ([]byte, error) {
	if b.Validar() != nil {
		return nil, ErrBorradorInformeJuridicoInvalido
	}
	return materialCanonicoInformeJuridico(b.estado.DatosBorradorInformeJuridico), nil
}

func (b BorradorInformeJuridico) HuellaSHA256() string {
	if b.Validar() != nil {
		return ""
	}
	return b.estado.HuellaSHA256
}

func (b BorradorInformeJuridico) VerificarHuellaSHA256(huella string) bool {
	return b.Validar() == nil && huellasInformeJuridicoIguales(b.estado.HuellaSHA256, huella)
}

func normalizarDatosBorradorInformeJuridico(
	datos DatosBorradorInformeJuridico,
) (DatosBorradorInformeJuridico, error) {
	if datos.Canon != CanonBorradorInformeJuridicoV1() ||
		!versionInformeJuridicoValida(datos.VersionEsperadaExpediente) ||
		len(datos.ReferenciasNormativas) == 0 ||
		len(datos.ReferenciasNormativas) > maximoReferenciasNormativasInformeJuridico ||
		len(datos.Anexos) > maximoAnexosInformeJuridico {
		return DatosBorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
	}
	// El presupuesto se descuenta antes de validadores, copias y ordenaciones:
	// una entrada sobredimensionada no debe provocar reservas proporcionales.
	restante := maximoBytesReferenciasInformeJuridico
	if !consumirBytesInformeJuridico(&restante, len(datos.ExpedienteRef)) ||
		!consumirBytesInformeJuridico(&restante, len(datos.Plantilla.PlantillaRef)) ||
		!consumirBytesInformeJuridico(&restante, len(datos.Plantilla.HuellaSHA256)) {
		return DatosBorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
	}
	for _, referencia := range datos.ReferenciasNormativas {
		if !consumirBytesInformeJuridico(&restante, len(referencia.NormaRef)) ||
			!consumirBytesInformeJuridico(&restante, len(referencia.HuellaSHA256)) {
			return DatosBorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
		}
	}
	for _, anexo := range datos.Anexos {
		if !consumirBytesInformeJuridico(&restante, len(anexo.DocumentoRef)) ||
			!consumirBytesInformeJuridico(&restante, len(anexo.HuellaSHA256)) {
			return DatosBorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
		}
	}
	if !referenciaValida(datos.ExpedienteRef) ||
		!referenciaPlantillaInformeJuridicoValida(datos.Plantilla) {
		return DatosBorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
	}
	for _, referencia := range datos.ReferenciasNormativas {
		if !referenciaNormativaInformeJuridicoValida(referencia) {
			return DatosBorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
		}
	}
	for _, anexo := range datos.Anexos {
		if !anexoDocumentalInformeJuridicoValido(anexo) {
			return DatosBorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
		}
	}
	if tieneDuplicadosInformeJuridico(datos) {
		return DatosBorradorInformeJuridico{}, ErrBorradorInformeJuridicoInvalido
	}
	normalizados := datos
	normalizados.ReferenciasNormativas = append(
		[]ReferenciaNormativaInformeJuridico(nil), datos.ReferenciasNormativas...,
	)
	normalizados.Anexos = append([]AnexoDocumentalInformeJuridico(nil), datos.Anexos...)
	sort.Slice(normalizados.ReferenciasNormativas, func(i, j int) bool {
		return normalizados.ReferenciasNormativas[i].NormaRef <
			normalizados.ReferenciasNormativas[j].NormaRef
	})
	sort.Slice(normalizados.Anexos, func(i, j int) bool {
		if normalizados.Anexos[i].DocumentoRef != normalizados.Anexos[j].DocumentoRef {
			return normalizados.Anexos[i].DocumentoRef < normalizados.Anexos[j].DocumentoRef
		}
		return normalizados.Anexos[i].VersionDocumento < normalizados.Anexos[j].VersionDocumento
	})
	return normalizados, nil
}

func consumirBytesInformeJuridico(restante *int, longitud int) bool {
	if longitud > *restante {
		return false
	}
	*restante -= longitud
	return true
}

func tieneDuplicadosInformeJuridico(datos DatosBorradorInformeJuridico) bool {
	for i, referencia := range datos.ReferenciasNormativas {
		for j := 0; j < i; j++ {
			if referencia.NormaRef == datos.ReferenciasNormativas[j].NormaRef {
				return true
			}
		}
	}
	for i, anexo := range datos.Anexos {
		for j := 0; j < i; j++ {
			if anexo.DocumentoRef == datos.Anexos[j].DocumentoRef &&
				anexo.VersionDocumento == datos.Anexos[j].VersionDocumento {
				return true
			}
		}
	}
	return false
}

func referenciaPlantillaInformeJuridicoValida(r ReferenciaPlantillaInformeJuridico) bool {
	return referenciaValida(r.PlantillaRef) && versionInformeJuridicoValida(r.Version) &&
		huellaValida(r.HuellaSHA256)
}

func referenciaNormativaInformeJuridicoValida(r ReferenciaNormativaInformeJuridico) bool {
	return referenciaValida(r.NormaRef) && versionInformeJuridicoValida(r.Version) &&
		huellaValida(r.HuellaSHA256)
}

func anexoDocumentalInformeJuridicoValido(a AnexoDocumentalInformeJuridico) bool {
	return referenciaValida(a.DocumentoRef) && versionInformeJuridicoValida(a.VersionDocumento) &&
		huellaValida(a.HuellaSHA256)
}

func versionInformeJuridicoValida(version uint64) bool {
	return version > 0 && version <= maximoEnteroSeguroInformeJuridico
}

func materialCanonicoInformeJuridico(datos DatosBorradorInformeJuridico) []byte {
	e := &escritorCanonInformeJuridico{}
	e.cadena(datos.Canon.Esquema)
	e.entero16(datos.Canon.VersionEsquema)
	e.cadena(datos.ExpedienteRef)
	e.entero64(datos.VersionEsperadaExpediente)
	e.cadena(datos.Plantilla.PlantillaRef)
	e.entero64(datos.Plantilla.Version)
	e.cadena(datos.Plantilla.HuellaSHA256)
	e.entero16(uint16(len(datos.ReferenciasNormativas)))
	for _, referencia := range datos.ReferenciasNormativas {
		e.cadena(referencia.NormaRef)
		e.entero64(referencia.Version)
		e.cadena(referencia.HuellaSHA256)
	}
	e.entero16(uint16(len(datos.Anexos)))
	for _, anexo := range datos.Anexos {
		e.cadena(anexo.DocumentoRef)
		e.entero64(anexo.VersionDocumento)
		e.cadena(anexo.HuellaSHA256)
	}
	return append([]byte(nil), e.Bytes()...)
}

type escritorCanonInformeJuridico struct{ bytes.Buffer }

func (e *escritorCanonInformeJuridico) cadena(valor string) {
	var longitud [4]byte
	binary.BigEndian.PutUint32(longitud[:], uint32(len(valor)))
	e.Write(longitud[:])
	e.WriteString(valor)
}

func (e *escritorCanonInformeJuridico) entero16(valor uint16) {
	var contenido [2]byte
	binary.BigEndian.PutUint16(contenido[:], valor)
	e.Write(contenido[:])
}

func (e *escritorCanonInformeJuridico) entero64(valor uint64) {
	var contenido [8]byte
	binary.BigEndian.PutUint64(contenido[:], valor)
	e.Write(contenido[:])
}

func huellaInformeJuridico(material []byte) string {
	suma := sha256.Sum256(material)
	return hex.EncodeToString(suma[:])
}

func huellasInformeJuridicoIguales(primera, segunda string) bool {
	return huellaValida(primera) && huellaValida(segunda) &&
		subtle.ConstantTimeCompare([]byte(primera), []byte(segunda)) == 1
}
