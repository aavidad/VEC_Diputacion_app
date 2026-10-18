package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestPerfilDerivacionNombramientoExigeAccionYAuditoriaPropias(t *testing.T) {
	p := politicaPerfilOfertaCTDesarrollo{accion: "contratacion_temporal.nombramiento.derivar",
		modulo: "contratacion_temporal", tipo: ports.TipoRecursoDerivacionNombramiento,
		finalidad: "gestionar_contratacion_temporal", obligaciones: []string{"auditar"}}
	c := core.ConcesionRol{Accion: p.accion, ModuloID: p.modulo, TipoRecurso: p.tipo,
		Finalidades: []string{p.finalidad}, GarantiaMinima: core.AuthAssuranceHigh,
		Obligaciones: []string{"auditar"}}
	if !concesionExactaPerfilOfertaCTDesarrollo(c, p) {
		t.Fatal("la concesión propia publicada debe poder consumirse")
	}
	c.Obligaciones = nil
	if concesionExactaPerfilOfertaCTDesarrollo(c, p) {
		t.Fatal("se aceptó derivar sin obligación de auditoría")
	}
	c.Obligaciones = []string{"auditar"}
	c.Accion = "contratacion_temporal.propuesta_formalizacion"
	if concesionExactaPerfilOfertaCTDesarrollo(c, p) {
		t.Fatal("se heredó el permiso de la propuesta")
	}
	// Los perfiles de oferta anteriores conservan su contrato de cero obligaciones.
	c = core.ConcesionRol{Accion: politicaVinculoOfertaCTDesarrollo().accion,
		ModuloID:       politicaVinculoOfertaCTDesarrollo().modulo,
		TipoRecurso:    politicaVinculoOfertaCTDesarrollo().tipo,
		Finalidades:    []string{politicaVinculoOfertaCTDesarrollo().finalidad},
		GarantiaMinima: core.AuthAssuranceHigh}
	if !concesionExactaPerfilOfertaCTDesarrollo(c, politicaVinculoOfertaCTDesarrollo()) {
		t.Fatal("cambió el perfil de oferta existente")
	}
}

func TestActivacionNombramientoPrivadaOpcionalYCerrada(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "identidad"), 0700); err != nil {
		t.Fatal(err)
	}
	cfg := configuracionOfertaCTPrueba(dir)
	if _, presente, err := leerActivacionNombramientoCTDesarrollo(cfg); presente || err != nil {
		t.Fatalf("ausencia no opcional: presente=%v err=%v", presente, err)
	}
	c := archivoActivacionNombramientoCT{Esquema: esquemaActivacionNombramientoCTDesarrollo,
		Derivacion: perfilActivacionOfertaCT{PerfilRef: "prf:ct:nombramiento-derivado",
			Rol: referenciaRolOfertaCTDesarrollo{RolID: "ct_nombramiento_derivado", Version: 1,
				HuellaSHA256: strings.Repeat("a", 64)},
			Motivo: core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_ct_nombramiento", CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_nombramiento_derivar"}},
		FuenteAutorizacionDSNFile: "identidad/fuente.dsn", MotivosAutorizacionDSNFile: "identidad/motivos.dsn"}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, archivoActivacionNombramientoCTDesarrollo)
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, presente, err := leerActivacionNombramientoCTDesarrollo(cfg); !presente || err != nil {
		t.Fatalf("activación válida rechazada: presente=%v err=%v", presente, err)
	}
	c.FuenteAutorizacionDSNFile = "../fuente.dsn"
	b, _ = json.Marshal(c)
	if err := os.WriteFile(ruta, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, presente, err := leerActivacionNombramientoCTDesarrollo(cfg); !presente || err == nil {
		t.Fatalf("ruta fuera del árbol admitida: presente=%v err=%v", presente, err)
	}
}
