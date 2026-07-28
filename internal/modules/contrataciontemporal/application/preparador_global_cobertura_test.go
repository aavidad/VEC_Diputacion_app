package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPreparadorGlobalCoberturaEsDeterministaConFinalizacionInvertida(
	t *testing.T,
) {
	vias := viasPreparacionGlobalPrueba(3, 2)
	invertido := nuevoEscenarioPreparacionGlobalPrueba(
		t, vias, 4, time.Second,
	)
	var mu sync.Mutex
	finalizaciones := make([]domain.ClaveCatalogo, 0, 6)
	invertido.antes = func(
		ctx context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) error {
		demora := time.Duration(7-int(solicitud.Comprobacion.Orden)) *
			5 * time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(demora):
		}
		mu.Lock()
		finalizaciones = append(
			finalizaciones,
			solicitud.Comprobacion.Clave,
		)
		mu.Unlock()
		return nil
	}
	primera, err := invertido.preparador.Preparar(
		context.Background(),
		invertido.datos,
	)
	if err != nil {
		t.Fatal(err)
	}

	directo := nuevoEscenarioPreparacionGlobalPrueba(
		t, vias, 4, time.Second,
	)
	directo.antes = func(
		ctx context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(
			time.Duration(solicitud.Comprobacion.Orden) *
				5 * time.Millisecond,
		):
			return nil
		}
	}
	segunda, err := directo.preparador.Preparar(
		context.Background(),
		directo.datos,
	)
	if err != nil {
		t.Fatal(err)
	}
	referenciaPrimera, _ := primera.Referencia()
	referenciaSegunda, _ := segunda.Referencia()
	huellaPrimera, _ := primera.HuellaSHA256()
	huellaSegunda, _ := segunda.HuellaSHA256()
	if referenciaPrimera != referenciaSegunda ||
		huellaPrimera != huellaSegunda {
		t.Fatal("la preparación depende del orden de finalización")
	}
	if len(finalizaciones) != 6 ||
		finalizaciones[0] == vias[0].Comprobaciones[0].Clave {
		t.Fatalf("la finalización no quedó invertida: %#v", finalizaciones)
	}
	ordenes, err := primera.OrdenesPendientesEn(
		invertido.entorno.reloj.Ahora(),
	)
	if err != nil || len(ordenes) != 6 {
		t.Fatalf("órdenes pendientes inesperadas: %d, %v", len(ordenes), err)
	}
	datosPropuesta, err := primera.DatosCrearPropuestaEn(
		invertido.entorno.reloj.Ahora(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.CrearPropuestaDecisionCobertura(
		datosPropuesta,
	); err != nil {
		t.Fatalf("resultado no apto para propuesta: %v", err)
	}
	exigirCeroConsumoPreparacionGlobal(t, invertido)
	exigirCeroConsumoPreparacionGlobal(t, directo)
}

func TestPreparadorGlobalCoberturaAcotaParalelismo(t *testing.T) {
	escenario := nuevoEscenarioPreparacionGlobalPrueba(
		t,
		viasPreparacionGlobalPrueba(4, 2),
		3,
		time.Second,
	)
	var activas atomic.Int32
	var maximas atomic.Int32
	continuar := make(chan struct{})
	var liberar sync.Once
	escenario.antes = func(
		ctx context.Context,
		_ ports.SolicitudConsultarCobertura,
	) error {
		actual := activas.Add(1)
		defer activas.Add(-1)
		for {
			maximo := maximas.Load()
			if actual <= maximo || maximas.CompareAndSwap(maximo, actual) {
				break
			}
		}
		if actual == 3 {
			liberar.Do(func() { close(continuar) })
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-continuar:
			return nil
		}
	}
	if _, err := escenario.preparador.Preparar(
		context.Background(),
		escenario.datos,
	); err != nil {
		t.Fatal(err)
	}
	if maximas.Load() != 3 || activas.Load() != 0 {
		t.Fatalf(
			"paralelismo inesperado: máximo=%d activas=%d",
			maximas.Load(),
			activas.Load(),
		)
	}
}

func TestPreparadorGlobalCoberturaRechazaReferenciaDuplicadaAntesDeConsultar(
	t *testing.T,
) {
	escenario := nuevoEscenarioPreparacionGlobalPrueba(
		t,
		viasPreparacionGlobalPrueba(2, 2),
		4,
		time.Second,
	)
	escenario.generador.repetirDesde = 2
	var consultas atomic.Int32
	escenario.antes = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) error {
		consultas.Add(1)
		return nil
	}
	preparacion, err := escenario.preparador.Preparar(
		context.Background(),
		escenario.datos,
	)
	if !errors.Is(
		err,
		ErrReferenciaPreparacionGlobalCoberturaRepetida,
	) {
		t.Fatalf("referencia duplicada aceptada: %v", err)
	}
	if consultas.Load() != 0 {
		t.Fatalf("se lanzaron %d consultas antes de rechazar", consultas.Load())
	}
	if _, err := preparacion.Referencia(); err == nil {
		t.Fatal("el fallo parcial devolvió una capacidad")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario)
}

func TestPreparadorGlobalCoberturaCancelaTodoAlPrimerFallo(t *testing.T) {
	escenario := nuevoEscenarioPreparacionGlobalPrueba(
		t,
		viasPreparacionGlobalPrueba(3, 2),
		4,
		time.Second,
	)
	var iniciadas atomic.Int32
	var canceladas atomic.Int32
	listas := make(chan struct{})
	var cerrar sync.Once
	escenario.antes = func(
		ctx context.Context,
		solicitud ports.SolicitudConsultarCobertura,
	) error {
		if iniciadas.Add(1) == 4 {
			cerrar.Do(func() { close(listas) })
		}
		select {
		case <-ctx.Done():
			canceladas.Add(1)
			return ctx.Err()
		case <-listas:
		}
		if solicitud.Comprobacion.Clave == "comprobacion_01_01" {
			return errors.New("fallo privado de proveedor")
		}
		<-ctx.Done()
		canceladas.Add(1)
		return ctx.Err()
	}
	preparacion, err := escenario.preparador.Preparar(
		context.Background(),
		escenario.datos,
	)
	if !errors.Is(err, ErrPreparacionGlobalCoberturaNoConfiable) {
		t.Fatalf("fallo parcial no cerrado: %v", err)
	}
	if strings.Contains(err.Error(), "proveedor") {
		t.Fatalf("se filtró detalle privado: %v", err)
	}
	if iniciadas.Load() != 4 || canceladas.Load() != 3 {
		t.Fatalf(
			"cancelación incompleta: iniciadas=%d canceladas=%d",
			iniciadas.Load(),
			canceladas.Load(),
		)
	}
	if _, err := preparacion.Referencia(); err == nil {
		t.Fatal("el fallo parcial devolvió una capacidad")
	}
	exigirCeroConsumoPreparacionGlobal(t, escenario)
}

func TestPreparadorGlobalCoberturaAplicaTimeoutTotal(t *testing.T) {
	escenario := nuevoEscenarioPreparacionGlobalPrueba(
		t,
		viasPreparacionGlobalPrueba(2, 2),
		2,
		80*time.Millisecond,
	)
	escenario.antes = func(
		ctx context.Context,
		_ ports.SolicitudConsultarCobertura,
	) error {
		<-ctx.Done()
		return ctx.Err()
	}
	inicio := time.Now()
	preparacion, err := escenario.preparador.Preparar(
		context.Background(),
		escenario.datos,
	)
	if !errors.Is(err, context.DeadlineExceeded) ||
		time.Since(inicio) > 500*time.Millisecond {
		t.Fatalf("timeout total no respetado: %v, %s", err, time.Since(inicio))
	}
	if _, err := preparacion.Referencia(); err == nil {
		t.Fatal("el timeout devolvió una capacidad")
	}
}

func TestPreparadorGlobalCoberturaRespetaCancelacionDelLlamador(
	t *testing.T,
) {
	escenario := nuevoEscenarioPreparacionGlobalPrueba(
		t,
		viasPreparacionGlobalPrueba(2, 2),
		2,
		time.Second,
	)
	iniciada := make(chan struct{})
	var anunciar sync.Once
	escenario.antes = func(
		ctx context.Context,
		_ ports.SolicitudConsultarCobertura,
	) error {
		anunciar.Do(func() { close(iniciada) })
		<-ctx.Done()
		return ctx.Err()
	}
	ctx, cancelar := context.WithCancel(context.Background())
	type resultado struct {
		err      error
		conValor bool
	}
	terminada := make(chan resultado, 1)
	go func() {
		preparacion, err := escenario.preparador.Preparar(
			ctx,
			escenario.datos,
		)
		_, errReferencia := preparacion.Referencia()
		terminada <- resultado{err: err, conValor: errReferencia == nil}
	}()
	<-iniciada
	cancelar()
	select {
	case obtenido := <-terminada:
		if !errors.Is(obtenido.err, context.Canceled) ||
			obtenido.conValor {
			t.Fatalf("cancelación no respetada: %#v", obtenido)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("los obreros no terminaron tras cancelar")
	}
}

func TestPreparadorGlobalCoberturaRechazaGobiernoYDatosInvalidos(
	t *testing.T,
) {
	escenario := nuevoEscenarioPreparacionGlobalPrueba(
		t,
		viasPreparacionGlobalPrueba(2, 1),
		2,
		time.Second,
	)
	otroCatalogo := publicarCatalogoGlobalC1(
		t,
		escenario.entorno.inicio,
		"catalogo_preparacion_global_ajeno_01",
		1,
		viasPreparacionGlobalPrueba(2, 1),
	)
	_, err := nuevosDatosPreparacionGlobalCobertura(
		analisisPreparacionGlobalRef,
		huellaAnalisisPreparacionGlobal,
		otroCatalogo,
		escenario.politica,
		organizacionCoberturaPrueba,
		"expediente_preparacion_global_012345",
		3,
		"categoria_trabajo_social",
		escenario.datos.periodo,
	)
	if !errors.Is(err, ErrDatosPreparacionGlobalCoberturaInvalidos) {
		t.Fatalf("catálogo/política cruzados aceptados: %v", err)
	}
	escenario.entorno.reloj.fijar(
		escenario.entorno.inicio.Add(31 * time.Minute),
	)
	if _, err := escenario.preparador.Preparar(
		context.Background(),
		escenario.datos,
	); !errors.Is(err, ErrDatosPreparacionGlobalCoberturaInvalidos) {
		t.Fatalf("gobierno caducado aceptado: %v", err)
	}
	if escenario.generador.llamadas() != 0 {
		t.Fatal("se generaron referencias con gobierno inválido")
	}
	if _, err := nuevosDatosPreparacionGlobalCobertura(
		analisisPreparacionGlobalRef,
		huellaAnalisisPreparacionGlobal,
		escenario.catalogo,
		escenario.politica,
		organizacionCoberturaPrueba,
		"expediente_preparacion_global_012345",
		maximoEnteroSeguroPreparacionGlobal,
		"categoria_trabajo_social",
		escenario.datos.periodo,
	); err != nil {
		t.Fatalf("máximo interoperable rechazado: %v", err)
	}

	casos := []struct {
		nombre       string
		analisisRef  string
		huella       string
		version      uint64
		organizacion string
		periodo      domain.PeriodoPrevisto
	}{
		{
			"análisis", "x", huellaAnalisisPreparacionGlobal, 3,
			organizacionCoberturaPrueba, escenario.datos.periodo,
		},
		{
			"huella", analisisPreparacionGlobalRef, "ABC", 3,
			organizacionCoberturaPrueba, escenario.datos.periodo,
		},
		{
			"versión", analisisPreparacionGlobalRef,
			huellaAnalisisPreparacionGlobal, 0,
			organizacionCoberturaPrueba, escenario.datos.periodo,
		},
		{
			"desbordamiento", analisisPreparacionGlobalRef,
			huellaAnalisisPreparacionGlobal,
			maximoEnteroSeguroPreparacionGlobal + 1,
			organizacionCoberturaPrueba, escenario.datos.periodo,
		},
		{
			"organización", analisisPreparacionGlobalRef,
			huellaAnalisisPreparacionGlobal, 3,
			"organizacion_ajena_012345", escenario.datos.periodo,
		},
		{
			"periodo", analisisPreparacionGlobalRef,
			huellaAnalisisPreparacionGlobal, 3,
			organizacionCoberturaPrueba, domain.PeriodoPrevisto{},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			_, err := nuevosDatosPreparacionGlobalCobertura(
				caso.analisisRef,
				caso.huella,
				escenario.catalogo,
				escenario.politica,
				caso.organizacion,
				"expediente_preparacion_global_012345",
				caso.version,
				"categoria_trabajo_social",
				caso.periodo,
			)
			if !errors.Is(
				err,
				ErrDatosPreparacionGlobalCoberturaInvalidos,
			) {
				t.Fatalf("datos inválidos aceptados: %v", err)
			}
		})
	}
}

func TestPreparadorGlobalCoberturaFronterasSeguras(t *testing.T) {
	escenario := nuevoEscenarioPreparacionGlobalPrueba(
		t,
		viasPreparacionGlobalPrueba(1, 1),
		1,
		time.Second,
	)
	if _, err := json.Marshal(escenario.datos); !errors.Is(
		err,
		ErrSerializacionDatosPreparacionGlobalCoberturaProhibida,
	) {
		t.Fatalf("datos opacos serializados: %v", err)
	}
	formateados := fmt.Sprintf("%#v", escenario.datos)
	for _, secreto := range []string{
		analisisPreparacionGlobalRef,
		"expediente_preparacion_global_012345",
		"categoria_trabajo_social",
	} {
		if strings.Contains(formateados, secreto) {
			t.Fatalf("el formato expuso %q: %s", secreto, formateados)
		}
	}
	var generadorNulo *generadorReferenciasPreparacionGlobalPrueba
	if _, err := NuevoPreparadorGlobalCobertura(
		escenario.preparador.consultas,
		generadorNulo,
		escenario.entorno.reloj,
		1,
		time.Second,
	); !errors.Is(err, ErrPreparadorGlobalCoberturaInvalido) {
		t.Fatalf("nil tipado aceptado: %v", err)
	}
	for _, concurrencia := range []int{0, 17} {
		if _, err := NuevoPreparadorGlobalCobertura(
			escenario.preparador.consultas,
			escenario.generador,
			escenario.entorno.reloj,
			concurrencia,
			time.Second,
		); !errors.Is(err, ErrPreparadorGlobalCoberturaInvalido) {
			t.Fatalf("concurrencia %d aceptada: %v", concurrencia, err)
		}
	}
	if _, err := NuevoPreparadorGlobalCobertura(
		escenario.preparador.consultas,
		escenario.generador,
		escenario.entorno.reloj,
		1,
		TiempoMaximoPreparacionGlobalCobertura+time.Microsecond,
	); !errors.Is(err, ErrPreparadorGlobalCoberturaInvalido) {
		t.Fatalf("plazo superior a tres segundos aceptado: %v", err)
	}
}

func TestPreparadorGlobalCoberturaLimiteOperativoPorVia(t *testing.T) {
	if total, err := validarCargaPreparacionGlobal(
		viasPreparacionGlobalPrueba(1, 32),
	); err != nil || total != 32 {
		t.Fatalf("límite de 32 rechazado: total=%d, err=%v", total, err)
	}
	if _, err := validarCargaPreparacionGlobal(
		viasPreparacionGlobalPrueba(1, 33),
	); !errors.Is(err, ErrDatosPreparacionGlobalCoberturaInvalidos) {
		t.Fatalf("33 comprobaciones aceptadas: %v", err)
	}
}
