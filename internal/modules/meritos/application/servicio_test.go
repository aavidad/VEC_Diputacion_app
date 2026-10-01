package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestDeclaracionYRecuperacionExigenAutoridadCadaVez(t *testing.T) {
	s, solicitud, auth, registro, audit := escenario(t)
	primero, err := s.Declarar(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := s.Declarar(context.Background(), solicitud)
	if err != nil || !reflect.DeepEqual(primero, segundo) || auth.llamadas != 2 || registro.confirmaciones != 1 || registro.recuperaciones != 1 || audit.llamadas != 0 {
		t.Fatal("reintento cambió historia o evitó autoridad", err)
	}
	auth.err = vec.ErrAutorizacionDenegada
	if _, err := s.Declarar(context.Background(), solicitud); !errors.Is(err, vec.ErrAutorizacionDenegada) || registro.recuperaciones != 1 || audit.llamadas != 1 {
		t.Fatal("replay sin autoridad vigente", err)
	}
}

func TestVerificarNoAcreditaNiConsultaRegistro(t *testing.T) {
	s, solicitud, auth, registro, audit := escenario(t)
	if _, err := s.Verificar(context.Background(), solicitud); !errors.Is(err, ErrAcreditacionPendiente) || auth.llamadas != 1 || registro.lecturas != 0 || registro.confirmaciones != 0 || audit.llamadas != 1 {
		t.Fatal("verificación produjo efecto o perdió pendiente", err)
	}
}

func TestSolicitudCruzaVinculoAntesDeAutoridad(t *testing.T) {
	s, solicitud, auth, registro, _ := escenario(t)
	solicitud.Vinculo = vec.VinculoAutenticacionActorV2{}
	if _, err := s.Declarar(context.Background(), solicitud); !errors.Is(err, vec.ErrAutorizacionDenegada) || auth.llamadas != 0 || registro.lecturas != 0 {
		t.Fatal(err)
	}
}

func TestMaterialFueraDeAudienciaYCamposFallaCerrado(t *testing.T) {
	for _, caso := range []string{"audiencia", "campos", "caducada"} {
		t.Run(caso, func(t *testing.T) {
			s, solicitud, auth, registro, audit := escenario(t)
			auth.defecto = caso
			if _, err := s.Declarar(context.Background(), solicitud); err == nil || registro.lecturas != 0 || registro.confirmaciones != 0 || audit.llamadas != 1 {
				t.Fatal("material ajeno alcanzó registro", err)
			}
		})
	}
}

func TestResultadoInciertoYReciboAlteradoNoConfirman(t *testing.T) {
	for _, caso := range []string{"commit", "recibo", "audit"} {
		t.Run(caso, func(t *testing.T) {
			s, solicitud, auth, registro, audit := escenario(t)
			switch caso {
			case "commit":
				registro.err = ports.ErrRegistroNoDisponible
			case "recibo":
				registro.mutar = true
			case "audit":
				auth.err = vec.ErrAutorizacionDenegada
				audit.err = errors.New("test")
			}
			if r, err := s.Declarar(context.Background(), solicitud); err == nil || r.Referencia != "" {
				t.Fatal("resultado no confirmado presentado como éxito", err)
			}
		})
	}
}

func TestRectificacionNoConservaAcreditacionYRevisionSeparaPersonas(t *testing.T) {
	_, solicitud, _, _, _ := escenario(t)
	actual := ports.RegistroActual{Hecho: copiarHecho(solicitud.Hecho), DeclaranteRef: solicitud.Hecho.PersonaRef}
	actual.Hecho.Estado = domain.Acreditado
	actual.Hecho.Revision = &domain.Revision{Referencia: "revision:previa", ActorRef: "persona:revisor", MotivoRef: "motivo:previo", Fecha: instante.Format(time.RFC3339)}
	solicitud.VersionEsperada, solicitud.Hecho.Version = 1, 2
	solicitud.Hecho.Denominacion = "Curso rectificado"
	o, _ := ordenSolicitud(solicitud, accionRectificar)
	cambio, err := prepararCambio(o, &actual, instante)
	if err != nil || cambio.Nuevo.Hecho.Estado != domain.Pendiente || cambio.Nuevo.Hecho.Revision != nil || actual.Hecho.Estado != domain.Acreditado {
		t.Fatal("rectificación acreditada o historia alterada", err)
	}
	actual.Hecho.Estado, actual.Hecho.Revision = domain.Declarado, nil
	solicitud.Hecho = copiarHecho(actual.Hecho)
	solicitud.Hecho.Version = 2
	o, _ = ordenSolicitud(solicitud, accionRechazar)
	if _, err := prepararCambio(o, &actual, instante); !errors.Is(err, vec.ErrAutorizacionDenegada) {
		t.Fatal("autorrevisión aceptada", err)
	}
	o.ActorRef = "persona:revisor"
	cambio, err = prepararCambio(o, &actual, instante)
	if err != nil || cambio.Nuevo.Hecho.Estado != domain.Rechazado || cambio.Nuevo.DeclaranteRef != actual.DeclaranteRef {
		t.Fatal("rechazo pierde declarante", err)
	}
	o.Hecho.PersonaRef = "persona:ajena"
	if _, err := prepararCambio(o, &actual, instante); err == nil {
		t.Fatal("rectifica identidad")
	}
}

func TestHuellaLigaMotivoVersionFuenteCorteYContenido(t *testing.T) {
	_, solicitud, _, _, _ := escenario(t)
	original, _ := ordenSolicitud(solicitud, accionDeclarar)
	for _, mutar := range []func(*Solicitud){
		func(s *Solicitud) { s.Motivo.CatalogoVersion++ },
		func(s *Solicitud) { s.Hecho.Procedencia.Version = "2" },
		func(s *Solicitud) { s.FechaCorte = "2026-10-02" },
		func(s *Solicitud) { s.Hecho.Denominacion = "Contenido diferente" },
	} {
		copia := solicitud
		copia.Hecho = copiarHecho(copia.Hecho)
		mutar(&copia)
		modificado, _ := ordenSolicitud(copia, accionDeclarar)
		if modificado.HuellaComando == original.HuellaComando {
			t.Fatal("huella no liga significado")
		}
	}
}

func TestConstructorRechazaDependenciasAusentesOTipadasNulas(t *testing.T) {
	var auth *autorizadorPrueba
	if s, err := NuevoServicio(auth, nil, nil, nil); err == nil || s != nil {
		t.Fatal("constructor permisivo")
	}
}
