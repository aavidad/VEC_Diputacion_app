package postgres

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/dietas/domain"
)

type filaReglaCatalogo struct{ dato []byte }

func (f filaReglaCatalogo) Scan(dest ...any) error {
	*dest[0].(*[]byte) = append([]byte(nil), f.dato...)
	return nil
}

type consultaReglaCatalogo struct {
	dato []byte
	sql  string
	args []any
}

func (c *consultaReglaCatalogo) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	c.sql = sql
	c.args = append([]any(nil), args...)
	return filaReglaCatalogo{c.dato}
}

func reglaCatalogoPrueba() domain.ReglaDevengoProvisional {
	return domain.ReglaDevengoProvisional{ReglaRef: "provisional:regla:nacional-ordinaria:20260923", VersionTarifaRef: "provisional:rd462:20260923", PaisISO2: "ES", Variante: "nacional_ordinaria", HuellaSHA256: strings.Repeat("a", 64), Configuracion: domain.ConfiguracionDevengoProvisional{Regla: "nacional_ordinaria_provisional_v1", Zona: "Europe/Madrid", DuracionMinimaMismoDiaHoras: 5, HoraSalida100AntesDe: 14, HoraSalida50AntesDe: 22, HoraRegreso50DespuesDe: 14, HoraRegresoMismoDiaDespuesDe: 16, DiasMaximos: 31, Alojamiento: "tope_pendiente_justificante", PorcentajeMismoDia: 50, PorcentajeSalidaTemprana: 100, PorcentajeSalidaMedia: 50, PorcentajeRegreso: 50, PorcentajeIntermedio: 100, PorcentajeAlojamientoTope: 100}}
}

func TestConsultarReglaCatalogadaExigeVersionYEsquemaCerrado(t *testing.T) {
	regla := reglaCatalogoPrueba()
	dato, err := json.Marshal(regla)
	if err != nil {
		t.Fatal(err)
	}
	c := &consultaReglaCatalogo{dato: dato}
	repo := &RepositorioTarifasProvisionales{consulta: c}
	fecha := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	leida, err := repo.ConsultarRegla(context.Background(), "", fecha)
	if err != nil || leida.ReglaRef != regla.ReglaRef || len(c.args) != 2 || c.args[0] != "" || c.args[1] != "2026-09-23" || c.sql != consultaReglaDevengoComision {
		t.Fatalf("selección catalogada: %+v %v %#v", leida, err, c.args)
	}
	if _, err = repo.ConsultarRegla(context.Background(), "provisional:rd462:20260924", fecha); err == nil {
		t.Fatal("versión distinta aceptada")
	}
	c.dato = append(dato[:len(dato)-1], []byte(`,"codigo_js":"eval"}`)...)
	if _, err = repo.ConsultarRegla(context.Background(), "", fecha); err == nil {
		t.Fatal("campo ejecutable ajeno al esquema aceptado")
	}
}
