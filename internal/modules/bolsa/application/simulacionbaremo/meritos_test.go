package simulacionbaremo

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestServicioMeritosRecorridoYReproduccion(t *testing.T) {
	for _, caso := range []struct{ reglas, entrada, estado, total string }{{"meritos_reglas_a", "meritos_entrada", "completado", "4000000"}, {"meritos_reglas_b", "meritos_entrada", "completado", "6000000"}, {"meritos_reglas_a", "meritos_bloqueada", "bloqueado", ""}} {
		t.Run(caso.reglas+caso.entrada, func(t *testing.T) {
			s := solicitudPrueba(t, caso.reglas, caso.entrada)
			r, err := (ServicioMeritos{}).SimularMeritos(s)
			if err != nil {
				t.Fatal(err)
			}
			var v struct {
				Esquema, Alcance string
				Huella           string `json:"huella_resultado_sha256"`
				Resultado        json.RawMessage
			}
			if err := json.Unmarshal(r.RepresentacionCanonica(), &v); err != nil {
				t.Fatal(err)
			}
			var semantica struct {
				Estado string
				Total  string
			}
			if err := json.Unmarshal(v.Resultado, &semantica); err != nil {
				t.Fatal(err)
			}
			if v.Esquema != "vec.bolsa.simulacion_meritos.v1" || v.Alcance != "simulacion" || semantica.Estado != caso.estado || semantica.Total != caso.total || v.Huella != huellaPrueba(v.Resultado) || v.Huella != r.HuellaResultadoSHA256() {
				t.Fatal("contrato incorrecto")
			}
			r2, err := (ServicioMeritos{}).SimularMeritos(s)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(r.RepresentacionCanonica(), r2.RepresentacionCanonica()) {
				t.Fatal("reproducción diferente")
			}
			copia := r.RepresentacionCanonica()
			copia[0] = 'x'
			if r.RepresentacionCanonica()[0] != '{' {
				t.Fatal("salida mutable")
			}
		})
	}
}
func TestServicioMeritosFalloTecnicoSinResultado(t *testing.T) {
	for _, fase := range []string{"conjunto", "entrada"} {
		t.Run(fase, func(t *testing.T) {
			s := solicitudPrueba(t, "meritos_reglas_a", "meritos_entrada")
			if fase == "conjunto" {
				s.HuellaConjuntoSHA256 = strings.Repeat("0", 64)
			} else {
				s.HuellaEntradaSHA256 = strings.Repeat("0", 64)
			}
			r, err := (ServicioMeritos{}).SimularMeritos(s)
			var f *Error
			if !errors.As(err, &f) || f.Fase != fase || len(r.RepresentacionCanonica()) != 0 {
				t.Fatal("error no segregado")
			}
		})
	}
}
func TestEjemplosEmbebidosSonCopiasYServicioConcurrente(t *testing.T) {
	r, e := DatosEjemploMeritos()
	if !bytes.Equal(r, solicitudPrueba(t, "meritos_reglas_a", "meritos_entrada").ConjuntoCanonico) {
		t.Fatal("ejemplo diferente")
	}
	r[0] = 'x'
	e[0] = 'x'
	r2, e2 := DatosEjemploMeritos()
	if r2[0] != '{' || e2[0] != '{' {
		t.Fatal("fixture compartida mutable")
	}
	rx, ex := DatosEjemploExperiencia()
	s := Solicitud{rx, huellaPrueba(rx), ex, huellaPrueba(ex)}
	if _, err := (Servicio{}).Simular(s); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := DatosEjemploMeritos()
			s := Solicitud{r, huellaPrueba(r), e, huellaPrueba(e)}
			if _, err := (ServicioMeritos{}).SimularMeritos(s); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}
