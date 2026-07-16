package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type escenarioFlujoFirmaConcurrencia struct {
	repositorio   *RepositorioFlujosFirmaBaremacion
	consulta      puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion
	arrendamiento puertosbolsa.ArrendamientoFlujoFirmaBaremacion
	siguiente     puertosbolsa.ExpedienteFlujoFirmaBaremacion
}

func nuevoEscenarioFlujoFirmaConcurrencia(t *testing.T, intento int) escenarioFlujoFirmaConcurrencia {
	t.Helper()
	ctx := context.Background()
	reloj := &relojMemoriaPrueba{instante: instanteMemoriaPrueba.Add(time.Duration(intento) * time.Minute)}
	sellador := selladorEstadoFlujoMemoriaPrueba{}
	repositorio, err := NuevoRepositorioFlujosFirmaBaremacion(reloj, sellador)
	if err != nil {
		t.Fatalf("crear repositorio: %v", err)
	}
	protector, err := NuevoProtectorEstadoFlujoFirmaBaremacion(
		"clave-estado-flujo-firma-concurrente-v1",
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatalf("crear protector: %v", err)
	}
	carga, err := puertosbolsa.NuevaCargaProtegida([]byte("estado-flujo-firma-concurrente"))
	if err != nil {
		t.Fatalf("crear carga protegida: %v", err)
	}
	estado, err := protector.ProtegerEstadoFlujoFirmaBaremacion(ctx, carga)
	if err != nil {
		t.Fatalf("proteger estado: %v", err)
	}
	ahora := reloj.Ahora()
	expediente := sellarExpedienteFlujoMemoriaPrueba(t, sellador, puertosbolsa.ExpedienteFlujoFirmaBaremacion{
		FlujoRef: "flujo-firma-concurrente-001", Version: 1,
		IndiceIdempotenciaHMAC: hmacEstadoFlujoMemoriaPrueba("indice-concurrente"),
		HuellaSolicitudHMAC:    hmacEstadoFlujoMemoriaPrueba("solicitud-concurrente"),
		VinculoActorHMAC:       hmacEstadoFlujoMemoriaPrueba("actor-concurrente"),
		PerfilActorClave:       perfilBaremacionMemoriaPrueba,
		ProcesoRef:             "proceso-firma-concurrente-001",
		SolicitudRef:           "solicitud-firma-concurrente-001",
		BaremacionMeritoRef:    "baremacion-firma-concurrente-001",
		DecisionRef:            "decision-firma-concurrente-001",
		Estado:                 puertosbolsa.EstadoExpedienteFirmaPreparando,
		EstadoProtegido:        estado,
		CreadoEn:               ahora,
		ActualizadoEn:          ahora,
	})
	if _, err := repositorio.CrearORecuperarFlujoFirmaBaremacion(
		ctx, puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion{Expediente: expediente},
	); err != nil {
		t.Fatalf("crear flujo: %v", err)
	}
	consulta := puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion{
		FlujoRef: expediente.FlujoRef, IndiceIdempotenciaHMAC: expediente.IndiceIdempotenciaHMAC,
		VinculoActorHMAC: expediente.VinculoActorHMAC,
	}
	adquirido, err := repositorio.AdquirirArrendamientoFlujoFirmaBaremacion(
		ctx, puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion{
			Consulta: consulta, VersionEsperada: 1,
			PropietarioRef: "propietario-firma-concurrente", Duracion: time.Minute,
		},
	)
	if err != nil {
		t.Fatalf("adquirir arrendamiento: %v", err)
	}
	siguiente, err := expediente.Clonar()
	if err != nil {
		t.Fatalf("clonar expediente: %v", err)
	}
	siguiente.Version = 2
	siguiente.ActualizadoEn = ahora
	siguiente.PuntosControl = append(siguiente.PuntosControl, puertosbolsa.PuntoControlFirmaBaremacion{
		Paso: puertosbolsa.PasoPrepararFirmaBaremacion, Estado: puertosbolsa.EstadoPuntoControlFirmaDeclarado,
		EfectoRef:             "efecto-firma-concurrente-001",
		ClaveIdempotenciaHMAC: hmacEstadoFlujoMemoriaPrueba("efecto-concurrente"),
		DeclaradoEn:           ahora,
	})
	siguiente.SelloEstadoHMAC = ""
	siguiente = sellarExpedienteFlujoMemoriaPrueba(t, sellador, siguiente)
	return escenarioFlujoFirmaConcurrencia{
		repositorio: repositorio, consulta: consulta,
		arrendamiento: adquirido.Arrendamiento, siguiente: siguiente,
	}
}

func TestRepositorioFlujoFirmaGuardarYLiberarSeLinealizan(t *testing.T) {
	type resultadoGuardar struct {
		expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion
		err        error
	}
	const intentos = 16
	for intento := 0; intento < intentos; intento++ {
		escenario := nuevoEscenarioFlujoFirmaConcurrencia(t, intento)
		inicio := make(chan struct{})
		guardados := make(chan resultadoGuardar, 1)
		liberaciones := make(chan error, 1)
		go func() {
			<-inicio
			expediente, err := escenario.repositorio.GuardarFlujoFirmaBaremacion(
				context.Background(), puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion{
					VersionEsperada: 1, Arrendamiento: escenario.arrendamiento,
					Siguiente: escenario.siguiente,
				},
			)
			guardados <- resultadoGuardar{expediente: expediente, err: err}
		}()
		go func() {
			<-inicio
			liberaciones <- escenario.repositorio.LiberarArrendamientoFlujoFirmaBaremacion(
				context.Background(), puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{
					Arrendamiento: escenario.arrendamiento,
				},
			)
		}()
		close(inicio)

		guardado := <-guardados
		if err := <-liberaciones; err != nil {
			t.Fatalf("intento %d: liberar arrendamiento: %v", intento, err)
		}
		versionEsperada, puntosEsperados := uint64(1), 0
		switch {
		case guardado.err == nil:
			versionEsperada, puntosEsperados = 2, 1
			if guardado.expediente.Version != 2 || len(guardado.expediente.PuntosControl) != 1 {
				t.Fatalf("intento %d: guardado concurrente incompleto: %+v", intento, guardado.expediente)
			}
		case errors.Is(guardado.err, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido):
			// Liberar obtuvo el mutex antes que Guardar: el guardado debe fallar cerrado.
		default:
			t.Fatalf("intento %d: guardar devolvio un error no linealizable: %v", intento, guardado.err)
		}

		persistido, err := escenario.repositorio.ObtenerFlujoFirmaBaremacion(
			context.Background(), escenario.consulta,
		)
		if err != nil || persistido.Version != versionEsperada ||
			len(persistido.PuntosControl) != puntosEsperados {
			t.Fatalf("intento %d: estado final incoherente: version=%d puntos=%d error=%v",
				intento, persistido.Version, len(persistido.PuntosControl), err)
		}
		escenario.repositorio.mu.RLock()
		_, existeArrendamiento := escenario.repositorio.arrendamientos[escenario.arrendamiento.FlujoRef]
		escenario.repositorio.mu.RUnlock()
		if existeArrendamiento {
			t.Fatalf("intento %d: el arrendamiento no fue liberado", intento)
		}
		if err := escenario.repositorio.LiberarArrendamientoFlujoFirmaBaremacion(
			context.Background(), puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion{
				Arrendamiento: escenario.arrendamiento,
			},
		); err != nil {
			t.Fatalf("intento %d: liberacion idempotente: %v", intento, err)
		}
	}
}
