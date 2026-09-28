package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type validadorMotivoAuditoriaPrueba struct{ llamadas int }

func TestSoportesAuditoriaConsultaDerivanPerfilesYOperacionesDedicados(t *testing.T) {
	directorio, base, principal, ahora := fixtureSoporteSesionBorradorBolsa(t)
	escribirManifiestoIdentidadBorradorBolsa(t, directorio, principal, ahora, nil)
	bolsa, err := nuevoSoporteSesionBorradorBolsaDesarrollo(directorio, base, ahora)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		ExecutionProfile:       config.ExecutionProfileDevelopment,
		AuthMode:               config.AuthModeDevelopment,
		DevelopmentGuard:       config.DevelopmentGuardAcknowledgement,
		DevelopmentMaterialDir: directorio,
	}
	soportes, err := nuevosSoportesAuditoriaConsultaDesarrollo(cfg,
		&dependenciasAltaContratacionTemporalDesarrollo{soporte: base}, bolsa,
		relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	if soportes.CT == nil || soportes.Bolsa == nil ||
		!perfilActivoSeguridadComunValido(soportes.PerfilCT) ||
		!perfilActivoSeguridadComunValido(soportes.PerfilBolsa) ||
		soportes.PerfilCT == soportes.PerfilBolsa ||
		soportes.PerfilCT == base.contexto.Resultado.Contexto.PerfilActivoRef ||
		soportes.PerfilBolsa == bolsa.soporteCanal.contexto.Resultado.Contexto.PerfilActivoRef ||
		soportes.OperacionContextoCT == soportes.OperacionContextoBolsa ||
		soportes.OperacionContextoCT == operacionContextoContratacionTemporalDesarrollo(base) {
		t.Fatalf("contextos de Auditoría no separados: %+v", soportes)
	}
	if _, err := nuevosSoportesAuditoriaConsultaDesarrollo(config.Config{},
		&dependenciasAltaContratacionTemporalDesarrollo{soporte: base}, bolsa,
		relojContratacionTemporalDesarrollo{}); !errors.Is(err, errAutoridadesAuditoriaConsultaDesarrollo) {
		t.Fatalf("soportes accesibles fuera de desarrollo: %v", err)
	}
}

func (v *validadorMotivoAuditoriaPrueba) ValidarReferenciaMotivoAutorizacionV2(context.Context, vecdomain.ReferenciaEntradaCatalogo, time.Time) error {
	v.llamadas++
	return nil
}

func TestOpcionesAuditoriaConsultaLeenCatalogoDemoYValidanMotivo(t *testing.T) {
	t.Setenv(config.EnvAuditoriaConsultaCatalogoPath, "../../../data/demo/reglas/auditoria_consulta.ejemplo.demo.json")
	cfg := config.Config{ExecutionProfile: config.ExecutionProfileDevelopment, AuthMode: config.AuthModeDevelopment, DevelopmentGuard: config.DevelopmentGuardAcknowledgement}
	validador := &validadorMotivoAuditoriaPrueba{}
	proveedor, opciones, err := nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(context.Background(), cfg, relojContratacionTemporalDesarrollo{}, validador)
	if err != nil || proveedor == nil || !opciones.EsEjemplo || opciones.PermisoRequerido != auditoria.AccionConsultar ||
		opciones.FinalidadRef != "revision_administrativa_auditoria_rrhh" || opciones.MotivoRef == "" || validador.llamadas != 1 {
		t.Fatalf("catalogo de consulta no configurado: opciones=%+v, llamadas=%d, error=%v", opciones, validador.llamadas, err)
	}
	if _, _, err := nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(context.Background(), config.Config{}, relojContratacionTemporalDesarrollo{}, validador); !errors.Is(err, errAutoridadesAuditoriaConsultaDesarrollo) {
		t.Fatalf("catalogo DEMO disponible fuera de desarrollo: %v", err)
	}
}

func TestInstantaneasAuditoriaConsultaSeparanFuenteExpedienteYCampos(t *testing.T) {
	ahora := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	finalidad := "revision_administrativa_auditoria_rrhh"
	casos := []struct{ fuente, perfil, expediente string }{
		{"ct", "prf_auditoria_ct_sintetico_001", "expediente:ct:sintetico:001"},
		{"bolsa", "prf_auditoria_bolsa_sintetico_001", "participacion:bolsa:sintetica:001"},
	}
	for _, caso := range casos {
		t.Run(caso.fuente, func(t *testing.T) {
			i, err := instantaneaAuditoriaConsultaNominalDesarrollo("principal:sintetico:rrhh", caso.perfil, caso.fuente, caso.expediente, finalidad, ahora)
			if err != nil || i.Validar() != nil {
				t.Fatalf("instantanea inválida: %v", err)
			}
			if i.AsignacionPerfil.PerfilActivoRef != caso.perfil || len(i.AsignacionPerfil.Ambitos) != 2 ||
				i.AsignacionPerfil.Ambitos[0].Clave != "expediente_ref" || !reflect.DeepEqual(i.AsignacionPerfil.Ambitos[0].Valores, []string{caso.expediente}) ||
				i.AsignacionPerfil.Ambitos[1].Clave != "fuente" || !reflect.DeepEqual(i.AsignacionPerfil.Ambitos[1].Valores, []string{caso.fuente}) {
				t.Fatalf("alcance nominal ampliado: %+v", i.AsignacionPerfil)
			}
			c := i.VersionRol.Concesiones
			if len(c) != 1 || c[0].Accion != auditoria.AccionConsultar || c[0].ModuloID != auditoria.ModuloAutorizacion ||
				c[0].TipoRecurso != auditoria.TipoRecurso || !reflect.DeepEqual(c[0].Finalidades, []string{finalidad}) ||
				!reflect.DeepEqual(c[0].CamposPermitidos, auditoria.CamposPermitidos()) {
				t.Fatalf("concesión nominal ampliada: %+v", c)
			}
		})
	}
	for _, invalido := range []struct{ fuente, perfil, expediente string }{
		{"ct", "auditoria_ct", "expediente:ct:sintetico:001"},
		{"bolsa", "consulta_auditoria_bolsa", "participacion:bolsa:sintetica:001"},
		{"otro", "prf_auditoria_ct_sintetico_001", "expediente:ct:sintetico:001"},
		{"ct", "prf_auditoria_ct_sintetico_001", ""},
	} {
		if _, err := instantaneaAuditoriaConsultaNominalDesarrollo("principal:sintetico:rrhh", invalido.perfil, invalido.fuente, invalido.expediente, finalidad, ahora); !errors.Is(err, errAutoridadesAuditoriaConsultaDesarrollo) {
			t.Fatalf("alcance inválido aceptado: %+v: %v", invalido, err)
		}
	}
}

type autorizadorAuditoriaDelegadoPrueba struct{ llamadas int }

func (a *autorizadorAuditoriaDelegadoPrueba) ExigirSolicitudLigadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	a.llamadas++
	return vecdomain.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, auditoria.ErrNoDisponible
}

func TestAutorizadorAuditoriaConsultaNominalNoCruzaFuenteNiExpediente(t *testing.T) {
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar")
	base, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	delegado := &autorizadorAuditoriaDelegadoPrueba{}
	a := autorizadorAuditoriaConsultaNominalDesarrollo{
		delegado: delegado, fuente: "ct", perfil: escenario.resultado.Contexto.PerfilActivoRef,
		expediente: "expediente:ct:sintetico:001", finalidad: base.Finalidad, motivo: base.ReferenciaMotivo,
	}
	peticion := func(fuente, expediente, finalidad string) vecdomain.SolicitudAutorizacionLigadaV3 {
		t.Helper()
		d := base
		d.Accion, d.Finalidad = auditoria.AccionConsultar, finalidad
		d.Recurso = vecdomain.RecursoAutorizable{
			Referencia: expediente, ModuloID: auditoria.ModuloAutorizacion, Tipo: auditoria.TipoRecurso,
			Ambitos:   map[string]string{"expediente_ref": expediente, "fuente": fuente},
			Atributos: map[string]string{"filtro_sha256": strings.Repeat("a", 64)},
		}
		s, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(d)
		if err != nil {
			t.Fatalf("solicitud: %v", err)
		}
		return s
	}
	for _, s := range []vecdomain.SolicitudAutorizacionLigadaV3{
		peticion("bolsa", a.expediente, a.finalidad),
		peticion("ct", "expediente:ct:ajeno", a.finalidad),
		peticion("ct", a.expediente, "otra_finalidad"),
	} {
		if _, _, err := a.ExigirSolicitudLigadaV3(context.Background(), s, escenario.resultado); !errors.Is(err, auditoria.ErrDenegada) {
			t.Fatalf("solicitud ajena no denegada: %v", err)
		}
	}
	if delegado.llamadas != 0 {
		t.Fatalf("solicitud ajena alcanzó PDP: %d", delegado.llamadas)
	}
	if _, _, err := a.ExigirSolicitudLigadaV3(context.Background(), peticion("ct", a.expediente, a.finalidad), escenario.resultado); !errors.Is(err, auditoria.ErrNoDisponible) || delegado.llamadas != 1 {
		t.Fatalf("solicitud nominal no alcanzó PDP: %v, llamadas=%d", err, delegado.llamadas)
	}
}
