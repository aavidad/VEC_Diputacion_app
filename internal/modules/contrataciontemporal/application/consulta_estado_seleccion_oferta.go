package application

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var ErrConsultaEstadoSeleccionOfertaInvalida = errors.New("contratacion temporal: consulta de estado de oferta invalida")

// La consulta no abre ventanas ni reintenta publicaciones. Recuperar una
// intención pendiente exige el POST explícito con la clave aquí descubierta.
type ServicioConsultaEstadoSeleccionOfertaCT struct {
	preparador  ports.PreparadorConsultaEstadoSeleccionOfertaCT
	autorizador ports.AutorizadorConsultaEstadoSeleccionOfertaCT
	registro    ports.RegistroConsultaEstadoSeleccionOfertaCT
	lector      ports.LectorExpedienteSeleccionLlamamiento
}

func NuevoServicioConsultaEstadoSeleccionOfertaCT(p ports.PreparadorConsultaEstadoSeleccionOfertaCT,
	a ports.AutorizadorConsultaEstadoSeleccionOfertaCT,
	r ports.RegistroConsultaEstadoSeleccionOfertaCT,
	l ports.LectorExpedienteSeleccionLlamamiento,
) (*ServicioConsultaEstadoSeleccionOfertaCT, error) {
	if dependenciaNula(p) || dependenciaNula(a) || dependenciaNula(r) || dependenciaNula(l) {
		return nil, ErrServicioSeleccionOfertaInvalido
	}
	return &ServicioConsultaEstadoSeleccionOfertaCT{p, a, r, l}, nil
}

type DatosEstadoSeleccionOfertaParaAdaptador struct {
	Estado            string
	ExpedienteRef     string
	VersionEsperada   uint64
	ClaveIdempotencia string
	VentanaAbierta    bool
	Oferta            *DatosReciboSeleccionOfertaParaAdaptador
	Descarte          *DatosReciboDescarteOfertaParaAdaptador
}

func (s *ServicioConsultaEstadoSeleccionOfertaCT) ConsultarEstadoSeleccionOfertaParaAdaptador(
	ctx context.Context, expedienteRef string,
) (DatosEstadoSeleccionOfertaParaAdaptador, error) {
	vacio := DatosEstadoSeleccionOfertaParaAdaptador{}
	if s == nil || ctx == nil || dependenciaNula(s.preparador) || dependenciaNula(s.autorizador) ||
		dependenciaNula(s.registro) || dependenciaNula(s.lector) {
		return vacio, ErrServicioSeleccionOfertaInvalido
	}
	if !domain.ReferenciaOpacaValida(expedienteRef) {
		return vacio, ErrConsultaEstadoSeleccionOfertaInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	operacion, cancelar := context.WithTimeout(ctx, tiempoMaximoSeleccionLlamamiento)
	defer cancelar()
	q, err := s.preparador.PrepararConsultaEstadoSeleccionOfertaCT(operacion, expedienteRef)
	if err != nil {
		return vacio, normalizarFalloSeleccionOferta(operacion, err)
	}
	if q.Validar() != nil || q.ExpedienteRef != expedienteRef {
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	canon, err := ports.CanonConsultaEstadoSeleccionOfertaCT(q)
	if err != nil {
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	a, err := s.autorizador.AutorizarConsultaEstadoSeleccionOfertaCT(operacion, q)
	if err != nil {
		return vacio, normalizarFalloSeleccionOferta(operacion, err)
	}
	if a.ActorRef != q.ActorRef || a.PerfilRef != q.PerfilRef || a.MaterialCanonico != canon ||
		a.Material.ValidarEstructura() != nil {
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	estado, err := s.registro.ConsultarEstadoPorExpediente(operacion, q, a)
	if err != nil {
		return vacio, normalizarFalloSeleccionOferta(operacion, err)
	}
	if !estado.Encontrada {
		if !reflect.DeepEqual(estado.Intencion, ports.IntencionSeleccionOfertaCT{}) ||
			!reflect.DeepEqual(estado.Estado, ports.EstadoSeleccionOfertaCT{}) {
			return vacio, ErrResultadoSeleccionOfertaNoConfiable
		}
		return DatosEstadoSeleccionOfertaParaAdaptador{Estado: "ausente", ExpedienteRef: expedienteRef}, nil
	}
	i := estado.Intencion
	e := estado.Estado
	if i.Validar() != nil || i.OrganizacionRef != q.OrganizacionRef ||
		i.ExpedienteRef != q.ExpedienteRef || i.CategoriaRef != q.CategoriaRef ||
		i.CentroRef != q.CentroRef || i.ActorRef != q.ActorRef || i.PerfilRef != q.PerfilRef ||
		i.VersionExpediente > q.VersionObservada || e.Comando.ValidarPara(i) != nil {
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	historia, err := s.lector.LeerExpedienteParaSeleccion(operacion, q.OrganizacionRef,
		q.ExpedienteRef, i.VersionExpediente)
	if err != nil {
		return vacio, normalizarFalloSeleccionOferta(operacion, err)
	}
	h := historia.Fiscalizado
	if historia.VersionActual != q.VersionObservada {
		return vacio, ErrSeleccionOfertaConcurrente
	}
	if h.Validar() != nil ||
		h.Referencia != i.ExpedienteRef || h.Version != i.VersionExpediente ||
		int64(h.Solicitud.NumeroPersonas) != int64(e.Comando.NumeroPlazas) ||
		h.Solicitud.CentroRef != i.CentroRef || h.Solicitud.CategoriaRef != i.CategoriaRef ||
		h.Analisis == nil || h.Analisis.CategoriaRef != i.CategoriaRef {
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	d := DatosEstadoSeleccionOfertaParaAdaptador{ExpedienteRef: expedienteRef,
		VersionEsperada: i.VersionExpediente, ClaveIdempotencia: i.ClaveIdempotencia,
		VentanaAbierta: e.VentanaAbierta}
	switch e.Situacion {
	case ports.SeleccionOfertaReservada, ports.SeleccionOfertaIndeterminada:
		if !domain.ReferenciaOpacaValida(e.ReservaRef) ||
			e.Recibo != (ports.ReciboSeleccionOfertaCT{}) ||
			e.ReciboDescarte != (ports.ReciboDescarteOfertaCT{}) ||
			(e.Situacion == ports.SeleccionOfertaIndeterminada && !e.VentanaAbierta) {
			return vacio, ErrResultadoSeleccionOfertaNoConfiable
		}
		d.Estado = "pendiente"
	case ports.SeleccionOfertaConfirmada:
		if e.ReservaRef != "" || e.Recibo.ValidarPara(i, e.Comando) != nil ||
			e.ReciboDescarte != (ports.ReciboDescarteOfertaCT{}) {
			return vacio, ErrResultadoSeleccionOfertaNoConfiable
		}
		d.Estado = "confirmada"
		d.VentanaAbierta = false
		o := datosReciboSeleccionOfertaParaAdaptador(e.Recibo)
		d.Oferta = &o
	case ports.SeleccionOfertaDescartada:
		if e.ReservaRef != "" || e.Recibo != (ports.ReciboSeleccionOfertaCT{}) ||
			e.ReciboDescarte.ValidarTerminalPara(i, e.Comando) != nil {
			return vacio, ErrResultadoSeleccionOfertaNoConfiable
		}
		d.Estado = "descartada"
		d.VentanaAbierta = false
		x := DatosReciboDescarteOfertaParaAdaptador{Esquema: e.ReciboDescarte.Esquema,
			Estado: e.ReciboDescarte.Estado, OrganizacionRef: e.ReciboDescarte.OrganizacionRef,
			ExpedienteRef:        e.ReciboDescarte.ExpedienteRef,
			ReciboCierreBolsaRef: e.ReciboDescarte.ReciboCierreRef,
			ReciboDescarteRef:    e.ReciboDescarte.ReciboDescarteRef,
			DescartadaEn:         e.ReciboDescarte.DescartadaEn}
		d.Descarte = &x
	default:
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	return d, nil
}
