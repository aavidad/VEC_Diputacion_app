package application_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/catalogo"
	seleccioninterno "vec-diputacion-granada/internal/modules/seleccion/adapters/httpinterno"
	seleccionpersonal "vec-diputacion-granada/internal/modules/seleccion/adapters/httppersonal"
	"vec-diputacion-granada/internal/modules/seleccion/adapters/nodisponible"
	seleccionpg "vec-diputacion-granada/internal/modules/seleccion/adapters/postgres"
	"vec-diputacion-granada/internal/modules/seleccion/adapters/referencias"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type preparadorFijo struct{ orden application.Orden }

func (p preparadorFijo) PrepararOrdenPersona(*http.Request) (application.Orden, error) {
	return p.orden, nil
}
func (p preparadorFijo) PrepararOrdenRRHH(*http.Request) (application.Orden, error) {
	return p.orden, nil
}

// Recorrido HTTP → aplicación → PostgreSQL con material V3 real (PDP y
// criptografía de prueba) y los dobles de las fachadas AD3-89/90: la persona
// guarda un borrador parcial y otro completo, lo presenta y ve su justificante;
// RRHH lista la solicitud minimizada y abre su ficha. Requiere
// VEC_SELECCION_PG_DSN (deploy/postgresql/seleccion/probar_pg18.sh).
func TestRecorridoHTTPConPostgreSQL(t *testing.T) {
	dsn := os.Getenv("VEC_SELECCION_PG_DSN")
	if dsn == "" {
		t.Skip("requiere VEC_SELECCION_PG_DSN (PostgreSQL 18 desechable con Selección 000001 y dobles AD3)")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := seleccionpg.NuevoRepositorio(pool)
	if err != nil {
		t.Fatal(err)
	}
	fuente, err := catalogo.NuevoFichero("../../../../data/demo/reglas/seleccion_convocatorias.ejemplo.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	convocatorias, _ := fuente.Convocatorias(ctx)
	c := convocatorias[0]
	sufijo := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000"), ".", "")
	c.Ref = "ensayo-http-" + sufijo
	c.AbreEn, c.CierraEn = time.Now().UTC().Add(-time.Hour).Truncate(time.Second), time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	if _, _, err := repo.PublicarConvocatoria(ctx, c); err != nil {
		t.Fatal(err)
	}

	ordenPersona, emisorPersona, reloj := application.EntornoV3Prueba(t, dominiovec.SuperficieAutenticacionExternaPersonalV1)
	externos := application.ServiciosExternos{Firma: nodisponible.Servicios{}, Registro: nodisponible.Servicios{}, Tasas: nodisponible.Servicios{}, Notificacion: nodisponible.Servicios{}}
	propias, err := application.NuevoServicioSolicitudesPropias(repo, repo, emisorPersona, application.ProtectorPrueba(), referencias.Aleatorio{}, externos, reloj)
	if err != nil {
		t.Fatal(err)
	}
	persona, err := seleccionpersonal.Nuevo(preparadorFijo{ordenPersona}, propias)
	if err != nil {
		t.Fatal(err)
	}
	pedir := func(h http.Handler, metodo, ruta, cuerpo, clave string, estado int) map[string]any {
		t.Helper()
		var r *http.Request
		if cuerpo == "" {
			r = httptest.NewRequest(metodo, ruta, nil)
		} else {
			r = httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Accept", "application/json")
		}
		if clave != "" {
			// Las claves son de la persona: se distinguen por ejecución.
			r.Header.Set("Idempotency-Key", clave+"-"+sufijo)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != estado {
			t.Fatalf("%s %s: %d %s (se esperaba %d)", metodo, ruta, w.Code, w.Body.String(), estado)
		}
		var v map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &v)
		return v
	}
	datos := func(v map[string]any) map[string]any { d, _ := v["data"].(map[string]any); return d }

	parcial := `{"convocatoria_ref":"` + c.Ref + `","version_esperada":0,"datos":{"nombre":"Antonio"},"requisitos":[{"clave":"nacionalidad","estado":"cumple"}],"meritos":[]}`
	r1 := datos(pedir(persona, http.MethodPut, seleccionpersonal.RutaBorrador, parcial, "clave-http-0001", 201))
	if r1["datos_completos"] != false || r1["version"] != float64(1) {
		t.Fatalf("borrador parcial: %v", r1)
	}
	pedir(persona, http.MethodPut, seleccionpersonal.RutaBorrador, parcial, "clave-http-0001", 200)
	pedir(persona, http.MethodPut, seleccionpersonal.RutaBorrador, strings.Replace(parcial, "Antonio", "Antonia", 1), "clave-http-0001", 409)
	sol := r1["solicitud_ref"].(string)
	pedir(persona, http.MethodPost, seleccionpersonal.RutaPresentacion, `{"solicitud_ref":"`+sol+`","version_esperada":1,"declaracion_responsable":true}`, "clave-http-pres-0", 422)
	completo := `{"convocatoria_ref":"` + c.Ref + `","version_esperada":1,"turno":"libre","datos":{"nombre":"Antonio","apellidos":"Reyes Álvarez",` +
		`"documento_identidad":"12345678Z","fecha_nacimiento":"1988-04-12","nacionalidad":"española","correo":"antonio.reyes@example.org","telefono":"600000000",` +
		`"direccion":{"via":"C/ Real 1","codigo_postal":"18001","municipio":"Granada","provincia":"Granada"}},` +
		`"requisitos":[{"clave":"nacionalidad","estado":"cumple"},{"clave":"titulacion","estado":"pendiente"}],` +
		`"meritos":[{"clave_grupo":"experiencia","clave_merito":"meses_administracion","descripcion":"Peón de mantenimiento","cantidad":"14"}]}`
	r2 := datos(pedir(persona, http.MethodPut, seleccionpersonal.RutaBorrador, completo, "clave-http-0002", 201))
	if r2["datos_completos"] != true || r2["puntuacion_autobaremo"] != "1.4" {
		t.Fatalf("borrador completo: %v", r2)
	}
	pedir(persona, http.MethodPut, seleccionpersonal.RutaBorrador, completo, "clave-http-0003", 409)
	b := datos(pedir(persona, http.MethodGet, seleccionpersonal.RutaBorrador+"?convocatoria_ref="+c.Ref, "", "", 200))
	if d, _ := b["datos"].(map[string]any); d["documento_identidad"] != "12345678Z" {
		t.Fatalf("lectura del borrador: %v", b)
	}
	presentar := `{"solicitud_ref":"` + sol + `","version_esperada":2,"declaracion_responsable":true}`
	recibo := datos(pedir(persona, http.MethodPost, seleccionpersonal.RutaPresentacion, presentar, "clave-http-pres-1", 201))
	repetido := datos(pedir(persona, http.MethodPost, seleccionpersonal.RutaPresentacion, presentar, "clave-http-pres-1", 200))
	if recibo["numero_justificante"] != repetido["numero_justificante"] || recibo["recibo_ref"] != repetido["recibo_ref"] ||
		!strings.Contains(recibo["numero_justificante"].(string), "/SOL-") {
		t.Fatalf("presentación y repetición: %v / %v", recibo, repetido)
	}
	if s, _ := recibo["servicios"].(map[string]any); s["firma"] != "no_disponible" || s["registro_sede"] != "no_disponible" {
		t.Fatalf("servicios externos: %v", recibo["servicios"])
	}
	pedir(persona, http.MethodPost, seleccionpersonal.RutaPresentacion, presentar, "clave-http-pres-2", 409)

	ordenRRHH, emisorRRHH, relojRRHH := application.EntornoV3Prueba(t, dominiovec.SuperficieAutenticacionInternaCorporativaV1)
	consulta, err := application.NuevoServicioConsultaRRHH(repo, repo, emisorRRHH, application.ProtectorPrueba(), relojRRHH)
	if err != nil {
		t.Fatal(err)
	}
	rrhh, err := seleccioninterno.Nuevo(preparadorFijo{ordenRRHH}, consulta)
	if err != nil {
		t.Fatal(err)
	}
	lista := datos(pedir(rrhh, http.MethodPost, seleccioninterno.RutaConsultas, `{"convocatoria_ref":"`+c.Ref+`","limite":10}`, "", 200))
	filas, _ := lista["solicitudes"].([]any)
	if len(filas) != 1 || filas[0].(map[string]any)["nombre_visible"] != "Reyes Álvarez, Antonio" || filas[0].(map[string]any)["documento_parcial"] != "***5678*" ||
		strings.Contains(lista["solicitudes"].([]any)[0].(map[string]any)["nombre_visible"].(string), "12345678Z") {
		t.Fatalf("listado de RRHH: %v", lista)
	}
	ficha := datos(pedir(rrhh, http.MethodPost, seleccioninterno.RutaDetalleConsultas, `{"solicitud_ref":"`+sol+`"}`, "", 200))
	meritos, _ := ficha["meritos"].([]any)
	if len(meritos) != 1 || meritos[0].(map[string]any)["titulo"] == "" || meritos[0].(map[string]any)["puntos"] != "1.4" || len(ficha["historia"].([]any)) != 3 {
		t.Fatalf("ficha de RRHH: %v", ficha)
	}
}
