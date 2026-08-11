package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
)

func (a *analisis) validarCFyFrontera() error {
	for _, nombre := range []string{"fatalContO3cM38", "fatalObservacionO3cM38", "fatalHandoffO3cM38", "fatalRevalidacionO3cM38"} {
		if err := fatalExacto(a.funciones[nombre]); err != nil {
			return fmt.Errorf("C20 %s: %w", nombre, err)
		}
	}
	fd := a.funciones["fatalO3cM38"]
	if fd == nil || fd.Body == nil || len(fd.Body.List) != 1 || !bytes.Equal(nodo(a.fset, fd.Body.List[0]), []byte("fatalO3aM38()")) {
		return fmt.Errorf("C20 fatal raíz divergente")
	}
	llamadas := map[string][]string{}
	for caller, fd := range a.funciones {
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok {
				llamadas[nombreLlamada(c)] = append(llamadas[nombreLlamada(c)], caller)
			}
			return true
		})
	}
	if len(llamadas["retirarAntesContO3cM38"]) != 0 {
		return fmt.Errorf("C20 retirada local invocada post-CONT por %v", llamadas["retirarAntesContO3cM38"])
	}
	if len(llamadas["esperarConLeaseO3aM38"]) != 1 || llamadas["esperarConLeaseO3aM38"][0] != "retirarAntesContO3cM38" {
		return fmt.Errorf("C21 Wait fuera C7: %v", llamadas["esperarConLeaseO3aM38"])
	}
	if len(llamadas["cerrarRecursosRetiradaO3cM38"]) != 1 || llamadas["cerrarRecursosRetiradaO3cM38"][0] != "retirarAntesContO3cM38" {
		return fmt.Errorf("C21 cierre TERMINAL fuera C7: %v", llamadas["cerrarRecursosRetiradaO3cM38"])
	}
	if contarEnTodos(a.contenido, "syscall.Syscall6(") != 2 || contarEnTodos(a.contenido, "syscall.SIGCONT") != 1 || contarEnTodos(a.contenido, "syscall.SIGKILL") != 1 || contarEnTodos(a.contenido, "syscall.SIGTERM") != 1 {
		return fmt.Errorf("C21 cardinalidad CONT/KILL/PDEATHSIG divergente")
	}
	for n, f := range a.archivos {
		var prohibida string
		ast.Inspect(f, func(x ast.Node) bool {
			switch x.(type) {
			case *ast.GoStmt:
				prohibida = "goroutine"
			case *ast.ChanType:
				prohibida = "canal"
			}
			return prohibida == ""
		})
		if prohibida != "" {
			return fmt.Errorf("C21 %s en %s", prohibida, n)
		}
		for _, p := range []string{"waitid", "Waitid", "SYS_WAITID", "time.NewTimer", "time.After(", "log.", "fmt.", "println(", "print(", "os.Stdout", "os.Stderr", ".terminal.Write", ".terminal.Close", "syscall.Write"} {
			if bytes.Contains(a.contenido[n], []byte(p)) {
				return fmt.Errorf("C20/C21 API %s en %s", p, n)
			}
		}
	}
	return nil
}

func fatalExacto(fd *ast.FuncDecl) error {
	if fd == nil || fd.Body == nil || len(fd.Body.List) != 2 {
		return fmt.Errorf("cardinalidad sentencias")
	}
	si, ok := fd.Body.List[0].(*ast.IfStmt)
	if !ok || expresion(si.Cond) != "a != nil" || len(si.Body.List) != 1 || !bytes.Contains(nodo(token.NewFileSet(), si.Body.List[0]), []byte("a.estado = continuacionCFFatalM38")) {
		return fmt.Errorf("marca CF divergente")
	}
	expr, ok := fd.Body.List[1].(*ast.ExprStmt)
	call, callOK := expr.X.(*ast.CallExpr)
	if !ok || !callOK || nombreLlamada(call) != "fatalO3cM38" || len(call.Args) != 0 {
		return fmt.Errorf("efecto posterior a CF")
	}
	return nil
}
