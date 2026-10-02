package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"io"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/web"
)

func ejemplo(t *testing.T) entrada {
	t.Helper()
	datos, err := os.ReadFile("ejemplo.json")
	if err != nil {
		t.Fatal(err)
	}
	var e entrada
	if err = json.Unmarshal(datos, &e); err != nil {
		t.Fatal(err)
	}
	return e
}
func correr(t *testing.T, e entrada) ([]byte, []byte, int) {
	t.Helper()
	datos, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var out, fallo bytes.Buffer
	estado := ejecutar(nil, bytes.NewReader(datos), &out, &fallo)
	return out.Bytes(), fallo.Bytes(), estado
}
func TestJSONHTMLMismoManifiestoYHuella(t *testing.T) {
	e := ejemplo(t)
	j, errout, estado := correr(t, e)
	if estado != 0 {
		t.Fatalf("%s", errout)
	}
	var s salida
	if err := json.Unmarshal(j, &s); err != nil {
		t.Fatal(err)
	}
	if s.Manifiesto.Comparacion.Dotaciones.Cambios[0].Tipo != "cambio" || len(s.Manifiesto.Comparacion.PuestosIndividuales.Cambios) != 1 {
		t.Fatal("comparison missing")
	}
	e.Formato = "html"
	h, errout, estado := correr(t, e)
	if estado != 0 {
		t.Fatalf("%s", errout)
	}
	if !bytes.Contains(h, []byte(s.HuellaSHA256)) {
		t.Fatal("different fingerprint")
	}
	const inicio = `<pre id="manifiesto">`
	partes := strings.Split(string(h), inicio)
	if len(partes) != 2 {
		t.Fatal("manifest missing")
	}
	manifiestoHTML := html.UnescapeString(strings.Split(partes[1], "</pre>")[0])
	var m manifiesto
	if err := json.Unmarshal([]byte(manifiestoHTML), &m); err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(s.Manifiesto)
	b, _ := json.Marshal(m)
	if !bytes.Equal(a, b) {
		t.Fatal("different manifests")
	}
	repetido, _, estado := correr(t, e)
	if estado != 0 || !bytes.Equal(h, repetido) {
		t.Fatal("nondeterministic HTML")
	}
	e.Formato = "json"
	repetido, _, estado = correr(t, e)
	if estado != 0 || !bytes.Equal(j, repetido) {
		t.Fatal("nondeterministic JSON")
	}
}
func TestInformeEscapaYCoberturaParcial(t *testing.T) {
	e := ejemplo(t)
	e.Formato = "html"
	e.Despues.Unidades[0].Etiqueta = `<script>alert(1)</script>`
	e.Despues.Cobertura.PuestosIndividuales = "parcial"
	h, errout, estado := correr(t, e)
	if estado != 0 {
		t.Fatalf("%s", errout)
	}
	if bytes.Contains(h, []byte("<script>")) || !bytes.Contains(h, []byte("&lt;script&gt;")) {
		t.Fatal("unescaped name")
	}
	if !bytes.Contains(h, []byte("No verificable con la cobertura declarada")) || bytes.Contains(h, []byte("Salida del corte</td><td>Código")) {
		t.Fatal("partial coverage lost")
	}
	e.Formato = "json"
	j, _, estado := correr(t, e)
	if estado != 0 {
		t.Fatal("json failed")
	}
	var s salida
	_ = json.Unmarshal(j, &s)
	if len(s.Manifiesto.Comparacion.PuestosIndividuales.Cambios) != 0 || len(s.Manifiesto.Comparacion.PuestosIndividuales.NoVerificables) != 1 {
		t.Fatal("absence inferred")
	}
}
func TestIdiomasDelCatalogo(t *testing.T) {
	catalogo, mensajes, err := web.CatalogoComparacionOrganizacion()
	if err != nil {
		t.Fatal(err)
	}
	for _, idioma := range catalogo.Locales() {
		e := ejemplo(t)
		e.Idioma = idioma
		e.Formato = "html"
		h, errout, estado := correr(t, e)
		if estado != 0 {
			t.Fatalf("%s", errout)
		}
		if !bytes.Contains(h, []byte(`lang="`+idioma+`"`)) || !bytes.Contains(h, []byte(mensajes[idioma]["titulo"])) {
			t.Fatal("incorrect locale")
		}
		for k := range mensajes[catalogo.DefaultLocale()] {
			if mensajes[idioma][k] == "" {
				t.Fatalf("missing %s", k)
			}
		}
	}
}
func TestEntradaInvalidaEsOpacaYSinSalida(t *testing.T) {
	e := ejemplo(t)
	valido, _ := json.Marshal(e)
	casos := [][]byte{[]byte(`{"secret":"dato_sensible"}`), append(append([]byte{}, valido...), []byte(`{}`)...), []byte(`{"sintetico":true,"sintetico":false,"secret":"dato_sensible"}`), bytes.Repeat([]byte("x"), limiteEntrada+1)}
	for _, mutar := range []func(*entrada){func(e *entrada) { e.Sintetico = false }, func(e *entrada) { e.Idioma = "unknown" }, func(e *entrada) { e.Formato = "pdf" }, func(e *entrada) { e.Despues.Selector.Cursor = "cursor" }, func(e *entrada) { e.Antes.Cobertura.Unidades = "unknown" }} {
		e := ejemplo(t)
		mutar(&e)
		b, _ := json.Marshal(e)
		casos = append(casos, b)
	}
	for _, dato := range casos {
		var out, fallo bytes.Buffer
		if ejecutar(nil, bytes.NewReader(dato), &out, &fallo) == 0 || out.Len() != 0 || bytes.Contains(fallo.Bytes(), []byte("dato_sensible")) || fallo.Len() == 0 {
			t.Fatal("invalid input accepted or disclosed")
		}
	}
	var out, fallo bytes.Buffer
	if ejecutar([]string{"archivo"}, bytes.NewReader(valido), &out, &fallo) == 0 || out.Len() != 0 {
		t.Fatal("arguments allowed")
	}
}
func TestFalloLecturaYEscritura(t *testing.T) {
	var out, fallo bytes.Buffer
	if ejecutar(nil, readerFalla{}, &out, &fallo) == 0 || out.Len() != 0 {
		t.Fatal("read failure accepted")
	}
	e := ejemplo(t)
	b, _ := json.Marshal(e)
	if ejecutar(nil, bytes.NewReader(b), writerFalla{}, &fallo) == 0 {
		t.Fatal("write failure accepted")
	}
}

type readerFalla struct{}

func (readerFalla) Read([]byte) (int, error) { return 0, errors.New("secret") }

type writerFalla struct{}

func (writerFalla) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestEnterosGrandesNoPierdenPrecision(t *testing.T) {
	e := ejemplo(t)
	e.Formato = "html"
	e.Despues.Unidades[0].CatalogoVersion = 9007199254740993
	h, fallo, estado := correr(t, e)
	if estado != 0 {
		t.Fatalf("%s", fallo)
	}
	if !bytes.Contains(h, []byte("9.007.199.254.740.993")) {
		t.Fatal("integer precision lost")
	}
}

func TestFalloDiagnosticoDevuelveErrorDeSalida(t *testing.T) {
	var out bytes.Buffer
	if estado := ejecutar([]string{"archivo"}, bytes.NewReader(nil), &out, writerFalla{}); estado != 2 || out.Len() != 0 {
		t.Fatalf("diagnostic write failure: status=%d, output=%d", estado, out.Len())
	}
}

func ejemploPaginado(t *testing.T) entrada {
	t.Helper()
	datos, err := os.ReadFile("ejemplo-paginado-sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var e entrada
	if err := json.Unmarshal(datos, &e); err != nil {
		t.Fatal(err)
	}
	return e
}
func TestPaginasSinteticasReutilizanMotorYManifiestoJSONHTML(t *testing.T) {
	paginado := ejemploPaginado(t)
	j, fallo, codigo := correr(t, paginado)
	if codigo != 0 {
		t.Fatalf("%s", fallo)
	}
	plano, _, codigo := correr(t, ejemplo(t))
	if codigo != 0 {
		t.Fatal("plain example failed")
	}
	var p, d salida
	if err := json.Unmarshal(j, &p); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(plano, &d); err != nil {
		t.Fatal(err)
	}
	esperado, _ := json.Marshal(d.Manifiesto.Comparacion)
	actual, _ := json.Marshal(p.Manifiesto.Comparacion)
	if !bytes.Equal(esperado, actual) {
		t.Fatal("paginated comparison differs from existing engine")
	}
	if len(p.Manifiesto.AntesPaginas) != len(paginado.AntesPaginas) || p.Manifiesto.Antes.Selector.Cursor != "" {
		t.Fatal("chain lost")
	}
	paginado.Formato = "html"
	h, fallo, codigo := correr(t, paginado)
	if codigo != 0 {
		t.Fatalf("%s", fallo)
	}
	const inicio = `<pre id="manifiesto">`
	partes := strings.Split(string(h), inicio)
	if len(partes) != 2 {
		t.Fatal("manifest missing")
	}
	var m manifiesto
	if err := json.Unmarshal([]byte(html.UnescapeString(strings.Split(partes[1], "</pre>")[0])), &m); err != nil {
		t.Fatal(err)
	}
	esperado, _ = json.Marshal(p.Manifiesto)
	actual, _ = json.Marshal(m)
	if !bytes.Equal(esperado, actual) || !bytes.Contains(h, []byte(p.HuellaSHA256)) {
		t.Fatal("JSON and HTML manifests differ")
	}
	repetido, _, codigo := correr(t, paginado)
	if codigo != 0 || !bytes.Equal(h, repetido) {
		t.Fatal("nondeterministic report")
	}
}
func TestPaginasSinteticasRechazanEntradaAmbiguaEIncompleta(t *testing.T) {
	for _, mutar := range []func(*entrada){
		func(e *entrada) { e.Sintetico = false },
		func(e *entrada) { e.Esquema = esquemaEntrada },
		func(e *entrada) { e.Antes = ejemplo(t).Antes },
		func(e *entrada) { e.AntesPaginas = nil },
		func(e *entrada) { e.DespuesPaginas = e.DespuesPaginas[:1] },
		func(e *entrada) { e.AntesPaginas[1].Instantanea.Selector.Cursor = "pagina_faltante" },
		func(e *entrada) { e.AntesPaginas[1].Instantanea.Cobertura.Plazas = "parcial" },
	} {
		e := ejemploPaginado(t)
		mutar(&e)
		out, fallo, codigo := correr(t, e)
		if codigo == 0 || len(out) != 0 || len(fallo) == 0 {
			t.Fatal("invalid chain accepted")
		}
	}
}
