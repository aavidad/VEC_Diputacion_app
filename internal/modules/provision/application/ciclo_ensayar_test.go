package application_test

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

func fixture(t *testing.T) ports.PeticionCiclo {
	t.Helper()
	p, err := simulacion.EjemploCiclo()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCicloRectificaConMotorYConservaProvisional(t *testing.T) {
	p := fixture(t)
	original, _ := json.Marshal(p)
	s, err := application.EnsayarCiclo(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Valoraciones) != 2 {
		t.Fatal("versiones", len(s.Valoraciones))
	}
	v1, v2 := s.Valoraciones[0], s.Valoraciones[1]
	recalculo, err := application.Simular(p.Configuracion, *p.Decisiones[0].EntradaCorregida)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Resultado.HuellaResultado != recalculo.HuellaResultado || v2.HuellaAnterior != v1.HuellaRevision || v2.VersionAnterior != 1 || v2.Version != 2 || v2.HuellaRevision == v1.HuellaRevision {
		t.Fatal("recalculo o genealogia")
	}
	if !v1.Entrada.Cursos[0].Acreditado || v2.Entrada.Cursos[0].Acreditado || v1.Resultado.Total.Micropuntos()-v2.Resultado.Total.Micropuntos() != 480000 {
		t.Fatal("correccion no aplicada por el motor")
	}
	despues, _ := json.Marshal(p)
	if string(original) != string(despues) {
		t.Fatal("entrada mutada")
	}
	repetido, err := application.EnsayarCiclo(p)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(s)
	z, _ := json.Marshal(repetido)
	if string(a) != string(z) {
		t.Fatal("no determinista")
	}
	if s.Resolucion.Estado != "borrador" || s.Resolucion.Firmada || s.Resolucion.Publicada || s.Resolucion.EfectoOficial || len(s.Resolucion.ReclamacionesPendientes) != 0 {
		t.Fatal("efecto oficial inventado")
	}
	// Las colecciones y punteros recibidos/salientes no comparten la historia.
	s.Valoraciones[1].Entrada.Cursos[0].EvidenciaRef = "alterada"
	s.Valoraciones[1].Resultado.Desglose[0].Detalles[0].EvidenciaRef = "alterada"
	*s.Valoraciones[1].Entrada.GradoPersonal = 1
	p.Decisiones[0].EntradaCorregida.Cursos[0].EvidenciaRef = "alterada_fuera"
	p.Configuracion.Reglas[0].Tramos[0].ID = "alterada_fuera"
	if v1.Entrada.Cursos[0].EvidenciaRef == "alterada" || *v1.Entrada.GradoPersonal != 24 || s.Decisiones[0].EntradaCorregida.Cursos[0].EvidenciaRef == "alterada_fuera" || s.Configuracion.Reglas[0].Tramos[0].ID == "alterada_fuera" {
		t.Fatal("historia comparte datos mutables")
	}
}

func TestMantenerAnadeRevisionMotivada(t *testing.T) {
	p := fixture(t)
	p.Decisiones[0].Tipo, p.Decisiones[0].EntradaCorregida = domain.MantenerValoracion, nil
	s, err := application.EnsayarCiclo(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.Valoraciones[0].Resultado.HuellaResultado != s.Valoraciones[1].Resultado.HuellaResultado || s.Valoraciones[0].HuellaRevision == s.Valoraciones[1].HuellaRevision {
		t.Fatal("mantener debe conservar cálculo y añadir revisión")
	}
	p.Decisiones[0].MotivacionRef = "motivacion:sintetica:otra"
	z, err := application.EnsayarCiclo(p)
	if err != nil {
		t.Fatal(err)
	}
	if z.Valoraciones[1].HuellaRevision == s.Valoraciones[1].HuellaRevision || z.Valoraciones[1].HuellaDecision == s.Valoraciones[1].HuellaDecision {
		t.Fatal("motivación fuera de huella")
	}
}

func TestCicloRechazaVersionCausaYDecisionIncoherentes(t *testing.T) {
	casos := []struct {
		nombre, codigo string
		cambiar        func(*ports.PeticionCiclo)
	}{
		{"version_cero", "valoracion_reclamada_no_coincide", func(p *ports.PeticionCiclo) { p.Reclamaciones[0].VersionValoracion = 0 }},
		{"huella", "valoracion_reclamada_no_coincide", func(p *ports.PeticionCiclo) {
			p.Reclamaciones[0].HuellaValoracion = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		}},
		{"causa", "causa_no_catalogada", func(p *ports.PeticionCiclo) { p.Reclamaciones[0].CausaCodigo = "inventada" }},
		{"catalogo", "version_catalogo_no_coincide", func(p *ports.PeticionCiclo) { p.Reclamaciones[0].VersionCatalogo = "otra" }},
		{"version_obsoleta", "version_esperada_no_coincide", func(p *ports.PeticionCiclo) { p.Decisiones[0].VersionEsperada = 2 }},
		{"estado_libre", "tipo_decision_invalido", func(p *ports.PeticionCiclo) { p.Decisiones[0].Tipo = "admitida" }},
		{"motivacion", "decision_invalida", func(p *ports.PeticionCiclo) { p.Decisiones[0].MotivacionRef = "" }},
		{"sin_correccion", "rectificacion_sin_instantanea", func(p *ports.PeticionCiclo) { p.Decisiones[0].EntradaCorregida = nil }},
		{"sobrescribir", "rectificacion_incompatible", func(p *ports.PeticionCiclo) {
			p.Decisiones[0].EntradaCorregida.InstantaneaRef = p.Entrada.InstantaneaRef
		}},
		{"otro_puesto", "rectificacion_incompatible", func(p *ports.PeticionCiclo) { p.Decisiones[0].EntradaCorregida.PuestoRef = "puesto:ajeno" }},
		{"mantener_cambia", "decision_incompatible", func(p *ports.PeticionCiclo) { p.Decisiones[0].Tipo = domain.MantenerValoracion }},
		{"claim_duplicada", "reclamacion_duplicada", func(p *ports.PeticionCiclo) { p.Reclamaciones = append(p.Reclamaciones, p.Reclamaciones[0]) }},
		{"decision_duplicada", "decision_sin_reclamacion_unica", func(p *ports.PeticionCiclo) {
			r := p.Reclamaciones[0]
			r.Referencia = "reclamacion:segunda"
			p.Reclamaciones = append(p.Reclamaciones, r)
			p.Decisiones = append(p.Decisiones, p.Decisiones[0])
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := fixture(t)
			c.cambiar(&p)
			_, err := application.EnsayarCiclo(p)
			var nominal *domain.Error
			if !errors.As(err, &nominal) || nominal.Codigo != c.codigo {
				t.Fatalf("%v esperado %s", err, c.codigo)
			}
		})
	}
}

func TestBorradorSenalaFuenteRevisionYReclamacionPendientes(t *testing.T) {
	p := fixture(t)
	p.Decisiones = []domain.DecisionRevision{}
	p.RevisionInicialRef = ""
	p.Entrada.GradoPersonal = nil
	p.Entrada.GradoEvidenciaRef = ""
	inicial := p
	inicial.Reclamaciones = []domain.Reclamacion{}
	i, err := application.EnsayarCiclo(inicial)
	if err != nil {
		t.Fatal(err)
	}
	p.Reclamaciones[0].HuellaValoracion = i.Valoraciones[0].HuellaRevision
	s, err := application.EnsayarCiclo(p)
	if err != nil {
		t.Fatal(err)
	}
	if s.Valoraciones[0].Resultado.Total != nil || s.Valoraciones[0].Resultado.Completo {
		t.Fatal("fuente desconocida convertida en cero")
	}
	for _, pendiente := range []string{"revision_inicial_pendiente", "fuentes_pendientes", "reclamaciones_pendientes"} {
		if !slices.Contains(s.Resolucion.Pendientes, pendiente) {
			t.Fatal("pendiente ausente", pendiente)
		}
	}
}

func TestDosDecisionesConVersionOptimista(t *testing.T) {
	p := fixture(t)
	r := p.Reclamaciones[0]
	r.Referencia = "reclamacion:sintetica:segunda"
	p.Reclamaciones = append(p.Reclamaciones, r)
	p.Decisiones = append(p.Decisiones, domain.DecisionRevision{Referencia: "decision:sintetica:segunda", ReclamacionRef: r.Referencia, VersionEsperada: 2, Tipo: domain.MantenerValoracion, MotivacionRef: "motivacion:sintetica:segunda", EvidenciaRef: "evidencia:sintetica:segunda"})
	s, err := application.EnsayarCiclo(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Valoraciones) != 3 || s.Valoraciones[2].HuellaAnterior != s.Valoraciones[1].HuellaRevision {
		t.Fatal("cadena rota")
	}
	p.Decisiones[1].VersionEsperada = 1
	if _, err = application.EnsayarCiclo(p); err == nil {
		t.Fatal("acepta versión obsoleta")
	}
}
