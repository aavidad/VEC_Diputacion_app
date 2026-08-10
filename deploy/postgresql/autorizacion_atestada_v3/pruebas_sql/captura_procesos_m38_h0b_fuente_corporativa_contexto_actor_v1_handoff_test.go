//go:build ignore && linux && amd64

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func identidadRealHandoffO3bPruebaM38(t *testing.T) *identidadProcesoO3bM38 {
	t.Helper()
	_, a := autoridadRealBarreraO3bPruebaM38(t)
	a.custodia.control.recepcion.sobre.ticket = ticketRealStopO3bPruebaM38(a)
	if ejecutarBarreraO3bM38(a) != nil || emitirYCerrarTicketO3bM38(a) != nil || observarStopO3bM38(a) != nil {
		t.Fatal("preparar B3")
	}
	identidad, err := acreditarIdentidadO3bM38(a)
	if err != nil || identidad == nil {
		t.Fatalf("preparar B4: %v", err)
	}
	return identidad
}

func TestHandoffO3bNominalConjuntoYConsumido(t *testing.T) {
	if os.Getenv("O3B_P6_HIJO") == "1" {
		identidad := identidadRealHandoffO3bPruebaM38(t)
		a := identidad.autoridad
		cmd, proceso := a.custodia.cmd, a.custodia.cmd.Process
		control, terminal := a.custodia.controlFD, a.custodia.terminal
		primario, reserva, opaco := a.custodia.pidfdPrimario, a.custodia.pidfdReserva, a.custodia.pidfdOpaco
		agregado, err := transferirCapturadoO3bM38(&identidad)
		if err != nil || identidad != nil || agregado == nil || agregado.estado != capturaB5CapturadoM38 || agregado.primera != nil ||
			agregado.custodia == nil || agregado.custodia.cmd != cmd || agregado.custodia.cmd.Process != proceso ||
			agregado.custodia.controlFD != control || agregado.custodia.terminal != terminal || agregado.custodia.ticketEscritor != nil ||
			agregado.custodia.pidfdPrimario != primario || agregado.custodia.pidfdReserva != reserva || agregado.custodia.pidfdOpaco != opaco ||
			agregado.custodia.lease.estado.Load() != 1 || agregado.custodia.baselineSenal&mascaraEstadoObservadorO3aM38 != 1 ||
			a.estado != capturaB5CapturadoM38 || a.custodia != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHandoffO3bNominalConjuntoYConsumido$")
	cmd.Env = append(os.Environ(), "O3B_P6_HIJO=1")
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("handoff nominal: %v: %s", err, salida)
	}
}

func TestHandoffO3bNegativosNoTransfieren(t *testing.T) {
	casoHijo := os.Getenv("O3B_P6_NEGATIVO")
	if casoHijo != "" {
		identidad := identidadRealHandoffO3bPruebaM38(t)
		a := identidad.autoridad
		switch casoHijo {
		case "contador":
			a.custodia.observador.palabra.Add(1 << 10)
		case "bootstrap":
			a.custodia.finBootstrap = time.Now()
		case "terminal":
			a.custodia.terminal = a.custodia.controlFD
		default:
			os.Exit(3)
		}
		agregado, err := transferirCapturadoO3bM38(&identidad)
		if err == nil || agregado != nil || identidad != nil || a.estado != capturaB7RetirandoM38 || a.custodia == nil ||
			a.custodia.lease.estado.Load() != 3 || a.custodia.baselineSenal&mascaraEstadoObservadorO3aM38 != 2 {
			os.Exit(4)
		}
		os.Exit(0)
	}
	for _, caso := range []string{"contador", "bootstrap", "terminal"} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHandoffO3bNegativosNoTransfieren$")
		cmd.Env = append(os.Environ(), "O3B_P6_NEGATIVO="+caso)
		if salida, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("negativo %s: %v: %s", caso, err, salida)
		}
	}
}

func TestHandoffO3bParticionIrreversibleEsFatal(t *testing.T) {
	if os.Getenv("O3B_P6_FATAL") == "1" {
		identidad := identidadRealHandoffO3bPruebaM38(t)
		a := identidad.autoridad
		salida := &agregadoO3cM38{}
		a.custodia.lease.estado.Store(4)
		_ = consolidarHandoffO3bM38(a, identidad, salida)
		os.Exit(3)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHandoffO3bParticionIrreversibleEsFatal$")
	cmd.Env = append(os.Environ(), "O3B_P6_FATAL=1")
	err := cmd.Run()
	if salida, ok := err.(*exec.ExitError); !ok || salida.ExitCode() != 65 {
		t.Fatalf("partición no fatal: %v", err)
	}
}

func TestHandoffO3bOrdenaObservadorAntesDeLease(t *testing.T) {
	_, prueba, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("ruta de prueba ausente")
	}
	ruta := filepath.Join(filepath.Dir(prueba), "captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go")
	fichero, err := parser.ParseFile(token.NewFileSet(), ruta, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	orden := make([]string, 0, 2)
	for _, declaracion := range fichero.Decls {
		funcion, ok := declaracion.(*ast.FuncDecl)
		if !ok || funcion.Name.Name != "consolidarHandoffO3bM38" {
			continue
		}
		ast.Inspect(funcion.Body, func(n ast.Node) bool {
			llamada, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			identificador, ok := llamada.Fun.(*ast.Ident)
			if ok && (identificador.Name == "transferirObservadorCapturadoO3bM38" || identificador.Name == "transferirLeaseCapturadaO3bM38") {
				orden = append(orden, identificador.Name)
			}
			return true
		})
	}
	if len(orden) != 2 || orden[0] != "transferirObservadorCapturadoO3bM38" || orden[1] != "transferirLeaseCapturadaO3bM38" {
		t.Fatalf("orden productivo inválido: %v", orden)
	}
}

func TestHandoffO3bLeasePerdidaEsFatalPorAPI(t *testing.T) {
	if os.Getenv("O3B_P6_LEASE_FATAL") == "1" {
		identidad := identidadRealHandoffO3bPruebaM38(t)
		identidad.autoridad.custodia.lease.estado.Store(4)
		_, _ = transferirCapturadoO3bM38(&identidad)
		os.Exit(3)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHandoffO3bLeasePerdidaEsFatalPorAPI$")
	cmd.Env = append(os.Environ(), "O3B_P6_LEASE_FATAL=1")
	err := cmd.Run()
	if salida, ok := err.(*exec.ExitError); !ok || salida.ExitCode() != 65 {
		t.Fatalf("lease perdida no fue fatal: %v", err)
	}
}

func TestHandoffO3bInventarioPreservaLeaseFatal(t *testing.T) {
	if os.Getenv("O3B_P6_INVENTARIO_LEASE") == "1" {
		identidad := identidadRealHandoffO3bPruebaM38(t)
		identidad.autoridad.custodia.lease.estado.Store(4)
		if err := inventarioPostTicketO3bM38(identidad.autoridad); err != errLeaseBarreraO3bM38 {
			os.Exit(2)
		}
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestHandoffO3bInventarioPreservaLeaseFatal$")
	cmd.Env = append(os.Environ(), "O3B_P6_INVENTARIO_LEASE=1")
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("inventario ocultó lease: %v: %s", err, salida)
	}
}
