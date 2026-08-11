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
	"syscall"
	"testing"
	"time"
)

func revalidacionRealO3cP3PruebaM38(t *testing.T) *revalidacionO3cM38 {
	t.Helper()
	a := autoridadRealO3cP2PruebaM38(t)
	r, err := revalidarAntesContO3cM38(a)
	if err != nil || r == nil || !a.es(continuacionC1RevalidadoM38) {
		t.Fatalf("preparar C1: %v", err)
	}
	return r
}

func ejecutarContAisladoO3cPruebaM38(t *testing.T, caso string, estado int) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestContO3cCasosAislados$")
	cmd.Env = append(os.Environ(), "O3C_P3_CASO="+caso)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if estado == 0 {
		if err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
			t.Fatalf("caso %s: err=%v stdout=%d stderr=%d", caso, err, stdout.Len(), stderr.Len())
		}
		return
	}
	var salida *exec.ExitError
	if !errors.As(err, &salida) || salida.ExitCode() != estado || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("caso fatal %s: err=%v stdout=%d stderr=%d", caso, err, stdout.Len(), stderr.Len())
	}
}

func TestContO3cCasosAislados(t *testing.T) {
	caso := os.Getenv("O3C_P3_CASO")
	if caso == "" {
		ejecutarContAisladoO3cPruebaM38(t, "nominal", 0)
		ejecutarContAisladoO3cPruebaM38(t, "error", 0)
		ejecutarContAisladoO3cPruebaM38(t, "reuso", estadoFallo)
		ejecutarContAisladoO3cPruebaM38(t, "permiso", estadoFallo)
		return
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	r := revalidacionRealO3cP3PruebaM38(t)
	alias := r
	if caso == "error" {
		_ = syscall.Close(r.autoridad.custodia.pidfdPrimario)
	}
	if caso == "permiso" {
		r.permiso.generacion++
		_ = intentarContO3cM38(&r)
		os.Exit(99)
	}
	a := intentarContO3cM38(&r)
	if caso == "reuso" {
		_ = intentarContO3cM38(&alias)
		os.Exit(99)
	}
	if r != nil || a == nil || !a.es(continuacionC2ContIntentadoM38) || alias.auto != nil ||
		a.custodia.lease.estado.Load() != 3 || a.salida.ahoraCaso.IsZero() || a.salida.finCaso.IsZero() ||
		!tiempoMonotonoO3cM38(a.salida.ahoraCaso) || !tiempoMonotonoO3cM38(a.salida.finCaso) ||
		a.salida.finCaso.Sub(a.salida.ahoraCaso) != duracionCasoO3cM38 {
		os.Exit(12)
	}
	if caso == "nominal" && a.salida.retornoCont != 0 {
		os.Exit(13)
	}
	if caso == "error" && a.salida.retornoCont != int(syscall.EBADF) {
		os.Exit(14)
	}
	os.Exit(0)
}

func TestContO3cMarcaRechazaNoMonotonoYOverflow(t *testing.T) {
	if pidfdSignalGrupoO3cM38 != uintptr(1<<2) {
		t.Fatalf("flag de grupo mutado: %d", pidfdSignalGrupoO3cM38)
	}
	sinMonotono := time.Unix(1, 0)
	if _, ok := finCasoExactoO3cM38(sinMonotono); ok {
		t.Fatal("marca externa aceptada")
	}
	ahora := time.Now()
	fin, ok := finCasoExactoO3cM38(ahora)
	if !ok || fin.Sub(ahora) != 180*time.Second || !fin.After(ahora) {
		t.Fatal("relación de 180 s no acreditada")
	}
	// Add conserva/satura el wall clock; una fecha sin componente monotónico
	// sigue cerrada y cubre el borde sin inventar un reloj global inyectable.
	if _, ok := finCasoExactoO3cM38(time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)); ok {
		t.Fatal("borde de overflow aceptado")
	}
}

func TestContO3cOrdenLiteralYAusenciaP4(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta ausente")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "continuacion_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_cont.go")
	f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	orden := make([]string, 0, 4)
	syscalls := 0
	consolidacionFatal := false
	retornoRawGuardado := false
	prohibidas := map[string]bool{"Poll": true, "Wait": true, "Signal": true, "Kill": true, "Sleep": true, "Close": true}
	ast.Inspect(f, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		nombre := nombreLlamadaO3cPruebaM38(c)
		if nombre == "permisoValido" || nombre == "consolidarCritico" || nombre == "fatalPendiente" {
			t.Fatalf("primitiva con Gettid prohibida en P3: %s", nombre)
		}
		return true
	})
	for _, declaracion := range f.Decls {
		fn, ok := declaracion.(*ast.FuncDecl)
		if ok && (fn.Name.Name == "autoridadContValidaO3cM38" || fn.Name.Name == "permisoContMemoriaValidoO3cM38" || fn.Name.Name == "consolidarContO3cM38") {
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if selector, ok := c.Fun.(*ast.SelectorExpr); ok {
					if base, ok := selector.X.(*ast.Ident); ok && base.Name == "syscall" {
						t.Fatal("syscall anterior a la marca")
					}
				}
				return true
			})
		}
		if !ok || fn.Name.Name != "intentarContO3cM38" {
			continue
		}
		if !tramoCriticoAdyacenteO3cPruebaM38(fn.Body.List) {
			t.Fatal("Now, marca, validación, syscall y consolidación no son adyacentes")
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n.(type) {
			case *ast.ForStmt, *ast.RangeStmt:
				t.Fatal("CONT no puede reintentarse")
			}
			condicional, esCondicional := n.(*ast.IfStmt)
			if esCondicional {
				negacion, ok := condicional.Cond.(*ast.UnaryExpr)
				if ok && negacion.Op == token.NOT {
					llamada, esLlamada := negacion.X.(*ast.CallExpr)
					if esLlamada {
						if nombreLlamadaO3cPruebaM38(llamada) == "consolidarContO3cM38" && len(condicional.Body.List) == 1 {
							expresion, ok := condicional.Body.List[0].(*ast.ExprStmt)
							if ok {
								fatal, esFatal := expresion.X.(*ast.CallExpr)
								if esFatal {
									nombre, esNombre := fatal.Fun.(*ast.Ident)
									consolidacionFatal = esNombre && nombre.Name == "fatalContO3cM38"
								}
							}
						}
					}
				}
			}
			asignacion, esAsignacion := n.(*ast.AssignStmt)
			if esAsignacion && len(asignacion.Lhs) == 1 && len(asignacion.Rhs) == 1 {
				campo, esCampo := asignacion.Lhs[0].(*ast.SelectorExpr)
				conversion, esConversion := asignacion.Rhs[0].(*ast.CallExpr)
				if esCampo && campo.Sel.Name == "retornoCont" && esConversion && len(conversion.Args) == 1 {
					nombre, ok := conversion.Args[0].(*ast.Ident)
					retornoRawGuardado = ok && nombre.Name == "retornoRaw"
				}
			}
			llamada, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch llamada := llamada.Fun.(type) {
			case *ast.SelectorExpr:
				if prohibidas[llamada.Sel.Name] {
					t.Fatalf("API P4+ prohibida: %s", llamada.Sel.Name)
				}
				if llamada.Sel.Name == "Syscall6" {
					syscalls++
					if len(n.(*ast.CallExpr).Args) != 7 || !argumentosContExactosO3cPruebaM38(n.(*ast.CallExpr).Args) {
						t.Fatal("pidfd_send_signal no usa primario/SIGCONT/NULL/grupo exactos")
					}
				}
				if llamada.Sel.Name == "Now" || llamada.Sel.Name == "Syscall6" || llamada.Sel.Name == "consolidarCritico" {
					orden = append(orden, llamada.Sel.Name)
				}
			case *ast.Ident:
				if llamada.Name == "finCasoExactoO3cM38" || llamada.Name == "consolidarContO3cM38" {
					orden = append(orden, llamada.Name)
				}
			}
			return true
		})
	}
	esperado := []string{"Now", "finCasoExactoO3cM38", "Syscall6", "consolidarContO3cM38"}
	if len(orden) != len(esperado) {
		t.Fatalf("orden crítico incompleto: %v", orden)
	}
	if syscalls != 1 || !consolidacionFatal || !retornoRawGuardado {
		t.Fatalf("syscall único, fatalidad o retorno raw no acreditados: syscalls=%d fatal=%t raw=%t", syscalls, consolidacionFatal, retornoRawGuardado)
	}
	for i := range esperado {
		if orden[i] != esperado[i] {
			t.Fatalf("orden crítico inválido: %v", orden)
		}
	}
}

func tramoCriticoAdyacenteO3cPruebaM38(sentencias []ast.Stmt) bool {
	for i := 0; i+4 < len(sentencias); i++ {
		ahora, okAhora := sentencias[i].(*ast.AssignStmt)
		fin, okFin := sentencias[i+1].(*ast.AssignStmt)
		validacion, okValidacion := sentencias[i+2].(*ast.IfStmt)
		syscallStmt, okSyscall := sentencias[i+3].(*ast.AssignStmt)
		consolidacion, okConsolidacion := sentencias[i+4].(*ast.IfStmt)
		if !okAhora || !okFin || !okValidacion || !okSyscall || !okConsolidacion {
			continue
		}
		if !asignacionLlamaO3cPruebaM38(ahora, "Now") || !asignacionLlamaO3cPruebaM38(fin, "finCasoExactoO3cM38") ||
			!asignacionLlamaO3cPruebaM38(syscallStmt, "Syscall6") || !condicionLlamaNegadaO3cPruebaM38(consolidacion, "consolidarContO3cM38") {
			continue
		}
		llamadas := make([]string, 0, 3)
		ast.Inspect(validacion.Cond, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if ok {
				llamadas = append(llamadas, nombreLlamadaO3cPruebaM38(c))
			}
			return true
		})
		return len(llamadas) == 1 && llamadas[0] == "Before"
	}
	return false
}

func asignacionLlamaO3cPruebaM38(a *ast.AssignStmt, nombre string) bool {
	if len(a.Rhs) != 1 {
		return false
	}
	c, ok := a.Rhs[0].(*ast.CallExpr)
	return ok && nombreLlamadaO3cPruebaM38(c) == nombre
}

func condicionLlamaNegadaO3cPruebaM38(i *ast.IfStmt, nombre string) bool {
	negacion, ok := i.Cond.(*ast.UnaryExpr)
	if !ok || negacion.Op != token.NOT {
		return false
	}
	c, ok := negacion.X.(*ast.CallExpr)
	return ok && nombreLlamadaO3cPruebaM38(c) == nombre
}

func nombreLlamadaO3cPruebaM38(c *ast.CallExpr) string {
	switch f := c.Fun.(type) {
	case *ast.Ident:
		return f.Name
	case *ast.SelectorExpr:
		return f.Sel.Name
	default:
		return ""
	}
}

func argumentosContExactosO3cPruebaM38(args []ast.Expr) bool {
	ident := func(expr ast.Expr, nombre string) bool {
		n, ok := expr.(*ast.Ident)
		return ok && n.Name == nombre
	}
	selector := func(expr ast.Expr, base, campo string) bool {
		s, ok := expr.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		b, esBase := s.X.(*ast.Ident)
		return esBase && b.Name == base && s.Sel.Name == campo
	}
	conversion := func(expr ast.Expr, base, campo string) bool {
		c, ok := expr.(*ast.CallExpr)
		return ok && len(c.Args) == 1 && ident(c.Fun, "uintptr") && selector(c.Args[0], base, campo)
	}
	cero := func(expr ast.Expr) bool {
		literal, ok := expr.(*ast.BasicLit)
		return ok && literal.Kind == token.INT && literal.Value == "0"
	}
	return ident(args[0], "sysPidfdSendSignal") && conversion(args[1], "c", "pidfdPrimario") &&
		conversion(args[2], "syscall", "SIGCONT") && cero(args[3]) &&
		ident(args[4], "pidfdSignalGrupoO3cM38") && cero(args[5]) && cero(args[6])
}
