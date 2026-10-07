package administracion

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpapi/adminselector"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type relojActivosPrueba struct{ ahora time.Time }

func (r relojActivosPrueba) Ahora() time.Time { return r.ahora }

type observadorActivosPrueba struct {
	o adminperfiles.ObservacionADMIN
}

func (o observadorActivosPrueba) ObservarADMIN(context.Context, *http.Request) (adminperfiles.ObservacionADMIN, error) {
	return o.o, nil
}

type auditorActivosPrueba struct{ llamadas int }

func (a *auditorActivosPrueba) RegistrarDenegacionADMIN(context.Context, api.DenegacionADMIN) error {
	a.llamadas++
	return nil
}

type fuenteActivosPrueba struct {
	resultado LecturaPropiosAuditadaADMIN
	llamadas  int
}

func (f *fuenteActivosPrueba) ListarPropiosAuditadosADMIN(context.Context, adminperfiles.ObservacionADMIN) (LecturaPropiosAuditadaADMIN, error) {
	f.llamadas++
	return f.resultado, nil
}
func (f *fuenteActivosPrueba) SeleccionarPerfilAuditadoADMIN(context.Context, adminperfiles.ObservacionADMIN, string, uint64) (SeleccionAuditadaADMIN, error) {
	return SeleccionAuditadaADMIN{}, ports.ErrAutoridadAdministracionPerfilesNoDisponible
}
func observacionActivosPrueba(ahora time.Time) adminperfiles.ObservacionADMIN {
	return adminperfiles.ObservacionADMIN{Entorno: "desarrollo", Host: "admin.invalid", Audiencia: "audiencia:admin", CertificadoSHA256: strings.Repeat("a", 64), CASHA256: strings.Repeat("b", 64), AutenticacionVerificadaEn: ahora.Add(-time.Minute), RevocacionVerificadaEn: ahora, CRLVigenteHasta: ahora.Add(time.Hour), CertificadoVigenteHasta: ahora.Add(time.Hour)}
}
func propiosActivosPrueba() adminperfiles.PerfilesPropios {
	return adminperfiles.PerfilesPropios{Perfiles: []adminperfiles.PerfilPropio{{PerfilRef: "prf_" + strings.Repeat("a", 22), RolVersionRef: "rol:administracion_perfiles:v1", ClaveI18N: "administracion_aplicacion", CategoriaADMIN: "aplicacion"}}}
}
func TestGateActivosExigeLecturaAuditadaSinPerfilActivo(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	o := observacionActivosPrueba(ahora)
	if !o.Valida(ahora) {
		t.Fatal("observacion sintetica invalida")
	}
	for _, caso := range []string{"ausente", "sin_recibo", "sin_perfiles", "sin_categoria", "host", "audiencia", "caducada", "confirmada"} {
		t.Run(caso, func(t *testing.T) {
			obs := o
			f := &fuenteActivosPrueba{resultado: LecturaPropiosAuditadaADMIN{Propios: propiosActivosPrueba(), AuditoriaComunRef: "auditoria:prueba:lectura"}}
			a := &auditorActivosPrueba{}
			h := handlerPerfilesADMIN{observador: observadorActivosPrueba{o: obs}, fuenteSeleccion: f, auditor: a, host: hostAdmin{nombre: "admin.invalid", autoridad: "admin.invalid"}, audienciaSelector: "audiencia:admin", reloj: relojActivosPrueba{ahora}}
			esperado := http.StatusServiceUnavailable
			switch caso {
			case "ausente":
				h.fuenteSeleccion = nil
			case "sin_recibo":
				f.resultado.AuditoriaComunRef = ""
			case "sin_perfiles":
				f.resultado.Propios.Perfiles = nil
				esperado = http.StatusForbidden
			case "sin_categoria":
				f.resultado.Propios.Perfiles[0].CategoriaADMIN = ""
				esperado = http.StatusForbidden
			case "host":
				obs.Host = "otro.invalid"
				h.observador = observadorActivosPrueba{o: obs}
				esperado = http.StatusUnauthorized
			case "audiencia":
				h.audienciaSelector = "audiencia:otra"
				esperado = http.StatusUnauthorized
			case "caducada":
				obs.CRLVigenteHasta = ahora
				h.observador = observadorActivosPrueba{o: obs}
				esperado = http.StatusUnauthorized
			case "confirmada":
				esperado = http.StatusOK
			}
			r := httptest.NewRequest(http.MethodGet, "https://admin.invalid/admin/usuarios/", nil)
			if actual := h.estadoActivosSelector(context.Background(), r); actual != esperado {
				t.Fatalf("estado %d, esperado %d", actual, esperado)
			}
			if (caso == "host" || caso == "audiencia" || caso == "caducada") && f.llamadas != 0 {
				t.Fatal("consulto una fuente tras observacion incompatible")
			}
			if esperado != http.StatusOK && a.llamadas != 1 {
				t.Fatal("denegacion sin auditor")
			}
		})
	}
}
func TestListaActivosIncluyeGrafoUsuariosSelectorYExcluyePruebas(t *testing.T) {
	activos := activosMapaPrueba()
	h, err := montarActivosPerfiles(http.NotFoundHandler(), DependenciasPerfiles{Activos: activos, ContextoConexion: func(ctx context.Context, _ net.Conn) context.Context { return ctx }}, hostAdmin{nombre: "admin.invalid", autoridad: "admin.invalid"})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{"/admin/usuarios/metadatos.js", "/admin/usuarios/propuestas.js", "/admin/usuarios/propuestas-contratos.js", "/favicon.svg", "/administracion-perfiles/", adminselector.RutaPropios, adminselector.RutaSeleccion} {
		if !h.atiende(r) {
			t.Fatalf("ruta propia ausente %s", r)
		}
	}
	for _, r := range []string{"/admin/usuarios/fixture.test.json", "/admin/usuarios/usuarios.test.mjs", "/portal-empleado/index.html", "/admin/modulos/"} {
		if h.atiende(r) {
			t.Fatalf("ruta ajena expuesta %s", r)
		}
	}
}
func TestFuentesAusentesNoFabricanLecturaNiSeleccion(t *testing.T) {
	ctx := context.Background()
	f := lecturasNoDisponibles{}
	roles, err := f.ListarRoles(ctx, domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{})
	if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || roles.Roles != nil {
		t.Fatal("roles fabricados")
	}
	propios, err := (seleccionAuditadaADMIN{}).ListarPropiosADMIN(ctx, adminperfiles.ObservacionADMIN{})
	if !errors.Is(err, ports.ErrAutoridadAdministracionPerfilesNoDisponible) || len(propios.Perfiles) != 0 {
		t.Fatal("lista propia fabricada")
	}
}

func activosMapaPrueba() fstest.MapFS {
	activos := fstest.MapFS{}
	rutas := []string{"admin/usuarios/index.html", "admin/usuarios/entry.js", "admin/usuarios/vista.js", "admin/usuarios/render.js", "admin/usuarios/contratos.js", "admin/usuarios/metadatos.js", "admin/usuarios/cliente.js", "admin/usuarios/lecturas-http.js", "admin/usuarios/propuestas.js", "admin/usuarios/propuestas-contratos.js", "admin/usuarios/usuarios.css", "administracion-perfiles/selector-perfil.js", "administracion-perfiles/selector-perfil.css", "favicon.svg", "comun/idioma.js", "comun/textos.js", "comun/tema-vec.css", "portal-empleado/portal.css", "portal-empleado/portal-componentes.css", "portal-empleado/portal-flujos.css", "portal-empleado/portal-patrones.css", "textos/es/admin-usuarios.json", "textos/es/admin-selector.json"}
	for _, r := range rutas {
		activos[r] = &fstest.MapFile{Data: []byte("material publico sintetico")}
	}
	activos["textos/idiomas.json"] = &fstest.MapFile{Data: []byte(`{"idiomas":[{"codigo":"es"}]}`)}
	return activos
}

func sesionActivosPrueba(t *testing.T) api.SesionConfiable {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	persona := "per_" + strings.Repeat("a", 22)
	perfil := "prf_" + strings.Repeat("b", 22)
	resultado, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		ahora, persona, perfil, domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	actor := resultado.Contexto
	rol := domain.VersionRol{
		RolID: "administracion_perfiles", Version: 1, Nombre: "Administrador de aplicación",
		Estado:       domain.EstadoVersionRolPublicada,
		Concesiones:  []domain.ConcesionRol{{Accion: "bolsa.expediente.leer", ModuloID: "bolsa", TipoRecurso: "expediente", Finalidades: []string{"gestion_bolsa"}, GarantiaMinima: domain.AuthAssuranceSubstantial}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-24 * time.Hour),
	}
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := domain.InstantaneaAutorizacion{
		AsignacionPerfil: domain.AsignacionPerfil{
			AsignacionID: "asig-admin", Version: 1, PerfilActivoRef: perfil, PrincipalID: persona,
			VersionRolRef: rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
			Ambitos:      []domain.AmbitoPerfil{{Clave: "unidad", Valores: []string{"seleccion"}}},
			VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
			EmitidaPor: "responsable-seguridad", EmitidaEn: ahora.Add(-2 * time.Hour),
		},
		VersionRol: rol,
		ControlVigenciaVersionRol: domain.ControlVigenciaVersionRol{
			VersionRolRef: rol.Referencia(), Revision: 1,
			Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn,
		},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella,
	}
	if err := snapshot.Validar(); err != nil {
		t.Fatal(err)
	}
	return api.SesionConfiable{Actor: actor,
		Evidencia:               domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: vinculo},
		InstantaneaAutorizacion: snapshot,
		CorrelacionRef:          "correlacion_" + strings.Repeat("e", 32)}
}

type sesionActivosStub struct{ resultado api.SesionConfiable }

func (s sesionActivosStub) ResolverSesionADMIN(context.Context, *http.Request) (api.SesionConfiable, error) {
	return s.resultado, nil
}

type capacidadesActivosStub struct {
	lecturasNoDisponibles
	datos api.Capacidades
}

func (c capacidadesActivosStub) Capacidades(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (api.Capacidades, error) {
	return c.datos, nil
}
func TestMontajeSinObservadorSoloAdmiteVersionProtocolaria1(t *testing.T) {
	sesion := sesionActivosPrueba(t)
	for _, version := range []string{"1", "v1", "", "2"} {
		t.Run(version, func(t *testing.T) {
			deps := DependenciasPerfiles{Activos: activosMapaPrueba(), ContextoConexion: func(ctx context.Context, _ net.Conn) context.Context { return ctx }, Sesiones: sesionActivosStub{sesion}, Lecturas: capacidadesActivosStub{datos: api.Capacidades{Version: version, ActorPersonaRef: sesion.Actor.PersonaRef, Acciones: []string{"consultar"}}}}
			h, err := montarActivosPerfiles(http.NotFoundHandler(), deps, hostAdmin{nombre: "admin.invalid", autoridad: "admin.invalid"})
			if err != nil || h.observador != nil {
				t.Fatal("constructor de lectura alterado", err)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "https://admin.invalid/admin/usuarios/", nil))
			esperado := http.StatusServiceUnavailable
			if version == "1" {
				esperado = http.StatusOK
			}
			if w.Code != esperado {
				t.Fatalf("version %s estado %d esperado %d", version, w.Code, esperado)
			}
		})
	}
}
