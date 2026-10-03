package plannominal

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func TestDescriptorFijaPlanYConservaContratoNominal(t *testing.T) {
	for _, orden := range []int{1, 2} {
		f, m, selector, publicacion, consulta := descriptorPrueba(t, orden)
		d, err := f.DescriptorPlanFijadoFirmaV2(t.Context(), m)
		if err != nil {
			t.Fatal(err)
		}
		sha, err := consulta.catalogo.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		esperado := vd.ReferenciaEntradaCatalogo{CatalogoID: consulta.catalogo.ID, CatalogoVersion: consulta.catalogo.Version,
			CatalogoHuellaSHA256: sha, EntradaClave: selector.paso.EntradaClave}
		if d.Plan != esperado || selector.llamadas != 1 || publicacion.llamadas != 1 {
			t.Fatal("el pin y el descriptor no proceden de la misma selección")
		}
		canon, err := firma.CanonicoDescriptorPlanFijadoFirmaV2(m, d)
		if err != nil || firma.ValidarDescriptorPlanFijadoFirmaV2(m, esperado, canon) != nil {
			t.Fatalf("envoltorio: %v", err)
		}
		var e struct {
			Descriptor json.RawMessage `json:"descriptor"`
		}
		if json.Unmarshal(canon, &e) != nil {
			t.Fatal("envoltorio no interpretable")
		}
		original, err := firma.CanonicoDescriptorFirmaVerificadaV2(m, d.Descriptor)
		if err != nil || !bytes.Equal(original, e.Descriptor) {
			t.Fatal("el descriptor nominal original cambió sus bytes")
		}
		var claves map[string]json.RawMessage
		if json.Unmarshal(original, &claves) != nil || len(claves) != 11 {
			t.Fatal("el descriptor original perdió su contrato de once claves")
		}
	}
}

func TestDescriptorFijadoRechazaSustitucionDelPlan(t *testing.T) {
	for nombre, cambiar := range map[string]func(*vd.ReferenciaEntradaCatalogo){
		"referencia": func(p *vd.ReferenciaEntradaCatalogo) { p.CatalogoID = "plan.otro" },
		"version":    func(p *vd.ReferenciaEntradaCatalogo) { p.CatalogoVersion++ },
		"huella":     func(p *vd.ReferenciaEntradaCatalogo) { p.CatalogoHuellaSHA256 = strings.Repeat("e", 64) },
		"entrada":    func(p *vd.ReferenciaEntradaCatalogo) { p.EntradaClave = "entrada_otra" },
	} {
		t.Run(nombre, func(t *testing.T) {
			f, m, _, _, _ := descriptorPrueba(t, 1)
			d, err := f.DescriptorPlanFijadoFirmaV2(t.Context(), m)
			if err != nil {
				t.Fatal(err)
			}
			esperado := d.Plan
			cambiar(&d.Plan)
			canon, err := firma.CanonicoDescriptorPlanFijadoFirmaV2(m, d)
			if err != nil {
				t.Fatal(err)
			}
			if firma.ValidarDescriptorPlanFijadoFirmaV2(m, esperado, canon) == nil {
				t.Fatal("se aceptó una entrada distinta de la autorizada")
			}
		})
	}
}

func TestDescriptorFijadoRechazaJSONAmbiguoYMaterialCruzado(t *testing.T) {
	f, m, _, _, _ := descriptorPrueba(t, 1)
	d, err := f.DescriptorPlanFijadoFirmaV2(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	canon, err := firma.CanonicoDescriptorPlanFijadoFirmaV2(m, d)
	if err != nil {
		t.Fatal(err)
	}
	for _, datos := range [][]byte{
		append(bytes.Clone(canon), []byte(`{}`)...),
		append([]byte(" "), canon...),
		bytes.Replace(canon, []byte(`"plan":`), []byte(`"ajeno":null,"plan":`), 1),
		bytes.Replace(canon, []byte(`"esquema":`), []byte(`"esquema":"otro","esquema":`), 1),
	} {
		if firma.ValidarDescriptorPlanFijadoFirmaV2(m, d.Plan, datos) == nil {
			t.Fatal("se aceptó un envoltorio ambiguo o no canónico")
		}
	}
	m.OriginalRef = "original:otro"
	if firma.ValidarDescriptorPlanFijadoFirmaV2(m, d.Plan, canon) == nil {
		t.Fatal("se aceptó un descriptor ligado a otro original")
	}
}
