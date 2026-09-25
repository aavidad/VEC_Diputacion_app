package domain

import "errors"

var ErrAlcanceProyeccionesContextoActorInvalido = errors.New(
	"vec: alcance de proyecciones de contexto de actor invalido",
)

// ProyeccionContextoActor nombra una proyeccion de otro modulo que la
// composicion servidor pide expresamente al resolver el contexto. Lista
// cerrada: hoy solo existe el empleado canonico de Personal.
type ProyeccionContextoActor string

const ProyeccionContextoActorEmpleado ProyeccionContextoActor = "empleado"

// prefijoVinculoProyeccionEmpleado identifica en el canon V2 la entrada de
// empleado que procede de la proyeccion gobernada de Personal (pep_), frente a
// los punteros heredados del nucleo (vin_). Su presencia es la forma durable
// del alcance: no hay claves nuevas en el canon.
const prefijoVinculoProyeccionEmpleado = "pep_"

// AlcanceProyeccionesContextoActor es un conjunto cerrado de proyecciones
// pedidas. Su valor cero es el alcance vacio: el contexto conserva la
// representacion heredada, byte a byte. Es comparable con ==.
type AlcanceProyeccionesContextoActor struct {
	empleado bool
}

// NuevoAlcanceProyeccionesContextoActor rechaza valores desconocidos y
// repetidos en lugar de ignorarlos.
func NuevoAlcanceProyeccionesContextoActor(
	proyecciones ...ProyeccionContextoActor,
) (AlcanceProyeccionesContextoActor, error) {
	var alcance AlcanceProyeccionesContextoActor
	for _, proyeccion := range proyecciones {
		if proyeccion != ProyeccionContextoActorEmpleado || alcance.empleado {
			return AlcanceProyeccionesContextoActor{}, ErrAlcanceProyeccionesContextoActorInvalido
		}
		alcance.empleado = true
	}
	return alcance, nil
}

func (a AlcanceProyeccionesContextoActor) Vacio() bool { return !a.empleado }

func (a AlcanceProyeccionesContextoActor) IncluyeEmpleado() bool { return a.empleado }

// Proyecciones devuelve la lista canonica ordenada, nunca nil.
func (a AlcanceProyeccionesContextoActor) Proyecciones() []ProyeccionContextoActor {
	if a.empleado {
		return []ProyeccionContextoActor{ProyeccionContextoActorEmpleado}
	}
	return []ProyeccionContextoActor{}
}

// AlcanceProyecciones deriva el alcance del propio contexto: incluye empleado
// si, y solo si, su entrada de empleado procede de la proyeccion de Personal.
func (i InstantaneaContextoActor) AlcanceProyecciones() AlcanceProyeccionesContextoActor {
	for _, vinculo := range i.Vinculos {
		if vinculo.Tipo == TipoReferenciaContextoActorEmpleado &&
			len(vinculo.VinculoRef) > len(prefijoVinculoProyeccionEmpleado) &&
			vinculo.VinculoRef[:len(prefijoVinculoProyeccionEmpleado)] == prefijoVinculoProyeccionEmpleado {
			return AlcanceProyeccionesContextoActor{empleado: true}
		}
	}
	return AlcanceProyeccionesContextoActor{}
}

// AlcanceProyecciones del contexto consumible; ver InstantaneaContextoActor.
func (c ContextoActor) AlcanceProyecciones() AlcanceProyeccionesContextoActor {
	return c.Instantanea.AlcanceProyecciones()
}

// prefijoVinculoReferenciaValido admite vin_ para cualquier tipo y pep_ solo
// para la entrada de empleado que aporta la proyeccion de Personal.
func prefijoVinculoReferenciaValido(
	valor string,
	tipo TipoReferenciaContextoActor,
	valido func(string, string) bool,
) bool {
	return valido(valor, "vin_") ||
		(tipo == TipoReferenciaContextoActorEmpleado && valido(valor, prefijoVinculoProyeccionEmpleado))
}
