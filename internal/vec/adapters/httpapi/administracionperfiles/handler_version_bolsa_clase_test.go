package administracionperfiles

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type lineaClaseVersionBolsa struct {
	Msg         string `json:"msg"`
	Correlacion string `json:"vec.correlacion"`
	Ruta        string `json:"ruta"`
	Clase       string `json:"clase"`
}

func registroClasePrueba() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewJSONHandler(&buf, nil)), &buf
}

func lineasClase(t *testing.T, buf *bytes.Buffer) []lineaClaseVersionBolsa {
	t.Helper()
	var salida []lineaClaseVersionBolsa
	for _, l := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if l == "" {
			continue
		}
		var x lineaClaseVersionBolsa
		if err := json.Unmarshal([]byte(l), &x); err != nil {
			t.Fatal(err)
		}
		salida = append(salida, x)
	}
	return salida
}

func TestVersionBolsaRegistraClaseCerradaEnCada503(t *testing.T) {
	sesion := sesionAplicacionNominalPrueba(t)
	ref := "propuesta_admin:" + strings.Repeat("a", 32)
	secreto := errors.New("SECRETO dsn=postgres://x")
	casos := []struct {
		nombre, clase, codigo string
		err                   error
		estado, auditorias    int
		falloAuditor          bool
	}{
		{"rol_sin_categoria", "rol_administrable_respuesta", "servicio_no_disponible",
			ports.ConClaseVersionBolsa("rol_administrable_respuesta", errors.Join(ports.ErrAutoridadAdministracionPerfilesNoDisponible, secreto)),
			http.StatusServiceUnavailable, 1, false},
		{"sin_concesion", "v3_sin_concesion", "servicio_no_disponible",
			ports.ConClaseVersionBolsa("v3_sin_concesion", ports.ErrAutoridadAdministracionPerfilesNoDisponible),
			http.StatusServiceUnavailable, 1, false},
		{"sin_clase_usa_etapa", "propuesta_servicio", "servicio_no_disponible",
			ports.ErrAutoridadAdministracionPerfilesNoDisponible, http.StatusServiceUnavailable, 1, false},
		{"intento_sql_error", "sql_intento_error", "servicio_no_disponible",
			errors.Join(ports.ErrGobiernoRolIntentoAuditado,
				ports.ConClaseVersionBolsa("sql_intento_error", ports.ErrAutoridadAdministracionPerfilesNoDisponible)),
			http.StatusServiceUnavailable, 0, false},
		// AUT72: el código SQLSTATE llega al registro técnico dentro de la clase.
		{"intento_sql_error_sqlstate", "sql_intento_error_42703", "servicio_no_disponible",
			errors.Join(ports.ErrGobiernoRolIntentoAuditado,
				ports.ConClaseVersionBolsa("sql_intento_error_42703", ports.ErrAutoridadAdministracionPerfilesNoDisponible)),
			http.StatusServiceUnavailable, 0, false},
		{"intento_denegado_sqlstate", "", "acceso_denegado",
			errors.Join(ports.ErrGobiernoRolIntentoAuditado,
				ports.ConClaseVersionBolsa("sql_intento_denegado_42501", domain.ErrAutorizacionDenegada)),
			http.StatusForbidden, 0, false},
		{"contexto", "contexto", "servicio_no_disponible", context.DeadlineExceeded,
			http.StatusServiceUnavailable, 1, false},
		{"auditoria_caida", "denegacion_auditoria", "servicio_no_disponible",
			ports.ConClaseVersionBolsa("v3_emision", ports.ErrAutoridadAdministracionPerfilesNoDisponible),
			http.StatusServiceUnavailable, 1, true},
		{"denegado", "", "acceso_denegado", domain.ErrAutorizacionDenegada, http.StatusForbidden, 1, false},
		{"intento_denegado", "", "acceso_denegado",
			errors.Join(ports.ErrGobiernoRolIntentoAuditado, domain.ErrAutorizacionDenegada), http.StatusForbidden, 0, false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			auditor := &auditorPrueba{}
			if caso.falloAuditor {
				auditor.err = errors.New("auditoria_caida")
			}
			registro, buf := registroClasePrueba()
			h := &Handler{auditor: auditor, registroVersionBolsa: registro}
			ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			correlacion, _ := ports.CorrelacionIncidenciasPeticion(ctx)
			peticion := httptest.NewRequest(http.MethodPost, "https://admin.example.test"+RutaVersionarRolBolsaProponer, nil).WithContext(ctx)
			w := httptest.NewRecorder()
			h.responderErrorVersionBolsa(w, peticion, sesion, caso.err, ref, "propuesta_servicio")
			if w.Code != caso.estado || auditor.llamadas != caso.auditorias ||
				!strings.Contains(w.Body.String(), `"codigo":"`+caso.codigo+`"`) {
				t.Fatalf("estado=%d auditorias=%d cuerpo=%s", w.Code, auditor.llamadas, w.Body.String())
			}
			lineas := lineasClase(t, buf)
			if caso.clase == "" {
				if len(lineas) != 0 {
					t.Fatalf("una denegación 403 dejó clase de 503: %+v", lineas)
				}
				return
			}
			if len(lineas) != 1 || lineas[0].Clase != caso.clase || lineas[0].Ruta != "propuestas" ||
				lineas[0].Correlacion != correlacion || lineas[0].Msg != "vec_admin_version_bolsa_no_disponible" {
				t.Fatalf("registro inesperado: %+v", lineas)
			}
			if strings.Contains(buf.String(), "SECRETO") || strings.Contains(w.Body.String(), "SECRETO") ||
				strings.Contains(buf.String(), sesion.Actor.PersonaRef) {
				t.Fatal("la causa o la identidad llegaron al registro o a la respuesta")
			}
		})
	}
}

func TestVersionBolsaClaseFueraDeListaSeRegistraSinClase(t *testing.T) {
	registro, buf := registroClasePrueba()
	h := &Handler{registroVersionBolsa: registro}
	h.registrarClaseVersionBolsa(httptest.NewRequest(http.MethodPost,
		"https://admin.example.test"+RutaVersionarRolBolsaCerrar, nil), "texto libre con datos")
	lineas := lineasClase(t, buf)
	if len(lineas) != 1 || lineas[0].Clase != "sin_clase" || lineas[0].Ruta != "cierres" {
		t.Fatalf("clase libre registrada: %+v", lineas)
	}
}

type fuenteVersionBolsaClasePrueba struct{ err error }

func (f fuenteVersionBolsaClasePrueba) ObtenerCatalogoAccionesAdministracionV1(context.Context,
	string, int, string) (domain.CatalogoAccionesAdministracionV1, error) {
	return domain.CatalogoAccionesAdministracionV1{}, f.err
}

func TestVersionBolsaCatalogoNoDisponibleDejaClaseEnRuta(t *testing.T) {
	for _, caso := range []struct {
		nombre, clase string
		err           error
	}{
		{"paquete", "catalogo_paquete", ports.ConClaseVersionBolsa("catalogo_paquete", ports.ErrAutoridadAdministracionPerfilesNoDisponible)},
		{"sin_clase", "catalogo_consulta", ports.ErrAutoridadAdministracionPerfilesNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			sesion := &sesionPrueba{resultado: sesionAplicacionNominalPrueba(t)}
			auditor := &auditorPrueba{}
			h, err := NuevoHandlerUsuariosMetadatos("https://admin.example.test", sesion, &lecturasPrueba{}, auditor)
			if err != nil {
				t.Fatal(err)
			}
			servicio := &servicioVersionBolsaPrueba{}
			if err := h.ConVersionarRolBolsa(servicio, fuenteVersionBolsaClasePrueba{err: caso.err}, relojFocal{}); err != nil {
				t.Fatal(err)
			}
			registro, buf := registroClasePrueba()
			h.registroVersionBolsa = registro
			cuerpo := `{"operacion_ref":"propuesta_admin:` + strings.Repeat("b", 32) + `","catalogo_ref":"catalogo:bolsa:carga_convoca:b1",` +
				`"catalogo_version":1,"catalogo_huella_sha256":"` + strings.Repeat("c", 64) + `","base_ref":"rol:x:v6",` +
				`"base_huella_sha256":"` + strings.Repeat("d", 64) + `","control_revision":1,"control_huella_sha256":"` +
				strings.Repeat("e", 64) + `","asignaciones":[{"asignacion_ref":"a","huella_sha256":"b","documento":{}}]}`
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticionADMIN(http.MethodPost, RutaVersionarRolBolsaProponer, cuerpo))
			lineas := lineasClase(t, buf)
			if w.Code != http.StatusServiceUnavailable || servicio.llamadas != 0 || auditor.llamadas != 1 ||
				len(lineas) != 1 || lineas[0].Clase != caso.clase {
				t.Fatalf("estado=%d servicio=%d auditor=%d lineas=%+v cuerpo=%s",
					w.Code, servicio.llamadas, auditor.llamadas, lineas, w.Body.String())
			}
		})
	}
}
