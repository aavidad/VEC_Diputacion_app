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
			h := handlerPerfilesADMIN{observador: observadorActivosPrueba{o: obs}, fuenteSeleccion: f, auditor: a, origen: "https://admin.invalid", audienciaSelector: "audiencia:admin", reloj: relojActivosPrueba{ahora}}
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
	activos := fstest.MapFS{}
	rutas := []string{"admin/usuarios/index.html", "admin/usuarios/entry.js", "admin/usuarios/vista.js", "admin/usuarios/render.js", "admin/usuarios/contratos.js", "admin/usuarios/cliente.js", "admin/usuarios/lecturas-http.js", "admin/usuarios/propuestas.js", "admin/usuarios/propuestas-contratos.js", "admin/usuarios/usuarios.css", "administracion-perfiles/selector-perfil.js", "administracion-perfiles/selector-perfil.css", "favicon.svg", "comun/idioma.js", "comun/textos.js", "comun/tema-vec.css", "portal-empleado/portal.css", "portal-empleado/portal-componentes.css", "portal-empleado/portal-flujos.css", "portal-empleado/portal-patrones.css", "textos/es/admin-usuarios.json", "textos/es/admin-selector.json"}
	for _, r := range rutas {
		activos[r] = &fstest.MapFile{Data: []byte("material publico sintetico")}
	}
	activos["textos/idiomas.json"] = &fstest.MapFile{Data: []byte(`{"idiomas":[{"codigo":"es"}]}`)}
	h, err := montarActivosPerfiles(http.NotFoundHandler(), DependenciasPerfiles{Activos: activos, ContextoConexion: func(ctx context.Context, _ net.Conn) context.Context { return ctx }}, "https://admin.invalid")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{"/admin/usuarios/propuestas.js", "/admin/usuarios/propuestas-contratos.js", "/favicon.svg", "/administracion-perfiles/", adminselector.RutaPropios, adminselector.RutaSeleccion} {
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
