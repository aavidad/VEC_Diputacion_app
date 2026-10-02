package preparacionbases

import (
	"strings"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestMaterialIncompletoConservaPendientesSinGobierno(t *testing.T) {
	m := Material{Contenido: bolsa.ContenidoPublicableConvocatoria{Titulo: "Propuesta sintética incompleta"}}
	canonico, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	pendientes, err := canonico.Pendientes()
	if err != nil || len(pendientes) != 15 {
		t.Fatalf("pendientes=%+v error=%v", pendientes, err)
	}
	if canonico.Contenido.Validar() == nil {
		t.Fatal("material incompleto tratado como publicable")
	}
	if (bolsa.ConfiguracionFijadaConvocatoria{}).ValidarPara(canonico.Contenido) == nil {
		t.Fatal("se ha rebajado el circuito formal")
	}
}

func TestHuellaCanonicaFijaReferenciasSinModificarPropuesta(t *testing.T) {
	m := Material{Referencias: []ReferenciaPropuesta{{Campo: "oep"}, {Campo: "plaza"}}, Contenido: bolsa.ContenidoPublicableConvocatoria{Categorias: []string{"auxiliar"}}}
	c, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	c.Contenido.Categorias[0] = "otra"
	if m.Contenido.Categorias[0] != "auxiliar" {
		t.Fatal("copia compartida")
	}
	h1, _ := m.HuellaSHA256()
	m.Referencias[0], m.Referencias[1] = m.Referencias[1], m.Referencias[0]
	h2, _ := m.HuellaSHA256()
	if h1 != h2 {
		t.Fatal("orden de referencias altera huella")
	}
	m.Contenido.Titulo = "Corrección sintética"
	h3, _ := m.HuellaSHA256()
	if h3 == h2 {
		t.Fatal("correccion conserva huella anterior")
	}
}

func TestMaterialLimitaTamanoYCamposAntesDeCopiar(t *testing.T) {
	for _, m := range []Material{
		{Referencias: []ReferenciaPropuesta{{Campo: "permiso"}}},
		{Referencias: []ReferenciaPropuesta{{Campo: "oep"}, {Campo: "oep"}}},
		{Contenido: bolsa.ContenidoPublicableConvocatoria{Titulo: strings.Repeat("x", 12001)}},
		{Contenido: bolsa.ContenidoPublicableConvocatoria{Categorias: make([]string, 101)}},
		{Contenido: bolsa.ContenidoPublicableConvocatoria{Titulo: "texto\x00"}},
		{Contenido: bolsa.ContenidoPublicableConvocatoria{Titulo: string([]byte{255})}},
	} {
		if _, err := m.Canonico(); err == nil {
			t.Fatal("material desproporcionado o ajeno admitido")
		}
	}
}

func TestCASExigeRevisionYHuellaAnteriores(t *testing.T) {
	for _, e := range []Esperada{{PreparacionRef: "prep:sintetica", Revision: 1}, {PreparacionRef: "prep:sintetica", HuellaMaterialSHA256: strings.Repeat("a", 64)}, {PreparacionRef: "prep:sintetica", Revision: -1}} {
		if e.Validar(true) == nil {
			t.Fatal("CAS incompleto admitido")
		}
	}
	e := Esperada{PreparacionRef: "prep:sintetica"}
	ambito, _ := bolsa.NuevoAmbitoOrganizativoConvocatoria("org_diputaciongranada", "uni_seleccionexterna")
	h1, _ := HuellaIntencion(e, Material{}, ambito)
	e.Revision = 1
	e.HuellaMaterialSHA256 = strings.Repeat("a", 64)
	h2, _ := HuellaIntencion(e, Material{}, ambito)
	if h1 == h2 {
		t.Fatal("preimagen no ligada a intencion")
	}
}
