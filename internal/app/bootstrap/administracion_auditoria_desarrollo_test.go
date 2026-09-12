package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	admin "vec-diputacion-granada/internal/modules/administracion"
	adminhttp "vec-diputacion-granada/internal/modules/administracion/adapters/http"
	docseg "vec-diputacion-granada/internal/vec/adapters/documentos/seguridad"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type sesionAuditoriaAdministracionPrueba struct {
	valor    contextoSesionAdministracionCorreoV3
	llamadas int
	despues  func()
}

func (s *sesionAuditoriaAdministracionPrueba) ResolverSesionAdministracionCorreoV3(context.Context) (contextoSesionAdministracionCorreoV3, error) {
	s.llamadas++
	if s.despues != nil {
		s.despues()
	}
	return s.valor, nil
}

type seudonimizadorAuditoriaAdministracionPrueba func(context.Context, vp.SolicitudSeudonimizarSujetoAlmacen) (string, error)

func (s seudonimizadorAuditoriaAdministracionPrueba) SeudonimizarSujetoAlmacen(ctx context.Context, solicitud vp.SolicitudSeudonimizarSujetoAlmacen) (string, error) {
	return s(ctx, solicitud)
}

type escenarioAuditoriaAdministracionPrueba struct {
	material       materialTLSAdministracionPrueba
	transporte     *autoridadAdministracionDesarrollo
	cfg            config.Config
	sesion         *sesionAuditoriaAdministracionPrueba
	seudonimizador *docseg.SelladorHMAC
	preparador     *preparacionAuditoriaAdministracionDesarrollo
}

func nuevaAuditoriaAdministracionPrueba(t *testing.T) escenarioAuditoriaAdministracionPrueba {
	t.Helper()
	c, _, _, reloj := materialEmisorAdministracionPrueba(t)
	reloj.ahora = time.Now().UTC().Truncate(time.Microsecond)
	solicitud, resultado, _ := solicitudEmisorAdministracionPrueba(t, c, reloj)
	datos, err := solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	m := nuevoMaterialTLSAdministracionPrueba(t)
	transporte, _, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	transporte.identidad.identidad.principal.ID = "administracion-sintetica"
	transporte.identidad.cuentaRef, transporte.identidad.cuentaOrdinariaRef = c.CuentaRef, c.CuentaOrdinariaRef
	transporte.identidad.personaRef, transporte.identidad.perfilRef = c.PersonaRef, c.PerfilRef
	principal := clonarPrincipalDesarrollo(resultado.Contexto.Principal)
	// Roles de la sesión de prueba, nunca copiados del certificado nominal.
	principal.Roles = []string{"rol:integraciones:v1", "rol:administracion:v2"}
	principal.Permissions = []string{admin.PermissionIntegrationsManage}
	sesion := &sesionAuditoriaAdministracionPrueba{valor: contextoSesionAdministracionCorreoV3{Principal: principal, Vinculo: datos.VinculoAutenticacionActor, Resultado: resultado}}
	seudonimizador, err := docseg.NuevoSelladorHMAC("auditoria-admin-prueba-v1", bytes.Repeat([]byte{0xa3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	preparador, err := nuevaPreparacionAuditoriaAdministracionDesarrollo(sesion, seudonimizador, seg.GeneradorReferenciasCriptograficas{}, docseg.RelojSistema{})
	if err != nil {
		t.Fatal(err)
	}
	return escenarioAuditoriaAdministracionPrueba{m, transporte, cfg, sesion, seudonimizador, preparador}
}

func comprobarAuditoriaAdministracionTLS(t *testing.T, e escenarioAuditoriaAdministracionPrueba, comprobar func(context.Context) error) {
	t.Helper()
	observacion := make(chan error, 1)
	url := iniciarServidorAdministracionPrueba(t, e.transporte, e.cfg, e.material, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observacion <- comprobar(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	cliente := clienteAdministracionPrueba(e.material, &e.material.admin)
	defer cliente.CloseIdleConnections()
	peticion, err := http.NewRequest(http.MethodPut, url+adminhttp.RutaConfiguracionCorreo, nil)
	if err != nil {
		t.Fatal(err)
	}
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusNoContent {
		t.Fatalf("estado TLS %d", respuesta.StatusCode)
	}
	if err := <-observacion; err != nil {
		t.Fatal(err)
	}
}

func TestAuditoriaAdministracionPreparaT13ConSesionYHMACReales(t *testing.T) {
	e := nuevaAuditoriaAdministracionPrueba(t)
	rolesOriginales := append([]string(nil), e.sesion.valor.Principal.Roles...)
	comprobarAuditoriaAdministracionTLS(t, e, func(ctx context.Context) error {
		a, err := e.preparador.PrepararAuditoriaConfiguracionCorreo(ctx, e.sesion.valor.Principal, 7)
		if err != nil {
			return err
		}
		if err := e.preparador.ValidarAuditoriaParaSesion(ctx, a, e.sesion.valor, 7); err != nil {
			return fmt.Errorf("entrada propia no ligada: %w", err)
		}
		s, err := vp.NuevaSolicitudSeudonimizarSujetoAlmacen(e.sesion.valor.Principal.ID, "bolsa_registro_accesos_t13")
		if err != nil {
			return err
		}
		esperado, err := e.seudonimizador.SeudonimizarSujetoAlmacen(ctx, s)
		if err != nil || a.ActorID != esperado || a.ActorID == e.sesion.valor.Principal.ID || a.ActorProfile != e.sesion.valor.Resultado.Contexto.PerfilActivoRef || !sort.StringsAreSorted(a.ActorRoles) || !reflect.DeepEqual(e.sesion.valor.Principal.Roles, rolesOriginales) || e.sesion.llamadas != 1 {
			return errors.New("identidad, orden o revalidación de auditoría incoherentes")
		}
		b, err := json.Marshal(a)
		if err != nil {
			return err
		}
		var campos map[string]json.RawMessage
		if json.Unmarshal(b, &campos) != nil || len(campos) != 16 || bytes.Contains(b, []byte(e.sesion.valor.Principal.ID)) || bytes.Contains(b, []byte("administracion-sintetica")) {
			return errors.New("DTO T13 no exacto o revela identidad nominal")
		}
		for _, clave := range []string{"id", "seq", "signature", "actor_id", "actor_profile", "actor_roles", "auth_method", "auth_assurance", "purpose", "action", "module_id", "subject_ref", "object_version", "result", "correlation_ref", "occurred_at"} {
			if _, ok := campos[clave]; !ok {
				return fmt.Errorf("falta campo T13 %s", clave)
			}
		}
		if a.ObjectVersion != 7 || a.AuthMethod != core.AuthMethodCertificate || a.AuthAssurance != core.AuthAssuranceHigh || a.Purpose != finalidadConfiguracionCorreoAdministracionV3 || a.Action != accionConfiguracionCorreoAdministracionV3 || a.ModuleID != admin.ModuleID || a.SubjectRef != referenciaConfiguracionCorreoAdministracionV3 || a.Result != "accepted" || a.OccurredAt != a.OccurredAt.UTC().Truncate(time.Microsecond) || !core.ReferenciaCorrelacionAutorizacionV2Valida(a.CorrelationRef) {
			return errors.New("contenido T13 incompleto")
		}
		segunda, err := e.preparador.PrepararAuditoriaConfiguracionCorreo(ctx, e.sesion.valor.Principal, 7)
		if err != nil || segunda.CorrelationRef == a.CorrelationRef || segunda.ActorID != a.ActorID || e.sesion.llamadas != 2 {
			return errors.New("correlación compartida o seudónimo inestable")
		}
		return nil
	})
}

func TestAuditoriaAdministracionDeniegaPrincipalContextoYCancelacion(t *testing.T) {
	for _, caso := range []string{"principal_ajeno", "roles_ajenos", "perfil_cruzado", "sin_roles", "sin_permiso", "cancelada_entrada", "cancelada_sesion", "cancelada_hmac", "hmac_no_disponible", "hmac_formato", "version_cero", "version_excesiva"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevaAuditoriaAdministracionPrueba(t)
			comprobarAuditoriaAdministracionTLS(t, e, func(ctx context.Context) error {
				ctx, cancelar := context.WithCancel(ctx)
				defer cancelar()
				principal := clonarPrincipalDesarrollo(e.sesion.valor.Principal)
				version := uint64(1)
				switch caso {
				case "principal_ajeno":
					principal.ID = "per_eeeeeeeeeeeeeeeeeeeeee"
				case "roles_ajenos":
					principal.Roles = []string{"rol:otro:v1"}
				case "perfil_cruzado":
					e.sesion.valor.Resultado.Contexto.PerfilActivoRef = "prf_eeeeeeeeeeeeeeeeeeeeee"
				case "sin_roles":
					e.sesion.valor.Principal.Roles = nil
				case "sin_permiso":
					e.sesion.valor.Principal.Permissions = nil
				case "cancelada_entrada":
					cancelar()
				case "cancelada_sesion":
					e.sesion.despues = cancelar
				case "cancelada_hmac":
					e.preparador.seudonimizador = seudonimizadorAuditoriaAdministracionPrueba(func(ctx context.Context, s vp.SolicitudSeudonimizarSujetoAlmacen) (string, error) {
						v, err := e.seudonimizador.SeudonimizarSujetoAlmacen(ctx, s)
						cancelar()
						return v, err
					})
				case "hmac_no_disponible":
					e.preparador.seudonimizador = seudonimizadorAuditoriaAdministracionPrueba(func(context.Context, vp.SolicitudSeudonimizarSujetoAlmacen) (string, error) {
						return "", errors.New("detalle privado del proveedor")
					})
				case "hmac_formato":
					e.preparador.seudonimizador = seudonimizadorAuditoriaAdministracionPrueba(func(context.Context, vp.SolicitudSeudonimizarSujetoAlmacen) (string, error) {
						return "hmac-sha256:clave:" + strings.Repeat("0", 64), nil
					})
				case "version_cero":
					version = 0
				case "version_excesiva":
					version = maximaVersionAuditoriaAdministracion + 1
				}
				a, err := e.preparador.PrepararAuditoriaConfiguracionCorreo(ctx, principal, version)
				if err != errPreparacionAuditoriaAdministracion || !reflect.DeepEqual(a, core.AuditEntry{}) {
					return fmt.Errorf("negativa %s no cerrada", caso)
				}
				return nil
			})
		})
	}
}

func TestAuditoriaAdministracionCotejaPseudonimoExactoAntesEmision(t *testing.T) {
	e := nuevaAuditoriaAdministracionPrueba(t)
	comprobarAuditoriaAdministracionTLS(t, e, func(ctx context.Context) error {
		a, err := e.preparador.PrepararAuditoriaConfiguracionCorreo(ctx, e.sesion.valor.Principal, 1)
		if err != nil {
			return err
		}
		s, err := vp.NuevaSolicitudSeudonimizarSujetoAlmacen("per_eeeeeeeeeeeeeeeeeeeeee", ambitoSeudonimoAuditoriaAdministracion)
		if err != nil {
			return err
		}
		ajeno, err := e.seudonimizador.SeudonimizarSujetoAlmacen(ctx, s)
		if err != nil {
			return err
		}
		for _, caso := range []string{"actor", "perfil", "roles", "version", "campo_extra", "firma", "fecha_futura", "nanosegundos", "cancelacion"} {
			otra := a
			activo := ctx
			switch caso {
			case "actor":
				otra.ActorID = ajeno
			case "perfil":
				otra.ActorProfile = "prf_eeeeeeeeeeeeeeeeeeeeee"
			case "roles":
				otra.ActorRoles = []string{"rol:otro:v1"}
			case "version":
				otra.ObjectVersion = 2
			case "campo_extra":
				otra.Metadata = map[string]string{"contenido": "no admitido"}
			case "firma":
				otra.Signature = "firma-fabricada"
			case "fecha_futura":
				otra.OccurredAt = otra.OccurredAt.Add(time.Hour)
			case "nanosegundos":
				otra.OccurredAt = otra.OccurredAt.Add(time.Nanosecond)
			case "cancelacion":
				var cancelar context.CancelFunc
				activo, cancelar = context.WithCancel(ctx)
				cancelar()
			}
			if e.preparador.ValidarAuditoriaParaSesion(activo, otra, e.sesion.valor, 1) == nil {
				return fmt.Errorf("auditoría %s aceptada", caso)
			}
		}
		if e.sesion.llamadas != 1 {
			return errors.New("el cotejo sustituyó la sesión que iba a emitir V3")
		}
		return e.preparador.ValidarAuditoriaParaSesion(ctx, a, e.sesion.valor, 1)
	})
}

func TestAuditoriaAdministracionConstructorCierraDependenciasYRedacta(t *testing.T) {
	e := nuevaAuditoriaAdministracionPrueba(t)
	var sesionNula *sesionAuditoriaAdministracionPrueba
	var seudonimizadorNulo *docseg.SelladorHMAC
	for _, caso := range []string{"sesion", "seudonimizador", "referencias", "reloj"} {
		var sesion sesionDurableAdministracionCorreoV3 = e.sesion
		var seudonimizador vp.SeudonimizadorSujetoAlmacen = e.seudonimizador
		var referencias vp.GeneradorReferenciasAutorizacionV2 = seg.GeneradorReferenciasCriptograficas{}
		var reloj vp.Reloj = docseg.RelojSistema{}
		switch caso {
		case "sesion":
			sesion = sesionNula
		case "seudonimizador":
			seudonimizador = seudonimizadorNulo
		case "referencias":
			referencias = nil
		case "reloj":
			reloj = nil
		}
		if p, err := nuevaPreparacionAuditoriaAdministracionDesarrollo(sesion, seudonimizador, referencias, reloj); p != nil || err == nil {
			t.Fatalf("dependencia %s aceptada", caso)
		}
	}
	for _, valor := range []any{e.preparador, *e.preparador} {
		for _, formato := range []string{"%v", "%+v", "%#v"} {
			if fmt.Sprintf(formato, valor) != "preparacionAuditoriaAdministracion{redactada}" {
				t.Fatal("preparador no redactado")
			}
		}
		b, err := json.Marshal(valor)
		if err != nil || string(b) != `{"redactado":true}` {
			t.Fatal("JSON no redactado")
		}
	}
}
