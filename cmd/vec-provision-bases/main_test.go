package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func ejecutarJSON(t *testing.T, args ...string) (respuesta, int, string) {
	t.Helper()
	var out, errout bytes.Buffer
	code := ejecutar(args, &out, &errout)
	var r respuesta
	if code == 0 {
		if err := json.Unmarshal(out.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
	}
	return r, code, errout.String()
}

func TestCatalogoPublicoConVersionesYModalidadesSeparadas(t *testing.T) {
	r, code, errout := ejecutarJSON(t)
	if code != 0 || len(r.Procesos) != 2 {
		t.Fatal(code, errout)
	}
	concurso := r.Procesos[0]
	if concurso.Expediente != "2026/PPT_01/000026" || concurso.FechaCorteExclusiva != "2026-05-05" || len(concurso.PuestosAnexoCodigo) != 9 || len(concurso.PuestosExcluidosRectificacionCodigo) != 2 || len(concurso.Fuentes) != 3 {
		t.Fatal("datos publicados incompletos")
	}
	for _, codigo := range concurso.PuestosAnexoCodigo {
		if codigo == "3724" || codigo == "2894" {
			t.Fatal("puesto eliminado presente en anexo", codigo)
		}
	}
	ld := r.Procesos[1]
	if ld.Modalidad != "libre_designacion" || ld.Expediente != "2025/PPT_01/000474" || len(ld.Fuentes) != 1 {
		t.Fatal("libre designación mezclada con concurso")
	}
	for _, a := range ld.Apartados {
		if a.FamiliaMotor != "" || a.MaximoMicropuntos != "" {
			t.Fatal("libre designación recibió puntuación de concurso")
		}
	}
}

func TestContrasteDelBorradorDetectaDiferenciasConcretas(t *testing.T) {
	original, err := os.Open("../vec-simular-provision/testdata/proceso.sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = original.Close() }()
	peticion, err := simulacion.DecodificarProceso(original)
	if err != nil {
		t.Fatal(err)
	}
	p := peticion.Proceso
	p.Configuracion.Reglas[0].Maximo = p.Configuracion.Reglas[1].Maximo
	archivo := filepath.Join(t.TempDir(), "proceso.json")
	datos, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivo, datos, 0600); err != nil {
		t.Fatal(err)
	}
	r, code, errout := ejecutarJSON(t, "-expediente", "2026/PPT_01/000026", "-proceso", archivo)
	if code != 0 || len(r.Procesos) != 1 || r.Procesos[0].Contraste == nil {
		t.Fatal(code, errout)
	}
	vistas := map[string]bool{}
	for _, d := range r.Procesos[0].Contraste.Diferencias {
		vistas[d.Codigo] = true
	}
	for _, codigo := range []string{"convocatoria_distinta", "base_sin_vinculo_publico", "fecha_corte_distinta", "maximo_familia_distinto", "regla_sin_vinculo_base"} {
		if !vistas[codigo] {
			t.Fatal("falta diferencia", codigo)
		}
	}
	if r.Procesos[0].Contraste.Estado != "cotejo_parcial" || r.Procesos[0].Contraste.CoberturaDatos != "parcial" || r.Procesos[0].Contraste.OfertaEstado != "no_cotejada" || len(r.Procesos[0].Contraste.Pendientes) == 0 {
		t.Fatal("el cotejo no conserva los pendientes")
	}
}

func TestOfertaAjenaNoSePresentaComoCotejada(t *testing.T) {
	original, err := os.Open("../vec-simular-provision/testdata/proceso.sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = original.Close() }()
	peticion, err := simulacion.DecodificarProceso(original)
	if err != nil {
		t.Fatal(err)
	}
	p := peticion.Proceso
	p.Referencia = "2026/PPT_01/000026"
	p.Configuracion.ConvocatoriaRef = p.Referencia
	p.Configuracion.BasesRef = "BOP-GRA-2026-060006"
	p.Configuracion.FechaCorte, err = baremacion.NuevaFechaCivil(2026, 5, 5)
	if err != nil {
		t.Fatal(err)
	}
	referencias := map[string]string{"valoracion_trabajo": "A", "grado": "B", "antiguedad": "C", "permanencia": "D", "titulaciones": "E", "cursos": "F"}
	for i := range p.Configuracion.Reglas {
		p.Configuracion.Reglas[i].ReferenciaBase = "BOP-GRA-2026-060006#base:6:" + referencias[string(p.Configuracion.Reglas[i].Familia)]
	}
	// El borrador sigue ofreciendo solo dos puestos sintéticos ajenos al anexo.
	archivo := filepath.Join(t.TempDir(), "proceso.json")
	datos, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivo, datos, 0600); err != nil {
		t.Fatal(err)
	}
	r, code, errout := ejecutarJSON(t, "-expediente", p.Referencia, "-proceso", archivo)
	if code != 0 {
		t.Fatal(code, errout)
	}
	c := r.Procesos[0].Contraste
	if c == nil || len(c.Diferencias) != 0 || c.Estado != "cotejo_parcial" || c.CoberturaDatos != "parcial" || c.OfertaEstado != "no_cotejada" {
		t.Fatal("oferta ajena presentada como cotejada", c)
	}
	if !strings.Contains(strings.Join(c.Pendientes, ","), "codigos_del_anexo_no_cotejados") {
		t.Fatal("falta pendiente de anexo", c.Pendientes)
	}
}

func TestFuenteLocalDiferenteYErroresSinRuta(t *testing.T) {
	archivo := filepath.Join(t.TempDir(), "documento-privado.pdf")
	if err := os.WriteFile(archivo, []byte("%PDF-distinto"), 0600); err != nil {
		t.Fatal(err)
	}
	r, code, errout := ejecutarJSON(t, "-expediente", "2026/PPT_01/000026", "-tipo-fuente", "bop_bases_y_anexo_rectificados", "-verificar-fuente", archivo)
	if code != 0 || r.Fuente == nil || r.Fuente.Estado != "difiere" || r.Fuente.SHA256 == "" {
		t.Fatal(code, errout)
	}
	_, code, errout = ejecutarJSON(t, "-expediente", "2026/PPT_01/000026", "-tipo-fuente", "bop_bases_y_anexo_rectificados", "-verificar-fuente", archivo+"-ausente")
	if code != 1 || strings.Contains(errout, archivo) || !strings.Contains(errout, "fuente_invalida") {
		t.Fatal(code, errout)
	}
	_, code, errout = ejecutarJSON(t, "-expediente", "2025/PPT_01/000474", "-proceso", archivo)
	if code != 1 || !strings.Contains(errout, "contraste_no_disponible") {
		t.Fatal(code, errout)
	}
}

func TestFIFOSeRechazaSinBloquearNiRevelarRuta(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "documento-privado-fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	type resultado struct {
		code int
		out  string
		err  string
	}
	done := make(chan resultado, 1)
	go func() {
		var out, errout bytes.Buffer
		code := ejecutar([]string{"-expediente", "2026/PPT_01/000026", "-tipo-fuente", "bop_bases_y_anexo_rectificados", "-verificar-fuente", fifo}, &out, &errout)
		done <- resultado{code: code, out: out.String(), err: errout.String()}
	}()
	select {
	case r := <-done:
		if r.code != 1 || r.out != "" || !strings.Contains(r.err, "fuente_invalida") || strings.Contains(r.err, fifo) {
			t.Fatal(r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("lectura de FIFO bloqueada")
	}
}
