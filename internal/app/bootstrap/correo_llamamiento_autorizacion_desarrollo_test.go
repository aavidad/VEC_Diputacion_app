package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	ctauditoria "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/auditoria"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridad "vec-diputacion-granada/internal/vec/adapters/documentos/seguridad"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Solo registro y publicación son dobles privados. El PDP, confirmación,
// COSE/Ed25519, confianza, HMAC y ambos autorizadores son los reales.
// No abre PostgreSQL, SMTP ni demuestra consumo transaccional.
func escenarioCorreoLlamamientoV3Prueba(t *testing.T) (*autorizadorCorreoLlamamientoDesarrollo, context.Context, ports.SolicitudDespacharCorreoLlamamiento) {
	t.Helper()
	s, consultas, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	v, err := s.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	s.instantaneaCorreo, err = nuevaInstantaneaCorreoLlamamientoDesarrollo(v.PrincipalID, v.PerfilActivoRef, s.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	s.motivoDespachoCorreo, s.motivoResultadoCorreo = motivoDespachoCorreoLlamamientoDesarrollo(), motivoResultadoCorreoLlamamientoDesarrollo()
	s.motivoComunicacion = motivoLlamamientoDesarrollo(true)
	s.instantaneaComunicacion, err = nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(v.PrincipalID, v.PerfilActivoRef, s.reloj.Ahora(),
		"comunicacion_prueba", "Comunicación de prueba", "comunicacion-prueba", []dominiovec.ConcesionRol{{
			Accion: postgresct.AccionRegistroComunicacionLlamamiento, ModuloID: ports.ModuloContratacion, TipoRecurso: postgresct.TipoRecursoRegistroComunicacionLlamamiento,
			Finalidades: []string{"gestionar_contratacion_temporal"}, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}}, []dominiovec.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionAltaContratacionTemporalDesarrollo}}})
	if err != nil {
		t.Fatal(err)
	}
	alta := &dependenciasAltaContratacionTemporalDesarrollo{soporte: s, autorizador: consultas.autorizador.(autorizadorLigadoContratacionTemporalDesarrollo)}
	proveedor := func(audiencia string, generacion uint32) *proveedorMaterialAltaContratacionTemporalDesarrollo {
		m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(nuevoDerivadorIdempotenciaPrueba(t, generacion, 1), s.reloj.Ahora())
		if err != nil {
			t.Fatal(err)
		}
		defer m.borrarCopiasEfimeras()
		m.capacidad, err = confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(m.claveHMACID, m.claveHMACVersion, m.claveHMAC, m.emisorID, audiencia,
			confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, m.validaDesde, m.validaHasta, time.Time{}, m.claveHMACRevision, m.claveHMACHuella)
		if err != nil {
			t.Fatal(err)
		}
		p, err := nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(m, s, relojContratacionTemporalDesarrollo{})
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	alta.postgresql.proveedorMaterialDespachoCorreo = proveedor(ctapplication.AudienciaDespachoCorreoLlamamientoV3, 2)
	alta.postgresql.proveedorMaterialResultadoCorreo = proveedor(ctapplication.AudienciaResultadoCorreoLlamamientoV3, 3)
	a, _, err := nuevosAutorizadoresCorreoLlamamientoDesarrollo(alta, s.reloj)
	if err != nil {
		t.Fatal(err)
	}
	p := preparacionLlamamientoDesarrollo{expediente: expedientePuenteBolsaPrueba(t)}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, httpinterno.RutaRegistroComunicacionLlamamiento)
	ctx = context.WithValue(ctx, clavePreparacionLlamamientoDesarrollo{}, p)
	solicitud := ports.SolicitudDespacharCorreoLlamamiento{OrganizacionRef: p.expediente.Fiscalizado.OrganizacionRef, ExpedienteRef: p.expediente.Fiscalizado.Referencia,
		LlamamientoRef: "llamamiento:correo-prueba", ComunicacionRef: "comunicacion:correo-prueba", IntencionEnvioRef: "intencion:correo-prueba"}
	return a.(*autorizadorCorreoLlamamientoDesarrollo), ctx, solicitud
}

func resultadoCorreoV3Prueba(t *testing.T, a *autorizadorCorreoLlamamientoDesarrollo, ctx context.Context, s ports.SolicitudDespacharCorreoLlamamiento, c ports.CapacidadDespachoCorreoLlamamiento) (ports.SolicitudRegistrarResultadoCorreoLlamamiento, ports.AuditoriaResultadoCorreoLlamamiento) {
	t.Helper()
	h, err := s.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	r := ports.SolicitudRegistrarResultadoCorreoLlamamiento{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, LlamamientoRef: s.LlamamientoRef, ComunicacionRef: s.ComunicacionRef, IntencionEnvioRef: s.IntencionEnvioRef,
		IntentoRef: "intento:correo-prueba", SolicitudHuella: h, Estado: ports.CorreoLlamamientoAceptadoPorRelay, PlantillaRef: ports.PlantillaCorreoLlamamientoV1, VersionEsperada: 1}
	seudo, err := seguridad.NuevoSelladorHMAC("prueba", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	preparador, err := ctauditoria.NuevoPreparadorResultadoCorreoLlamamiento(seudo, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	audit, err := preparador.PrepararAuditoriaResultadoCorreoLlamamiento(ctx, r, c, a.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	return r, audit
}

func TestCorreoLlamamientoV3DespachoYResultadoRealesConservanAudit17(t *testing.T) {
	a, ctx, s := escenarioCorreoLlamamientoV3Prueba(t)
	despacho, err := a.AutorizarDespachoCorreoLlamamiento(ctx, s)
	if err != nil {
		t.Fatalf("despacho V3 real: %v", err)
	}
	if err = ctapplication.ValidarCapacidadDespachoCorreoLlamamiento(despacho, s, a.reloj.Ahora()); err != nil {
		t.Fatal(err)
	}
	r, audit := resultadoCorreoV3Prueba(t, a, ctx, s, despacho)
	resultado, err := a.AutorizarResultadoCorreoLlamamiento(ctx, r, audit)
	if err != nil {
		t.Fatalf("resultado V3 real: %v", err)
	}
	md, mr := despacho.ExportarMaterialParaConsumidor(), resultado.ExportarMaterialParaConsumidor()
	pd, err := dominiovec.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(md.PayloadVECAD3())
	if err != nil {
		t.Fatal(err)
	}
	pr, err := dominiovec.ParsearMensajeAtestacionAutorizacionV3NoAutoritativo(mr.PayloadVECAD3())
	if err != nil {
		t.Fatal(err)
	}
	rd, _ := pd.VersionRolRef()
	rr, _ := pr.VersionRolRef()
	cd, _ := pd.CorrelacionRef()
	cr, _ := pr.CorrelacionRef()
	nominal, err := audit.CorrelacionPara(r)
	if err != nil {
		t.Fatal(err)
	}
	ca, _ := nominal.ValorCanonico()
	canon, err := audit.SerializarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	var entrada map[string]json.RawMessage
	if json.Unmarshal(canon, &entrada) != nil {
		t.Fatal("Audit17 no canónico")
	}
	var roles []string
	if json.Unmarshal(entrada["actor_roles"], &roles) != nil {
		t.Fatal("roles audit")
	}
	if len(entrada) != 17 || len(roles) != 1 || roles[0] != rr || rd != rr || cr != ca || cd == cr {
		t.Fatal("correlación o rol no corresponden a Audit17 y decisión final")
	}
	if md.ResumenCapacidad().DecisionRef() == mr.ResumenCapacidad().DecisionRef() {
		t.Fatal("resultado reutilizó decisión de despacho")
	}
	if md.ResumenCapacidad().AudienciaConsumo() == mr.ResumenCapacidad().AudienciaConsumo() {
		t.Fatal("audiencias mezcladas")
	}
	ssoporte := a.despacho.alta.soporte
	if len(ssoporte.instantaneaCorreo.VersionRol.Concesiones) != 2 || len(ssoporte.instantaneaComunicacion.VersionRol.Concesiones) != 1 {
		t.Fatal("rol heredado ampliado")
	}
	registro := ssoporte.registroDecisionesAnalisis.(*registroDecisionesAnalisisContratacionTemporalDesarrolloPrueba)
	if registro.concesiones != 2 {
		t.Fatalf("concesiones=%d", registro.concesiones)
	}
	for _, i := range ssoporte.instantaneasPorSolicitud {
		if len(i.AsignacionPerfil.Ambitos) != 1 || i.AsignacionPerfil.Ambitos[0].Valores[0] != s.OrganizacionRef {
			t.Fatal("ámbito fijo alterado")
		}
	}
	if _, err = ctapplication.NuevaCapacidadDespachoCorreoLlamamiento(s, mr, a.reloj.Ahora()); err == nil {
		t.Fatal("audiencia resultado admitida como despacho")
	}
	if _, err = ctapplication.NuevaCapacidadResultadoCorreoLlamamiento(r, audit, md, a.reloj.Ahora()); err == nil {
		t.Fatal("audiencia despacho admitida como resultado")
	}
	if ctapplication.ValidarCapacidadDespachoCorreoLlamamiento(despacho, s, md.ResumenCapacidad().ExpiraEn()) == nil {
		t.Fatal("despacho vencido aceptado")
	}
	if ctapplication.ValidarCapacidadResultadoCorreoLlamamiento(resultado, r, audit, mr.ResumenCapacidad().ExpiraEn()) == nil {
		t.Fatal("resultado vencido aceptado")
	}
}

func TestCorreoLlamamientoV3PreservaComunicacionCT54(t *testing.T) {
	a, ctx, s := escenarioCorreoLlamamientoV3Prueba(t)
	original := &autorizadorLlamamientoDesarrollo{alta: a.despacho.alta, material: a.despacho.material, comunicacion: true}
	recurso := dominiovec.RecursoAutorizable{Referencia: s.ExpedienteRef, ModuloID: ports.ModuloContratacion, Tipo: postgresct.TipoRecursoRegistroComunicacionLlamamiento, Ambitos: map[string]string{"organizacion_ref": s.OrganizacionRef}, Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)}}
	solicitud, decision, confirmacion, err := original.exigirOperacion(ctx, postgresct.AccionRegistroComunicacionLlamamiento, recurso)
	if err != nil {
		t.Fatalf("CT54 sin claves correo denegado: %v", err)
	}
	if decision.ValidarPara(solicitud) != nil {
		t.Fatal("CT54 sin decisión válida")
	}
	_ = confirmacion
	if _, err = a.despacho.AutorizarOperacion(ctx, postgresct.AccionRegistroComunicacionLlamamiento, recurso); err == nil {
		t.Fatal("rol correo autorizó CT54")
	}
}

func TestCorreoLlamamientoV3DeniegaCruces(t *testing.T) {
	for nombre, mutar := range map[string]func(*autorizadorCorreoLlamamientoDesarrollo, context.Context, *ports.SolicitudDespacharCorreoLlamamiento) context.Context{
		"organizacion": func(_ *autorizadorCorreoLlamamientoDesarrollo, c context.Context, s *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			s.OrganizacionRef = "organizacion:ajena"
			return c
		},
		"expediente": func(_ *autorizadorCorreoLlamamientoDesarrollo, c context.Context, s *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			s.ExpedienteRef = "expediente:ajeno"
			return c
		},
		"preparacion ausente": func(_ *autorizadorCorreoLlamamientoDesarrollo, c context.Context, _ *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			return context.WithValue(c, clavePreparacionLlamamientoDesarrollo{}, preparacionLlamamientoDesarrollo{})
		},
		"actor": func(_ *autorizadorCorreoLlamamientoDesarrollo, c context.Context, _ *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			v := c.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
			v.principal.ID = "actor:ajeno"
			return context.WithValue(c, claveCapacidadConsultasContratacionTemporalDesarrollo{}, v)
		},
		"modo": func(a *autorizadorCorreoLlamamientoDesarrollo, c context.Context, _ *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			a.despacho.comunicacion = true
			return c
		},
		"audiencia": func(a *autorizadorCorreoLlamamientoDesarrollo, c context.Context, _ *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			a.despacho.material = a.resultado.material
			return c
		},
		"rol CT54": func(a *autorizadorCorreoLlamamientoDesarrollo, c context.Context, _ *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			a.despacho.alta.soporte.instantaneaCorreo = a.despacho.alta.soporte.instantaneaComunicacion
			return c
		},
		"ambito publicado": func(a *autorizadorCorreoLlamamientoDesarrollo, c context.Context, _ *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			a.despacho.alta.soporte.instantaneaCorreo.AsignacionPerfil.Ambitos[0].Valores = []string{"organizacion:ajena"}
			return c
		},
		"asignacion vencida": func(a *autorizadorCorreoLlamamientoDesarrollo, c context.Context, _ *ports.SolicitudDespacharCorreoLlamamiento) context.Context {
			a.despacho.alta.soporte.instantaneaCorreo.AsignacionPerfil.VigenteHasta = a.reloj.Ahora().Add(-time.Second)
			return c
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			a, ctx, s := escenarioCorreoLlamamientoV3Prueba(t)
			ctx = mutar(a, ctx, &s)
			if _, err := a.AutorizarDespachoCorreoLlamamiento(ctx, s); err == nil {
				t.Fatal("cruce admitido")
			}
		})
	}
}

func TestCorreoLlamamientoV3ResultadoNoCruzaSolicitudNiRecurso(t *testing.T) {
	a, ctx, s := escenarioCorreoLlamamientoV3Prueba(t)
	c, err := a.AutorizarDespachoCorreoLlamamiento(ctx, s)
	if err != nil {
		t.Fatal(err)
	}
	r, audit := resultadoCorreoV3Prueba(t, a, ctx, s, c)
	for nombre, mutar := range map[string]func(*ports.SolicitudRegistrarResultadoCorreoLlamamiento){
		"organizacion": func(r *ports.SolicitudRegistrarResultadoCorreoLlamamiento) { r.OrganizacionRef = "organizacion:ajena" },
		"expediente":   func(r *ports.SolicitudRegistrarResultadoCorreoLlamamiento) { r.ExpedienteRef = "expediente:ajeno" },
		"intento":      func(r *ports.SolicitudRegistrarResultadoCorreoLlamamiento) { r.IntentoRef = "intento:ajeno" },
	} {
		t.Run(nombre, func(t *testing.T) {
			otra := r
			mutar(&otra)
			if _, err := a.AutorizarResultadoCorreoLlamamiento(ctx, otra, audit); err == nil {
				t.Fatal("Audit17 admitido para otra solicitud")
			}
		})
	}
	recurso, _ := ctapplication.RecursoDespachoCorreoLlamamiento(s)
	ctx = context.WithValue(ctx, claveSolicitudDespachoCorreoLlamamientoDesarrollo{}, s)
	recurso.Referencia = "intencion:ajena"
	if _, err := a.despacho.AutorizarOperacion(ctx, ctapplication.AccionDespacharCorreoLlamamiento, recurso); err == nil {
		t.Fatal("recurso ajeno admitido")
	}
}

func TestCorreoLlamamientoV3ResultadoValidaPreparacionYFronteras(t *testing.T) {
	for _, campo := range []string{"organizacion", "expediente"} {
		t.Run(campo+" con auditoria propia", func(t *testing.T) {
			a, ctx, s := escenarioCorreoLlamamientoV3Prueba(t)
			c, err := a.AutorizarDespachoCorreoLlamamiento(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			if campo == "organizacion" {
				s.OrganizacionRef = "organizacion:ajena"
			} else {
				s.ExpedienteRef = "expediente:ajeno"
			}
			r, audit := resultadoCorreoV3Prueba(t, a, ctx, s, c)
			// La auditoría corresponde exactamente a la solicitud cruzada. El rechazo
			// debe proceder del expediente preparado, no de un hash inconsistente.
			if _, err = audit.CorrelacionPara(r); err != nil {
				t.Fatal(err)
			}
			if _, err = a.AutorizarResultadoCorreoLlamamiento(ctx, r, audit); err == nil {
				t.Fatal("resultado fuera de preparación confiable")
			}
		})
	}
	for _, frontera := range []string{"actor", "modo", "audiencia"} {
		t.Run(frontera, func(t *testing.T) {
			a, ctx, s := escenarioCorreoLlamamientoV3Prueba(t)
			c, err := a.AutorizarDespachoCorreoLlamamiento(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			r, audit := resultadoCorreoV3Prueba(t, a, ctx, s, c)
			switch frontera {
			case "actor":
				v := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
				v.principal.ID = "actor:ajeno"
				ctx = context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, v)
			case "modo":
				a.resultado.despachoCorreo = true
			case "audiencia":
				a.resultado.material = a.despacho.material
			}
			if _, err = a.AutorizarResultadoCorreoLlamamiento(ctx, r, audit); err == nil {
				t.Fatal("frontera de resultado omitida")
			}
		})
	}
}

func TestCorreoLlamamientoAutorizacionErrorConservaExitoYCancelacion(t *testing.T) {
	if errorAutorizacionCorreoLlamamientoDesarrollo(context.Background(), nil) != nil {
		t.Fatal("éxito convertido en denegación")
	}
	if errorAutorizacionCorreoLlamamientoDesarrollo(context.Background(), errors.New("privado")) != ports.ErrDespachoCorreoLlamamientoDenegado {
		t.Fatal("error no redactado")
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if !errors.Is(errorAutorizacionCorreoLlamamientoDesarrollo(ctx, nil), context.Canceled) {
		t.Fatal("cancelación ignorada")
	}
	if _, _, err := nuevosAutorizadoresCorreoLlamamientoDesarrollo(nil, relojContratacionTemporalDesarrollo{}); err == nil {
		t.Fatal("constructor sin autoridad")
	}
}
