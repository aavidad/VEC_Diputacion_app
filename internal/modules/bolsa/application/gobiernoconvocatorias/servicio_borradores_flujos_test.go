package gobiernoconvocatorias

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

func TestBorradorRelojAvanzaReiniciaYRotaSinCambiarF(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 1)
	primero, err := e.servicio.Crear(context.Background(), e.orden)
	if err != nil {
		t.Fatal(err)
	}
	e.reloj.avanzar(45 * time.Second)
	reiniciado := e.reiniciar(t, 2, 1)
	segundo, err := reiniciado.Crear(context.Background(), e.orden)
	if err != nil || !reflect.DeepEqual(segundo, primero) {
		t.Fatalf("retry con reloj/rotacion no recupero recibo exacto: %v", err)
	}
	if e.catalogo.preparaciones != 1 || e.autorizador.llamadas != 1 ||
		e.diario.reservas != 1 || e.confirmador.llamadas != 1 || e.confirmador.efectos != 1 {
		t.Fatalf("replay repitio trabajo: preparaciones=%d pdp=%d reservas=%d confirmaciones=%d efectos=%d",
			e.catalogo.preparaciones, e.autorizador.llamadas, e.diario.reservas,
			e.confirmador.llamadas, e.confirmador.efectos)
	}
	conflicto := e.orden
	conflicto.Contenido.Titulo = "Otra solicitud semantica"
	if _, err := reiniciado.Crear(context.Background(), conflicto); !errors.Is(err, puertosbolsa.ErrClaveIdempotenciaConvocatoriaReusada) {
		t.Fatalf("misma L y F distinta no produjo conflicto: %v", err)
	}
}

func TestAmbitoIdempotenciaAislaPersonaPerfilYAccion(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	intencionAlta, err := nuevaIntencionAltaBorradorCanonica(
		e.catalogo.plantilla.Referencia, e.orden.CodigoVersionPublica, e.orden.ExpedienteRef,
		e.orden.Contenido, e.orden.MotivoCatalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	derivar := func(actor dominiovec.ContextoActor, intencion IntencionBorradorCanonica) ProyeccionIdentidadOperacion {
		solicitud, err := nuevaSolicitudDerivacionIdempotencia(e.orden.ClaveCliente, intencion, actor)
		if err != nil {
			t.Fatal(err)
		}
		conjunto, err := e.derivador.Derivar(context.Background(), solicitud)
		if err != nil {
			t.Fatal(err)
		}
		primaria, err := conjunto.primaria()
		if err != nil {
			t.Fatal(err)
		}
		return primaria
	}
	base := derivar(e.orden.Actor, intencionAlta)
	otroPerfil, _, err := pruebasvec.NuevoContextoYVinculo(
		instanteBorradorPrueba, e.orden.Actor.PersonaRef, "prf_otra234567890abcdefghijkl",
		dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	otraPersona, _, err := pruebasvec.NuevoContextoYVinculo(
		instanteBorradorPrueba, "per_otra234567890abcdefghijkl", "prf_otra345678901abcdefghijkl",
		dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	esperada, err := puertosbolsa.EstadoVersionConvocatoria(e.inicial)
	if err != nil {
		t.Fatal(err)
	}
	motivoActualizar := e.orden.MotivoCatalogo
	motivoActualizar.EntradaClave = "actualizar_borrador"
	intencionActualizar, err := nuevaIntencionActualizacionBorradorCanonica(
		esperada, e.orden.Contenido, motivoActualizar,
	)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, identidad := range map[string]ProyeccionIdentidadOperacion{
		"perfil":  derivar(otroPerfil, intencionAlta),
		"persona": derivar(otraPersona, intencionAlta),
		"accion":  derivar(e.orden.Actor, intencionActualizar),
	} {
		if proyeccionesHMACCoinciden(base.Localizador, identidad.Localizador, dominioClaveHMACLocalizador) {
			t.Fatalf("L no aislo %s", nombre)
		}
	}
}

func TestRotacionRechazaGeneracionDominioDuplicadosYAmbiguedad(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	intencion, err := nuevaIntencionAltaBorradorCanonica(
		e.catalogo.plantilla.Referencia, e.orden.CodigoVersionPublica, e.orden.ExpedienteRef,
		e.orden.Contenido, e.orden.MotivoCatalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, _ := nuevaSolicitudDerivacionIdempotencia(e.orden.ClaveCliente, intencion, e.orden.Actor)
	conjunto, err := e.derivador.Derivar(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	identidades := conjunto.identidades
	if _, err := NuevoConjuntoIdentidadesOperacion(identidades[1], identidades[0]); err == nil {
		t.Fatal("se aceptaron generaciones desordenadas")
	}
	if _, err := NuevoConjuntoIdentidadesOperacion(identidades[0], identidades[0]); err == nil {
		t.Fatal("se acepto identidad duplicada")
	}
	if _, err := NuevaIdentidadOperacionDerivada(
		identidades[0].Localizador, identidades[1].HuellaSolicitud,
	); err == nil {
		t.Fatal("se emparejaron generaciones L/F distintas")
	}
	refF, _ := NuevaReferenciaClaveHMACHuellaSolicitud("clave:hmac:convocatorias:huella:v2", 2)
	if _, err := NuevoLocalizadorOperacion(2, refF, huellaHexPrueba('d')); err == nil {
		t.Fatal("el dominio de huella se acepto como localizador")
	}
	consulta, err := nuevaSolicitudConsultaIdentidadesBorrador(conjunto)
	if err != nil {
		t.Fatal(err)
	}
	ambiguo := ResultadoConsultaIdentidadesBorrador{Coincidencias: []CoincidenciaIdentidadBorrador{
		{
			Resolucion: ResolucionIdentidadBorrador{
				IdentidadesConsultadas: []ProyeccionIdentidadOperacion{consulta.Identidades[0]},
				IdentidadPrimaria:      consulta.Identidades[0],
			},
			Resultado: ResultadoOperacionDiario{Estado: ResultadoDiarioConflicto},
		},
		{
			Resolucion: ResolucionIdentidadBorrador{
				IdentidadesConsultadas: []ProyeccionIdentidadOperacion{consulta.Identidades[1]},
				IdentidadPrimaria:      consulta.Identidades[1],
			},
			Resultado: ResultadoOperacionDiario{Estado: ResultadoDiarioConflicto},
		},
	}}
	if !errors.Is(ambiguo.ValidarPara(consulta), ErrConsultaIdempotenciaAmbigua) {
		t.Fatal("dos generaciones coincidentes no fallaron de forma cerrada")
	}
}

func TestPDPDenegadoYDerribadoNoSeConfunden(t *testing.T) {
	t.Run("denegado", func(t *testing.T) {
		e := nuevoEscenario(t, confirmarBien, 2, 1)
		e.autorizador.modo = pdpDenegar
		_, err := e.servicio.Crear(context.Background(), e.orden)
		if !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || e.diario.reservas != 0 {
			t.Fatalf("denegacion no se clasifico exactamente: %v", err)
		}
	})
	t.Run("indisponible", func(t *testing.T) {
		e := nuevoEscenario(t, confirmarBien, 2, 1)
		e.autorizador.modo = pdpCaido
		_, err := e.servicio.Crear(context.Background(), e.orden)
		if err == nil || errors.Is(err, dominiovec.ErrAutorizacionDenegada) || e.diario.reservas != 0 {
			t.Fatalf("caida PDP se convirtio en denegacion: %v", err)
		}
	})
}

func TestAltaRechazaHuellaPlantillaIncorrectaAntesDePDP(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	orden := e.orden
	orden.Plantilla.HuellaContenidoSHA256 = huellaHexPrueba('0')
	_, err := e.servicio.Crear(context.Background(), orden)
	if err == nil || e.autorizador.llamadas != 0 || e.diario.reservas != 0 ||
		e.catalogo.preparaciones != 0 {
		t.Fatalf("huella de plantilla manipulada alcanzo PDP/reserva: %v", err)
	}
}

func TestBorradoresCASConcurrenteProduceUnSoloEfecto(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	const competidores = 64
	errores := make(chan error, competidores)
	var grupo sync.WaitGroup
	grupo.Add(competidores)
	for indice := 0; indice < competidores; indice++ {
		go func() {
			defer grupo.Done()
			_, err := e.servicio.Crear(context.Background(), e.orden)
			if err != nil && !errors.Is(err, ErrOperacionBorradorEnCurso) {
				errores <- err
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatalf("competidor obtuvo error inesperado: %v", err)
	}
	if e.diario.reservas != 1 || e.confirmador.efectos != 1 || e.confirmador.llamadas != 1 {
		t.Fatalf("CAS duplico efecto: reservas=%d confirmaciones=%d efectos=%d",
			e.diario.reservas, e.confirmador.llamadas, e.confirmador.efectos)
	}
}

func TestActualizacionConservaConfiguracionFijada(t *testing.T) {
	e := nuevoEscenario(t, confirmarBien, 2, 1)
	esperada, err := puertosbolsa.EstadoVersionConvocatoria(e.inicial)
	if err != nil {
		t.Fatal(err)
	}
	contenido := e.inicial.Contenido
	contenido.Titulo = "Bolsa temporal actualizada"
	motivo := e.orden.MotivoCatalogo
	motivo.EntradaClave = "actualizar_borrador"
	_, err = e.servicio.Actualizar(context.Background(), OrdenActualizarBorrador{
		ClaveCliente: e.orden.ClaveCliente, Actor: e.orden.Actor,
		VinculoAutenticacionActor: e.orden.VinculoAutenticacionActor,
		Esperada:                  esperada, Contenido: contenido, MotivoCatalogo: motivo,
		CorrelacionRef: e.orden.CorrelacionRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	solicitud := e.confirmador.ultima
	if solicitud == nil || solicitud.Version.Contenido.Titulo != contenido.Titulo ||
		!reflect.DeepEqual(solicitud.Version.Configuracion, e.inicial.Configuracion) {
		t.Fatal("la actualizacion sustituyo la configuracion/documentos fijados")
	}
}
