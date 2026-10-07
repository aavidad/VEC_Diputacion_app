package preparacionbases

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestMaterialCanonicoConservaHuellaYRechazaOtraRepresentacion(t *testing.T) {
	m := Material{Contenido: bolsa.ContenidoPublicableConvocatoria{Titulo: "Propuesta sintética"}}
	b, err := m.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(b)
	huella, _ := m.HuellaSHA256()
	if hex.EncodeToString(suma[:]) != huella {
		t.Fatal("persistencia altera preimagen de huella")
	}
	recuperado, err := DecodificarMaterialCanonico(b)
	if err != nil {
		t.Fatal(err)
	}
	recodificado, _ := recuperado.RepresentacionCanonica()
	if !bytes.Equal(b, recodificado) {
		t.Fatal("recuperacion cambia bytes")
	}
	for _, entrada := range [][]byte{
		append(append([]byte{}, b...), ' '),
		bytes.Replace(b, []byte(`"material":`), []byte(`"extra":0,"material":`), 1),
		bytes.Replace(b, []byte(`"titulo":"Propuesta sintética"`), []byte(`"titulo":"otra","titulo":"Propuesta sintética"`), 1),
		[]byte(`{"esquema":"bolsa.preparacion_bases.material.v0","material":{}}`),
		bytes.Repeat([]byte("x"), MaximoBytesMaterial+129),
	} {
		if _, err := DecodificarMaterialCanonico(entrada); !errors.Is(err, ErrMaterialInvalido) {
			t.Fatalf("bytes no canonicos aceptados: %v", err)
		}
	}
}

func TestEvaluacionVaciaConservaLos23ParesOrdenados(t *testing.T) {
	evaluacion, err := EvaluarMaterialBases(Material{})
	if err != nil {
		t.Fatal(err)
	}
	esperados := []Pendiente{
		{"fuente_bases", "referencia_ausente"}, {"catalogos", "referencia_ausente"}, {"calendario", "referencia_ausente"},
		{"reglas_baremacion", "referencia_ausente"}, {"flujo_proceso", "referencia_ausente"}, {"flujo_solicitud", "referencia_ausente"},
		{"plantilla", "referencia_ausente"}, {"plaza", "referencia_ausente"}, {"oep", "referencia_ausente"}, {"rpt", "referencia_ausente"},
		{"identificador_publico", "material_ausente"}, {"tipo", "material_ausente"}, {"titulo", "material_ausente"}, {"resumen", "material_ausente"},
		{"catalogo_categorias", "material_ausente"}, {"categorias", "material_ausente"}, {"plazos", "material_ausente"}, {"documentos_propuestos", "material_ausente"},
		{"contenido", "contenido_no_validado"}, {"documentos_admitidos", "circuito_pendiente"}, {"firma_y_custodia", "circuito_pendiente"},
		{"acto_aprobacion", "circuito_pendiente"}, {"publicacion_oficial", "circuito_pendiente"},
	}
	if evaluacion.ContenidoCanonico != nil || !reflect.DeepEqual(evaluacion.Pendientes, esperados) {
		t.Fatalf("evaluacion=%+v", evaluacion)
	}
	pendientes, err := (Material{}).Pendientes()
	if err != nil || !reflect.DeepEqual(pendientes, evaluacion.Pendientes) {
		t.Fatal("Pendientes no delega en la evaluacion comun")
	}
}

func TestMaterialIncompletoConservaPendientesSinGobierno(t *testing.T) {
	m := Material{Contenido: bolsa.ContenidoPublicableConvocatoria{Titulo: "Propuesta sintética incompleta"}}
	canonico, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	pendientes, err := canonico.Pendientes()
	if err != nil || len(pendientes) != 22 {
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
