package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioSeleccionOfertaInvalido     = errors.New("contratacion temporal: servicio de seleccion de oferta invalido")
	ErrSolicitudSeleccionOfertaInvalida    = errors.New("contratacion temporal: solicitud de seleccion de oferta invalida")
	ErrSeleccionOfertaNoDisponible         = errors.New("contratacion temporal: seleccion de oferta no disponible")
	ErrResultadoSeleccionOfertaNoConfiable = errors.New("contratacion temporal: resultado de seleccion de oferta no confiable")
	ErrClaveSeleccionOfertaEnColision      = errors.New("contratacion temporal: clave de oferta usada con otra operacion")
	ErrSeleccionOfertaConcurrente          = errors.New("contratacion temporal: seleccion de oferta concurrente")
	ErrSeleccionOfertaIndeterminada        = errors.New("contratacion temporal: seleccion de oferta indeterminada")
)

type ServicioSeleccionOfertaCT struct {
	preparador  ports.PreparadorSeleccionOfertaCT
	autorizador ports.AutorizadorSeleccionOfertaCT
	registro    ports.RegistroSeleccionOfertaCT
	publicador  ports.PublicadorOfertaBolsaCT
	reloj       ports.Reloj
}

func NuevoServicioSeleccionOfertaCT(
	preparador ports.PreparadorSeleccionOfertaCT,
	autorizador ports.AutorizadorSeleccionOfertaCT,
	registro ports.RegistroSeleccionOfertaCT,
	publicador ports.PublicadorOfertaBolsaCT,
	reloj ports.Reloj,
) (*ServicioSeleccionOfertaCT, error) {
	if dependenciaNula(preparador) || dependenciaNula(autorizador) ||
		dependenciaNula(registro) || dependenciaNula(publicador) || dependenciaNula(reloj) {
		return nil, ErrServicioSeleccionOfertaInvalido
	}
	return &ServicioSeleccionOfertaCT{preparador, autorizador, registro, publicador, reloj}, nil
}

// SeleccionarOferta consulta primero la ejecución durable de la intención.
// Solo si está ausente lee Bolsa, congela la captura B110 y reserva. Un
// reintento pendiente publica con el comando guardado, sin refrescar corte,
// candidatos, política ni datos del expediente.
func (s *ServicioSeleccionOfertaCT) SeleccionarOferta(
	ctx context.Context, q SolicitudSeleccionLlamamiento,
) (ports.ReciboSeleccionOfertaCT, error) {
	vacio := ports.ReciboSeleccionOfertaCT{}
	if s == nil || ctx == nil || dependenciaNula(s.preparador) ||
		dependenciaNula(s.autorizador) || dependenciaNula(s.registro) ||
		dependenciaNula(s.publicador) || dependenciaNula(s.reloj) {
		return vacio, ErrServicioSeleccionOfertaInvalido
	}
	if !domain.ReferenciaOpacaValida(q.ExpedienteRef) || q.VersionEsperada == 0 ||
		q.VersionEsperada > 1<<53-1 || !ports.ClaveIdempotenciaValida(q.ClaveIdempotencia) {
		return vacio, ErrSolicitudSeleccionOfertaInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	operacion, cancelar := context.WithTimeout(ctx, tiempoMaximoSeleccionLlamamiento)
	defer cancelar()
	i, err := s.preparador.PrepararIntencion(operacion, q.ExpedienteRef, q.VersionEsperada, q.ClaveIdempotencia)
	if err != nil {
		return vacio, normalizarFalloSeleccionOferta(operacion, err)
	}
	if i.Validar() != nil || i.ExpedienteRef != q.ExpedienteRef ||
		i.VersionExpediente != q.VersionEsperada || i.ClaveIdempotencia != q.ClaveIdempotencia {
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	consultaAut, err := s.autorizar(operacion, ports.EtapaSeleccionOfertaConsultar, i, nil, nil)
	if err != nil {
		return vacio, err
	}
	estado, err := s.registro.Consultar(operacion, i, consultaAut)
	if err != nil {
		return vacio, normalizarFalloSeleccionOferta(operacion, err)
	}
	if estado.Situacion == ports.SeleccionOfertaConfirmada {
		if estado.ReservaRef != "" || estado.Comando.ValidarPara(i) != nil ||
			estado.Recibo.ValidarPara(i, estado.Comando) != nil {
			return vacio, ErrResultadoSeleccionOfertaNoConfiable
		}
		return estado.Recibo, nil
	}
	var reserva ports.ReservaSeleccionOfertaCT
	switch estado.Situacion {
	case ports.SeleccionOfertaAusente:
		if estado.ReservaRef != "" || estado.Recibo != vacio ||
			!reflect.DeepEqual(estado.Comando, ports.ComandoOfertaBolsaCT{}) {
			return vacio, ErrResultadoSeleccionOfertaNoConfiable
		}
		comando, err := s.preparador.PrepararComando(operacion, i)
		if err != nil {
			return vacio, normalizarFalloSeleccionOferta(operacion, err)
		}
		if comando.ValidarPara(i) != nil {
			return vacio, ErrSeleccionOfertaNoDisponible
		}
		aut, err := s.autorizar(operacion, ports.EtapaSeleccionOfertaReservar, i, &comando, nil)
		if err != nil {
			return vacio, err
		}
		estado, err = s.registro.Reservar(operacion, i, comando, aut)
		if err != nil {
			return vacio, normalizarFalloSeleccionOferta(operacion, err)
		}
		if estado.Situacion == ports.SeleccionOfertaConfirmada {
			if estado.ReservaRef != "" || estado.Comando.ValidarPara(i) != nil ||
				!comandosOfertaEquivalentes(estado.Comando, comando) ||
				estado.Recibo.ValidarPara(i, estado.Comando) != nil {
				return vacio, ErrResultadoSeleccionOfertaNoConfiable
			}
			return estado.Recibo, nil
		}
		if estado.Situacion != ports.SeleccionOfertaReservada ||
			!comandosOfertaEquivalentes(estado.Comando, comando) {
			return vacio, errorSituacionSeleccionOferta(estado.Situacion)
		}
		fallthrough
	case ports.SeleccionOfertaReservada, ports.SeleccionOfertaIndeterminada:
		if !domain.ReferenciaOpacaValida(estado.ReservaRef) ||
			estado.Comando.ValidarPara(i) != nil || estado.Recibo != vacio {
			return vacio, ErrResultadoSeleccionOfertaNoConfiable
		}
		if estado.Situacion == ports.SeleccionOfertaIndeterminada && !estado.VentanaAbierta {
			return vacio, ErrResultadoSeleccionOfertaNoConfiable
		}
		reserva = ports.ReservaSeleccionOfertaCT{Intencion: i,
			ReservaRef: estado.ReservaRef, Comando: estado.Comando}
	case ports.SeleccionOfertaOcupada, ports.SeleccionOfertaColision:
		return vacio, errorSituacionSeleccionOferta(estado.Situacion)
	default:
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	if !estado.VentanaAbierta {
		aut, err := s.autorizar(operacion, ports.EtapaSeleccionOfertaAbrirVentana, i, &reserva.Comando, nil)
		if err != nil {
			return vacio, err
		}
		if err := s.registro.AbrirVentana(operacion, reserva, aut); err != nil {
			return vacio, normalizarFalloSeleccionOferta(operacion, err)
		}
	}
	// Desde esta ventana cualquier fallo externo es incierto. Solo un POST
	// nuevo con la misma clave recupera la orden; no hay repetición silenciosa.
	oferta, err := s.publicador.PublicarORecuperar(operacion, i, reserva.Comando)
	if err != nil {
		return vacio, errorIndeterminadoSeleccionOferta(operacion, err)
	}
	if oferta.ValidarPara(reserva.Comando) != nil {
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	aut, err := s.autorizar(operacion, ports.EtapaSeleccionOfertaConfirmar, i, &reserva.Comando, &oferta)
	if err != nil {
		// La oferta ya puede estar publicada. El siguiente POST con la misma
		// clave recupera el comando y la oferta antes de volver a confirmar.
		return vacio, errorIndeterminadoSeleccionOferta(operacion, err)
	}
	recibo, err := s.registro.ConfirmarVinculo(operacion, reserva, oferta, aut)
	if err != nil {
		return vacio, errorIndeterminadoSeleccionOferta(operacion, err)
	}
	if recibo.ValidarPara(i, reserva.Comando) != nil ||
		recibo.OfertaRef != oferta.OfertaRef || recibo.ReciboOfertaRef != oferta.ReciboOfertaRef ||
		!recibo.PublicadaEn.Equal(oferta.PublicadaEn) || !recibo.VenceAntesDe.Equal(oferta.VenceAntesDe) {
		return vacio, ErrResultadoSeleccionOfertaNoConfiable
	}
	return recibo, nil
}

func (s *ServicioSeleccionOfertaCT) autorizar(ctx context.Context, etapa string,
	i ports.IntencionSeleccionOfertaCT, comando *ports.ComandoOfertaBolsaCT,
	oferta *ports.OfertaPublicadaSeleccionCT,
) (ports.AutorizacionSeleccionOfertaCT, error) {
	solicitud := ports.SolicitudAutorizacionSeleccionOfertaCT{
		Etapa: etapa, Intencion: i, Comando: comando, Oferta: oferta,
	}
	canon, err := ports.CanonEtapaSeleccionOfertaCT(solicitud)
	if err != nil {
		return ports.AutorizacionSeleccionOfertaCT{}, ErrResultadoSeleccionOfertaNoConfiable
	}
	a, err := s.autorizador.AutorizarSeleccionOfertaCT(ctx, solicitud)
	if err != nil {
		return ports.AutorizacionSeleccionOfertaCT{}, normalizarFalloSeleccionOferta(ctx, err)
	}
	if a.ActorRef != i.ActorRef || a.PerfilRef != i.PerfilRef || a.MaterialCanonico != canon {
		return ports.AutorizacionSeleccionOfertaCT{}, ErrResultadoSeleccionOfertaNoConfiable
	}
	return a, nil
}

func comandosOfertaEquivalentes(a, b ports.ComandoOfertaBolsaCT) bool {
	if a.OrganizacionRef != b.OrganizacionRef || a.ExpedienteRef != b.ExpedienteRef ||
		a.VersionExpediente != b.VersionExpediente || a.ActorRef != b.ActorRef ||
		a.BolsaRef != b.BolsaRef || a.NecesidadRef != b.NecesidadRef ||
		a.NumeroPlazas != b.NumeroPlazas || a.NumeroElegibles != b.NumeroElegibles ||
		a.Datos != b.Datos || a.ClaveBolsa != b.ClaveBolsa || a.InicioPlazo != b.InicioPlazo {
		return false
	}
	var origenA, origenB any
	decoderA, decoderB := json.NewDecoder(bytes.NewReader(a.OrigenCT)), json.NewDecoder(bytes.NewReader(b.OrigenCT))
	decoderA.UseNumber()
	decoderB.UseNumber()
	return decoderA.Decode(&origenA) == nil && decoderB.Decode(&origenB) == nil && reflect.DeepEqual(origenA, origenB)
}

func errorSituacionSeleccionOferta(s ports.SituacionSeleccionOfertaCT) error {
	switch s {
	case ports.SeleccionOfertaOcupada:
		return ErrSeleccionOfertaConcurrente
	case ports.SeleccionOfertaColision:
		return ErrClaveSeleccionOfertaEnColision
	case ports.SeleccionOfertaIndeterminada:
		return ErrSeleccionOfertaIndeterminada
	default:
		return ErrResultadoSeleccionOfertaNoConfiable
	}
}

func normalizarFalloSeleccionOferta(ctx context.Context, err error) error {
	if errors.Is(err, ports.ErrAutorizacionDenegada) {
		return ports.ErrAutorizacionDenegada
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return ErrSeleccionOfertaNoDisponible
}

func errorIndeterminadoSeleccionOferta(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return errors.Join(ErrSeleccionOfertaIndeterminada, err, ctx.Err())
	}
	return errors.Join(ErrSeleccionOfertaIndeterminada, err)
}

// DatosReciboSeleccionOfertaParaAdaptador es el resultado mínimo del POST de
// selección. La ficha no confunde OfertaRef con LlamamientoRef singular.
type DatosReciboSeleccionOfertaParaAdaptador struct {
	Esquema          string    `json:"esquema"`
	Estado           string    `json:"estado"`
	OrganizacionRef  string    `json:"organizacion_ref"`
	ExpedienteRef    string    `json:"expediente_ref"`
	BolsaRef         string    `json:"bolsa_ref"`
	OfertaRef        string    `json:"oferta_ref"`
	ReciboOfertaRef  string    `json:"recibo_oferta_ref"`
	ReciboVinculoRef string    `json:"recibo_vinculo_ref"`
	NumeroPlazas     int       `json:"numero_plazas"`
	PublicadaEn      time.Time `json:"publicada_en"`
	VenceAntesDe     time.Time `json:"vence_antes_de"`
}

func (s *ServicioSeleccionOfertaCT) SeleccionarOfertaParaAdaptador(
	ctx context.Context, q SolicitudSeleccionLlamamiento,
) (DatosReciboSeleccionOfertaParaAdaptador, error) {
	r, err := s.SeleccionarOferta(ctx, q)
	if err != nil {
		return DatosReciboSeleccionOfertaParaAdaptador{}, err
	}
	return DatosReciboSeleccionOfertaParaAdaptador{
		Esquema: r.Esquema, Estado: r.Estado, OrganizacionRef: r.OrganizacionRef,
		ExpedienteRef: r.ExpedienteRef, BolsaRef: r.BolsaRef, OfertaRef: r.OfertaRef,
		ReciboOfertaRef: r.ReciboOfertaRef, ReciboVinculoRef: r.ReciboVinculoRef,
		NumeroPlazas: r.NumeroPlazas, PublicadaEn: r.PublicadaEn, VenceAntesDe: r.VenceAntesDe,
	}, nil
}
