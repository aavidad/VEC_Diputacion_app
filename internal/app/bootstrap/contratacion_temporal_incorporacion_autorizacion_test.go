package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	lectura "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Sólo dobles de frontera para comprobar delegación y ausencia de efectos en
// negativas. No se conectan al runtime ni acreditan publicación PostgreSQL.
type planesPermisoIncorporacionPrueba struct{ plan inc.PlanPreparacionDurableV2 }

func (p planesPermisoIncorporacionPrueba) ResolverPlan(_ context.Context, org, exp string) (inc.PlanPreparacionDurableV2, error) {
	if org != p.plan.OrganizacionRef || exp != p.plan.SolicitudPersonal.ExpedienteRef {
		return inc.PlanPreparacionDurableV2{}, ct.ErrComposicionIncorporacionAplicacion
	}
	return p.plan, nil
}

type publicadorPermisoIncorporacionPrueba struct {
	orden  []string
	ultima core.InstantaneaAutorizacion
	falla  string
}

func (p *publicadorPermisoIncorporacionPrueba) PrepararInstantanea(_ context.Context, i core.InstantaneaAutorizacion) (core.InstantaneaAutorizacion, error) {
	p.orden = append(p.orden, "preparar")
	if p.falla == "preparar" {
		return core.InstantaneaAutorizacion{}, ct.ErrAutorizacionDenegada
	}
	i.AsignacionPerfil.Version = 2
	return i, nil
}
func (p *publicadorPermisoIncorporacionPrueba) PublicarInstantanea(_ context.Context, i core.InstantaneaAutorizacion) error {
	p.orden = append(p.orden, "publicar")
	p.ultima = i
	if p.falla == "publicar" {
		return ct.ErrAutorizacionDenegada
	}
	return nil
}

type pdpPermisoIncorporacionPrueba struct {
	publicador *publicadorPermisoIncorporacionPrueba
	solicitud  core.SolicitudAutorizacionLigadaV3
	resultado  core.ResultadoContextoActorRegistradoV2
}

var errPDPIncorporacionPrueba = errors.New("respuesta del PDP nominal de prueba")

func (p *pdpPermisoIncorporacionPrueba) ExigirSolicitudLigadaV3(_ context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	p.publicador.orden = append(p.publicador.orden, "pdp")
	p.solicitud, p.resultado = s, r
	return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, errPDPIncorporacionPrueba
}

func escenarioPermisoIncorporacionPrueba(t *testing.T) (*autoridadOperacionesIncorporacionV2, context.Context, core.DatosSolicitudAutorizacionLigadaV3, *publicadorPermisoIncorporacionPrueba) {
	t.Helper()
	alta, consultas, principal := escenarioConsultasRRHHDesarrolloPrueba(t)
	s := alta.soporte
	v, _ := s.contexto.Vinculo.Datos()
	refs := ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: v.PrincipalID, PerfilV3Ref: v.PerfilActivoRef, OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo, UnidadRef: "unidad:desarrollo:rrhh", ActorRef: v.PrincipalID}
	datos := datosSolicitudConsultasRRHHDesarrolloPrueba(t, s, httpinterno.RutaConsultaDetalleRRHH)
	datos.Recurso.Referencia = "expediente:ejercicio:ct:0001"
	contenido, err := os.ReadFile("../../modules/personal/adapters/fuenteejercicio/testdata/fuente-ejercicio.json")
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(contenido)
	terna := fuenteejercicio.TernaEsperada{Referencia: "fuente:personal:ejercicio:20260908", Version: 1, HuellaSHA256: hex.EncodeToString(h[:])}
	solicitud := ct.SolicitudAltaPersonalRPT{Esquema: ct.EsquemaAltaPersonalRPT, ContratoVersion: ct.VersionContratoAltaPersonalRPT, SolicitudRef: "solicitud:ejercicio:0001", ExpedienteRef: datos.Recurso.Referencia, VersionExpediente: 7, CapacidadRef: "capacidad:ejercicio:0001", CorrelacionRef: "correlacion:ejercicio:0001", IdempotenciaRef: "idempotencia:ejercicio:0001", FuenteRPT: ct.ReferenciaVersionadaPersonalRPT{Referencia: "rpt:ejercicio:20260908", Version: 3, HuellaSHA256: "a81229be023f01b99c16b76c9d4f6283dc6efaf5177340b063d36ab30dc654ac"}, PuestoRef: "puesto:ejercicio:0001", PlazaRef: "plaza:ejercicio:0001"}
	plan := inc.PlanPreparacionDurableV2{OrganizacionRef: refs.OrganizacionRef, UnidadRef: refs.UnidadRef, SolicitudPersonal: solicitud, FuentePersonal: terna, MotivoV3: datos.ReferenciaMotivo}
	cfg := archivoIncorporacionV2{Referencias: refs, TernaPersonal: terna, MotivoAlta: datos.ReferenciaMotivo, MotivoLectura: datos.ReferenciaMotivo}
	publicador := &publicadorPermisoIncorporacionPrueba{}
	s.autoridadAsignaciones = publicador
	delegado := &pdpPermisoIncorporacionPrueba{publicador: publicador}
	a, err := nuevaAutoridadOperacionesIncorporacionV2(s, consultas, delegado, cfg, planesPermisoIncorporacionPrueba{plan}, contenido, datos.ReferenciaMotivo, s.reloj)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextoRutaConsultasRRHHDesarrolloPrueba(s, principal, httpinterno.RutaConsultaDetalleRRHH)
	return a, context.WithValue(ctx, claveIncorporacionV2Desarrollo{}, s.sello), datos, publicador
}

func datosOperacionPermisoIncorporacionPrueba(t *testing.T, a *autoridadOperacionesIncorporacionV2, d core.DatosSolicitudAutorizacionLigadaV3, operacion string) core.DatosSolicitudAutorizacionLigadaV3 {
	t.Helper()
	p, err := a.planes.ResolverPlan(context.Background(), a.referencias.OrganizacionRef, d.Recurso.Referencia)
	if err != nil {
		t.Fatal(err)
	}
	switch operacion {
	case "detalle":
	case "alta":
		fuente, err := fuenteejercicio.NuevaFuenteEjercicio(a.personal, a.ternaPersonal)
		if err != nil {
			t.Fatal(err)
		}
		v, err := fuente.Resolver(context.Background(), p.SolicitudPersonal)
		if err != nil {
			t.Fatal(err)
		}
		d.Recurso, err = personal.RecursoAltaEjercicio(personal.MaterialAlta{Preparacion: personal.PreparacionAlta{Solicitud: p.SolicitudPersonal, Fuente: p.FuentePersonal, Vinculo: v}, OrganizacionRef: a.referencias.OrganizacionRef, ActorRef: a.referencias.PrincipalV3Ref, PerfilRef: a.referencias.PerfilV3Ref})
		if err != nil {
			t.Fatal(err)
		}
		d.Accion, d.Finalidad = personal.AccionAltaEjercicio, personal.FinalidadAltaEjercicio
	case "lectura":
		d.Accion, d.Finalidad = lectura.Accion, lectura.Finalidad
		d.Recurso = core.RecursoAutorizable{Referencia: "resultado:ejercicio:0001", ModuloID: "personal", Tipo: lectura.TipoRecursoV2, Ambitos: map[string]string{"organizacion_ref": a.referencias.OrganizacionRef, "unidad_ref": a.referencias.UnidadRef}, Atributos: map[string]string{"expediente_ref": p.SolicitudPersonal.ExpedienteRef, "solicitud_ref": p.SolicitudPersonal.SolicitudRef, "version_expediente": "7", "resultado_ref": "resultado:ejercicio:0001", "recibo_ref": "recibo:ejercicio:0001", "relacion_ref": "relacion:ejercicio:0001", "ocupacion_ref": "ocupacion:ejercicio:0001", "material_sha256": strings.Repeat("a", 64), "tipo_validacion": "ejercicio_sintetico"}}
	case "ct":
		d.Accion, d.Finalidad = ct.AccionConfirmarIncorporacion, ct.FinalidadConfirmarIncorporacion
		cor, _ := d.Correlacion.ValorCanonico()
		d.Recurso = core.RecursoAutorizable{Referencia: p.SolicitudPersonal.ExpedienteRef, ModuloID: ct.ModuloContratacion, Tipo: ct.TipoRecursoConfirmacionIncorporacionV2, Ambitos: map[string]string{"organizacion_ref": a.referencias.OrganizacionRef, "unidad_ref": a.referencias.UnidadRef}, Atributos: map[string]string{"principal_v3_ref": a.referencias.PrincipalV3Ref, "perfil_v3_ref": a.referencias.PerfilV3Ref, "actor_seguimiento_ref": a.referencias.ActorRef, "correlacion_v3_ref": cor, "motivo_v3_ref": p.MotivoV3.EntradaClave, "correlacion_seguimiento_ref": "correlacion:ejercicio:0001", "material_sha256": strings.Repeat("a", 64), "tipo_validacion": "ejercicio_sintetico"}}
	}
	return d
}

func TestIncorporacionV2PermisosNominalesPublicanAntesDelPDP(t *testing.T) {
	for _, operacion := range []string{"detalle", "alta", "lectura", "ct"} {
		t.Run(operacion, func(t *testing.T) {
			a, ctx, d, p := escenarioPermisoIncorporacionPrueba(t)
			d = datosOperacionPermisoIncorporacionPrueba(t, a, d, operacion)
			solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
			if err != nil {
				t.Fatal(err)
			}
			resultado := a.soporte.contexto.Resultado
			_, _, err = a.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
			if !errors.Is(err, errPDPIncorporacionPrueba) || !reflect.DeepEqual(p.orden, []string{"preparar", "publicar", "pdp"}) {
				t.Fatalf("delegación fuera de orden: %v %v", p.orden, err)
			}
			cantidad, indice := 1, 0
			if operacion == "lectura" || operacion == "ct" {
				cantidad = 2
			}
			if operacion == "ct" {
				indice = 1
			}
			if p.ultima.Validar() != nil || p.ultima.AsignacionPerfil.Version != 2 || !p.ultima.AsignacionPerfil.Cubre(d.Recurso) || len(p.ultima.VersionRol.Concesiones) != cantidad || p.ultima.VersionRol.Concesiones[indice].Accion != d.Accion {
				t.Fatal("no publicó el único permiso acotado preparado")
			}
			delegado := a.delegado.(*pdpPermisoIncorporacionPrueba)
			recibida, _ := delegado.solicitud.Datos()
			if !reflect.DeepEqual(recibida, d) || !reflect.DeepEqual(delegado.resultado, resultado) {
				t.Fatal("alteró solicitud o contexto del PDP")
			}
		})
	}
}

func TestIncorporacionV2PermisosNominalesRechazanCrucesSinPublicar(t *testing.T) {
	for _, operacion := range []string{"detalle", "alta", "lectura", "ct"} {
		for _, caso := range []string{"sello", "actor", "perfil", "organizacion", "ambito_extra", "accion", "finalidad", "tipo", "motivo", "expediente"} {
			t.Run(operacion+"/"+caso, func(t *testing.T) {
				a, ctx, d, p := escenarioPermisoIncorporacionPrueba(t)
				d = datosOperacionPermisoIncorporacionPrueba(t, a, d, operacion)
				switch caso {
				case "sello":
					ctx = context.WithValue(ctx, claveIncorporacionV2Desarrollo{}, new(int))
				case "actor":
					a.referencias.PrincipalV3Ref = "per_" + strings.Repeat("b", 32)
				case "perfil":
					a.referencias.PerfilV3Ref = "prf_" + strings.Repeat("b", 32)
				case "organizacion":
					d.Recurso.Ambitos["organizacion_ref"] = "organizacion:ajena"
				case "ambito_extra":
					d.Recurso.Ambitos["otro_ref"] = "unidad:ajena"
				case "accion":
					d.Accion = "personal.otra"
				case "finalidad":
					d.Finalidad = "otra_finalidad"
				case "tipo":
					d.Recurso.Tipo = "otro_tipo"
				case "motivo":
					d.ReferenciaMotivo.CatalogoVersion++
				case "expediente":
					if operacion == "alta" || operacion == "lectura" {
						d.Recurso.Atributos["expediente_ref"] = "expediente:ajeno"
					} else {
						d.Recurso.Referencia = "expediente:ajeno"
					}
				}
				solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
				if err != nil {
					t.Fatal(err)
				}
				_, _, err = a.ExigirSolicitudLigadaV3(ctx, solicitud, a.soporte.contexto.Resultado)
				if !errors.Is(err, ct.ErrAutorizacionDenegada) || len(p.orden) != 0 {
					t.Fatalf("cruce alcanzó publicación/PDP: %v %v", p.orden, err)
				}
			})
		}
	}
}

func TestIncorporacionV2PermisosNominalesFalloPublicadorYCancelacion(t *testing.T) {
	for _, caso := range []string{"preparar", "publicar", "cancelada"} {
		t.Run(caso, func(t *testing.T) {
			a, ctx, d, p := escenarioPermisoIncorporacionPrueba(t)
			p.falla = caso
			if caso == "cancelada" {
				var cancelar context.CancelFunc
				ctx, cancelar = context.WithCancel(ctx)
				cancelar()
			}
			solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = a.ExigirSolicitudLigadaV3(ctx, solicitud, a.soporte.contexto.Resultado)
			if err == nil {
				t.Fatal("aceptó fallo")
			}
			for _, paso := range p.orden {
				if paso == "pdp" {
					t.Fatal("delegó tras fallo")
				}
			}
			if caso == "cancelada" && (!errors.Is(err, context.Canceled) || len(p.orden) != 0) {
				t.Fatal("perdió cancelación")
			}
		})
	}
}

func segundaCapturaPermisoIncorporacionPrueba(t *testing.T, a *autoridadOperacionesIncorporacionV2) ct.ContextoAutorizacionAltaV3 {
	t.Helper()
	base, err := a.soporte.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	r, err := a.soporte.contexto.Resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	ahora := a.reloj.Ahora()
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: base.CuentaRef, Metodo: base.MetodoObservado, Garantia: base.GarantiaObservada}
	r.Contexto, err = core.NuevoContextoActor(cuenta, r.Contexto.Instantanea, ahora)
	if err != nil {
		t.Fatal(err)
	}
	r.RepresentacionCanonica, err = r.Contexto.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	r.HuellaSHA256, err = r.Contexto.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	r.ResueltoEnAutoritativo = ahora
	r.RegistroContextoRef = referenciaAltaContratacionTemporalDesarrollo("rca_", "segunda-captura-incorporacion")
	auth := base.Autenticacion()
	auth.SesionRevalidadaEn = ahora
	v, r, err := core.CrearVinculoAutenticacionActorV2ConResultado(context.Background(), revalidadorAutenticacionAltaContratacionTemporalDesarrollo{valor: auth}, core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: base.AutenticacionRef, SesionRef: base.SesionRef}, resolutorContextoAltaContratacionTemporalDesarrollo{valor: r}, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: base.PerfilActivoRef}, a.reloj)
	if err != nil {
		t.Fatal(err)
	}
	if v.CoincideExactamenteCon(a.soporte.contexto.Vinculo) {
		t.Fatal("la prueba no creó una segunda captura")
	}
	return ct.ContextoAutorizacionAltaV3{Vinculo: v, Resultado: r}
}

func TestIncorporacionV2PermisosNominalesSegundaCapturaMismaSesion(t *testing.T) {
	for _, operacion := range []string{"alta", "lectura", "ct"} {
		t.Run(operacion, func(t *testing.T) {
			a, ctx, d, p := escenarioPermisoIncorporacionPrueba(t)
			d = datosOperacionPermisoIncorporacionPrueba(t, a, d, operacion)
			propio := segundaCapturaPermisoIncorporacionPrueba(t, a)
			d.VinculoAutenticacionActor = propio.Vinculo
			solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = a.ExigirSolicitudLigadaV3(ctx, solicitud, propio.Resultado)
			if !errors.Is(err, errPDPIncorporacionPrueba) || !reflect.DeepEqual(p.orden, []string{"preparar", "publicar", "pdp"}) {
				t.Fatalf("rechazó captura legítima: %v %v", p.orden, err)
			}
			if !reflect.DeepEqual(a.delegado.(*pdpPermisoIncorporacionPrueba).resultado, propio.Resultado) {
				t.Fatal("sustituyó recibo de contexto")
			}
		})
	}
	a, ctx, d, p := escenarioPermisoIncorporacionPrueba(t)
	ajeno := contextoFrescoConsultasRRHHDesarrolloPrueba(t, a.soporte, "sesion-distinta-incorporacion")
	d.VinculoAutenticacionActor = ajeno.Vinculo
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = a.ExigirSolicitudLigadaV3(ctx, solicitud, ajeno.Resultado)
	if !errors.Is(err, ct.ErrAutorizacionDenegada) || len(p.orden) != 0 {
		t.Fatal("aceptó otra sesión del mismo actor")
	}
}

func TestIncorporacionV2PermisosNominalesConservanCTDuranteLecturas(t *testing.T) {
	a, ctx, base, p := escenarioPermisoIncorporacionPrueba(t)
	var anterior core.InstantaneaAutorizacion
	for i, operacion := range []string{"lectura", "ct", "lectura", "lectura"} {
		d := datosOperacionPermisoIncorporacionPrueba(t, a, base, operacion)
		solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = a.ExigirSolicitudLigadaV3(ctx, solicitud, a.soporte.contexto.Resultado)
		if !errors.Is(err, errPDPIncorporacionPrueba) {
			t.Fatal(err)
		}
		if i > 0 && !reflect.DeepEqual(anterior, p.ultima) {
			t.Fatal("la lectura cambió la asignación que sostiene el permiso CT pendiente")
		}
		anterior = p.ultima
	}
	permisos := anterior.VersionRol.Concesiones
	if len(permisos) != 2 || permisos[0].Accion != lectura.Accion || permisos[0].TipoRecurso != lectura.TipoRecursoV2 || permisos[1].Accion != ct.AccionConfirmarIncorporacion || permisos[1].TipoRecurso != ct.TipoRecursoConfirmacionIncorporacionV2 {
		t.Fatal("el par no conserva sus dos contratos cerrados")
	}
	// Alta y detalle conservan otras dimensiones y no pueden reutilizar este rol.
	for _, operacion := range []string{"alta", "detalle"} {
		d := datosOperacionPermisoIncorporacionPrueba(t, a, base, operacion)
		if anterior.AsignacionPerfil.Cubre(d.Recurso) {
			t.Fatal("el rol conjunto cubre ámbitos de alta/detalle")
		}
	}
}
