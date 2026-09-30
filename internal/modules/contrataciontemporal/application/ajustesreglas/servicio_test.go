package ajustesreglas

import (
	"context"
	"os"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecpruebas "vec-diputacion-granada/internal/vec/pruebas"
	"vec-diputacion-granada/internal/vec/reglas"
)

type relojFijo struct{ ahora time.Time }

func (r relojFijo) Ahora() time.Time { return r.ahora }

type repoPrueba struct {
	lectura     Lectura
	material    Material
	operaciones int
}

func (r *repoPrueba) Consultar(context.Context, vecdomain.ContextoActor, int, *int64) (Lectura, error) {
	return r.lectura, nil
}
func (r *repoPrueba) Operar(_ context.Context, _ vecdomain.ContextoActor, m Material) (Resultado, error) {
	r.operaciones++
	r.material = m
	return Resultado{Replay: r.lectura.Vigente != nil,
		Recibo: Recibo{ReciboRef: "recibo:00000000-0000-4000-8000-000000000001", ClaveIdempotencia: m.ClaveIdempotencia,
			Version: r.lectura.VigenteVersionResultado(), HuellaSHA256: m.AjustesHuellaSHA256,
			VigenteDesde: time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC),
			DecisionRef:  "decision:prueba", AuditoriaRef: "auditoria:prueba", ConsumoHuellaSHA256: m.AjustesHuellaSHA256}}, nil
}
func (l Lectura) VigenteVersionResultado() int {
	if l.Vigente == nil {
		return 1
	}
	return l.Vigente.Version
}

func TestPublicarPreparaValorAnteriorDeBaseYReplayNoMueveCAS(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	archivo, err := fichero.NuevaConsultaCatalogos("../../../../../data/demo/reglas/ct_reglas.ejemplo.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: archivo, Metadatos: archivo, CatalogoID: reglas.CatalogoContratacionTemporal,
		ModuloID: reglas.ModuloContratacionTemporal, Reloj: relojFijo{ahora},
	})
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := os.ReadFile("../../../../../data/catalogos/contratacion_temporal/motivos_ajuste_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	motivos, err := LeerCatalogoMotivos(contenido)
	if err != nil {
		t.Fatal(err)
	}
	actor, _, err := vecpruebas.NuevoContextoYVinculo(ahora, "per_0123456789abcdef0123456789abcdef",
		"prf_0123456789abcdef0123456789abcdef", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repoPrueba{}
	servicio, err := NuevoServicio(repo, resolver, motivos, relojFijo{ahora})
	if err != nil {
		t.Fatal(err)
	}
	versionEsperada := 0
	solicitud := Solicitud{ClaveIdempotencia: "12345678-1234-4234-8234-123456789abc", VersionEsperada: &versionEsperada,
		Cambios:     []CambioSolicitado{{ReglaClave: reglas.CTPlazoFiscalizacion, Campo: reglas.CampoCantidad, Nuevo: "7"}},
		MotivoClave: "respuesta_rrhh_duda"}
	if _, err := servicio.Publicar(t.Context(), actor, solicitud); err != nil {
		t.Fatal(err)
	}
	if repo.material.VersionEsperada != 0 || len(repo.material.Cambios) != 1 || repo.material.Cambios[0].Anterior != "10" ||
		repo.material.Cambios[0].Nuevo != "7" || repo.material.OrganizacionRef != "" {
		t.Fatalf("primera preparación insegura: %+v", repo.material)
	}
	huella, err := reglas.HuellaAjustes(map[string]map[string]string{reglas.CTPlazoFiscalizacion: {reglas.CampoCantidad: "7"}})
	if err != nil {
		t.Fatal(err)
	}
	repo.lectura = Lectura{Vigente: &reglas.VersionAjustes{CatalogoID: reglas.CatalogoAjustesDe(reglas.CatalogoContratacionTemporal),
		Version: 1, HuellaSHA256: huella, VigenteDesde: ahora.Add(-time.Second),
		Ajustes: map[string]map[string]string{reglas.CTPlazoFiscalizacion: {reglas.CampoCantidad: "7"}}},
		VigenteBaseVersion: 1, VigenteBaseHuella: repo.material.BaseHuellaSHA256}
	if _, err := servicio.Publicar(t.Context(), actor, solicitud); err != nil {
		t.Fatal(err)
	}
	if repo.material.VersionEsperada != 0 || repo.material.Cambios[0].Anterior != "7" || repo.material.Cambios[0].Nuevo != "7" || repo.operaciones != 2 {
		t.Fatalf("replay no conservó petición original: %+v", repo.material)
	}
}
