package application

import (
	"context"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

type consultaOrdenPrueba struct {
	orden dominiobolsa.OrdenVigenteBolsa
}

func (c consultaOrdenPrueba) ConsultarOrdenVigente(context.Context, string) (dominiobolsa.OrdenVigenteBolsa, error) {
	return c.orden, nil
}

func TestServicioOrdenVigenteValidaLaFuente(t *testing.T) {
	uno := uint64(1)
	orden := dominiobolsa.OrdenVigenteBolsa{Politica: dominiobolsa.PoliticaOrdenBolsa{PoliticaRef: "politica:1", BolsaRef: "bolsa:1", Version: 1, Criterio: dominiobolsa.CriterioPuntuacionDescActa, TipoLista: dominiobolsa.TipoListaRotatoria, Reposicion: dominiobolsa.ReposicionMismaPosicion, Rotulo: "Provisional, pendiente de RRHH", Actor: "sistema:migracion", VigenteDesde: time.Now(), Provisional: true}, Posiciones: []dominiobolsa.PosicionOrdenBolsa{{ParticipacionRef: "participacion:1", OrdenActa: 1, OrdenVigente: &uno, Situacion: "disponible", Razon: "orden_acta"}}}
	servicio, err := NuevoServicioOrdenVigente(consultaOrdenPrueba{orden})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Consultar(context.Background(), "bolsa:1"); err != nil {
		t.Fatal(err)
	}
}
