package httpinterno

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const causaPrivadaDiagnosticoCoberturaPrueba = "postgres://persona:secreto@privado/rrhh"

type resolutorContextoDiagnosticoCoberturaPrueba struct{}

func (resolutorContextoDiagnosticoCoberturaPrueba) ResolverContextoAutorizacionAltaV3(
	context.Context,
	ports.SolicitudResolverContextoAutorizacionAltaV3,
) (ports.ContextoAutorizacionAltaV3, error) {
	return ports.ContextoAutorizacionAltaV3{}, errors.New(causaPrivadaDiagnosticoCoberturaPrueba)
}

type autorizadorDiagnosticoCoberturaPrueba struct{}

func (autorizadorDiagnosticoCoberturaPrueba) AutorizarPresentacionPropuestaCobertura(
	context.Context,
	ports.SolicitudResolverContextoAutorizacionAltaV3,
	ports.ContextoAutorizacionAltaV3,
	cobertura.SolicitudInstantaneaAnalisisDurableO3,
	time.Time,
) error {
	return nil
}

type lectorDiagnosticoCoberturaPrueba struct{}

func (lectorDiagnosticoCoberturaPrueba) LeerExpedienteAnalisisDurableO3(
	context.Context,
	cobertura.SolicitudInstantaneaAnalisisDurableO3,
) (domain.Expediente, error) {
	return domain.Expediente{}, nil
}

type relojDiagnosticoCoberturaPrueba struct{}

func (relojDiagnosticoCoberturaPrueba) AhoraGobiernoOperacionCobertura(context.Context) (time.Time, error) {
	return time.Now().UTC(), nil
}

type gobiernoDiagnosticoCoberturaPrueba struct{}

func (gobiernoDiagnosticoCoberturaPrueba) ResolverGobiernoOperacionCobertura(
	context.Context,
	cobertura.SolicitudResolucionGobiernoOperacionCobertura,
) (cobertura.PublicacionGobiernoOperacionCobertura, error) {
	return cobertura.PublicacionGobiernoOperacionCobertura{}, nil
}

type motivosDiagnosticoCoberturaPrueba struct{}

func (motivosDiagnosticoCoberturaPrueba) ResolverClave(
	context.Context,
	domain.ClaveCatalogo,
	time.Time,
) (cobertura.ResolucionMotivoDecisionCobertura, error) {
	return cobertura.ResolucionMotivoDecisionCobertura{}, nil
}

type decisorDiagnosticoCoberturaPrueba struct{}

func (decisorDiagnosticoCoberturaPrueba) DecidirParaAdaptador(
	context.Context,
	application.SolicitudDecidirCobertura,
) (application.ResultadoDecisionCoberturaParaAdaptador, error) {
	return application.ResultadoDecisionCoberturaParaAdaptador{}, nil
}

func (decisorDiagnosticoCoberturaPrueba) RectificarParaAdaptador(
	context.Context,
	application.SolicitudRectificarCobertura,
) (application.ResultadoDecisionCoberturaParaAdaptador, error) {
	return application.ResultadoDecisionCoberturaParaAdaptador{}, nil
}

func nuevoServicioDiagnosticoCoberturaPrueba(t *testing.T) *application.ServicioPresentacionPropuestaCobertura {
	t.Helper()
	servicio, err := application.NuevoServicioPresentacionPropuestaCobertura(
		resolutorContextoDiagnosticoCoberturaPrueba{},
		autorizadorDiagnosticoCoberturaPrueba{},
		lectorDiagnosticoCoberturaPrueba{},
		relojDiagnosticoCoberturaPrueba{},
		gobiernoDiagnosticoCoberturaPrueba{},
		motivosDiagnosticoCoberturaPrueba{},
		[]application.MotivoAlternativaCobertura{{
			ViaClave: "bolsa_vigente", Clave: "motivo_prueba", EtiquetaI18n: "contratacion_temporal.cobertura.motivo.prueba",
		}},
		&application.PreparadorGlobalCobertura{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func TestManejadorPropuestaCoberturaRegistraSoloEtapaDiagnosticaInterna(t *testing.T) {
	servicio := nuevoServicioDiagnosticoCoberturaPrueba(t)
	contexto := contextoCoberturaValidoPrueba()
	solicitud := application.SolicitudProponerCobertura{
		AutenticacionRef: contexto.AutenticacionRef, SesionRef: contexto.SesionRef,
		PerfilRef: contexto.PerfilRef, OrganizacionRef: contexto.OrganizacionRef,
		ExpedienteRef: "expediente:ct:0001", VersionEsperada: 2,
	}
	if _, err := servicio.ProponerParaAdaptador(context.Background(), solicitud); !errors.Is(err, application.ErrPresentacionPropuestaCoberturaNoDisponible) {
		t.Fatalf("ProponerParaAdaptador perdió el centinela de indisponibilidad: %v", err)
	} else if etapa, ok := application.EtapaDiagnosticoDePresentacionPropuestaCobertura(err); !ok || etapa != application.EtapaDiagnosticoPresentacionContexto {
		t.Fatalf("ProponerParaAdaptador perdió la etapa interna: etapa=%q ok=%t", etapa, ok)
	}

	var salidaRegistro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&salidaRegistro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	manejador, err := NuevoManejadorCobertura(
		autoridadCoberturaPrueba{contexto: contexto}, servicio, decisorDiagnosticoCoberturaPrueba{},
	)
	if err != nil {
		t.Fatal(err)
	}
	respuesta := httptest.NewRecorder()
	manejador.ServeHTTP(respuesta, nuevaPeticionCoberturaPrueba(
		RutaPropuestaCobertura,
		`{"expediente_ref":"expediente:ct:0001","version_esperada":2}`,
	))
	if respuesta.Code != http.StatusServiceUnavailable || !strings.Contains(respuesta.Body.String(), `"servicio_no_disponible"`) {
		t.Fatalf("el contrato HTTP cambió: estado=%d cuerpo=%s", respuesta.Code, respuesta.Body.String())
	}
	if strings.Contains(respuesta.Body.String(), causaPrivadaDiagnosticoCoberturaPrueba) || strings.Contains(respuesta.Body.String(), string(application.EtapaDiagnosticoPresentacionContexto)) {
		t.Fatalf("la respuesta pública filtró diagnóstico o causa: %s", respuesta.Body.String())
	}

	var registro map[string]any
	if err := json.Unmarshal(salidaRegistro.Bytes(), &registro); err != nil {
		t.Fatalf("registro interno no es JSON: %v (%s)", err, salidaRegistro.String())
	}
	if registro["msg"] != mensajeDiagnosticoPropuestaCoberturaNoDisponible || registro["etapa"] != string(application.EtapaDiagnosticoPresentacionContexto) {
		t.Fatalf("registro no contiene mensaje y etapa fijos: %#v", registro)
	}
	for _, clave := range []string{"error", "causa", "expediente_ref", "sesion_ref", "cuerpo"} {
		if _, existe := registro[clave]; existe {
			t.Fatalf("registro incluyó %q: %#v", clave, registro)
		}
	}
	if strings.Contains(salidaRegistro.String(), causaPrivadaDiagnosticoCoberturaPrueba) {
		t.Fatalf("registro filtró causa privada: %s", salidaRegistro.String())
	}
}

func TestRegistroDiagnosticoPropuestaCoberturaIgnoraErrorAjeno(t *testing.T) {
	var salidaRegistro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&salidaRegistro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	registrarDiagnosticoPropuestaCoberturaNoDisponible(errors.New(causaPrivadaDiagnosticoCoberturaPrueba))
	if salidaRegistro.Len() != 0 {
		t.Fatalf("un error ajeno alcanzó el registro: %s", salidaRegistro.String())
	}
}
