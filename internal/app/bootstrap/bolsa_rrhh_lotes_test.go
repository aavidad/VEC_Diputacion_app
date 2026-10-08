package bootstrap

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	bolsaapplication "vec-diputacion-granada/internal/modules/bolsa/application"
	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	bolsadominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	importaciondominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// contadoresBolsasRRHHPrueba registra cuántas lecturas hace la fuente para
// fijar que el coste por petición no crece con el número de participaciones.
type contadoresBolsasRRHHPrueba struct {
	situacionIndividual, situacionLote atomic.Int64
	ceseIndividual, ceseLote           atomic.Int64
	orden, recuperar                   atomic.Int64
}

type repositorioVariasBolsasRRHHPrueba struct {
	ports.RepositorioConstitucion
	vigentes []ports.ConstitucionVigente
	entradas map[string][]ports.EntradaConstitucion
}

func (r repositorioVariasBolsasRRHHPrueba) ListarVigentes(context.Context) ([]ports.ConstitucionVigente, error) {
	return r.vigentes, nil
}

func (r repositorioVariasBolsasRRHHPrueba) Entradas(_ context.Context, instantanea string, _ uint64) ([]ports.EntradaConstitucion, error) {
	return r.entradas[instantanea], nil
}

type situacionesContadasRRHHPrueba struct {
	situacionesBolsasRRHHPrueba
	c *contadoresBolsasRRHHPrueba
}

func (s situacionesContadasRRHHPrueba) SituacionVigente(_ context.Context, ref string) (ports.SituacionParticipacion, error) {
	s.c.situacionIndividual.Add(1)
	return situacionPruebaLote(ref), nil
}

func (s situacionesContadasRRHHPrueba) SituacionesVigentes(_ context.Context, refs []string) (map[string]ports.SituacionParticipacion, error) {
	s.c.situacionLote.Add(1)
	salida := make(map[string]ports.SituacionParticipacion, len(refs))
	for _, ref := range refs {
		salida[ref] = situacionPruebaLote(ref)
	}
	return salida, nil
}

func situacionPruebaLote(ref string) ports.SituacionParticipacion {
	return ports.SituacionParticipacion{ParticipacionRef: ref, Situacion: "trabajando", Desde: time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)}
}

type cesesContadosRRHHPrueba struct{ c *contadoresBolsasRRHHPrueba }

// La primera participación de cada bolsa tiene un cese acreditado que la
// devuelve a «disponible»; así se comprueba que el lote se proyecta igual.
func estadoCesePrueba(ref string) (ports.EstadoCese, bool) {
	if len(ref) < 4 || ref[len(ref)-4:] != ":001" {
		return ports.EstadoCese{}, false
	}
	efecto := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	return ports.EstadoCese{FechaEfecto: efecto, DisponibleDesde: efecto, TrabajoCesado: true}, true
}

func (c cesesContadosRRHHPrueba) ConsultarEstadoCese(_ context.Context, ref string, _ time.Time) (ports.EstadoCese, bool, error) {
	c.c.ceseIndividual.Add(1)
	estado, presente := estadoCesePrueba(ref)
	return estado, presente, nil
}

func (c cesesContadosRRHHPrueba) ConsultarEstadosCese(_ context.Context, refs []string, _ time.Time) (map[string]ports.EstadoCese, error) {
	c.c.ceseLote.Add(1)
	salida := map[string]ports.EstadoCese{}
	for _, ref := range refs {
		if estado, presente := estadoCesePrueba(ref); presente {
			salida[ref] = estado
		}
	}
	return salida, nil
}

type ordenContadoRRHHPrueba struct {
	c        *contadoresBolsasRRHHPrueba
	entradas map[string][]ports.EntradaConstitucion
}

func (o ordenContadoRRHHPrueba) ConsultarOrdenVigente(_ context.Context, bolsa string) (bolsadominio.OrdenVigenteBolsa, error) {
	o.c.orden.Add(1)
	orden := bolsadominio.OrdenVigenteBolsa{Politica: bolsadominio.PoliticaOrdenBolsa{
		PoliticaRef: "politica:" + bolsa, BolsaRef: bolsa, Version: 1, Criterio: bolsadominio.CriterioPuntuacionDescActa,
		TipoLista: bolsadominio.TipoListaRotatoria, Reposicion: bolsadominio.ReposicionMismaPosicion,
		Rotulo: "Provisional", Actor: "sistema:prueba", VigenteDesde: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Provisional: true,
	}}
	for _, entrada := range o.entradas["instantanea:"+bolsa] {
		orden.Posiciones = append(orden.Posiciones, bolsadominio.PosicionOrdenBolsa{ParticipacionRef: entrada.ParticipacionRef, OrdenActa: entrada.Orden, Situacion: "disponible", Razon: "orden_acta"})
	}
	return orden, nil
}

type emisionesPrueba struct{}

func (emisionesPrueba) ContarEnCurso(context.Context, string) (int, error) { return 0, nil }

type recuperadorContadoRRHHPrueba struct{ c *contadoresBolsasRRHHPrueba }

func (r recuperadorContadoRRHHPrueba) RecuperarLote(context.Context, string, string) (importaciondominio.LoteValidado, importacionapp.EstadoImportacion, bool, error) {
	r.c.recuperar.Add(1)
	var lote importaciondominio.LoteValidado
	for numero := 2; numero < 2+500; numero++ {
		var fila importaciondominio.FilaAceptada
		fila.Numero = numero
		fila.Identidad.Nombre, fila.Identidad.PrimerApellido, fila.Identidad.Documento = "Antonio", "Reyes Álvarez", "***4567**"
		lote.Aceptadas = append(lote.Aceptadas, fila)
	}
	return lote, importacionapp.EstadoImportacion{}, true, nil
}

// fuenteVariasBolsasRRHHPrueba compone la fuente real con dobles contados:
// bolsas × participaciones, con o sin la lectura por lotes en los adaptadores.
func fuenteVariasBolsasRRHHPrueba(t *testing.T, bolsas, participaciones int, conLotes bool) (*fuenteConstituidaRRHHDesarrollo, *contadoresBolsasRRHHPrueba) {
	t.Helper()
	c := &contadoresBolsasRRHHPrueba{}
	repo := repositorioVariasBolsasRRHHPrueba{entradas: map[string][]ports.EntradaConstitucion{}}
	for b := 1; b <= bolsas; b++ {
		bolsa := fmt.Sprintf("bolsa:prueba:%02d", b)
		repo.vigentes = append(repo.vigentes, ports.ConstitucionVigente{
			CategoriaRef: fmt.Sprintf("categoria:rpt:prueba-%02d", b),
			Bolsa:        bolsadominio.BolsaConstituida{BolsaRef: bolsa, HuellaListadoSHA256: "huella:" + bolsa},
			Instantanea:  bolsadominio.InstantaneaOrdenBolsa{InstantaneaRef: "instantanea:" + bolsa, Version: 1},
		})
		for p := 1; p <= participaciones; p++ {
			repo.entradas["instantanea:"+bolsa] = append(repo.entradas["instantanea:"+bolsa], ports.EntradaConstitucion{Orden: uint64(p), ParticipacionRef: fmt.Sprintf("participacion:%02d:%03d", b, p), FilaNumero: p + 1})
		}
	}
	orden, err := bolsaapplication.NuevoServicioOrdenVigente(ordenContadoRRHHPrueba{c: c, entradas: repo.entradas})
	if err != nil {
		t.Fatal(err)
	}
	f := &fuenteConstituidaRRHHDesarrollo{
		repositorio: repo, orden: orden, emisiones: emisionesPrueba{}, recuperador: recuperadorContadoRRHHPrueba{c: c},
		ceseActivo: true, ahora: func() time.Time { return time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC) },
	}
	if conLotes {
		f.situaciones, f.estadosCese = situacionesContadasRRHHPrueba{c: c}, cesesContadosRRHHPrueba{c: c}
	} else {
		f.situaciones, f.estadosCese = situacionesSoloIndividualPrueba{situacionesContadasRRHHPrueba{c: c}}, cesesSoloIndividualPrueba{cesesContadosRRHHPrueba{c: c}}
	}
	return f, c
}

// Envolturas que ocultan los métodos por lotes, como un adaptador antiguo.
type situacionesSoloIndividualPrueba struct{ s situacionesContadasRRHHPrueba }

func (s situacionesSoloIndividualPrueba) ParticipacionPerteneceABolsa(ctx context.Context, b, p string) (bool, error) {
	return s.s.ParticipacionPerteneceABolsa(ctx, b, p)
}
func (s situacionesSoloIndividualPrueba) SituacionVigente(ctx context.Context, ref string) (ports.SituacionParticipacion, error) {
	return s.s.SituacionVigente(ctx, ref)
}
func (s situacionesSoloIndividualPrueba) BuscarRegistroSituacion(ctx context.Context, ref, clave string) (ports.RegistroSituacionParticipacion, error) {
	return s.s.BuscarRegistroSituacion(ctx, ref, clave)
}
func (s situacionesSoloIndividualPrueba) RegistrarSituacion(ctx context.Context, c ports.ComandoCambiarSituacionParticipacion) (ports.RegistroSituacionParticipacion, error) {
	return s.s.RegistrarSituacion(ctx, c)
}

type cesesSoloIndividualPrueba struct{ c cesesContadosRRHHPrueba }

func (c cesesSoloIndividualPrueba) ConsultarEstadoCese(ctx context.Context, ref string, corte time.Time) (ports.EstadoCese, bool, error) {
	return c.c.ConsultarEstadoCese(ctx, ref, corte)
}

// La carga de RRHH hace dos lecturas por bolsa (situaciones y ceses), no dos
// por participación, y el resultado es idéntico al de la lectura individual.
func TestFuenteConstituidaRRHHLeeSituacionesYCesesPorLotes(t *testing.T) {
	const bolsas, participaciones = 3, 200
	individual, ci := fuenteVariasBolsasRRHHPrueba(t, bolsas, participaciones, false)
	esperado, err := individual.cargar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ci.situacionIndividual.Load() != bolsas*participaciones || ci.ceseIndividual.Load() != bolsas*participaciones {
		t.Fatalf("la referencia individual no leyó todo: %d/%d", ci.situacionIndividual.Load(), ci.ceseIndividual.Load())
	}
	porLotes, c := fuenteVariasBolsasRRHHPrueba(t, bolsas, participaciones, true)
	obtenido, err := porLotes.cargar(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if c.situacionIndividual.Load() != 0 || c.ceseIndividual.Load() != 0 || c.situacionLote.Load() != bolsas || c.ceseLote.Load() != bolsas {
		t.Fatalf("lecturas: situacion individual=%d lote=%d, cese individual=%d lote=%d; se esperaban 0/%d y 0/%d",
			c.situacionIndividual.Load(), c.situacionLote.Load(), c.ceseIndividual.Load(), c.ceseLote.Load(), bolsas, bolsas)
	}
	if fmt.Sprint(obtenido.Candidaturas) != fmt.Sprint(esperado.Candidaturas) || len(obtenido.Candidaturas) != bolsas*participaciones {
		t.Fatal("la lectura por lotes cambió las candidaturas")
	}
	disponibles := 0
	for _, candidata := range obtenido.Candidaturas {
		if candidata.Estado == "disponible" {
			disponibles++
		}
	}
	if disponibles != bolsas {
		t.Fatalf("la proyección de cese no se aplicó desde el lote: %d disponibles", disponibles)
	}
}
