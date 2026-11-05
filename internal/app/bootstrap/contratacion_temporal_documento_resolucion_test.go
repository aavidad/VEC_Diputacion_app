package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type actoResolucionDocumentoPrueba struct {
	r   ports.ResultadoResolucionFormalizacion
	err error
	exp string
}

func (a *actoResolucionDocumentoPrueba) ConsultarActoResolucionFormalizacion(_ context.Context, expediente string) (ports.ResultadoResolucionFormalizacion, error) {
	a.exp = expediente
	return a.r, a.err
}

// Los documentos leen el acto con la consulta del acto (CT261), no con la
// preparación limitada a v7/v8; sin ese lector la fuente falla cerrada.
func TestFuenteResolucionDocumentoCTUsaLaConsultaDelActo(t *testing.T) {
	if _, err := fuenteResolucionDocumentoCT(nil)(context.Background(), "expediente:prueba"); !errors.Is(err, ports.ErrResolucionFormalizacionNoDisponible) {
		t.Fatal("fuente sin lector admitida", err)
	}
	soloPreparacion := struct {
		ports.ConsultorPreparacionResolucionFormalizacion
	}{}
	if _, err := fuenteResolucionDocumentoCT(soloPreparacion)(context.Background(), "expediente:prueba"); !errors.Is(err, ports.ErrResolucionFormalizacionNoDisponible) {
		t.Fatal("la preparación v7/v8 sustituye a la consulta del acto", err)
	}
	acto := &actoResolucionDocumentoPrueba{r: ports.ResultadoResolucionFormalizacion{ReciboRef: "recibo:prueba"}}
	r, err := fuenteResolucionDocumentoCT(acto)(context.Background(), "expediente:prueba")
	if err != nil || r == nil || r.ReciboRef != "recibo:prueba" || acto.exp != "expediente:prueba" {
		t.Fatal("acto no recuperado", err)
	}
	acto.err = ports.ErrResultadoResolucionFormalizacionNoConfiable
	if r, err := fuenteResolucionDocumentoCT(acto)(context.Background(), "expediente:prueba"); r != nil || !errors.Is(err, ports.ErrResultadoResolucionFormalizacionNoConfiable) {
		t.Fatal("rechazo del lector ocultado", err)
	}
}
