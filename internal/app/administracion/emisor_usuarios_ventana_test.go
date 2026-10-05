package administracion

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type registroConcesionUsuariosPrueba struct{ registradaEn time.Time }

func (r registroConcesionUsuariosPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return r.registradaEn, nil
}

// La decisión V3 en memoria nunca está «vigente» por diseño; la ventana que
// acota la entrega del material es la de la confirmación durable ligada a
// esa misma decisión. El registro de prueba no persiste nada: sólo fabrica la
// confirmación nominal que devolvería el PDP tras su COMMIT.
func TestDecisionUsuariosUsaVentanaDeLaConfirmacionLigada(t *testing.T) {
	e, actor, evidencia, snapshot, entrada := escenarioSolicitudUsuarios(t)
	ctx := context.Background()
	solicitud, resultado, emision, err := e.solicitud(ctx, actor, evidencia, snapshot, entrada)
	if err != nil {
		t.Fatal(err)
	}
	motivo := e.motivos[emision.Audiencia]
	emitida := e.reloj.Ahora().UTC().Truncate(time.Microsecond)
	concesion := func(ref string) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) {
		t.Helper()
		ev, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, snapshot, ref, emitida, emitida.Add(90*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		d, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, ev)
		if err != nil {
			t.Fatal(err)
		}
		if ok, _, err := d.Resultado(); err != nil || !ok {
			t.Fatalf("la instantánea de ensayo no concede: %v", err)
		}
		orden, err := ports.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, d, motivo, resultado)
		if err != nil {
			t.Fatal(err)
		}
		c, err := ports.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, registroConcesionUsuariosPrueba{emitida.Add(time.Second)}, orden)
		if err != nil {
			t.Fatal(err)
		}
		return d, c
	}
	d, c := concesion("dec_" + strings.Repeat("a", 32))
	if err := validarDecisionUsuarios(d, c, solicitud, motivo, resultado, emision.Audiencia, emitida.Add(2*time.Second)); err != nil {
		t.Fatalf("concesión registrada y vigente rechazada: %v", err)
	}
	for nombre, ahora := range map[string]time.Time{
		"antes del registro": emitida.Add(500 * time.Millisecond),
		"fin de la ventana":  emitida.Add(90 * time.Second),
		"caducada":           emitida.Add(time.Hour),
	} {
		if validarDecisionUsuarios(d, c, solicitud, motivo, resultado, emision.Audiencia, ahora) == nil {
			t.Fatalf("%s: concesión fuera de ventana aceptada", nombre)
		}
	}
	_, ajena := concesion("dec_" + strings.Repeat("b", 32))
	if validarDecisionUsuarios(d, ajena, solicitud, motivo, resultado, emision.Audiencia, emitida.Add(2*time.Second)) == nil {
		t.Fatal("confirmación de otra decisión aceptada")
	}
	if validarDecisionUsuarios(d, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, solicitud, motivo, resultado, emision.Audiencia, emitida.Add(2*time.Second)) == nil {
		t.Fatal("decisión sin confirmación durable aceptada")
	}
	otroMotivo := motivo
	otroMotivo.EntradaClave = "motivo_" + strings.Repeat("e", 32)
	if validarDecisionUsuarios(d, c, solicitud, otroMotivo, resultado, emision.Audiencia, emitida.Add(2*time.Second)) == nil {
		t.Fatal("confirmación ligada a otro motivo aceptada")
	}
}
