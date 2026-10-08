package inscripcion

import "testing"

func TestPresentacionRechazaCategoriaAjenaYDeclaracionesDuplicadas(t *testing.T) {
	base := Presentacion{
		ConvocatoriaRef: "cv1_YXV4aWxpYXI_v1", CategoriaRef: "categoria:rpt:auxiliar",
		CatalogoVersion: 1, ClaveIdempotencia: "inscripcion-00000001",
	}
	if err := base.Validar(); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre  string
		cambiar func(*Presentacion)
	}{
		{"sin categoria", func(p *Presentacion) { p.CategoriaRef = "" }},
		{"clave demasiado corta", func(p *Presentacion) { p.ClaveIdempotencia = "otra" }},
		{"evidencia con ruta", func(p *Presentacion) {
			p.Declaraciones = []Declaracion{{RequisitoCodigo: "identidad_certificada", EvidenciaRef: "../../ajena"}}
		}},
		{"requisito duplicado", func(p *Presentacion) {
			p.Declaraciones = []Declaracion{{RequisitoCodigo: "identidad_certificada"}, {RequisitoCodigo: "identidad_certificada"}}
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			p := base
			caso.cambiar(&p)
			if p.Validar() == nil {
				t.Fatal("entrada insegura aceptada")
			}
		})
	}
}
