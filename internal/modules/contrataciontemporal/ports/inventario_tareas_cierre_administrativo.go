package ports

import (
	"context"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// EntradaInventarioTareasCierreAdministrativoEjercicio procede de la frontera transaccional.
// Completo no es una entrada: se deriva del snapshot validado.
type EntradaInventarioTareasCierreAdministrativoEjercicio struct {
	Libro           domain.PublicacionLibroTareasCierreAdministrativoEjercicio
	OrganizacionRef string
	ExpedienteRef   string
	SeguimientoRef  string
	Estados         []domain.EstadoTareaCierreAdministrativoEjercicio
}

func (e EntradaInventarioTareasCierreAdministrativoEjercicio) Preparar() (InventarioTareasCierreAdministrativo, error) {
	s, err := domain.NuevoSnapshotTareasCierreAdministrativoEjercicio(domain.SnapshotTareasCierreAdministrativoEjercicio{Libro: e.Libro, OrganizacionRef: e.OrganizacionRef, ExpedienteRef: e.ExpedienteRef, SeguimientoRef: e.SeguimientoRef, Estados: e.Estados})
	if err != nil {
		return InventarioTareasCierreAdministrativo{}, ErrPreparacionCierreAdministrativoInvalida
	}
	i := InventarioTareasCierreAdministrativo{Referencia: s.Libro.Referencia, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, SeguimientoRef: s.SeguimientoRef, Version: s.Libro.Version, Total: s.Total(), Pendientes: s.Pendientes(), Completo: true}
	if i.Validar() != nil {
		return InventarioTareasCierreAdministrativo{}, ErrPreparacionCierreAdministrativoInvalida
	}
	return i, nil
}

// PreparadorInventarioTareasCierreAdministrativoEjercicio pertenece al adaptador durable;
// bloquea fuentes y entrega el DTO al producto transaccional, nunca al canal HTTP.
type PreparadorInventarioTareasCierreAdministrativoEjercicio interface {
	PrepararInventarioTareasCierreAdministrativoEjercicio(context.Context, EntradaInventarioTareasCierreAdministrativoEjercicio) (InventarioTareasCierreAdministrativo, error)
}
