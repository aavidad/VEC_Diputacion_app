package postgres

import "vec-diputacion-granada/internal/vec/ports"

// falloValidacionRedactado consume la causa interna de un predicado cerrado.
// Un parser puede incluir el valor recibido (p. ej. una fecha) en Error();
// ese texto no debe cruzar la frontera ADMIN ni entrar en la auditoría común.
func falloValidacionRedactado(_ error) error {
	return ports.ErrLecturaUsuariosAdministrablesNoDisponible
}
