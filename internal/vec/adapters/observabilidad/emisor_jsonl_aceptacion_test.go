package observabilidad

import (
	"context"
	"testing"
	"time"
)

func TestAceptarConContextoDistingueColaAceptadaYDescartada(t *testing.T) {
	destino := nuevoDestinoBloqueado()
	emisor := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Capacidad: 1})
	if !emisor.AceptarConContexto(context.Background(), solicitudValida()) {
		t.Fatal("la primera incidencia no entró en cola")
	}
	select {
	case <-destino.entrado: // el trabajador bloquea la escritura y libera la única plaza
	case <-time.After(2 * time.Second):
		close(destino.liberar)
		t.Fatal("el trabajador no llegó al destino")
	}
	if !emisor.AceptarConContexto(context.Background(), solicitudValida()) {
		t.Fatal("la segunda incidencia no entró en cola")
	}
	if emisor.AceptarConContexto(context.Background(), solicitudValida()) {
		t.Fatal("aceptó una incidencia con cola llena")
	}
	if got := emisor.MetricasEmision(); got.Aceptadas != 2 || got.Descartadas != 1 {
		t.Fatalf("métricas antes de vaciar = %+v", got)
	}
	close(destino.liberar)
	cerrar(t, emisor)
	if emisor.AceptarConContexto(context.Background(), solicitudValida()) {
		t.Fatal("aceptó después del cierre")
	}
	var ausente *EmisorJSONLines
	if ausente.AceptarConContexto(context.Background(), solicitudValida()) {
		t.Fatal("un emisor nulo confirmó aceptación")
	}
}
