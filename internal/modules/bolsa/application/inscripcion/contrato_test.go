package inscripcion

import (
	"strconv"
	"testing"
	"time"
)

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

func TestConvocatoriaConTodasLasCategoriasPublicadasNoDesaparece(t *testing.T) {
	b := BolsaAbierta{ConvocatoriaRef: "cv1_" + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + "_v1",
		Titulo: "Convocatoria de prueba", NumeroCategorias: 128, CatalogoVersion: 1,
		PlazoInicio: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), PlazoFin: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		RequisitosResumen: "Consulte los requisitos", PuedeIniciar: true}
	for i := 1; i <= 128; i++ {
		b.Categorias = append(b.Categorias, Categoria{CategoriaRef: "categoria:rpt:" + strconv.Itoa(i), Categoria: "Categoría"})
	}
	if !bolsaAbiertaValida(b, true) {
		t.Fatal("convocatoria valida con 128 categorias desaparecio")
	}
	b.Categorias = append(b.Categorias, Categoria{CategoriaRef: "categoria:rpt:129", Categoria: "Categoría"})
	if bolsaAbiertaValida(b, true) {
		t.Fatal("limite de fuente superado")
	}
	b.Categorias = nil
	if !bolsaAbiertaValida(b, false) {
		t.Fatal("resumen sin array debia ser valido")
	}
}
