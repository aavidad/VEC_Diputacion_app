package domain

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestLlamamientoAbiertoConstruyeValorValido(t *testing.T) {
	datos := datosLlamamientoAbiertoPrueba()
	llamamiento, err := NuevoLlamamientoAbierto(datos)
	if err != nil {
		t.Fatalf("crear llamamiento abierto: %v", err)
	}
	if err := llamamiento.Validar(); err != nil {
		t.Fatalf("validar llamamiento abierto: %v", err)
	}
	if llamamiento.Estado() != EstadoLlamamientoAbierto ||
		llamamiento.EsTerminal() {
		t.Fatalf("estado inicial inesperado: %q", llamamiento.Estado())
	}
	if terminal, existe := llamamiento.Terminal(); existe || terminal != (TerminalLlamamiento{}) {
		t.Fatalf("un llamamiento abierto expuso terminal: %+v / %t", terminal, existe)
	}
	if llamamiento.Datos() != datos {
		t.Fatalf("datos de apertura distintos: %+v", llamamiento.Datos())
	}
}

func TestLlamamientoAbiertoRechazaConstruccionesInvalidasYValorCero(t *testing.T) {
	base := datosLlamamientoAbiertoPrueba()
	casos := []struct {
		nombre string
		mutar  func(*DatosLlamamientoAbierto)
	}{
		{"llamamiento_vacio", func(d *DatosLlamamientoAbierto) { d.LlamamientoRef = "" }},
		{"bolsa_no_opaca", func(d *DatosLlamamientoAbierto) { d.BolsaRef = " bolsa:otra" }},
		{"necesidad_con_comodin", func(d *DatosLlamamientoAbierto) { d.NecesidadRef = "necesidad:*" }},
		{"propuesta_con_documento", func(d *DatosLlamamientoAbierto) { d.PropuestaRef = "propuesta:00000000A" }},
		{"version_cero", func(d *DatosLlamamientoAbierto) { d.Version = 0 }},
		{"version_sin_sucesora_segura", func(d *DatosLlamamientoAbierto) {
			d.Version = maximoEnteroSeguroVersionLlamamiento
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := base
			caso.mutar(&datos)
			obtenido, err := NuevoLlamamientoAbierto(datos)
			if !errors.Is(err, ErrLlamamientoAbiertoInvalido) {
				t.Fatalf("construccion invalida admitida: %v", err)
			}
			if obtenido != (LlamamientoAbierto{}) {
				t.Fatalf("error devolvio estado parcial: %+v", obtenido)
			}
		})
	}

	var cero LlamamientoAbierto
	if err := cero.Validar(); !errors.Is(err, ErrLlamamientoAbiertoInvalido) {
		t.Fatalf("valor cero admitido: %v", err)
	}
	if _, err := cero.TransicionarATerminal(
		1,
		&TerminalLlamamiento{Estado: EstadoLlamamientoAceptado, OperacionRef: "operacion:terminal"},
	); !errors.Is(err, ErrLlamamientoAbiertoInvalido) {
		t.Fatalf("valor cero transicionable: %v", err)
	}
}

func TestLlamamientoAbiertoAdmiteTerminalesExactos(t *testing.T) {
	casos := []EstadoLlamamiento{
		EstadoLlamamientoAceptado,
		EstadoLlamamientoRenunciado,
		EstadoLlamamientoExpirado,
	}
	for indice, estado := range casos {
		t.Run(string(estado), func(t *testing.T) {
			base := llamamientoAbiertoPrueba(t)
			terminal := TerminalLlamamiento{
				Estado:       estado,
				OperacionRef: "operacion:terminal:" + string(rune('a'+indice)),
			}
			cerrado, err := base.TransicionarATerminal(base.Datos().Version, &terminal)
			if err != nil {
				t.Fatalf("transicion terminal %q: %v", estado, err)
			}
			if cerrado.Estado() != estado || !cerrado.EsTerminal() ||
				cerrado.Datos().Version != base.Datos().Version+1 {
				t.Fatalf("terminal incoherente: estado=%q datos=%+v", cerrado.Estado(), cerrado.Datos())
			}
			obtenido, existe := cerrado.Terminal()
			if !existe || obtenido != terminal {
				t.Fatalf("hecho terminal distinto: %+v / %t", obtenido, existe)
			}
		})
	}
}

func TestLlamamientoAbiertoAplicaCASAntesDeEfectos(t *testing.T) {
	base := llamamientoAbiertoPrueba(t)
	antes := base
	terminal := terminalLlamamientoPrueba(EstadoLlamamientoAceptado)
	for _, version := range []uint64{0, base.Datos().Version - 1, base.Datos().Version + 1} {
		resultado, err := base.TransicionarATerminal(version, &terminal)
		if !errors.Is(err, ErrVersionLlamamientoEnConflicto) {
			t.Fatalf("version %d no produjo conflicto: %v", version, err)
		}
		if resultado != (LlamamientoAbierto{}) || base != antes ||
			base.Estado() != EstadoLlamamientoAbierto {
			t.Fatalf("conflicto de version produjo efectos: %+v / %+v", resultado, base)
		}
	}
}

func TestLlamamientoAbiertoRepiteTerminalExactoSinNuevaVersion(t *testing.T) {
	base := llamamientoAbiertoPrueba(t)
	versionEsperada := base.Datos().Version
	terminal := terminalLlamamientoPrueba(EstadoLlamamientoRenunciado)
	primero, err := base.TransicionarATerminal(versionEsperada, &terminal)
	if err != nil {
		t.Fatal(err)
	}
	terminalEntrada := terminal
	repetido, err := primero.TransicionarATerminal(versionEsperada, &terminalEntrada)
	if err != nil {
		t.Fatalf("replay exacto rechazado: %v", err)
	}
	if repetido != primero || repetido.Datos().Version != versionEsperada+1 {
		t.Fatalf("replay creo otro valor: %+v / %+v", primero, repetido)
	}

	terminalEntrada.OperacionRef = "operacion:mutada-despues"
	obtenido, _ := primero.Terminal()
	if obtenido != terminal {
		t.Fatalf("el agregado comparte el puntero de entrada: %+v", obtenido)
	}
	datos := primero.Datos()
	datos.LlamamientoRef = "llamamiento:mutado-despues"
	if primero.Datos().LlamamientoRef == datos.LlamamientoRef {
		t.Fatal("Datos comparte estado mutable con el agregado")
	}
}

func TestLlamamientoAbiertoRechazaTerminalIncompatible(t *testing.T) {
	base := llamamientoAbiertoPrueba(t)
	versionEsperada := base.Datos().Version
	terminal := terminalLlamamientoPrueba(EstadoLlamamientoAceptado)
	cerrado, err := base.TransicionarATerminal(versionEsperada, &terminal)
	if err != nil {
		t.Fatal(err)
	}
	antes := cerrado
	casos := []TerminalLlamamiento{
		{Estado: EstadoLlamamientoRenunciado, OperacionRef: terminal.OperacionRef},
		{Estado: EstadoLlamamientoAceptado, OperacionRef: "operacion:terminal-distinta"},
	}
	for _, incompatible := range casos {
		resultado, err := cerrado.TransicionarATerminal(versionEsperada, &incompatible)
		if !errors.Is(err, ErrTerminalLlamamientoIncompatible) {
			t.Fatalf("terminal incompatible admitido: %+v / %v", incompatible, err)
		}
		if resultado != (LlamamientoAbierto{}) || cerrado != antes {
			t.Fatalf("terminal incompatible produjo efectos: %+v", resultado)
		}
	}
	if _, err := cerrado.TransicionarATerminal(cerrado.Datos().Version, &terminal); !errors.Is(err, ErrVersionLlamamientoEnConflicto) {
		t.Fatalf("replay con CAS distinto admitido: %v", err)
	}
}

func TestLlamamientoAbiertoRechazaTerminalNuloOCorrupto(t *testing.T) {
	base := llamamientoAbiertoPrueba(t)
	casos := []*TerminalLlamamiento{
		nil,
		{},
		{Estado: EstadoLlamamientoAbierto, OperacionRef: "operacion:no-terminal"},
		{Estado: EstadoLlamamiento("inventado"), OperacionRef: "operacion:inventada"},
		{Estado: EstadoLlamamientoAceptado, OperacionRef: "operacion:*"},
		{Estado: EstadoLlamamientoRenunciado, OperacionRef: "operacion:X0000000A"},
	}
	for _, terminal := range casos {
		resultado, err := base.TransicionarATerminal(base.Datos().Version, terminal)
		if !errors.Is(err, ErrTerminalLlamamientoInvalido) {
			t.Fatalf("terminal invalido admitido: %+v / %v", terminal, err)
		}
		if resultado != (LlamamientoAbierto{}) || base.EsTerminal() {
			t.Fatalf("terminal invalido produjo efectos: %+v", resultado)
		}
	}
}

func TestLlamamientoAbiertoRespetaBordeDeVersionSegura(t *testing.T) {
	datos := datosLlamamientoAbiertoPrueba()
	datos.Version = maximoEnteroSeguroVersionLlamamiento - 1
	base, err := NuevoLlamamientoAbierto(datos)
	if err != nil {
		t.Fatalf("penultima version segura rechazada: %v", err)
	}
	terminal := terminalLlamamientoPrueba(EstadoLlamamientoExpirado)
	cerrado, err := base.TransicionarATerminal(datos.Version, &terminal)
	if err != nil {
		t.Fatalf("ultima transicion segura rechazada: %v", err)
	}
	if cerrado.Datos().Version != maximoEnteroSeguroVersionLlamamiento ||
		cerrado.Validar() != nil {
		t.Fatalf("borde terminal inseguro: %+v", cerrado.Datos())
	}
}

func TestLlamamientoAbiertoEsValorPuroEnDerivacionesConcurrentes(t *testing.T) {
	base := llamamientoAbiertoPrueba(t)
	antes := base
	const trabajadores = 64
	var grupo sync.WaitGroup
	errores := make(chan error, trabajadores)
	grupo.Add(trabajadores)
	for indice := 0; indice < trabajadores; indice++ {
		indice := indice
		go func() {
			defer grupo.Done()
			estado := EstadoLlamamientoAceptado
			if indice%2 == 1 {
				estado = EstadoLlamamientoRenunciado
			}
			terminal := TerminalLlamamiento{
				Estado:       estado,
				OperacionRef: "operacion:concurrente:" + cadenaDecimalLlamamiento(indice),
			}
			derivado, err := base.TransicionarATerminal(base.Datos().Version, &terminal)
			if err != nil {
				errores <- err
				return
			}
			obtenido, existe := derivado.Terminal()
			if !existe || obtenido != terminal ||
				derivado.Datos().Version != base.Datos().Version+1 {
				errores <- ErrLlamamientoAbiertoInvalido
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("derivacion concurrente: %v", err)
	}
	if base != antes || base.Estado() != EstadoLlamamientoAbierto || base.EsTerminal() {
		t.Fatal("las derivaciones concurrentes mutaron el valor compartido")
	}
}

func TestLlamamientoAbiertoNoExponeCamposPersonalesNiSeleccion(t *testing.T) {
	for _, tipo := range []reflect.Type{
		reflect.TypeOf(DatosLlamamientoAbierto{}),
		reflect.TypeOf(TerminalLlamamiento{}),
	} {
		for indice := 0; indice < tipo.NumField(); indice++ {
			nombre := strings.ToLower(tipo.Field(indice).Name)
			for _, prohibido := range []string{
				"dni", "nie", "nombre", "correo", "telefono", "direccion",
				"persona", "sujeto", "participacion", "candidato", "seleccion", "sucesor",
			} {
				if strings.Contains(nombre, prohibido) {
					t.Fatalf("%s expone campo prohibido %s", tipo.Name(), nombre)
				}
			}
		}
	}
}

func TestErroresLlamamientoAbiertoSonOpacos(t *testing.T) {
	datos := datosLlamamientoAbiertoPrueba()
	datos.LlamamientoRef = "llamamiento:00000000A"
	_, err := NuevoLlamamientoAbierto(datos)
	if !errors.Is(err, ErrLlamamientoAbiertoInvalido) ||
		strings.Contains(err.Error(), datos.LlamamientoRef) {
		t.Fatalf("error de construccion filtra entrada: %v", err)
	}

	base := llamamientoAbiertoPrueba(t)
	terminal := TerminalLlamamiento{
		Estado:       EstadoLlamamientoAceptado,
		OperacionRef: "operacion:00000000A",
	}
	_, err = base.TransicionarATerminal(base.Datos().Version, &terminal)
	if !errors.Is(err, ErrTerminalLlamamientoInvalido) ||
		strings.Contains(err.Error(), terminal.OperacionRef) {
		t.Fatalf("error terminal filtra entrada: %v", err)
	}
}

func datosLlamamientoAbiertoPrueba() DatosLlamamientoAbierto {
	return DatosLlamamientoAbierto{
		LlamamientoRef: "llamamiento:01K3B2AGREGADO",
		BolsaRef:       "bolsa:01K3B2AGREGADO",
		NecesidadRef:   "necesidad:01K3B2AGREGADO",
		PropuestaRef:   "propuesta:01K3B2AGREGADO",
		Version:        7,
	}
}

func llamamientoAbiertoPrueba(t *testing.T) LlamamientoAbierto {
	t.Helper()
	llamamiento, err := NuevoLlamamientoAbierto(datosLlamamientoAbiertoPrueba())
	if err != nil {
		t.Fatalf("crear llamamiento de prueba: %v", err)
	}
	return llamamiento
}

func terminalLlamamientoPrueba(estado EstadoLlamamiento) TerminalLlamamiento {
	return TerminalLlamamiento{
		Estado:       estado,
		OperacionRef: "operacion:01K3B2TERMINAL",
	}
}

func cadenaDecimalLlamamiento(valor int) string {
	if valor == 0 {
		return "0"
	}
	var buf [20]byte
	indice := len(buf)
	for valor > 0 {
		indice--
		buf[indice] = byte('0' + valor%10)
		valor /= 10
	}
	return string(buf[indice:])
}
