package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const plazoConfirmacionEfectoDocumental = 3 * time.Second

// contenidoGeneracionDocumental mantiene juntos los bytes ya validados y la
// declaracion exacta que los compromete. Ningun valor de esta estructura se
// obtiene del PDP ni se reconstruye despues de reservar el efecto.
type contenidoGeneracionDocumental struct {
	declaracion ports.DeclaracionRepresentacionGeneracionDocumental
	contenido   []byte
}

type solicitudPlanGeneracionDocumental struct {
	principal       domain.Principal
	perfilActivo    string
	finalidad       string
	motivo          string
	correlacionRef  string
	clasificacion   string
	plantilla       domain.PlantillaDocumento
	recursoBase     domain.RecursoAutorizable
	sujetoRef       string
	ambitoSujetoRef string
	operacionRef    string
	cargaRef        string
	efectoRef       string
	huellaSolicitud string
	contenidos      []contenidoGeneracionDocumental
}

type resultadoPasoGeneracionDocumental struct {
	objeto             ports.ReferenciaObjetoAlmacen
	conectorID         string
	evidenciaOperacion string
}

type resultadoPlanGeneracionDocumental struct {
	decision        domain.DecisionAutorizacion
	efectoReservado bool
	pasos           map[string]resultadoPasoGeneracionDocumental
}

// ejecutarPlanGeneracionDocumental es el unico camino de escritura para una
// generacion simple (N=1) o logica (N>0). Los bytes deben llegar renderizados
// y validados: solo entonces se construye el manifiesto, se vincula el recurso
// que vera el PDP y se reserva de forma durable la DecisionRef antes del
// primer efecto remoto.
func (s *ServicioDocumental) ejecutarPlanGeneracionDocumental(
	ctx context.Context,
	solicitud solicitudPlanGeneracionDocumental,
) (resultadoPlanGeneracionDocumental, error) {
	if ctx == nil || s.registroEfectos == nil || len(solicitud.contenidos) == 0 {
		return resultadoPlanGeneracionDocumental{}, ErrDependenciaDocumentalRequerida
	}
	if err := ctx.Err(); err != nil {
		return resultadoPlanGeneracionDocumental{}, err
	}

	declaraciones := make([]ports.DeclaracionRepresentacionGeneracionDocumental, 0, len(solicitud.contenidos))
	porReferencia := make(map[string]contenidoGeneracionDocumental, len(solicitud.contenidos))
	for _, preparado := range solicitud.contenidos {
		declaraciones = append(declaraciones, preparado.declaracion)
		porReferencia[preparado.declaracion.ReferenciaLogica] = contenidoGeneracionDocumental{
			declaracion: preparado.declaracion,
			contenido:   append([]byte(nil), preparado.contenido...),
		}
	}
	if len(porReferencia) != len(solicitud.contenidos) {
		return resultadoPlanGeneracionDocumental{}, ports.ErrManifiestoGeneracionDocumentalInvalido
	}
	manifiesto, err := ports.NuevoManifiestoGeneracionDocumental(solicitud.plantilla, declaraciones)
	if err != nil {
		return resultadoPlanGeneracionDocumental{}, err
	}
	proyeccionManifiesto, err := manifiesto.Proyeccion()
	if err != nil || proyeccionManifiesto.PermisoGenerar != solicitud.plantilla.PermisoGenerar {
		return resultadoPlanGeneracionDocumental{}, ports.ErrManifiestoGeneracionDocumentalInvalido
	}

	seudonimoSujeto, err := s.seudonimizarSujetoDocumento(ctx, solicitud.sujetoRef, solicitud.ambitoSujetoRef)
	if err != nil {
		return resultadoPlanGeneracionDocumental{}, err
	}
	vinculos := ports.VinculosOperacionAlmacen{
		OperacionRef: solicitud.operacionRef, CargaRef: solicitud.cargaRef,
		Clasificacion: solicitud.clasificacion, SujetoSeudonimoHMAC: seudonimoSujeto,
		HuellaSolicitudHMAC: solicitud.huellaSolicitud, EfectoRef: solicitud.efectoRef,
	}
	recurso, err := ports.VincularRecursoGeneracionDocumental(solicitud.recursoBase, manifiesto, vinculos)
	if err != nil {
		return resultadoPlanGeneracionDocumental{}, err
	}

	decision, err := s.exigirAutorizacionDocumental(
		ctx, solicitud.principal, solicitud.perfilActivo, proyeccionManifiesto.PermisoGenerar,
		recurso, solicitud.finalidad, solicitud.correlacionRef, solicitud.motivo,
	)
	if err != nil {
		return resultadoPlanGeneracionDocumental{}, err
	}
	verificadaEn := s.reloj.Ahora().UTC()
	contextoInicial, err := ports.NuevoContextoGeneracionDocumentalAlmacen(
		decision, recurso, manifiesto, vinculos, verificadaEn,
	)
	if err != nil {
		return resultadoPlanGeneracionDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}

	peticionReserva := ports.SolicitudReservarEfectoGeneracionDocumental{
		Contexto: contextoInicial, Manifiesto: manifiesto,
	}
	instanteReserva := s.reloj.Ahora().UTC()
	if err := peticionReserva.ValidarEn(instanteReserva); err != nil {
		return resultadoPlanGeneracionDocumental{}, errors.Join(domain.ErrAutorizacionDenegada, err)
	}
	reserva, err := s.registroEfectos.ReservarEfectoGeneracionDocumental(ctx, peticionReserva)
	if err != nil {
		return resultadoPlanGeneracionDocumental{}, fmt.Errorf("reservar efecto de generacion documental: %w", err)
	}
	if err := reserva.ValidarContra(peticionReserva); err != nil {
		return resultadoPlanGeneracionDocumental{}, err
	}
	resultadoParcial := resultadoPlanGeneracionDocumental{
		decision: decision, efectoReservado: true,
	}

	resultados := make(map[string]resultadoPasoGeneracionDocumental, len(proyeccionManifiesto.Pasos))
	for indice, paso := range proyeccionManifiesto.Pasos {
		estado := reserva.Pasos[indice]
		preparado, existe := porReferencia[paso.ReferenciaLogica]
		if !existe || preparado.declaracion.ClaveIdempotencia != paso.ClaveIdempotencia ||
			preparado.declaracion.HuellaSHA256 != paso.HuellaSHA256 {
			return resultadoParcial, ports.ErrManifiestoGeneracionDocumentalInvalido
		}
		switch estado.Estado {
		case ports.EstadoPasoEfectoDocumentalIndeterminado:
			return resultadoParcial, errors.Join(
				ports.ErrPasoGeneracionDocumentalIndeterminado,
				fmt.Errorf("incidente documental %s", estado.IncidenteRef),
			)
		case ports.EstadoPasoEfectoDocumentalConfirmado:
			resultados[paso.ReferenciaLogica] = resultadoPasoGeneracionDocumental{
				objeto: estado.Objeto, conectorID: estado.ConectorID,
				evidenciaOperacion: estado.EvidenciaOperacionRef,
			}
			continue
		case ports.EstadoPasoEfectoDocumentalReservado:
		default:
			return resultadoParcial, ports.ErrReservaEfectoGeneracionDocumentalInvalida
		}

		contextoPaso, err := contextoInicial.DerivarPaso(paso.PasoRef)
		if err != nil {
			return resultadoParcial, err
		}
		peticionGuardado := ports.SolicitudGuardarContenido{
			Contexto: contextoPaso, ClaveIdempotencia: paso.ClaveIdempotencia,
			DocumentoID: paso.ReferenciaLogica, Zona: paso.Zona, MIME: paso.MIME,
			HuellaSHA256: paso.HuellaSHA256, Tamano: paso.Tamano,
			Contenido: append([]byte(nil), preparado.contenido...),
		}
		// Esta comprobacion se mantiene inmediatamente pegada al efecto remoto.
		// Una decision que caduque durante renderizado, reserva o pasos previos no
		// llega al conector de objetos.
		if err := peticionGuardado.ValidarEn(s.reloj.Ahora().UTC()); err != nil {
			return resultadoParcial, errors.Join(domain.ErrAutorizacionDenegada, err)
		}
		guardado, err := s.almacen.GuardarContenido(ctx, peticionGuardado)
		if err != nil {
			return resultadoParcial, s.marcarPasoDocumentalIndeterminado(
				ctx, reserva.ReservaRef, contextoPaso, paso.HuellaPasoSHA256, err,
			)
		}
		if err := guardado.ValidarContra(peticionGuardado); err != nil {
			return resultadoParcial, s.marcarPasoDocumentalIndeterminado(
				ctx, reserva.ReservaRef, contextoPaso, paso.HuellaPasoSHA256, err,
			)
		}
		confirmacion := ports.SolicitudConfirmarPasoGeneracionDocumental{
			ReservaRef: reserva.ReservaRef, Contexto: contextoPaso, Guardado: guardado,
		}
		if err := confirmacion.Validar(); err != nil {
			return resultadoParcial, s.marcarPasoDocumentalIndeterminado(
				ctx, reserva.ReservaRef, contextoPaso, paso.HuellaPasoSHA256, err,
			)
		}
		ctxConfirmacion, cancelar := contextoDuraderoDocumental(ctx)
		err = s.registroEfectos.ConfirmarPasoGeneracionDocumental(ctxConfirmacion, confirmacion)
		cancelar()
		if err != nil {
			return resultadoParcial, s.marcarPasoDocumentalIndeterminado(
				ctx, reserva.ReservaRef, contextoPaso, paso.HuellaPasoSHA256, err,
			)
		}
		resultados[paso.ReferenciaLogica] = resultadoPasoGeneracionDocumental{
			objeto:             ports.ReferenciaObjetoAlmacen{Referencia: guardado.Referencia, Version: guardado.Version},
			conectorID:         guardado.ConectorID,
			evidenciaOperacion: guardado.EvidenciaOperacion.Referencia,
		}
	}
	if len(resultados) != len(proyeccionManifiesto.Pasos) {
		return resultadoParcial, ports.ErrReservaEfectoGeneracionDocumentalInvalida
	}
	resultadoParcial.pasos = resultados
	return resultadoParcial, nil
}

func (s *ServicioDocumental) marcarPasoDocumentalIndeterminado(
	ctx context.Context,
	reservaRef string,
	contexto ports.ContextoOperacionAlmacen,
	huellaPaso string,
	causa error,
) error {
	incidenteRef := "incidente:generacion-documental:" + huellaPaso
	solicitud := ports.SolicitudMarcarPasoGeneracionDocumentalIndeterminado{
		ReservaRef: reservaRef, Contexto: contexto, IncidenteRef: incidenteRef,
	}
	if err := solicitud.Validar(); err != nil {
		return errors.Join(ports.ErrPasoGeneracionDocumentalIndeterminado, causa, err)
	}
	ctxDuradero, cancelar := contextoDuraderoDocumental(ctx)
	err := s.registroEfectos.MarcarPasoGeneracionDocumentalIndeterminado(ctxDuradero, solicitud)
	cancelar()
	return errors.Join(ports.ErrPasoGeneracionDocumentalIndeterminado, causa, err)
}

func contextoDuraderoDocumental(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(ctx), plazoConfirmacionEfectoDocumental)
}
