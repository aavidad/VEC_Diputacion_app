package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaRelacionUnidadExpedienteCT = "vec.contratacion-temporal.relacion-unidad-expediente.v1"
	maximaRelacionUnidadCTBytes       = 16 << 10
	formatoFechaRelacionUnidadCT      = "2006-01-02T15:04:05.000000Z"
)

type relacionUnidadExpedienteCTSQL struct {
	Esquema                          string `json:"esquema"`
	OrganizacionRef                  string `json:"organizacion_ref"`
	ExpedienteRef                    string `json:"expediente_ref"`
	UnidadRefEsperada                string `json:"unidad_ref_esperada"`
	VersionExpedienteSolicitada      uint64 `json:"version_expediente_solicitada"`
	UnidadRef                        string `json:"unidad_ref"`
	UnidadSnapshotSolicitadoRef      string `json:"unidad_snapshot_solicitado_ref"`
	VersionExpedienteActual          uint64 `json:"version_expediente_actual"`
	VersionOrigenVinculo             uint64 `json:"version_origen_vinculo"`
	OperacionOrigenRef               string `json:"operacion_origen_ref"`
	ReservaAsignacionRef             string `json:"reserva_asignacion_ref"`
	ReciboAsignacionRef              string `json:"recibo_asignacion_ref"`
	PruebaSnapshotOrigenHuellaSHA256 string `json:"prueba_snapshot_origen_huella_sha256"`
	TipoEventoOrigen                 string `json:"tipo_evento_origen"`
	EventoAsignacionRef              string `json:"evento_asignacion_ref"`
	EventoPayloadHuellaSHA256        string `json:"evento_payload_huella_sha256"`
	AsignacionConfirmadaEn           string `json:"asignacion_confirmada_en"`
}

// DecodificarRelacionUnidadExpedienteCT traduce el resultado fuente acordado
// con CT174. No llama al helper privado ni abre una conexión o transacción.
// La fachada del efecto debe producir estos hechos tras consumir V3 y su
// auditoría, derivar el vínculo CT y revalidar K dentro de su única llamada.
// Una decodificación correcta no acredita que esas operaciones se ejecutaron.
func DecodificarRelacionUnidadExpedienteCT(contenido []byte, q ports.SolicitudRelacionUnidadExpedienteCT) (ports.RelacionUnidadExpedienteCT, error) {
	var cero ports.RelacionUnidadExpedienteCT
	if err := q.Validar(); err != nil {
		return cero, err
	}
	if len(contenido) == 0 {
		return cero, falloRelacionUnidadCT(io.ErrUnexpectedEOF)
	}
	if len(contenido) > maximaRelacionUnidadCTBytes {
		return cero, falloRelacionUnidadCT(errors.New("ct.relacion_unidad_expediente.json.limite"))
	}
	if err := clavesRelacionUnidadCTExactas(contenido); err != nil {
		return cero, falloRelacionUnidadCT(err)
	}
	var s relacionUnidadExpedienteCTSQL
	if err := decodificarJSONEstricto(contenido, &s); err != nil {
		return cero, falloRelacionUnidadCT(err)
	}
	if s.Esquema != esquemaRelacionUnidadExpedienteCT {
		return cero, falloRelacionUnidadCT(errors.New("ct.relacion_unidad_expediente.json.esquema"))
	}
	fecha, err := time.Parse(formatoFechaRelacionUnidadCT, s.AsignacionConfirmadaEn)
	if err != nil {
		return cero, falloRelacionUnidadCT(err)
	}
	if fecha.Format(formatoFechaRelacionUnidadCT) != s.AsignacionConfirmadaEn {
		return cero, falloRelacionUnidadCT(errors.New("ct.relacion_unidad_expediente.fecha.canon"))
	}
	r := ports.RelacionUnidadExpedienteCT{
		Solicitud: ports.SolicitudRelacionUnidadExpedienteCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
			UnidadRefEsperada: s.UnidadRefEsperada, VersionExpedienteSolicitada: s.VersionExpedienteSolicitada},
		UnidadRef: s.UnidadRef, UnidadSnapshotSolicitadoRef: s.UnidadSnapshotSolicitadoRef,
		VersionExpedienteActual: s.VersionExpedienteActual, VersionOrigenVinculo: s.VersionOrigenVinculo,
		OperacionOrigenRef: s.OperacionOrigenRef, ReservaAsignacionRef: s.ReservaAsignacionRef, ReciboAsignacionRef: s.ReciboAsignacionRef,
		PruebaSnapshotOrigenHuellaSHA256: s.PruebaSnapshotOrigenHuellaSHA256, TipoEventoOrigen: s.TipoEventoOrigen,
		EventoAsignacionRef: s.EventoAsignacionRef, EventoPayloadHuellaSHA256: s.EventoPayloadHuellaSHA256,
		AsignacionConfirmadaEn: fecha,
	}
	if err := r.ValidarPara(q); err != nil {
		return cero, err
	}
	return r, nil
}

// JSONB no conserva claves repetidas. La frontera Go rechaza también JSON
// ajeno a esa forma, claves no declaradas y aliases de capitalización.
func clavesRelacionUnidadCTExactas(contenido []byte) error {
	permitidas := map[string]bool{
		"esquema": true, "organizacion_ref": true, "expediente_ref": true, "unidad_ref_esperada": true,
		"version_expediente_solicitada": true, "unidad_ref": true, "unidad_snapshot_solicitado_ref": true,
		"version_expediente_actual": true, "version_origen_vinculo": true, "operacion_origen_ref": true,
		"reserva_asignacion_ref": true, "recibo_asignacion_ref": true, "prueba_snapshot_origen_huella_sha256": true,
		"tipo_evento_origen": true, "evento_asignacion_ref": true, "evento_payload_huella_sha256": true,
		"asignacion_confirmada_en": true,
	}
	vistos := make(map[string]bool, len(permitidas))
	d := json.NewDecoder(bytes.NewReader(contenido))
	token, err := d.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return errors.New("ct.relacion_unidad_expediente.json.objeto")
	}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return err
		}
		clave, ok := token.(string)
		if !ok || !permitidas[clave] {
			return errors.New("ct.relacion_unidad_expediente.json.clave")
		}
		if vistos[clave] {
			return errors.New("ct.relacion_unidad_expediente.json.clave_repetida")
		}
		vistos[clave] = true
		var valor json.RawMessage
		if err := d.Decode(&valor); err != nil {
			return err
		}
	}
	token, err = d.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('}') {
		return errors.New("ct.relacion_unidad_expediente.json.cierre")
	}
	var extra any
	err = d.Decode(&extra)
	if err != io.EOF {
		if err != nil {
			return err
		}
		return errors.New("ct.relacion_unidad_expediente.json.contenido_sobrante")
	}
	if len(vistos) != len(permitidas) {
		return errors.New("ct.relacion_unidad_expediente.json.claves_incompletas")
	}
	return nil
}

// El mensaje público es cerrado. La causa queda accesible a la clasificación
// técnica por errors.As/Is, sin incorporar valores privados a Error().
type errorDecodificacionRelacionUnidadCT struct{ causa error }

func (e errorDecodificacionRelacionUnidadCT) Error() string {
	return ports.ErrRelacionUnidadExpedienteCTInvalida.Error()
}
func (e errorDecodificacionRelacionUnidadCT) Unwrap() []error {
	return []error{ports.ErrRelacionUnidadExpedienteCTInvalida, e.causa}
}
func falloRelacionUnidadCT(causa error) error {
	return errorDecodificacionRelacionUnidadCT{causa: causa}
}
