package ajustesreglas

import (
	"context"
	"errors"
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
	activacion  ActivacionBase
	material    Material
	original    Material
	operaciones int
}

func (r *repoPrueba) Consultar(context.Context, vecdomain.ContextoActor, int, *int64) (Lectura, error) {
	lectura := r.lectura
	if lectura.CorteEn.IsZero() {
		lectura.CorteEn = time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	}
	return lectura, nil
}
func (r *repoPrueba) LeerActivacion(context.Context) (ActivacionBase, error) {
	return r.activacion, nil
}
func (r *repoPrueba) LeerPreimagen(_ context.Context, _ vecdomain.ContextoActor, clave string) (Material, bool, error) {
	return r.original, r.original.ClaveIdempotencia == clave, nil
}

func activarBasePrueba(t *testing.T, repo *repoPrueba, fuente FuenteReglas) {
	t.Helper()
	base, huella, _, err := fuente.CatalogoVigente(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	repo.activacion = ActivacionBase{Estado: "activa", Secuencia: 1, CatalogoID: base.ID,
		Version: base.Version, HuellaSHA256: huella, AprobacionRef: base.AprobacionRef}
}
func (r *repoPrueba) Operar(_ context.Context, _ vecdomain.ContextoActor, m Material) (Resultado, error) {
	r.operaciones++
	r.material = m
	if r.original.ClaveIdempotencia == "" {
		r.original = m
	}
	replay := r.original.ClaveIdempotencia == m.ClaveIdempotencia && r.lectura.Vigente != nil
	version := r.lectura.VigenteVersionResultado()
	vigenteDesde := time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC)
	if m.VigenteDesde != nil {
		vigenteDesde, _ = time.Parse("2006-01-02T15:04:05.000000Z", *m.VigenteDesde)
	}
	if replay {
		version = r.original.VersionEsperada + 1
	}
	return Resultado{Replay: replay,
		Recibo: Recibo{ReciboRef: "recibo:00000000-0000-4000-8000-000000000001", ClaveIdempotencia: m.ClaveIdempotencia,
			Version: version, HuellaSHA256: m.AjustesHuellaSHA256,
			VigenteDesde: vigenteDesde,
			PublicadaEn:  time.Date(2026, 9, 30, 10, 0, 0, 0, time.UTC),
			DecisionRef:  "decision:prueba", AuditoriaRef: "auditoria:prueba", ConsumoHuellaSHA256: m.AjustesHuellaSHA256}}, nil
}
func (l Lectura) VigenteVersionResultado() int {
	if l.Vigente == nil {
		return 1
	}
	return l.Vigente.Version + 1
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
	motivos := motivosPrueba()
	actor, _, err := vecpruebas.NuevoContextoYVinculo(ahora, "per_0123456789abcdef0123456789abcdef",
		"prf_0123456789abcdef0123456789abcdef", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repoPrueba{}
	activarBasePrueba(t, repo, resolver)
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
	repo.lectura.VigenteHoy = repo.lectura.Vigente
	if _, err := servicio.Publicar(t.Context(), actor, solicitud); err != nil {
		t.Fatal(err)
	}
	if repo.material.VersionEsperada != 0 || repo.material.Cambios[0].Anterior != "10" || repo.material.Cambios[0].Nuevo != "7" || repo.operaciones != 2 {
		t.Fatalf("replay no conservó petición original: %+v", repo.material)
	}
	// Otra publicación posterior no cambia la preimagen del primer recibo.
	repo.lectura.Vigente.Version = 2
	if resultado, err := servicio.Publicar(t.Context(), actor, solicitud); err != nil || !resultado.Replay || resultado.Recibo.Version != 1 ||
		repo.material.AjustesHuellaSHA256 != repo.original.AjustesHuellaSHA256 || repo.operaciones != 3 {
		t.Fatalf("replay tras cabeza posterior cambió material: %+v, %v", resultado, err)
	}
	fechaDistinta := "2026-10-01T00:00:00Z"
	solicitud.VigenteDesde = &fechaDistinta
	if _, err := servicio.Publicar(t.Context(), actor, solicitud); !errors.Is(err, ErrConflicto) || repo.operaciones != 3 {
		t.Fatalf("replay con fecha cambiada no quedó en conflicto: %v", err)
	}
	solicitud.VigenteDesde = nil
	// La guarda CT158 sucede tras el replay: una activación retirada no debe
	// ocultar el recibo de una petición ya confirmada.
	repo.activacion = ActivacionBase{Estado: "inactiva", Secuencia: 2}
	if _, err := servicio.Publicar(t.Context(), actor, solicitud); err != nil || repo.operaciones != 4 {
		t.Fatalf("replay histórico bloqueado por activación posterior: %v", err)
	}
	versionEsperada = 2
	solicitud.ClaveIdempotencia = "12345678-1234-4234-8234-123456789abd"
	solicitud.Cambios[0].Nuevo = "8"
	if _, err := servicio.Publicar(t.Context(), actor, solicitud); !errors.Is(err, ErrNoDisponible) || repo.operaciones != 4 {
		t.Fatalf("publicación sin base activa llegó al efecto: %v", err)
	}
	activarBasePrueba(t, repo, resolver)
	fechaFutura := "2026-10-01T09:00:00Z"
	solicitud.VigenteDesde = &fechaFutura
	resultado, err := servicio.Publicar(t.Context(), actor, solicitud)
	if err != nil || resultado.Replay || resultado.Recibo.Version != 3 ||
		repo.material.VigenteDesde == nil || *repo.material.VigenteDesde != "2026-10-01T09:00:00.000000Z" {
		t.Fatalf("programación nueva perdió fecha de efecto: %+v, %+v, %v", resultado, repo.material, err)
	}
}

func TestConsultaProyectaValorVigenteDeCabezaAutorizada(t *testing.T) {
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	archivo, err := fichero.NuevaConsultaCatalogos("../../../../../data/demo/reglas/ct_reglas.ejemplo.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: archivo, Metadatos: archivo, CatalogoID: reglas.CatalogoContratacionTemporal,
		ModuloID: reglas.ModuloContratacionTemporal, Reloj: relojFijo{ahora},
	})
	if err != nil {
		t.Fatal(err)
	}
	motivos := motivosPrueba()
	actor, _, err := vecpruebas.NuevoContextoYVinculo(ahora, "per_0123456789abcdef0123456789abcdef",
		"prf_0123456789abcdef0123456789abcdef", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	ajustes := map[string]map[string]string{reglas.CTPlazoFiscalizacion: {reglas.CampoCantidad: "7"}}
	huella, err := reglas.HuellaAjustes(ajustes)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repoPrueba{lectura: Lectura{Vigente: &reglas.VersionAjustes{
		CatalogoID: reglas.CatalogoAjustesDe(reglas.CatalogoContratacionTemporal), Version: 1,
		HuellaSHA256: huella, VigenteDesde: ahora.Add(-time.Hour), Ajustes: ajustes,
	}, VigenteBaseVersion: 1, VigenteBaseHuella: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	repo.lectura.VigenteHoy = repo.lectura.Vigente
	activarBasePrueba(t, repo, resolutor)
	servicio, err := NuevoServicio(repo, resolutor, motivos, relojFijo{ahora})
	if err != nil {
		t.Fatal(err)
	}
	lectura, err := servicio.Consultar(t.Context(), actor, 20, nil)
	if err != nil {
		t.Fatal(err)
	}
	if lectura.Activacion.Estado != "activa" || lectura.Activacion.HuellaSHA256 != repo.activacion.HuellaSHA256 {
		t.Fatalf("GET perdió estado nominal de activación: %+v", lectura.Activacion)
	}
	// Una cabeza futura no cambia el valor efectivo que muestra la ficha.
	ajustesFuturos := map[string]map[string]string{reglas.CTPlazoFiscalizacion: {reglas.CampoCantidad: "8"}}
	huellaFutura, err := reglas.HuellaAjustes(ajustesFuturos)
	if err != nil {
		t.Fatal(err)
	}
	repo.lectura.Vigente = &reglas.VersionAjustes{CatalogoID: reglas.CatalogoAjustesDe(reglas.CatalogoContratacionTemporal),
		Version: 2, HuellaSHA256: huellaFutura, VigenteDesde: ahora.Add(time.Hour), Ajustes: ajustesFuturos}
	lecturaFutura, err := servicio.Consultar(t.Context(), actor, 20, nil)
	if err != nil || lecturaFutura.Vigente.Version != 2 || lecturaFutura.VigenteHoy.Version != 1 {
		t.Fatalf("GET mezcló cabeza futura y vigente: %+v, %v", lecturaFutura, err)
	}
	for _, regla := range lecturaFutura.Reglas {
		if regla.Clave == reglas.CTPlazoFiscalizacion && (regla.Cantidad != 7 || regla.Ajuste.Version != 1) {
			t.Fatalf("GET aplicó regla futura: %+v", regla)
		}
	}
	repo.lectura.PuedeAjustar = true
	repo.activacion = ActivacionBase{Estado: "inactiva", Secuencia: 2}
	pausa, err := servicio.Consultar(t.Context(), actor, 20, nil)
	if err != nil || pausa.PuedeAjustar || len(pausa.Reglas) != 0 || pausa.Vigente == nil ||
		pausa.Activacion.Estado != "inactiva" || pausa.Activacion.Secuencia != 2 {
		t.Fatalf("GET inactivo atribuyó reglas vigentes o perdió historia: %+v, %v", pausa, err)
	}
	repo.activacion = ActivacionBase{Estado: "sin_publicar"}
	pausa, err = servicio.Consultar(t.Context(), actor, 20, nil)
	if err != nil || pausa.PuedeAjustar || len(pausa.Reglas) != 0 || pausa.Activacion.Estado != "sin_publicar" {
		t.Fatalf("GET sin publicación atribuyó reglas vigentes: %+v, %v", pausa, err)
	}
	activarBasePrueba(t, repo, resolutor)
	repo.activacion.HuellaSHA256 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if _, err := servicio.Consultar(t.Context(), actor, 20, nil); !errors.Is(err, ErrNoDisponible) {
		t.Fatalf("GET proyectó catálogo no activado: %v", err)
	}
	for _, regla := range lectura.Reglas {
		if regla.Clave == reglas.CTPlazoFiscalizacion {
			if regla.Cantidad != 7 || regla.Ajuste == nil || regla.Ajuste.Version != lectura.VigenteHoy.Version {
				t.Fatalf("GET mezcló cabeza e inicial: %+v", regla)
			}
			return
		}
	}
	t.Fatal("GET sin c03")
}

func motivosPrueba() CatalogoMotivos {
	return CatalogoMotivos{ID: "vec.contratacion_temporal.reglas.motivos_ajuste", Version: 1,
		Motivos: []Motivo{{Clave: "respuesta_rrhh_duda", TextoClave: "ajustesMotivo_respuesta_rrhh_duda"}}}
}
