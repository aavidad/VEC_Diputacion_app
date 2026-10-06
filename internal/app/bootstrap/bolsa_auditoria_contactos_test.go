package bootstrap

import (
	"context"
	"testing"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestCorrelacionConsultaContactosSoloDeLaPeticionEnSusRutas(t *testing.T) {
	base, err := puertosvec.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	dePeticion, err := puertosvec.ReferenciaCorrelacionAutorizacionV2DePeticion(base)
	if err != nil {
		t.Fatal(err)
	}
	esperada, _ := dePeticion.ValorCanonico()
	p := &preparadorBorradorLlamamientoDesarrollo{}
	conRuta := func(ctx context.Context, ruta string) context.Context {
		return context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{ruta: ruta})
	}
	for _, ruta := range []string{bolsahttp.RutaBolsasGestion + "/bolsa:1/candidatos/participacion:1/contactos", bolsahttp.RutaContactosOferta} {
		correlacion, err := p.correlacionConsultaContactos(conRuta(base, ruta))
		if valor, _ := correlacion.ValorCanonico(); err != nil || valor != esperada {
			t.Fatalf("%s: correlación de la petición perdida", ruta)
		}
		if _, err := p.correlacionConsultaContactos(conRuta(context.Background(), ruta)); err == nil {
			t.Fatalf("%s: correlación inventada sin petición", ruta)
		}
	}
	// La página RRHH de la bolsa pagina varias lecturas: cada una conserva
	// una correlación propia.
	for _, ruta := range []string{"/portal/bolsa/rrhh", bolsahttp.RutaBolsasGestion + "/bolsa:1/candidatos/participacion:1/datos-contacto"} {
		correlacion, err := p.correlacionConsultaContactos(conRuta(base, ruta))
		if valor, _ := correlacion.ValorCanonico(); err != nil || valor == esperada {
			t.Fatalf("%s: correlación de la petición fuera de las rutas de contactos", ruta)
		}
	}
}
