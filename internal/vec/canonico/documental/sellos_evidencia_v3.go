package documental

import (
	"fmt"
	"io"
	"log/slog"
	"time"
)

const AudienciaSelloEvidenciaRenderizadoV3 = "vec.documentos.evidencia-renderizado.v3"

// PerfilSelloEvidenciaV3 es una seleccion nominal de clave y audiencia. No
// contiene la clave ni concede capacidad para firmar.
type PerfilSelloEvidenciaV3 struct {
	Algoritmo string
	ClaveID   string
	Audiencia string
}

func NuevoPerfilSelloEvidenciaHMACSHA256V3(claveID string) (PerfilSelloEvidenciaV3, error) {
	perfil := PerfilSelloEvidenciaV3{
		Algoritmo: AlgoritmoHMACSHA256V3, ClaveID: claveID,
		Audiencia: AudienciaSelloEvidenciaRenderizadoV3,
	}
	if perfil.Validar() != nil {
		return PerfilSelloEvidenciaV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return perfil, nil
}

func (p PerfilSelloEvidenciaV3) Validar() error {
	if p.Algoritmo != AlgoritmoHMACSHA256V3 || !ReferenciaEjecucionV3Valida(p.ClaveID) ||
		p.Audiencia != AudienciaSelloEvidenciaRenderizadoV3 {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

func (PerfilSelloEvidenciaV3) String() string {
	return "[PERFIL-SELLO-EVIDENCIA-V3-HMAC-REDACTADO]"
}
func (p PerfilSelloEvidenciaV3) GoString() string { return p.String() }
func (p PerfilSelloEvidenciaV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p PerfilSelloEvidenciaV3) LogValue() slog.Value { return slog.StringValue(p.String()) }
func (PerfilSelloEvidenciaV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PerfilSelloEvidenciaV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (PerfilSelloEvidenciaV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PerfilSelloEvidenciaV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (PerfilSelloEvidenciaV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*PerfilSelloEvidenciaV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// DatosSelloEvidenciaV3 transporta una firma nominal y sus ligaduras.
type DatosSelloEvidenciaV3 struct {
	Algoritmo             string
	ClaveID               string
	Audiencia             string
	HuellaMensajeSHA256   string
	Firma                 []byte
	EvidenciaOperacionRef string
	FirmadoEn             time.Time
}

func (d DatosSelloEvidenciaV3) ValidarPara(
	perfil PerfilSelloEvidenciaV3,
	huellaMensaje string,
) bool {
	return perfil.Validar() == nil && d.Algoritmo == perfil.Algoritmo &&
		d.ClaveID == perfil.ClaveID && d.Audiencia == perfil.Audiencia &&
		SHA256HexadecimalValido(huellaMensaje) && d.HuellaMensajeSHA256 == huellaMensaje &&
		len(d.Firma) == TamanoFirmaHMACSHA256V3 && BytesNoNulos(d.Firma) &&
		ReferenciaEjecucionV3Valida(d.EvidenciaOperacionRef) && InstanteV3Valido(d.FirmadoEn)
}

// HuellaSolicitudVerificacionEvidenciaV3 liga el mensaje firmado, el perfil,
// la firma nominal y su evidencia operativa en el orden historico V3.
func HuellaSolicitudVerificacionEvidenciaV3(
	huellaMensaje string,
	datos DatosSelloEvidenciaV3,
) string {
	return HuellaCamposSHA256V3([]string{
		"vec.documentos.solicitud-verificacion-evidencia.v3", huellaMensaje,
		datos.Algoritmo, datos.ClaveID, datos.Audiencia,
		HuellaBytesSHA256(datos.Firma), datos.EvidenciaOperacionRef,
		datos.FirmadoEn.Format(time.RFC3339Nano),
	})
}

func (DatosSelloEvidenciaV3) String() string {
	return "[DATOS-SELLO-EVIDENCIA-V3-CONFIDENCIALES-REDACTADOS]"
}
func (d DatosSelloEvidenciaV3) GoString() string { return d.String() }
func (d DatosSelloEvidenciaV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}
func (d DatosSelloEvidenciaV3) LogValue() slog.Value { return slog.StringValue(d.String()) }
func (DatosSelloEvidenciaV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSelloEvidenciaV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosSelloEvidenciaV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSelloEvidenciaV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (DatosSelloEvidenciaV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*DatosSelloEvidenciaV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}

// SelloEvidenciaV3 es opaco y autocontenible, pero siempre nominal: solo una
// comprobacion privada con relectura durable puede promover su resultado.
type SelloEvidenciaV3 struct{ datos DatosSelloEvidenciaV3 }

func NuevoSelloEvidenciaV3(
	perfil PerfilSelloEvidenciaV3,
	huellaMensaje string,
	firma []byte,
	evidenciaOperacionRef string,
	firmadoEn time.Time,
) (SelloEvidenciaV3, error) {
	datos := DatosSelloEvidenciaV3{
		Algoritmo: perfil.Algoritmo, ClaveID: perfil.ClaveID, Audiencia: perfil.Audiencia,
		HuellaMensajeSHA256: huellaMensaje, Firma: append([]byte(nil), firma...),
		EvidenciaOperacionRef: evidenciaOperacionRef, FirmadoEn: firmadoEn,
	}
	sello := RestaurarSelloEvidenciaV3(datos)
	if sello.ValidarPara(perfil, huellaMensaje) != nil {
		return SelloEvidenciaV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	return sello, nil
}

func RestaurarSelloEvidenciaV3(datos DatosSelloEvidenciaV3) SelloEvidenciaV3 {
	datos.Firma = append([]byte(nil), datos.Firma...)
	return SelloEvidenciaV3{datos: datos}
}

func (s SelloEvidenciaV3) ValidarPara(perfil PerfilSelloEvidenciaV3, huellaMensaje string) error {
	if !s.datos.ValidarPara(perfil, huellaMensaje) {
		return ErrSelloEvidenciaDocumentalV3Invalido
	}
	return nil
}

func (s SelloEvidenciaV3) Datos() (DatosSelloEvidenciaV3, error) {
	perfil := PerfilSelloEvidenciaV3{
		Algoritmo: s.datos.Algoritmo, ClaveID: s.datos.ClaveID, Audiencia: s.datos.Audiencia,
	}
	if s.ValidarPara(perfil, s.datos.HuellaMensajeSHA256) != nil {
		return DatosSelloEvidenciaV3{}, ErrSelloEvidenciaDocumentalV3Invalido
	}
	datos := s.datos
	datos.Firma = append([]byte(nil), s.datos.Firma...)
	return datos, nil
}

func (s SelloEvidenciaV3) EsCero() bool {
	return s.datos.Algoritmo == "" && s.datos.ClaveID == "" && s.datos.Audiencia == "" &&
		s.datos.HuellaMensajeSHA256 == "" && len(s.datos.Firma) == 0 &&
		s.datos.EvidenciaOperacionRef == "" && s.datos.FirmadoEn.IsZero()
}

func (SelloEvidenciaV3) String() string {
	return "[SELLO-EVIDENCIA-DOCUMENTAL-V3-NOMINAL-NO-AUTORITATIVO-REDACTADO]"
}
func (s SelloEvidenciaV3) GoString() string { return s.String() }
func (s SelloEvidenciaV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SelloEvidenciaV3) LogValue() slog.Value { return slog.StringValue(s.String()) }
func (SelloEvidenciaV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (SelloEvidenciaV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (SelloEvidenciaV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionSecretoDocumentalV3
}
func (*SelloEvidenciaV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*SelloEvidenciaV3) UnmarshalText([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
func (*SelloEvidenciaV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionSecretoDocumentalV3
}
