package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// fuentePaginadaPrueba sirve el tramo pedido y anota las llamadas para
// comprobar que el manejador no pide la lista completa al paginar.
type fuentePaginadaPrueba struct {
	fuentePublicaPrueba
	pedidas  *[][2]int
	completa *int
}

func (f fuentePaginadaPrueba) ListaPublica(ctx context.Context, ref string) (BolsaPublica, []PosicionPublica, time.Time, error) {
	*f.completa++
	return f.fuentePublicaPrueba.ListaPublica(ctx, ref)
}

func (f fuentePaginadaPrueba) PaginaListaPublica(ctx context.Context, ref string, desde, cantidad int) (BolsaPublica, []PosicionPublica, time.Time, error) {
	*f.pedidas = append(*f.pedidas, [2]int{desde, cantidad})
	bolsa, todas, instante, err := f.fuentePublicaPrueba.ListaPublica(ctx, ref)
	if err != nil {
		return bolsa, nil, instante, err
	}
	var tramo []PosicionPublica
	for _, p := range todas {
		if p.Orden >= desde && len(tramo) < cantidad {
			tramo = append(tramo, p)
		}
	}
	return bolsa, tramo, instante, nil
}

func TestBolsasPublicasPaginaPideSoloElTramoYRespondeIgual(t *testing.T) {
	base := fuentePruebaConPosiciones(237)
	var pedidas [][2]int
	completas := 0
	paginada := fuentePaginadaPrueba{fuentePublicaPrueba: base, pedidas: &pedidas, completa: &completas}
	ruta := RutaBolsasPublicas + "/bolsa:administrativo:2026-09-17/lista"
	consultas := []string{"", "?limite=1", "?limite=100", "?limite=10&cursor=2", "?limite=50&cursor=188",
		"?limite=50&cursor=237", "?limite=50&cursor=238", "?cursor=100000", "?limite=100&cursor=138"}
	for _, consulta := range consultas {
		esperada := servirPublico(t, base, http.MethodGet, ruta+consulta)
		obtenida := servirPublico(t, paginada, http.MethodGet, ruta+consulta)
		if esperada.Code != http.StatusOK || obtenida.Code != esperada.Code || obtenida.Body.String() != esperada.Body.String() {
			t.Fatalf("%s: paginada %d %s; completa %d %s", consulta, obtenida.Code, obtenida.Body.String(), esperada.Code, esperada.Body.String())
		}
	}
	if completas != 0 || len(pedidas) != len(consultas) {
		t.Fatalf("el manejador debe pedir solo tramos: completas=%d tramos=%v", completas, pedidas)
	}
	if pedidas[0] != [2]int{1, 51} || pedidas[3] != [2]int{2, 11} || pedidas[8] != [2]int{138, 101} {
		t.Fatalf("tramos pedidos inesperados: %v", pedidas)
	}
}

func TestBolsasPublicasBusquedaPorDocumentoSigueLeyendoLaListaCompleta(t *testing.T) {
	base := fuentePruebaConPosiciones(60)
	var pedidas [][2]int
	completas := 0
	paginada := fuentePaginadaPrueba{fuentePublicaPrueba: base, pedidas: &pedidas, completa: &completas}
	ruta := RutaBolsasPublicas + "/bolsa:administrativo:2026-09-17/lista?documento=" + fmt.Sprintf("***%04d**", 42)
	esperada := servirPublico(t, base, http.MethodGet, ruta)
	obtenida := servirPublico(t, paginada, http.MethodGet, ruta)
	if obtenida.Code != http.StatusOK || obtenida.Body.String() != esperada.Body.String() {
		t.Fatalf("búsqueda por documento: %d %s", obtenida.Code, obtenida.Body.String())
	}
	if completas != 1 || len(pedidas) != 0 {
		t.Fatalf("la búsqueda por documento debe leer la lista completa: completas=%d tramos=%v", completas, pedidas)
	}
}
