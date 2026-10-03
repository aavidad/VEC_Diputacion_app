package application_test

import (
	"context"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func propuestaAportacion(t *testing.T) application.MaterialAportacionAdmision {
	t.Helper()
	s4 := materialAdmision(t)
	a, err := application.IdentificarAntecedenteAdmision(context.Background(), s4)
	if err != nil {
		t.Fatal(err)
	}
	return application.MaterialAportacionAdmision{MaterialS4: s4, Propuesta: domain.PropuestaAportacionAdmision{
		Alcance: domain.AlcanceAdmisionPreparacion, AportacionRef: "aportacion:propuesta", Revision: 1,
		Antecedente: a, RequisitoRef: s4.Requisitos[0].Referencia, RequisitoVersion: s4.Requisitos[0].Version,
		SoportesPropuestos: []domain.SoporteAportacionAdmision{{Hecho: s4.Requisitos[0].HechosEsperados[0], Documentos: []vec.ReferenciaDocumento{{ID: "documento:propuesto", Version: 1}}}},
	}}
}

func TestAportacionConservaCausasPendientesYS3SinCambiarRUM(t *testing.T) {
	m := propuestaAportacion(t)
	anterior, err := application.PrepararAdmision(context.Background(), m.MaterialS4)
	if err != nil {
		t.Fatal(err)
	}
	out, err := application.PrepararAportacionAdmision(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	if out.Resuelto || out.Persistido || out.Presentado || out.RequerimientoEmitido || out.Estado != "pendiente_revision_competente" ||
		!reflect.DeepEqual(out.RequisitoAnterior, anterior.Requisitos[0]) || !reflect.DeepEqual(out.SolicitudContexto, anterior.SolicitudContexto) || out.Propuesta.Antecedente != m.Propuesta.Antecedente {
		t.Fatalf("incorrect boundary: %+v", out)
	}
	if m.MaterialS4.Hechos.Hechos[0].Estado != "declarado" || len(m.MaterialS4.Hechos.Hechos[0].Evidencias) != 0 {
		t.Fatal("modified RUM proposal")
	}
	m.Propuesta.SoportesPropuestos[0].Documentos[0].Version = 2
	m.MaterialS4.SolicitudContexto.Version = "2"
	if out.Propuesta.SoportesPropuestos[0].Documentos[0].Version != 1 || out.SolicitudContexto.Version != "1" {
		t.Fatal("aliased input")
	}
}

func TestAportacionSinSoportesYHechoAusenteSiguenPendientes(t *testing.T) {
	for _, sinSoportes := range []bool{false, true} {
		m := propuestaAportacion(t)
		m.MaterialS4.Hechos = nil // La referencia esperada existe, aunque el hecho no se aportó.
		m.MaterialS4.SolicitudContexto = nil
		var err error
		m.Propuesta.Antecedente, err = application.IdentificarAntecedenteAdmision(context.Background(), m.MaterialS4)
		if err != nil {
			t.Fatal(err)
		}
		if sinSoportes {
			m.Propuesta.SoportesPropuestos = nil
		}
		out, err := application.PrepararAportacionAdmision(context.Background(), m)
		if err != nil || out.Resuelto || out.RequisitoAnterior.Estado != "pendiente" || out.SolicitudContexto != nil || out.Propuesta.SoportesPropuestos == nil {
			t.Fatalf("%+v %v", out, err)
		}
		if sinSoportes && len(out.Pendientes) != 6 {
			t.Fatal("missing incomplete support warning")
		}
	}
}

func TestAportacionInvalidaNoProduceObjeto(t *testing.T) {
	casos := map[string]func(*application.MaterialAportacionAdmision){
		"hash":                func(m *application.MaterialAportacionAdmision) { m.Propuesta.Antecedente.HuellaMaterialSHA256 = "a" },
		"revision":            func(m *application.MaterialAportacionAdmision) { m.Propuesta.Antecedente.Revision++ },
		"serialization":       func(m *application.MaterialAportacionAdmision) { m.Propuesta.Antecedente.EsquemaMaterial = "universal" },
		"changed_material":    func(m *application.MaterialAportacionAdmision) { m.MaterialS4.Requisitos[0].TituloPropuesto += ":" },
		"requirement_version": func(m *application.MaterialAportacionAdmision) { m.Propuesta.RequisitoVersion = "2" },
		"missing_requirement": func(m *application.MaterialAportacionAdmision) { m.Propuesta.RequisitoRef = "requisito:otro" },
		"fact_version":        func(m *application.MaterialAportacionAdmision) { m.Propuesta.SoportesPropuestos[0].Hecho.Version = 2 },
		"fact_absent": func(m *application.MaterialAportacionAdmision) {
			m.Propuesta.SoportesPropuestos[0].Hecho = domain.ReferenciaHechoAdmision{}
		},
		"repeated_fact": func(m *application.MaterialAportacionAdmision) {
			m.Propuesta.SoportesPropuestos = append(m.Propuesta.SoportesPropuestos, m.Propuesta.SoportesPropuestos[0])
		},
		"contradictory_document": func(m *application.MaterialAportacionAdmision) {
			m.Propuesta.SoportesPropuestos[0].Documentos = append(m.Propuesta.SoportesPropuestos[0].Documentos, vec.ReferenciaDocumento{ID: "documento:propuesto", Version: 2})
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			m := propuestaAportacion(t)
			cambiar(&m)
			out, err := application.PrepararAportacionAdmision(context.Background(), m)
			if err == nil || out.Esquema != "" {
				t.Fatalf("%+v %v", out, err)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if out, err := application.PrepararAportacionAdmision(ctx, propuestaAportacion(t)); err == nil || out.Esquema != "" {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestAntecedenteVersionaSerializacionLocalDeListas(t *testing.T) {
	m := materialAdmision(t)
	m.Requisitos[1].HechosEsperados = nil
	nulo, err := application.IdentificarAntecedenteAdmision(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	m.Requisitos[1].HechosEsperados = []domain.ReferenciaHechoAdmision{}
	vacio, err := application.IdentificarAntecedenteAdmision(context.Background(), m)
	if err != nil || nulo.HuellaMaterialSHA256 == vacio.HuellaMaterialSHA256 || vacio.EsquemaMaterial != domain.EsquemaMaterialAdmisionLocal {
		t.Fatalf("%+v %+v %v", nulo, vacio, err)
	}
}
