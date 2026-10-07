package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type clienteNominalPrueba struct {
	resultados []personalports.ResultadoConsultaOrganizacionHistorica
	selectores []personal.SelectorOrganizacionHistorica
	cerrado    bool
	errorEn    int
	mutar      func(int, *personalports.ResultadoConsultaOrganizacionHistorica)
}

func (c *clienteNominalPrueba) Consultar(ctx context.Context, s personal.SelectorOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	n := len(c.selectores)
	c.selectores = append(c.selectores, s)
	d, ok := ctx.Deadline()
	if !ok || time.Until(d) > tiempoConsultaNominal {
		return personalports.ResultadoConsultaOrganizacionHistorica{}, errors.New("missing bounded deadline")
	}
	if c.errorEn == n+1 || n >= len(c.resultados) {
		return personalports.ResultadoConsultaOrganizacionHistorica{}, errors.New("private source unavailable")
	}
	r := c.resultados[n]
	if c.mutar != nil {
		c.mutar(n, &r)
	}
	return r, nil
}
func (c *clienteNominalPrueba) Cerrar() { c.cerrado = true }

func preparacionNominal(t *testing.T) (entradaConsultaNominal, *clienteNominalPrueba) {
	t.Helper()
	datos, err := os.ReadFile("ejemplo-paginado-sintetico.json")
	if err != nil {
		t.Fatal(err)
	}
	var e entrada
	if err := json.Unmarshal(datos, &e); err != nil {
		t.Fatal(err)
	}
	c := &clienteNominalPrueba{}
	for _, corte := range [][]personal.PaginaInstantaneaOrganizacion{e.AntesPaginas, e.DespuesPaginas} {
		for _, p := range corte {
			i := p.Instantanea
			n := len(c.resultados) + 1
			c.resultados = append(c.resultados, personalports.ResultadoConsultaOrganizacionHistorica{
				Pagina: personalports.PaginaOrganizacionHistorica{Selector: i.Selector, VersionRPTRef: p.VersionRPTRef, VersionPlantillaRef: p.VersionPlantillaRef,
					Cobertura: personalports.CoberturaFuentesOrganizacionHistorica(i.Cobertura), Unidades: i.Unidades, PuestosTipo: i.PuestosTipo,
					Dotaciones: i.Dotaciones, Plazas: i.Plazas, PuestosIndividuales: i.PuestosIndividuales, Vinculos: i.Vinculos, CursorSiguiente: p.CursorSiguiente},
				Evidencia: personalports.EvidenciaConsultaOrganizacionHistorica{ReciboRef: fmt.Sprintf("recibo:prueba_%d", n), DecisionRef: fmt.Sprintf("decision:prueba_%d", n),
					AuditoriaRef: fmt.Sprintf("auditoria:prueba_%d", n), EfectoRef: i.Selector.OrganismoRef, ConsumoHuellaSHA256: fmt.Sprintf("%064x", n),
					ConsultadaEn: time.Date(2026, 1, 1, 0, 0, n, 0, time.UTC)},
			})
		}
	}
	return entradaConsultaNominal{Esquema: esquemaConsultaNominal, Formato: "json", Antes: e.AntesPaginas[0].Instantanea.Selector, Despues: e.DespuesPaginas[0].Instantanea.Selector}, c
}

func configNominalPrueba(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "consulta.json")
	cfg := configuracionConsultaNominal{Version: 1, Origen: "https://servidor.example.invalid", AutoridadCA: "ca.pem", CertificadoCliente: "cliente.pem", ClaveCliente: "clave.pem", MaximoBytesPagina: 1 << 20}
	datos, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, datos, 0600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func correrNominalPrueba(t *testing.T, ruta string, datos []byte, c *clienteNominalPrueba) (salidaConsultaNominal, []byte, []byte, int, int) {
	t.Helper()
	var out, errout bytes.Buffer
	creaciones := 0
	estado := ejecutarConsultaNominal(ruta, bytes.NewReader(datos), &out, &errout, func(cfg configuracionConsultaNominal) (clienteConsultaNominal, error) {
		creaciones++
		if cfg.directorio != filepath.Dir(ruta) || cfg.MaximoBytesPagina != 1<<20 {
			t.Fatal("configuration not passed to client factory")
		}
		return c, nil
	})
	var salida salidaConsultaNominal
	if estado == 0 && json.Unmarshal(out.Bytes(), &salida) != nil {
		t.Fatal("invalid output")
	}
	return salida, out.Bytes(), errout.Bytes(), estado, creaciones
}

func TestConsultaNominalPaginaCompletaYHuella(t *testing.T) {
	e, c := preparacionNominal(t)
	datos, _ := json.Marshal(e)
	s, out, errout, estado, creaciones := correrNominalPrueba(t, configNominalPrueba(t), datos, c)
	if estado != 0 || len(errout) != 0 || creaciones != 1 || !c.cerrado {
		t.Fatalf("status=%d error=%s", estado, errout)
	}
	m := s.Manifiesto
	if m.Esquema != esquemaManifiestoConsultaNominal || m.Modo != "consulta_autorizada" || m.PaginasAntes != 6 || m.PaginasDespues != 4 || m.TotalHechosAntes != 6 || m.TotalHechosDespues != 4 {
		t.Fatalf("unexpected counts %+v", m)
	}
	if len(m.Comparacion.Dotaciones.Cambios) != 1 || len(m.Comparacion.PuestosIndividuales.Cambios) != 1 {
		t.Fatal("domain comparison missing")
	}
	canonico, _ := json.Marshal(m)
	if s.HuellaSHA256 != huella(canonico) {
		t.Fatal("typed manifest fingerprint mismatch")
	}
	for _, corte := range [][]lecturaConsultaNominal{m.LecturasAntes, m.LecturasDespues} {
		for _, l := range corte {
			if l.Evidencia.EfectoRef != e.Antes.OrganismoRef {
				t.Fatal("effect reference changed")
			}
		}
	}
	for n, selector := range c.selectores {
		if selector != c.resultados[n].Pagina.Selector {
			t.Fatal("wrong selector or cursor sent")
		}
	}
	for _, secreto := range []string{"servidor.example.invalid", "ca.pem", "cliente.pem", "clave.pem", "sintetico", "autoridad_ca", "directorio", "catalogo_id"} {
		if bytes.Contains(out, []byte(secreto)) {
			t.Fatalf("unexpected configuration or full collection: %s", secreto)
		}
	}
	_, c2 := preparacionNominal(t)
	_, out2, _, estado2, _ := correrNominalPrueba(t, configNominalPrueba(t), datos, c2)
	if estado2 != 0 || !bytes.Equal(out, out2) {
		t.Fatal("configuration path changed manifest")
	}
}

func TestConsultaNominalValidaAmbosSelectoresAntesDeCliente(t *testing.T) {
	for _, caso := range []string{"segundo_invalido", "organismo", "unidad", "cursor_antes", "cursor_despues", "html", "esquema"} {
		t.Run(caso, func(t *testing.T) {
			e, c := preparacionNominal(t)
			switch caso {
			case "segundo_invalido":
				e.Despues.Limite = 0
			case "organismo":
				e.Despues.OrganismoRef = "organismo:otro"
			case "unidad":
				e.Despues.UnidadClave = "unidad:otra"
			case "cursor_antes":
				e.Antes.Cursor = "continuacion"
			case "cursor_despues":
				e.Despues.Cursor = "continuacion"
			case "html":
				e.Formato = "html"
			case "esquema":
				e.Esquema = esquemaEntradaPaginada
			}
			datos, _ := json.Marshal(e)
			_, out, errout, estado, creados := correrNominalPrueba(t, configNominalPrueba(t), datos, c)
			if estado == 0 || len(out) != 0 || len(errout) == 0 || creados != 0 || len(c.selectores) != 0 {
				t.Fatal("invalid pair caused a read or output")
			}
		})
	}
}

func TestConsultaNominalJSONEstricto(t *testing.T) {
	e, _ := preparacionNominal(t)
	datos, _ := json.Marshal(e)
	for _, caso := range []string{strings.Replace(string(datos), `"formato":"json"`, `"formato":"json","sintetico":true`, 1),
		strings.Replace(string(datos), `"formato":"json"`, `"formato":"json","formato":"json"`, 1),
		strings.Replace(string(datos), `"formato":"json"`, `"formato":"json","Formato":"json"`, 1),
		strings.Replace(string(datos), `"organismo_ref"`, `"Organismo_ref"`, 1),
		strings.Replace(string(datos), `"json"`, "\"js\xffon\"", 1), string(datos) + " {}", "null", strings.Repeat(" ", limiteEntrada+1)} {
		_, c := preparacionNominal(t)
		_, out, _, estado, creados := correrNominalPrueba(t, configNominalPrueba(t), []byte(caso), c)
		if estado == 0 || len(out) != 0 || creados != 0 {
			t.Fatal("invalid JSON caused output or client construction")
		}
	}
}

func TestConsultaNominalDeniegaCadenaIncoherenteSinSalidaParcial(t *testing.T) {
	for _, caso := range []string{"pagina_falla", "ambito", "selector", "version", "cobertura", "cursor", "hecho_duplicado", "recibo", "decision", "auditoria", "consumo", "evidencia_corte", "efecto", "instante", "pagina_exceso"} {
		t.Run(caso, func(t *testing.T) {
			e, c := preparacionNominal(t)
			if caso == "pagina_falla" {
				c.errorEn = 3
			}
			c.mutar = func(n int, r *personalports.ResultadoConsultaOrganizacionHistorica) {
				if caso == "evidencia_corte" && n == 6 {
					r.Evidencia = c.resultados[0].Evidencia
				}
				if n != 1 {
					return
				}
				switch caso {
				case "ambito":
					r.Pagina.Selector.OrganismoRef = "organismo:otra"
				case "selector":
					r.Pagina.Selector.VigenteEn = "2024-12-30"
				case "version":
					r.Pagina.VersionRPTRef = "rpt:otra"
				case "cobertura":
					r.Pagina.Cobertura.Unidades = "parcial"
				case "cursor":
					r.Pagina.CursorSiguiente = r.Pagina.Selector.Cursor
				case "hecho_duplicado":
					r.Pagina.PuestosTipo[0].Traza.ID = c.resultados[0].Pagina.Unidades[0].Traza.ID
				case "recibo":
					r.Evidencia.ReciboRef = c.resultados[0].Evidencia.ReciboRef
				case "decision":
					r.Evidencia.DecisionRef = c.resultados[0].Evidencia.DecisionRef
				case "auditoria":
					r.Evidencia.AuditoriaRef = c.resultados[0].Evidencia.AuditoriaRef
				case "consumo":
					r.Evidencia.ConsumoHuellaSHA256 = c.resultados[0].Evidencia.ConsumoHuellaSHA256
				case "efecto":
					r.Evidencia.EfectoRef = "organismo:otra"
				case "instante":
					r.Evidencia.ConsultadaEn = time.Time{}
				case "pagina_exceso":
					r.Pagina.PuestosTipo = make([]personal.PuestoTipoOrganizacionHistorica, 101)
				}
			}
			datos, _ := json.Marshal(e)
			_, out, errout, estado, _ := correrNominalPrueba(t, configNominalPrueba(t), datos, c)
			if estado == 0 || len(out) != 0 || len(errout) == 0 || !c.cerrado {
				t.Fatal("inconsistent page emitted output or client not closed")
			}
		})
	}
}

func TestConsultaNominalConfiguracionPrivada(t *testing.T) {
	for _, caso := range []string{"relativa", "modo_archivo", "modo_directorio", "symlink_archivo", "symlink_directorio", "git", "git_ancestro", "fifo", "directorio", "sin_maximo", "maximo_desbordado", "version", "desconocida", "duplicada", "alias", "utf8", "exceso"} {
		t.Run(caso, func(t *testing.T) {
			ruta := configNominalPrueba(t)
			datos, _ := os.ReadFile(ruta)
			switch caso {
			case "relativa":
				ruta = "consulta.json"
			case "modo_archivo":
				if err := os.Chmod(ruta, 0644); err != nil {
					t.Fatal(err)
				}
			case "modo_directorio":
				if err := os.Chmod(filepath.Dir(ruta), 0755); err != nil {
					t.Fatal(err)
				}
			case "symlink_archivo":
				enlace := ruta + ".link"
				if err := os.Symlink(ruta, enlace); err != nil {
					t.Fatal(err)
				}
				ruta = enlace
			case "symlink_directorio":
				enlace := filepath.Join(t.TempDir(), "privado")
				if err := os.Symlink(filepath.Dir(ruta), enlace); err != nil {
					t.Fatal(err)
				}
				ruta = filepath.Join(enlace, filepath.Base(ruta))
			case "git":
				if err := os.WriteFile(filepath.Join(filepath.Dir(ruta), ".git"), []byte("gitdir: elsewhere"), 0600); err != nil {
					t.Fatal(err)
				}
			case "git_ancestro":
				if err := os.WriteFile(filepath.Join(filepath.Dir(filepath.Dir(ruta)), ".git"), []byte("gitdir: elsewhere"), 0600); err != nil {
					t.Fatal(err)
				}
			case "fifo":
				if err := os.Remove(ruta); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(ruta, 0600); err != nil {
					t.Fatal(err)
				}
			case "directorio":
				ruta = filepath.Dir(ruta)
			case "sin_maximo":
				datos = bytes.Replace(datos, []byte(`"maximo_bytes_pagina":1048576`), []byte(`"maximo_bytes_pagina":0`), 1)
			case "maximo_desbordado":
				datos = bytes.Replace(datos, []byte(`"maximo_bytes_pagina":1048576`), []byte(`"maximo_bytes_pagina":9223372036854775807`), 1)
			case "version":
				datos = bytes.Replace(datos, []byte(`"version":1`), []byte(`"version":2`), 1)
			case "desconocida":
				datos = bytes.Replace(datos, []byte(`"version":1`), []byte(`"version":1,"actor_ref":"actor:intruso"`), 1)
			case "duplicada":
				datos = bytes.Replace(datos, []byte(`"version":1`), []byte(`"version":1,"version":1`), 1)
			case "alias":
				datos = bytes.Replace(datos, []byte(`"version":1`), []byte(`"version":1,"Version":1`), 1)
			case "utf8":
				datos = bytes.Replace(datos, []byte(`"clave.pem"`), []byte("\"clave\xff.pem\""), 1)
			case "exceso":
				datos = []byte(strings.Repeat(" ", limiteConfiguracionConsultaNominal+1))
			}
			if caso == "sin_maximo" || caso == "maximo_desbordado" || caso == "version" || caso == "desconocida" || caso == "duplicada" || caso == "alias" || caso == "utf8" || caso == "exceso" {
				if err := os.WriteFile(ruta, datos, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := leerConfiguracionConsultaNominal(ruta); err == nil {
				t.Fatal("unsafe configuration accepted")
			}
		})
	}
}

func TestConsultaNominalFabricaRealConectadaYErroresOpacos(t *testing.T) {
	e, _ := preparacionNominal(t)
	datos, _ := json.Marshal(e)
	ruta := configNominalPrueba(t) // Ficheros TLS ausentes: la fábrica real debe denegar.
	var out, errout bytes.Buffer
	estado := ejecutar([]string{"-consulta-nominal", ruta}, bytes.NewReader(datos), &out, &errout)
	if estado == 0 || out.Len() != 0 || errout.Len() == 0 || strings.Contains(errout.String(), ruta) || strings.Contains(errout.String(), "servidor.example.invalid") {
		t.Fatal("real factory disconnected or leaked private configuration")
	}
}

func TestReunirConsultaNominalContextoCancelado(t *testing.T) {
	e, c := preparacionNominal(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, lecturas, err := reunirConsultaNominal(ctx, c, e.Antes, map[string]bool{})
	if !errors.Is(err, context.Canceled) || len(c.selectores) != 0 || len(lecturas) != 0 {
		t.Fatal("canceled context still consumed page")
	}
}

func TestConsultaNominalCoberturaParcialNoInfiereAusencia(t *testing.T) {
	e, c := preparacionNominal(t)
	for n := range c.resultados {
		c.resultados[n].Pagina.Cobertura.PuestosIndividuales = "parcial"
	}
	datos, _ := json.Marshal(e)
	s, _, errout, estado, _ := correrNominalPrueba(t, configNominalPrueba(t), datos, c)
	if estado != 0 {
		t.Fatalf("partial coverage rejected: %s", errout)
	}
	if s.Manifiesto.Comparacion.CoberturaDespues.PuestosIndividuales != "parcial" || len(s.Manifiesto.Comparacion.PuestosIndividuales.Cambios) != 0 || len(s.Manifiesto.Comparacion.PuestosIndividuales.NoVerificables) != 1 {
		t.Fatal("pagination completion incorrectly implied completeness")
	}
}

func TestConsultaNominalLimiteAcumuladoHechos(t *testing.T) {
	e, c := preparacionNominal(t)
	base := c.resultados[0]
	c.resultados = nil
	for pagina := 0; pagina < 101; pagina++ {
		r := base
		r.Pagina.Selector = e.Antes
		if pagina > 0 {
			r.Pagina.Selector.Cursor = fmt.Sprintf("pagina_%d", pagina)
		}
		r.Pagina.CursorSiguiente = fmt.Sprintf("pagina_%d", pagina+1)
		r.Evidencia.ReciboRef = fmt.Sprintf("recibo:pagina_%d", pagina)
		r.Evidencia.DecisionRef = fmt.Sprintf("decision:pagina_%d", pagina)
		r.Evidencia.AuditoriaRef = fmt.Sprintf("auditoria:pagina_%d", pagina)
		r.Evidencia.ConsumoHuellaSHA256 = fmt.Sprintf("%064x", pagina+1)
		r.Pagina.Unidades = make([]personal.UnidadOrganizacionHistorica, 100)
		for hecho := range r.Pagina.Unidades {
			u := base.Pagina.Unidades[0]
			u.Traza.ID = fmt.Sprintf("unidad:pagina_%d_hecho_%d", pagina, hecho)
			r.Pagina.Unidades[hecho] = u
		}
		c.resultados = append(c.resultados, r)
	}
	datos, _ := json.Marshal(e)
	_, out, _, estado, _ := correrNominalPrueba(t, configNominalPrueba(t), datos, c)
	if estado == 0 || len(out) != 0 || len(c.selectores) != 101 || !c.cerrado {
		t.Fatal("cumulative fact limit failed or denied before the limit")
	}
}

type escritorNominalCorto struct{}

func (escritorNominalCorto) Write(b []byte) (int, error) { return len(b) - 1, nil }

func TestConsultaNominalEscrituraCorta(t *testing.T) {
	e, c := preparacionNominal(t)
	datos, _ := json.Marshal(e)
	var errout bytes.Buffer
	estado := ejecutarConsultaNominal(configNominalPrueba(t), bytes.NewReader(datos), escritorNominalCorto{}, &errout, func(configuracionConsultaNominal) (clienteConsultaNominal, error) { return c, nil })
	if estado == 0 || errout.Len() == 0 || !c.cerrado {
		t.Fatal("short write reported success")
	}
}
