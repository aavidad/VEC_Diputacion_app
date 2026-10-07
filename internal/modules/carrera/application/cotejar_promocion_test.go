package application_test

import (
	"context"
	"errors"
	"os"
	"testing"
	adapter "vec-diputacion-granada/internal/modules/carrera/adapters/cotejopromocionjson"
	app "vec-diputacion-granada/internal/modules/carrera/application"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

type lectorFunc func(context.Context, ports.ConsultaPromocionSintetica) (ports.DictamenPromocionSintetico, error)

func (f lectorFunc) ConsultarCotejoPromocionSintetico(c context.Context, q ports.ConsultaPromocionSintetica) (ports.DictamenPromocionSintetico, error) {
	return f(c, q)
}
func ejemplo(t *testing.T) adapter.Entrada {
	t.Helper()
	f, err := os.Open("../../../../cmd/vec-carrera-cotejar-promocion/testdata/entrada.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	e, err := adapter.Leer(f)
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func TestCotejoConservaDictamenTresEstadosYPendientesSinAdmitir(t *testing.T) {
	e := ejemplo(t)
	r, err := (app.Servicio{}).CotejarPromocionSintetica(context.Background(), e.Consulta, adapter.Lector{Dictamen: e.Dictamen})
	if err != nil {
		t.Fatal(err)
	}
	if r.EstadoGlobal != "pendiente" || r.BasesVerificadas || len(r.Pendientes) != 4 || len(r.Resultados) != 3 {
		t.Fatal("presenta efecto o pierde circuito pendiente")
	}
	for i, estado := range []string{"cumple", "no_cumple", "pendiente"} {
		if r.Resultados[i].EstadoMostrado != estado {
			t.Fatal("altera dictamen válido")
		}
	}
	e.Consulta.Hechos[0].Version = "otra"
	e.Dictamen.Comprobaciones[0].HechosReferencias[0] = "otro"
	if r.Consulta.Hechos[0].Version != "ensayo-1" || r.Dictamen.Comprobaciones[0].HechosReferencias[0] != "hecho-1" {
		t.Fatal("comparte salida mutable")
	}
}
func TestCotejoGuardaDeclaracionTextoLibreYReglaSinCrearContradiccion(t *testing.T) {
	for _, tipo := range []string{"declaracion", "texto", "regla", "regla_espacios_cumple", "regla_espacios_no_cumple", "sin_hecho"} {
		t.Run(tipo, func(t *testing.T) {
			e := ejemplo(t)
			switch tipo {
			case "declaracion":
				e.Consulta.Hechos[0].EstadoAportado = "declarado"
			case "texto":
				e.Consulta.Requisitos[0].Representacion = "texto_libre"
			case "regla":
				e.Consulta.Requisitos[0].ReglaRef = ""
			case "regla_espacios_cumple", "regla_espacios_no_cumple":
				e.Consulta.Requisitos[0].ReglaRef = "   "
				if tipo == "regla_espacios_no_cumple" {
					e.Dictamen.Comprobaciones[0].EstadoAportado = "no_cumple"
				}
			case "sin_hecho":
				e.Dictamen.Comprobaciones[0].HechosReferencias = nil
			}
			h, _ := app.HuellaConsultaPromocion(e.Consulta)
			e.Dictamen.HuellaConsultaSHA256 = h
			r, err := (app.Servicio{}).CotejarPromocionSintetica(context.Background(), e.Consulta, adapter.Lector{Dictamen: e.Dictamen})
			if err != nil {
				t.Fatal(err)
			}
			if r.Resultados[0].EstadoMostrado != "pendiente" || r.Resultados[0].MotivoGuardaClave == "" || r.Dictamen.Comprobaciones[0].EstadoAportado != e.Dictamen.Comprobaciones[0].EstadoAportado {
				t.Fatal("convierte declaración o pierde estado recibido")
			}
		})
	}
}
func TestCotejoIncongruenteYProveedorCaidoNoDevuelvenListaParcial(t *testing.T) {
	for _, tipo := range []string{"persona", "bases", "hecho", "hito", "omision", "duplicado", "desconocido", "caida", "cancelado", "mutacion"} {
		t.Run(tipo, func(t *testing.T) {
			e := ejemplo(t)
			ctx := context.Background()
			var fallo error
			switch tipo {
			case "persona":
				e.Consulta.PersonaRef = "persona-ajena"
			case "bases":
				e.Consulta.BasesVersion = "otra"
			case "hecho":
				e.Dictamen.Comprobaciones[0].HechosReferencias = []string{"ajeno"}
			case "hito":
				e.Dictamen.Comprobaciones[0].HitoFecha = "2026-10-02"
			case "omision":
				e.Dictamen.Comprobaciones = e.Dictamen.Comprobaciones[:2]
			case "duplicado":
				e.Dictamen.Comprobaciones[1] = e.Dictamen.Comprobaciones[0]
			case "desconocido":
				e.Dictamen.Comprobaciones[0].EstadoAportado = "admitido"
			case "caida":
				fallo = errors.New("dato privado")
			case "cancelado":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			l := lectorFunc(func(_ context.Context, q ports.ConsultaPromocionSintetica) (ports.DictamenPromocionSintetico, error) {
				if tipo == "mutacion" {
					q.Hechos[0].Version = "ajena"
					return ports.DictamenPromocionSintetico{}, fallo
				}
				return *e.Dictamen, fallo
			})
			r, err := (app.Servicio{}).CotejarPromocionSintetica(ctx, e.Consulta, l)
			if !errors.Is(err, app.ErrCotejoPromocionNoDisponible) || len(r.Resultados) != 0 {
				t.Fatal("respuesta parcial o no disponible engañosa")
			}
			if tipo == "mutacion" && e.Consulta.Hechos[0].Version != "ensayo-1" {
				t.Fatal("lector modifica consulta original")
			}
		})
	}
	e := ejemplo(t)
	if _, err := (app.Servicio{}).CotejarPromocionSintetica(context.Background(), e.Consulta, nil); !errors.Is(err, app.ErrCotejoPromocionNoDisponible) {
		t.Fatal("proveedor ausente no falla")
	}
}
