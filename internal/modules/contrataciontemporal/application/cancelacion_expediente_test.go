package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type reglasCancelacionDoble struct {
	regla ports.ReglaCancelacion
	base  reglasSeguimientoDoble
}

func (r reglasCancelacionDoble) ReglaCancelacion(context.Context, time.Time) (ports.ReglaCancelacion, ports.PoliticaOperacionSeguimiento, error) {
	return r.regla, r.base.politica("motivos_cancelacion_contratacion_temporal"), nil
}

type lectorCancelacionDoble struct{}

func (lectorCancelacionDoble) ConsultarEstadoCancelacion(_ context.Context, _, exp string) (ports.EstadoCancelacionExpediente, error) {
	return ports.EstadoCancelacionExpediente{ExpedienteRef: exp}, nil
}

type escenarioCancelacion struct {
	servicio    *ServicioCancelacionExpediente
	repo        *repositorioSeguimientoDoble
	autorizador *autorizadorSeguimientoDoble
	canal       ContextoCanalSeguimiento
}

func expedienteFixtureAplicacion(t *testing.T, fichero string) domain.Expediente {
	t.Helper()
	contenido, err := os.ReadFile("../domain/testdata/" + fichero)
	if err != nil {
		t.Fatal(err)
	}
	var e domain.Expediente
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		t.Fatal(err)
	}
	return e
}

func nuevoEscenarioCancelacion(t *testing.T, fichero string, canal domain.CanalCancelacion) escenarioCancelacion {
	t.Helper()
	e := expedienteFixtureAplicacion(t, fichero)
	ahora := e.ActualizadoEn.Add(time.Hour)
	contexto := contextoAutorizacionAltaV3Prueba(t, ahora)
	v, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	reglas := reglasCancelacionDoble{base: reglasSeguimientoDoble{ahora: ahora}, regla: ports.ReglaCancelacion{
		Fases: []domain.ClaveFase{"solicitud", "asignacion_unidad", "informe_juridico"},
		Motivos: []ports.MotivoCancelacion{
			{Clave: "necesidad_desaparecida", Etiqueta: "Ha desaparecido la necesidad", Canales: []domain.CanalCancelacion{"centro", "rrhh"}},
			{Clave: "falta_credito", Etiqueta: "Sin crédito disponible", Canales: []domain.CanalCancelacion{"rrhh"}},
		}}}
	repo := &repositorioSeguimientoDoble{expediente: e}
	autorizador := &autorizadorSeguimientoDoble{}
	s, err := NuevoServicioCancelacionExpediente(DependenciasCancelacionExpediente{Canal: canal, Contextos: contextosSeguimientoDoble{contexto},
		Sellos: sellosSeguimientoDoble{}, Repositorio: repo, Reglas: reglas, Autorizador: autorizador, Referencias: referenciasSeguimientoDoble{},
		Lector: lectorCancelacionDoble{}, Reloj: relojSeguimientoFijo{ahora}})
	if err != nil {
		t.Fatal(err)
	}
	return escenarioCancelacion{servicio: s, repo: repo, autorizador: autorizador,
		canal: ContextoCanalSeguimiento{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef, OrganizacionRef: e.OrganizacionRef}}
}

func TestCancelacionRRHHLigaMotivoFasesYContextoAutorizado(t *testing.T) {
	e := nuevoEscenarioCancelacion(t, "expediente_asignacion_v3.json", domain.CanalCancelacionRRHH)
	sol := SolicitudCancelarExpediente{Canal: e.canal, ExpedienteRef: e.repo.expediente.Referencia, VersionEsperada: 3,
		ClaveIdempotencia: "44444444-4444-4444-8444-444444444444", MotivoClave: "falta_credito", Observaciones: "Sin partida este ejercicio"}
	recibo, err := e.servicio.CancelarExpediente(context.Background(), sol)
	if err != nil || recibo.EstadoResultante != domain.EstadoCancelado || recibo.FaseResultante != "asignacion_unidad" || recibo.VersionResultante != 4 {
		t.Fatalf("cancelación: %+v %v", recibo, err)
	}
	orden := e.repo.ordenes[0]
	a := orden.Contexto.Atributos
	if a["canal"] != "rrhh" || a["motivo_clave"] != "falta_credito" || a["fases_admitidas"] != "solicitud,asignacion_unidad,informe_juridico" ||
		a["observaciones_huella_sha256"] != huellaTexto("Sin partida este ejercicio") || a["version_expediente"] != "3" ||
		orden.Contexto.Ambitos["fase_previa"] != "asignacion_unidad" || orden.Contexto.Ambitos["estado_previo"] != "en_curso" ||
		orden.Contexto.Ambitos["centro_ref"] != "" || len(orden.Contexto.Ambitos) != 4 {
		t.Fatalf("contexto autorizado inesperado: %+v %+v", orden.Contexto.Ambitos, a)
	}
	s := e.autorizador.solicitudes[0]
	if s.Audiencia != ports.AudienciaConsumoCancelacionV1 || s.Accion != domain.AccionCancelarExpediente ||
		s.Finalidad != ports.FinalidadCancelarExpediente || s.Recurso.Tipo != ports.TipoRecursoCancelacion {
		t.Fatalf("solicitud de autorización inesperada: %+v", s)
	}
	if orden.Siguiente.Actuaciones[3].Observaciones != "Sin partida este ejercicio" || orden.Siguiente.Actuaciones[3].UnidadRef != e.repo.expediente.UnidadActual() {
		t.Fatalf("actuación inesperada: %+v", orden.Siguiente.Actuaciones[3])
	}
}

func TestCancelacionDelCentroSoloConMotivosDeSuCanalYConSuCentro(t *testing.T) {
	e := nuevoEscenarioCancelacion(t, "expediente_asignacion_v3.json", domain.CanalCancelacionCentro)
	sol := SolicitudCancelarExpediente{Canal: e.canal, ExpedienteRef: e.repo.expediente.Referencia, VersionEsperada: 3,
		ClaveIdempotencia: "55555555-5555-4555-8555-555555555555", MotivoClave: "falta_credito"}
	if _, err := e.servicio.CancelarExpediente(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) || len(e.autorizador.solicitudes) != 0 {
		t.Fatalf("motivo reservado a RRHH: %v", err)
	}
	opciones, err := e.servicio.Opciones(context.Background())
	if err != nil || len(opciones.Motivos) != 1 || opciones.Motivos[0].Clave != "necesidad_desaparecida" {
		t.Fatalf("opciones del centro: %+v %v", opciones, err)
	}
	sol.MotivoClave = "necesidad_desaparecida"
	if _, err := e.servicio.CancelarExpediente(context.Background(), sol); err != nil {
		t.Fatal(err)
	}
	if got := e.repo.ordenes[0].Contexto.Ambitos["centro_ref"]; got != e.repo.expediente.Solicitud.CentroRef || got == "" {
		t.Fatalf("el centro debe quedar en el ámbito autorizado: %q", got)
	}
}

func TestCancelacionTrasLaFiscalizacionSeRechazaSinAutorizar(t *testing.T) {
	e := nuevoEscenarioCancelacion(t, "expediente_nombramiento_v7.json", domain.CanalCancelacionRRHH)
	sol := SolicitudCancelarExpediente{Canal: e.canal, ExpedienteRef: e.repo.expediente.Referencia, VersionEsperada: 7,
		ClaveIdempotencia: "66666666-6666-4666-8666-666666666666", MotivoClave: "necesidad_desaparecida"}
	if _, err := e.servicio.CancelarExpediente(context.Background(), sol); !errors.Is(err, ports.ErrCancelacionTrasFiscalizacion) ||
		len(e.autorizador.solicitudes) != 0 || len(e.repo.ordenes) != 0 {
		t.Fatalf("cancelación tras fiscalización: %v", err)
	}
	if !strings.Contains(ports.ErrCancelacionTrasFiscalizacion.Error(), "fiscalizacion") {
		t.Fatal("mensaje interno inesperado")
	}
}
