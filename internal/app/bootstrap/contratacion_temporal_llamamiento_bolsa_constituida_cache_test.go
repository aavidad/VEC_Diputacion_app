package bootstrap

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestOrigenConstituidoUnaLecturaPorPeticion(t *testing.T) {
	d := preparacionLlamamientoDesarrollo{clave: "clave:seleccion"}
	d.expediente.Fiscalizado.OrganizacionRef = "organizacion:prueba"
	d.expediente.Fiscalizado.Referencia = "expediente:prueba"
	d.expediente.Fiscalizado.Version = 6
	estado := &estadoOrigenConstituidoCT{
		organizacion: d.expediente.Fiscalizado.OrganizacionRef,
		expediente:   d.expediente.Fiscalizado.Referencia,
		version:      d.expediente.Fiscalizado.Version, clave: d.clave,
	}
	var consultas atomic.Int32
	consultar := func() (preparacionOrigenConstituidoCT, error) {
		consultas.Add(1)
		return preparacionOrigenConstituidoCT{origen: origenConstituidoVerificadoCT{
			registro:     puertosbolsa.OrigenLlamamientoCT{ActaRef: "acta:prueba"},
			auditoriaRef: "aud_v3_i_12345678901234567890123456789012",
		}}, nil
	}
	const etapas = 6
	var grupo sync.WaitGroup
	grupo.Add(etapas)
	for range etapas {
		go func() {
			defer grupo.Done()
			r, err := estado.cargar(context.Background(), d, consultar)
			if err != nil || r.origen.registro.ActaRef != "acta:prueba" || r.origen.auditoriaRef == "" {
				t.Errorf("etapa sin snapshot original: %v", err)
			}
		}()
	}
	grupo.Wait()
	if consultas.Load() != 1 {
		t.Fatalf("se auditaron %d veces para una petición", consultas.Load())
	}
	d.clave += ":otra"
	if _, err := estado.cargar(context.Background(), d, consultar); err == nil || consultas.Load() != 1 {
		t.Fatal("otra clave reutilizó la lectura de una petición distinta")
	}
}

func TestLecturaOrigenClonadaAntesDeEmitirB7(t *testing.T) {
	orden := uint64(1)
	vigencia := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	original := puertosbolsa.OrigenLlamamientoCT{
		BolsaCanonica: []byte("bolsa"), InstantaneaCanonica: []byte("orden"),
		BolsaVigenteHasta: &vigencia,
		Entradas:          []puertosbolsa.EntradaOrigenLlamamientoCT{{OrdenVigente: &orden}},
	}
	copia := clonarLecturaOrigenCT(original)
	copia.BolsaCanonica[0], copia.InstantaneaCanonica[0] = 'x', 'x'
	*copia.BolsaVigenteHasta, *copia.Entradas[0].OrdenVigente = vigencia.AddDate(1, 0, 0), 9
	if string(original.BolsaCanonica) != "bolsa" || string(original.InstantaneaCanonica) != "orden" ||
		original.BolsaVigenteHasta.Year() != 2027 || *original.Entradas[0].OrdenVigente != 1 {
		t.Fatal("la emisión mutó el snapshot compartido por la petición")
	}
}
