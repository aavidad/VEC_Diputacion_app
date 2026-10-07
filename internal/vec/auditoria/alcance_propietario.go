package auditoria

import (
	"time"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// RecursoFiltroConAsignacionNominal extrae el ámbito de una asignación V3
// obtenida por la autoridad del servidor. No consulta ni autoriza el recurso.
// Antes de emitir V3, el propietario debe comprobar existencia y competencia
// dentro de la transacción que después consume la decisión y audita la lectura;
// una consulta previa separada no acredita esa pertenencia.
func RecursoFiltroConAsignacionNominal(
	f Filtro, contexto ContextoConsulta,
	asignacion vecdomain.AsignacionPerfil, ahora time.Time,
) (vecdomain.RecursoAutorizable, error) {
	var vacio vecdomain.RecursoAutorizable
	if f.Validar() != nil || contexto.Resultado.Validar() != nil ||
		contexto.Vinculo.ValidarPara(contexto.Resultado) != nil ||
		ahora.IsZero() || ahora.Location() != time.UTC || ahora.Nanosecond()%1000 != 0 ||
		!contexto.Vinculo.VigenteEn(ahora, contexto.Resultado) ||
		asignacion.Validar() != nil ||
		!asignacion.VigenteEn(ahora) ||
		asignacion.PrincipalID != contexto.Resultado.Contexto.Principal.ID ||
		asignacion.PerfilActivoRef != contexto.Resultado.Contexto.PerfilActivoRef ||
		len(asignacion.Ambitos) != 2 {
		return vacio, ErrDenegada
	}
	dimension := "organizacion_ref"
	if f.Fuente == "bolsa" {
		dimension = "bolsa_ref"
	}
	var fuente, alcance string
	for _, ambito := range asignacion.Ambitos {
		if len(ambito.Valores) != 1 {
			return vacio, ErrDenegada
		}
		switch ambito.Clave {
		case "fuente":
			fuente = ambito.Valores[0]
		case dimension:
			alcance = ambito.Valores[0]
		default:
			return vacio, ErrDenegada
		}
	}
	if fuente != f.Fuente || alcance == "" {
		return vacio, ErrDenegada
	}
	huella, err := HuellaFiltro(f)
	if err != nil {
		return vacio, ErrDenegada
	}
	recurso := vecdomain.RecursoAutorizable{
		Referencia: f.ExpedienteRef, ModuloID: ModuloAutorizacion, Tipo: TipoRecurso,
		Ambitos:   map[string]string{"fuente": fuente, dimension: alcance},
		Atributos: map[string]string{"filtro_sha256": huella},
	}
	if recurso.Validar() != nil || !asignacion.Cubre(recurso) {
		return vacio, ErrDenegada
	}
	return recurso, nil
}
