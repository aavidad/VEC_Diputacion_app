package bootstrap

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// lectorResumenPrueba sirve con dos lecturas lo mismo que los dobles de
// situaciones, ceses y orden de fuenteVariasBolsasRRHHPrueba.
type lectorResumenPrueba struct {
	repo     repositorioVariasBolsasRRHHPrueba
	lecturas *atomic.Int64
}

func (l lectorResumenPrueba) LeerResumenSituaciones(context.Context, time.Time) ([]ports.SituacionResumenParticipacion, error) {
	l.lecturas.Add(1)
	var filas []ports.SituacionResumenParticipacion
	for _, vigente := range l.repo.vigentes {
		for _, entrada := range l.repo.entradas[vigente.Instantanea.InstantaneaRef] {
			situacion := situacionPruebaLote(entrada.ParticipacionRef)
			fila := ports.SituacionResumenParticipacion{BolsaRef: vigente.Bolsa.BolsaRef, CategoriaRef: vigente.CategoriaRef, ConfirmadaEn: vigente.ConfirmadaEn, InstantaneaRef: vigente.Instantanea.InstantaneaRef, VersionInstantanea: vigente.Instantanea.Version,
				Orden: entrada.Orden, ParticipacionRef: entrada.ParticipacionRef, Situacion: &situacion}
			if estado, presente := estadoCesePrueba(entrada.ParticipacionRef); presente {
				fila.Cese = &estado
			}
			filas = append(filas, fila)
		}
	}
	return filas, nil
}

func (l lectorResumenPrueba) LeerPoliticasOrdenVigentes(context.Context, time.Time) (map[string]bolsadominio.PoliticaOrdenBolsa, error) {
	l.lecturas.Add(1)
	salida := map[string]bolsadominio.PoliticaOrdenBolsa{}
	for _, vigente := range l.repo.vigentes {
		orden, _ := ordenContadoRRHHPrueba{c: &contadoresBolsasRRHHPrueba{}, entradas: l.repo.entradas}.ConsultarOrdenVigente(context.Background(), vigente.Bolsa.BolsaRef)
		salida[vigente.Bolsa.BolsaRef] = orden.Politica
	}
	return salida, nil
}

// Con Bolsa 000082 el cuadro hace dos lecturas de conjunto, ninguna por bolsa
// ni por participación, y responde exactamente lo mismo.
func TestFuenteConstituidaRRHHResumenConjuntoIgualQueLecturaPorBolsa(t *testing.T) {
	const bolsas, participaciones = 6, 250
	porBolsa, _ := fuenteVariasBolsasRRHHPrueba(t, bolsas, participaciones, true)
	esperado, err := porBolsa.cargarResumen(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f, c := fuenteVariasBolsasRRHHPrueba(t, bolsas, participaciones, true)
	var lecturas atomic.Int64
	f.resumenConjunto = lectorResumenPrueba{repo: f.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &lecturas}
	// Sin repositorio: el resumen de conjunto no descarga las instantáneas.
	f.repositorio = nil
	obtenido, err := f.cargarResumen(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if lecturas.Load() != 2 || c.orden.Load() != 0 || c.situacionLote.Load() != 0 || c.ceseLote.Load() != 0 ||
		c.situacionIndividual.Load() != 0 || c.ceseIndividual.Load() != 0 || c.recuperar.Load() != 0 {
		t.Fatalf("lecturas: conjunto=%d orden=%d situaciones=%d/%d ceses=%d/%d actas=%d", lecturas.Load(), c.orden.Load(),
			c.situacionLote.Load(), c.situacionIndividual.Load(), c.ceseLote.Load(), c.ceseIndividual.Load(), c.recuperar.Load())
	}
	for _, par := range [][2]any{
		{(&bolsasRRHHDesarrolloDatos{datos: esperado}).respuestaBolsas(), (&bolsasRRHHDesarrolloDatos{datos: obtenido}).respuestaBolsas()},
		{(&bolsasRRHHDesarrolloDatos{datos: esperado}).respuestaEstadisticas(), (&bolsasRRHHDesarrolloDatos{datos: obtenido}).respuestaEstadisticas()},
	} {
		a, _ := json.Marshal(par[0])
		b, _ := json.Marshal(par[1])
		if string(a) != string(b) {
			t.Fatalf("respuesta distinta:\n%s\n%s", a, b)
		}
	}
}

// Una bolsa sin política vigente o una participación sin situación falla
// cerrada, como la lectura por bolsa.
func TestFuenteConstituidaRRHHResumenConjuntoFallaCerrado(t *testing.T) {
	f, _ := fuenteVariasBolsasRRHHPrueba(t, 2, 3, true)
	var lecturas atomic.Int64
	base := lectorResumenPrueba{repo: f.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &lecturas}
	f.resumenConjunto = sinPoliticaPrueba{base}
	if _, err := f.cargarResumen(context.Background()); err == nil {
		t.Fatal("se sirvió una bolsa sin política de orden")
	}
	f.resumenConjunto = sinSituacionPrueba{base}
	if _, err := f.cargarResumen(context.Background()); err == nil {
		t.Fatal("se sirvió una participación sin situación")
	}
}

type sinPoliticaPrueba struct{ lectorResumenPrueba }

func (sinPoliticaPrueba) LeerPoliticasOrdenVigentes(context.Context, time.Time) (map[string]bolsadominio.PoliticaOrdenBolsa, error) {
	return map[string]bolsadominio.PoliticaOrdenBolsa{}, nil
}

type sinSituacionPrueba struct{ lectorResumenPrueba }

func (s sinSituacionPrueba) LeerResumenSituaciones(ctx context.Context, corte time.Time) ([]ports.SituacionResumenParticipacion, error) {
	filas, err := s.lectorResumenPrueba.LeerResumenSituaciones(ctx, corte)
	filas[len(filas)-1].Situacion = nil
	return filas, err
}
