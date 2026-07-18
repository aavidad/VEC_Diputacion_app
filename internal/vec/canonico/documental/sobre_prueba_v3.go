package documental

import (
	"fmt"
	"io"
	"log/slog"
)

// SobrePruebaAtestacionDespachoV3 mantiene opacos y copiados defensivamente el
// mensaje y su firma. Es material nominal: no representa una comprobacion KMS.
type SobrePruebaAtestacionDespachoV3 struct {
	datos DatosPruebaAtestacionDespachoV3
}

func NuevoSobrePruebaAtestacionDespachoV3(
	algoritmo, audiencia, contexto, claveGestionadaRef string,
	revisionClaveGestionada uint64,
	evidenciaOperacionRef string,
	mensajeCanonico, sobreCriptografico []byte,
) (SobrePruebaAtestacionDespachoV3, error) {
	datos := DatosPruebaAtestacionDespachoV3{
		Algoritmo: algoritmo, Audiencia: audiencia, Contexto: contexto,
		ClaveGestionadaRef: claveGestionadaRef, RevisionClaveGestionada: revisionClaveGestionada,
		EvidenciaOperacionRef: evidenciaOperacionRef,
		MensajeCanonico:       append([]byte(nil), mensajeCanonico...),
		SobreCriptografico:    append([]byte(nil), sobreCriptografico...),
		HuellaMensajeSHA256:   HuellaBytesSHA256(mensajeCanonico),
		HuellaSobreSHA256:     HuellaBytesSHA256(sobreCriptografico),
	}
	sobre := RestaurarSobrePruebaAtestacionDespachoV3(datos)
	if sobre.Validar() != nil {
		return SobrePruebaAtestacionDespachoV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	return sobre, nil
}

// RestaurarSobrePruebaAtestacionDespachoV3 no promueve autoridad. Permite que
// el puerto restaure datos y que Validar detecte cualquier alteracion.
func RestaurarSobrePruebaAtestacionDespachoV3(
	datos DatosPruebaAtestacionDespachoV3,
) SobrePruebaAtestacionDespachoV3 {
	datos.MensajeCanonico = append([]byte(nil), datos.MensajeCanonico...)
	datos.SobreCriptografico = append([]byte(nil), datos.SobreCriptografico...)
	return SobrePruebaAtestacionDespachoV3{datos: datos}
}

func (s SobrePruebaAtestacionDespachoV3) Validar() error {
	if !s.datos.Validar() {
		return ErrOrdenDespachoDocumentalV3Invalida
	}
	return nil
}

func (s SobrePruebaAtestacionDespachoV3) Perfil() (
	algoritmo, audiencia, contexto, claveGestionadaRef string,
	revisionClaveGestionada uint64,
	err error,
) {
	if s.Validar() != nil {
		return "", "", "", "", 0, ErrOrdenDespachoDocumentalV3Invalida
	}
	return s.datos.Algoritmo, s.datos.Audiencia, s.datos.Contexto,
		s.datos.ClaveGestionadaRef, s.datos.RevisionClaveGestionada, nil
}

func (s SobrePruebaAtestacionDespachoV3) MensajeCanonico() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrOrdenDespachoDocumentalV3Invalida
	}
	return append([]byte(nil), s.datos.MensajeCanonico...), nil
}

func (s SobrePruebaAtestacionDespachoV3) SobreCriptografico() ([]byte, error) {
	if s.Validar() != nil {
		return nil, ErrOrdenDespachoDocumentalV3Invalida
	}
	return append([]byte(nil), s.datos.SobreCriptografico...), nil
}

func (s SobrePruebaAtestacionDespachoV3) EvidenciaOperacionRef() (string, error) {
	if s.Validar() != nil {
		return "", ErrOrdenDespachoDocumentalV3Invalida
	}
	return s.datos.EvidenciaOperacionRef, nil
}

func (s SobrePruebaAtestacionDespachoV3) HuellasSHA256() (mensaje, sobre string, err error) {
	if s.Validar() != nil {
		return "", "", ErrOrdenDespachoDocumentalV3Invalida
	}
	return s.datos.HuellaMensajeSHA256, s.datos.HuellaSobreSHA256, nil
}

func (s SobrePruebaAtestacionDespachoV3) HuellaSHA256() string {
	return s.datos.HuellaSHA256()
}

func (s SobrePruebaAtestacionDespachoV3) Datos() (DatosPruebaAtestacionDespachoV3, error) {
	if s.Validar() != nil {
		return DatosPruebaAtestacionDespachoV3{}, ErrOrdenDespachoDocumentalV3Invalida
	}
	datos := s.datos
	datos.MensajeCanonico = append([]byte(nil), s.datos.MensajeCanonico...)
	datos.SobreCriptografico = append([]byte(nil), s.datos.SobreCriptografico...)
	return datos, nil
}

func (SobrePruebaAtestacionDespachoV3) String() string {
	return "[PRUEBA-CRUDA-ATESTACION-DESPACHO-V3-NOMINAL-NO-AUTORITATIVA-REDACTADA]"
}
func (s SobrePruebaAtestacionDespachoV3) GoString() string { return s.String() }
func (s SobrePruebaAtestacionDespachoV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SobrePruebaAtestacionDespachoV3) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SobrePruebaAtestacionDespachoV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobrePruebaAtestacionDespachoV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SobrePruebaAtestacionDespachoV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobrePruebaAtestacionDespachoV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (SobrePruebaAtestacionDespachoV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SobrePruebaAtestacionDespachoV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
