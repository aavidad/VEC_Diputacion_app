package interna

import (
	"context"
	"errors"
	"net"
	"net/http"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type registroCierreAplicacionInternaPrueba struct {
	mu    sync.Mutex
	orden []string
}

func (r *registroCierreAplicacionInternaPrueba) anotar(nombre string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orden = append(r.orden, nombre)
}

func (r *registroCierreAplicacionInternaPrueba) copia() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.orden...)
}

type recursoAplicacionInternaPrueba struct {
	nombre   string
	registro *registroCierreAplicacionInternaPrueba
	error    error
	panic    bool
	poseido  atomic.Bool
	llamadas atomic.Int32
}

func (r *recursoAplicacionInternaPrueba) reclamarPropiedad() bool {
	return r != nil && r.poseido.CompareAndSwap(false, true)
}

func (r *recursoAplicacionInternaPrueba) cerrar() error {
	r.llamadas.Add(1)
	r.registro.anotar(r.nombre)
	if r.panic {
		panic("MARCADOR_PRIVADO_CIERRE")
	}
	return r.error
}

func TestNuevaAplicacionInternaFallaCerradoYLiberaInventario(t *testing.T) {
	t.Run("servidor ausente", func(t *testing.T) {
		registro := &registroCierreAplicacionInternaPrueba{}
		uno := &recursoAplicacionInternaPrueba{nombre: "uno", registro: registro}
		dos := &recursoAplicacionInternaPrueba{nombre: "dos", registro: registro}
		aplicacion, err := nuevaAplicacionInterna(nil, uno, dos)
		if aplicacion != nil || !errors.Is(err, ErrAplicacionInternaNoDisponible) {
			t.Fatalf("constructor = (%v, %v)", aplicacion, err)
		}
		if recibida := registro.copia(); !reflect.DeepEqual(recibida, []string{"dos", "uno"}) {
			t.Fatalf("orden de cierre = %v", recibida)
		}
	})

	t.Run("inventario vacio", func(t *testing.T) {
		servidor := nuevoServidorAplicacionInternaPrueba(t)
		aplicacion, err := nuevaAplicacionInterna(servidor)
		if aplicacion != nil || !errors.Is(err, ErrAplicacionInternaNoDisponible) {
			t.Fatalf("constructor = (%v, %v)", aplicacion, err)
		}
		if err := servidor.EscucharYServir(); !errors.Is(err, ErrServidorInternoInvalido) {
			t.Fatalf("servidor sobrevivio al fallo: %v", err)
		}
	})

	t.Run("nulo tipado", func(t *testing.T) {
		servidor := nuevoServidorAplicacionInternaPrueba(t)
		registro := &registroCierreAplicacionInternaPrueba{}
		uno := &recursoAplicacionInternaPrueba{nombre: "uno", registro: registro}
		dos := &recursoAplicacionInternaPrueba{nombre: "dos", registro: registro}
		var nulo *recursoAplicacionInternaPrueba
		aplicacion, err := nuevaAplicacionInterna(servidor, uno, nulo, dos)
		if aplicacion != nil || !errors.Is(err, ErrAplicacionInternaNoDisponible) {
			t.Fatalf("constructor = (%v, %v)", aplicacion, err)
		}
		if recibida := registro.copia(); !reflect.DeepEqual(recibida, []string{"dos", "uno"}) {
			t.Fatalf("orden de cierre = %v", recibida)
		}
		if uno.llamadas.Load() != 1 || dos.llamadas.Load() != 1 {
			t.Fatalf("cierres = (%d, %d)", uno.llamadas.Load(), dos.llamadas.Load())
		}
		if err := servidor.EscucharYServir(); !errors.Is(err, ErrServidorInternoInvalido) {
			t.Fatalf("servidor sobrevivio al fallo: %v", err)
		}
	})

	t.Run("duplicado", func(t *testing.T) {
		servidor := nuevoServidorAplicacionInternaPrueba(t)
		registro := &registroCierreAplicacionInternaPrueba{}
		uno := &recursoAplicacionInternaPrueba{nombre: "uno", registro: registro}
		dos := &recursoAplicacionInternaPrueba{nombre: "dos", registro: registro}
		aplicacion, err := nuevaAplicacionInterna(servidor, uno, dos, uno)
		if aplicacion != nil || !errors.Is(err, ErrAplicacionInternaNoDisponible) {
			t.Fatalf("constructor = (%v, %v)", aplicacion, err)
		}
		if recibida := registro.copia(); !reflect.DeepEqual(recibida, []string{"dos", "uno"}) {
			t.Fatalf("orden de cierre = %v", recibida)
		}
		if uno.llamadas.Load() != 1 || dos.llamadas.Load() != 1 {
			t.Fatalf("duplicado se cerro mas de una vez: (%d, %d)", uno.llamadas.Load(), dos.llamadas.Load())
		}
	})
}

func TestNuevaAplicacionInternaRechazaPropiedadCompartidaEntreAplicaciones(
	t *testing.T,
) {
	servidor := nuevoServidorAplicacionInternaPrueba(t)
	registro := &registroCierreAplicacionInternaPrueba{}
	compartido := &recursoAplicacionInternaPrueba{
		nombre: "compartido", registro: registro,
	}
	primera, err := nuevaAplicacionInterna(servidor, compartido)
	if err != nil {
		t.Fatal(err)
	}

	segunda, err := nuevaAplicacionInterna(servidor, compartido)
	if segunda != nil || !errors.Is(err, ErrAplicacionInternaNoDisponible) {
		t.Fatalf("segunda aplicacion = (%v, %v)", segunda, err)
	}
	if compartido.llamadas.Load() != 0 {
		t.Fatal("el constructor fallido cerro un recurso de otra aplicacion")
	}
	if err := primera.Cerrar(); err != nil {
		t.Fatal(err)
	}
	if compartido.llamadas.Load() != 1 {
		t.Fatalf("cierres del propietario = %d", compartido.llamadas.Load())
	}
}

func TestAplicacionInternaApagarAntesDeEscucharEsperaYCierraDespues(t *testing.T) {
	servidor := nuevoServidorAplicacionInternaPrueba(t)
	registro := &registroCierreAplicacionInternaPrueba{}
	recurso := &recursoAplicacionInternaPrueba{nombre: "unico", registro: registro}
	aplicacion, err := nuevaAplicacionInterna(servidor, recurso)
	if err != nil {
		t.Fatal(err)
	}
	if err := aplicacion.Apagar(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-aplicacion.terminado:
	default:
		t.Fatal("Apagar no publico la terminacion")
	}
	if recurso.llamadas.Load() != 0 {
		t.Fatal("Apagar cerro recursos antes de Cerrar")
	}
	if err := aplicacion.Cerrar(); err != nil {
		t.Fatal(err)
	}
	if recurso.llamadas.Load() != 1 {
		t.Fatalf("cierres = %d", recurso.llamadas.Load())
	}
}

func TestAplicacionInternaCerrarEsInversoIdempotenteYSaneado(t *testing.T) {
	servidor := nuevoServidorAplicacionInternaPrueba(t)
	registro := &registroCierreAplicacionInternaPrueba{}
	uno := &recursoAplicacionInternaPrueba{nombre: "uno", registro: registro}
	dos := &recursoAplicacionInternaPrueba{
		nombre: "dos", registro: registro,
		error: errors.New("MARCADOR_PRIVADO_CIERRE"),
	}
	tres := &recursoAplicacionInternaPrueba{
		nombre: "tres", registro: registro, panic: true,
	}
	aplicacion, err := nuevaAplicacionInterna(servidor, uno, dos, tres)
	if err != nil {
		t.Fatal(err)
	}

	const simultaneas = 24
	resultados := make(chan error, simultaneas)
	var grupo sync.WaitGroup
	for range simultaneas {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			resultados <- aplicacion.Cerrar()
		}()
	}
	grupo.Wait()
	close(resultados)
	for err := range resultados {
		if !errors.Is(err, ErrAplicacionInternaNoDisponible) ||
			err.Error() != ErrAplicacionInternaNoDisponible.Error() {
			t.Fatalf("error no saneado: %v", err)
		}
	}
	if recibida := registro.copia(); !reflect.DeepEqual(
		recibida,
		[]string{"tres", "dos", "uno"},
	) {
		t.Fatalf("orden de cierre = %v", recibida)
	}
	for _, recurso := range []*recursoAplicacionInternaPrueba{uno, dos, tres} {
		if recurso.llamadas.Load() != 1 {
			t.Fatalf("%s se cerro %d veces", recurso.nombre, recurso.llamadas.Load())
		}
	}
	select {
	case <-aplicacion.terminado:
	default:
		t.Fatal("Cerrar antes de escuchar no termino la capsula")
	}
}

func TestAplicacionInternaNoCierraDuranteEscucha(t *testing.T) {
	servidor := nuevoServidorAplicacionInternaPrueba(t)
	registro := &registroCierreAplicacionInternaPrueba{}
	uno := &recursoAplicacionInternaPrueba{nombre: "uno", registro: registro}
	dos := &recursoAplicacionInternaPrueba{nombre: "dos", registro: registro}
	aplicacion, err := nuevaAplicacionInterna(servidor, uno, dos)
	if err != nil {
		t.Fatal(err)
	}
	terminado := iniciarAplicacionInternaPrueba(t, aplicacion)

	if err := aplicacion.Cerrar(); !errors.Is(err, ErrAplicacionInternaNoDisponible) {
		t.Fatalf("cierre durante escucha = %v", err)
	}
	if len(registro.copia()) != 0 {
		t.Fatalf("recursos cerrados durante escucha: %v", registro.copia())
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	const apagadosSimultaneos = 12
	erroresApagado := make(chan error, apagadosSimultaneos)
	var apagados sync.WaitGroup
	for range apagadosSimultaneos {
		apagados.Add(1)
		go func() {
			defer apagados.Done()
			erroresApagado <- aplicacion.Apagar(ctx)
		}()
	}
	apagados.Wait()
	close(erroresApagado)
	for err := range erroresApagado {
		if err != nil {
			t.Fatalf("apagado concurrente = %v", err)
		}
	}
	select {
	case <-aplicacion.terminado:
	default:
		t.Fatal("Apagar retorno antes de la terminacion de la aplicacion")
	}
	if err := aplicacion.Cerrar(); err != nil {
		t.Fatal(err)
	}
	if err := <-terminado; err != nil {
		t.Fatalf("escucha = %v", err)
	}
	if recibida := registro.copia(); !reflect.DeepEqual(recibida, []string{"dos", "uno"}) {
		t.Fatalf("orden de cierre = %v", recibida)
	}
}

func TestEsperaTerminacionAplicacionInternaRespetaElContexto(t *testing.T) {
	terminado := make(chan struct{})
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := esperarTerminacionAplicacionInterna(
		ctx,
		terminado,
	); !errors.Is(err, ErrAplicacionInternaNoDisponible) {
		t.Fatalf("espera sin limite de contexto = %v", err)
	}

	close(terminado)
	if err := esperarTerminacionAplicacionInterna(ctx, terminado); err != nil {
		t.Fatalf("terminacion ya publicada = %v", err)
	}
}

func TestAplicacionInternaOpacaYRechazaCopias(t *testing.T) {
	tipoValor := reflect.TypeOf(AplicacionInterna{})
	for indice := range tipoValor.NumField() {
		campo := tipoValor.Field(indice)
		if campo.IsExported() || campo.Anonymous {
			t.Fatalf("campo expuesto: %s", campo.Name)
		}
	}
	tipoPuntero := reflect.TypeOf((*AplicacionInterna)(nil))
	if tipoPuntero.NumMethod() != 3 {
		t.Fatalf("metodos publicos = %d", tipoPuntero.NumMethod())
	}
	for _, nombre := range []string{"EscucharYServir", "Apagar", "Cerrar"} {
		if _, existe := tipoPuntero.MethodByName(nombre); !existe {
			t.Fatalf("metodo ausente: %s", nombre)
		}
	}

	servidor := nuevoServidorAplicacionInternaPrueba(t)
	registro := &registroCierreAplicacionInternaPrueba{}
	recurso := &recursoAplicacionInternaPrueba{nombre: "unico", registro: registro}
	original, err := nuevaAplicacionInterna(servidor, recurso)
	if err != nil {
		t.Fatal(err)
	}
	capsulaAjena := &AplicacionInterna{
		propietario: original,
		servidor:    original.servidor,
		recursos:    original.recursos,
		terminado:   original.terminado,
	}
	if err := capsulaAjena.Cerrar(); !errors.Is(err, ErrAplicacionInternaNoDisponible) {
		t.Fatalf("copia aceptada: %v", err)
	}
	if recurso.llamadas.Load() != 0 {
		t.Fatal("copia cerro recursos del propietario")
	}
	if err := original.Cerrar(); err != nil {
		t.Fatal(err)
	}
}

func nuevoServidorAplicacionInternaPrueba(t *testing.T) *ServidorInterno {
	t.Helper()
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInternoPrueba(
		t,
		material.cfg,
		http.NotFoundHandler(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return servidor
}

func iniciarAplicacionInternaPrueba(
	t *testing.T,
	aplicacion *AplicacionInterna,
) <-chan error {
	t.Helper()
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	direccion := escucha.Addr().String()
	if err := escucha.Close(); err != nil {
		t.Fatal(err)
	}
	aplicacion.servidor.direccionEscucha = direccion
	aplicacion.servidor.manejador.direccionEscucha = direccion
	terminado := make(chan error, 1)
	go func() { terminado <- aplicacion.EscucharYServir() }()
	limite := time.Now().Add(2 * time.Second)
	for {
		aplicacion.servidor.ejecucion.mu.Lock()
		activo := aplicacion.servidor.ejecucion.servidorActivo != nil
		aplicacion.servidor.ejecucion.mu.Unlock()
		if activo {
			return terminado
		}
		select {
		case err := <-terminado:
			t.Fatalf("escucha termino antes de arrancar: %v", err)
		default:
		}
		if time.Now().After(limite) {
			t.Fatal("escucha no arranco")
		}
		time.Sleep(time.Millisecond)
	}
}
