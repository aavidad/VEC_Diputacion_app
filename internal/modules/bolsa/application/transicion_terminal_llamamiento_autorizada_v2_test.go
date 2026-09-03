package application

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

func TestTransicionTerminalAutorizadaV2DerivaLasTresTerminales(t *testing.T) {
	estados := []dominiobolsa.EstadoLlamamiento{
		dominiobolsa.EstadoLlamamientoAceptado,
		dominiobolsa.EstadoLlamamientoRenunciado,
		dominiobolsa.EstadoLlamamientoExpirado,
	}
	for _, estado := range estados {
		t.Run(string(estado), func(t *testing.T) {
			escenario := nuevoEscenarioOrdenTerminalPrueba(t, estado)
			original := escenario.llamamiento
			orden, err := escenario.emitir(context.Background())
			if err != nil {
				t.Fatalf("emitir orden PRE-CAP: %v", err)
			}

			derivado, err := TransicionarLlamamientoConOrdenTerminalAutorizadaV2(orden)
			if err != nil {
				t.Fatalf("derivar terminal %q: %v", estado, err)
			}
			terminal, existe := derivado.Terminal()
			if derivado.Validar() != nil || !derivado.EsTerminal() ||
				derivado.Estado() != estado || !existe || terminal != escenario.terminal ||
				derivado.Datos().Version != escenario.version+1 {
				t.Fatalf("derivacion incoherente: estado=%q terminal=%+v version=%d", derivado.Estado(), terminal, derivado.Datos().Version)
			}
			if escenario.llamamiento != original || escenario.llamamiento.EsTerminal() ||
				escenario.llamamiento.Estado() != dominiobolsa.EstadoLlamamientoAbierto ||
				escenario.llamamiento.Datos().Version != escenario.version {
				t.Fatalf("el agregado original fue alterado: %+v", escenario.llamamiento.Datos())
			}
			proyeccion, err := orden.ReacreditarYProyectar()
			agregadoOrden, versionOrden, terminalOrden, errDatos := proyeccion.Datos()
			if err != nil || errDatos != nil || agregadoOrden != original ||
				versionOrden != escenario.version || terminalOrden != escenario.terminal {
				t.Fatalf("la orden cambio tras derivarla: errores=%v/%v", err, errDatos)
			}
		})
	}
}

func TestTransicionTerminalAutorizadaV2ReplayExactoNoIncrementaDosVeces(t *testing.T) {
	escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoRenunciado)
	orden, err := escenario.emitir(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	primero, errPrimero := TransicionarLlamamientoConOrdenTerminalAutorizadaV2(orden)
	segundo, errSegundo := TransicionarLlamamientoConOrdenTerminalAutorizadaV2(orden)
	if errPrimero != nil || errSegundo != nil || primero != segundo {
		t.Fatalf("replay divergente: errores=%v/%v valores=%+v/%+v", errPrimero, errSegundo, primero.Datos(), segundo.Datos())
	}
	if primero.Datos().Version != escenario.version+1 {
		t.Fatalf("replay incremento dos veces la version: %d", primero.Datos().Version)
	}
	terminal, existe := primero.Terminal()
	if !existe || terminal != escenario.terminal {
		t.Fatalf("replay perdio la operacion ligada: %+v / %t", terminal, existe)
	}
}

func TestTransicionTerminalAutorizadaV2FallaOpacaConOrdenInvalida(t *testing.T) {
	escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoAceptado)
	orden, err := escenario.emitir(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	corrupta := orden
	datosCorruptos := *orden.datos
	datosCorruptos.versionEsperada++
	corrupta.datos = &datosCorruptos

	casos := []struct {
		nombre string
		orden  OrdenTerminalLlamamientoAutorizadaV2
	}{
		{"valor_cero", OrdenTerminalLlamamientoAutorizadaV2{}},
		{"estructura_corrupta", corrupta},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			derivado, err := TransicionarLlamamientoConOrdenTerminalAutorizadaV2(caso.orden)
			if derivado != (dominiobolsa.LlamamientoAbierto{}) ||
				!errors.Is(err, ErrOrdenTerminalLlamamientoInvalida) ||
				err != ErrOrdenTerminalLlamamientoInvalida {
				t.Fatalf("entrada invalida no fallo cerrada: derivado=%+v error=%v", derivado, err)
			}
			if strings.Contains(err.Error(), escenario.terminal.OperacionRef) ||
				strings.Contains(err.Error(), escenario.llamamiento.Datos().LlamamientoRef) {
				t.Fatalf("error filtro referencias: %v", err)
			}
		})
	}
}

func TestTransicionTerminalAutorizadaV2RechazaProyeccionCorruptaODivergente(t *testing.T) {
	escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoExpirado)
	orden, err := escenario.emitir(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	valida, err := orden.ReacreditarYProyectar()
	if err != nil {
		t.Fatal(err)
	}

	corrupta := valida
	corrupta.sello[0] ^= 0xff
	versionDivergente := valida
	versionDivergente.versionEsperada++
	versionDivergente.sello = selloProyeccionOrdenTerminal(
		versionDivergente.llamamiento, versionDivergente.versionEsperada,
		versionDivergente.terminal,
	)
	terminalInvalida := valida
	terminalInvalida.terminal = dominiobolsa.TerminalLlamamiento{
		Estado: dominiobolsa.EstadoLlamamientoAbierto, OperacionRef: "operacion:01K3INVALIDA",
	}
	terminalInvalida.sello = selloProyeccionOrdenTerminal(
		terminalInvalida.llamamiento, terminalInvalida.versionEsperada,
		terminalInvalida.terminal,
	)

	for nombre, proyeccion := range map[string]ProyeccionOrdenTerminalLlamamientoAutorizadaV2{
		"sello_corrupto":         corrupta,
		"version_divergente":     versionDivergente,
		"transicion_no_terminal": terminalInvalida,
	} {
		t.Run(nombre, func(t *testing.T) {
			derivado, err := transicionarLlamamientoProyectadoAutorizadoV2(proyeccion)
			if derivado != (dominiobolsa.LlamamientoAbierto{}) ||
				err != ErrOrdenTerminalLlamamientoInvalida {
				t.Fatalf("proyeccion invalida produjo estado: %+v / %v", derivado, err)
			}
		})
	}
}

func TestTransicionTerminalAutorizadaV2DerivaConcurrentementeSinMutacion(t *testing.T) {
	escenario := nuevoEscenarioOrdenTerminalPrueba(t, dominiobolsa.EstadoLlamamientoAceptado)
	orden, err := escenario.emitir(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	original := escenario.llamamiento
	const trabajadores = 64
	resultados := make(chan dominiobolsa.LlamamientoAbierto, trabajadores)
	errores := make(chan error, trabajadores)
	var grupo sync.WaitGroup
	grupo.Add(trabajadores)
	for i := 0; i < trabajadores; i++ {
		go func() {
			defer grupo.Done()
			derivado, err := TransicionarLlamamientoConOrdenTerminalAutorizadaV2(orden)
			if err != nil {
				errores <- err
				return
			}
			resultados <- derivado
		}()
	}
	grupo.Wait()
	close(resultados)
	close(errores)
	for err := range errores {
		t.Fatalf("derivacion concurrente: %v", err)
	}
	var esperado dominiobolsa.LlamamientoAbierto
	for derivado := range resultados {
		if esperado == (dominiobolsa.LlamamientoAbierto{}) {
			esperado = derivado
		}
		if derivado != esperado || derivado.Datos().Version != escenario.version+1 {
			t.Fatalf("derivaciones concurrentes divergentes: %+v / %+v", esperado.Datos(), derivado.Datos())
		}
	}
	if escenario.llamamiento != original || escenario.llamamiento.EsTerminal() ||
		orden.Validar() != nil {
		t.Fatal("la derivacion concurrente muto su entrada")
	}
}

func TestTransicionTerminalAutorizadaV2SuperficiePuraYMinima(t *testing.T) {
	contenido, err := os.ReadFile("transicion_terminal_llamamiento_autorizada_v2.go")
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)
	if strings.Count(texto, ".TransicionarATerminal(") != 1 {
		t.Fatalf("numero de transiciones B2 distinto de uno")
	}
	for _, requerido := range []string{"orden.ReacreditarYProyectar()", "proyeccion.Datos()"} {
		if !strings.Contains(texto, requerido) {
			t.Fatalf("falta consumo PRE-CAP %q", requerido)
		}
	}
	for _, prohibido := range []string{
		"ContextoActor", "VinculoAutenticacion", "PoliticaUso", "EvidenciaUso",
		"DecisionAutorizacion", "Fachada", "Exigidor", "Candidato", "Sucesor",
		"time.", "http.", "sql.", "persist", "auditor", "outbox",
	} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("produccion contiene superficie prohibida %q", prohibido)
		}
	}

	tipoFuncion := reflect.TypeOf(TransicionarLlamamientoConOrdenTerminalAutorizadaV2)
	if tipoFuncion.NumIn() != 1 ||
		tipoFuncion.In(0) != reflect.TypeOf(OrdenTerminalLlamamientoAutorizadaV2{}) ||
		tipoFuncion.NumOut() != 2 ||
		tipoFuncion.Out(0) != reflect.TypeOf(dominiobolsa.LlamamientoAbierto{}) ||
		!tipoFuncion.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		t.Fatalf("firma publica inesperada: %v", tipoFuncion)
	}
	for _, tipo := range []reflect.Type{
		reflect.TypeOf(TransicionarLlamamientoConOrdenTerminalAutorizadaV2),
		reflect.TypeOf(transicionarLlamamientoProyectadoAutorizadoV2),
	} {
		for indice := 0; indice < tipo.NumIn(); indice++ {
			if tipo.In(indice).Kind() == reflect.String {
				t.Fatalf("la transicion acepta selector libre: %v", tipo)
			}
		}
	}
}
