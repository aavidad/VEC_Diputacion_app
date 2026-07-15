package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrSobreCriptograficoDocumentalCrudoV4Invalido  = errors.New("vec: sobre criptografico documental crudo v4 invalido")
	ErrPruebaCriptograficaDocumentalCrudaV4Invalida = errors.New("vec: prueba criptografica documental cruda v4 invalida")
	ErrSerializacionPruebaCriptograficaCrudaV4      = errors.New("vec: serializacion de prueba criptografica documental cruda v4 prohibida")
)

const (
	minimoBytesSobreCriptograficoDocumentalCrudoV4 = 16
	// COSE deja un margen estricto sobre el payload PDP maximo para tag,
	// cabeceras y firma. El nucleo aplica despues el limite por audiencia.
	maximoBytesSobreCriptograficoDocumentalCrudoV4 = domain.TamanoMaximoMensajeAtestacionAutorizacionV1 + 4*1024
)

// SobreCriptograficoDocumentalCrudoV4 transporta los bytes opacos de un
// COSE_Sign1. Solo acredita limites y ausencia de alias mutables: no concede
// autoridad, no interpreta cabeceras y no verifica ninguna firma.
type SobreCriptograficoDocumentalCrudoV4 struct {
	coseSign1         []byte
	huellaSobreSHA256 string
}

func NuevoSobreCriptograficoDocumentalCrudoV4(
	coseSign1 []byte,
) (SobreCriptograficoDocumentalCrudoV4, error) {
	sobre := SobreCriptograficoDocumentalCrudoV4{
		coseSign1: append([]byte(nil), coseSign1...),
	}
	sobre.huellaSobreSHA256 = huellaSobreCriptograficoDocumentalCrudoV4(sobre.coseSign1)
	if sobre.ValidarSintaxis() != nil {
		return SobreCriptograficoDocumentalCrudoV4{}, ErrSobreCriptograficoDocumentalCrudoV4Invalido
	}
	return sobre, nil
}

// ValidarSintaxis comprueba exclusivamente el contenedor crudo. Su exito no
// significa que el contenido sea COSE valido ni que su firma sea confiable.
func (s SobreCriptograficoDocumentalCrudoV4) ValidarSintaxis() error {
	if len(s.coseSign1) < minimoBytesSobreCriptograficoDocumentalCrudoV4 ||
		len(s.coseSign1) > maximoBytesSobreCriptograficoDocumentalCrudoV4 ||
		bytesCriptograficosDocumentalesV4Nulos(s.coseSign1) ||
		len(s.huellaSobreSHA256) != sha256.Size*2 ||
		s.huellaSobreSHA256 != huellaSobreCriptograficoDocumentalCrudoV4(s.coseSign1) {
		return ErrSobreCriptograficoDocumentalCrudoV4Invalido
	}
	return nil
}

// COSESign1 entrega una copia para su verificacion local. Es la unica salida
// binaria deliberada; los serializadores generales permanecen bloqueados.
func (s SobreCriptograficoDocumentalCrudoV4) COSESign1() ([]byte, error) {
	if s.ValidarSintaxis() != nil {
		return nil, ErrSobreCriptograficoDocumentalCrudoV4Invalido
	}
	return append([]byte(nil), s.coseSign1...), nil
}

func (s SobreCriptograficoDocumentalCrudoV4) HuellaSHA256() (string, error) {
	if s.ValidarSintaxis() != nil {
		return "", ErrSobreCriptograficoDocumentalCrudoV4Invalido
	}
	return s.huellaSobreSHA256, nil
}

func (SobreCriptograficoDocumentalCrudoV4) String() string {
	return "[SOBRE-CRIPTOGRAFICO-DOCUMENTAL-CRUDO-V4-REDACTADO]"
}

func (s SobreCriptograficoDocumentalCrudoV4) GoString() string { return s.String() }

func (s SobreCriptograficoDocumentalCrudoV4) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

func (s SobreCriptograficoDocumentalCrudoV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (SobreCriptograficoDocumentalCrudoV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}

func (*SobreCriptograficoDocumentalCrudoV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}

func (SobreCriptograficoDocumentalCrudoV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}

func (*SobreCriptograficoDocumentalCrudoV4) UnmarshalText([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}

func (SobreCriptograficoDocumentalCrudoV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}

func (*SobreCriptograficoDocumentalCrudoV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}

// Los cuatro tipos siguientes son nominales: evitan intercambiar por error
// pruebas de protocolos distintos. Ninguno expone metodos de autorizacion o
// verificacion; todos siguen siendo transporte criptografico no confiable.

type PruebaCrudaReciboComponenteDocumentalV4 struct {
	sobre SobreCriptograficoDocumentalCrudoV4
}

func NuevaPruebaCrudaReciboComponenteDocumentalV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (PruebaCrudaReciboComponenteDocumentalV4, error) {
	if sobre.ValidarSintaxis() != nil {
		return PruebaCrudaReciboComponenteDocumentalV4{}, ErrPruebaCriptograficaDocumentalCrudaV4Invalida
	}
	return PruebaCrudaReciboComponenteDocumentalV4{sobre: sobre}, nil
}

func (p PruebaCrudaReciboComponenteDocumentalV4) ValidarSintaxis() error {
	return validarPruebaCriptograficaDocumentalCrudaV4(p.sobre)
}

func (p PruebaCrudaReciboComponenteDocumentalV4) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
) {
	return sobreDePruebaCriptograficaDocumentalCrudaV4(p.sobre)
}

func (PruebaCrudaReciboComponenteDocumentalV4) String() string {
	return "[PRUEBA-CRUDA-RECIBO-COMPONENTE-DOCUMENTAL-V4-REDACTADA]"
}

func (p PruebaCrudaReciboComponenteDocumentalV4) GoString() string { return p.String() }
func (p PruebaCrudaReciboComponenteDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (p PruebaCrudaReciboComponenteDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (PruebaCrudaReciboComponenteDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*PruebaCrudaReciboComponenteDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}
func (PruebaCrudaReciboComponenteDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*PruebaCrudaReciboComponenteDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}
func (PruebaCrudaReciboComponenteDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*PruebaCrudaReciboComponenteDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}

type PruebaCrudaTokenCercadoDocumentalV4 struct {
	sobre SobreCriptograficoDocumentalCrudoV4
}

func NuevaPruebaCrudaTokenCercadoDocumentalV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (PruebaCrudaTokenCercadoDocumentalV4, error) {
	if sobre.ValidarSintaxis() != nil {
		return PruebaCrudaTokenCercadoDocumentalV4{}, ErrPruebaCriptograficaDocumentalCrudaV4Invalida
	}
	return PruebaCrudaTokenCercadoDocumentalV4{sobre: sobre}, nil
}

func (p PruebaCrudaTokenCercadoDocumentalV4) ValidarSintaxis() error {
	return validarPruebaCriptograficaDocumentalCrudaV4(p.sobre)
}

func (p PruebaCrudaTokenCercadoDocumentalV4) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
) {
	return sobreDePruebaCriptograficaDocumentalCrudaV4(p.sobre)
}

func (PruebaCrudaTokenCercadoDocumentalV4) String() string {
	return "[PRUEBA-CRUDA-TOKEN-CERCADO-DOCUMENTAL-V4-REDACTADA]"
}

func (p PruebaCrudaTokenCercadoDocumentalV4) GoString() string { return p.String() }
func (p PruebaCrudaTokenCercadoDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (p PruebaCrudaTokenCercadoDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (PruebaCrudaTokenCercadoDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*PruebaCrudaTokenCercadoDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}
func (PruebaCrudaTokenCercadoDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*PruebaCrudaTokenCercadoDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}
func (PruebaCrudaTokenCercadoDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*PruebaCrudaTokenCercadoDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}

type FirmaCrudaEvidenciaDocumentalV4 struct {
	sobre SobreCriptograficoDocumentalCrudoV4
}

func NuevaFirmaCrudaEvidenciaDocumentalV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (FirmaCrudaEvidenciaDocumentalV4, error) {
	if sobre.ValidarSintaxis() != nil {
		return FirmaCrudaEvidenciaDocumentalV4{}, ErrPruebaCriptograficaDocumentalCrudaV4Invalida
	}
	return FirmaCrudaEvidenciaDocumentalV4{sobre: sobre}, nil
}

func (p FirmaCrudaEvidenciaDocumentalV4) ValidarSintaxis() error {
	return validarPruebaCriptograficaDocumentalCrudaV4(p.sobre)
}

func (p FirmaCrudaEvidenciaDocumentalV4) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
) {
	return sobreDePruebaCriptograficaDocumentalCrudaV4(p.sobre)
}

func (FirmaCrudaEvidenciaDocumentalV4) String() string {
	return "[FIRMA-CRUDA-EVIDENCIA-DOCUMENTAL-V4-REDACTADA]"
}

func (p FirmaCrudaEvidenciaDocumentalV4) GoString() string { return p.String() }
func (p FirmaCrudaEvidenciaDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (p FirmaCrudaEvidenciaDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (FirmaCrudaEvidenciaDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*FirmaCrudaEvidenciaDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}
func (FirmaCrudaEvidenciaDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*FirmaCrudaEvidenciaDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}
func (FirmaCrudaEvidenciaDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*FirmaCrudaEvidenciaDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}

type AtestacionCrudaReconciliacionDocumentalV4 struct {
	sobre SobreCriptograficoDocumentalCrudoV4
}

func NuevaAtestacionCrudaReconciliacionDocumentalV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (AtestacionCrudaReconciliacionDocumentalV4, error) {
	if sobre.ValidarSintaxis() != nil {
		return AtestacionCrudaReconciliacionDocumentalV4{}, ErrPruebaCriptograficaDocumentalCrudaV4Invalida
	}
	return AtestacionCrudaReconciliacionDocumentalV4{sobre: sobre}, nil
}

func (p AtestacionCrudaReconciliacionDocumentalV4) ValidarSintaxis() error {
	return validarPruebaCriptograficaDocumentalCrudaV4(p.sobre)
}

func (p AtestacionCrudaReconciliacionDocumentalV4) SobreCrudo() (
	SobreCriptograficoDocumentalCrudoV4,
	error,
) {
	return sobreDePruebaCriptograficaDocumentalCrudaV4(p.sobre)
}

func (AtestacionCrudaReconciliacionDocumentalV4) String() string {
	return "[ATESTACION-CRUDA-RECONCILIACION-DOCUMENTAL-V4-REDACTADA]"
}

func (p AtestacionCrudaReconciliacionDocumentalV4) GoString() string { return p.String() }
func (p AtestacionCrudaReconciliacionDocumentalV4) LogValue() slog.Value {
	return slog.StringValue(p.String())
}
func (p AtestacionCrudaReconciliacionDocumentalV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (AtestacionCrudaReconciliacionDocumentalV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*AtestacionCrudaReconciliacionDocumentalV4) UnmarshalJSON([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}
func (AtestacionCrudaReconciliacionDocumentalV4) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*AtestacionCrudaReconciliacionDocumentalV4) UnmarshalText([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}
func (AtestacionCrudaReconciliacionDocumentalV4) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionPruebaCriptograficaCrudaV4
}
func (*AtestacionCrudaReconciliacionDocumentalV4) UnmarshalBinary([]byte) error {
	return ErrSerializacionPruebaCriptograficaCrudaV4
}

func validarPruebaCriptograficaDocumentalCrudaV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) error {
	if sobre.ValidarSintaxis() != nil {
		return ErrPruebaCriptograficaDocumentalCrudaV4Invalida
	}
	return nil
}

func sobreDePruebaCriptograficaDocumentalCrudaV4(
	sobre SobreCriptograficoDocumentalCrudoV4,
) (SobreCriptograficoDocumentalCrudoV4, error) {
	if validarPruebaCriptograficaDocumentalCrudaV4(sobre) != nil {
		return SobreCriptograficoDocumentalCrudoV4{}, ErrPruebaCriptograficaDocumentalCrudaV4Invalida
	}
	return sobre, nil
}

func huellaSobreCriptograficoDocumentalCrudoV4(contenido []byte) string {
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:])
}

func bytesCriptograficosDocumentalesV4Nulos(contenido []byte) bool {
	if len(contenido) == 0 {
		return true
	}
	for _, valor := range contenido {
		if valor != 0 {
			return false
		}
	}
	return true
}
