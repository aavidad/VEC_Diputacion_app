package informemovimientos

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/ports"
	"vec-diputacion-granada/internal/vec/adapters/documentos/pdf"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

type rendererPrueba struct {
	contenido       vecdomain.ContenidoDocumento
	datos           []byte
	fallo           error
	falloValidacion error
	validaciones    int
}

func (*rendererPrueba) Formato() vecdomain.FormatoDocumento { return vecdomain.FormatoDocumentoPDF }
func (r *rendererPrueba) Renderizar(_ context.Context, c vecdomain.ContenidoDocumento) ([]byte, error) {
	r.contenido = c
	if r.fallo != nil {
		return nil, r.fallo
	}
	if r.datos != nil {
		return r.datos, nil
	}
	return []byte("%PDF-1.7\n/Lang (es-ES)\n%%EOF"), nil
}
func (r *rendererPrueba) ValidarSalida(context.Context, []byte) error {
	r.validaciones++
	return r.falloValidacion
}
func catalogoPrueba(t *testing.T, idioma string) []byte {
	t.Helper()
	b, err := os.ReadFile("../../../../../web/static/textos/" + idioma + "/cronos-informe-movimientos.json")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func ejemploPrueba(t *testing.T) EjemploSintetico {
	t.Helper()
	b, err := os.ReadFile("../../../../../cmd/vec-cronos-informe-saldo/testdata/movimientos.json")
	if err != nil {
		t.Fatal(err)
	}
	e, err := LeerEjemploSintetico(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func TestInformeListaHechosOrdenadosConCoincidenciasZonaYFuenteIncompleta(t *testing.T) {
	r := &rendererPrueba{}
	raw := catalogoPrueba(t, "es")
	p, err := Nuevo(r, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	e := ejemploPrueba(t)
	original := e.Marcajes[0]
	d, err := p.PrepararEjemploSintetico(context.Background(), e)
	if err != nil {
		t.Fatal(err)
	}
	if e.Marcajes[0].InstanteUTC != original.InstanteUTC || r.validaciones != 1 {
		t.Fatal("se modificó la fuente o no se validó el PDF")
	}
	parrafos := r.contenido.Parrafos
	if len(parrafos) != 10 {
		t.Fatal(parrafos)
	}
	for _, fragmento := range []string{"Ejemplo sintético", "Europe/Madrid", "fuente del ejemplo es incompleta", "02:30:00 +02:00 CEST. Entrada. Origen: Terminal (declarado)", "02:30:00 +02:00 CEST. Entrada. Origen: Sin verificar", "02:45:00 +02:00 CEST. Inicio de pausa", "02:30:00 +01:00 CET. Salida"} {
		if !strings.Contains(strings.Join(parrafos, "\n"), fragmento) {
			t.Fatalf("no figura %q: %v", fragmento, parrafos)
		}
	}
	if !strings.Contains(parrafos[5], "Terminal") || !strings.Contains(parrafos[6], "Sin verificar") || !strings.Contains(parrafos[8], "Salida") {
		t.Fatal("perdió coincidencias u orden estable", parrafos)
	}
	for _, datoAjeno := range []string{"marcaje_ref", "canal_ref", "empleado_ref", "motivo", "documento_ref", "horas trabajadas:", "Saldo:"} {
		if strings.Contains(strings.Join(parrafos, "\n"), datoAjeno) {
			t.Fatal("dato ajeno", datoAjeno)
		}
	}
	suma := sha256.Sum256(raw)
	if d.CatalogoSHA256 != hex.EncodeToString(suma[:]) || d.CatalogoVersion != 1 || d.CatalogoRef != "cronos-informe-movimientos-es" {
		t.Fatal("catálogo no ligado", d)
	}
}
func TestInformeVacioNoAcreditaAusenciaYFuenteCompletaEsDeclarada(t *testing.T) {
	r := &rendererPrueba{}
	p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	e := ejemploPrueba(t)
	e.Marcajes = []ports.MarcajeDia{}
	completo := true
	e.Completo = &completo
	if _, err := p.PrepararEjemploSintetico(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	s := strings.Join(r.contenido.Parrafos, "\n")
	if !strings.Contains(s, "no acredita una ausencia laboral") || !strings.Contains(s, "fuente del ejemplo se declara completa") || !strings.Contains(s, "No acredita el registro real") {
		t.Fatal(s)
	}
}
func TestInformeRealESENConIdiomaYDeterminismo(t *testing.T) {
	for _, c := range []struct{ carpeta, idioma string }{{"es", "es-ES"}, {"en", "en-GB"}} {
		t.Run(c.carpeta, func(t *testing.T) {
			p, err := Nuevo(pdf.Renderizador{Idioma: c.idioma}, bytes.NewReader(catalogoPrueba(t, c.carpeta)))
			if err != nil {
				t.Fatal(err)
			}
			e := ejemploPrueba(t)
			d, err := p.PrepararEjemploSintetico(context.Background(), e)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasPrefix(d.Contenido, []byte("%PDF-")) || !bytes.Contains(d.Contenido, []byte("/Lang ("+c.idioma+")")) {
				t.Fatal("PDF o idioma discordante")
			}
			otra, err := p.PrepararEjemploSintetico(context.Background(), e)
			if err != nil || !bytes.Equal(d.Contenido, otra.Contenido) {
				t.Fatal("PDF no determinista", err)
			}
		})
	}
}
func TestInformeErroresDeSalidaNoDevuelvenBytes(t *testing.T) {
	for _, r := range []*rendererPrueba{{fallo: errors.New("renderizar")}, {falloValidacion: pdf.ErrSalidaPDFInvalida}, {datos: []byte("%PDF-1.7\n/Lang (en-GB)\n%%EOF")}, {datos: make([]byte, 2*1024*1024+1)}} {
		p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, "es")))
		if err != nil {
			t.Fatal(err)
		}
		d, err := p.PrepararEjemploSintetico(context.Background(), ejemploPrueba(t))
		if err == nil || len(d.Contenido) != 0 {
			t.Fatal("salida abierta", err)
		}
	}
}
func TestInformeRechazaDatosAjenosOInvalidosAntesDelRenderer(t *testing.T) {
	casos := map[string]func(*EjemploSintetico){
		"sin_demo": func(e *EjemploSintetico) { e.Demo = false }, "nombre_control": func(e *EjemploSintetico) { e.Nombre = "Carmen\nMolina" }, "completo_ausente": func(e *EjemploSintetico) { e.Completo = nil }, "lista_ausente": func(e *EjemploSintetico) { e.Marcajes = nil }, "zona_desconocida": func(e *EjemploSintetico) { e.ZonaHoraria = "No/Existe" }, "zona_vacia": func(e *EjemploSintetico) { e.ZonaHoraria = "" }, "tipo_ajeno": func(e *EjemploSintetico) { e.Periodo.Tipo = "otro" }, "periodo_invertido": func(e *EjemploSintetico) { e.Periodo.Hasta = "2026-10-24" }, "fecha_imposible": func(e *EjemploSintetico) { e.Periodo.Desde = "2026-02-30" }, "rango_excesivo": func(e *EjemploSintetico) { e.Periodo.Tipo = ports.PeriodoSaldoRango; e.Periodo.Hasta = "2027-10-26" }, "hoy_varios_dias": func(e *EjemploSintetico) { e.Periodo.Hasta = "2026-10-26" }, "movimiento_ajeno": func(e *EjemploSintetico) { e.Marcajes[0].Movimiento = "otro" }, "instante_cero": func(e *EjemploSintetico) { e.Marcajes[0].InstanteUTC = time.Time{} }, "no_utc": func(e *EjemploSintetico) {
			e.Marcajes[0].InstanteUTC = e.Marcajes[0].InstanteUTC.In(time.FixedZone("otra", 3600))
		}, "precision_ajena": func(e *EjemploSintetico) { e.Marcajes[0].InstanteUTC = e.Marcajes[0].InstanteUTC.Add(time.Nanosecond) }, "fuera_periodo": func(e *EjemploSintetico) { e.Marcajes[0].InstanteUTC = time.Date(2026, 10, 25, 23, 0, 0, 0, time.UTC) }, "origen_inventado": func(e *EjemploSintetico) { origen := "portal"; e.Marcajes[0].Origen = &origen }, "limite_marcajes": func(e *EjemploSintetico) { e.Marcajes = make([]ports.MarcajeDia, 10001) },
	}
	for nombre, cambio := range casos {
		t.Run(nombre, func(t *testing.T) {
			r := &rendererPrueba{}
			p, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, "es")))
			if err != nil {
				t.Fatal(err)
			}
			e := ejemploPrueba(t)
			cambio(&e)
			d, err := p.PrepararEjemploSintetico(context.Background(), e)
			if !errors.Is(err, ErrEjemploInvalido) || len(d.Contenido) != 0 || r.contenido.Titulo != "" {
				t.Fatal("dato inválido llegó al renderer", err)
			}
		})
	}
}
func TestInformePeriodosExistentesYLimitesCiviles(t *testing.T) {
	for _, p := range []ports.PeriodoConsultaSaldo{{Tipo: ports.PeriodoSaldoHoy, Desde: "2026-10-25", Hasta: "2026-10-25"}, {Tipo: ports.PeriodoSaldoSemana, Desde: "2026-10-19", Hasta: "2026-10-25"}, {Tipo: ports.PeriodoSaldoMes, Desde: "2026-10-01", Hasta: "2026-10-31"}, {Tipo: ports.PeriodoSaldoAnio, Desde: "2026-01-01", Hasta: "2026-12-31"}, {Tipo: ports.PeriodoSaldoRango, Desde: "2026-10-25", Hasta: "2027-10-25"}} {
		r := &rendererPrueba{}
		preparador, err := Nuevo(r, bytes.NewReader(catalogoPrueba(t, "es")))
		if err != nil {
			t.Fatal(err)
		}
		e := ejemploPrueba(t)
		e.Periodo = p
		if _, err := preparador.PrepararEjemploSintetico(context.Background(), e); err != nil {
			t.Fatal(p, err)
		}
	}
}
func TestInformeCanceladoNoProduceDocumento(t *testing.T) {
	p, err := Nuevo(&rendererPrueba{}, bytes.NewReader(catalogoPrueba(t, "es")))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d, err := p.PrepararEjemploSintetico(ctx, ejemploPrueba(t))
	if !errors.Is(err, context.Canceled) || len(d.Contenido) != 0 {
		t.Fatal(err)
	}
}
func TestCatalogoEstrictoIdiomaYPlantillas(t *testing.T) {
	base := catalogoPrueba(t, "es")
	for nombre, alterar := range map[string]func(*Catalogo){"version_alias": func(c *Catalogo) { c.Version = "01" }, "idioma_alias": func(c *Catalogo) { c.Idioma = "es_ES" }, "hora_sin_desfase": func(c *Catalogo) { c.FormatoHora = "15:04" }, "movimiento_ajeno": func(c *Catalogo) { c.Movimientos["inventado"] = "Inventado" }, "origen_faltante": func(c *Catalogo) { delete(c.Origenes, "terminal") }, "marcador_faltante": func(c *Catalogo) { c.Fila = "{{fecha}} {{hora}} {{movimiento}}" }, "marcador_ajeno": func(c *Catalogo) { c.Fila += " {{motivo}}" }, "control": func(c *Catalogo) { c.Vacio = "\x00" }} {
		t.Run(nombre, func(t *testing.T) {
			var c Catalogo
			if json.Unmarshal(base, &c) != nil {
				t.Fatal("catálogo base")
			}
			alterar(&c)
			raw, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Nuevo(&rendererPrueba{}, bytes.NewReader(raw)); !errors.Is(err, ErrEjemploInvalido) {
				t.Fatal(err)
			}
		})
	}
	for _, raw := range [][]byte{bytes.Replace(base, []byte(`"version": "1"`), []byte(`"version": "1", "version": "1"`), 1), bytes.Replace(base, []byte(`"version": "1"`), []byte(`"version": "1", "VERSION": "1"`), 1), append(append([]byte(nil), base...), []byte(`{}`)...), append(append([]byte(nil), base...), bytes.Repeat([]byte(" "), 65537)...)} {
		if _, err := Nuevo(&rendererPrueba{}, bytes.NewReader(raw)); !errors.Is(err, ErrEjemploInvalido) {
			t.Fatal("catálogo no estricto", err)
		}
	}
}

func TestLeerEjemploRechazaAliasUnicodeAntesDeDecodificar(t *testing.T) {
	raw, err := os.ReadFile("../../../../../cmd/vec-cronos-informe-saldo/testdata/movimientos.json")
	if err != nil {
		t.Fatal(err)
	}
	objeto := bytes.TrimSpace(raw)
	for nombre, clave := range map[string]string{"ascii": "MARCAJES", "unicode": "marcajeſ", "unicode_escapado": `marcaje\u017f`} {
		t.Run(nombre, func(t *testing.T) {
			datos := append(append([]byte(nil), objeto[:len(objeto)-1]...), []byte(`, "`+clave+`": []}`)...)
			e, err := LeerEjemploSintetico(bytes.NewReader(datos))
			if !errors.Is(err, ErrEjemploInvalido) || e.Marcajes != nil {
				t.Fatal("un alias sobrescribió la lista", err)
			}
		})
	}
}
