package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorArtefactoAnalisisDoble struct {
	delegado  ports.PreparadorArtefactoAnalisisO3
	err       error
	llamadas  int
	solicitud ports.SolicitudPrepararArtefactoAnalisis
	forzado   *ports.ArtefactoAnalisisPreparado
}

func (d *preparadorArtefactoAnalisisDoble) PrepararArtefactoAnalisis(
	ctx context.Context,
	solicitud ports.SolicitudPrepararArtefactoAnalisis,
) (ports.ArtefactoAnalisisPreparado, error) {
	d.llamadas++
	d.solicitud = solicitud
	if d.err != nil {
		return ports.ArtefactoAnalisisPreparado{}, d.err
	}
	if d.forzado != nil {
		return *d.forzado, nil
	}
	if d.delegado == nil {
		return ports.ArtefactoAnalisisPreparado{},
			errors.New("preparador-real-sintetico-ausente")
	}
	return d.delegado.PrepararArtefactoAnalisis(
		ctx,
		solicitud,
	)
}

type selladorOperacionAnalisisDobleSaneado struct {
	err         error
	llamadas    int
	antes       func()
	preimagenes []ports.PreimagenesOperacionAnalisis
}

func (d *selladorOperacionAnalisisDobleSaneado) SellarOperacionAnalisis(
	_ context.Context,
	preimagenes ports.PreimagenesOperacionAnalisis,
) (ports.SellosOperacionAnalisis, error) {
	d.llamadas++
	d.preimagenes = append(d.preimagenes, preimagenes)
	if d.antes != nil {
		d.antes()
	}
	if d.err != nil {
		return ports.SellosOperacionAnalisis{}, d.err
	}
	ambito, errAmbito := preimagenes.BytesAmbito()
	semantica, errSemantica := preimagenes.BytesSemantica()
	if errAmbito != nil || errSemantica != nil {
		return ports.SellosOperacionAnalisis{}, errors.New(
			"fallo-sintetico-preimagen",
		)
	}
	sumaAmbito := sha256.Sum256(ambito)
	sumaSemantica := sha256.Sum256(semantica)
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal.analisis.ambito-idempotencia/v2:"+
			hex.EncodeToString(sumaAmbito[:]),
		nil,
	)
	if err != nil {
		return ports.SellosOperacionAnalisis{}, err
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(
		"hmac-sha256:vec.contratacion-temporal.analisis.huella-semantica/v2:"+
			hex.EncodeToString(sumaSemantica[:]),
		nil,
	)
	if err != nil {
		return ports.SellosOperacionAnalisis{}, err
	}
	return ports.SellosOperacionAnalisis{
		AmbitosIdempotenciaHMAC: ambitos,
		HuellasSemanticasHMAC:   huellas,
	}, nil
}

type preparadorOperacionAnalisisDobleSaneado struct {
	expediente         domain.Expediente
	confirmado         *ports.ReciboOperacionAnalisis
	consultaConfirmada *ports.ReciboOperacionAnalisis
	errConsulta        error
	consultas          int
	solicitudConsulta  ports.SolicitudConsultarOperacionAnalisisConfirmada
	err                error
	llamadas           int
	solicitud          ports.SolicitudPrepararOperacionAnalisis
}

func (d *preparadorOperacionAnalisisDobleSaneado) ConsultarOperacionAnalisisConfirmada(
	_ context.Context,
	solicitud ports.SolicitudConsultarOperacionAnalisisConfirmada,
) (ports.ReciboOperacionAnalisis, bool, error) {
	d.consultas++
	d.solicitudConsulta = solicitud
	if d.errConsulta != nil {
		return ports.ReciboOperacionAnalisis{}, false, d.errConsulta
	}
	if d.consultaConfirmada == nil {
		return ports.ReciboOperacionAnalisis{}, false, nil
	}
	return *d.consultaConfirmada, true, nil
}

func (d *preparadorOperacionAnalisisDobleSaneado) PrepararOperacionAnalisis(
	_ context.Context,
	solicitud ports.SolicitudPrepararOperacionAnalisis,
) (ports.PreparacionOperacionAnalisis, error) {
	d.llamadas++
	d.solicitud = solicitud
	if d.err != nil {
		return ports.PreparacionOperacionAnalisis{}, d.err
	}
	ambito, semantica, err := solicitud.Sellos.ParActivo()
	if err != nil {
		return ports.PreparacionOperacionAnalisis{}, err
	}
	datos := ports.DatosPreparacionOperacionAnalisis{
		ReservaRef:             "reserva:operacion-sintetica-001",
		ReciboRef:              "recibo:operacion-sintetica-001",
		Operacion:              solicitud.Operacion,
		OrganizacionRef:        solicitud.OrganizacionRef,
		ExpedienteRef:          solicitud.ExpedienteRef,
		VersionExpediente:      solicitud.VersionExpediente,
		ActorRef:               solicitud.ActorRef,
		PerfilRef:              solicitud.PerfilRef,
		ArtefactoRef:           solicitud.ArtefactoRef,
		ArtefactoHuellaSHA256:  solicitud.ArtefactoHuellaSHA256,
		AmbitoIdempotenciaHMAC: ambito,
		HuellaSemanticaHMAC:    semantica,
		Estado:                 ports.PreparacionOperacionAnalisisReservada,
		ExpedienteAnterior:     &d.expediente,
	}
	if d.confirmado != nil {
		datos.Estado = ports.PreparacionOperacionAnalisisConfirmada
		datos.ExpedienteAnterior = nil
		datos.ReciboConfirmado = d.confirmado
	}
	return ports.NuevaPreparacionOperacionAnalisis(solicitud, datos)
}

type resolutorPoliticaOperacionAnalisisDobleSaneado struct {
	motivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	exigeActorDistinto bool
	err                error
	llamadas           int
	solicitud          ports.SolicitudResolverPoliticaOperacionAnalisis
	transformar        func(*ports.PoliticaOperacionAnalisis)
}

func (d *resolutorPoliticaOperacionAnalisisDobleSaneado) ResolverPoliticaOperacionAnalisis(
	_ context.Context,
	solicitud ports.SolicitudResolverPoliticaOperacionAnalisis,
) (ports.PoliticaOperacionAnalisis, error) {
	d.llamadas++
	d.solicitud = solicitud
	if d.err != nil {
		return ports.PoliticaOperacionAnalisis{}, d.err
	}
	politica := ports.PoliticaOperacionAnalisis{
		Operacion:                solicitud.Operacion,
		OrganizacionRef:          solicitud.OrganizacionRef,
		ExpedienteRef:            solicitud.ExpedienteRef,
		VersionExpediente:        solicitud.VersionExpediente,
		FasePrevia:               solicitud.FasePrevia,
		EstadoPrevio:             solicitud.EstadoPrevio,
		ActorRef:                 solicitud.ActorRef,
		ActorAnalisisAnteriorRef: solicitud.ActorAnalisisAnteriorRef,
		ArtefactoRef:             solicitud.ArtefactoRef,
		ArtefactoHuellaSHA256:    solicitud.ArtefactoHuellaSHA256,
		DefinicionRef:            "politica:analisis-sintetica-001",
		Version:                  3,
		HuellaSHA256:             strings.Repeat("3", 64),
		Finalidad:                "analisis.tramitar",
		UnidadRef:                "unidad:rrhh-sintetica-001",
		MotivoAutorizacion:       d.motivoAutorizacion,
		ExigeActorDistinto:       d.exigeActorDistinto,
		EvaluadaEn:               solicitud.Instante,
	}
	if solicitud.Operacion == ports.OperacionRegistrarAnalisis {
		politica.Accion = domain.ClaveCatalogo(
			ports.AccionRegistrarAnalisis,
		)
	} else {
		politica.Accion = domain.ClaveCatalogo(
			ports.AccionRectificarAnalisis,
		)
		politica.MotivoRectificacion =
			ports.MotivoRectificacionGobernado{
				ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
					CatalogoID:           "motivos_rectificacion_analisis",
					CatalogoVersion:      2,
					CatalogoHuellaSHA256: strings.Repeat("4", 64),
					EntradaClave: string(
						solicitud.MotivoRectificacionClave,
					),
				},
				ClaveMensajeI18N: solicitud.MotivoRectificacionClave,
			}
	}
	if d.transformar != nil {
		d.transformar(&politica)
	}
	return politica, nil
}

type transaccionOperacionAnalisisDobleSaneado struct {
	err                 error
	llamadas            int
	orden               ports.OrdenConfirmarOperacionAnalisis
	despues             func()
	desfaseConfirmacion time.Duration
	consumosFuentes     int
	consumosV3          int
	commits             int
	adulterarSalida     bool
	confirmado          *ports.ReciboOperacionAnalisis
}

func (d *transaccionOperacionAnalisisDobleSaneado) ConfirmarOperacionAnalisis(
	_ context.Context,
	orden ports.OrdenConfirmarOperacionAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	d.llamadas++
	d.orden = orden
	if d.err != nil {
		return ports.ReciboOperacionAnalisis{}, d.err
	}
	evidencia, err := orden.Datos()
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	confirmadaEn := evidencia.InstanteEfecto.Add(d.desfaseConfirmacion)
	if err := orden.ValidarConfirmacionDentroDeTransaccion(
		confirmadaEn,
	); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	preparacion, err := evidencia.Preparacion.DatosPara(
		evidencia.SolicitudPreparacion,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	ordenConsumo, err := evidencia.OrdenConsumoFuentes.Datos()
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	reciboRC, err := ports.NuevoReciboConsumoRespuestaFuenteAnalisis(
		ordenConsumo.OrdenRC,
		"consumo_validacion_rc_sintetico_012345",
		confirmadaEn,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	var reciboCoste *ports.ReciboConsumoRespuestaFuenteAnalisis
	if ordenConsumo.OrdenCoste != nil {
		coste, errCoste := ports.NuevoReciboConsumoRespuestaFuenteAnalisis(
			*ordenConsumo.OrdenCoste,
			"consumo_calculo_coste_sintetico_012345",
			confirmadaEn,
		)
		if errCoste != nil {
			return ports.ReciboOperacionAnalisis{}, errCoste
		}
		reciboCoste = &coste
	}
	reciboConsumo, err := ports.NuevoReciboConsumoConjuntoFuentesAnalisisO3(
		evidencia.OrdenConsumoFuentes,
		"consumo_conjunto_sintetico_012345",
		reciboRC,
		reciboCoste,
		confirmadaEn,
	)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	confirmacionV3, err := evidencia.ConfirmacionV3.Datos()
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	recibo := ports.ReciboOperacionAnalisis{
		Operacion:              preparacion.Operacion,
		OrganizacionRef:        preparacion.OrganizacionRef,
		ExpedienteRef:          preparacion.ExpedienteRef,
		VersionAnterior:        preparacion.VersionExpediente,
		VersionResultante:      evidencia.ExpedienteSiguiente.Version,
		SecuenciaActuacion:     evidencia.ExpedienteSiguiente.Version,
		ArtefactoRef:           preparacion.ArtefactoRef,
		ArtefactoHuellaSHA256:  preparacion.ArtefactoHuellaSHA256,
		ReciboRef:              preparacion.ReciboRef,
		AuditoriaRef:           "auditoria:operacion-sintetica-001",
		EventoRef:              "evento:operacion-sintetica-001",
		ConsumoFuentesRef:      reciboConsumo.ConsumoConjuntoRef,
		HuellaConsumoFuentes:   reciboConsumo.HuellaConjunto,
		ConcesionV3DecisionRef: confirmacionV3.DecisionRef,
		HuellaSemanticaHMAC:    preparacion.HuellaSemanticaHMAC,
		ConfirmadaEn:           confirmadaEn,
	}
	if err := recibo.ValidarParaOrdenDentroDeTransaccion(orden); err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	d.consumosFuentes++
	d.consumosV3++
	d.commits++
	confirmado := recibo
	d.confirmado = &confirmado
	if d.despues != nil {
		d.despues()
	}
	if d.adulterarSalida {
		recibo.EventoRef = ""
	}
	return recibo, nil
}

type escenarioOperacionAnalisisSaneado struct {
	instante            time.Time
	contexto            ports.ContextoAutorizacionAltaV3
	expediente          domain.Expediente
	funcionales         ports.DatosFuncionalesOperacionAnalisis
	motivoAutorizacion  dominiovec.ReferenciaEntradaCatalogo
	motivoRectificacion domain.ClaveCatalogo
	registrar           SolicitudRegistrarAnalisis
	rectificar          SolicitudRectificarAnalisis
}

type dependenciasOperacionAnalisisSaneado struct {
	contextos     *resolutorContextoDoble
	artefactos    *preparadorArtefactoAnalisisDoble
	sellador      *selladorOperacionAnalisisDobleSaneado
	preparaciones *preparadorOperacionAnalisisDobleSaneado
	politicas     *resolutorPoliticaOperacionAnalisisDobleSaneado
	correlaciones *generadorReferenciasDoble
	autorizador   *autorizadorV3Doble
	reloj         *relojMutable
	transaccion   *transaccionOperacionAnalisisDobleSaneado
}

func construirServicioOperacionAnalisisSaneado(
	t *testing.T,
	escenario escenarioOperacionAnalisisSaneado,
) (*ServicioOperacionAnalisis, *dependenciasOperacionAnalisisSaneado) {
	t.Helper()
	d := &dependenciasOperacionAnalisisSaneado{
		contextos: &resolutorContextoDoble{contexto: escenario.contexto},
		artefactos: &preparadorArtefactoAnalisisDoble{
			delegado: nuevoPreparadorArtefactoAnalisisO3AplicacionPrueba(
				t,
				escenario.instante,
			),
		},
		sellador: &selladorOperacionAnalisisDobleSaneado{},
		preparaciones: &preparadorOperacionAnalisisDobleSaneado{
			expediente: escenario.expediente,
		},
		politicas: &resolutorPoliticaOperacionAnalisisDobleSaneado{
			motivoAutorizacion: escenario.motivoAutorizacion,
			exigeActorDistinto: escenario.expediente.Analisis != nil,
		},
		correlaciones: &generadorReferenciasDoble{
			correlacion: "correlacion_44444444444444444444444444444444",
		},
		autorizador: &autorizadorV3Doble{
			t:        t,
			instante: escenario.instante,
			motivo:   escenario.motivoAutorizacion,
		},
		reloj:       &relojMutable{instante: escenario.instante},
		transaccion: &transaccionOperacionAnalisisDobleSaneado{},
	}
	servicio, err := NuevoServicioOperacionAnalisis(
		d.contextos,
		d.artefactos,
		d.sellador,
		d.preparaciones,
		d.politicas,
		d.correlaciones,
		d.autorizador,
		d.reloj,
		d.transaccion,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio, d
}
