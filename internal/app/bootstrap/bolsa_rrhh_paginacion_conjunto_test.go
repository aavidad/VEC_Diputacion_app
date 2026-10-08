package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

type lectorBolsaConjuntoPrueba struct {
	repo     repositorioVariasBolsasRRHHPrueba
	lecturas *atomic.Int64
	alterar  func([]ports.SituacionBolsaRRHH)
	fallo    error
}

func (l lectorBolsaConjuntoPrueba) LeerBolsa(ctx context.Context, bolsa string, corte time.Time) (dominiobolsa.OrdenVigenteBolsa, []ports.SituacionBolsaRRHH, int, error) {
	l.lecturas.Add(1)
	if l.fallo != nil {
		return dominiobolsa.OrdenVigenteBolsa{}, nil, 0, l.fallo
	}
	orden, err := ordenContadoRRHHPrueba{c: &contadoresBolsasRRHHPrueba{}, entradas: l.repo.entradas}.ConsultarOrdenVigente(ctx, bolsa)
	if err != nil {
		return orden, nil, 0, err
	}
	resumen, err := (lectorResumenPrueba{repo: l.repo, lecturas: &atomic.Int64{}}).situaciones()
	if err != nil {
		return orden, nil, 0, err
	}
	filas := make([]ports.SituacionBolsaRRHH, 0, len(orden.Posiciones))
	numeros := make(map[string]int, len(l.repo.entradas["instantanea:"+bolsa]))
	for _, entrada := range l.repo.entradas["instantanea:"+bolsa] {
		numeros[entrada.ParticipacionRef] = entrada.FilaNumero
	}
	for _, fila := range resumen {
		if fila.BolsaRef == bolsa {
			filas = append(filas, ports.SituacionBolsaRRHH{SituacionResumenParticipacion: fila, FilaNumero: numeros[fila.ParticipacionRef]})
		}
	}
	if l.alterar != nil {
		l.alterar(filas)
	}
	return orden, filas, 0, nil
}

func TestFuenteConstituidaRRHHBolsaConjuntoConservaListaSinConsultasPorParticipacion(t *testing.T) {
	const bolsas, personasPorBolsa = 2, 200
	legado, _ := fuenteVariasBolsasRRHHPrueba(t, bolsas, personasPorBolsa, true)
	ref := "bolsa:prueba:01"
	esperado, err := legado.cargarAlcance(context.Background(), alcanceCargaBolsasRRHH{bolsa: ref, detalle: true})
	if err != nil {
		t.Fatal(err)
	}
	actual, contadores := fuenteVariasBolsasRRHHPrueba(t, bolsas, personasPorBolsa, true)
	var lecturas atomic.Int64
	actual.bolsaConjunto = lectorBolsaConjuntoPrueba{repo: actual.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &lecturas}
	obtenido, err := actual.cargarBolsaRRHH(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if lecturas.Load() != 1 || contadores.orden.Load() != 0 || contadores.situacionLote.Load() != 0 ||
		contadores.ceseLote.Load() != 0 || contadores.situacionIndividual.Load() != 0 || contadores.ceseIndividual.Load() != 0 {
		t.Fatalf("lecturas conjunto=%d orden=%d situaciones=%d/%d ceses=%d/%d", lecturas.Load(), contadores.orden.Load(),
			contadores.situacionLote.Load(), contadores.situacionIndividual.Load(), contadores.ceseLote.Load(), contadores.ceseIndividual.Load())
	}
	if fmt.Sprint(obtenido.Bolsas) != fmt.Sprint(esperado.Bolsas) || fmt.Sprint(obtenido.Candidaturas) != fmt.Sprint(esperado.Candidaturas) {
		t.Fatal("la lectura de conjunto cambió bolsas o candidaturas")
	}
}

func TestFuenteConstituidaRRHHBolsaConjuntoRechazaInstantaneaDivergente(t *testing.T) {
	f, _ := fuenteVariasBolsasRRHHPrueba(t, 1, 2, true)
	repo := f.repositorio.(repositorioVariasBolsasRRHHPrueba)
	lecturas := &atomic.Int64{}
	lector := lectorBolsaConjuntoPrueba{repo: repo, lecturas: lecturas}
	_, filas, _, err := lector.LeerBolsa(context.Background(), repo.vigentes[0].Bolsa.BolsaRef, f.ahora())
	if err != nil {
		t.Fatal(err)
	}
	filas[0].VersionInstantanea++
	if _, _, err := mapearSituacionesConjuntoBolsa(repo.vigentes[0], repo.entradas[repo.vigentes[0].Instantanea.InstantaneaRef], filas); err == nil {
		t.Fatal("se aceptó una situación de otra instantánea")
	}
	filas[0].VersionInstantanea--
	filas[0].FilaNumero++
	if _, _, err := mapearSituacionesConjuntoBolsa(repo.vigentes[0], repo.entradas[repo.vigentes[0].Instantanea.InstantaneaRef], filas); err == nil {
		t.Fatal("se aceptó un número de fila distinto del acta")
	}
}

func TestFuenteConstituidaRRHHBolsaConjuntoConservaExclusionAnteCesePendiente(t *testing.T) {
	f, _ := fuenteVariasBolsasRRHHPrueba(t, 1, 2, true)
	repo := f.repositorio.(repositorioVariasBolsasRRHHPrueba)
	pendienteDesde := f.ahora().Add(-time.Hour)
	excluidaDesde := f.ahora().Add(-24 * time.Hour)
	var lecturas atomic.Int64
	f.bolsaConjunto = lectorBolsaConjuntoPrueba{repo: repo, lecturas: &lecturas, alterar: func(filas []ports.SituacionBolsaRRHH) {
		filas[0].Situacion = &ports.SituacionParticipacion{ParticipacionRef: filas[0].ParticipacionRef, Situacion: "disponible", Desde: excluidaDesde}
		filas[0].Cese = &ports.EstadoCese{CesePendiente: true, PendienteDesde: pendienteDesde}
		filas[1].Situacion = &ports.SituacionParticipacion{ParticipacionRef: filas[1].ParticipacionRef, Situacion: "excluido", Desde: excluidaDesde}
		filas[1].Cese = &ports.EstadoCese{CesePendiente: true, PendienteDesde: pendienteDesde}
	}}
	datos, err := f.cargarBolsaRRHH(context.Background(), repo.vigentes[0].Bolsa.BolsaRef)
	if err != nil || lecturas.Load() != 1 || len(datos.Candidaturas) != 2 {
		t.Fatalf("lectura pendiente: error=%v lecturas=%d candidaturas=%d", err, lecturas.Load(), len(datos.Candidaturas))
	}
	if datos.Candidaturas[0].Estado != "no_disponible" || datos.Candidaturas[0].EstadoDesde != pendienteDesde.UTC().Format(time.RFC3339) ||
		datos.Candidaturas[1].Estado != "excluido" || datos.Candidaturas[1].EstadoDesde != excluidaDesde.UTC().Format(time.RFC3339) {
		t.Fatalf("estados después de B13: primera=%+v segunda=%+v", datos.Candidaturas[0], datos.Candidaturas[1])
	}
}

func TestBolsaPublicaNoDependeDelLectorB92RRHH(t *testing.T) {
	f, _ := fuenteVariasBolsasRRHHPrueba(t, 1, 2, true)
	repo := f.repositorio.(repositorioVariasBolsasRRHHPrueba)
	var lecturas atomic.Int64
	f.bolsaConjunto = lectorBolsaConjuntoPrueba{repo: repo, lecturas: &lecturas, fallo: errors.New("lector B92 no disponible")}
	publica := &fuenteBolsasPublicasDesarrollo{fuente: f}
	bolsa, posiciones, _, err := publica.ListaPublica(context.Background(), repo.vigentes[0].Bolsa.BolsaRef)
	if err != nil || bolsa.BolsaRef != repo.vigentes[0].Bolsa.BolsaRef || len(posiciones) != 2 || lecturas.Load() != 0 {
		t.Fatalf("B10 depende de B92: bolsa=%+v posiciones=%d lecturas=%d error=%v", bolsa, len(posiciones), lecturas.Load(), err)
	}
	h := nuevoManejadorBolsasRRHHDesarrollo(f.cargar)
	h.cargarBolsa = f.cargarBolsaRRHH
	respuesta := httptest.NewRecorder()
	h.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/"+bolsa.BolsaRef+"/candidatos", nil))
	if respuesta.Code != http.StatusServiceUnavailable || lecturas.Load() != 1 {
		t.Fatalf("RRHH no rechazó fallo B92: status=%d lecturas=%d", respuesta.Code, lecturas.Load())
	}
}
