package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// expedientesDerivacionPrueba devuelve un padre fiscalizado v6 de dos plazas,
// su hijo derivado (F1-19) y un expediente ordinario de una plaza.
func expedientesDerivacionPrueba(t *testing.T) (padre, hijo, ordinario domain.Expediente) {
	t.Helper()
	contenido, err := os.ReadFile("../../domain/testdata/expediente_nombramiento_v7.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(contenido, &padre); err != nil {
		t.Fatal(err)
	}
	padre.Actuaciones = padre.Actuaciones[:6]
	padre.Version = 6
	padre.FaseActual = domain.FaseFiscalizacion
	padre.ActualizadoEn = padre.Actuaciones[5].RealizadaEn
	padre.Solicitud.NumeroPersonas = 2
	padre.Solicitud.JornadaMinutos = 420
	padre.Solicitud.PuestoSolicitado = "puesto:administrativo:01"
	if err := padre.Validar(); err != nil {
		t.Fatalf("fixture padre: %v", err)
	}
	preimagen, err := json.Marshal(padre)
	if err != nil {
		t.Fatal(err)
	}
	hijo, err = domain.NuevoExpedienteDerivado(domain.AltaExpedienteDerivado{
		Padre: padre, PreimagenPadreJSON: string(preimagen),
		Referencia: ports.HijoRefDerivacionNombramiento(padre.OrganizacionRef, padre.Referencia, 1),
		OfertaRef:  "oferta:" + strings.Repeat("a", 64), NumeroPlaza: 1,
		ActorRef: "actor:rrhh:sintetico:01", ReciboRef: "recibo:derivacion:sintetica:01",
		CreadaEn: padre.ActualizadoEn.Add(time.Minute),
	})
	if err != nil || hijo.InstruccionCompartida == nil {
		t.Fatalf("fixture hijo: %v", err)
	}
	ordinario = padre.Clonar()
	ordinario.Solicitud.NumeroPersonas = 1
	if err := ordinario.Validar(); err != nil {
		t.Fatalf("fixture ordinario: %v", err)
	}
	return padre, hijo, ordinario
}

func lectorConExpedientePrueba(t *testing.T, e domain.Expediente) (
	*LectorExpedienteSeleccionLlamamientoPostgreSQL, *transaccionConsultaRRHHPrueba,
) {
	t.Helper()
	contenido, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	eventos := []string{}
	fila := &filaConsultaRRHHPrueba{eventos: &eventos, llenar: func(d []any) error {
		*(d[0].(*[]byte)) = append([]byte(nil), contenido...)
		*(d[1].(*int64)) = int64(e.Version)
		return nil
	}}
	tx := &transaccionConsultaRRHHPrueba{fila: fila, eventos: &eventos}
	return &LectorExpedienteSeleccionLlamamientoPostgreSQL{pool: &iniciadorEjecucionSeleccionO6Prueba{tx: tx}}, tx
}

// Ningún caso de uso recibe el agregado del hijo, que lleva dentro el del
// padre: ni la selección ni la recuperación de aviso, respuesta o resolución.
func TestLectoresSeleccionRechazanNombramientoDerivado(t *testing.T) {
	_, hijo, ordinario := expedientesDerivacionPrueba(t)
	for _, c := range []struct {
		nombre  string
		e       domain.Expediente
		cerrado bool
	}{{"hijo", hijo, true}, {"ordinario", ordinario, false}} {
		lector, tx := lectorConExpedientePrueba(t, c.e)
		got, err := lector.LeerExpedienteParaSeleccion(context.Background(), c.e.OrganizacionRef, c.e.Referencia, 6)
		if c.cerrado != errors.Is(err, ports.ErrEstadoExpedienteNoSeleccionable) ||
			(!c.cerrado && (err != nil || got.Fiscalizado.Referencia != c.e.Referencia)) ||
			(c.cerrado && (got.Fiscalizado.Referencia != "" || tx.confirmaciones != 0)) {
			t.Fatalf("selección %s: err=%v ref=%q commits=%d", c.nombre, err, got.Fiscalizado.Referencia, tx.confirmaciones)
		}
		lector, tx = lectorConExpedientePrueba(t, c.e)
		got, err = lector.LeerExpedienteParaAvisoConfirmado(context.Background(), c.e.OrganizacionRef, c.e.Referencia, "llamamiento:ct")
		if c.cerrado != errors.Is(err, ports.ErrIntegracionBolsaNoDisponible) ||
			(!c.cerrado && (err != nil || got.Fiscalizado.Referencia != c.e.Referencia)) ||
			(c.cerrado && (got.Fiscalizado.Referencia != "" || tx.confirmaciones != 0)) {
			t.Fatalf("aviso %s: err=%v ref=%q commits=%d", c.nombre, err, got.Fiscalizado.Referencia, tx.confirmaciones)
		}
	}
}
