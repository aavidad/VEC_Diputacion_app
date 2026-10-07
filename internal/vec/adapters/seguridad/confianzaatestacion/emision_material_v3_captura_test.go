package confianzaatestacion

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type fuenteEmisionConCapturaPrueba struct {
	instantanea domain.InstantaneaAutorizacion
	consultas   int
}

func (f *fuenteEmisionConCapturaPrueba) ObtenerInstantaneaAutorizacion(context.Context, string, string) (domain.InstantaneaAutorizacion, error) {
	f.consultas++
	return f.instantanea, nil
}

type registroEmisionConCapturaPrueba struct {
	instante  time.Time
	registros int
}

func (r *registroEmisionConCapturaPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, ports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	r.registros++
	return r.instante, nil
}

type denegacionesEmisionConCapturaPrueba struct{}

func (denegacionesEmisionConCapturaPrueba) RegistrarDenegacionAutorizacionLigadaV3(context.Context, ports.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	return nil
}

type motivoEmisionConCapturaPrueba struct{}

func (motivoEmisionConCapturaPrueba) ValidarReferenciaMotivoAutorizacionV2(context.Context, domain.ReferenciaEntradaCatalogo, time.Time) error {
	return nil
}

type decisionRefEmisionConCapturaPrueba struct{}

func (decisionRefEmisionConCapturaPrueba) NuevaReferenciaDecisionAutorizacion() (string, error) {
	return "dec_0123456789abcdef0123456789abcdef", nil
}

type capturaCancelacionEmisionPrueba struct{ cancelar context.CancelFunc }

func (capturaCancelacionEmisionPrueba) String() string         { return "captura sintetica" }
func (c capturaCancelacionEmisionPrueba) LogValue() slog.Value { return slog.StringValue(c.String()) }
func (c capturaCancelacionEmisionPrueba) LigarMaterial(domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2, domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, string) (ports.CapturaEvaluacionSolicitudLigadaV3, error) {
	c.cancelar()
	return c, nil
}
func (capturaCancelacionEmisionPrueba) InstantaneaPara(domain.SolicitudAutorizacionLigadaV3, domain.ResultadoContextoActorRegistradoV2, domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, string, time.Time) (domain.InstantaneaAutorizacion, error) {
	return domain.InstantaneaAutorizacion{}, errors.New("solo prueba")
}

type autorizadorCancelacionEmisionPrueba struct {
	*autorizadorEmisionMaterialV3Prueba
	captura ports.CapturaEvaluacionSolicitudLigadaV3
}

func (a autorizadorCancelacionEmisionPrueba) ExigirSolicitudLigadaV3ConCaptura(ctx context.Context, solicitud domain.SolicitudAutorizacionLigadaV3, resultado domain.ResultadoContextoActorRegistradoV2) (domain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.CapturaEvaluacionSolicitudLigadaV3, error) {
	decision, confirmacion, err := a.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
	return decision, confirmacion, a.captura, err
}

func instantaneaEmisionConCapturaPrueba(t *testing.T, e escenarioEmisionMaterialV3Prueba) domain.InstantaneaAutorizacion {
	t.Helper()
	ahora := e.base.ahora
	actor := e.base.resultado.Contexto.Instantanea
	solicitud, err := e.base.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	version := domain.VersionRol{
		RolID: "tecnico_rrhh", Version: 1, Nombre: "Tecnico RRHH",
		Estado: domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion:         solicitud.Accion,
			ModuloID:       solicitud.Recurso.ModuloID,
			TipoRecurso:    solicitud.Recurso.Tipo,
			Finalidades:    []string{solicitud.Finalidad},
			GarantiaMinima: domain.AuthAssuranceSubstantial,
		}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	return domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-rrhh", Version: 1,
			PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.PersonaRef,
			VersionRolRef: version.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
			Ambitos: []domain.AmbitoPerfil{
				{Clave: "organizacion_ref", Valores: []string{"organizacion:dipgra"}},
				{Clave: "centro_ref", Valores: []string{"centro:servicios-generales"}},
				{Clave: "categoria_ref", Valores: []string{"categoria:auxiliar-administrativo"}},
			},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: version,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: version.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
}

func TestEmisionMaterialV3ConCapturaRechazaAutorizadorSinCaptura(t *testing.T) {
	e := nuevoEscenarioEmisionMaterialV3Prueba(t)
	_, _, material, captura, err := e.emisorMaterial.EmitirMaterialAutorizacionAtestadaV3ConCaptura(
		context.Background(), e.base.solicitud, e.base.resultado,
	)
	if err == nil || material != nil || captura != nil ||
		e.autorizador.invocaciones != 0 || e.atestador.invocaciones != 0 {
		t.Fatalf("emision sin autorizador de captura no cerro antes del PDP: %v", err)
	}
}

func TestEmisionMaterialV3ConCapturaConservaPDPYMaterialExactos(t *testing.T) {
	e := nuevoEscenarioEmisionMaterialV3Prueba(t)
	fuente := &fuenteEmisionConCapturaPrueba{instantanea: instantaneaEmisionConCapturaPrueba(t, e)}
	registro := &registroEmisionConCapturaPrueba{instante: e.base.ahora}
	servicio, err := application.NuevoServicioAutorizacionSolicitudLigadaV3(
		fuente, registro, denegacionesEmisionConCapturaPrueba{}, motivoEmisionConCapturaPrueba{},
		&relojConfianzaAtestacionV3Prueba{ahora: e.base.ahora}, decisionRefEmisionConCapturaPrueba{},
		application.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatal(err)
	}
	emisor, err := NuevoEmisorMaterialAutorizacionAtestadaV3(
		servicio, e.atestador, e.confianza, e.emisorCapacidades,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, confirmacion, material, captura, err := emisor.EmitirMaterialAutorizacionAtestadaV3ConCaptura(
		context.Background(), e.base.solicitud, e.base.resultado,
	)
	if err != nil || material == nil || captura == nil || fuente.consultas != 1 || registro.registros != 1 {
		t.Fatalf("emision no conservo PDP y CAS unicos: %v", err)
	}
	huella, err := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	huellaBase, errBase := domain.HuellaSHA256DecisionAutorizacionV3(e.base.decision)
	if err != nil || errBase != nil || huella != huellaBase {
		t.Fatalf("PDP altero decision del escenario: %v / %v", err, errBase)
	}
	exportacion, err := material.ExportarMaterialParaConsumidor()
	if err != nil {
		t.Fatal(err)
	}
	audiencia := e.emisorCapacidades.clave.audienciaConsumo
	ahora := e.relojCapacidad.ahora.Add(time.Microsecond)
	primera, err := captura.InstantaneaPara(e.base.solicitud, e.base.resultado, decision, confirmacion, exportacion, audiencia, ahora)
	if err != nil {
		t.Fatalf("captura material real: %v", err)
	}
	primera.AsignacionPerfil.Ambitos[0].Valores[0] = "alterado"
	segunda, err := captura.InstantaneaPara(e.base.solicitud, e.base.resultado, decision, confirmacion, exportacion, audiencia, ahora)
	if err != nil || segunda.AsignacionPerfil.Ambitos[0].Valores[0] != "organizacion:dipgra" {
		t.Fatalf("getter compartio memoria con consumidor: %v", err)
	}
	if _, err := captura.InstantaneaPara(e.base.solicitud, e.base.resultado, decision, confirmacion, exportacion, "otra-audiencia", ahora); !errors.Is(err, application.ErrCapturaEvaluacionSolicitudLigadaV3Invalida) {
		t.Fatalf("getter acepto audiencia ajena: %v", err)
	}
	if _, err := captura.InstantaneaPara(e.base.solicitud, e.base.resultado, decision, confirmacion, exportacion, audiencia, e.base.ahora.Add(3*time.Minute)); !errors.Is(err, application.ErrCapturaEvaluacionSolicitudLigadaV3Invalida) {
		t.Fatalf("getter acepto ventana vencida: %v", err)
	}
	payload := exportacion.PayloadVECAD3()
	payload[len(payload)-1] ^= 1
	materialAlterado, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		exportacion.CapacidadCanonica(), exportacion.ResumenCapacidad(),
		exportacion.DecisionCanonica(), exportacion.MotivoCanonico(), exportacion.ContextoActorCanonico(),
		exportacion.PersonaVersion(), exportacion.PerfilVersion(), payload,
		exportacion.SobreCOSESign1(), exportacion.EvidenciaVerificacion(), exportacion.RaizPublicaSPKI(),
	)
	if err != nil {
		t.Fatalf("no se construyo el material estructural alterado: %v", err)
	}
	if _, err := captura.InstantaneaPara(e.base.solicitud, e.base.resultado, decision, confirmacion, materialAlterado, audiencia, ahora); !errors.Is(err, application.ErrCapturaEvaluacionSolicitudLigadaV3Invalida) {
		t.Fatalf("getter acepto material alterado: %v", err)
	}
}

func TestEmisionMaterialV3ConCapturaCancelaTrasLigaduraSinEntregarMaterial(t *testing.T) {
	e := nuevoEscenarioEmisionMaterialV3Prueba(t)
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	autorizador := autorizadorCancelacionEmisionPrueba{
		autorizadorEmisionMaterialV3Prueba: e.autorizador,
		captura:                            capturaCancelacionEmisionPrueba{cancelar: cancelar},
	}
	emisor, err := NuevoEmisorMaterialAutorizacionAtestadaV3(
		autorizador, e.atestador, e.confianza, e.emisorCapacidades,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, confirmacion, material, captura, err := emisor.EmitirMaterialAutorizacionAtestadaV3ConCaptura(
		ctx, e.base.solicitud, e.base.resultado,
	)
	if !errors.Is(err, context.Canceled) || material != nil || captura != nil ||
		decision.ValidarPara(e.base.solicitud) != nil || confirmacion.Validar() != nil ||
		e.autorizador.invocaciones != 1 {
		t.Fatalf("cancelacion posterior a ligadura no preservo durable sin material: %v", err)
	}
}
