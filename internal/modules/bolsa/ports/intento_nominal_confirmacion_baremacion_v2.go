package ports

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
)

var ErrSerializacionConfirmacionNominalBaremacionV2Prohibida = errors.New(
	"bolsa: serializacion generica de confirmacion nominal V2 prohibida",
)

const mensajeIntentoNominalConfirmacionBaremacionV2 = "[INTENTO-NOMINAL-CONFIRMACION-BAREMACION-V2-PROTEGIDO]"

// IntentoNominalConfirmacionBaremacionV2 mantiene aislado del flujo V1 el
// sobre probatorio exacto de un intento. El identificador debe existir
// durablemente antes de construirlo y su sello debe cubrirlo junto al efecto.
//
// Este tipo es nominal: solo acredita forma y permite producir el canonico. No
// acredita autenticidad, preparacion durable, persistencia ni resultado de
// COMMIT y no habilita ningun efecto. El flujo sigue cerrado hasta disponer de
// servicio TCB, PrepararOperacion, resultado canonico y reconciliador V2.
type IntentoNominalConfirmacionBaremacionV2 struct {
	IdentificadorOperacion IdentificadorOperacionTransaccionalBaremacion
	Confirmacion           SolicitudConfirmarCambioBaremacion
}

func (s IntentoNominalConfirmacionBaremacionV2) ValidarForma() error {
	if s.IdentificadorOperacion.Validar() != nil || s.Confirmacion.Validar() != nil {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (s IntentoNominalConfirmacionBaremacionV2) Clonar() (IntentoNominalConfirmacionBaremacionV2, error) {
	if err := s.ValidarForma(); err != nil {
		return IntentoNominalConfirmacionBaremacionV2{}, err
	}
	identificador, err := s.IdentificadorOperacion.Clonar()
	if err != nil {
		return IntentoNominalConfirmacionBaremacionV2{}, ErrSolicitudBaremacionInvalida
	}
	confirmacion, err := s.Confirmacion.Clonar()
	if err != nil {
		return IntentoNominalConfirmacionBaremacionV2{}, ErrSolicitudBaremacionInvalida
	}
	return IntentoNominalConfirmacionBaremacionV2{
		IdentificadorOperacion: identificador,
		Confirmacion:           confirmacion,
	}, nil
}

func (IntentoNominalConfirmacionBaremacionV2) String() string {
	return mensajeIntentoNominalConfirmacionBaremacionV2
}
func (IntentoNominalConfirmacionBaremacionV2) GoString() string {
	return "ports.IntentoNominalConfirmacionBaremacionV2{[PROTEGIDO]}"
}
func (s IntentoNominalConfirmacionBaremacionV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (IntentoNominalConfirmacionBaremacionV2) LogValue() slog.Value {
	return slog.StringValue(mensajeIntentoNominalConfirmacionBaremacionV2)
}
func (IntentoNominalConfirmacionBaremacionV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (*IntentoNominalConfirmacionBaremacionV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (IntentoNominalConfirmacionBaremacionV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (*IntentoNominalConfirmacionBaremacionV2) UnmarshalText([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (IntentoNominalConfirmacionBaremacionV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (*IntentoNominalConfirmacionBaremacionV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}

// ResultadoNominalConfirmacionBaremacionV2 coteja solo la forma del eco de un
// adaptador nominal. La coincidencia sintactica del identificador no demuestra
// que el COMMIT lo persistiera ni que version, auditoria y evento nacieran
// atomicamente. Esa atribucion requerira el resultado canonico autenticado V2.
type ResultadoNominalConfirmacionBaremacionV2 struct {
	IdentificadorOperacion IdentificadorOperacionTransaccionalBaremacion
	Resultado              ResultadoConfirmarCambioBaremacion
}

func (r ResultadoNominalConfirmacionBaremacionV2) ValidarFormaPara(
	s IntentoNominalConfirmacionBaremacionV2,
) error {
	if s.ValidarForma() != nil || r.IdentificadorOperacion.Validar() != nil ||
		!r.IdentificadorOperacion.CoincideExactamenteCon(s.IdentificadorOperacion) ||
		r.Resultado.ValidarPara(s.Confirmacion) != nil {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

func (r ResultadoNominalConfirmacionBaremacionV2) ClonarFormaPara(
	s IntentoNominalConfirmacionBaremacionV2,
) (ResultadoNominalConfirmacionBaremacionV2, error) {
	if err := r.ValidarFormaPara(s); err != nil {
		return ResultadoNominalConfirmacionBaremacionV2{}, err
	}
	identificador, err := r.IdentificadorOperacion.Clonar()
	if err != nil {
		return ResultadoNominalConfirmacionBaremacionV2{}, ErrSolicitudBaremacionInvalida
	}
	resultado, err := r.Resultado.Clonar()
	if err != nil {
		return ResultadoNominalConfirmacionBaremacionV2{}, ErrSolicitudBaremacionInvalida
	}
	return ResultadoNominalConfirmacionBaremacionV2{
		IdentificadorOperacion: identificador,
		Resultado:              resultado,
	}, nil
}

func (ResultadoNominalConfirmacionBaremacionV2) String() string {
	return "[RESULTADO-NOMINAL-CONFIRMACION-BAREMACION-V2-PROTEGIDO]"
}
func (ResultadoNominalConfirmacionBaremacionV2) GoString() string {
	return "ports.ResultadoNominalConfirmacionBaremacionV2{[PROTEGIDO]}"
}
func (r ResultadoNominalConfirmacionBaremacionV2) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r ResultadoNominalConfirmacionBaremacionV2) LogValue() slog.Value {
	return slog.StringValue(r.String())
}
func (ResultadoNominalConfirmacionBaremacionV2) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (*ResultadoNominalConfirmacionBaremacionV2) UnmarshalJSON([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (ResultadoNominalConfirmacionBaremacionV2) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (*ResultadoNominalConfirmacionBaremacionV2) UnmarshalText([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (ResultadoNominalConfirmacionBaremacionV2) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
func (*ResultadoNominalConfirmacionBaremacionV2) UnmarshalBinary([]byte) error {
	return ErrSerializacionConfirmacionNominalBaremacionV2Prohibida
}
