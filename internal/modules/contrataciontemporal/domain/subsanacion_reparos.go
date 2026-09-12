package domain

import "errors"

const AccionRegistrarSubsanacionReparo ClaveCatalogo = "contratacion_temporal.subsanacion_reparos.registrar"

var ErrSubsanacionReparoInvalida = errors.New("contratacion temporal: subsanacion de reparo invalida")

// DatosSubsanacionReparo identifica exclusivamente el retorno creado por la
// fiscalización desfavorable. No expresa conformidad ni altera su resultado.
type DatosSubsanacionReparo struct {
	RetornoRef    string
	Observaciones string
}

func (d DatosSubsanacionReparo) Validar() error {
	if !referenciaValida(d.RetornoRef) || !textoValido(d.Observaciones, 2000, false) {
		return ErrSubsanacionReparoInvalida
	}
	return nil
}

// RegistrarSubsanacionReparo agrega una actuación ligada al reparo vigente y
// mantiene el expediente en incidencia. La fiscalización desfavorable queda
// intacta y ninguna subsanación equivale a una fiscalización favorable.
func (e Expediente) RegistrarSubsanacionReparo(
	versionEsperada uint64,
	datos DatosSubsanacionReparo,
	actuacion DatosActuacion,
) (Expediente, error) {
	if e.Validar() != nil || datos.Validar() != nil || actuacion.validar() != nil ||
		actuacion.AccionClave != AccionRegistrarSubsanacionReparo ||
		e.Fiscalizacion == nil || e.Fiscalizacion.Resultado != FiscalizacionDesfavorable ||
		e.Fiscalizacion.Retorno == nil || e.Fiscalizacion.Retorno.RetornoRef != datos.RetornoRef ||
		e.FaseActual != FaseSubsanacionUnidad || e.EstadoActual != EstadoIncidencia ||
		actuacion.FaseDestino != FaseSubsanacionUnidad ||
		actuacion.EstadoDestino != EstadoIncidencia ||
		actuacion.Observaciones != datos.Observaciones ||
		actuacion.RetornoRef != datos.RetornoRef ||
		e.tieneSubsanacionParaRetorno(datos.RetornoRef) {
		return Expediente{}, ErrTransicionInvalida
	}
	siguiente, err := e.prepararTransicion(versionEsperada, actuacion)
	if err != nil {
		return Expediente{}, err
	}
	return siguiente.confirmarTransicion(actuacion)
}

func (e Expediente) tieneSubsanacionParaRetorno(retornoRef string) bool {
	for _, actuacion := range e.Actuaciones {
		if actuacion.AccionClave == AccionRegistrarSubsanacionReparo && actuacion.RetornoRef == retornoRef {
			return true
		}
	}
	return false
}
