package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	vd "vec-diputacion-granada/internal/vec/domain"
)

func TestFicheroExactoValidadoSinReescribirNiExponerDatos(t *testing.T) {
	b, h := materialPrueba(t)
	ruta := filepath.Join(t.TempDir(), "material-actor-privado.json")
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if codigo := ejecutar([]string{ruta, h}, &salida); codigo != 0 {
		t.Fatalf("código=%d", codigo)
	}
	var r resumen
	if json.Unmarshal(salida.Bytes(), &r) != nil || r.SHA256 != h || r.Operacion != "publicar" ||
		r.CatalogoID != "ct.plan.firma.sintetico" || r.Version != 1 || r.Revision != 1 || r.Estado != "material_validado_sin_autorizacion" {
		t.Fatal("resumen inválido")
	}
	for _, privado := range []string{"actor:publicador:001", "clave-sintetica-0001", ruta, "catalogo_canonico_base64"} {
		if strings.Contains(salida.String(), privado) {
			t.Fatal("resumen expone material privado")
		}
	}
	despues, err := os.ReadFile(ruta)
	if err != nil || !bytes.Equal(b, despues) {
		t.Fatal("fichero reescrito")
	}
}

func TestRechazaSHAJSONDuplicadoYMaterialCruzado(t *testing.T) {
	b, h := materialPrueba(t)
	var falso material
	if json.Unmarshal(b, &falso) != nil {
		t.Fatal("fixture inválida")
	}
	falso.CatalogoSHA = strings.Repeat("0", 64)
	subSHA, _ := json.Marshal(falso)
	var duplicado material
	if json.Unmarshal(b, &duplicado) != nil {
		t.Fatal("fixture inválida")
	}
	canon, _ := base64.StdEncoding.DecodeString(duplicado.CatalogoBase64)
	canon = bytes.Replace(canon, []byte(`"id":`), []byte(`"id":"ct.plan.firma.ajeno","id":`), 1)
	duplicado.CatalogoBase64 = base64.StdEncoding.EncodeToString(canon)
	duplicado.CatalogoSHA = huella(canon)
	canonDuplicado, _ := json.Marshal(duplicado)
	casos := map[string]struct {
		b   []byte
		sha string
	}{
		"sha":                  {b, strings.Repeat("0", 64)},
		"bytes":                {append(bytes.Clone(b), ' '), h},
		"duplicado":            {bytes.Replace(b, []byte(`"operacion":`), []byte(`"operacion":"crear","operacion":`), 1), ""},
		"catalogo ajeno":       {bytes.Replace(b, []byte(`"catalogo_id":"ct.plan.firma.sintetico"`), []byte(`"catalogo_id":"ct.plan.firma.ajeno"`), 1), ""},
		"subSHA":               {subSHA, ""},
		"canon duplicado":      {canonDuplicado, ""},
		"raiz con mayúscula":   {bytes.Replace(b, []byte(`"operacion":`), []byte(`"Operacion":`), 1), ""},
		"revisión nula":        {bytes.Replace(b, []byte(`"revision_esperada":1`), []byte(`"revision_esperada":null`), 1), ""},
		"traza con mayúscula":  {mutarBloquePrueba(t, b, true, `"actor_id":`, `"Actor_ID":`), ""},
		"traza nula":           {mutarBloquePrueba(t, b, true, `"actor_profile":"perfil:sintetico:001"`, `"actor_profile":null`), ""},
		"evento con mayúscula": {mutarBloquePrueba(t, b, false, `"actor_id":`, `"Actor_ID":`), ""},
		"evento nulo":          {mutarBloquePrueba(t, b, false, `"actor_id":"actor:publicador:001"`, `"actor_id":null`), ""},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			if caso.sha == "" {
				caso.sha = huella(caso.b)
			}
			ruta := filepath.Join(t.TempDir(), "material.json")
			if err := os.WriteFile(ruta, caso.b, 0600); err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if codigo := ejecutar([]string{ruta, caso.sha}, &salida); codigo != 1 || salida.Len() != 0 {
				t.Fatalf("aceptado: %d", codigo)
			}
		})
	}
}

func TestRechazaFicheroMayorQueCuatroMiB(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "material-grande.json")
	if err := os.WriteFile(ruta, bytes.Repeat([]byte{'x'}, maxMaterial+1), 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if codigo := ejecutar([]string{ruta, strings.Repeat("a", 64)}, &salida); codigo != 1 || salida.Len() != 0 {
		t.Fatalf("límite ignorado: %d", codigo)
	}
}

func TestRechazoRegistraSoloEtapaSinRutaNiActor(t *testing.T) {
	anterior := slog.Default()
	var registro bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, nil)))
	defer slog.SetDefault(anterior)
	ruta := filepath.Join(t.TempDir(), "actor-privado-no-exponer.json")
	var salida bytes.Buffer
	if codigo := ejecutar([]string{ruta, strings.Repeat("a", 64)}, &salida); codigo != 1 || salida.Len() != 0 {
		t.Fatal("rechazo no observable")
	}
	if !strings.Contains(registro.String(), "etapa=entrada") || strings.Contains(registro.String(), ruta) || strings.Contains(registro.String(), "actor-privado") {
		t.Fatal("registro de error expone la ruta")
	}
}

func mutarBloquePrueba(t *testing.T, b []byte, traza bool, anterior, nuevo string) []byte {
	t.Helper()
	var m material
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	s := m.EventoBase64
	if traza {
		s = m.TrazaBase64
	}
	contenido, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	cambiado := bytes.Replace(contenido, []byte(anterior), []byte(nuevo), 1)
	if bytes.Equal(cambiado, contenido) {
		t.Fatal("mutación de prueba ausente")
	}
	if traza {
		m.TrazaBase64 = base64.StdEncoding.EncodeToString(cambiado)
		m.TrazaSHA = huella(cambiado)
	} else {
		m.EventoBase64 = base64.StdEncoding.EncodeToString(cambiado)
		m.EventoSHA = huella(cambiado)
	}
	salida, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return salida
}

func materialPrueba(t *testing.T) ([]byte, string) {
	t.Helper()
	desde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	creado := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	a := map[string]string{
		"esquema": "ct.plan-competencia-firma.v2", "circuito_ref": "catalogo:circuito:sintetico", "circuito_version": "1", "circuito_sha256": strings.Repeat("a", 64),
		"documento": "informe_definitivo", "paso_ref": "paso:direccion", "paso_orden": "1", "perfil_esperado_ref": "perfil:firma:direccion", "rol_id": "ct_direccion_rrhh", "cargo_ref": "cargo:direccion",
		"organizacion_ref": "organizacion:central", "unidad_ref": "unidad:rrhh", "accion_competencial": "contratacion_temporal.documento.firma_vec.registrar", "finalidad": "gestionar_contratacion_temporal",
		"tipo_recurso": "documento_contratacion_temporal", "esquema_contexto": "vec.contexto.firma.ct.v1", "mapeo_version": "1", "mapeo_fuente_ref": "fuente:plan:ct",
	}
	c := vd.CatalogoConfigurable{ID: "ct.plan.firma.sintetico", Version: 1, Revision: 1, ModuloID: "contratacion_temporal", Nombre: "Plan de firma sintético", FuenteRef: "fuente:rrhh:sintetica", MotivoCreacion: "Ejercicio sintético", Estado: vd.EstadoCatalogoBorrador, CreadoPor: "actor:creador:001", CreadoEn: creado, Entradas: []vd.EntradaCatalogoConfigurable{{Clave: "paso_1", Etiqueta: "Paso 1", Orden: 1, VigenteDesde: desde, Atributos: a}}}
	p, err := c.Publicar("actor:publicador:001", "aprobacion:sintetica:001", "Revisión sintética", creado.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	sh := huella(canon)
	ref := p.Referencia()
	accion := vd.AccionCatalogoPublicado
	traza := vd.AuditEntry{ActorID: p.PublicadoPor, ActorProfile: "perfil:sintetico:001", AuthorizationRef: "decision:sintetica:001", Purpose: "gestionar_contratacion_temporal", Action: accion, ModuleID: p.ModuloID, SubjectRef: ref, ObjectVersion: p.Version, RuleRef: p.AprobacionRef, Reason: p.MotivoPublicacion, Result: "correcto", BeforeHash: strings.Repeat("b", 64), AfterHash: sh, CorrelationRef: "correlacion:sintetica:001", OccurredAt: p.PublicadoEn,
		Metadata: map[string]string{"catalogo_id": p.ID, "catalogo_version": "1", "estado": "publicado", "revision": "1"}}
	evento := vd.Event{Type: accion, ModuleID: p.ModuloID, SubjectRef: ref, ActorID: p.PublicadoPor, OccurredAt: p.PublicadoEn, Payload: map[string]string{"catalogo_id": p.ID, "catalogo_version": "1", "catalogo_revision": "1", "estado": "publicado", "huella_sha256": sh}}
	btraza, _ := json.Marshal(traza)
	bevento, _ := json.Marshal(evento)
	m := material{Esquema: "vec.catalogos.plan-firma.gobierno.v1", Operacion: "publicar", CatalogoID: p.ID, Version: 1, RevisionEsperada: 1, HuellaEsperada: &traza.BeforeHash, ClaveOperacion: "clave-sintetica-0001", CatalogoBase64: base64.StdEncoding.EncodeToString(canon), CatalogoSHA: sh, TrazaBase64: base64.StdEncoding.EncodeToString(btraza), TrazaSHA: huella(btraza), EventoBase64: base64.StdEncoding.EncodeToString(bevento), EventoSHA: huella(bevento)}
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return b, huella(b)
}
