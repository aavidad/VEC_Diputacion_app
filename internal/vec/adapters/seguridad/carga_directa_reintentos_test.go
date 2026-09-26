package seguridad

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestReciboCargaDirectaRecuperaLaMismaIntencionSinOtroConsumo(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	solicitud := solicitudConsumoCargaDirectaPrueba(recibo)
	comprobante, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("primer consumo: %v", err)
	}
	confirmacion, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), solicitud.Contexto, solicitud.SesionRef, comprobante, adaptador,
	)
	if err != nil {
		t.Fatalf("atestacion invalida: %v", err)
	}
	_, _, intencion, huellaIntencion, referencia, emitidoEn, consumidoEn, expiraEn, validaHasta, err := confirmacion.RevelarParaConector()
	if err != nil {
		t.Fatalf("confirmacion no revelable: %v", err)
	}
	if !strings.HasPrefix(intencion, "confirmacion-intencion-v1:") ||
		!strings.HasPrefix(referencia, "recibo-consumo-v1:") ||
		!strings.HasPrefix(huellaIntencion, "hmac-sha256:atestacion_recibo_v1:") ||
		!consumidoEn.Equal(repositorio.horaDurable) || emitidoEn.After(consumidoEn) ||
		!expiraEn.After(consumidoEn) || !validaHasta.After(consumidoEn) {
		t.Fatalf("evidencia no opaca o fecha incorrecta")
	}
	repetido, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("repeticion exacta no recuperada: %v", err)
	}
	_, _, _, evidenciaRepetida, intencionRepetida, huellaRepetida,
		_, consumidoRepetido, _, _, _, err := repetido.RevelarParaVerificacion()
	if err != nil || evidenciaRepetida != referencia || intencionRepetida != intencion ||
		huellaRepetida != huellaIntencion || !consumidoRepetido.Equal(consumidoEn) {
		t.Fatal("la recuperacion cambio la intencion o la fecha del consumo")
	}
	_, consumos, _, usados := repositorio.estado()
	if consumos != 2 || usados != 1 {
		t.Fatalf("consumos=%d usados=%d", consumos, usados)
	}
}

func TestReciboCargaDirectaReintentoConDecisionNuevaConservaIntencion(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	primera := solicitudConsumoCargaDirectaPrueba(recibo)
	comprobantePrimero, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), primera)
	if err != nil {
		t.Fatal(err)
	}
	segunda := solicitudConsumoCargaDirectaPrueba(recibo)
	mutarContextoConsumoCargaDirecta(func(d *domain.DecisionAutorizacion, _ *domain.RecursoAutorizable, _ *ports.VinculosOperacionAlmacen) {
		d.DecisionRef = "autorizacion:confirmar:reintento"
	})(&segunda)
	comprobanteSegundo, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), segunda)
	if err != nil {
		t.Fatalf("reintento con decision nueva: %v", err)
	}
	_, _, _, evidenciaPrimera, intencionPrimera, huellaPrimera,
		_, consumidoPrimero, _, _, atestacionPrimera, _ := comprobantePrimero.RevelarParaVerificacion()
	_, _, _, evidenciaSegunda, intencionSegunda, huellaSegunda,
		_, consumidoSegundo, _, _, atestacionSegunda, _ := comprobanteSegundo.RevelarParaVerificacion()
	if evidenciaPrimera != evidenciaSegunda || intencionPrimera != intencionSegunda ||
		huellaPrimera != huellaSegunda || !consumidoPrimero.Equal(consumidoSegundo) ||
		atestacionPrimera == atestacionSegunda {
		t.Fatal("la recuperacion no mantuvo la intencion con atestacion nueva")
	}
	if _, err := ports.NuevaSolicitudConfirmarCargaDirecta(
		context.Background(), segunda.Contexto, segunda.SesionRef, comprobanteSegundo, adaptador,
	); err != nil {
		t.Fatalf("atestacion del nuevo contexto: %v", err)
	}
	if _, _, _, usados := repositorio.estado(); usados != 1 {
		t.Fatalf("consumos durables=%d", usados)
	}
}

func TestConsumoConcurrenteDeReciboCargaDirectaConservaUnaIntencion(t *testing.T) {
	adaptador, repositorio, reloj := nuevoAdaptadorCargaDirectaPrueba(t)
	recibo := emitirReciboCargaDirectaPrueba(t, context.Background(), adaptador, reloj)
	solicitud := solicitudConsumoCargaDirectaPrueba(recibo)
	const trabajadores = 48
	var exitos atomic.Int32
	var fallosInesperados atomic.Int32
	inicio := make(chan struct{})
	var grupo sync.WaitGroup
	for indice := 0; indice < trabajadores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			_, err := adaptador.ConsumirReciboCargaDirecta(context.Background(), solicitud)
			if err == nil {
				exitos.Add(1)
				return
			}
			if !errors.Is(err, ports.ErrReciboCargaDirectaNoValido) {
				fallosInesperados.Add(1)
			}
		}()
	}
	close(inicio)
	grupo.Wait()
	if exitos.Load() != trabajadores || fallosInesperados.Load() != 0 {
		t.Fatalf("exitos=%d fallos inesperados=%d", exitos.Load(), fallosInesperados.Load())
	}
	_, _, _, usados := repositorio.estado()
	if usados != 1 {
		t.Fatalf("usos persistidos=%d", usados)
	}
}
