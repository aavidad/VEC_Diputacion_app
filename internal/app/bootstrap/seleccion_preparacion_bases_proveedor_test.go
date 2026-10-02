package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Fixture unitario del holder sellado: no se conecta como infraestructura.
func brokerPreparacionBasesPrueba(t *testing.T, i int) (*proveedorPreparacionBasesV3, context.Context) {
	t.Helper()
	ps := perfilesPreparacionBasesPrueba(t)
	ds, err := fronterasPreparacionBasesHTTPV3(ps[0].perfilRef(), ps[1].perfilRef())
	if err != nil {
		t.Fatal(err)
	}
	cat, err := nuevoCatalogoFronterasComunDesarrollo(ds)
	if err != nil {
		t.Fatal(err)
	}
	var ss [2]*proveedorSesionConsultaRRHHDesarrollo
	for j, p := range ps {
		p.soporte.contextoEsperadoRegistrado = p.soporte.contexto.Resultado
		ss[j] = &proveedorSesionConsultaRRHHDesarrollo{soporte: p.soporte, base: p.soporte.contexto.Resultado, fronteras: cat}
	}
	p := &proveedorPreparacionBasesV3{perfiles: ps, sesiones: ss}
	perfil := ps[i]
	principal := core.Principal{ID: perfil.soporte.principalID, Roles: []string{"tecnico_rrhh"}, AuthMethod: core.AuthMethodCertificate, AuthAssurance: core.AuthAssuranceHigh,
		Attributes: map[string]string{"certificate_sha256": perfil.soporte.certificadoSHA256, "autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": "desarrollo"}}
	c := capacidadConsultaContratacionTemporalDesarrollo{sello: perfil.soporte.sello, principal: principal, ruta: perfil.ruta, metodo: http.MethodPost,
		contextoOperacion: &contextoOperacionCTDesarrollo{soporte: perfil.soporte, contexto: perfil.soporte.contexto}}
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
	f := fronteraSeguridadComunDesarrollo{metodo: http.MethodPost, ruta: perfil.ruta, superficie: superficieInternaSeguridadComunDesarrollo, catalogo: cat, descriptor: ds[i]}
	return p, context.WithValue(ctx, claveFronteraSeguridadComunDesarrollo{}, f)
}

func TestPreparacionBasesBrokerConservaContextoYSegregaHolder(t *testing.T) {
	for i := range 2 {
		p, ctx := brokerPreparacionBasesPrueba(t, i)
		i1, err := p.ResolverIdentidadConvocatoria(ctx)
		if err != nil {
			t.Fatal(err)
		}
		i2, err := p.ResolverIdentidadConvocatoria(ctx)
		if err != nil || !bytes.Equal(i1.Resultado.RepresentacionCanonica, i2.Resultado.RepresentacionCanonica) {
			t.Fatal("identidad no estable en la petición")
		}
		r := httptest.NewRequest(http.MethodPost, p.perfiles[i].ruta, nil).WithContext(ctx)
		httpctx, err := p.ResolverContextoHTTP(r)
		if err != nil || httpctx.Actor.PerfilActivoRef != p.perfiles[i].perfilRef() || httpctx.Ambito != p.perfiles[i].ambito || httpctx.Correlacion.Validar() != nil {
			t.Fatal("contexto HTTP no procede de autoridad fijada")
		}
		cancelado, cancelar := context.WithCancel(ctx)
		cancelar()
		if _, err := p.ResolverIdentidadConvocatoria(cancelado); !errors.Is(err, bolsaports.ErrPreparacionBasesNoDisponible) {
			t.Fatal("cancelación declarada mal clasificada")
		}
		c := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		c.contextoOperacion = &contextoOperacionCTDesarrollo{soporte: p.perfiles[1-i].soporte, contexto: p.perfiles[1-i].soporte.contexto}
		if _, err := p.ResolverIdentidadConvocatoria(context.WithValue(ctx, claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)); !errors.Is(err, bolsaports.ErrPreparacionBasesDenegada) {
			t.Fatal("holder de otro perfil aceptado")
		}
		if _, err := p.ResolverIdentidadConvocatoria(context.Background()); !errors.Is(err, bolsaports.ErrPreparacionBasesDenegada) {
			t.Fatal("identidad sin capacidad aceptada")
		}
	}
}

func solicitudPreparacionBasesBrokerPrueba(t *testing.T, p *proveedorPreparacionBasesV3, ctx context.Context) core.SolicitudAutorizacionLigadaV3 {
	t.Helper()
	z, i, err := p.contexto(ctx)
	if err != nil {
		t.Fatal(err)
	}
	perfil := p.perfiles[i]
	c, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: z.Vinculo,
		ReferenciaMotivo: perfil.motivo, Accion: perfil.accion, Finalidad: bolsaports.FinalidadPreparacionBases, Correlacion: c,
		Recurso: core.RecursoAutorizable{Referencia: "preparacion:unidad", ModuloID: "bolsa", Tipo: bolsaports.TipoRecursoPreparacionBases,
			Ambitos: map[string]string{"organizacion_ref": perfil.ambito.OrganizacionRef(), "unidad_gestion_ref": perfil.ambito.UnidadGestionRef()}, Atributos: map[string]string{"material_sha256": strings.Repeat("a", 64)}}})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPreparacionBasesBrokerRechazaRecursoYOperacionCruzados(t *testing.T) {
	p, ctx := brokerPreparacionBasesPrueba(t, 0)
	z, _, err := p.contexto(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s := solicitudPreparacionBasesBrokerPrueba(t, p, ctx)
	if validarSolicitudMaterialPreparacionBasesV3(s, z, p.perfiles[0]) != nil {
		t.Fatal("solicitud nominal rechazada")
	}
	for nombre, mutar := range map[string]func(*core.DatosSolicitudAutorizacionLigadaV3){
		"consulta_sobre_guardar": func(d *core.DatosSolicitudAutorizacionLigadaV3) {
			d.Accion = bolsaports.AccionConsultarPreparacionBases
		},
		"ambito_distinto": func(d *core.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos["organizacion_ref"] = "org_" + strings.Repeat("c", 16)
		},
		"ambito_adicional":   func(d *core.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Ambitos["centro_ref"] = "centro:otro" },
		"material_adicional": func(d *core.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Atributos["permiso"] = "cliente" },
	} {
		t.Run(nombre, func(t *testing.T) {
			d, err := s.Datos()
			if err != nil {
				t.Fatal(err)
			}
			mutar(&d)
			otra, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
			if err == nil && validarSolicitudMaterialPreparacionBasesV3(otra, z, p.perfiles[0]) == nil {
				t.Fatal("solicitud cruzada aceptada")
			}
		})
	}
}

type pdpPreparacionBasesDenegadoPrueba struct{ llamadas int }

func (p *pdpPreparacionBasesDenegadoPrueba) ExigirSolicitudLigadaV3(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	p.llamadas++
	return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, core.ErrAutorizacionDenegada
}

func TestPreparacionBasesBrokerDenegacionNoEmiteMaterial(t *testing.T) {
	p, ctx := brokerPreparacionBasesPrueba(t, 0)
	pdp := &pdpPreparacionBasesDenegadoPrueba{}
	p.pdp = pdp
	z, _, err := p.contexto(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, _, exportador, err := p.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitudPreparacionBasesBrokerPrueba(t, p, ctx), z.Resultado)
	if !errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) || exportador != nil || pdp.llamadas != 1 {
		t.Fatal("PDP denegado emitió material o clasificación distinta")
	}
}

func TestPreparacionBasesBrokerFallosPDPCombinadosSonTecnicos(t *testing.T) {
	for _, err := range []error{vecports.ErrFuenteAutorizacionNoDisponible, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, errAutorizacionComunDesarrolloNoDisponible, context.Canceled} {
		if !errorTecnicoPDPPreparacionBasesV3(context.Background(), errors.Join(core.ErrAutorizacionDenegada, err)) {
			t.Fatal("caída del PDP clasificada como concesión denegada")
		}
	}
	if errorTecnicoPDPPreparacionBasesV3(context.Background(), core.ErrAutorizacionDenegada) {
		t.Fatal("denegación vigente clasificada como caída")
	}
}

func TestPreparacionBasesMontajeSinDependenciasYFallback(t *testing.T) {
	if _, _, err := (*MontajePreparacionBasesV3)(nil).Componer(context.Background(), DependenciasMontajePreparacionBasesV3{}); err == nil {
		t.Fatal("montaje vacío admitido")
	}
	for _, ruta := range RutasPreparacionBasesIndisponiblesV3() {
		w := httptest.NewRecorder()
		ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodPost, ruta.Ruta, nil))
		if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" || bytes.Contains(w.Body.Bytes(), []byte("dsn")) {
			t.Fatal("fallo no cerrado o datos del montaje expuestos")
		}
	}
}
