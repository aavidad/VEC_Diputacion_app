package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	admin "vec-diputacion-granada/internal/modules/administracion"
	adminhttp "vec-diputacion-granada/internal/modules/administracion/adapters/http"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorAmbitoAdministracionPrueba func(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error)

func (f autorizadorAmbitoAdministracionPrueba) ExigirSolicitudLigadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	return f(ctx, s, r)
}

// Fuente y guardia son las de producción. Sólo el PDP final es un observador
// que deniega: se prueba el ensamblaje hasta él, no una concesión PostgreSQL.
func TestAmbitoAdministracionFuenteRealTransmiteYGuardiaRestringe(t *testing.T) {
	c, _, _, reloj := materialEmisorAdministracionPrueba(t)
	base, resultado, _ := solicitudEmisorAdministracionPrueba(t, c, reloj)
	datosBase, err := base.Datos()
	if err != nil {
		t.Fatal(err)
	}
	m := nuevoMaterialTLSAdministracionPrueba(t)
	transporte, _, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	transporte.identidad.identidad.principal.ID = "administracion-sintetica"
	transporte.identidad.cuentaRef, transporte.identidad.cuentaOrdinariaRef = c.CuentaRef, c.CuentaOrdinariaRef
	transporte.identidad.personaRef, transporte.identidad.perfilRef = c.PersonaRef, c.PerfilRef
	if transporte.identidad.identidad.principal.ID == resultado.Contexto.Principal.ID {
		t.Fatal("la prueba requiere sujeto de certificado distinto de persona")
	}

	var capturada core.SolicitudAutorizacionLigadaV3
	llamadas := 0
	actorExacto := false
	errPDP := errors.New("PDP observador deniega deliberadamente")
	delegado := autorizadorAmbitoAdministracionPrueba(func(_ context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
		llamadas++
		capturada = s
		actorExacto = r.HuellaSHA256 == resultado.HuellaSHA256 && r.Contexto.Principal.ID == c.PersonaRef && r.Contexto.PerfilActivoRef == c.PerfilRef
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, errPDP
	})
	autoridad := &autoridadEmisorAdministracionDesarrollo{delegado: delegado, identidad: *transporte.identidad, motivo: c.Motivo, reloj: reloj}
	fuente, err := nuevaFuentePermisoAdministracionGobernadaV3(autoridad, c.Motivo, &referenciasPermisoAdministracionPrueba{}, reloj)
	if err != nil {
		t.Fatal(err)
	}

	comprobar := func(ctx context.Context) error {
		principal, err := fuente.ResolverPrincipalAdministracionCorreoV3(ctx, datosBase.VinculoAutenticacionActor, resultado)
		if llamadas != 1 || !actorExacto || err != ErrConfiguracionCorreoAdministracionNoDisponible || !reflect.DeepEqual(principal, core.Principal{}) {
			return fmt.Errorf("fuente real: llamadas PDP=%d; denegación propagada=%t; principal vacío=%t", llamadas, err == ErrConfiguracionCorreoAdministracionNoDisponible, reflect.DeepEqual(principal, core.Principal{}))
		}
		d, err := capturada.Datos()
		if err != nil || d.Accion != admin.PermissionIntegrationsManage || d.ReferenciaMotivo != c.Motivo || d.Recurso.Referencia != referenciaConfiguracionCorreoAdministracionV3 || d.Recurso.ModuloID != admin.ModuleID || d.Recurso.Tipo != tipoRecursoConfiguracionCorreoAdministracion || d.Finalidad != finalidadConfiguracionCorreoAdministracionV3 || len(d.Recurso.Atributos) != 0 || len(d.Recurso.Ambitos) != 1 || d.Recurso.Ambitos["organizacion_ref"] != "organizacion:dipgra" || !d.VinculoAutenticacionActor.CoincideExactamenteCon(datosBase.VinculoAutenticacionActor) {
			return errors.New("la fuente real no transmite recurso, ámbito e identidad exactos")
		}
		for _, caso := range []string{"ajeno", "ausente", "adicional"} {
			d, err := capturada.Datos()
			if err != nil {
				return err
			}
			switch caso {
			case "ajeno":
				d.Recurso.Ambitos["organizacion_ref"] = "organizacion:otra"
			case "ausente":
				d.Recurso.Ambitos = nil
			case "adicional":
				d.Recurso.Ambitos["unidad_ref"] = "unidad:otra"
			}
			cruzada, err := core.NuevaSolicitudAutorizacionLigadaV3(d)
			if err != nil {
				return fmt.Errorf("fixture %s: %w", caso, err)
			}
			_, _, err = autoridad.ExigirSolicitudLigadaV3(ctx, cruzada, resultado)
			if err == nil || llamadas != 1 {
				return fmt.Errorf("ámbito %s alcanzó el PDP", caso)
			}
		}
		return nil
	}
	observacion := make(chan error, 1)
	url := iniciarServidorAdministracionPrueba(t, transporte, cfg, m, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observacion <- comprobar(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	cliente := clienteAdministracionPrueba(m, &m.admin)
	defer cliente.CloseIdleConnections()
	respuesta, err := cliente.Get(url + adminhttp.RutaConfiguracionCorreo)
	if err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusNoContent {
		t.Fatalf("estado TLS=%d", respuesta.StatusCode)
	}
	if err := <-observacion; err != nil {
		t.Fatal(err)
	}
}
