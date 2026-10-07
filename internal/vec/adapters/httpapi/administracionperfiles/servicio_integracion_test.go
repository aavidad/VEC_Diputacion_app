package administracionperfiles

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type catalogoFocal struct {
	rol   domain.RolAdministrable
	actor domain.RolAdministrable
}

func (c catalogoFocal) ResolverRolAdministrable(_ context.Context, ref string) (domain.RolAdministrable, error) {
	if ref == c.actor.VersionRef {
		return c.actor, nil
	}
	return c.rol, nil
}

type relojFocal struct{ ahora time.Time }

func (r relojFocal) Ahora() time.Time { return r.ahora }

type autoridadFocal struct {
	llamada    *domain.SolicitudActoAdministracionPerfiles
	alteracion bool
}

func (a *autoridadFocal) AplicarActoOrdinario(_ context.Context, s domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	a.llamada = &s
	r := domain.ReciboAdministracionPerfiles{OperacionRef: s.OperacionRef, ActoRef: s.OperacionRef, ReciboRef: "recibo_admin:" + strings.Repeat("a", 32), AuditoriaRef: "auditoria:prueba", ObjetivoPersonaRef: s.Objetivo.PersonaRef, PerfilRef: s.Objetivo.PerfilRef, VinculoRef: s.Objetivo.VinculoRef, VersionPosterior: 1, EstadoPosterior: domain.EstadoVinculoContextoActorActivo, HuellaAntesSHA256: s.Objetivo.HuellaSHA256, HuellaDespuesSHA256: strings.Repeat("f", 64), ConfirmadoEn: s.Actor.ResueltoEn, UnidadRef: s.Objetivo.UnidadRef, ReferenciaActo: s.ReferenciaActo, CentroRef: s.Objetivo.CentroRef, ActorPersonaRef: s.Actor.PersonaRef, PerfilActivoRef: s.Actor.PerfilActivoRef, AsignacionPerfilRef: s.InstantaneaAutorizacion.AsignacionPerfil.Referencia(), CorrelacionRef: s.CorrelacionRef, RolVersionRef: s.RolVersionRef, VigenteDesde: s.Objetivo.VigenteDesde, VigenteHasta: s.Objetivo.VigenteHasta, Motivo: s.Motivo}
	if a.alteracion {
		r.UnidadRef = "unidad:otra"
	}
	return r, nil
}
func (a *autoridadFocal) ProponerActoSensible(context.Context, domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error) {
	return ports.PropuestaAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func (a *autoridadFocal) CerrarPropuestaSensible(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	return ports.CierrePropuestaAdministracionPerfiles{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}

func TestActoHTTPConservaUnidadYReferenciaOpcional(t *testing.T) {
	for _, caso := range []struct {
		nombre, unidad, referencia string
		alteracion                 bool
		estado                     int
		efecto                     bool
		categoria                  string
		rolActorID                 string
	}{
		{"sin acto legal", "unidad:prueba", "", false, http.StatusOK, true, "aplicacion", "administracion_perfiles"},
		{"con acto legal", "unidad:prueba", "Resolución 2026/123", false, http.StatusOK, true, "aplicacion", "administracion_perfiles"},
		{"sin ámbito obligatorio", "", "", false, http.StatusBadRequest, false, "aplicacion", "administracion_perfiles"},
		{"recibo de otra unidad", "unidad:prueba", "", true, http.StatusServiceUnavailable, true, "aplicacion", "administracion_perfiles"},
		{"referencia multilinea", "unidad:prueba", "acto\n123", false, http.StatusBadRequest, false, "aplicacion", "administracion_perfiles"},
		{"Sistemas con misma clase administrativa", "unidad:prueba", "", false, http.StatusServiceUnavailable, false, "sistemas", "administracion_perfiles"},
		{"categoría no acreditada", "unidad:prueba", "", false, http.StatusServiceUnavailable, false, "", "administracion_perfiles"},
		{"otro rol con categoría Aplicación", "unidad:prueba", "", false, http.StatusServiceUnavailable, false, "aplicacion", "otro_perfil"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			sesion := sesionADMINPrueba(t)
			sesion.InstantaneaAutorizacion.VersionRol.RolID = caso.rolActorID
			sesion.InstantaneaAutorizacion.AsignacionPerfil.VersionRolRef = sesion.InstantaneaAutorizacion.VersionRol.Referencia()
			sesion.InstantaneaAutorizacion.ControlVigenciaVersionRol.VersionRolRef = sesion.InstantaneaAutorizacion.VersionRol.Referencia()
			huellaActor, err := sesion.InstantaneaAutorizacion.VersionRol.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			rolActor := domain.RolAdministrable{VersionRef: sesion.InstantaneaAutorizacion.VersionRol.Referencia(), Clase: domain.ClaseControlPerfilAdministrador, CategoriaAdmin: caso.categoria, HuellaSHA256: huellaActor, VigenteDesde: sesion.Actor.ResueltoEn.Add(-time.Hour), VigenteHasta: sesion.Actor.ResueltoEn.Add(time.Hour)}
			rol := domain.RolAdministrable{VersionRef: "rol:dietas_liquidacion_rrhh:v1", Clase: domain.ClaseControlPerfilOrdinario, HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: sesion.Actor.ResueltoEn.Add(-time.Hour), VigenteHasta: sesion.Actor.ResueltoEn.Add(time.Hour), UnidadRequerida: true}
			autoridad := &autoridadFocal{alteracion: caso.alteracion}
			servicio, err := application.NuevoServicioAdministracionPerfiles(catalogoFocal{rol: rol, actor: rolActor}, autoridad, relojFocal{sesion.Actor.ResueltoEn})
			if err != nil {
				t.Fatal(err)
			}
			h, err := NuevoHandler("https://admin.example.test", &sesionPrueba{resultado: sesion}, &lecturasPrueba{}, catalogoFocal{rol: rol, actor: rolActor}, servicio, &auditorPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			dto := SolicitudActo{OperacionRef: "acto_admin:" + strings.Repeat("b", 32), Operacion: "otorgar", RolVersionRef: rol.VersionRef, ReferenciaActo: caso.referencia,
				Objetivo: Objetivo{VigenteDesde: sesion.Actor.ResueltoEn.Add(-time.Hour), UnidadRef: caso.unidad, CuentaRef: "cta_" + strings.Repeat("f", 22), CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("f", 22), PersonaVersion: 1, PerfilRef: "prf_" + strings.Repeat("f", 22), VinculoRef: "vca_" + strings.Repeat("f", 22), HuellaSHA256: strings.Repeat("c", 64), ProcedenciaRef: "procedencia:maestra:prueba", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("d", 64), VigenteHasta: sesion.Actor.ResueltoEn.Add(time.Hour)},
				Motivo:   Motivo{CatalogoID: "motivos_admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "provision"}}
			cuerpo, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/actos-ordinarios", string(cuerpo)))
			if w.Code != caso.estado {
				t.Fatalf("estado=%d esperado=%d cuerpo=%s", w.Code, caso.estado, w.Body.String())
			}
			if (autoridad.llamada != nil) != caso.efecto {
				t.Fatal("efecto de autoridad inesperado")
			}
			if autoridad.llamada != nil && (autoridad.llamada.ReferenciaActo != caso.referencia ||
				autoridad.llamada.Objetivo.UnidadRef != caso.unidad ||
				autoridad.llamada.Evidencia.ValidarPara(sesion.Actor) != nil) {
				t.Fatal("se perdieron unidad, acto o evidencia de sesión")
			}
		})
	}
}

// Este proveedor incumple deliberadamente el contrato del servicio. El HTTP
// debe rechazar su resultado, aunque la referencia de operación coincida.
type servicioReciboDefectuoso struct {
	actosPrueba
	alterar func(*domain.ReciboAdministracionPerfiles)
}

func (s *servicioReciboDefectuoso) AplicarOrdinario(ctx context.Context, solicitud domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error) {
	s.llamadas++
	autoridad := &autoridadFocal{}
	recibo, err := autoridad.AplicarActoOrdinario(ctx, solicitud)
	if err != nil {
		return recibo, err
	}
	s.alterar(&recibo)
	return recibo, nil
}

func TestHTTPRechazaReciboCruzadoAunqueOperacionCoincida(t *testing.T) {
	for nombre, alterar := range map[string]func(*domain.ReciboAdministracionPerfiles){
		"actor":         func(r *domain.ReciboAdministracionPerfiles) { r.ActorPersonaRef = "per_" + strings.Repeat("g", 22) },
		"perfil activo": func(r *domain.ReciboAdministracionPerfiles) { r.PerfilActivoRef = "prf_" + strings.Repeat("g", 22) },
		"asignación":    func(r *domain.ReciboAdministracionPerfiles) { r.AsignacionPerfilRef = "asignacion:otra:v1" },
		"correlación": func(r *domain.ReciboAdministracionPerfiles) {
			r.CorrelacionRef = "correlacion_" + strings.Repeat("g", 32)
		},
		"destinatario": func(r *domain.ReciboAdministracionPerfiles) { r.ObjetivoPersonaRef = "per_" + strings.Repeat("g", 22) },
		"unidad":       func(r *domain.ReciboAdministracionPerfiles) { r.UnidadRef = "unidad:otra" },
		"centro":       func(r *domain.ReciboAdministracionPerfiles) { r.CentroRef = "centro:otro" },
		"motivo":       func(r *domain.ReciboAdministracionPerfiles) { r.Motivo.EntradaClave = "otro" },
		"rol":          func(r *domain.ReciboAdministracionPerfiles) { r.RolVersionRef = "rol:otro:v1" },
		"vigencia":     func(r *domain.ReciboAdministracionPerfiles) { r.VigenteHasta = r.VigenteHasta.Add(time.Hour) },
	} {
		t.Run(nombre, func(t *testing.T) {
			sesion := sesionAplicacionNominalPrueba(t)
			rol := domain.RolAdministrable{VersionRef: "rol:dietas_liquidacion_rrhh:v1", Clase: domain.ClaseControlPerfilOrdinario, HuellaSHA256: strings.Repeat("a", 64), VigenteDesde: sesion.Actor.ResueltoEn.Add(-time.Hour), VigenteHasta: sesion.Actor.ResueltoEn.Add(time.Hour), UnidadRequerida: true}
			servicio := &servicioReciboDefectuoso{alterar: alterar}
			h, err := NuevoHandler("https://admin.example.test", &sesionPrueba{resultado: sesion}, &lecturasPrueba{}, catalogoFocal{rol: rol}, servicio, &auditorPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			dto := SolicitudActo{OperacionRef: "acto_admin:" + strings.Repeat("b", 32), Operacion: "otorgar", RolVersionRef: rol.VersionRef,
				Objetivo: Objetivo{VigenteDesde: sesion.Actor.ResueltoEn.Add(-time.Hour), VigenteHasta: sesion.Actor.ResueltoEn.Add(time.Hour), UnidadRef: "unidad:prueba", CuentaRef: "cta_" + strings.Repeat("f", 22), CuentaVersion: 1, PersonaRef: "per_" + strings.Repeat("f", 22), PersonaVersion: 1, PerfilRef: "prf_" + strings.Repeat("f", 22), VinculoRef: "vca_" + strings.Repeat("f", 22), HuellaSHA256: strings.Repeat("c", 64), ProcedenciaRef: "procedencia:maestra:prueba", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("d", 64)},
				Motivo:   Motivo{CatalogoID: "motivos_admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("e", 64), EntradaClave: "provision"}}
			b, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/actos-ordinarios", string(b)))
			if w.Code != http.StatusServiceUnavailable || servicio.llamadas != 1 || strings.Contains(w.Body.String(), `"recibo"`) || strings.Contains(w.Body.String(), "per_") || strings.Contains(w.Body.String(), "unidad:") {
				t.Fatalf("estado=%d llamadas=%d cuerpo=%s", w.Code, servicio.llamadas, w.Body.String())
			}
		})
	}
}

type servicioCierreFocal struct {
	actosPrueba
	cruzado bool
}

func (s *servicioCierreFocal) CerrarPropuestaSensible(_ context.Context, solicitud domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error) {
	s.llamadas++
	cierre := ports.CierrePropuestaAdministracionPerfiles{OperacionRef: solicitud.OperacionRef, PropuestaRef: solicitud.PropuestaRef, PropuestaHuellaSHA256: solicitud.PropuestaHuellaSHA256, Decision: solicitud.Decision, HuellaCierreSHA256: strings.Repeat("e", 64), ConfirmadoEn: solicitud.Aprobador.ResueltoEn}
	if s.cruzado {
		cierre.PropuestaHuellaSHA256 = strings.Repeat("f", 64)
	}
	return cierre, nil
}

func TestCierreHTTPConservaHuellaPropuestaYRechazaCruce(t *testing.T) {
	for _, cruzado := range []bool{false, true} {
		t.Run(map[bool]string{false: "huella aprobada", true: "otra huella"}[cruzado], func(t *testing.T) {
			sesion := sesionAplicacionNominalPrueba(t)
			ref := "propuesta_admin:" + strings.Repeat("a", 32)
			huella := strings.Repeat("b", 64)
			l := &lecturasPrueba{propuesta: Propuesta{PropuestaRef: ref, HuellaSHA256: huella, ProponentePersonaRef: "per_" + strings.Repeat("f", 22), ObjetivoPersonaRef: "per_" + strings.Repeat("g", 22)}}
			servicio := &servicioCierreFocal{cruzado: cruzado}
			h, err := NuevoHandler("https://admin.example.test", &sesionPrueba{resultado: sesion}, l, &catalogoPrueba{}, servicio, &auditorPrueba{})
			if err != nil {
				t.Fatal(err)
			}
			dto := SolicitudCierre{OperacionRef: "cierre_admin:" + strings.Repeat("c", 32), PropuestaHuellaSHA256: huella, Decision: string(domain.DecisionRechazarPropuestaPerfil), Motivo: Motivo{CatalogoID: "motivos_admin", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "revision"}}
			b, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodPost, PrefijoV1+"/propuestas/"+ref+"/cierre", string(b)))
			esperado := http.StatusOK
			if cruzado {
				esperado = http.StatusServiceUnavailable
			}
			if w.Code != esperado || servicio.llamadas != 1 {
				t.Fatalf("estado=%d llamadas=%d cuerpo=%s", w.Code, servicio.llamadas, w.Body.String())
			}
			if cruzado && strings.Contains(w.Body.String(), `"cierre"`) {
				t.Fatal("cierre cruzado expuesto")
			}
			if !cruzado {
				var respuesta struct {
					Cierre struct {
						Huella string `json:"propuesta_huella_sha256"`
					} `json:"cierre"`
				}
				if json.Unmarshal(w.Body.Bytes(), &respuesta) != nil || respuesta.Cierre.Huella != huella {
					t.Fatal("no se conserva la huella de la propuesta")
				}
			}
		})
	}
}

func sesionAplicacionNominalPrueba(t *testing.T) SesionConfiable {
	t.Helper()
	sesion := sesionADMINPrueba(t)
	sesion.InstantaneaAutorizacion.VersionRol.RolID = "administracion_perfiles"
	sesion.InstantaneaAutorizacion.AsignacionPerfil.VersionRolRef = sesion.InstantaneaAutorizacion.VersionRol.Referencia()
	sesion.InstantaneaAutorizacion.ControlVigenciaVersionRol.VersionRolRef = sesion.InstantaneaAutorizacion.VersionRol.Referencia()
	return sesion
}
