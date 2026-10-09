package bootstrap

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type consultaSituacionBolsaCoberturaPrueba struct {
	situacion ports.SituacionBolsaCobertura
	err       error
	lecturas  atomic.Int32
}

func (c *consultaSituacionBolsaCoberturaPrueba) SituacionBolsaCobertura(
	context.Context, string,
) (ports.SituacionBolsaCobertura, error) {
	c.lecturas.Add(1)
	return c.situacion, c.err
}

// La vía de bolsa recibe sus dos datos de la situación real de Bolsa; antes
// llegaban siempre «no consta» y la propuesta quedaba incompleta para
// cualquier categoría, hubiera o no bolsa con personas disponibles.
func TestFuenteCoberturaDesarrolloRespondeViaBolsaConSituacionDeBolsa(t *testing.T) {
	t.Parallel()
	periodo := domain.PeriodoPrevisto{
		Inicio: time.Date(2027, 2, 4, 0, 0, 0, 0, time.UTC),
		Fin:    time.Date(2027, 5, 5, 0, 0, 0, 0, time.UTC),
	}
	constituida := time.Date(2026, 9, 18, 0, 43, 27, 0, time.UTC)
	casos := []struct {
		nombre      string
		situacion   ports.SituacionBolsaCobertura
		err         error
		existe      domain.ResultadoComprobacion
		disponibles domain.ResultadoComprobacion
	}{
		{"con_disponibles", ports.SituacionBolsaCobertura{Existe: true, BolsaRef: "bolsa:x", ConstituidaEn: constituida, Integrantes: 42, Disponibles: 27},
			nil, domain.ComprobacionAfirmativa, domain.ComprobacionAfirmativa},
		{"agotada", ports.SituacionBolsaCobertura{Existe: true, BolsaRef: "bolsa:x", ConstituidaEn: constituida, Integrantes: 5},
			nil, domain.ComprobacionAfirmativa, domain.ComprobacionNegativa},
		{"bolsa_no_responde", ports.SituacionBolsaCobertura{}, ports.ErrSituacionBolsaCoberturaNoDisponible,
			domain.ComprobacionNoConsta, domain.ComprobacionNoConsta},
		{"resumen_incoherente", ports.SituacionBolsaCobertura{Existe: true, BolsaRef: "bolsa:x", ConstituidaEn: constituida, Integrantes: 1, Disponibles: 2},
			nil, domain.ComprobacionNoConsta, domain.ComprobacionNoConsta},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			consulta := &consultaSituacionBolsaCoberturaPrueba{situacion: caso.situacion, err: caso.err}
			enlace := &situacionBolsaCoberturaFijable{}
			enlace.fijar(consulta)
			fuente := &fuenteComprobacionCoberturaDesarrollo{bolsa: enlace}
			pedir := func(via, comprobacion, procedencia domain.ClaveCatalogo) domain.ResultadoComprobacion {
				t.Helper()
				resultado, existe := fuente.resultadoPara(context.Background(), "categoria:desarrollo:a2", periodo, via, comprobacion, procedencia)
				if !existe {
					t.Fatalf("sin respuesta para %s/%s", via, comprobacion)
				}
				return resultado
			}
			if r := pedir("bolsa_vigente", "existe_bolsa_vigente", "bolsa"); r != caso.existe {
				t.Fatalf("existe_bolsa_vigente=%q, se esperaba %q", r, caso.existe)
			}
			if r := pedir("bolsa_vigente", "hay_candidaturas_disponibles", "bolsa"); r != caso.disponibles {
				t.Fatalf("hay_candidaturas_disponibles=%q, se esperaba %q", r, caso.disponibles)
			}
			if r := pedir("nueva_convocatoria_bolsa", "requiere_nueva_convocatoria", "bolsa"); r != domain.ComprobacionNoConsta {
				t.Fatalf("una comprobación sin fuente no debe responderse: %q", r)
			}
			if r := pedir("oferta_sae", "oferta_sae_disponible", "sae"); r != domain.ComprobacionNoConsta {
				t.Fatalf("SAE no tiene fuente: %q", r)
			}
			if caso.err == nil && caso.situacion.Validar() == nil && consulta.lecturas.Load() != 1 {
				t.Fatalf("Bolsa debe leerse una vez por propuesta: %d lecturas", consulta.lecturas.Load())
			}
		})
	}
}

func TestSituacionBolsaCoberturaFijableCaducaYNoSeReemplaza(t *testing.T) {
	t.Parallel()
	ahora := time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)
	primera := &consultaSituacionBolsaCoberturaPrueba{situacion: ports.SituacionBolsaCobertura{
		Existe: true, BolsaRef: "bolsa:x", ConstituidaEn: ahora.Add(-time.Hour), Integrantes: 3, Disponibles: 1,
	}}
	segunda := &consultaSituacionBolsaCoberturaPrueba{err: errors.New("no debe usarse")}
	enlace := &situacionBolsaCoberturaFijable{ahora: func() time.Time { return ahora }}
	var sinEnlace *situacionBolsaCoberturaFijable
	if _, ok := sinEnlace.situacion(context.Background(), "categoria:x"); ok {
		t.Fatal("sin enlace no hay situación")
	}
	if _, ok := enlace.situacion(context.Background(), "categoria:x"); ok {
		t.Fatal("antes de fijar no hay situación")
	}
	enlace.fijar(primera)
	enlace.fijar(segunda)
	for range 2 {
		if _, ok := enlace.situacion(context.Background(), "categoria:x"); !ok {
			t.Fatal("situación no leída")
		}
	}
	if primera.lecturas.Load() != 1 || segunda.lecturas.Load() != 0 {
		t.Fatalf("lecturas inesperadas: %d/%d", primera.lecturas.Load(), segunda.lecturas.Load())
	}
	ahora = ahora.Add(validezSituacionBolsaCoberturaCT)
	if _, ok := enlace.situacion(context.Background(), "categoria:x"); !ok || primera.lecturas.Load() != 2 {
		t.Fatalf("la memoria debe caducar: %d lecturas", primera.lecturas.Load())
	}
}

// Una propuesta pregunta a Bolsa por la vía de bolsa (dos comprobaciones) y
// por los avisos de la vía: las tres preguntas se sirven con una sola lectura.
func TestPropuestaCoberturaLeeBolsaUnaSolaVez(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	compuestas, err := nuevasReglasEjemploDesarrollo(
		configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, relojReglasEjemploPrueba{ahora: ahora},
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta := &consultaSituacionBolsaCoberturaPrueba{situacion: ports.SituacionBolsaCobertura{
		Existe: true, BolsaRef: "bolsa:x", ConstituidaEn: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), Integrantes: 42, Disponibles: 27,
	}}
	enlace := &situacionBolsaCoberturaFijable{ahora: func() time.Time { return ahora }}
	enlace.fijar(consulta)
	evaluador, err := application.NuevoEvaluadorAvisosViaCobertura(enlace, compuestas.bolsa, relojFijoAvisosViaPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuenteComprobacionCoberturaDesarrollo{bolsa: enlace}
	periodo := domain.PeriodoPrevisto{Inicio: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Fin: time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)}
	categoria := "categoria:desarrollo:a2"
	for _, comprobacion := range []domain.ClaveCatalogo{"existe_bolsa_vigente", "hay_candidaturas_disponibles"} {
		if r, ok := fuente.resultadoPara(context.Background(), categoria, periodo, "bolsa_vigente", comprobacion, "bolsa"); !ok || r != domain.ComprobacionAfirmativa {
			t.Fatalf("%s=%q", comprobacion, r)
		}
	}
	if resultado, evaluado := evaluador.Evaluar(t.Context(), categoria, periodo); !evaluado || resultado.Estado != application.EstadoAvisosEvaluados {
		t.Fatalf("avisos inesperados: %+v", resultado)
	}
	if n := consulta.lecturas.Load(); n != 1 {
		t.Fatalf("Bolsa leída %d veces en una propuesta; debe ser una", n)
	}
}
