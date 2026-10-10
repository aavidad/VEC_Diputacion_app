package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

type fuentePaquetePrueba struct {
	ModuloID      string                                 `json:"modulo_id"`
	Referencia    string                                 `json:"referencia"`
	Version       int                                    `json:"version"`
	HuellaSHA256  string                                 `json:"huella_sha256"`
	EntradasCanon string                                 `json:"entradas_canon"`
	Entradas      []domain.EntradaAccionAdministracionV1 `json:"entradas"`
}

type paqueteAdmisionPrueba struct {
	Esquema    string                                   `json:"esquema"`
	Referencia string                                   `json:"referencia"`
	Version    int                                      `json:"version"`
	Fuentes    []fuentePaquetePrueba                    `json:"fuentes"`
	Perfiles   []domain.PerfilPublicadoAdministracionV1 `json:"perfiles"`
}

func TestPlanRequiereCatalogoCanonicoYHuellaAprobada(t *testing.T) {
	desde := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	concesion := domain.ConcesionRol{Accion: "sintetico.consultar", ModuloID: "sintetico", TipoRecurso: "expediente",
		Finalidades: []string{"revision"}, GarantiaMinima: domain.AuthAssuranceHigh}
	rol := domain.VersionRol{RolID: "revision_sintetica", Version: 1, Nombre: "revision_sintetica",
		Estado: domain.EstadoVersionRolPublicada, Concesiones: []domain.ConcesionRol{concesion},
		PublicadaPor: "actor:fuente", PublicadaEn: desde}
	c := domain.CatalogoAccionesAdministracionV1{Referencia: "catalogo:sintetico", Version: 1,
		FuenteRef: "paquete:sintetico", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("a", 64),
		VigenteDesde: desde, VigenteHasta: desde.Add(72 * time.Hour),
		Entradas: []domain.EntradaAccionAdministracionV1{{Referencia: "accion:sintetica", Version: 1,
			FuenteRef: "fuente:sintetica", FuenteVersion: 1, FuenteHuellaSHA256: strings.Repeat("b", 64),
			Concesion: concesion, DimensionesAmbito: []string{"unidad"}, ClaseControl: "consulta_auditada",
			VigenteDesde: desde, VigenteHasta: desde.Add(48 * time.Hour)}},
		Perfiles: []domain.PerfilPublicadoAdministracionV1{{Rol: rol,
			TipoPerfil: domain.TipoPerfilAdministracionFijoSistemaV1,
			ControlVigencia: domain.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1,
				Estado: domain.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: "actor:fuente", ActualizadoEn: desde}}}}
	const preimagen = `[{"referencia":"accion:sintetica","version":1,"fuente_ref":"fuente:sintetica","fuente_version":1,"concesion":{"accion":"sintetico.consultar","modulo_id":"sintetico","tipo_recurso":"expediente","finalidades":["revision"],"garantia_minima":"alto"},"dimensiones_ambito":["unidad"],"clase_control":"consulta_auditada","vigente_desde":"2026-10-08T00:00:00Z","vigente_hasta":"2026-10-10T00:00:00Z"}]`
	hs := sha256.Sum256([]byte(preimagen))
	c.Entradas[0].FuenteHuellaSHA256 = hex.EncodeToString(hs[:])
	paquete, err := json.Marshal(paqueteAdmisionPrueba{Esquema: "vec.admin.catalogo-acciones.paquete.v2", Referencia: c.FuenteRef,
		Version: c.FuenteVersion,
		Fuentes: []fuentePaquetePrueba{{ModuloID: "sintetico", Referencia: c.Entradas[0].FuenteRef, Version: 1,
			HuellaSHA256: c.Entradas[0].FuenteHuellaSHA256, EntradasCanon: preimagen, Entradas: c.Entradas}},
		Perfiles: c.Perfiles})
	if err != nil {
		t.Fatal(err)
	}
	sPaquete := sha256.Sum256(paquete)
	c.FuenteHuellaSHA256 = hex.EncodeToString(sPaquete[:])
	canon, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	plan := func(huella string, contenido []byte) ([]byte, string) {
		b, err := json.Marshal(planAdmision{Esquema: "vec.admin.catalogo-acciones.plan.v2",
			OperacionRef:  "caa_" + strings.Repeat("a", 22),
			PaqueteCanon:  string(paquete),
			CatalogoCanon: string(contenido), CatalogoRef: c.Referencia, CatalogoVersion: "1",
			CatalogoSHA256: huella, PaqueteRef: c.FuenteRef, PaqueteVersion: "1",
			PaqueteSHA256: c.FuenteHuellaSHA256, AprobacionRef: "aprobacion:sintetica",
			AprobacionSHA256: strings.Repeat("c", 64), EsperadoVersion: "0",
			EsperadoSHA256: strings.Repeat("0", 64), PreparadoEn: "2026-10-07T00:00:00Z",
			CaducaEn: "2026-10-09T00:00:00Z", Entorno: "desarrollo"})
		if err != nil {
			t.Fatal(err)
		}
		s := sha256.Sum256(b)
		return b, hex.EncodeToString(s[:])
	}
	p, hp := plan(h, canon)
	if _, err := validarPlanAntesDeEnviar(p, hp); err != nil {
		t.Fatal(err)
	}
	catalogoBase := c
	if _, err := validarPlanAntesDeEnviar(p, strings.Repeat("f", 64)); err == nil {
		t.Fatal("plan sin la huella aprobada")
	}
	alterado, ha := plan(strings.Repeat("e", 64), canon)
	if _, err := validarPlanAntesDeEnviar(alterado, ha); err == nil {
		t.Fatal("catálogo alterado bajo un plan rehuellado")
	}
	c.Perfiles = nil
	sinCenso, _ := json.Marshal(c)
	alterado, ha = plan(h, sinCenso)
	if _, err := validarPlanAntesDeEnviar(alterado, ha); err == nil {
		t.Fatal("censo omitido")
	}
	rechazarAunqueRehuellado := func(nombre string, documento []byte) {
		t.Run(nombre, func(t *testing.T) {
			s := sha256.Sum256(documento)
			if _, err := validarPlanAntesDeEnviar(documento, hex.EncodeToString(s[:])); err == nil {
				t.Fatal("JSON no cerrado aceptado con SHA del propio documento")
			}
		})
	}
	rechazarAunqueRehuellado("raiz extra", append(append([]byte{}, p[:len(p)-1]...), []byte(`,"extra":"ajeno"}`)...))
	rechazarAunqueRehuellado("raiz duplicada", append([]byte(`{"esquema":"ajeno",`), p[1:]...))
	rechazarAunqueRehuellado("raiz nula", bytes.Replace(p, []byte(`"entorno":"desarrollo"`), []byte(`"entorno":null`), 1))
	rechazarAunqueRehuellado("raiz incompleta", bytes.Replace(p, []byte(`,"entorno":"desarrollo"`), nil, 1))
	catalogoExtra := append(append([]byte{}, canon[:len(canon)-1]...), []byte(`,"extra":"ajeno"}`)...)
	catalogoDuplicado := append([]byte(`{"referencia":"catalogo:otro",`), canon[1:]...)
	for nombre, contenido := range map[string][]byte{
		"catalogo extra":     catalogoExtra,
		"catalogo duplicado": catalogoDuplicado,
		"catalogo nulo":      bytes.Replace(canon, []byte(`"vigente_desde":"2026-10-08T00:00:00Z"`), []byte(`"vigente_desde":null`), 1),
	} {
		s := sha256.Sum256(contenido)
		mutado, _ := plan(hex.EncodeToString(s[:]), contenido)
		rechazarAunqueRehuellado(nombre, mutado)
	}
	var incompleto map[string]json.RawMessage
	if err := json.Unmarshal(canon, &incompleto); err != nil {
		t.Fatal(err)
	}
	delete(incompleto, "perfiles")
	catalogoIncompleto, _ := json.Marshal(incompleto)
	s := sha256.Sum256(catalogoIncompleto)
	mutado, _ := plan(hex.EncodeToString(s[:]), catalogoIncompleto)
	rechazarAunqueRehuellado("catalogo incompleto", mutado)
	var planBase planAdmision
	if err := json.Unmarshal(p, &planBase); err != nil {
		t.Fatal(err)
	}
	for nombre, paqueteAlterado := range map[string][]byte{
		"paquete v1":         bytes.Replace(paquete, []byte(`paquete.v2`), []byte(`paquete.v1`), 1),
		"fuente sin huella":  bytes.Replace(paquete, []byte(catalogoBase.Entradas[0].FuenteHuellaSHA256), []byte(strings.Repeat("f", 64)), 1),
		"fuente extra":       bytes.Replace(paquete, []byte(`"modulo_id":"sintetico"`), []byte(`"modulo_id":"sintetico","extra":true`), 1),
		"fuente duplicada":   bytes.Replace(paquete, []byte(`"modulo_id":"sintetico"`), []byte(`"modulo_id":"ajeno","modulo_id":"sintetico"`), 1),
		"unicode descriptor": bytes.Replace(paquete, []byte(`"referencia":"fuente:sintetica"`), []byte(`"referencia":"fuente:ñ"`), 1),
		"HTML escapado":      bytes.Replace(paquete, []byte(`"referencia":"fuente:sintetica"`), []byte(`"referencia":"fuente:\u003csintetica"`), 1),
		"número no canónico": bytes.Replace(paquete, []byte(`"version":1`), []byte(`"version":1.0`), 1),
		"fecha alterada":     bytes.Replace(paquete, []byte(`"vigente_hasta":"2026-10-10T00:00:00Z"`), []byte(`"vigente_hasta":"2026-10-09T00:00:00Z"`), 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			if bytes.Equal(paqueteAlterado, paquete) {
				t.Fatal("mutación de prueba no aplicada")
			}
			paqueteSHA := sha256.Sum256(paqueteAlterado)
			otroCatalogo := catalogoBase
			otroCatalogo.FuenteHuellaSHA256 = hex.EncodeToString(paqueteSHA[:])
			catalogoCanon, err := json.Marshal(otroCatalogo)
			if err != nil {
				t.Fatal(err)
			}
			catalogoSHA, err := otroCatalogo.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			otroPlan := planBase
			otroPlan.PaqueteCanon = string(paqueteAlterado)
			otroPlan.PaqueteSHA256 = otroCatalogo.FuenteHuellaSHA256
			otroPlan.CatalogoCanon = string(catalogoCanon)
			otroPlan.CatalogoSHA256 = catalogoSHA
			b, err := json.Marshal(otroPlan)
			if err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(b)
			if _, err := validarPlanAntesDeEnviar(b, hex.EncodeToString(h[:])); err == nil {
				t.Fatal("preflight admitió paquete alterado antes de transacción")
			}
		})
	}
}

func TestArchivoPrivadoNiegaPermisosAmpliosYEnlaces(t *testing.T) {
	raiz := t.TempDir()
	ruta := filepath.Join(raiz, "plan.json")
	if err := os.WriteFile(ruta, []byte(`{"plan":"sintetico"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerPrivado(ruta, 1024); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(raiz, "enlace.json")
	if err := os.Symlink(ruta, enlace); err != nil {
		t.Fatal(err)
	}
	if _, err := leerPrivado(enlace, 1024); err == nil {
		t.Fatal("enlace al plan aceptado")
	}
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerPrivado(ruta, 1024); err == nil {
		t.Fatal("plan legible por otros aceptado")
	}
}

func TestFalloCatalogoClasificaSinFiltrarCausa(t *testing.T) {
	var salida bytes.Buffer
	if codigo := informarFalloCatalogoCLI(&salida, os.ErrNotExist); codigo != 2 ||
		!bytes.Contains(salida.Bytes(), []byte(`"catalogo_ausente"`)) {
		t.Fatalf("fallo de catálogo sin causa clasificada: %d %s", codigo, salida.String())
	}
	salida.Reset()
	if codigo := informarFalloCatalogoCLI(&salida, os.ErrPermission); codigo != 2 ||
		!bytes.Contains(salida.Bytes(), []byte(`"catalogo_invalido"`)) ||
		bytes.Contains(salida.Bytes(), []byte("permission denied")) {
		t.Fatalf("fallo de catálogo filtró la causa privada: %d %s", codigo, salida.String())
	}
}
