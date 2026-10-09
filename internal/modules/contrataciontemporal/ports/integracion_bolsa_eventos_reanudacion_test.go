package ports

import (
	"context"
	"testing"
	"time"
)

func TestEventoLlamamientoRecuperadoConservaFechaYVinculoAutenticado(t *testing.T) {
	ctx, base := context.Background(), instanteBolsaPrueba()
	ahora := base.Add(3 * time.Minute)
	sellador := selladorRespuestaBolsaPrueba()
	c := comandoLlamamientoPrueba(t, base, sellador)
	r := reciboLlamamientoPrueba(t, c, base)
	r.LlamamientoRecuperado = true
	r.ConfirmadaEn = base.Add(-time.Minute)
	firmarLlamamientoPrueba(t, sellador, c, &r)
	prueba := autenticarReciboLlamamientoPrueba(t, c, r, ahora)
	enlace, err := NuevoEnlaceEventoLlamamientoBolsa(PreparacionEnlaceEventoLlamamientoBolsa{Comando: c, Recibo: r, Comprobante: prueba})
	if err != nil {
		t.Fatal(err)
	}
	evento := nuevoEventoParaEnlaceBolsaPrueba(t, base, enlace)
	v := verificadorRespuestaBolsaPrueba(claveRespuestaBolsaV1Prueba)
	comp, _, err := v.VerificarEvento(ctx, evento, enlace, ahora)
	if err != nil {
		t.Fatal("evento recuperado:", err)
	}
	comando, err := NuevoComandoRegistrarEventoBolsa(evento, enlace, comp, ahora)
	if err != nil {
		t.Fatal(err)
	}
	datos, _, err := comando.DatosParaEfectoEn(ahora)
	if err != nil || datos != evento || datos.ReciboConfirmadaEn != r.ConfirmadaEn {
		t.Fatal("evento cambió confirmación", err)
	}
	copia := *enlace.datos
	copia.llamamientoRecuperado = false
	if evento.ValidarParaEn(EnlaceEventoLlamamientoBolsa{datos: &copia}, ahora) == nil {
		t.Fatal("fecha antigua aceptada sin enlace de recuperación")
	}
	mutado := evento
	mutado.ReciboEvidenciaEmitidaEn = base.Add(-time.Second)
	mutado.HuellaCargaSHA256 = huellaBytesBolsa(materialEventoBolsa(mutado))
	if mutado.ValidarEn(ahora) == nil {
		t.Fatal("recuperación admitió evidencia anterior a la petición")
	}
	r.LlamamientoRecuperado = false
	if _, err = NuevoEnlaceEventoLlamamientoBolsa(PreparacionEnlaceEventoLlamamientoBolsa{Comando: c, Recibo: r, Comprobante: prueba}); err == nil {
		t.Fatal("marca retirada conservó enlace")
	}
}
