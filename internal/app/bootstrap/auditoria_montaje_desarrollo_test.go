package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autoridadRutaAuditoriaSintetica struct{}

func (autoridadRutaAuditoriaSintetica) AutorizarRutaExacta(context.Context, string) error { return nil }

type registradorRutaAuditoriaSintetica struct{}

func (registradorRutaAuditoriaSintetica) RegistrarAuditoriaFronteraRutaExacta(context.Context, vecports.OrdenAuditoriaFronteraRutaExacta) error {
	return nil
}

func TestAuditoriaNoReestablecePerfilRevocadoORestringido(t *testing.T) {
	soporte, _, _ := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := soporte.reloj.Ahora()
	base := soporte.contexto.Resultado.Contexto
	semilla, err := instantaneaAuditoriaConsultaNominalDesarrollo(
		base.Principal.ID, base.PerfilActivoRef, "ct", "expediente:ct:sintetico:001",
		"revision_administrativa_auditoria_rrhh", ahora)
	if err != nil || semilla.Validar() != nil || !instantaneaAuditoriaConsultaVigenteExacta(semilla, semilla, ahora) {
		t.Fatalf("semilla nominal inválida: %v", err)
	}
	if !instantaneaInicialAuditoriaConsultaExacta(semilla, semilla) {
		t.Fatal("alta inicial exacta rechazada")
	}
	competidora := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	competidora.AsignacionPerfil.Version = 2
	if competidora.Validar() != nil || instantaneaInicialAuditoriaConsultaExacta(competidora, semilla) {
		t.Fatal("una asignación aparecida durante la preparación se aceptó como alta inicial")
	}
	revocada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	revocada.AsignacionPerfil.Estado = vecdomain.EstadoAsignacionPerfilRevocada
	revocada.AsignacionPerfil.RevocadaPor = "seguridad:desarrollo"
	revocada.AsignacionPerfil.RevocacionRef = "revocacion:auditoria:prueba"
	revocada.AsignacionPerfil.RevocadaEn = ahora
	if revocada.Validar() != nil || instantaneaAuditoriaConsultaVigenteExacta(revocada, semilla, ahora) {
		t.Fatal("una asignación revocada se consideró reiniciable")
	}
	restringida := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	restringida.AsignacionPerfil.Ambitos[0].Valores[0] = "expediente:ct:otro:001"
	if restringida.Validar() != nil || instantaneaAuditoriaConsultaVigenteExacta(restringida, semilla, ahora) {
		t.Fatal("una asignación de ámbito ajeno se consideró exacta")
	}
	concesionAjena := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	concesionAjena.VersionRol.Concesiones[0].Finalidades = []string{"otra_revision"}
	if concesionAjena.Validar() != nil || instantaneaAuditoriaConsultaVigenteExacta(concesionAjena, semilla, ahora) {
		t.Fatal("una concesión distinta se consideró exacta")
	}
}

func TestRaizExactaAuditoriaSirveOpcionesYDeniegaFuenteAjena(t *testing.T) {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	escenario := nuevoEscenarioMaterialRutasDietasPrueba(t, "dietas.ruta.catalogo.consultar", ahora)
	datos, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	identidadCT := &identidadAuditoriaConsultaPrueba{resuelta: auditoria.IdentidadResuelta{
		Vinculo: datos.VinculoAutenticacionActor, Resultado: escenario.resultado, Correlacion: datos.Correlacion}}
	identidadBolsa := &identidadAuditoriaConsultaPrueba{err: auditoria.ErrDenegada}
	opciones := auditoria.Opciones{FinalidadRef: "revision_administrativa_auditoria_rrhh",
		MotivoRef: escenario.motivo.Referencia(), PermisoRequerido: auditoria.AccionConsultar,
		Fuentes: []string{"ct", "bolsa"}, Motivo: escenario.motivo}
	rutas, err := nuevasRutasAuditoriaConsultaConIdentidadesRRHH(dependenciasIdentidadAuditoriaConsultaRRHH{
		PoolCT: &pgxpool.Pool{}, PoolBolsa: &pgxpool.Pool{},
		EmisorCT: &emisorAuditoriaConsultaPrueba{}, EmisorBolsa: &emisorAuditoriaConsultaPrueba{},
		IdentidadOpciones: identidadCT, IdentidadCT: identidadCT, IdentidadBolsa: identidadBolsa,
		Opciones: &opcionesAuditoriaConsultaPrueba{opciones: opciones},
	})
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := vechttp.NewHandlerSoloRutasExactas(rutas, autoridadRutaAuditoriaSintetica{}, registradorRutaAuditoriaSintetica{})
	if err != nil {
		t.Fatal(err)
	}
	get := httptest.NewRecorder()
	raiz.ServeHTTP(get, httptest.NewRequest(http.MethodGet, auditoria.RutaOpciones, nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), opciones.FinalidadRef) {
		t.Fatalf("GET opciones en raíz = %d %s", get.Code, get.Body.String())
	}
	cuerpo, _ := json.Marshal(map[string]any{"fuente": "bolsa", "expediente_ref": "expediente:opaco:123",
		"desde": ahora.Add(-time.Hour).Format(time.RFC3339Nano), "hasta": ahora.Add(time.Hour).Format(time.RFC3339Nano),
		"limite": 1, "finalidad_ref": opciones.FinalidadRef, "motivo_ref": opciones.MotivoRef})
	peticion := httptest.NewRequest(http.MethodPost, auditoria.RutaConsulta, strings.NewReader(string(cuerpo)))
	peticion.Header.Set("Content-Type", "application/json")
	post := httptest.NewRecorder()
	raiz.ServeHTTP(post, peticion)
	if post.Code != http.StatusForbidden {
		t.Fatalf("fuente Bolsa con identidad CT en raíz = HTTP %d", post.Code)
	}
}
