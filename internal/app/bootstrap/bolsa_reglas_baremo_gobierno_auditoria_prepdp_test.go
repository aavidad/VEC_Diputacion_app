package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
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
	correlacion, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: pedido.Motivo,
		Accion: pedido.Accion, Recurso: pedido.Recurso, Finalidad: pedido.Finalidad, Correlacion: correlacion,
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
		broker := &ProveedorGobiernoReglasBaremoV3{pdp: pdp, auditarAntesPDP: func(context.Context, error, *contextoSeguridadComunDesarrollo) error { auditorias++; return nil }}
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
		registrador := &registradorIntentosBaremoPrueba{err: falloAuditor}
		broker := &ProveedorGobiernoReglasBaremoV3{perfil: perfil, sesion: sesion, pdp: pdp, reloj: relojContratacionTemporalDesarrollo{},
			rutas:           RutasGobiernoReglasBaremoV3{bolsahttp.RutaAltaGobiernoReglasBaremoV3, bolsahttp.RutaConsultaGobiernoReglasBaremoV3, bolsahttp.RutaRecuperarGobiernoReglasBaremoV3},
			auditarAntesPDP: auditorDenegacionAntesPDPGobiernoReglasBaremoHTTPV3(sesion, &auditorConsultaReciboPrueba{}, auditorIntentosBaremoPrueba(sesion, registrador))}
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
		o, errOrden := registrador.ordenes[0].Datos()
		if errOrden != nil || o.ResultadoContexto.Contexto.PersonaRef != perfil.soporte.contexto.Resultado.Contexto.PersonaRef || o.Datos.Accion != "bolsa.reglas_baremo.borrador.crear" || o.Datos.Resultado != vd.ResultadoIntentoAuditoriaDenegado || o.Datos.RecursoRef != pedido.Recurso.Referencia {
			t.Fatal("intento distinto de la identidad nominal común")
		}
		correlacion, errCanon := ctx.Value(claveIntentoGobiernoBaremoHTTPV3{}).(*intentoGobiernoBaremoHTTPV3).correlacion.ValorCanonico()
		if errCanon != nil || o.Datos.CorrelacionRef != correlacion {
			t.Fatal("se perdió correlación del intento")
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
	registrador := &registradorIntentosBaremoPrueba{}
	callback := auditorDenegacionAntesPDPGobiernoReglasBaremoHTTPV3(sesion, &auditorConsultaReciboPrueba{}, auditorIntentosBaremoPrueba(sesion, registrador))
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
		if err := callback(pedido, app.ErrGobiernoV3Prohibido, &contextoSeguridadComunDesarrollo{Resultado: resultado, Vinculo: holder.contexto.Vinculo}); !errors.Is(err, app.ErrGobiernoV3NoDisponible) {
			t.Fatal("registró frontera ajena")
		}
	}
	for _, actor := range []vd.ResultadoContextoActorRegistradoV2{{}, resultado} {
		actor.Contexto.PersonaRef = "per_aaaaaaaaaaaaaaaaaaaaaaaa"
		if err := callback(ctx, app.ErrGobiernoV3Prohibido, &contextoSeguridadComunDesarrollo{Resultado: actor, Vinculo: holder.contexto.Vinculo}); !errors.Is(err, app.ErrGobiernoV3NoDisponible) {
			t.Fatal("registró actor declarado")
		}
	}
	if len(registrador.ordenes) != 0 {
		t.Fatal("observación postcontexto fabricada")
	}
}

type registradorIntentosBaremoPrueba struct {
	ordenes []vp.OrdenIntentoAuditoria
	err     error
}

func (r *registradorIntentosBaremoPrueba) AppendIntentoAuditoria(ctx context.Context, o vp.OrdenIntentoAuditoria) (vp.AcuseIntentoAuditoria, error) {
	if err := ctx.Err(); err != nil {
		return vp.AcuseIntentoAuditoria{}, err
	}
	r.ordenes = append(r.ordenes, o)
	if r.err != nil {
		return vp.AcuseIntentoAuditoria{}, r.err
	}
	d, err := o.Datos()
	if err != nil {
		return vp.AcuseIntentoAuditoria{}, err
	}
	return vp.AcuseIntentoAuditoria{AuditoriaRef: "aud_prueba", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}, nil
}
func auditorIntentosBaremoPrueba(s *proveedorSesionConsultaRRHHDesarrollo, r vp.RegistradorIntentosAuditoria) *auditorGobiernoBaremoHTTPV3 {
	return &auditorGobiernoBaremoHTTPV3{sesion: s, registrador: r, proceso: "vec-server", recursoRef: "expediente:prueba", motivoDenegado: motivoCatalogoPlantillasCTDesarrollo(), motivoError: motivoCatalogoPlantillasCTDesarrollo()}
}

func TestGobiernoBaremoHTTPAuditaErrorTrasRetornoSinDuplicarDenegacion(t *testing.T) {
	for _, par := range paresGobiernoReglasBaremoHTTPV3() {
		sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, par.ruta)
		capacidad := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		holder := capacidad.contextoOperacion
		holder.soporte = sesion.soporte
		holder.contexto.Vinculo, holder.contexto.Resultado = sesion.soporte.contexto.Vinculo, sesion.soporte.contexto.Resultado
		p := &ProveedorGobiernoReglasBaremoV3{perfil: &PerfilGobiernoReglasBaremoV3{soporte: sesion.soporte}, sesion: sesion, pdp: &pdpNoInvocadoGobiernoBaremoPrueba{}, reloj: relojContratacionTemporalDesarrollo{}, rutas: RutasGobiernoReglasBaremoV3{bolsahttp.RutaAltaGobiernoReglasBaremoV3, bolsahttp.RutaConsultaGobiernoReglasBaremoV3, bolsahttp.RutaRecuperarGobiernoReglasBaremoV3}}
		p.perfil.plantilla.AsignacionPerfil.PerfilActivoRef = sesion.base.Contexto.PerfilActivoRef
		for _, caso := range []struct {
			fallo, errorAuditor error
			cantidad            int
			esperado            error
		}{
			{errors.New("repositorio no disponible"), nil, 1, nil},
			{app.ErrGobiernoV3Prohibido, nil, 1, app.ErrGobiernoV3Prohibido},
			{errorAuditadoGobiernoBaremoHTTPV3{app.ErrGobiernoV3Prohibido, true}, nil, 0, app.ErrGobiernoV3Prohibido},
			{errors.New("repositorio no disponible"), errors.New("registrador no disponible"), 1, app.ErrGobiernoV3NoDisponible},
			{nil, nil, 0, nil},
		} {
			r := &registradorIntentosBaremoPrueba{err: caso.errorAuditor}
			o := &operadorAuditadoGobiernoBaremoHTTPV3{proveedor: p, operador: &operadorNoInvocadoBaremoHTTPPrueba{err: caso.fallo}, auditor: auditorIntentosBaremoPrueba(sesion, r)}
			var err error
			switch par.ruta {
			case bolsahttp.RutaAltaGobiernoReglasBaremoV3:
				_, err = o.GuardarAltaBorrador(ctx, app.CredencialesGobiernoV3{}, app.PeticionAltaBorradorV3{})
			case bolsahttp.RutaConsultaGobiernoReglasBaremoV3:
				_, err = o.ConsultarExacta(ctx, app.CredencialesGobiernoV3{}, app.PeticionConsultaExactaV3{})
			default:
				_, err = o.RecuperarRecibo(ctx, app.CredencialesGobiernoV3{}, app.PeticionRecuperarReciboV3{})
			}
			if len(r.ordenes) != caso.cantidad || (caso.esperado != nil && !errors.Is(err, caso.esperado)) || (caso.fallo == nil && err != nil) {
				t.Fatalf("error sin registro exacto tras retorno: %v registros=%d", err, len(r.ordenes))
			}
			if len(r.ordenes) > 0 {
				d, e := r.ordenes[0].Datos()
				if e != nil || d.Datos.Accion != par.accion || d.ResultadoContexto.Contexto.PerfilActivoRef != sesion.base.Contexto.PerfilActivoRef {
					t.Fatal("registro pierde acción o perfil nominal")
				}
			}
		}
	}
}

func TestGobiernoBaremoHTTPConservaIdentidadDelIntentoCancelado(t *testing.T) {
	sesion, ctx := contextoSesionGobiernoBaremoHTTPPrueba(t, bolsahttp.RutaAltaGobiernoReglasBaremoV3)
	c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	c.contextoOperacion.soporte = sesion.soporte
	c.contextoOperacion.contexto.Vinculo, c.contextoOperacion.contexto.Resultado = sesion.soporte.contexto.Vinculo, sesion.soporte.contexto.Resultado
	r := &registradorIntentosBaremoPrueba{}
	a := auditorIntentosBaremoPrueba(sesion, r)
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	historico, err := a.contextoHistorico(cancelado)
	if err != nil || a.registrar(cancelado, historico, context.Canceled) != nil || len(r.ordenes) != 1 {
		t.Fatal("cancelación perdió la identidad acreditada")
	}
	d, err := r.ordenes[0].Datos()
	if err != nil || d.Datos.Resultado != vd.ResultadoIntentoAuditoriaError || d.ResultadoContexto.RegistroContextoRef != sesion.soporte.contexto.Resultado.RegistroContextoRef {
		t.Fatal("el fallo fabricó otro contexto")
	}
}
