package bootstrap

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const datasetAvisosViaPrueba = `{
  "bolsas": [
    {"bolsa_ref": "bolsa:antigua", "categoria_ref": "categoria:rpt:auxiliar", "vigente_desde": "2019-05-01T00:00:00Z"},
    {"bolsa_ref": "bolsa:reciente", "categoria_ref": "categoria:rpt:auxiliar", "vigente_desde": "2021-02-01T00:00:00Z"},
    {"bolsa_ref": "bolsa:otra", "categoria_ref": "categoria:rpt:otra", "vigente_desde": "2025-01-01T00:00:00Z"}
  ],
  "candidaturas": [
    {"candidatura_ref": "c1", "bolsa_ref": "bolsa:antigua", "estado_clave": "trabajando"},
    {"candidatura_ref": "c2", "bolsa_ref": "bolsa:reciente", "estado_clave": "excluido"},
    {"candidatura_ref": "c3", "bolsa_ref": "bolsa:reciente", "estado_clave": "disponible_desde", "disponible_desde": "2026-12-01T00:00:00Z"},
    {"candidatura_ref": "c4", "bolsa_ref": "bolsa:otra", "estado_clave": "disponible"}
  ]
}`

type situacionFijaAvisosViaPrueba struct{ situacion ports.SituacionBolsaCobertura }

func (s situacionFijaAvisosViaPrueba) SituacionBolsaCobertura(context.Context, string) (ports.SituacionBolsaCobertura, error) {
	return s.situacion, nil
}

type relojFijoAvisosViaPrueba struct{ instante time.Time }

func (r relojFijoAvisosViaPrueba) Ahora() time.Time { return r.instante }

func TestResumenSituacionBolsaCoberturaSoloCuentaDisponibles(t *testing.T) {
	var datos datasetBolsasRRHHDesarrollo
	if err := json.Unmarshal([]byte(datasetAvisosViaPrueba), &datos); err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	situacion, err := resumirSituacionBolsaCobertura(datos, "categoria:rpt:auxiliar", ahora)
	if err != nil || !situacion.Existe || situacion.BolsaRef != "bolsa:reciente" || situacion.Integrantes != 3 || situacion.Disponibles != 0 ||
		!situacion.ConstituidaEn.Equal(time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("resumen inesperado: %+v %v", situacion, err)
	}
	// La fecha de disponibilidad ya alcanzada cuenta como disponible (art. 9).
	if despues, _ := resumirSituacionBolsaCobertura(datos, "categoria:rpt:auxiliar", time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC)); despues.Disponibles != 1 {
		t.Fatalf("disponibilidad alcanzada no contada: %+v", despues)
	}
	if sinBolsa, err := resumirSituacionBolsaCobertura(datos, "categoria:rpt:inexistente", ahora); err != nil || sinBolsa.Existe {
		t.Fatalf("sin bolsa: %+v %v", sinBolsa, err)
	}
}

// Con el paquete de reglas de ejemplo real: la bolsa sin disponibles se
// propone agotada, con oferta al SAE de nueve meses y nueva convocatoria por
// agotamiento y por superar los cinco años.
func TestAvisosViaCoberturaConReglasDeEjemplo(t *testing.T) {
	reloj := relojReglasEjemploPrueba{ahora: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)}
	compuestas, err := nuevasReglasEjemploDesarrollo(
		configuracionDesarrolloReglasEjemplo(rutaReglasBolsaEjemploPrueba, ""), nil, reloj,
	)
	if err != nil {
		t.Fatal(err)
	}
	evaluador, err := application.NuevoEvaluadorAvisosViaCobertura(situacionFijaAvisosViaPrueba{ports.SituacionBolsaCobertura{
		Existe: true, BolsaRef: "bolsa:reciente", ConstituidaEn: time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC), Integrantes: 3,
	}}, compuestas.bolsa, relojFijoAvisosViaPrueba{reloj.ahora})
	if err != nil {
		t.Fatal(err)
	}
	periodo := domain.PeriodoPrevisto{Inicio: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), Fin: time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)}
	resultado, evaluado := evaluador.Evaluar(t.Context(), "categoria:rpt:auxiliar", periodo)
	if !evaluado || resultado.Estado != application.EstadoAvisosEvaluados || len(resultado.Avisos) != 3 {
		t.Fatalf("resultado inesperado: %+v", resultado)
	}
	agotada, sae, convocatoria := resultado.Avisos[0], resultado.Avisos[1], resultado.Avisos[2]
	if agotada.Clave != application.AvisoBolsaAgotadaProvisionalmente || agotada.Umbral != 0 ||
		agotada.Reglas[0].Articulo != "arts. 3.3 y 7.c" || !agotada.Reglas[0].Ejemplo {
		t.Fatalf("bolsa agotada inesperada: %+v", agotada)
	}
	if sae.DuracionMaximaMeses != 9 || sae.FinMaximo != "2027-07-01" || sae.ExcedeDuracion || sae.Reglas[0].Clave != reglas.BolsaSAEDuracionMaxima {
		t.Fatalf("oferta SAE inesperada: %+v", sae)
	}
	if len(convocatoria.Motivos) != 2 || convocatoria.VigenciaHasta != "2026-02-01" || convocatoria.Reglas[0].Articulo != "arts. 7.b y 3.4" {
		t.Fatalf("nueva convocatoria inesperada: %+v", convocatoria)
	}
	// Sin catálogo o sin fuente no se compone nada y no hay pánico.
	configurarAvisosViaCoberturaDesarrollo(nil, compuestas.bolsa, nil)
	configurarAvisosViaCoberturaDesarrollo(&autoridadConsultasContratacionTemporalDesarrollo{}, nil, nil)
}
