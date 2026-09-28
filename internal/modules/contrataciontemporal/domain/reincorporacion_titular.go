package domain

import (
	"errors"
	"time"
)

const AccionRegistrarReincorporacionTitular ClaveCatalogo = "contratacion_temporal.seguimiento.registrar_reincorporacion_titular"

var ErrReincorporacionTitularInvalida = errors.New("contratacion temporal: reincorporacion titular invalida")

// DatosReincorporacionTitular acredita el retorno del titular. El cese CT115
// y la relación CT75 se cotejan bajo bloqueo en persistencia.
type DatosReincorporacionTitular struct {
	RelacionRef, DocumentoRef, DocumentoSHA256 string
	FechaEfectiva                              time.Time
}

func (d DatosReincorporacionTitular) Validar() error {
	if !referenciaValida(d.RelacionRef) || !referenciaValida(d.DocumentoRef) ||
		!huellaEntradaValida(d.DocumentoSHA256) || !fechaCivilCanonica(d.FechaEfectiva) {
		return ErrReincorporacionTitularInvalida
	}
	return nil
}

// RegistrarReincorporacionTitular añade una actuación sin cerrar ni reabrir
// el expediente. El retorno no modifica la relación de Personal ni Bolsa.
func (e Expediente) RegistrarReincorporacionTitular(versionEsperada uint64, datos DatosReincorporacionTitular, actuacion DatosActuacion) (Expediente, error) {
	if e.Validar() != nil || datos.Validar() != nil || actuacion.validar() != nil ||
		!e.enNombramientoVigente() || !e.TieneAccion(AccionCesarNombramiento) ||
		e.TieneAccion(AccionRegistrarReincorporacionTitular) ||
		actuacion.AccionClave != AccionRegistrarReincorporacionTitular ||
		actuacion.FaseDestino != FaseNombramiento || actuacion.EstadoDestino != EstadoEnCurso ||
		actuacion.UnidadRef != e.Asignacion.UnidadRef || actuacion.Observaciones != "" ||
		len(actuacion.DocumentosRef) != 1 || actuacion.DocumentosRef[0] != datos.DocumentoRef ||
		actuacion.RetornoRef != "" {
		return Expediente{}, ErrReincorporacionTitularInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	return siguiente.confirmarTransicion(actuacion)
}
