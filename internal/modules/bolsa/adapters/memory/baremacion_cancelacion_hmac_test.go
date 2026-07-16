package memory

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type verificadorCancelacionBaremacionPrueba struct {
	delegado puertosbolsa.VerificadorSellosBaremacion

	mu          sync.Mutex
	finalidad   puertosbolsa.FinalidadSelloBaremacion
	objetivo    int
	llamadas    int
	cancelar    context.CancelFunc
	esperar     bool
	devolverNil bool
}

type selladorHMACMemoriaPrueba struct{}

func (selladorHMACMemoriaPrueba) SellarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudSellarSelloBaremacion,
) (string, error) {
	if ctx == nil || solicitud.Validar() != nil {
		return "", puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return calcularSelloMemoria(solicitud.Finalidad, solicitud.RepresentacionCanonica.Revelar()), nil
}

type verificadorPausaHistoricaBaremacionPrueba struct {
	delegado puertosbolsa.VerificadorSellosBaremacion

	mu        sync.Mutex
	objetivo  int
	llamadas  int
	alcanzada chan struct{}
	liberar   <-chan struct{}
}

func (v *verificadorPausaHistoricaBaremacionPrueba) VerificarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	pausar := false
	if solicitud.Finalidad == puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3 {
		v.mu.Lock()
		v.llamadas++
		pausar = v.objetivo > 0 && v.llamadas == v.objetivo
		v.mu.Unlock()
	}
	if pausar {
		close(v.alcanzada)
		select {
		case <-v.liberar:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return v.delegado.VerificarSelloBaremacion(ctx, solicitud)
}

func (v *verificadorCancelacionBaremacionPrueba) preparar(
	finalidad puertosbolsa.FinalidadSelloBaremacion,
	objetivo int,
	cancelar context.CancelFunc,
	esperar, devolverNil bool,
) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.finalidad = finalidad
	v.objetivo = objetivo
	v.llamadas = 0
	v.cancelar = cancelar
	v.esperar = esperar
	v.devolverNil = devolverNil
}

func (v *verificadorCancelacionBaremacionPrueba) numeroLlamadas() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.llamadas
}

func (v *verificadorCancelacionBaremacionPrueba) VerificarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	v.mu.Lock()
	disparar := false
	var cancelar context.CancelFunc
	var esperar, devolverNil bool
	if solicitud.Finalidad == v.finalidad {
		v.llamadas++
		disparar = v.objetivo > 0 && v.llamadas == v.objetivo
		if disparar {
			cancelar, esperar, devolverNil = v.cancelar, v.esperar, v.devolverNil
		}
	}
	v.mu.Unlock()
	if disparar {
		if cancelar != nil {
			cancelar()
		}
		if esperar {
			<-ctx.Done()
		}
		if devolverNil {
			return nil
		}
		return ctx.Err()
	}
	return v.delegado.VerificarSelloBaremacion(ctx, solicitud)
}

func nuevoRepositorioCancelacionBaremacionPrueba(
	t *testing.T,
	verificador puertosbolsa.VerificadorSellosBaremacion,
) *RepositorioBaremaciones {
	t.Helper()
	repositorio, err := NuevoRepositorioBaremaciones(
		&relojMemoriaPrueba{instante: instanteMemoriaPrueba},
		verificador,
		PerfilRepositorioBaremacionesSoloPruebas(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return repositorio
}

func comprobarReservaCanceladaSinEfectosPrueba(t *testing.T, repositorio *RepositorioBaremaciones) {
	t.Helper()
	repositorio.mu.RLock()
	defer repositorio.mu.RUnlock()
	if len(repositorio.reservasPorAmbito) != 0 || len(repositorio.ambitoPorHuellaToken) != 0 ||
		len(repositorio.ambitoActivoPorBaremacion) != 0 || len(repositorio.usosAutorizacion) != 0 ||
		len(repositorio.auditorias) != 0 || len(repositorio.eventosOutbox) != 0 {
		t.Fatalf("la reserva cancelada produjo efectos: reservas=%d tokens=%d ambitos=%d usos=%d",
			len(repositorio.reservasPorAmbito), len(repositorio.ambitoPorHuellaToken),
			len(repositorio.ambitoActivoPorBaremacion), len(repositorio.usosAutorizacion))
	}
}

func TestRepositorioBaremacionesPreservaCancelacionYPlazoEnSelloReserva(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		plazo    bool
		esperado error
	}{
		{nombre: "cancelacion", esperado: context.Canceled},
		{nombre: "plazo", plazo: true, esperado: context.DeadlineExceeded},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			verificador := &verificadorCancelacionBaremacionPrueba{delegado: verificadorHMACMemoriaPrueba{}}
			repositorio := nuevoRepositorioCancelacionBaremacionPrueba(t, verificador)
			var ctx context.Context
			var cancelar context.CancelFunc
			if caso.plazo {
				ctx, cancelar = context.WithTimeout(context.Background(), 10*time.Millisecond)
				verificador.preparar(puertosbolsa.FinalidadSelloReservaBaremacion, 1, nil, true, false)
			} else {
				ctx, cancelar = context.WithCancel(context.Background())
				verificador.preparar(puertosbolsa.FinalidadSelloReservaBaremacion, 1, cancelar, false, false)
			}
			defer cancelar()
			if _, err := repositorio.ReservarCambio(ctx, solicitudReservaAltaMemoria()); !errors.Is(err, caso.esperado) {
				t.Fatalf("no se preservo el error del contexto: %v", err)
			}
			comprobarReservaCanceladaSinEfectosPrueba(t, repositorio)
			verificador.preparar(puertosbolsa.FinalidadSelloReservaBaremacion, 0, nil, false, false)
			if _, err := repositorio.ReservarCambio(context.Background(), solicitudReservaAltaMemoria()); err != nil {
				t.Fatalf("la cancelacion dejo bloqueo o efectos: %v", err)
			}
		})
	}
}

func TestRepositorioBaremacionesDetectaCancelacionAunqueConectorDevuelvaNil(t *testing.T) {
	for _, finalidad := range []puertosbolsa.FinalidadSelloBaremacion{
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3,
		puertosbolsa.FinalidadSelloConfirmacionBaremacionV2,
	} {
		t.Run(string(finalidad), func(t *testing.T) {
			verificador := &verificadorCancelacionBaremacionPrueba{delegado: verificadorHMACMemoriaPrueba{}}
			escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificador)
			ctx, cancelar := context.WithCancel(context.Background())
			verificador.preparar(finalidad, 1, cancelar, false, true)
			if _, err := escenario.repositorio.ConfirmarCambio(ctx, escenario.confirmar); !errors.Is(err, context.Canceled) {
				t.Fatalf("conector cancelo y devolvio nil sin propagacion: %v", err)
			}
			comprobarDecisionSinEfectosPrueba(t, escenario.repositorio)
			verificador.preparar(finalidad, 0, nil, false, false)
			if _, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar); err != nil {
				t.Fatalf("la cancelacion consumio la reserva: %v", err)
			}
		})
	}
}

func TestRepositorioBaremacionesDetectaCancelacionConHistoricoVacioAunqueConectorDevuelvaNil(t *testing.T) {
	verificador := &verificadorCancelacionBaremacionPrueba{delegado: verificadorHMACMemoriaPrueba{}}
	repositorio := nuevoRepositorioCancelacionBaremacionPrueba(t, verificador)
	reserva, err := repositorio.ReservarCambio(context.Background(), solicitudReservaAltaMemoria())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	verificador.preparar(puertosbolsa.FinalidadSelloConfirmacionBaremacionV2, 1, cancelar, false, true)
	if _, err := repositorio.ConfirmarCambio(
		ctx, solicitudConfirmarAltaMemoria(reserva.Token, nuevaBaremacionMemoriaPrueba(t)),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("confirmacion sin historico oculto la cancelacion: %v", err)
	}
	repositorio.mu.RLock()
	defer repositorio.mu.RUnlock()
	if len(repositorio.versionesPorBaremacion) != 0 || len(repositorio.auditorias) != 0 ||
		len(repositorio.eventosOutbox) != 0 {
		t.Fatal("confirmacion cancelada con historico vacio produjo efectos")
	}
}

func TestRepositorioBaremacionesPreservaCancelacionEnLecturasHistoricas(t *testing.T) {
	verificador := &verificadorCancelacionBaremacionPrueba{delegado: verificadorHMACMemoriaPrueba{}}
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificador)
	resultado, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, operacion := range operacionesLecturaManifiestoConContextoPrueba(escenario, resultado) {
		t.Run(nombre, func(t *testing.T) {
			ctx, cancelar := context.WithCancel(context.Background())
			verificador.preparar(
				puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, 1, cancelar, false, true,
			)
			if err := operacion(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("lectura no preservo cancelacion: %v", err)
			}
			verificador.preparar(
				puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, 0, nil, false, false,
			)
			if err := operacion(context.Background()); err != nil {
				t.Fatalf("lectura posterior quedo bloqueada: %v", err)
			}
		})
	}
}

func TestRepositorioBaremacionesReintentoExactoReverificaManifiestos(t *testing.T) {
	verificador := &verificadorCancelacionBaremacionPrueba{delegado: verificadorHMACMemoriaPrueba{}}
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificador)
	primero, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancelar := context.WithCancel(context.Background())
	verificador.preparar(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, 2, cancelar, false, true,
	)
	if _, err := escenario.repositorio.ConfirmarCambio(ctx, escenario.confirmar); !errors.Is(err, context.Canceled) {
		t.Fatalf("preflight historico no preservo cancelacion: %v", err)
	}
	if llamadas := verificador.numeroLlamadas(); llamadas != 2 {
		t.Fatalf("preflight no alcanzo el manifiesto historico: llamadas=%d", llamadas)
	}
	comprobarLongitudesInternas(t, escenario.repositorio, 2, 2, 2)

	verificador.preparar(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, 0, nil, false, false,
	)
	repetido, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatalf("reintento exacto rechazado: %v", err)
	}
	if !reflect.DeepEqual(primero, repetido) {
		t.Fatalf("reintento exacto no devolvio el mismo recibo: %+v / %+v", primero, repetido)
	}
	if llamadas := verificador.numeroLlamadas(); llamadas != 3 {
		t.Fatalf("reintento no reverifico entrada, preflight y snapshot final: llamadas=%d", llamadas)
	}
	comprobarLongitudesInternas(t, escenario.repositorio, 2, 2, 2)
}

func TestRepositorioBaremacionesReintentoReservaConfirmadaVerificaHistoricoFueraDeBloqueo(t *testing.T) {
	verificador := &verificadorCancelacionBaremacionPrueba{delegado: verificadorHMACMemoriaPrueba{}}
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificador)
	confirmada, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatal(err)
	}
	solicitud := solicitudReservaDecisionMemoria(escenario.alta.Version.Referencia)
	ctx, cancelar := context.WithCancel(context.Background())
	verificador.preparar(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, 1, cancelar, false, true,
	)
	respuesta, err := escenario.repositorio.ReservarCambio(ctx, solicitud)
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(respuesta, puertosbolsa.ReservaCambioBaremacion{}) {
		t.Fatalf("reintento revelo resultado sin verificar: respuesta=%+v err=%v", respuesta, err)
	}
	verificador.preparar(
		puertosbolsa.FinalidadSelloManifiestoProbatorioBaremacionV3, 0, nil, false, false,
	)
	respuesta, err = escenario.repositorio.ReservarCambio(context.Background(), solicitud)
	if err != nil || respuesta.VersionConfirmada == nil ||
		respuesta.VersionConfirmada.Referencia != confirmada.Version.Referencia {
		t.Fatalf("reintento valido no recupero version: respuesta=%+v err=%v", respuesta, err)
	}
	if llamadas := verificador.numeroLlamadas(); llamadas != 1 {
		t.Fatalf("reserva confirmada no reverifico todo su historico: llamadas=%d", llamadas)
	}
}

func TestRepositorioBaremacionesOCCMientrasVerificaHistoricoFueraDelBloqueo(t *testing.T) {
	alcanzada := make(chan struct{})
	liberar := make(chan struct{})
	var liberarUnaVez sync.Once
	desbloquear := func() { liberarUnaVez.Do(func() { close(liberar) }) }
	defer desbloquear()
	verificador := &verificadorPausaHistoricaBaremacionPrueba{
		delegado: verificadorHMACMemoriaPrueba{}, objetivo: 2, alcanzada: alcanzada, liberar: liberar,
	}
	escenario := nuevoEscenarioManifiestoHistoricoPrueba(t, verificador)
	versionDos, err := escenario.repositorio.ConfirmarCambio(context.Background(), escenario.confirmar)
	if err != nil {
		t.Fatal(err)
	}
	instanteTres := instanteMemoriaPrueba.Add(30 * time.Minute)
	agregadoTres := incorporarRectificacionHistoricaPrueba(t, versionDos.Version.Agregado, "003", instanteTres)
	ultimaTres, _ := agregadoTres.UltimaDecision()
	reservaTres := reservarDecisionHistoricaPrueba(
		t, escenario.repositorio, escenario.reloj, versionDos.Version.Referencia,
		ultimaTres.Contenido.CorrelacionRef, "v3", instanteTres,
	)
	confirmacion := solicitudConfirmarDecisionHistoricaPrueba(
		t, reservaTres.Token, versionDos.Version.Referencia, agregadoTres, "v3", instanteTres,
		selladorHMACMemoriaPrueba{},
	)
	verificador.mu.Lock()
	verificador.llamadas = 0
	verificador.mu.Unlock()

	type resultadoPrueba struct {
		resultado puertosbolsa.ResultadoConfirmarCambioBaremacion
		err       error
	}
	primero := make(chan resultadoPrueba, 1)
	segundo := make(chan resultadoPrueba, 1)
	go func() {
		resultado, err := escenario.repositorio.ConfirmarCambio(context.Background(), confirmacion)
		primero <- resultadoPrueba{resultado: resultado, err: err}
	}()
	select {
	case <-alcanzada:
	case <-time.After(5 * time.Second):
		t.Fatal("la primera confirmacion no alcanzo la verificacion historica pausada")
	}
	go func() {
		resultado, err := escenario.repositorio.ConfirmarCambio(context.Background(), confirmacion)
		segundo <- resultadoPrueba{resultado: resultado, err: err}
	}()
	var resultadoSegundo resultadoPrueba
	select {
	case resultadoSegundo = <-segundo:
	case <-time.After(5 * time.Second):
		t.Fatal("el verificador historico retenia indebidamente el mutex")
	}
	if resultadoSegundo.err != nil {
		t.Fatalf("segunda confirmacion: %v", resultadoSegundo.err)
	}
	desbloquear()
	var resultadoPrimero resultadoPrueba
	select {
	case resultadoPrimero = <-primero:
	case <-time.After(5 * time.Second):
		t.Fatal("la primera confirmacion no concluyo tras liberar el verificador")
	}
	if resultadoPrimero.err != nil || !reflect.DeepEqual(resultadoPrimero.resultado, resultadoSegundo.resultado) {
		t.Fatalf("OCC no devolvio el mismo recibo: primero=%+v segundo=%+v", resultadoPrimero, resultadoSegundo)
	}
	comprobarLongitudesInternas(t, escenario.repositorio, 3, 3, 3)
}

func operacionesLecturaManifiestoConContextoPrueba(
	escenario escenarioManifiestoHistoricoPrueba,
	resultado puertosbolsa.ResultadoConfirmarCambioBaremacion,
) map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"vigente": func(ctx context.Context) error {
			_, err := escenario.repositorio.ObtenerVersionVigente(ctx, puertosbolsa.SolicitudObtenerBaremacionVigente{
				Contexto: contextoMemoriaPrueba(
					puertosbolsa.AccionConsultarBaremacionVigente, escenario.base.ID, escenario.reloj.Ahora(),
				),
				BaremacionMeritoRef: escenario.base.ID,
			})
			return err
		},
		"version": func(ctx context.Context) error {
			_, err := escenario.repositorio.ObtenerVersion(ctx, puertosbolsa.SolicitudObtenerVersionBaremacion{
				Contexto: contextoMemoriaPrueba(
					puertosbolsa.AccionConsultarVersionBaremacion, escenario.base.ID, escenario.reloj.Ahora(),
				),
				BaremacionMeritoRef: escenario.base.ID,
				Numero:              resultado.Version.Referencia.Numero,
			})
			return err
		},
		"evidencia": func(ctx context.Context) error {
			_, err := escenario.repositorio.ObtenerEvidenciaTransaccion(
				ctx,
				puertosbolsa.SolicitudObtenerEvidenciaTransaccionBaremacion{
					Contexto: contextoMemoriaPrueba(
						puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
						resultado.Evidencia.AuditoriaRef, escenario.reloj.Ahora(),
					),
					BaremacionMeritoRef: escenario.base.ID,
					NumeroVersion:       resultado.Version.Referencia.Numero,
					AuditoriaRef:        resultado.Evidencia.AuditoriaRef,
					EventoOutboxRef:     resultado.Evidencia.EventoOutboxRef,
				},
			)
			return err
		},
	}
}
