package bootstrap

import (
	"context"
	"errors"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type pdpNoInvocadoGobiernoBaremoPrueba struct {
	llamadas int
	err      error
}

func (p *pdpNoInvocadoGobiernoBaremoPrueba) ExigirSolicitudLigadaV3(context.Context, vd.SolicitudAutorizacionLigadaV3, vd.ResultadoContextoActorRegistradoV2) (vd.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	p.llamadas++
	if p.err != nil {
		return vd.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, p.err
	}
	return vd.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, vd.ErrAutorizacionDenegada
}

// Aísla el contrato de errores del PDP; no acredita un rechazo PostgreSQL.
func TestGobiernoBaremoHTTPDenegacionPDPExigeConfirmacionSinDuplicarAuditor(t *testing.T) {
	perfil := perfilGobiernoReglasBaremoPrueba(t)
	operativo := contextoSeguridadComunDesarrollo{Vinculo: perfil.soporte.contexto.Vinculo, Resultado: perfil.soporte.contexto.Resultado}
	pedido := pedidoGobiernoReglasV3Prueba(t, perfil, "alta_borrador", time.Now().UTC().Truncate(time.Microsecond))
	solicitud, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: pedido.Motivo,
		Accion: pedido.Accion, Recurso: pedido.Recurso, Finalidad: pedido.Finalidad, Correlacion: "corr_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct{ err, esperado error }{
		{vp.ErrDenegacionExplicitaAutorizacionLigadaV3, app.ErrGobiernoV3Prohibido},
		{vd.ErrAutorizacionDenegada, app.ErrGobiernoV3NoDisponible},
		{errors.Join(vd.ErrAutorizacionDenegada, errors.New("registro de denegación no disponible")), app.ErrGobiernoV3NoDisponible},
	} {
		pdp := &pdpNoInvocadoGobiernoBaremoPrueba{err: caso.err}
		auditorias := 0
		broker := &ProveedorGobiernoReglasBaremoV3{pdp: pdp, auditarAntesPDP: func(context.Context, error, *vd.ResultadoContextoActorRegistradoV2) error { auditorias++; return nil }}
		_, err := broker.emitirMaterialTrasPDP(context.Background(), solicitud, operativo, pedido)
		if !errors.Is(err, caso.esperado) || auditorias != 0 || pdp.llamadas != 1 {
			t.Fatalf("clasificación incorrecta o auditor duplicado: %v", err)
		}
	}
}

func TestGobiernoBaremoHTTPAuditaAmbitoAjenoAntesPDP(t *testing.T) {
	for _, falloAuditor := range []error{nil, errors.New("auditor segregado caído")} {
		sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, bolsahttp.RutaAltaGobiernoReglasBaremoV3)
		perfil := &PerfilGobiernoReglasBaremoV3{soporte: sesion.soporte, convocatoriaRef: "convocatoria:prueba", expedienteRef: "expediente:prueba"}
		perfil.plantilla.AsignacionPerfil.PerfilActivoRef = sesion.soporte.contexto.Resultado.Contexto.PerfilActivoRef
		capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		holder := capacidad.contextoOperacion
		holder.soporte = perfil.soporte
		holder.contexto.Vinculo, holder.contexto.Resultado = perfil.soporte.contexto.Vinculo, perfil.soporte.contexto.Resultado
		pdp := &pdpNoInvocadoGobiernoBaremoPrueba{}
		registrador := &auditorConsultaReciboPrueba{err: falloAuditor}
		broker := &ProveedorGobiernoReglasBaremoV3{perfil: perfil, sesion: sesion, pdp: pdp, reloj: relojContratacionTemporalDesarrollo{},
			rutas:           RutasGobiernoReglasBaremoV3{bolsahttp.RutaAltaGobiernoReglasBaremoV3, bolsahttp.RutaConsultaGobiernoReglasBaremoV3, bolsahttp.RutaRecuperarGobiernoReglasBaremoV3},
			auditarAntesPDP: auditorDenegacionAntesPDPGobiernoReglasBaremoHTTPV3(sesion, registrador)}
		pedido := pedidoGobiernoReglasV3Prueba(t, perfil, "alta_borrador", time.Now().UTC().Truncate(time.Microsecond))
		pedido.Recurso.Ambitos["expediente_ref"] = "expediente:ajeno"
		_, err := broker.ProveerMaterialGobiernoReglasV3(ctx, perfil.soporte.contexto.Vinculo, pedido)
		esperado := app.ErrGobiernoV3Prohibido
		if falloAuditor != nil {
			esperado = app.ErrGobiernoV3NoDisponible
		}
		if !errors.Is(err, esperado) || pdp.llamadas != 0 || len(registrador.ordenes) != 1 {
			t.Fatalf("rechazo sin auditoría previa, o invocó PDP: %v", err)
		}
		o := registrador.ordenes[0]
		if o.Validar() != nil || o.ActorRef != perfil.soporte.contexto.Resultado.Contexto.PersonaRef ||
			o.Superficie != "api.bolsa.reglas_baremo.contexto_validado_pre_pdp.ruta_exacta" ||
			o.Ruta != bolsahttp.RutaAltaGobiernoReglasBaremoV3 || o.Motivo != vp.MotivoAuditoriaFronteraRutaExactaAccesoDenegado {
			t.Fatal("observación distinta del contrato postcontexto CT173")
		}
	}
}

func TestGobiernoBaremoHTTPAuditorPostContextoRechazaActorOFronteraAjenos(t *testing.T) {
	sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, bolsahttp.RutaAltaGobiernoReglasBaremoV3)
	capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	holder := capacidad.contextoOperacion
	holder.soporte = sesion.soporte
	holder.contexto.Vinculo, holder.contexto.Resultado = sesion.soporte.contexto.Vinculo, sesion.soporte.contexto.Resultado
	resultado, err := holder.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	registrador := &auditorConsultaReciboPrueba{}
	callback := auditorDenegacionAntesPDPGobiernoReglasBaremoHTTPV3(sesion, registrador)
	frontera, _ := fronteraSeguridadComunDesdeContexto(ctx)
	ajeno, err := nuevoCatalogoFronterasComunDesarrollo([]descriptorFronteraComunDesarrollo{frontera.descriptor})
	if err != nil {
		t.Fatal(err)
	}
	for _, mutar := range []func(*fronteraSeguridadComunDesarrollo){
		func(f *fronteraSeguridadComunDesarrollo) { f.catalogo = ajeno },
		func(f *fronteraSeguridadComunDesarrollo) { f.ruta += "/ajena" },
	} {
		f := frontera
		mutar(&f)
		pedido := context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, f)
		if err := callback(pedido, app.ErrGobiernoV3Prohibido, &resultado); !errors.Is(err, app.ErrGobiernoV3NoDisponible) {
			t.Fatal("registró frontera ajena")
		}
	}
	for _, actor := range []vd.ResultadoContextoActorRegistradoV2{{}, resultado} {
		actor.Contexto.PersonaRef = "per_aaaaaaaaaaaaaaaaaaaaaaaa"
		if err := callback(ctx, app.ErrGobiernoV3Prohibido, &actor); !errors.Is(err, app.ErrGobiernoV3NoDisponible) {
			t.Fatal("registró actor declarado")
		}
	}
	if len(registrador.ordenes) != 0 {
		t.Fatal("observación postcontexto fabricada")
	}
}
