package main

import (
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
		b, err := json.Marshal(planMinimo{CatalogoCanon: string(contenido), CatalogoSHA256: huella})
		if err != nil {
			t.Fatal(err)
		}
		s := sha256.Sum256(b)
		return b, hex.EncodeToString(s[:])
	}
	p, hp := plan(h, canon)
	if err := validarPlanAntesDeEnviar(p, hp); err != nil {
		t.Fatal(err)
	}
	if err := validarPlanAntesDeEnviar(p, strings.Repeat("f", 64)); err == nil {
		t.Fatal("plan sin la huella aprobada")
	}
	alterado, ha := plan(strings.Repeat("e", 64), canon)
	if err := validarPlanAntesDeEnviar(alterado, ha); err == nil {
		t.Fatal("catálogo alterado bajo un plan rehuellado")
	}
	c.Perfiles = nil
	sinCenso, _ := json.Marshal(c)
	alterado, ha = plan(h, sinCenso)
	if err := validarPlanAntesDeEnviar(alterado, ha); err == nil {
		t.Fatal("censo omitido")
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
