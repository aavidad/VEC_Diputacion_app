package plannominal

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func shaPrueba(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// materialGobiernoPrueba arma el material de trece claves igual que el kit.
func materialGobiernoPrueba(t *testing.T, operacion string, catalogo vd.CatalogoConfigurable) []byte {
	t.Helper()
	canon, err := json.Marshal(catalogo)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]any{"esquema": EsquemaMaterialGobiernoPlanFirma, "operacion": operacion,
		"catalogo_id": catalogo.ID, "version": catalogo.Version, "revision_esperada": 1,
		"huella_esperada": strings.Repeat("b", 64), "clave_operacion": "clave-sintetica-0001",
		"catalogo_canonico_base64": base64.StdEncoding.EncodeToString(canon), "catalogo_sha256": shaPrueba(canon),
		"traza_canonica_base64": "e30=", "traza_sha256": shaPrueba([]byte("{}")),
		"evento_canonico_base64": "e30=", "evento_sha256": shaPrueba([]byte("{}"))}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func catalogoPublicadoPrueba(t *testing.T) vd.CatalogoConfigurable {
	t.Helper()
	creado := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	c := vd.CatalogoConfigurable{ID: "ct.plan.firma.sintetico", Version: 1, Revision: 1, ModuloID: "contratacion_temporal",
		Nombre: "Plan de firma sintético", FuenteRef: "fuente:rrhh:sintetica", MotivoCreacion: "Ejercicio sintético",
		Estado: vd.EstadoCatalogoBorrador, CreadoPor: "actor:creador:001", CreadoEn: creado,
		Entradas: []vd.EntradaCatalogoConfigurable{{Clave: "paso_1", Etiqueta: "Paso 1", Orden: 1, VigenteDesde: creado}}}
	p, err := c.Publicar("actor:publicador:001", "aprobacion:sintetica:001", "Revisión sintética", creado.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

var ambitoPrueba = AmbitoGobiernoPlanFirma{OrganizacionRef: "org_fbcc10ed290261ec40d52db731f395c8",
	UnidadRef: "unidad:sintetica:h10:688c11eaf7ad8c060db295a7c0a22fb0"}

// La huella debe ser exactamente la que calcula AD201 en SQL:
// sha256('{"ambitos":{"organizacion_ref":O,"unidad_ref":U},"atributos":{"estado":E,"material_sha256":M,"revision":R}}').
func TestRecursoGobiernoPlanFirmaCoincideConAD201(t *testing.T) {
	c := catalogoPublicadoPrueba(t)
	material := materialGobiernoPrueba(t, "publicar", c)
	accion, r, err := RecursoGobiernoPlanFirma(material, ambitoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if accion != "vec.catalogos.publicar" || r.Referencia != "ct.plan.firma.sintetico:1" ||
		r.ModuloID != "contratacion_temporal" || r.Tipo != "catalogo_configurable" || len(r.Ambitos) != 2 ||
		r.Ambitos["organizacion_ref"] != ambitoPrueba.OrganizacionRef || r.Ambitos["unidad_ref"] != ambitoPrueba.UnidadRef {
		t.Fatalf("recurso divergente: %s %+v", accion, r)
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	revision := r.Atributos["revision"]
	sql := `{"ambitos":{"organizacion_ref":"` + ambitoPrueba.OrganizacionRef + `","unidad_ref":"` + ambitoPrueba.UnidadRef +
		`"},"atributos":{"estado":"publicado","material_sha256":"` + shaPrueba(material) + `","revision":"` + revision + `"}}`
	if h != shaPrueba([]byte(sql)) || revision == "" {
		t.Fatalf("huella %s distinta de la fórmula SQL de AD201 (%s)", h, sql)
	}
}

// El PDP común sólo concede si la asignación cubre exactamente las dimensiones
// del recurso; sin ámbitos ninguna asignación lo cubre. Ámbitos con otro formato
// no llegan al PDP.
func TestRecursoGobiernoPlanFirmaLoCubreLaAsignacionDelAdministrador(t *testing.T) {
	_, r, err := RecursoGobiernoPlanFirma(materialGobiernoPrueba(t, "publicar", catalogoPublicadoPrueba(t)), ambitoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	a := vd.AsignacionPerfil{AsignacionID: "asignacion-prueba-gobierno", Version: 1, PrincipalID: "per_0123456789abcdefghijkl",
		PerfilActivoRef: "prf_0123456789abcdefghijkl", VersionRolRef: "rol:administracion_perfiles:v7", Estado: vd.EstadoAsignacionPerfilActiva,
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "actor:prueba", EmitidaEn: ahora.Add(-time.Hour),
		Ambitos: []vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{ambitoPrueba.OrganizacionRef}},
			{Clave: "unidad_ref", Valores: []string{ambitoPrueba.UnidadRef}}}}
	if a.Validar() != nil {
		t.Fatalf("asignación de prueba inválida: %v", a.Validar())
	}
	if !a.Cubre(r) {
		t.Fatal("la asignación del administrador no cubre el recurso de gobierno")
	}
	ajena := ambitoPrueba
	ajena.UnidadRef = "unidad:otra"
	_, otro, err := RecursoGobiernoPlanFirma(materialGobiernoPrueba(t, "publicar", catalogoPublicadoPrueba(t)), ajena)
	if err != nil || a.Cubre(otro) {
		t.Fatal("una unidad ajena quedó cubierta")
	}
	for _, mal := range []AmbitoGobiernoPlanFirma{{}, {OrganizacionRef: "org_corta", UnidadRef: ambitoPrueba.UnidadRef},
		{OrganizacionRef: ambitoPrueba.OrganizacionRef, UnidadRef: `u"x`}} {
		if _, _, err := RecursoGobiernoPlanFirma(materialGobiernoPrueba(t, "publicar", catalogoPublicadoPrueba(t)), mal); !errors.Is(err, ct.ErrPlanCompetenciaFirmaV2) {
			t.Fatalf("ámbito no admitido aceptado: %+v", mal)
		}
	}
}

func TestRecursoGobiernoPlanFirmaRechazaMaterialIncoherente(t *testing.T) {
	c := catalogoPublicadoPrueba(t)
	valido := materialGobiernoPrueba(t, "publicar", c)
	var m map[string]any
	if err := json.Unmarshal(valido, &m); err != nil {
		t.Fatal(err)
	}
	cambiar := func(clave string, valor any) []byte {
		copia := map[string]any{}
		for k, v := range m {
			copia[k] = v
		}
		if valor == nil {
			delete(copia, clave)
		} else {
			copia[clave] = valor
		}
		b, _ := json.Marshal(copia)
		return b
	}
	casos := map[string][]byte{
		"vacío":                nil,
		"no JSON":              []byte("no-json"),
		"falta clave":          cambiar("evento_sha256", nil),
		"clave de más":         cambiar("extra", "x"),
		"esquema ajeno":        cambiar("esquema", "otro"),
		"operación ajena":      cambiar("operacion", "consultar"),
		"estado incoherente":   cambiar("operacion", "retirar"),
		"catálogo con comodín": cambiar("catalogo_id", "ct*plan"),
		"versión distinta":     cambiar("version", 3),
		"versión como texto":   cambiar("version", "2"),
		"sha del canon ajeno":  cambiar("catalogo_sha256", strings.Repeat("0", 64)),
		"clave repetida":       []byte(strings.Replace(string(valido), `"esquema":`, `"esquema":"x","esquema":`, 1)),
	}
	for nombre, material := range casos {
		if _, _, err := RecursoGobiernoPlanFirma(material, ambitoPrueba); !errors.Is(err, ct.ErrPlanCompetenciaFirmaV2) {
			t.Errorf("%s: aceptado (%v)", nombre, err)
		}
	}
}

// Los bytes exactos cuentan: reserializar el mismo material cambia la huella.
func TestRecursoGobiernoPlanFirmaLigaLosBytesExactos(t *testing.T) {
	c := catalogoPublicadoPrueba(t)
	material := materialGobiernoPrueba(t, "publicar", c)
	_, a, err := RecursoGobiernoPlanFirma(material, ambitoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	var crudo map[string]any
	_ = json.Unmarshal(material, &crudo)
	sangrado, _ := json.MarshalIndent(crudo, "", " ")
	_, b, err := RecursoGobiernoPlanFirma(sangrado, ambitoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	if a.Atributos["material_sha256"] == b.Atributos["material_sha256"] {
		t.Fatal("dos serializaciones distintas dan la misma huella")
	}
}

// Material con el catálogo canónico sustituido por bytes arbitrarios.
func materialConCanonPrueba(t *testing.T, canon []byte) []byte {
	t.Helper()
	m := map[string]any{"esquema": EsquemaMaterialGobiernoPlanFirma, "operacion": "publicar",
		"catalogo_id": "ct.plan.firma.sintetico", "version": 1, "revision_esperada": 1,
		"huella_esperada": strings.Repeat("b", 64), "clave_operacion": "clave-sintetica-0001",
		"catalogo_canonico_base64": base64.StdEncoding.EncodeToString(canon), "catalogo_sha256": shaPrueba(canon),
		"traza_canonica_base64": "e30=", "traza_sha256": shaPrueba([]byte("{}")),
		"evento_canonico_base64": "e30=", "evento_sha256": shaPrueba([]byte("{}"))}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Casos en que AD177 rechaza y Go debe rechazar también (paridad con jsonb).
func TestRecursoGobiernoPlanFirmaParidadConJSONB(t *testing.T) {
	base := `"id":"ct.plan.firma.sintetico","version":1,"modulo_id":"contratacion_temporal","estado":"publicado"`
	casos := map[string]string{
		"revisión en cadena":     `{` + base + `,"revision":"2"}`,
		"revisión decimal":       `{` + base + `,"revision":2.0}`,
		"revisión con exponente": `{` + base + `,"revision":2e0}`,
		"versión en cadena":      `{"id":"ct.plan.firma.sintetico","version":"1","modulo_id":"contratacion_temporal","estado":"publicado","revision":2}`,
		"texto de sobra":         `{` + base + `,"revision":2} xx`,
		"estado distinto":        `{"id":"ct.plan.firma.sintetico","version":1,"modulo_id":"contratacion_temporal","estado":"borrador","revision":2}`,
		"nulo escapado":          `{` + base + `,"revision":2,"nombre":"a\u0000b"}`,
	}
	for nombre, canon := range casos {
		if _, _, err := RecursoGobiernoPlanFirma(materialConCanonPrueba(t, []byte(canon)), ambitoPrueba); !errors.Is(err, ct.ErrPlanCompetenciaFirmaV2) {
			t.Errorf("%s: aceptado (%v)", nombre, err)
		}
	}
	if _, _, err := RecursoGobiernoPlanFirma(materialConCanonPrueba(t, []byte(`{`+base+`,"revision":2}`)), ambitoPrueba); err != nil {
		t.Fatalf("canon mínimo válido rechazado: %v", err)
	}
	// jsonb distingue mayúsculas: con "estado" y "ESTADO" toma "estado", igual que aquí.
	_, r, err := RecursoGobiernoPlanFirma(materialConCanonPrueba(t, []byte(`{`+base+`,"revision":2,"ESTADO":"borrador"}`)), ambitoPrueba)
	if err != nil || r.Atributos["estado"] != "publicado" {
		t.Fatalf("clave con otras mayúsculas: %v %v", r.Atributos, err)
	}
	if _, _, err := RecursoGobiernoPlanFirma(materialConCanonPrueba(t, []byte{'{', 0xff, '}'}), ambitoPrueba); err == nil {
		t.Fatal("UTF-8 inválido aceptado")
	}
}

func TestRecursoGobiernoPlanFirmaAceptaCadaOperacion(t *testing.T) {
	for operacion, estado := range estadoPorOperacionGobierno {
		canon := `{"id":"ct.plan.firma.sintetico","version":1,"modulo_id":"contratacion_temporal","estado":"` + estado + `","revision":1}`
		var m map[string]any
		_ = json.Unmarshal(materialConCanonPrueba(t, []byte(canon)), &m)
		m["operacion"] = operacion
		b, _ := json.Marshal(m)
		accion, r, err := RecursoGobiernoPlanFirma(b, ambitoPrueba)
		if err != nil || accion != "vec.catalogos."+operacion || r.Atributos["estado"] != estado {
			t.Errorf("%s: %s %v %v", operacion, accion, r.Atributos, err)
		}
	}
}
