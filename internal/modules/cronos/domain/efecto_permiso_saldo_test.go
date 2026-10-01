package domain_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func cargarEfectos(t *testing.T) (ports.SnapshotEnsayoSaldoPermisos, domain.PoliticaEfectosPermisoSaldo) {
	t.Helper()
	b, err := os.ReadFile("../../../../data/demo/cronos/escenarios-saldo.json")
	if err != nil {
		t.Fatal(err)
	}
	var s ports.SnapshotEnsayoSaldoPermisos
	if err = json.Unmarshal(b, &s); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile("../../../../data/demo/reglas/cronos-efectos-permisos.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := catalogoefectos.Cargar(b, s.PoliticaSHA256)
	if err != nil {
		t.Fatal(err)
	}
	return s, c.Politica
}

func TestEfectoPermisoUsaProgramacionSinReescribirTrabajo(t *testing.T) {
	s, politica := cargarEfectos(t)
	causas := []string{"permiso_computado", "permiso_computado", "permiso_computado", "permiso_parcial", "trabajo_concurrente", "permisos_coincidentes", "programacion_ausente", "hechos_incompletos", "hechos_incompletos", "regla_no_disponible", "regla_no_disponible", "concesion_no_acreditada", "sin_permisos"}
	if len(causas) != len(s.Escenarios) {
		t.Fatal("escenarios sin cubrir")
	}
	for i, e := range s.Escenarios {
		t.Run(e.Referencia, func(t *testing.T) {
			d := hechoDominio(e.Dias[0])
			r, err := domain.CalcularEfectoPermisoSaldo(d, politica)
			if err != nil || r.Causa != causas[i] {
				t.Fatalf("causa %q, error %v", r.Causa, err)
			}
			if *r.TrabajadosMinutos != *d.TrabajadosMinutos {
				t.Fatal("trabajo reescrito")
			}
			if i < 3 {
				if r.PermisoComputadoMinutos == nil || *r.PermisoComputadoMinutos != *d.Programacion.MinutosPrevistos || r.SaldoAjustadoMinutos == nil || *r.SaldoAjustadoMinutos != *d.TrabajadosMinutos || r.ReglaRef == "" || r.ResolucionRef != d.Permisos[0].ResolucionRef || r.VersionProgramacionRef != d.Programacion.PoliticaVersionRef || r.FuenteProgramacionRef != d.Programacion.FuenteRef {
					t.Fatal("crédito sin trazabilidad")
				}
			} else if r.Causa != "sin_permisos" && (r.PermisoComputadoMinutos != nil || r.SaldoAjustadoMinutos != nil) {
				t.Fatal("ajuste inventado")
			}
		})
	}
	d := hechoDominio(s.Escenarios[0].Dias[0])
	p := politica
	nuevo := *d.Programacion.MinutosPrevistos + 1
	d.Programacion = &domain.ProgramacionDiaPermisoSaldo{Fecha: d.Fecha, Referencia: d.Programacion.Referencia, PoliticaVersionRef: d.Programacion.PoliticaVersionRef, FuenteRef: d.Programacion.FuenteRef, Acreditada: true, MinutosPrevistos: &nuevo}
	r, err := domain.CalcularEfectoPermisoSaldo(d, p)
	if err != nil || *r.PermisoComputadoMinutos != nuevo {
		t.Fatal("cuantía fija")
	}
	d.Programacion.MinutosPrevistos = nil
	r, err = domain.CalcularEfectoPermisoSaldo(d, p)
	if err != nil || r.Causa != "programacion_ausente" || r.SaldoAjustadoMinutos != nil {
		t.Fatal("programación incompleta tratada como cero")
	}
	p.Reglas = []domain.ReglaEfectoPermisoSaldo{}
	d = hechoDominio(s.Escenarios[0].Dias[0])
	r, err = domain.CalcularEfectoPermisoSaldo(d, p)
	if err != nil || r.Causa != "regla_no_disponible" || r.SaldoAjustadoMinutos != nil {
		t.Fatal("regla retirada aplicada")
	}
	p = politica
	p.Reglas = append([]domain.ReglaEfectoPermisoSaldo{p.Reglas[0]}, p.Reglas...)
	if p.Validar() == nil {
		t.Fatal("selección ambigua admitida")
	}
	p = politica
	p.Reglas = append([]domain.ReglaEfectoPermisoSaldo{}, p.Reglas...)
	p.Reglas[0].ColectivoRef = "*"
	if p.Validar() == nil {
		t.Fatal("comodín admitido")
	}
}

func TestEfectoExigeVigenciaTrabajoExactoYConcesionCompleta(t *testing.T) {
	s, p := cargarEfectos(t)
	inicio, err := time.Parse("2006-01-02", p.Reglas[0].VigenteDesde)
	if err != nil {
		t.Fatal(err)
	}
	fin, err := time.Parse("2006-01-02", p.Reglas[0].HastaExclusivo)
	if err != nil {
		t.Fatal(err)
	}
	for fecha, causa := range map[string]string{inicio.AddDate(0, 0, -1).Format("2006-01-02"): "fuera_vigencia", inicio.Format("2006-01-02"): "permiso_computado", fin.AddDate(0, 0, -1).Format("2006-01-02"): "permiso_computado", fin.Format("2006-01-02"): "fuera_vigencia", "1990-01-01": "fuera_vigencia"} {
		d := hechoDominio(s.Escenarios[0].Dias[0])
		j := *d.Programacion
		d.Programacion = &j
		d.Fecha = fecha
		d.Programacion.Fecha = fecha
		d.Permisos[0].Desde = fecha
		d.Permisos[0].Hasta = fecha
		r, err := domain.CalcularEfectoPermisoSaldo(d, p)
		if err != nil || r.Causa != causa || (causa == "fuera_vigencia" && (r.SaldoAjustadoMinutos != nil || r.PermisoComputadoMinutos != nil)) {
			t.Fatalf("fecha %s: %s %v", fecha, r.Causa, err)
		}
	}
	for nombre, mutar := range map[string]func(*domain.DiaSaldoPermisos){
		"treinta_segundos_redondeados": func(d *domain.DiaSaldoPermisos) { hay := false; d.SinTrabajo = &hay },
		"ausencia_no_acreditada":       func(d *domain.DiaSaldoPermisos) { d.SinTrabajo = nil },
		"catalogo_original_ausente":    func(d *domain.DiaSaldoPermisos) { d.Permisos[0].CatalogoVersionRef = "" },
	} {
		t.Run(nombre, func(t *testing.T) {
			d := hechoDominio(s.Escenarios[0].Dias[0])
			mutar(&d)
			r, err := domain.CalcularEfectoPermisoSaldo(d, p)
			if err != nil || r.PermisoComputadoMinutos != nil || r.SaldoAjustadoMinutos != nil {
				t.Fatal("crédito sin hechos suficientes")
			}
			esperada := map[string]string{"treinta_segundos_redondeados": "trabajo_concurrente", "ausencia_no_acreditada": "hechos_incompletos", "catalogo_original_ausente": "concesion_incompleta"}[nombre]
			if r.Causa != esperada {
				t.Fatalf("causa %q", r.Causa)
			}
		})
	}
}

func hechoDominio(d ports.DiaSaldoPermisos) domain.DiaSaldoPermisos {
	r := domain.DiaSaldoPermisos{Fecha: d.Fecha, Programacion: (*domain.ProgramacionDiaPermisoSaldo)(d.Programacion), TrabajadosMinutos: d.TrabajadosMinutos, Completo: d.Completo, SinAnomalias: d.SinAnomalias, SinTrabajo: d.SinTrabajo}
	if d.Permisos != nil {
		r.Permisos = make([]domain.ConcesionPermisoSaldo, len(d.Permisos))
		for i, c := range d.Permisos {
			r.Permisos[i] = domain.ConcesionPermisoSaldo(c)
		}
	}
	return r
}
