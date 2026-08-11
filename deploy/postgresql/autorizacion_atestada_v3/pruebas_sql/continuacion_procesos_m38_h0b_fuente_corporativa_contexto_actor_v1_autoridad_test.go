//go:build ignore && linux && amd64

package main

import (
	"bytes"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func agregadoNominalO3cPruebaM38(t *testing.T) *agregadoO3cM38 {
	t.Helper()
	c := custodiaNominalO3bPruebaM38(t)
	c.ticketEscritor = nil
	c.lease.estado.Store(1)
	c.observador.palabra.Store(1)
	c.baselineSenal = 1
	return &agregadoO3cM38{
		estado:   capturaB5CapturadoM38,
		custodia: c,
		identidad: muestraStatO3bM38{pid: c.cmd.Process.Pid, estado: 'T', ppid: c.ppid,
			pgid: c.cmd.Process.Pid, sid: 1, inicio: 1},
	}
}

func TestAutoridadO3cConsumoNominalAliasClonYReuso(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	origen := agregadoNominalO3cPruebaM38(t)
	alias := origen
	c := origen.custodia
	identidad := origen.identidad
	a, err := consumirAutoridadO3cM38(&origen)
	if err != nil || origen != nil || !a.es(continuacionC0RecibidoM38) {
		t.Fatalf("consumo nominal: autoridad=%v err=%v", a, err)
	}
	if a.custodia != c || a.identidad != identidad || a.salida == nil || a.salida.auto != a.salida ||
		a.salida.custodia != c || a.salida.identidad != identidad || a.salida.primera.Load() != 0 ||
		!a.autoridad.poseeO3c() || c.observador.palabra.Load() != 2 || c.lease.estado.Load() != 3 ||
		c.baselineSenal != 2 {
		t.Fatal("consumo no consolidó autoridad conjunta exacta")
	}
	if segundo, err := consumirAutoridadO3cM38(&alias); !errors.Is(err, errUsoConsumidoO3cM38) || alias != nil ||
		!segundo.es(continuacionC8RetiradoM38) {
		t.Fatalf("alias aceptado: autoridad=%v err=%v", segundo, err)
	}
	clon := &agregadoO3cM38{estado: capturaB5CapturadoM38, custodia: c, identidad: identidad}
	if segundo, err := consumirAutoridadO3cM38(&clon); !errors.Is(err, errUsoConsumidoO3cM38) || clon != nil ||
		!segundo.es(continuacionC8RetiradoM38) {
		t.Fatalf("clon aceptado: autoridad=%v err=%v", segundo, err)
	}
}

func TestAutoridadO3cNulosYB5ReadOnly(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if a, err := consumirAutoridadO3cM38(nil); a != nil || !errors.Is(err, errEntradaO3cM38) {
		t.Fatal("puntero nulo aceptado")
	}
	var nulo *agregadoO3cM38
	if a, err := consumirAutoridadO3cM38(&nulo); a != nil || !errors.Is(err, errEntradaO3cM38) {
		t.Fatal("agregado nulo aceptado")
	}
	origen := agregadoNominalO3cPruebaM38(t)
	alias := origen
	estado := origen.estado
	if _, err := consumirAutoridadO3cM38(&origen); err != nil || alias.estado != estado || estado != capturaB5CapturadoM38 {
		t.Fatalf("B5 fue alterado: %v", err)
	}
	verdes := map[[2]estadoContinuacionO3cM38]bool{
		{continuacionC0RecibidoM38, continuacionC1RevalidadoM38}:      true,
		{continuacionC0RecibidoM38, continuacionC7RetirandoM38}:       true,
		{continuacionC0RecibidoM38, continuacionCFFatalM38}:           true,
		{continuacionC1RevalidadoM38, continuacionC2ContIntentadoM38}: true,
		{continuacionC1RevalidadoM38, continuacionCFFatalM38}:         true,
		{continuacionC2ContIntentadoM38, continuacionC3ObservadoM38}:  true,
		{continuacionC2ContIntentadoM38, continuacionCFFatalM38}:      true,
		{continuacionC3ObservadoM38, continuacionC4TTransfiriendoM38}: true,
		{continuacionC3ObservadoM38, continuacionCFFatalM38}:          true,
		{continuacionC4TTransfiriendoM38, continuacionC5EntregadoM38}: true,
		{continuacionC4TTransfiriendoM38, continuacionCFFatalM38}:     true,
		{continuacionC7RetirandoM38, continuacionC8RetiradoM38}:       true,
		{continuacionC7RetirandoM38, continuacionCFFatalM38}:          true,
	}
	for desde := continuacionC0RecibidoM38; desde <= continuacionCFFatalM38; desde++ {
		for hacia := continuacionC0RecibidoM38; hacia <= continuacionCFFatalM38; hacia++ {
			if transicionContinuacionO3cM38(desde, hacia) != verdes[[2]estadoContinuacionO3cM38{desde, hacia}] {
				t.Fatalf("transición %d→%d incorrecta", desde, hacia)
			}
		}
	}
}

func TestAutoridadO3cRechazosFatales(t *testing.T) {
	caso := os.Getenv("O3C_P1_FATAL")
	if caso != "" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		origen := agregadoNominalO3cPruebaM38(t)
		switch caso {
		case "auto":
			origen.custodia.lease.auto = &leaseGuardiaO3aM38{}
			_, _ = consumirAutoridadO3cM38(&origen)
		case "auto_consumido":
			origen.custodia.lease.estado.Store(3)
			origen.custodia.observador.palabra.Store(2)
			origen.custodia.baselineSenal = 2
			origen.custodia.lease.auto = &leaseGuardiaO3aM38{}
			_, _ = consumirAutoridadO3cM38(&origen)
		case "parcial":
			a := nuevaAutoridadCustodiaO3cM38()
			origen.custodia.lease.estado.Store(4)
			_ = consolidarEntradaO3cM38(origen.custodia, a)
		case "posterior":
			origen.custodia.terminal = nil
			_, _ = consumirAutoridadO3cM38(&origen)
		case "senal":
			origen.custodia.baselineSenal = 1 | 1<<2
			origen.custodia.observador.palabra.Store(origen.custodia.baselineSenal)
			_, _ = consumirAutoridadO3cM38(&origen)
		default:
			os.Exit(3)
		}
		os.Exit(99)
	}
	for _, nombre := range []string{"auto", "auto_consumido", "parcial", "posterior", "senal"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestAutoridadO3cRechazosFatales$")
		cmd.Env = append(os.Environ(), "O3C_P1_FATAL="+nombre)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		var salida *exec.ExitError
		if !errors.As(err, &salida) || salida.ExitCode() != estadoFallo || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("fatal %s no fue 65/0/0: err=%v out=%d errout=%d", nombre, err, stdout.Len(), stderr.Len())
		}
	}
}

func TestAutoridadO3cNoObservaCustodiaAntesDeLosCAS(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta de prueba ausente")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_autoridad.go")
	fichero, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	orden := make([]string, 0, 2)
	selectoresPreCAS := make([]string, 0, 16)
	for _, declaracion := range fichero.Decls {
		fn, ok := declaracion.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if fn.Name.Name == "clasificarEntradaO3cM38" {
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				selector, ok := n.(*ast.SelectorExpr)
				if ok {
					selectoresPreCAS = append(selectoresPreCAS, selector.Sel.Name)
				}
				return true
			})
			continue
		}
		if fn.Name.Name != "consumirAutoridadO3cM38" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			llamada, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := llamada.Fun.(*ast.Ident)
			if ok && (id.Name == "consolidarEntradaO3cM38" || id.Name == "custodiaConsumidaValidaO3cM38") {
				orden = append(orden, id.Name)
			}
			return true
		})
	}
	if len(orden) != 2 || orden[0] != "consolidarEntradaO3cM38" || orden[1] != "custodiaConsumidaValidaO3cM38" {
		t.Fatalf("orden CAS→custodia inválido: %v", orden)
	}
	permitidos := map[string]bool{
		"estado": true, "custodia": true, "lease": true, "observador": true,
		"Load": true, "palabra": true, "auto": true, "registro": true,
		"tid": true, "generacion": true, "leases": true, "observadores": true,
		"baselineSenal": true,
	}
	for _, selector := range selectoresPreCAS {
		if !permitidos[selector] {
			t.Fatalf("campo no autoritativo observado antes de CAS: %s", selector)
		}
	}
}

func TestAutoridadO3cSinEfectosPosteriores(t *testing.T) {
	// Los sellos de fases futuras permanecen vacíos y no hay método de efecto.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	origen := agregadoNominalO3cPruebaM38(t)
	a, err := consumirAutoridadO3cM38(&origen)
	if err != nil || !a.salida.ahoraCaso.IsZero() || !a.salida.finCaso.IsZero() ||
		a.salida.retornoCont != 0 || a.salida.primera.Load() != 0 {
		t.Fatalf("P1 abrió efectos posteriores: %v", err)
	}
}

func TestAutoridadO3cConsumeHandoffO3bReal(t *testing.T) {
	if os.Getenv("O3C_P1_REAL") == "1" {
		identidad := identidadRealHandoffO3bPruebaM38(t)
		agregado, err := transferirCapturadoO3bM38(&identidad)
		if err != nil || agregado == nil {
			os.Exit(2)
		}
		a, err := consumirAutoridadO3cM38(&agregado)
		if err != nil || agregado != nil || !a.es(continuacionC0RecibidoM38) ||
			a.custodia.lease.estado.Load() != 3 || a.custodia.observador.palabra.Load()&mascaraEstadoObservadorO3aM38 != 2 {
			os.Exit(4)
		}
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAutoridadO3cConsumeHandoffO3bReal$")
	cmd.Env = append(os.Environ(), "O3C_P1_REAL=1")
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("handoff O3b real rechazado: %v: %s", err, salida)
	}
}
