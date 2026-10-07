package application

import (
	"context"
	"errors"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// OperadorContactosRRHH es el servicio de contactos existente. Sus dos
// lecturas ya consumen la autorización V3 en su transacción positiva; el
// decorador solo deja rastro AD169 de las denegaciones y errores.
type OperadorContactosRRHH interface {
	RegistrarContactoParticipacion(context.Context, ports.SolicitudRegistrarContactoParticipacion) (ports.RegistroContactoParticipacion, error)
	ListarContactosParticipacion(context.Context, ports.ConsultaContactosParticipacion) (ports.PaginaContactosParticipacion, error)
	ListarContactosBolsa(context.Context, ports.ConsultaContactosBolsa) (ports.PaginaContactosParticipacion, error)
}

// evaluadorIntentosContactosRRHH es opcional en el servicio envuelto: el
// transporte lo descubre por aserción y el decorador no debe ocultarlo.
type evaluadorIntentosContactosRRHH interface {
	EstadoIntentosTelefonicos(context.Context, string, []dominiobolsa.ContactoParticipacion, bool) (ports.EstadoIntentosContacto, error)
}

type ContactosRRHHAuditados struct {
	operador    OperadorContactosRRHH
	registrador vecports.RegistradorIntentosAuditoria
	proceso     string
}

func NuevosContactosRRHHAuditados(operador OperadorContactosRRHH,
	registrador vecports.RegistradorIntentosAuditoria, proceso string,
) (*ContactosRRHHAuditados, error) {
	if dependenciaLlamamientoNula(operador) || dependenciaLlamamientoNula(registrador) || !procesoAuditoriaDocumentales.MatchString(proceso) {
		return nil, ports.ErrContactoParticipacionNoDisponible
	}
	return &ContactosRRHHAuditados{operador: operador, registrador: registrador, proceso: proceso}, nil
}

// RegistrarContactoParticipacion es una escritura fuera de este corte: se
// delega sin cambios, con su propia autorización e idempotencia.
func (s *ContactosRRHHAuditados) RegistrarContactoParticipacion(ctx context.Context,
	q ports.SolicitudRegistrarContactoParticipacion,
) (ports.RegistroContactoParticipacion, error) {
	if s == nil || dependenciaLlamamientoNula(s.operador) {
		return ports.RegistroContactoParticipacion{}, ports.ErrContactoParticipacionNoDisponible
	}
	return s.operador.RegistrarContactoParticipacion(ctx, q)
}

// EstadoIntentosTelefonicos conserva la conducta del servicio envuelto. Sin
// evaluador devuelve un estado no configurado, igual que el transporte.
func (s *ContactosRRHHAuditados) EstadoIntentosTelefonicos(ctx context.Context, llamamientoRef string,
	contactos []dominiobolsa.ContactoParticipacion, completo bool,
) (ports.EstadoIntentosContacto, error) {
	if s == nil {
		return ports.EstadoIntentosContacto{}, ports.ErrContactoParticipacionNoDisponible
	}
	evaluador, ok := s.operador.(evaluadorIntentosContactosRRHH)
	if !ok || dependenciaLlamamientoNula(evaluador) {
		return ports.EstadoIntentosContacto{LlamamientoRef: llamamientoRef}, nil
	}
	return evaluador.EstadoIntentosTelefonicos(ctx, llamamientoRef, contactos, completo)
}

func (s *ContactosRRHHAuditados) ListarContactosParticipacion(ctx context.Context,
	q ports.ConsultaContactosParticipacion,
) (ports.PaginaContactosParticipacion, error) {
	if q.ParticipacionRef == "" {
		return ports.PaginaContactosParticipacion{}, falloAuditoriaConsultaContactos()
	}
	sobre := sobreConsultaContactos{resultado: q.ResultadoContexto, vinculo: q.Vinculo, correlacion: q.Correlacion,
		motivo: q.MotivoAutorizacion, recurso: q.ParticipacionRef, bolsa: q.BolsaRef, participacion: q.ParticipacionRef,
		oferta: q.OfertaRef, limite: q.Limite}
	return s.auditar(ctx, sobre, func() (ports.PaginaContactosParticipacion, error) {
		return s.operador.ListarContactosParticipacion(ctx, q)
	})
}

func (s *ContactosRRHHAuditados) ListarContactosBolsa(ctx context.Context,
	q ports.ConsultaContactosBolsa,
) (ports.PaginaContactosParticipacion, error) {
	// El servicio autoriza la lista de la bolsa con la bolsa como recurso.
	sobre := sobreConsultaContactos{resultado: q.ResultadoContexto, vinculo: q.Vinculo, correlacion: q.Correlacion,
		motivo: q.MotivoAutorizacion, recurso: q.BolsaRef, bolsa: q.BolsaRef, oferta: q.OfertaRef, limite: q.Limite}
	return s.auditar(ctx, sobre, func() (ports.PaginaContactosParticipacion, error) {
		return s.operador.ListarContactosBolsa(ctx, q)
	})
}

// sobreConsultaContactos reúne solo referencias opacas de la consulta. Nunca
// contiene correo, teléfonos ni anotaciones de los contactos leídos.
type sobreConsultaContactos struct {
	resultado                             core.ResultadoContextoActorRegistradoV2
	vinculo                               core.VinculoAutenticacionActorV2
	correlacion                           core.ReferenciaCorrelacionAutorizacionV2
	motivo                                core.ReferenciaEntradaCatalogo
	recurso, bolsa, participacion, oferta string
	limite                                int
}

func (s *ContactosRRHHAuditados) auditar(ctx context.Context, q sobreConsultaContactos,
	leer func() (ports.PaginaContactosParticipacion, error),
) (ports.PaginaContactosParticipacion, error) {
	vacia := ports.PaginaContactosParticipacion{}
	if ctx == nil || s == nil || dependenciaLlamamientoNula(s.operador) || dependenciaLlamamientoNula(s.registrador) ||
		!procesoAuditoriaDocumentales.MatchString(s.proceso) || q.bolsa == "" || q.recurso == "" || q.limite < 1 || q.limite > 100 ||
		q.resultado.Validar() != nil || q.vinculo.ValidarPara(q.resultado) != nil {
		return vacia, falloAuditoriaConsultaContactos()
	}
	actor, err := q.resultado.Clonar()
	vinculo, errVinculo := q.vinculo.Datos()
	correlacion, errCorrelacion := q.correlacion.ValorCanonico()
	if err != nil || errVinculo != nil || errCorrelacion != nil || vinculo.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 {
		return vacia, falloAuditoriaConsultaContactos()
	}
	// El actor se captura antes de leer: una revocación o desconexión
	// posterior no sustituye la identidad de este intento.
	pagina, causa := leer()
	if ctx.Err() != nil {
		causa = ctx.Err()
	} else if causa != nil && (len(pagina.Contactos) != 0 || pagina.CursorSiguiente != "") {
		causa = ports.ErrContactoParticipacionNoDisponible
	} else if causa == nil {
		causa = validarPaginaContactosRRHH(pagina, q)
	}
	if causa == nil {
		return pagina, nil // El servicio ya confirmó la lectura y su consumo V3.
	}
	referencia, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return vacia, falloAuditoriaConsultaContactos()
	}
	orden, err := vecports.NuevaOrdenIntentoAuditoria(referencia, actor, q.vinculo, core.DatosIntentoAuditoria{
		Accion: ports.AccionConsultarContactoParticipacion, ModuloID: ports.ModuloSituacionParticipacion,
		RecursoRef: q.recurso, FinalidadRef: ports.FinalidadConsultarContactoParticipacion,
		Resultado: resultadoFalloConsultaContactos(causa), Motivo: q.motivo,
		Proceso: s.proceso, Canal: string(vinculo.Superficie), CorrelacionRef: correlacion,
	})
	if err != nil {
		return vacia, falloAuditoriaConsultaContactos()
	}
	// El registro AD169 usa su propia transacción y no depende del plazo HTTP.
	acuse, err := s.registrador.AppendIntentoAuditoria(context.WithoutCancel(ctx), orden)
	if err != nil || acuse.ValidarPara(orden) != nil {
		return vacia, falloAuditoriaConsultaContactos()
	}
	return vacia, falloConsultaContactosAuditado{causa: causa, acuse: acuse}
}

// validarPaginaContactosRRHH cierra una página incoherente con la consulta
// antes de que alcance el serializador HTTP.
func validarPaginaContactosRRHH(p ports.PaginaContactosParticipacion, q sobreConsultaContactos) error {
	if len(p.Contactos) > q.limite {
		return ports.ErrContactoParticipacionNoDisponible
	}
	for _, c := range p.Contactos {
		if c.Validar() != nil || c.BolsaRef != q.bolsa || (q.participacion != "" && c.ParticipacionRef != q.participacion) ||
			(q.oferta != "" && c.OfertaRef != q.oferta) {
			return ports.ErrContactoParticipacionNoDisponible
		}
	}
	if (len(p.Contactos) == 0 && p.CursorSiguiente != "") ||
		(len(p.Contactos) != 0 && p.CursorSiguiente != p.Contactos[len(p.Contactos)-1].ContactoRef) {
		return ports.ErrContactoParticipacionNoDisponible
	}
	return nil
}

func falloAuditoriaConsultaContactos() error {
	return errors.Join(ports.ErrContactoParticipacionNoDisponible, vecports.ErrIntentoAuditoriaNoDisponible)
}

// Un fallo técnico prevalece sobre cualquier denegación que lo acompañe; la
// clasificación común es la misma que la de la consulta documental.
func resultadoFalloConsultaContactos(causa error) core.ResultadoIntentoAuditoria {
	if errors.Is(causa, ports.ErrContactoParticipacionNoDisponible) {
		return core.ResultadoIntentoAuditoriaError
	}
	return resultadoFalloConsultaDocumentales(causa)
}

type falloConsultaContactosAuditado struct {
	causa error
	acuse vecports.AcuseIntentoAuditoria
}

func (falloConsultaContactosAuditado) Error() string {
	return "bolsa: consulta de contactos fallida auditada"
}
func (e falloConsultaContactosAuditado) Unwrap() error { return e.causa }

// AcuseConsultaContactosFallida devuelve el acuse ya validado de un intento
// fallido para que el transporte publique sus referencias.
func AcuseConsultaContactosFallida(err error) (vecports.AcuseIntentoAuditoria, bool) {
	var auditado falloConsultaContactosAuditado
	if !errors.As(err, &auditado) {
		return vecports.AcuseIntentoAuditoria{}, false
	}
	return auditado.acuse, true
}
