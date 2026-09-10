package lecturaincorporacion

import "time"

func (a AutorizacionV2) validar(m MaterialV2, ahora time.Time) error {
	r, err := m.Recurso()
	if err != nil {
		return ErrDenegada
	}
	return validarAutorizacionLectura(Autorizacion(a), m.base.contexto, m.PreparadoEn(), r, AudienciaV2, ahora)
}
