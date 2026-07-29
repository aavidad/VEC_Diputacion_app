package ports

import dominiovec "vec-diputacion-granada/internal/vec/domain"

// RecursosConsultaRRHH conserva el recurso autorizable resuelto por el
// servidor sin exponer sus mapas ni permitir que cruce una frontera de
// transporte o registro. Solo las fábricas nominales del paquete pueden
// producir valores válidos.
type RecursosConsultaRRHH struct {
	bloqueoSerializacionConsultaRRHH
	recurso dominiovec.RecursoAutorizable
}

// validarEstructura comprueba únicamente las invariantes comunes. Las
// fábricas nominales de cada consulta deben imponer además referencia, tipo,
// ámbitos, atributos y huella exactos.
func (r RecursosConsultaRRHH) validarEstructura() error {
	if r.recurso.Validar() != nil ||
		r.recurso.ModuloID != ModuloContratacion {
		return ErrCapacidadConsultaRRHHInvalida
	}
	return nil
}
