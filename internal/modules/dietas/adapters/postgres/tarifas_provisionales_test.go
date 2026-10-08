package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
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

type filaTarifaCatalogo struct{ datos []any }

func (f filaTarifaCatalogo) Scan(dest ...any) error {
	if len(dest) != len(f.datos) {
		return errors.New("columnas inesperadas")
	}
	for i, valor := range f.datos {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(valor))
	}
	return nil
}

type consultaTarifaCatalogo struct {
	datos []any
	sql   string
	args  []any
}

func (c *consultaTarifaCatalogo) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	c.sql = sql
	c.args = append([]any(nil), args...)
	return filaTarifaCatalogo{c.datos}
}

func TestConsultarTarifaDevuelveFuentesYConservaImportesProvisionales(t *testing.T) {
	version := "provisional:rd462:20260923"
	fecha := time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)
	c := &consultaTarifaCatalogo{datos: []any{version, domain.RotuloTarifaProvisional,
		"BOE-A-2005-19988 / RD 462/2002", "BOE-A-2023-16462 / RD 462/2002",
		"ES", 2, "2026-09-23", "", int64(3740), int64(6597), "0.2600"}}
	repo := &RepositorioTarifasProvisionales{consulta: c}
	leida, err := repo.Consultar(context.Background(), version, 2, "automovil", fecha)
	if err != nil || leida.ReferenciaDietas != c.datos[2] || leida.ReferenciaKilometraje != c.datos[3] ||
		leida.Dieta.ManutencionCentimos != 3740 || leida.Dieta.AlojamientoTopeCentimos != 6597 ||
		leida.EURPorKM != "0.2600" || leida.Dieta.Rotulo != domain.RotuloTarifaProvisional ||
		c.sql != consultaTarifaComisionProvisional || len(c.args) != 4 || c.args[0] != version || c.args[1] != 2 || c.args[2] != "automovil" || c.args[3] != "2026-09-23" {
		t.Fatalf("lectura de tarifa y fuentes: %+v %v %#v", leida, err, c.args)
	}
	for _, valor := range []string{"", "fuente sin formato"} {
		c.datos[2] = valor
		if _, err := repo.Consultar(context.Background(), version, 2, "automovil", fecha); !errors.Is(err, ErrTarifaProvisionalNoDisponible) {
			t.Fatalf("referencia de dietas %q aceptada: %v", valor, err)
		}
	}
	c.datos[2] = "BOE-A-2005-19988 / RD 462/2002"
	c.datos[3] = ""
	if _, err := repo.Consultar(context.Background(), version, 2, "automovil", fecha); !errors.Is(err, ErrTarifaProvisionalNoDisponible) {
		t.Fatalf("referencia de kilometraje ausente aceptada: %v", err)
	}
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
