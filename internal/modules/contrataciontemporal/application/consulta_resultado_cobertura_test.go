package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestConsultaResultadoConfirmadoSinCatalogoNiEfectos(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	recibo, err := escenario.servicio.Decidir(
		context.Background(),
		escenario.solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	evidencia := evidenciaHistoricaConfirmacionPrueba(
		t,
		escenario.idempotencia,
		recibo,
	)
	accesos := &autorizadorLecturaResultadoCoberturaPrueba{
		organizacionAdmitida: escenario.solicitud.OrganizacionRef,
	}
	lector := &lectorResultadoHistoricoCoberturaPrueba{
		evidencia: &evidencia,
	}
	servicio := nuevoServicioConsultaResultadoPrueba(
		t,
		contextoRecuperacionDesdeDecisionPrueba(escenario),
		accesos,
		&selladorAmbitoConsultaCoberturaPrueba{},
		escenario.base.reloj,
		lector,
	)

	resultado, err := servicio.Consultar(
		context.Background(),
		solicitudConsultaDesdeDecisionPrueba(escenario),
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, valida := resultado.DatosParaAdaptador()
	if !valida || datos.Estado != ResultadoCoberturaConfirmado ||
		datos.Recibo == nil ||
		datos.Recibo.ReciboRef != recibo.ReciboRef ||
		datos.Recibo.Estado != "aplicada" {
		t.Fatalf("resultado confirmado inesperado: %#v, %t", datos, valida)
	}
	if accesos.total() != 2 {
		t.Fatalf("autorización no revalidada: %d", accesos.total())
	}
	comprobarAutorizacionConsultaResultado(
		t,
		accesos,
		escenario.solicitud.OrganizacionRef,
		escenario.solicitud.ExpedienteRef,
	)
	if consultas, efectos := lector.totales(); consultas != 1 || efectos != 0 {
		t.Fatalf("lectura produjo efectos: %d/%d", consultas, efectos)
	}
	tipoEjecutor := reflect.TypeOf(
		(*cobertura.EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB)(nil),
	).Elem()
	if tipoEjecutor.NumMethod() != 1 ||
		tipoEjecutor.Method(0).Name !=
			"EjecutarLecturaResultadoHistoricoTCB" {
		t.Fatalf("ejecutor TCB expone más superficie: %v", tipoEjecutor)
	}
	tipoSesion := reflect.TypeOf(
		(*cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB)(nil),
	).Elem()
	if tipoSesion.NumMethod() != 1 ||
		tipoSesion.Method(0).Name != "LeerResultadoHistoricoTCB" {
		t.Fatalf("sesión TCB expone más superficie: %v", tipoSesion)
	}
}

func TestConsultaResultadoRectificacionSobreviveRetiradaCatalogo(t *testing.T) {
	escenario := nuevoEscenarioRectificacionConfirmacionCobertura(t, true)
	recibo, err := escenario.servicio.Rectificar(
		context.Background(),
		escenario.solicitudRectificar,
	)
	if err != nil {
		t.Fatal(err)
	}
	evidencia := evidenciaHistoricaConfirmacionPrueba(
		t,
		escenario.idempotencia,
		recibo,
	)
	contextos := &resolutorContextoRecuperacionCoberturaPrueba{
		origen: escenario.base.contextos,
		solicitud: ports.SolicitudResolverContextoAutorizacionAltaV3{
			AutenticacionRef: escenario.solicitudRectificar.AutenticacionRef,
			SesionRef:        escenario.solicitudRectificar.SesionRef,
			PerfilRef:        escenario.solicitudRectificar.PerfilRef,
		},
		organizacion: escenario.solicitudRectificar.OrganizacionRef,
		reloj:        escenario.base.reloj,
	}
	servicio := nuevoServicioConsultaResultadoPrueba(
		t,
		contextos,
		&autorizadorLecturaResultadoCoberturaPrueba{},
		&selladorAmbitoConsultaCoberturaPrueba{},
		escenario.base.reloj,
		&lectorResultadoHistoricoCoberturaPrueba{evidencia: &evidencia},
	)
	resultado, err := servicio.Consultar(
		context.Background(),
		SolicitudConsultaResultadoCobertura{
			ClaveIdempotencia: escenario.solicitudRectificar.
				ClaveIdempotencia,
			ExpedienteRef: escenario.solicitudRectificar.ExpedienteRef,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, valida := resultado.DatosParaAdaptador()
	if !valida || datos.Estado != ResultadoCoberturaConfirmado ||
		datos.Recibo == nil || datos.Recibo.Estado != "denegada" {
		t.Fatalf("resultado histórico tras retirada: %#v/%t", datos, valida)
	}
}

func TestConsultaResultadoNoObservableNoHabilitaReintento(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	lector := &lectorResultadoHistoricoCoberturaPrueba{
		noObservable: true,
	}
	servicio := nuevoServicioConsultaResultadoPrueba(
		t,
		contextoRecuperacionDesdeDecisionPrueba(escenario),
		&autorizadorLecturaResultadoCoberturaPrueba{},
		&selladorAmbitoConsultaCoberturaPrueba{},
		escenario.base.reloj,
		lector,
	)
	resultado, err := servicio.Consultar(
		context.Background(),
		solicitudConsultaDesdeDecisionPrueba(escenario),
	)
	if err != nil {
		t.Fatal(err)
	}
	datos, valida := resultado.DatosParaAdaptador()
	if !valida || datos.Estado != ResultadoCoberturaNoObservable ||
		datos.Recibo != nil {
		t.Fatalf("unión no observable inválida: %#v/%t", datos, valida)
	}
	if consultas, efectos := lector.totales(); consultas != 1 || efectos != 0 {
		t.Fatalf("no observable produjo efecto: %d/%d", consultas, efectos)
	}
}

func TestConsultaResultadoDenegadaOContextoCruzadoNoLee(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	casos := map[string]struct {
		contextos *resolutorContextoRecuperacionCoberturaPrueba
		accesos   *autorizadorLecturaResultadoCoberturaPrueba
	}{
		"denegada": {
			contextos: contextoRecuperacionDesdeDecisionPrueba(escenario),
			accesos: &autorizadorLecturaResultadoCoberturaPrueba{
				resultado: ports.AutorizacionLecturaResultadoCoberturaDenegada,
			},
		},
		"organizacion_cruzada": {
			contextos: func() *resolutorContextoRecuperacionCoberturaPrueba {
				contextos := contextoRecuperacionDesdeDecisionPrueba(escenario)
				contextos.organizacion = "organizacion_cruzada_resultado_01"
				return contextos
			}(),
			accesos: &autorizadorLecturaResultadoCoberturaPrueba{
				organizacionAdmitida: escenario.solicitud.OrganizacionRef,
			},
		},
		"contexto_denegado": {
			contextos: func() *resolutorContextoRecuperacionCoberturaPrueba {
				contextos := contextoRecuperacionDesdeDecisionPrueba(escenario)
				contextos.err = ports.ErrAutorizacionDenegada
				return contextos
			}(),
			accesos: &autorizadorLecturaResultadoCoberturaPrueba{},
		},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			lector := &lectorResultadoHistoricoCoberturaPrueba{
				noObservable: true,
			}
			servicio := nuevoServicioConsultaResultadoPrueba(
				t,
				caso.contextos,
				caso.accesos,
				&selladorAmbitoConsultaCoberturaPrueba{},
				escenario.base.reloj,
				lector,
			)
			_, err := servicio.Consultar(
				context.Background(),
				solicitudConsultaDesdeDecisionPrueba(escenario),
			)
			if !errors.Is(err, ErrConsultaResultadoCoberturaDenegada) {
				t.Fatalf("denegación no cerrada: %v", err)
			}
			if consultas, efectos := lector.totales(); consultas != 0 ||
				efectos != 0 {
				t.Fatalf("denegación alcanzó lector: %d/%d", consultas, efectos)
			}
		})
	}
}

func TestConsultaResultadoContradictorioFallaCerrado(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	lector := &lectorResultadoHistoricoCoberturaPrueba{
		contradictorio: true,
	}
	servicio := nuevoServicioConsultaResultadoPrueba(
		t,
		contextoRecuperacionDesdeDecisionPrueba(escenario),
		&autorizadorLecturaResultadoCoberturaPrueba{},
		&selladorAmbitoConsultaCoberturaPrueba{},
		escenario.base.reloj,
		lector,
	)
	resultado, err := servicio.Consultar(
		context.Background(),
		solicitudConsultaDesdeDecisionPrueba(escenario),
	)
	if !errors.Is(err, ErrConsultaResultadoCoberturaNoConfiable) {
		t.Fatalf("contradicción aceptada: %v", err)
	}
	if _, valida := resultado.DatosParaAdaptador(); valida {
		t.Fatal("contradicción publicó unión válida")
	}
}

func TestConsultaResultadoRespetaCancelacionSinEfectos(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	t.Run("antes", func(t *testing.T) {
		lector := &lectorResultadoHistoricoCoberturaPrueba{
			noObservable: true,
		}
		servicio := nuevoServicioConsultaResultadoPrueba(
			t,
			contextoRecuperacionDesdeDecisionPrueba(escenario),
			&autorizadorLecturaResultadoCoberturaPrueba{},
			&selladorAmbitoConsultaCoberturaPrueba{},
			escenario.base.reloj,
			lector,
		)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		_, err := servicio.Consultar(
			ctx,
			solicitudConsultaDesdeDecisionPrueba(escenario),
		)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelación no preservada: %v", err)
		}
		if consultas, efectos := lector.totales(); consultas != 0 ||
			efectos != 0 {
			t.Fatalf("cancelación produjo actividad: %d/%d", consultas, efectos)
		}
	})

	t.Run("durante_lectura", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		lector := &lectorResultadoHistoricoCoberturaPrueba{
			noObservable: true,
			cancelar:     cancelar,
		}
		servicio := nuevoServicioConsultaResultadoPrueba(
			t,
			contextoRecuperacionDesdeDecisionPrueba(escenario),
			&autorizadorLecturaResultadoCoberturaPrueba{},
			&selladorAmbitoConsultaCoberturaPrueba{},
			escenario.base.reloj,
			lector,
		)
		resultado, err := servicio.Consultar(
			ctx,
			solicitudConsultaDesdeDecisionPrueba(escenario),
		)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelación se volvió no observable: %v", err)
		}
		if _, valida := resultado.DatosParaAdaptador(); valida {
			t.Fatal("cancelación publicó unión válida")
		}
	})
}

func TestConsultaResultadoEntradaYSalidaRedactadas(t *testing.T) {
	escenario := nuevoEscenarioConfirmacionCobertura(t, true)
	solicitud := solicitudConsultaDesdeDecisionPrueba(escenario)
	servicio := nuevoServicioConsultaResultadoPrueba(
		t,
		contextoRecuperacionDesdeDecisionPrueba(escenario),
		&autorizadorLecturaResultadoCoberturaPrueba{},
		&selladorAmbitoConsultaCoberturaPrueba{},
		escenario.base.reloj,
		&lectorResultadoHistoricoCoberturaPrueba{noObservable: true},
	)
	resultado, err := servicio.Consultar(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, valor := range map[string]string{
		"solicitud": fmt.Sprintf("%+v", solicitud),
		"resultado": fmt.Sprintf("%+v", resultado),
	} {
		if strings.Contains(valor, solicitud.ExpedienteRef) ||
			strings.Contains(valor, solicitud.ClaveIdempotencia) ||
			!strings.Contains(valor, "REDACTAD") {
			t.Fatalf("%s filtró datos: %q", nombre, valor)
		}
	}
	if tipo := reflect.TypeOf(solicitud); tipo.NumField() != 2 {
		t.Fatalf("DTO cliente dejó de ser mínimo: %v", tipo)
	}
}

var _ cobertura.EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB = (*lectorResultadoHistoricoCoberturaPrueba)(nil)
var _ cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB = (*lectorResultadoHistoricoCoberturaPrueba)(nil)
