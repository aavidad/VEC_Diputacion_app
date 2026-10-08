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
	canon, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	h, err := c.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	plan := func(huella string, contenido []byte) ([]byte, string) {
		b, err := json.Marshal(planAdmision{Esquema: "vec.admin.catalogo-acciones.plan.v1",
			OperacionRef:  "caa_" + strings.Repeat("a", 22),
			PaqueteCanon:  `{}`,
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
