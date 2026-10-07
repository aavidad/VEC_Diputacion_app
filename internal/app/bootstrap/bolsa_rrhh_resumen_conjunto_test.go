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
	conteos  map[string]int
}

func (l lectorResumenPrueba) LeerResumen(ctx context.Context, corte time.Time) (ports.ResumenBolsasRRHH, error) {
	l.lecturas.Add(1)
	filas, _ := l.situaciones()
	politicas, _ := l.politicas()
	conteos := make(map[string]int, len(l.repo.vigentes))
	for _, vigente := range l.repo.vigentes {
		conteos[vigente.Bolsa.BolsaRef] = l.conteos[vigente.Bolsa.BolsaRef]
	}
	return ports.ResumenBolsasRRHH{Situaciones: filas, Politicas: politicas, LlamamientosEnCurso: conteos}, nil
}

func (l lectorResumenPrueba) situaciones() ([]ports.SituacionResumenParticipacion, error) {
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

func (l lectorResumenPrueba) politicas() (map[string]bolsadominio.PoliticaOrdenBolsa, error) {
	salida := map[string]bolsadominio.PoliticaOrdenBolsa{}
	for _, vigente := range l.repo.vigentes {
		orden, _ := ordenContadoRRHHPrueba{c: &contadoresBolsasRRHHPrueba{}, entradas: l.repo.entradas}.ConsultarOrdenVigente(context.Background(), vigente.Bolsa.BolsaRef)
		salida[vigente.Bolsa.BolsaRef] = orden.Politica
	}
	return salida, nil
}

type emisionesContadasResumenPrueba struct{ llamadas *atomic.Int64 }

func (e emisionesContadasResumenPrueba) ContarEnCurso(context.Context, string) (int, error) {
	e.llamadas.Add(1)
	return 0, nil
}

// Con B82/B85 el cuadro hace una lectura de conjunto (tres funciones en una
// transacción), ninguna por bolsa ni por participación, y conserva el JSON.
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
	var emisiones atomic.Int64
	f.emisiones = emisionesContadasResumenPrueba{llamadas: &emisiones}
	// Sin repositorio: el resumen de conjunto no descarga las instantáneas.
	f.repositorio = nil
	obtenido, err := f.cargarResumen(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if lecturas.Load() != 1 || emisiones.Load() != 0 || c.orden.Load() != 0 || c.situacionLote.Load() != 0 || c.ceseLote.Load() != 0 ||
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

func TestFuenteConstituidaRRHHResumenConjuntoUsaConteosAgrupados(t *testing.T) {
	f, _ := fuenteVariasBolsasRRHHPrueba(t, 2, 3, true)
	var lecturas, emisiones atomic.Int64
	f.resumenConjunto = lectorResumenPrueba{repo: f.repositorio.(repositorioVariasBolsasRRHHPrueba), lecturas: &lecturas,
		conteos: map[string]int{"bolsa:prueba:01": 2}}
	f.emisiones = emisionesContadasResumenPrueba{llamadas: &emisiones}
	f.repositorio = nil
	datos, err := f.cargarResumen(context.Background())
	if err != nil || lecturas.Load() != 1 || emisiones.Load() != 0 || len(datos.Bolsas) != 2 ||
		datos.Bolsas[0].LlamamientosEnCurso != 2 || datos.Bolsas[1].LlamamientosEnCurso != 0 {
		t.Fatalf("recuentos agrupados: lecturas=%d emisiones=%d bolsas=%d error=%v", lecturas.Load(), emisiones.Load(), len(datos.Bolsas), err)
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
	f.resumenConjunto = sinRecuentoPrueba{base}
	if _, err := f.cargarResumen(context.Background()); err == nil {
		t.Fatal("se sirvió una bolsa sin recuento agrupado")
	}
}

type sinPoliticaPrueba struct{ lectorResumenPrueba }

func (s sinPoliticaPrueba) LeerResumen(context.Context, time.Time) (ports.ResumenBolsasRRHH, error) {
	filas, _ := s.situaciones()
	conteos := map[string]int{}
	for _, vigente := range s.repo.vigentes {
		conteos[vigente.Bolsa.BolsaRef] = 0
	}
	return ports.ResumenBolsasRRHH{Situaciones: filas, Politicas: map[string]bolsadominio.PoliticaOrdenBolsa{}, LlamamientosEnCurso: conteos}, nil
}

type sinSituacionPrueba struct{ lectorResumenPrueba }

func (s sinSituacionPrueba) LeerResumen(ctx context.Context, corte time.Time) (ports.ResumenBolsasRRHH, error) {
	resumen, err := s.lectorResumenPrueba.LeerResumen(ctx, corte)
	resumen.Situaciones[len(resumen.Situaciones)-1].Situacion = nil
	return resumen, err
}

type sinRecuentoPrueba struct{ lectorResumenPrueba }

func (s sinRecuentoPrueba) LeerResumen(ctx context.Context, corte time.Time) (ports.ResumenBolsasRRHH, error) {
	resumen, err := s.lectorResumenPrueba.LeerResumen(ctx, corte)
	delete(resumen.LlamamientosEnCurso, s.repo.vigentes[0].Bolsa.BolsaRef)
	return resumen, err
}
