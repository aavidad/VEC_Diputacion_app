package importacionconvoca_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	memoria "vec-diputacion-granada/internal/modules/bolsa/adapters/memory"
	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

func TestServicioImportaConSHA256ActaEIdempotenciaPorContenido(t *testing.T) {
	repositorio := memoria.NuevoRepositorioImportacionesConvoca()
	servicio, err := aplicacion.NuevoServicio(
		decodificadorResumenValido{}, repositorio,
		func() time.Time { return time.Date(2026, 7, 18, 12, 30, 0, 123456789, time.FixedZone("prueba", 7200)) },
	)
	if err != nil {
		t.Fatalf("componer servicio: %v", err)
	}
	contenido := []byte("libro-xls-sintetico-opaco")
	primero, err := servicio.Importar(context.Background(), aplicacion.SolicitudImportacion{
		NombreFichero: "convoca-sintetico.xls", ActorRef: "actor:rrhh:tecnico-001", Contenido: contenido,
	})
	if err != nil {
		t.Fatalf("primera importacion: %v", err)
	}
	segundo, err := servicio.Importar(context.Background(), aplicacion.SolicitudImportacion{
		NombreFichero: "mismo-contenido.xls", ActorRef: "actor:rrhh:tecnico-002", Contenido: contenido,
	})
	if err != nil {
		t.Fatalf("reimportacion: %v", err)
	}
	suma := sha256.Sum256(contenido)
	huella := hex.EncodeToString(suma[:])
	if primero.Reutilizada || !segundo.Reutilizada || repositorio.NumeroLotes() != 1 {
		t.Fatalf("idempotencia inesperada: primero=%v segundo=%v lotes=%d",
			primero.Reutilizada, segundo.Reutilizada, repositorio.NumeroLotes())
	}
	if primero.Acta.HuellaFicheroSHA256 != huella ||
		primero.Acta.ActaRef != "acta:importacion-convoca:"+huella ||
		primero.Acta.ImportacionRef != "importacion:convoca:"+huella ||
		primero.Acta.ActorRef != "actor:rrhh:tecnico-001" ||
		segundo.Acta.ActorRef != primero.Acta.ActorRef ||
		primero.Acta.RegistradaEn.Location() != time.UTC ||
		primero.Acta.RegistradaEn.Nanosecond()%1000 != 0 ||
		primero.Acta.FilasLeidas != 1 || primero.Acta.FilasAceptadas != 1 ||
		primero.Acta.FilasRechazadas != 0 ||
		primero.Acta.Procedencia != dominio.NuevaProcedenciaNoAutoritativa() {
		t.Fatalf("acta inesperada: %#v", primero.Acta)
	}
}

func TestServicioIdempotenciaConcurrenteTieneUnSoloGanador(t *testing.T) {
	repositorio := memoria.NuevoRepositorioImportacionesConvoca()
	servicio, err := aplicacion.NuevoServicio(
		decodificadorResumenValido{}, repositorio,
		func() time.Time { return time.Date(2026, 7, 18, 10, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatalf("componer servicio: %v", err)
	}
	const trabajadores = 32
	inicio := make(chan struct{})
	errores := make(chan error, trabajadores)
	var nuevos atomic.Int32
	var grupo sync.WaitGroup
	for i := 0; i < trabajadores; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			resultado, err := servicio.Importar(context.Background(), aplicacion.SolicitudImportacion{
				NombreFichero: "concurrente.xls", ActorRef: "actor:rrhh:concurrente",
				Contenido: []byte("contenido-xls-concurrente-sintetico"),
			})
			if err == nil && !resultado.Reutilizada {
				nuevos.Add(1)
			}
			errores <- err
		}()
	}
	close(inicio)
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("importacion concurrente: %v", err)
		}
	}
	if nuevos.Load() != 1 || repositorio.NumeroLotes() != 1 {
		t.Fatalf("CAS no fue unico: nuevos=%d lotes=%d", nuevos.Load(), repositorio.NumeroLotes())
	}
}

func TestServicioRechazaSolicitudAntesDeDecodificar(t *testing.T) {
	decodificador := &decodificadorContador{}
	servicio, err := aplicacion.NuevoServicio(
		decodificador, memoria.NuevoRepositorioImportacionesConvoca(), time.Now,
	)
	if err != nil {
		t.Fatalf("componer servicio: %v", err)
	}
	casos := []aplicacion.SolicitudImportacion{
		{},
		{NombreFichero: "../real.xls", ActorRef: "actor:rrhh:001", Contenido: []byte("x")},
		{NombreFichero: "real.xlsx", ActorRef: "actor:rrhh:001", Contenido: []byte("x")},
		{NombreFichero: "real.xls", ActorRef: "Nombre Apellidos", Contenido: []byte("x")},
		{NombreFichero: "real.xls", ActorRef: "actor:rrhh:001", Contenido: make([]byte, aplicacion.MaximoBytesExportacion+1)},
	}
	for i, solicitud := range casos {
		if _, err := servicio.Importar(context.Background(), solicitud); !errors.Is(err, aplicacion.ErrSolicitudInvalida) {
			t.Fatalf("caso %d no rechazado: %v", i, err)
		}
	}
	if decodificador.llamadas.Load() != 0 {
		t.Fatalf("se decodificaron solicitudes invalidas: %d", decodificador.llamadas.Load())
	}
}

func TestServicioRespetaCancelacionYFallaAnteRepositorioInseguro(t *testing.T) {
	decodificador := &decodificadorContador{hoja: hojaResumenValida()}
	servicio, err := aplicacion.NuevoServicio(decodificador, repositorioCorrupto{}, time.Now)
	if err != nil {
		t.Fatalf("componer servicio: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err = servicio.Importar(ctx, solicitudValida())
	if !errors.Is(err, context.Canceled) || decodificador.llamadas.Load() != 0 {
		t.Fatalf("cancelacion no preservada: llamadas=%d error=%v", decodificador.llamadas.Load(), err)
	}
	_, err = servicio.Importar(context.Background(), solicitudValida())
	if !errors.Is(err, aplicacion.ErrResultadoInseguro) {
		t.Fatalf("repositorio corrupto aceptado: %v", err)
	}
}

type decodificadorResumenValido struct{}

func (decodificadorResumenValido) Decodificar(context.Context, io.ReadSeeker) (dominio.HojaStaging, error) {
	return hojaResumenValida(), nil
}

type decodificadorContador struct {
	llamadas atomic.Int32
	hoja     dominio.HojaStaging
}

func (d *decodificadorContador) Decodificar(context.Context, io.ReadSeeker) (dominio.HojaStaging, error) {
	d.llamadas.Add(1)
	return d.hoja, nil
}

type repositorioCorrupto struct{}

func (repositorioCorrupto) GuardarSiAusente(
	context.Context, dominio.LoteValidado,
) (dominio.LoteValidado, bool, error) {
	return dominio.LoteValidado{}, false, nil
}

func hojaResumenValida() dominio.HojaStaging {
	return dominio.HojaStaging{
		Esquema:   dominio.EsquemaResumenPersona,
		Cabeceras: dominio.EsquemaResumenPersona.Cabeceras(),
		Filas: []dominio.FilaStaging{{
			Numero: 2,
			Celdas: []dominio.CeldaStaging{
				{Tipo: dominio.CeldaTexto, Valor: "***0001**"},
				{Tipo: dominio.CeldaTexto, Valor: "Sintetica"},
				{Tipo: dominio.CeldaVacia},
				{Tipo: dominio.CeldaTexto, Valor: "Ana"},
				{Tipo: dominio.CeldaTexto, Valor: "Libre"},
				{Tipo: dominio.CeldaNumero, Valor: "1"},
				{Tipo: dominio.CeldaNumero, Valor: "1"},
				{Tipo: dominio.CeldaNumero, Valor: "2"},
			},
		}},
	}
}

func solicitudValida() aplicacion.SolicitudImportacion {
	return aplicacion.SolicitudImportacion{
		NombreFichero: "sintetico.xls", ActorRef: "actor:rrhh:prueba",
		Contenido: []byte("contenido-sintetico"),
	}
}
