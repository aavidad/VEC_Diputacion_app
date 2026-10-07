package internactproveedores

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/modules/personal/domain"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestMaterialRPTPublicaV3ExigePerfilYCapacidadPropios(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "rpt_publica_v3.json")
	motivo := map[string]any{"catalogo_id": "motivos.rpt", "catalogo_version": 1, "catalogo_huella_sha256": strings.Repeat("a", 64), "entrada_clave": "motivo_0123456789abcdef0123456789abcdef"}
	doc := map[string]any{"version": 1, "catalogo_motivos": "motivos.rpt", "motivo_consulta": motivo,
		"capacidad": map[string]any{"archivo": "rpt-publica.key"},
		"perfiles":  map[string]any{"cta_0123456789abcdefghijkl": map[string]any{"perfil_activo_ref": "prf_0123456789abcdefghijkl", "perfil_version": 5}}}
	escribir := func() {
		t.Helper()
		b, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir()
	m, err := CargarMaterialRPTPublicaV3(dir)
	if err != nil || len(m.Perfiles) != 1 {
		t.Fatalf("material válido: %v", err)
	}
	_ = m.Cerrar()
	doc["perfiles"].(map[string]any)["cta_0123456789abcdefghijkl"].(map[string]any)["perfil_version"] = 0
	escribir()
	if _, err := CargarMaterialRPTPublicaV3(dir); !errors.Is(err, ErrRPTPublicaV3NoDisponible) {
		t.Fatalf("perfil sin versión: %v", err)
	}
	doc["perfiles"].(map[string]any)["cta_0123456789abcdefghijkl"].(map[string]any)["perfil_version"] = 5
	doc["capacidad"].(map[string]any)["archivo"] = "../capacidad.key"
	escribir()
	if _, err := CargarMaterialRPTPublicaV3(dir); !errors.Is(err, ErrRPTPublicaV3NoDisponible) {
		t.Fatalf("escape de raíz privada: %v", err)
	}
}

func TestRPTPublicaV3SinAutoridadesFallaCerrada(t *testing.T) {
	if _, err := ConstruirRPTPublicaV3(context.Background(), MaterialRPTPublicaV3{}, nil, nil, nil); !errors.Is(err, ErrRPTPublicaV3NoDisponible) {
		t.Fatalf("constructor: %v", err)
	}
	var p *ProveedorAutorizacionRPTPublicaV3
	if _, err := p.AutorizarConsultaRPTPublicaV2(context.Background(), domain.MaterialConsultaRPTPublicaV2{}); !errors.Is(err, ErrRPTPublicaV3NoDisponible) {
		t.Fatalf("consulta: %v", err)
	}
}

type fuenteCamposRPTPrueba struct{ identidad ct.ContextoAutorizacionAltaV3 }

func (f fuenteCamposRPTPrueba) ContextoVinculadoRPTPublicaV2(context.Context) (ct.ContextoAutorizacionAltaV3, error) {
	return f.identidad, nil
}

type exportadorCamposRPTPrueba struct{ llamadas *int }

func (e exportadorCamposRPTPrueba) String() string       { return "[exportador de prueba]" }
func (e exportadorCamposRPTPrueba) LogValue() slog.Value { return slog.StringValue(e.String()) }
func (e exportadorCamposRPTPrueba) ExportarMaterialParaConsumidor() (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	*e.llamadas++
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin firma COSE en esta prueba")
}

type emisorCamposRPTPrueba struct {
	t        *testing.T
	ahora    time.Time
	campos   []string
	llamadas *int
}

func (e emisorCamposRPTPrueba) EmitirMaterialAutorizacionAtestadaV3(_ context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Helper()
	datos, err := s.Datos()
	if err != nil {
		e.t.Fatal(err)
	}
	rol := core.VersionRol{RolID: "tecnico_rrhh", Version: 1, Nombre: "Tecnico RRHH", Estado: core.EstadoVersionRolPublicada,
		Concesiones:  []core.ConcesionRol{{Accion: domain.AccionConsultaRPTPublicaV2, ModuloID: "personal", TipoRecurso: "rpt_publica_publicacion", Finalidades: []string{domain.FinalidadConsultaRPTPublicaV2}, GarantiaMinima: core.AuthAssuranceSubstantial, CamposPermitidos: e.campos}},
		PublicadaPor: "responsable-seguridad", PublicadaEn: e.ahora.Add(-24 * time.Hour)}
	huella, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		e.t.Fatal(err)
	}
	instantanea := core.InstantaneaAutorizacion{AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-rpt", Version: 1, PerfilActivoRef: r.Contexto.PerfilActivoRef, PrincipalID: r.Contexto.PersonaRef, VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva,
		Ambitos:      []core.AmbitoPerfil{{Clave: "publicacion_ref", Valores: []string{"rpt-publicada:2026-05-07"}}},
		VigenteDesde: e.ahora.Add(-time.Hour), VigenteHasta: e.ahora.Add(time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: e.ahora.Add(-2 * time.Hour)},
		VersionRol: rol, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn},
		RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
	if datos.Recurso.Ambitos["publicacion_ref"] == "" {
		e.t.Fatal("recurso sin ámbito de publicación")
	}
	evidencia, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(s, instantanea, "dec_0123456789abcdef0123456789abcdef", e.ahora, e.ahora.Add(90*time.Second))
	if err != nil {
		e.t.Fatal(err)
	}
	decision, err := core.NuevaDecisionAutorizacionLigadaV3(s, evidencia)
	if err != nil {
		e.t.Fatal(err)
	}
	return decision, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, exportadorCamposRPTPrueba{e.llamadas}, nil
}

func materialCamposRPTPrueba(t *testing.T, actor core.ContextoActor, vista string) domain.MaterialConsultaRPTPublicaV2 {
	t.Helper()
	catalogo := domain.CatalogoRPTPublicaV2{Esquema: domain.EsquemaCandidatoRPTPublicaV2, Estado: domain.EstadoCandidatoRPTPublicaV2,
		Fuente:     domain.FuenteRPTPublica{Documento: "RPT publicada", Importacion: "rpt-2026", GeneradoEn: "2026-09-17", Aviso: "Sin ocupantes."},
		Resumen:    domain.ResumenRPTPublica{Puestos: 1, Dotacion: 1, Categorias: 1, Centros: 1},
		Categorias: []domain.CategoriaRPTPublicaV2{{Clave: "auxiliar", Denominacion: "AUXILIAR", Origen: "categoria", Grupos: []string{"C2"}, Escalas: []string{}, Puestos: 1, Dotacion: 1}},
		Puestos:    []domain.PuestoRPTPublicoV2{{PuestoRPTPublico: domain.PuestoRPTPublico{Codigo: "430-101-001", Denominacion: "AUXILIAR", CentroCodigo: "101", Centro: "CENTRO", Delegacion: "AREA", Grupos: []string{"C2"}, CategoriaClave: "auxiliar", Dotacion: 1, Tipo: "N", Provision: "C"}, CategoriasClaves: []string{"auxiliar"}}}, CategoriasPendientesGrupo: []string{}}
	m, err := domain.NuevoMaterialConsultaRPTPublicaV2(domain.SolicitudConsultaRPTPublicaV2{Actor: actor,
		Filtro: domain.FiltroRPTPublicaV2{Vista: vista, Limite: 25}, Snapshot: domain.SnapshotRPTPublicaV2{PublicacionRef: "rpt-publicada:2026-05-07", Corte: "2026-05-07", HuellaSHA256: strings.Repeat("a", 64), Catalogo: catalogo}})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestRPTPublicaV3NoExportaDecisionDeSoloMetadatos(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	identidad := contextoOHPrueba(t, ahora)
	for _, vista := range []string{"categorias", "puestos"} {
		t.Run(vista, func(t *testing.T) {
			material := materialCamposRPTPrueba(t, identidad.Resultado.Contexto, vista)
			for _, caso := range []struct {
				nombre        string
				campos        []string
				exportaciones int
			}{
				{"solo_metadatos", []string{"corte", "huella_sha256", "publicacion_ref"}, 0},
				{"sobre_completo", domain.CamposRespuestaRPTPublicaV2(), 1},
			} {
				t.Run(caso.nombre, func(t *testing.T) {
					llamadas := 0
					p := &ProveedorAutorizacionRPTPublicaV3{fuente: fuenteCamposRPTPrueba{identidad}, reloj: &relojB2{ahora: ahora},
						motivo:   core.ReferenciaEntradaCatalogo{CatalogoID: "motivos.rpt", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"},
						perfiles: map[string]PerfilRPTPublicaV3{identidad.Resultado.Contexto.Instantanea.CuentaRef: {PerfilActivoRef: identidad.Resultado.Contexto.PerfilActivoRef, PerfilVersion: identidad.Resultado.Contexto.Instantanea.PerfilVersion}},
						emisor:   emisorCamposRPTPrueba{t: t, ahora: ahora, campos: caso.campos, llamadas: &llamadas}, firmante: &firmanteRPTPublicaV3{}}
					_, err := p.AutorizarConsultaRPTPublicaV2(context.Background(), material)
					if caso.exportaciones == 0 && !errors.Is(err, domain.ErrRPTPublicaV2Denegada) || llamadas != caso.exportaciones {
						t.Fatalf("campos=%v exportaciones=%d err=%v", caso.campos, llamadas, err)
					}
				})
			}
		})
	}
}

func TestRPTPublicaV3AmbitoFijoNoCubreOtraPublicacion(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	actor := contextoOHPrueba(t, ahora).Resultado.Contexto
	material := materialCamposRPTPrueba(t, actor, "puestos")
	recurso := material.Recurso()
	if recurso.Ambitos["publicacion_ref"] != "rpt-publicada:2026-05-07" {
		t.Fatalf("ámbito=%v", recurso.Ambitos)
	}
	asignacion := core.AsignacionPerfil{AsignacionID: "asig-rpt", Version: 1, PerfilActivoRef: actor.PerfilActivoRef,
		PrincipalID: actor.Principal.ID, VersionRolRef: "rol:rpt:v1", Estado: core.EstadoAsignacionPerfilActiva,
		Ambitos:      []core.AmbitoPerfil{{Clave: "publicacion_ref", Valores: []string{"rpt-publicada:2026-05-07"}}},
		VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-2 * time.Hour)}
	if !asignacion.Cubre(recurso) {
		t.Fatal("la publicación fijada no quedó cubierta")
	}
	otra := material.Recurso()
	otra.Ambitos["publicacion_ref"] = "rpt-publicada:2026-06-01"
	if asignacion.Cubre(otra) || material.Recurso().Ambitos["publicacion_ref"] != "rpt-publicada:2026-05-07" {
		t.Fatal("se cubrió otra publicación o se mutó el material sellado")
	}
	filtrado := material.Recurso()
	filtrado.Atributos["categoria_clave"] = "auxiliar"
	if !asignacion.Cubre(filtrado) || material.Recurso().Atributos["categoria_clave"] != "sin_filtro" {
		t.Fatal("un filtro cambió el ámbito o mutó el recurso original")
	}
}
