package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestErrorEmisionFueraDeOrdenLeeLaPrimeraSaltada(t *testing.T) {
	err := errorEmision(&pgconn.PgError{Code: "VBL02", Detail: `{"orden_vigente": 2, "participacion_ref": "participacion:2"}`})
	var fuera *ports.LlamamientoFueraDeOrden
	if !errors.As(err, &fuera) || !errors.Is(err, ports.ErrLlamamientoFueraDeOrden) || fuera.ParticipacionRef != "participacion:2" || fuera.OrdenVigente != 2 {
		t.Fatalf("rechazo por orden mal traducido: %#v", err)
	}
	for _, detalle := range []string{"", "{", `{"orden_vigente":0,"participacion_ref":"p"}`, `{"orden_vigente":1}`} {
		err = errorEmision(&pgconn.PgError{Code: "VBL02", Detail: detalle})
		if !errors.As(err, &fuera) || fuera.ParticipacionRef != "" || fuera.OrdenVigente != 0 {
			t.Fatalf("detalle %q no debe identificar a nadie: %#v", detalle, err)
		}
	}
}
