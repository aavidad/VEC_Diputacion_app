package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type registroIntentoDocumentosPrueba struct {
	orden    vecports.OrdenIntentoAuditoria
	llamadas int
	vivo     bool
}

func (r *registroIntentoDocumentosPrueba) AppendIntentoAuditoria(ctx context.Context, orden vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	r.llamadas++
	r.orden = orden
	_, plazo := ctx.Deadline()
	r.vivo = ctx.Err() == nil && plazo
	datos, err := orden.Datos()
	if err != nil || !r.vivo {
		return vecports.AcuseIntentoAuditoria{}, vecports.ErrIntentoAuditoriaNoDisponible
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_bolsa_documentos_1", Secuencia: 1,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef,
		RegistradaEn: time.Now().UTC()}, nil
}

type fronteraDocumentosPrueba struct {
	orden    vecports.OrdenAuditoriaFronteraRutaExacta
	llamadas int
	vivo     bool
}

type sesionFalloDocumentosPrueba struct{ err error }

func (s sesionFalloDocumentosPrueba) ResolverContexto(context.Context) (contextoSeguridadComunDesarrollo, error) {
	return contextoSeguridadComunDesarrollo{}, s.err
}

func (r *fronteraDocumentosPrueba) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	r.llamadas++
	r.orden = orden
	_, plazo := ctx.Deadline()
	r.vivo = ctx.Err() == nil && plazo
	if !r.vivo || orden.Validar() != nil {
		return vecports.ErrOrdenAuditoriaFronteraRutaExactaInvalida
	}
	return nil
}

func contextoAuditoriaDocumentosPrueba(t *testing.T, ruta string) (context.Context, context.CancelFunc, *soporteAltaContratacionTemporalDesarrollo, capacidadConsultaContratacionTemporalDesarrollo) {
	t.Helper()
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	capacidad := capacidadConsultaContratacionTemporalDesarrollo{sello: soporte.sello, ruta: ruta,
		metodo: http.MethodGet, principal: principal, contextoOperacion: &contextoOperacionCTDesarrollo{},
		certificadoVerificadoEn: time.Now().UTC().Add(-time.Second), certificadoValidoHasta: time.Now().UTC().Add(time.Hour)}
	base := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidad)
	ctx, cancelar := context.WithCancel(context.WithValue(base, claveCapturaNominalDocumentosBolsa{}, &capturaNominalDocumentosBolsa{}))
	return ctx, cancelar, soporte, capacidad
}

func TestAuditoriaDocumentosBolsaConservaActorTrasCancelacion(t *testing.T) {
	ruta := rutaBolsasRRHHDesarrollo + "/bolsa:01/documentos"
	ctx, cancelar, soporte, _ := contextoAuditoriaDocumentosPrueba(t, ruta)
	defer cancelar()
	if err := soporte.autorizarDocumentosBolsa(ctx, "bolsa:01", accionConsultarDocumentosBolsa); err != nil {
		t.Fatal(err)
	}
	intentos, frontera := &registroIntentoDocumentosPrueba{}, &fronteraDocumentosPrueba{}
	resoluciones := 0
	auditar := nuevoAuditorDocumentosBolsa(func(context.Context) (*soporteAltaContratacionTemporalDesarrollo, error) {
		resoluciones++
		return nil, errors.New("no se debe resolver de nuevo")
	}, intentos, "vec-rrhh", frontera)
	if auditar == nil {
		t.Fatal("auditor nominal ausente")
	}
	cancelar() // la lectura terminó antes de que el auditor registre el éxito.
	if err := auditar(ctx, "bolsa:01", accionConsultarDocumentosBolsa, nil); err != nil {
		t.Fatalf("lectura completada sin auditoría: %v", err)
	}
	datos, err := intentos.orden.Datos()
	if err != nil || intentos.llamadas != 1 || !intentos.vivo || frontera.llamadas != 0 || resoluciones != 0 ||
		datos.Datos.Accion != accionConsultarDocumentosBolsa || datos.Datos.ModuloID != "bolsa" ||
		datos.Datos.RecursoRef != "bolsa:01" || datos.Datos.FinalidadRef != "gestion_bolsa_rrhh" ||
		datos.Datos.Resultado != vecdomain.ResultadoIntentoAuditoriaExito ||
		datos.ResultadoContexto.Contexto.Principal.ID != soporte.contexto.Resultado.Contexto.Principal.ID ||
		datos.Vinculo.ValidarPara(datos.ResultadoContexto) != nil {
		t.Fatalf("orden nominal incorrecta: llamadas=%d viva=%t frontera=%d resolver=%d actor=%s esperado=%s vinculo=%v datos=%+v error=%v",
			intentos.llamadas, intentos.vivo, frontera.llamadas, resoluciones, datos.ResultadoContexto.Contexto.Principal.ID,
			soporte.contexto.Resultado.Contexto.Principal.ID, datos.Vinculo.ValidarPara(datos.ResultadoContexto), datos.Datos, err)
	}
}

func TestAuditoriaDocumentosBolsaSinCapturaUsaFronteraReal(t *testing.T) {
	ruta := rutaBolsasRRHHDesarrollo + "/bolsa:01/documentos"
	for _, caso := range []struct {
		nombre    string
		sesionErr error
		esperado  error
		motivo    vecports.MotivoAuditoriaFronteraRutaExacta
	}{
		{"sesion_revocada", vecdomain.ErrAutorizacionDenegada, vecdomain.ErrAutorizacionDenegada, vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado},
		{"fuente_caida", ctports.ErrConsultaRRHHNoDisponible, bolsaapp.ErrConsultaDocumentosBolsaNoDisponible, vecports.MotivoAuditoriaFronteraRutaExactaServicioNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ctx, cancelar, soporte, capacidad := contextoAuditoriaDocumentosPrueba(t, ruta)
			defer cancelar()
			soporte.sesionOperativa = sesionFalloDocumentosPrueba{err: caso.sesionErr}
			fallo := soporte.autorizarDocumentosBolsa(ctx, "bolsa:01", accionConsultarDocumentosBolsa)
			if !errors.Is(fallo, caso.esperado) {
				t.Fatalf("fallo de autorización distinto: %v", fallo)
			}
			if _, err := contextoNominalAuditadoDocumentosBolsa(ctx, "bolsa:01", accionConsultarDocumentosBolsa); err == nil {
				t.Fatal("una sesión fallida creó actor nominal")
			}
			intentos, frontera := &registroIntentoDocumentosPrueba{}, &fronteraDocumentosPrueba{}
			auditar := nuevoAuditorDocumentosBolsa(func(context.Context) (*soporteAltaContratacionTemporalDesarrollo, error) {
				return soporte, nil
			}, intentos, "vec-rrhh", frontera)
			cancelar()
			if err := auditar(ctx, "bolsa:01", accionConsultarDocumentosBolsa, fallo); err != nil {
				t.Fatalf("rechazo sin apunte de frontera: %v", err)
			}
			if intentos.llamadas != 0 || frontera.llamadas != 1 || !frontera.vivo ||
				frontera.orden.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaDocumentosBolsaRRHH ||
				frontera.orden.Ruta != ruta || frontera.orden.ActorRef != capacidad.principal.ID ||
				frontera.orden.Motivo != caso.motivo {
				t.Fatalf("frontera nominal incorrecta: intentos=%d frontera=%+v viva=%t", intentos.llamadas, frontera.orden, frontera.vivo)
			}
		})
	}
}

func TestRutaAuditoriaDocumentosBolsaRRHHLimites(t *testing.T) {
	base := rutaBolsasRRHHDesarrollo + "/"
	bolsa200 := "bolsa:" + strings.Repeat("a", 194)
	acta512 := "acta:" + strings.Repeat("a", 507)
	for _, ruta := range []string{
		base + "bolsa:01/documentos", base + bolsa200 + "/documentos/" + acta512 + "/recibo",
	} {
		if !vecdomain.RutaAuditoriaDocumentosBolsaRRHHValida(ruta) {
			t.Fatalf("ruta legítima rechazada: len=%d", len(ruta))
		}
	}
	for _, ruta := range []string{
		base + "bolsa:01/documentos?limite=1",
		base + "bolsa:01/documentos/acta%3A01/recibo",
		base + "bolsa:01/documentos/../recibo",
		base + "bolsa:01/ajena",
		base + "bolsa:" + strings.Repeat("a", 195) + "/documentos",
		base + "bolsa:01/documentos/acta:" + strings.Repeat("a", 508) + "/recibo",
	} {
		if vecdomain.RutaAuditoriaDocumentosBolsaRRHHValida(ruta) {
			t.Fatalf("ruta ajena admitida: len=%d", len(ruta))
		}
	}
}
