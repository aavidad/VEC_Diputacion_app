package domain

import (
	"testing"
	"time"
)

func TestOrdenVigenteExigePoliticaRotuladaYPosicionesUnicas(t *testing.T) {
	uno := uint64(1)
	politica := PoliticaOrdenBolsa{PoliticaRef: "politica:1", BolsaRef: "bolsa:1", Version: 1, Criterio: CriterioPuntuacionDescActa, TipoLista: TipoListaRotatoria, Reposicion: ReposicionMismaPosicion, Rotulo: "Provisional, pendiente de RRHH", Actor: "sistema:migracion", VigenteDesde: time.Now(), Provisional: true}
	orden := OrdenVigenteBolsa{Politica: politica, Posiciones: []PosicionOrdenBolsa{{ParticipacionRef: "participacion:1", OrdenActa: 1, OrdenVigente: &uno, Situacion: "disponible", Razon: "orden_acta"}}}
	if err := orden.Validar(); err != nil {
		t.Fatalf("orden valido rechazado: %v", err)
	}
	orden.Posiciones = append(orden.Posiciones, orden.Posiciones[0])
	if err := orden.Validar(); err == nil {
		t.Fatal("se admitio una participacion duplicada")
	}
}
