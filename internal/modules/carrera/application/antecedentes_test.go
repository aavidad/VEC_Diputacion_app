package application

import (
	"context"
	"errors"
	"slices"
	"testing"

	"vec-diputacion-granada/internal/modules/carrera/domain"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

type lectorAntecedentesFunc func(context.Context, ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error)

func (f lectorAntecedentesFunc) ConsultarAntecedentesSinteticos(ctx context.Context, q ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
	return f(ctx, q)
}

func escenarioAntecedentes() (domain.Escenario, ports.InstantaneaAntecedentesSinteticos) {
	f := domain.Fuente{Referencia: "fuente-sintetica", Version: "v2"}
	ev := domain.Evidencia{Referencia: "acto-sintetico", Fuente: f.Referencia, Version: f.Version}
	nivel := 22
	e := domain.Escenario{Alcance: domain.AlcanceSintetico, Version: "instantanea-2", Fuente: f, Casos: []domain.Caso{{Referencia: "caso-1", Via: "grado", PersonaNombre: "Persona sintética"}}}
	a := ports.InstantaneaAntecedentesSinteticos{
		Alcance: domain.AlcanceSintetico, CasoRef: "caso-1", Version: e.Version,
		PersonaRef: "persona-sintetica", EmpleadoRef: "empleado-sintetico", RelacionRef: "relacion-sintetica",
		CorteEfectivo: "2026-10-01", CorteConocimiento: "2026-10-01T10:00:00Z", Cobertura: "parcial",
		Regimen: "funcionario_carrera", GrupoSubgrupo: "A2", Fuentes: []domain.Fuente{f},
		Ocupaciones: []ports.OcupacionAntecedente{{Referencia: "ocupacion-sintetica", Nivel: &nivel, Periodo: domain.Periodo{Inicio: "2021-01-01", Fin: "2026-10-01", Evidencia: ev}}},
		Grado:       &ports.GradoAntecedente{Valor: 20, Evidencia: ev},
		Servicios:   []ports.ServicioAntecedente{{Referencia: "servicio-sintetico", Estado: "reconocido", Periodo: domain.Periodo{Inicio: "2021-01-01", Fin: "2022-01-01", Evidencia: ev}}},
	}
	return e, a
}

func TestAntecedentesConectanPreparacionConProcedenciaSinReconocerDerechos(t *testing.T) {
	e, a := escenarioAntecedentes()
	lector := lectorAntecedentesFunc(func(_ context.Context, q ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
		if q.CasoRef != a.CasoRef || q.VersionEsperada != a.Version {
			t.Fatal("consulta pierde la referencia o versión")
		}
		return a, nil
	})
	out, err := (Servicio{}).PrepararConAntecedentesSinteticos(context.Background(), e, lector)
	if err != nil {
		t.Fatal(err)
	}
	p := out.Preparacion.Casos[0]
	if p.EstadoGlobal != "pendiente" || *p.NivelPuesto != 22 || *p.GradoPersonal != 20 || len(p.Antecedentes.Periodos) != 1 || p.Fuentes[0] != a.Fuentes[0] || out.Casos[0].Instantanea.Cobertura != "parcial" {
		t.Fatal("pierde antecedentes o confunde nivel y grado")
	}
	if !slices.Contains(p.Pendientes, "carrera.pendiente.reconocimiento") || !slices.Contains(out.Casos[0].Faltantes, "autorizacion_carrera_h08") {
		t.Fatal("omite dependencias pendientes")
	}
	*a.Ocupaciones[0].Nivel = 99
	a.Grado.Valor = 99
	a.Fuentes[0].Version = "otra"
	a.Servicios[0].Periodo.Inicio = "otra"
	if *p.NivelPuesto != 22 || *p.GradoPersonal != 20 || out.Casos[0].Instantanea.Grado.Valor != 20 || out.Casos[0].Instantanea.Fuentes[0].Version != "v2" || out.Casos[0].Instantanea.Servicios[0].Periodo.Inicio != "2021-01-01" || e.Casos[0].Regimen != "" {
		t.Fatal("la proyección comparte datos mutables")
	}
}

func TestAntecedentesAusentesDeclaradosOIncompletosQuedanPendientes(t *testing.T) {
	e, a := escenarioAntecedentes()
	a.PersonaRef, a.CorteConocimiento, a.Cobertura = "", "", ""
	a.Grado.Evidencia.Version = "version-ajena"
	a.Ocupaciones = nil
	a.Servicios[0].Estado = "declarado"
	out, err := (Servicio{}).PrepararConAntecedentesSinteticos(context.Background(), e, lectorAntecedentesFunc(func(context.Context, ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
		return a, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	p := out.Preparacion.Casos[0]
	if p.NivelPuesto != nil || p.GradoPersonal != nil || len(p.Antecedentes.Periodos) != 0 || len(out.Casos[0].Instantanea.Servicios) != 1 || out.Casos[0].Instantanea.Servicios[0].Estado != "declarado" {
		t.Fatal("transforma ausencia o declaración en un hecho reconocido")
	}
	for _, campo := range []string{"persona_ref", "corte_conocimiento", "cobertura", "nivel_puesto", "grado_personal", "servicio:servicio-sintetico"} {
		if !slices.Contains(out.Casos[0].Faltantes, campo) {
			t.Fatalf("falta pendiente %s", campo)
		}
	}
}

func TestAntecedentesDenieganFuenteCaidaReferenciaAjenaVersionYAlcance(t *testing.T) {
	for _, tipo := range []string{"caida", "referencia", "version", "alcance", "limite", "estado"} {
		t.Run(tipo, func(t *testing.T) {
			e, a := escenarioAntecedentes()
			var fallo error
			switch tipo {
			case "caida":
				fallo = errors.New("contenido privado")
			case "referencia":
				a.CasoRef = "otro"
			case "version":
				a.Version = "anterior"
			case "alcance":
				a.Alcance = "produccion"
			case "limite":
				a.Servicios = make([]ports.ServicioAntecedente, 129)
			case "estado":
				a.Servicios[0].Estado = "desconocido"
			}
			out, err := (Servicio{}).PrepararConAntecedentesSinteticos(context.Background(), e, lectorAntecedentesFunc(func(context.Context, ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
				return a, fallo
			}))
			if !errors.Is(err, ErrAntecedentes) || len(out.Casos) != 0 || len(out.Preparacion.Casos) != 0 {
				t.Fatal("permite datos no disponibles o salida parcial")
			}
		})
	}
}

func TestAntecedentesNoConsultanSinLectorAlcanceSinteticoOCancelados(t *testing.T) {
	e, _ := escenarioAntecedentes()
	lector := lectorAntecedentesFunc(func(context.Context, ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
		t.Fatal("lectura inesperada")
		return ports.InstantaneaAntecedentesSinteticos{}, nil
	})
	if _, err := (Servicio{}).PrepararConAntecedentesSinteticos(context.Background(), e, nil); !errors.Is(err, ErrAntecedentes) {
		t.Fatal("sin lector no deniega")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (Servicio{}).PrepararConAntecedentesSinteticos(ctx, e, lector); !errors.Is(err, ErrAntecedentes) {
		t.Fatal("consulta cancelada")
	}
	e.Alcance = "produccion"
	if _, err := (Servicio{}).PrepararConAntecedentesSinteticos(context.Background(), e, lector); !errors.Is(err, ErrAntecedentes) {
		t.Fatal("consume alcance productivo")
	}
}

func TestAntecedentesConservanFuentesPropiasYSolapesSinSumarPeriodos(t *testing.T) {
	e, a := escenarioAntecedentes()
	f := domain.Fuente{Referencia: "fuente-politica-sintetica", Version: "v3"}
	e.Casos[0].Fuentes = []domain.Fuente{f}
	e.Casos[0].Politica = domain.Politica{Referencia: "politica-sintetica", Fuente: f.Referencia, Version: f.Version, AprobacionReferencia: "aprobacion-de-ensayo", Regimenes: []string{"funcionario_carrera"}}
	a.Servicios = append(a.Servicios, a.Servicios[0])
	a.Servicios[1].Referencia = "otro-servicio"
	out, err := (Servicio{}).PrepararConAntecedentesSinteticos(context.Background(), e, lectorAntecedentesFunc(func(context.Context, ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
		return a, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	p := out.Preparacion.Casos[0]
	if len(p.Fuentes) != 2 || len(p.Antecedentes.Periodos) != 2 || p.Antecedentes.Politica.Version != "v3" {
		t.Fatal("pierde la fuente propia o combina periodos")
	}
	for _, c := range p.Comprobaciones {
		if c.Clave == "carrera.comprobacion.periodos" && c.MotivoClave == "carrera.motivo.periodos_solapados" {
			return
		}
	}
	t.Fatal("omite el solape en la preparación existente")
}

func TestAntecedentesCorteInvalidoMantieneNivelPendiente(t *testing.T) {
	e, a := escenarioAntecedentes()
	a.CorteEfectivo = "2026-02-30"
	out, err := (Servicio{}).PrepararConAntecedentesSinteticos(context.Background(), e, lectorAntecedentesFunc(func(context.Context, ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
		return a, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Preparacion.Casos[0].NivelPuesto != nil || !slices.Contains(out.Casos[0].Faltantes, "corte_efectivo") || !slices.Contains(out.Casos[0].Faltantes, "nivel_puesto") {
		t.Fatal("una fecha de corte inválida permite consumir el nivel")
	}
}
