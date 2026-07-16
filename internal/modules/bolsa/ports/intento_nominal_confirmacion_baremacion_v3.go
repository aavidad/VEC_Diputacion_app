package ports

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var ErrSerializacionConfirmacionNominalBaremacionV3Prohibida = errors.New(
	"bolsa: serializacion generica de confirmacion nominal V3 prohibida",
)

const mensajeIntentoNominalConfirmacionBaremacionV3 = "[INTENTO-NOMINAL-CONFIRMACION-BAREMACION-V3-PROTEGIDO]"

// IntentoNominalConfirmacionBaremacionV3 sustituye al sobre V2 retirado e
// incorpora la autorizacion de prevalidacion dentro del canonico autenticado.
// Sigue siendo un contrato nominal: no acredita persistencia ni resultado.
type IntentoNominalConfirmacionBaremacionV3 struct {
	IdentificadorOperacion IdentificadorOperacionTransaccionalBaremacion
	Confirmacion           SolicitudConfirmarCambioBaremacion
}

func (s IntentoNominalConfirmacionBaremacionV3) ValidarForma() error {
	if s.IdentificadorOperacion.Validar() != nil || s.Confirmacion.Validar() != nil {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (s IntentoNominalConfirmacionBaremacionV3) Clonar() (IntentoNominalConfirmacionBaremacionV3, error) {
	if err := s.ValidarForma(); err != nil {
		return IntentoNominalConfirmacionBaremacionV3{}, err
	}
	identificador, err := s.IdentificadorOperacion.Clonar()
	if err != nil {
		return IntentoNominalConfirmacionBaremacionV3{}, ErrSolicitudBaremacionInvalida
	}
	confirmacion, err := s.Confirmacion.Clonar()
	if err != nil {
		return IntentoNominalConfirmacionBaremacionV3{}, ErrSolicitudBaremacionInvalida
	}
	return IntentoNominalConfirmacionBaremacionV3{
		IdentificadorOperacion: identificador,
		Confirmacion:           confirmacion,
	}, nil
}

func (IntentoNominalConfirmacionBaremacionV3) String() string {
	return mensajeIntentoNominalConfirmacionBaremacionV3
}
func (IntentoNominalConfirmacionBaremacionV3) GoString() string {
	return "ports.IntentoNominalConfirmacionBaremacionV3{[PROTEGIDO]}"
}
func (s IntentoNominalConfirmacionBaremacionV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (IntentoNominalConfirmacionBaremacionV3) LogValue() slog.Value {
	return slog.StringValue(mensajeIntentoNominalConfirmacionBaremacionV3)
}
func (IntentoNominalConfirmacionBaremacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (*IntentoNominalConfirmacionBaremacionV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (IntentoNominalConfirmacionBaremacionV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (*IntentoNominalConfirmacionBaremacionV3) UnmarshalText([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (IntentoNominalConfirmacionBaremacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (*IntentoNominalConfirmacionBaremacionV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}

// ResultadoNominalConfirmacionBaremacionV3 valida solo el eco nominal; no
// eleva el sobre a prueba de COMMIT ni relaja el fail-closed transaccional.
type ResultadoNominalConfirmacionBaremacionV3 struct {
	IdentificadorOperacion IdentificadorOperacionTransaccionalBaremacion
	Resultado              ResultadoConfirmarCambioBaremacion
}

func (r ResultadoNominalConfirmacionBaremacionV3) ValidarFormaPara(
	s IntentoNominalConfirmacionBaremacionV3,
) error {
	if s.ValidarForma() != nil || r.IdentificadorOperacion.Validar() != nil ||
		!r.IdentificadorOperacion.CoincideExactamenteCon(s.IdentificadorOperacion) ||
		r.Resultado.ValidarPara(s.Confirmacion) != nil {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (r ResultadoNominalConfirmacionBaremacionV3) ClonarFormaPara(
	s IntentoNominalConfirmacionBaremacionV3,
) (ResultadoNominalConfirmacionBaremacionV3, error) {
	if err := r.ValidarFormaPara(s); err != nil {
		return ResultadoNominalConfirmacionBaremacionV3{}, err
	}
	identificador, err := r.IdentificadorOperacion.Clonar()
	if err != nil {
		return ResultadoNominalConfirmacionBaremacionV3{}, ErrSolicitudBaremacionInvalida
	}
	resultado, err := r.Resultado.Clonar()
	if err != nil {
		return ResultadoNominalConfirmacionBaremacionV3{}, ErrSolicitudBaremacionInvalida
	}
	return ResultadoNominalConfirmacionBaremacionV3{
		IdentificadorOperacion: identificador,
		Resultado:              resultado,
	}, nil
}

func (ResultadoNominalConfirmacionBaremacionV3) String() string {
	return "[RESULTADO-NOMINAL-CONFIRMACION-BAREMACION-V3-PROTEGIDO]"
}
func (ResultadoNominalConfirmacionBaremacionV3) GoString() string {
	return "ports.ResultadoNominalConfirmacionBaremacionV3{[PROTEGIDO]}"
}
func (r ResultadoNominalConfirmacionBaremacionV3) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ResultadoNominalConfirmacionBaremacionV3) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (ResultadoNominalConfirmacionBaremacionV3) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (*ResultadoNominalConfirmacionBaremacionV3) UnmarshalJSON([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (ResultadoNominalConfirmacionBaremacionV3) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (*ResultadoNominalConfirmacionBaremacionV3) UnmarshalText([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (ResultadoNominalConfirmacionBaremacionV3) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
func (*ResultadoNominalConfirmacionBaremacionV3) UnmarshalBinary([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV3Prohibida
}
