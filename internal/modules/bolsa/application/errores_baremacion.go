package application

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	mensajeDocumentoFirmadoHuerfano     = "bolsa: documento firmado custodiado sin decision eficaz"
	mensajeCustodiaBaremacionIncompleta = "bolsa: objeto custodiado pendiente de reconciliacion"
)

// ErrorDocumentoFirmadoHuerfano conserva la referencia institucional del
// documento ya firmado y retenido cuando OCC impide convertirlo en decision
// eficaz. El reconciliador puede inventariarlo sin volver a firmar ni borrarlo.
//
// Sus campos son deliberadamente accesibles al reconciliador, pero todas las
// representaciones genericas se reducen a un mensaje fijo. Asi fmt, slog y los
// codificadores habituales no convierten por accidente recibos o referencias
// internas en datos de una respuesta HTTP o de un registro operacional.
type ErrorDocumentoFirmadoHuerfano struct {
	DecisionRef string
	Documento   puertosbolsa.DocumentoFirmadoCustodiado
	Causa       error
}

func (*ErrorDocumentoFirmadoHuerfano) Error() string { return mensajeDocumentoFirmadoHuerfano }

func (e *ErrorDocumentoFirmadoHuerfano) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Causa
}

func (e *ErrorDocumentoFirmadoHuerfano) String() string   { return e.Error() }
func (e *ErrorDocumentoFirmadoHuerfano) GoString() string { return e.String() }
func (e *ErrorDocumentoFirmadoHuerfano) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (e *ErrorDocumentoFirmadoHuerfano) LogValue() slog.Value {
	return slog.StringValue(e.String())
}
func (e *ErrorDocumentoFirmadoHuerfano) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}
func (e *ErrorDocumentoFirmadoHuerfano) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}
func (e *ErrorDocumentoFirmadoHuerfano) MarshalBinary() ([]byte, error) {
	return []byte(e.String()), nil
}

// ErrorCustodiaBaremacionIncompleta conserva el recibo de cualquier objeto
// que el almacen haya creado antes de fallar su validacion, retencion o enlace
// con el expediente. Nunca se descarta esa referencia: un reconciliador debe
// completar la retencion o inmovilizar el objeto con intervencion registrada.
type ErrorCustodiaBaremacionIncompleta struct {
	DecisionRef  string
	DocumentoRef string
	Escritura    puertosvec.ResultadoOperacionObjeto
	Retencion    *puertosvec.ResultadoOperacionObjeto
	Causa        error
}

func (*ErrorCustodiaBaremacionIncompleta) Error() string {
	return mensajeCustodiaBaremacionIncompleta
}

func (e *ErrorCustodiaBaremacionIncompleta) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Causa
}

func (e *ErrorCustodiaBaremacionIncompleta) String() string {
	return e.Error()
}
func (e *ErrorCustodiaBaremacionIncompleta) GoString() string { return e.String() }
func (e *ErrorCustodiaBaremacionIncompleta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (e *ErrorCustodiaBaremacionIncompleta) LogValue() slog.Value {
	return slog.StringValue(e.String())
}
func (e *ErrorCustodiaBaremacionIncompleta) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.String())
}
func (e *ErrorCustodiaBaremacionIncompleta) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}
func (e *ErrorCustodiaBaremacionIncompleta) MarshalBinary() ([]byte, error) {
	return []byte(e.String()), nil
}
