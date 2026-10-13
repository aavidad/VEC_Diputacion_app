package bootstrap

import (
	"context"
	"slices"
	"strings"

	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

type loteNombreAlegacionB7 struct{ huella, categoria string }

// El sobre privado B114 ya liga cada participación autorizada a su acta B7
// vigente y a una fila. RecuperarLote es la única salida adicional: ninguna
// consulta SQL por acta ni catálogo global se ejecuta después del acuse.
func (f *fuenteConstituidaRRHHDesarrollo) nombresAlegacionesB7(ctx context.Context,
	alegaciones []inscripcion.Alegacion) (map[string]string, error) {
	if ctx == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	if f == nil || f.recuperador == nil || len(alegaciones) == 0 {
		return nil, nil
	}
	grupos := make(map[loteNombreAlegacionB7][]inscripcion.MetadatoNombreAlegacionB7)
	for _, a := range alegaciones {
		if a.MetadatoNombreB7 == nil {
			continue
		}
		m := *a.MetadatoNombreB7
		grupos[loteNombreAlegacionB7{m.HuellaListadoSHA256, m.CategoriaRef}] = append(
			grupos[loteNombreAlegacionB7{m.HuellaListadoSHA256, m.CategoriaRef}], m)
	}
	nombres := make(map[string]string)
	for clave, metadatos := range grupos {
		lote, _, existe, err := f.recuperador.RecuperarLote(ctx, clave.huella, clave.categoria)
		if err != nil {
			registrarFalloFuenteConstituidaRRHHDesarrollo("alegaciones_recuperar_acta", err)
			return nil, inscripcion.ErrNoDisponible
		}
		if !existe {
			continue
		}
		if lote.Acta.HuellaFicheroSHA256 != clave.huella || lote.Acta.CategoriaRef != clave.categoria {
			return nil, inscripcion.ErrNoDisponible
		}
		filas := make(map[int]string, len(lote.Aceptadas))
		for _, fila := range lote.Aceptadas {
			if _, duplicada := filas[fila.Numero]; duplicada {
				return nil, inscripcion.ErrNoDisponible
			}
			nombre := strings.Join(strings.Fields(strings.Join([]string{fila.Identidad.Nombre,
				fila.Identidad.PrimerApellido, fila.Identidad.SegundoApellido}, " ")), " ")
			filas[fila.Numero] = nombre
		}
		for _, m := range metadatos {
			nombre, ok := filas[m.FilaNumero]
			if m.ActaRef != lote.Acta.ActaRef || !ok || nombre == "" || len(nombre) > 512 {
				return nil, inscripcion.ErrNoDisponible
			}
			nombres[m.ParticipacionRef] = nombre
		}
	}
	return nombres, nil
}

// La envoltura recibe sólo el resultado de la lectura RRHH autorizada y
// auditada. El nombre no se solicita para objetos sin procedencia B7 vigente.
type aplicacionAlegacionesRRHHConNombres struct {
	httpinscripcion.AplicacionAlegaciones
	fuente *fuenteConstituidaRRHHDesarrollo
}

func (a aplicacionAlegacionesRRHHConNombres) AlegacionesRRHH(ctx context.Context, actor inscripcion.Actor,
	tipo, ref string, limite int, cursor string) (inscripcion.PaginaAlegaciones, error) {
	pagina, err := a.AplicacionAlegaciones.AlegacionesRRHH(ctx, actor, tipo, ref, limite, cursor)
	if err != nil || a.fuente == nil || actor.Canal != "interna_corporativa" || actor.Lectura == nil ||
		!slices.Contains(actor.Lectura.Campos, "alegaciones[].persona_resumen") {
		return pagina, err
	}
	nombres, err := a.fuente.nombresAlegacionesB7(ctx, pagina.Alegaciones)
	if err != nil {
		return inscripcion.PaginaAlegaciones{}, err
	}
	for i := range pagina.Alegaciones {
		if nombre := nombres[pagina.Alegaciones[i].Objeto.ParticipacionRef]; nombre != "" {
			pagina.Alegaciones[i].PersonaResumen = nombre
		}
	}
	return pagina, nil
}

func (a aplicacionAlegacionesRRHHConNombres) AlegacionRRHH(ctx context.Context, actor inscripcion.Actor,
	ref string) (inscripcion.Alegacion, error) {
	detalle, err := a.AplicacionAlegaciones.AlegacionRRHH(ctx, actor, ref)
	if err != nil || a.fuente == nil || actor.Canal != "interna_corporativa" || actor.Lectura == nil ||
		!slices.Contains(actor.Lectura.Campos, "persona_resumen") {
		return detalle, err
	}
	nombres, err := a.fuente.nombresAlegacionesB7(ctx, []inscripcion.Alegacion{detalle})
	if err != nil {
		return inscripcion.Alegacion{}, err
	}
	if nombre := nombres[detalle.Objeto.ParticipacionRef]; nombre != "" {
		detalle.PersonaResumen = nombre
	}
	return detalle, nil
}
