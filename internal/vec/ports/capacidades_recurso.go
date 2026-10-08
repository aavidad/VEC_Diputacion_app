package ports

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
)

// RecursoLeidoCapacidad sólo lo construye el propietario de una respuesta ya
// autorizada y auditada. RutaID identifica una ruta montada por la composición,
// no una URL ni un valor suministrado por el navegador.
type RecursoLeidoCapacidad struct {
	RutaID    string                    `json:"-"`
	Metodo    string                    `json:"-"`
	Accion    string                    `json:"-"`
	Finalidad string                    `json:"-"`
	Recurso   domain.RecursoAutorizable `json:"-"`
}

// LoteRecursosLeidosCapacidad contiene un único perfil F1. Perfiles fijos de
// rutas CT distintas requieren lotes separados, cada uno con su propia fuente.
type LoteRecursosLeidosCapacidad struct {
	Vinculo    domain.VinculoAutenticacionActorV2        `json:"-"`
	Resultado  domain.ResultadoContextoActorRegistradoV2 `json:"-"`
	Superficie string                                    `json:"-"`
	Recursos   []RecursoLeidoCapacidad                   `json:"-"`
}

type RutaMontadaCapacidadRecurso struct {
	RutaID      string
	Metodo      string
	ModuloID    string
	TipoRecurso string
	Accion      string
	Finalidad   string
}

// Una vista incompleta nunca permite concluir que una ruta está ausente.
type VistaMontajeCapacidadesRecurso struct {
	Superficie      string
	PerfilActivoRef string
	Completa        bool
	Rutas           []RutaMontadaCapacidadRecurso
}

// RegistroMontajeCapacidadesRecurso entrega una vista única de la composición
// realmente activa para superficie y perfil; no consulta una ruta por tarjeta.
type RegistroMontajeCapacidadesRecurso interface {
	VistaMontada(context.Context, string, string) (VistaMontajeCapacidadesRecurso, error)
}
