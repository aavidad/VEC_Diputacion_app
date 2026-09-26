package domain

import (
	"errors"
	"time"
)

// Incorporación acreditada (dudas 11 y 12 de RRHH): RRHH registra la
// confirmación de GINPIX de la ficha de la incorporación, con su número de
// alta, y el cierre toma de ella ese número. La confirmación del centro no
// cambia la versión del expediente y vive en su propia historia.
const AccionConfirmarGINPIX ClaveCatalogo = "contratacion_temporal.ginpix.confirmar"

var ErrConfirmacionGINPIXInvalida = errors.New("contratacion temporal: confirmacion de GINPIX invalida")

// DatosConfirmacionGINPIX es el número de alta que devolvió GINPIX y la fecha
// civil de la confirmación, con observaciones opcionales.
type DatosConfirmacionGINPIX struct {
	Numero        string
	ConfirmadaEn  time.Time
	Observaciones string
}

func (d DatosConfirmacionGINPIX) Validar() error {
	if !patronNumeroGINPIX.MatchString(d.Numero) || !fechaCivilCanonica(d.ConfirmadaEn) || !textoValido(d.Observaciones, 2000, true) {
		return ErrConfirmacionGINPIXInvalida
	}
	return nil
}

// DocumentoGINPIX es la referencia documental de la ficha confirmada; la
// misma que conserva después la actuación del cierre.
func (d DatosConfirmacionGINPIX) DocumentoGINPIX() string {
	return prefijoDocumentoGINPIX + d.Numero
}

// NumeroGINPIXValido comprueba el formato del número de alta de GINPIX.
func NumeroGINPIXValido(numero string) bool { return patronNumeroGINPIX.MatchString(numero) }

// ConfirmarGINPIX añade la actuación de la confirmación de GINPIX. Conserva
// fase y estado; SQL comprueba la incorporación y que sea la primera.
func (e Expediente) ConfirmarGINPIX(versionEsperada uint64, datos DatosConfirmacionGINPIX, actuacion DatosActuacion) (Expediente, error) {
	if e.Validar() != nil || datos.Validar() != nil || actuacion.validar() != nil || !e.enNombramientoVigente() ||
		actuacion.AccionClave != AccionConfirmarGINPIX ||
		actuacion.FaseDestino != FaseNombramiento || actuacion.EstadoDestino != EstadoEnCurso ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef || actuacion.Observaciones != datos.Observaciones ||
		len(actuacion.DocumentosRef) != 1 || actuacion.DocumentosRef[0] != datos.DocumentoGINPIX() ||
		actuacion.RetornoRef != "" {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	return siguiente.confirmarTransicion(actuacion)
}
