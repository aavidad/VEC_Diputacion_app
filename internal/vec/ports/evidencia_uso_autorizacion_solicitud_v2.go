package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 es la proyeccion
// defensiva y no reconstruible que un adaptador durable necesita para cotejar
// una decision V2. No es asignable al contrato historico V1.
type DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 struct {
	EsquemaHuella          string
	HuellaDecisionSHA256   string
	Decision               domain.DecisionAutorizacion
	VerificadaEn           time.Time
	representacionCanonica []byte
}

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) RepresentacionCanonica() ([]byte, error) {
	if len(d.representacionCanonica) == 0 ||
		d.EsquemaHuella != EsquemaHuellaDecisionAutorizacionReforzadaV2 ||
		!esHuellaSHA256EvidenciaUsoAutorizacion(d.HuellaDecisionSHA256) {
		return nil, errorEvidenciaUsoDecisionAutorizacion()
	}
	suma := sha256.Sum256(d.representacionCanonica)
	if hex.EncodeToString(suma[:]) != d.HuellaDecisionSHA256 {
		return nil, errorEvidenciaUsoDecisionAutorizacion()
	}
	return append([]byte(nil), d.representacionCanonica...), nil
}

// ValidarMotivo coteja la referencia completa ya resuelta por la frontera
// confiable. La existencia y vigencia del catalogo deben revalidarse en la
// transaccion durable; esta huella acredita integridad, no procedencia.
func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) ValidarMotivo(
	motivo domain.ReferenciaEntradaCatalogo,
) error {
	if d.validarEstructura() != nil {
		return errorEvidenciaUsoDecisionAutorizacion()
	}
	huella, err := domain.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil || d.Decision.EsquemaHuellaMotivo != domain.EsquemaHuellaMotivoAutorizacionV2 ||
		d.Decision.MotivoHuellaSHA256 != huella {
		return errorEvidenciaUsoDecisionAutorizacion()
	}
	return nil
}

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) validarEstructura() error {
	representacion, err := d.RepresentacionCanonica()
	if err != nil || !instanteEvidenciaUsoAutorizacionCanonico(d.VerificadaEn) ||
		d.Decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil ||
		!d.Decision.VigenteParaEfectoEn(d.VerificadaEn) {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	recalculada, err := serializarDecisionAutorizacionReforzadaV2(d.Decision)
	if err != nil || !bytes.Equal(representacion, recalculada) {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return nil
}

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalText([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GobDecode([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalCBOR([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalYAML() (any, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) String() string {
	return "[DATOS-EVIDENCIA-USO-AUTORIZACION-SOLICITUD-LIGADA-V2]"
}

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GoString() string { return d.String() }

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, d.String())
}

func (d DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) LogValue() slog.Value {
	return slog.StringValue(d.String())
}

type datosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 struct {
	DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
}

// EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 es una capacidad opaca
// exclusiva de efectos V2. No existe conversion desde V1 ni constructor desde
// bytes o una proyeccion historica.
type EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2 struct {
	datos *datosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
}

func NuevaEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
	decision domain.DecisionAutorizacion,
	verificadaEn time.Time,
) (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error) {
	if decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil || !decision.Concedida ||
		!instanteEvidenciaUsoAutorizacionCanonico(verificadaEn) ||
		!decision.VigenteParaEfectoEn(verificadaEn) || contieneComodinDecisionAutorizacion(decision) ||
		len(decision.Obligaciones) != 0 {
		return EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, errorEvidenciaUsoDecisionAutorizacion()
	}
	decisionCanonica := clonarDecisionAutorizacionCanonica(decision)
	representacion, err := serializarDecisionAutorizacionReforzadaV2(decisionCanonica)
	if err != nil {
		return EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, errorEvidenciaUsoDecisionAutorizacion()
	}
	suma := sha256.Sum256(representacion)
	evidencia := EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{
		datos: &datosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{
			DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2: DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{
				EsquemaHuella:        EsquemaHuellaDecisionAutorizacionReforzadaV2,
				HuellaDecisionSHA256: hex.EncodeToString(suma[:]),
				Decision:             decisionCanonica, VerificadaEn: verificadaEn,
				representacionCanonica: append([]byte(nil), representacion...),
			},
		},
	}
	if evidencia.validarEstructura() != nil {
		return EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, errorEvidenciaUsoDecisionAutorizacion()
	}
	return evidencia, nil
}

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) Datos() (
	DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	error,
) {
	if e.validarEstructura() != nil {
		return DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, errorEvidenciaUsoDecisionAutorizacion()
	}
	resultado := e.datos.DatosEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	resultado.Decision = clonarDecisionAutorizacionCanonica(resultado.Decision)
	resultado.representacionCanonica = append([]byte(nil), resultado.representacionCanonica...)
	return resultado, nil
}

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) ValidarEn(instante time.Time) error {
	if e.validarEstructura() != nil || instante.IsZero() {
		return errorEvidenciaUsoDecisionAutorizacion()
	}
	instante = instante.UTC()
	if instante.Before(e.datos.VerificadaEn) || !e.datos.Decision.VigenteParaEfectoEn(instante) {
		return errorEvidenciaUsoDecisionAutorizacion()
	}
	return nil
}

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) ValidarMotivo(
	motivo domain.ReferenciaEntradaCatalogo,
) error {
	datos, err := e.Datos()
	if err != nil {
		return err
	}
	return datos.ValidarMotivo(motivo)
}

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) validarEstructura() error {
	if e.datos == nil || e.datos.EsquemaHuella != EsquemaHuellaDecisionAutorizacionReforzadaV2 ||
		!esHuellaSHA256EvidenciaUsoAutorizacion(e.datos.HuellaDecisionSHA256) ||
		!instanteEvidenciaUsoAutorizacionCanonico(e.datos.VerificadaEn) {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	decision := e.datos.Decision
	if decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2() != nil || !decision.Concedida ||
		!decision.VigenteParaEfectoEn(e.datos.VerificadaEn) || contieneComodinDecisionAutorizacion(decision) ||
		len(decision.Obligaciones) != 0 {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	representacion, err := serializarDecisionAutorizacionReforzadaV2(decision)
	if err != nil || !bytes.Equal(representacion, e.datos.representacionCanonica) {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	suma := sha256.Sum256(representacion)
	if hex.EncodeToString(suma[:]) != e.datos.HuellaDecisionSHA256 {
		return ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return nil
}

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalText([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GobDecode([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalCBOR([]byte) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) MarshalYAML() (any, error) {
	return nil, ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (*EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionEvidenciaUsoAutorizacionProhibida
}

func (EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) String() string {
	return "[EVIDENCIA-USO-AUTORIZACION-SOLICITUD-LIGADA-V2]"
}

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) GoString() string { return e.String() }

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}

func (e EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2) LogValue() slog.Value {
	return slog.StringValue(e.String())
}

var (
	_ json.Marshaler = EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}
	_ fmt.Stringer   = EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}
	_ slog.LogValuer = EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}
	_ encodingTextV2 = EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}
)

type encodingTextV2 interface {
	MarshalText() ([]byte, error)
}
