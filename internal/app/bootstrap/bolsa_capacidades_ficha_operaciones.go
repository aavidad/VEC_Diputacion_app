package bootstrap

import (
	"context"
	"net/http"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Esta superficie cubre exactamente los dos adjuntos opcionales del GET B8.
// Su registro se congela con los handlers que entran en el dispatcher real.
const superficieFichaOperacionesBolsa = "vec.bolsa.rrhh.ficha_operaciones"

type montajeFichaOperacionesBolsa struct {
	perfilRef                       string
	documentales, reincorporaciones bool
}

func (m montajeFichaOperacionesBolsa) VistaMontada(ctx context.Context, superficie, perfil string) (vecports.VistaMontajeCapacidadesRecurso, error) {
	if ctx == nil || ctx.Err() != nil || superficie != superficieFichaOperacionesBolsa || perfil != m.perfilRef || perfil == "" {
		return vecports.VistaMontajeCapacidadesRecurso{}, vecdomain.ErrConfiguracionAccesoInvalida
	}
	vista := vecports.VistaMontajeCapacidadesRecurso{Superficie: superficie, PerfilActivoRef: perfil, Completa: true}
	if m.documentales {
		vista.Rutas = append(vista.Rutas, vecports.RutaMontadaCapacidadRecurso{
			RutaID: claveFronteraSolicitudesDocumentalesRRHH, Metodo: http.MethodGet,
			ModuloID: bolsapuertos.ModuloSituacionParticipacion, TipoRecurso: bolsapuertos.TipoRecursoSituacionParticipacion,
			Accion:    bolsapuertos.AccionConsultarSolicitudesDocumentalesRRHH,
			Finalidad: bolsapuertos.FinalidadCambiarSituacionParticipacion,
		})
	}
	if m.reincorporaciones {
		vista.Rutas = append(vista.Rutas, vecports.RutaMontadaCapacidadRecurso{
			RutaID: claveFronteraReincorporacionesTitularBolsa, Metodo: http.MethodGet,
			ModuloID: bolsapuertos.ModuloSituacionParticipacion, TipoRecurso: bolsapuertos.TipoRecursoSituacionParticipacion,
			Accion:    bolsapuertos.AccionConsultarReincorporacionTitular,
			Finalidad: bolsapuertos.FinalidadConsultarReincorporacionTitular,
		})
	}
	return vista, nil
}

type proyectorDisponibilidadFichaOperacionesBolsa struct {
	preparador *preparadorBorradorLlamamientoDesarrollo
	proyector  *vecapp.ProyectorCapacidadesRecurso
}

func nuevoProyectorDisponibilidadFichaOperacionesBolsa(
	preparador *preparadorBorradorLlamamientoDesarrollo,
	fuente vecports.FuenteAutorizacion, reloj vecports.Reloj,
	documentales, reincorporaciones http.Handler,
) (bolsahttp.ProyectorDisponibilidadFichaOperaciones, error) {
	if preparador == nil || preparador.soporte == nil || preparador.soporte.soporteCanal == nil || fuente == nil || reloj == nil {
		return nil, vecdomain.ErrConfiguracionAccesoInvalida
	}
	datos, err := preparador.soporte.soporteCanal.contexto.Vinculo.Datos()
	if err != nil || datos.PerfilActivoRef == "" {
		return nil, vecdomain.ErrConfiguracionAccesoInvalida
	}
	montaje := montajeFichaOperacionesBolsa{perfilRef: datos.PerfilActivoRef,
		documentales: documentales != nil, reincorporaciones: reincorporaciones != nil}
	p, err := vecapp.NuevoProyectorCapacidadesRecurso(fuente, montaje, reloj)
	if err != nil {
		return nil, err
	}
	return proyectorDisponibilidadFichaOperacionesBolsa{preparador: preparador, proyector: p}, nil
}

func (p proyectorDisponibilidadFichaOperacionesBolsa) ProyectarDisponibilidadFichaOperaciones(
	ctx context.Context, q bolsapuertos.SolicitudCambiarSituacionParticipacion,
) (bolsahttp.DisponibilidadFichaOperaciones, error) {
	vacio := bolsahttp.DisponibilidadFichaOperaciones{}
	if p.preparador == nil || p.proyector == nil || ctx == nil || ctx.Err() != nil || q.Validar() != nil {
		return vacio, vecdomain.ErrConfiguracionAccesoInvalida
	}
	resuelto, err := p.preparador.ResolverContextoSituacionParticipacion(ctx, q.ResultadoContexto.Contexto, q.BolsaRef, q.ParticipacionRef)
	if err != nil || resuelto.Validar() != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	recurso := func() vecdomain.RecursoAutorizable {
		return vecdomain.RecursoAutorizable{Referencia: q.ParticipacionRef,
			ModuloID: bolsapuertos.ModuloSituacionParticipacion, Tipo: bolsapuertos.TipoRecursoSituacionParticipacion,
			Ambitos: map[string]string{"unidad_ref": resuelto.UnidadRef, "ambito_ref": resuelto.AmbitoRef}}
	}
	proyeccion, err := p.proyector.Proyectar(ctx, vecports.LoteRecursosLeidosCapacidad{
		Vinculo: q.Vinculo, Resultado: q.ResultadoContexto, Superficie: superficieFichaOperacionesBolsa,
		Recursos: []vecports.RecursoLeidoCapacidad{
			{RutaID: claveFronteraSolicitudesDocumentalesRRHH, Metodo: http.MethodGet,
				Accion:    bolsapuertos.AccionConsultarSolicitudesDocumentalesRRHH,
				Finalidad: bolsapuertos.FinalidadCambiarSituacionParticipacion, Recurso: recurso()},
			{RutaID: claveFronteraReincorporacionesTitularBolsa, Metodo: http.MethodGet,
				Accion:    bolsapuertos.AccionConsultarReincorporacionTitular,
				Finalidad: bolsapuertos.FinalidadConsultarReincorporacionTitular, Recurso: recurso()},
		},
	})
	if err != nil || len(proyeccion.Resultados) != 2 || proyeccion.Resultados[0].Indice != 0 || proyeccion.Resultados[1].Indice != 1 {
		return vacio, vecdomain.ErrConfiguracionAccesoInvalida
	}
	estado := func(i int) bolsahttp.EstadoDisponibilidadFichaOperaciones {
		return bolsahttp.EstadoDisponibilidadFichaOperaciones{Estado: string(proyeccion.Resultados[i].Estado),
			BolsaRef: q.BolsaRef, ParticipacionRef: q.ParticipacionRef}
	}
	return bolsahttp.DisponibilidadFichaOperaciones{
		SolicitudesDocumentales: estado(0), ReincorporacionesTitular: estado(1),
	}, nil
}
