package auditoria

import vecdomain "vec-diputacion-granada/internal/vec/domain"

// validarRecursoConsultaLigadoAlFiltro comprueba el recurso YA incluido en la
// solicitud V3. La pertenencia de la organización o bolsa corresponde a la
// transacción SQL del propietario, junto al consumo y la auditoría.
func validarRecursoConsultaLigadoAlFiltro(recurso vecdomain.RecursoAutorizable, filtro Filtro) error {
	huella, err := HuellaFiltro(filtro)
	if err != nil {
		return err
	}
	if recurso.Validar() != nil || recurso.Referencia != filtro.ExpedienteRef ||
		recurso.ModuloID != ModuloAutorizacion || recurso.Tipo != TipoRecurso ||
		len(recurso.Atributos) != 1 || recurso.Atributos["filtro_sha256"] != huella ||
		len(recurso.Ambitos) != 2 || recurso.Ambitos["fuente"] != filtro.Fuente {
		return ErrDenegada
	}
	if expediente, ok := recurso.Ambitos["expediente_ref"]; ok {
		if expediente == filtro.ExpedienteRef {
			return nil
		}
		return ErrDenegada
	}
	switch filtro.Fuente {
	case "ct":
		organizacion, ok := recurso.Ambitos["organizacion_ref"]
		if ok && referenciaExacta(organizacion, 512) {
			return nil
		}
	case "bolsa":
		bolsa, ok := recurso.Ambitos["bolsa_ref"]
		if ok && referenciaExacta(bolsa, 512) {
			return nil
		}
	}
	return ErrDenegada
}
