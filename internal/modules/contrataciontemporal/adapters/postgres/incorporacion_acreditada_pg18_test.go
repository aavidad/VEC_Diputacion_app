package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Prueba de contrato Go↔SQL de CT124. Solo se ejecuta dentro del ensayo
// desechable (probar_ct124_incorporacion_acreditada_pg18.sh con
// VEC_CT124_GO=1), con migraciones, fixtures y dobles de las fachadas AD3:
// prueba la transacción CT y la canonización del contexto, no la
// criptografía V3.
func TestIncorporacionAcreditadaPostgreSQLContratoGoSQL(t *testing.T) {
	dsn := os.Getenv("VEC_CT124_PG_DSN")
	if dsn == "" {
		t.Skip("solo en el ensayo PostgreSQL 18 desechable de CT124")
	}
	org, expA, expB := os.Getenv("VEC_CT124_ORG"), os.Getenv("VEC_CT124_EXP_A"), os.Getenv("VEC_CT124_EXP_B")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo, err := NuevoRepositorioOperacionSeguimientoPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	politica := func() ports.PoliticaOperacionSeguimiento {
		ahora := time.Now().UTC().Truncate(time.Microsecond)
		return ports.PoliticaOperacionSeguimiento{DefinicionRef: "vec.contratacion_temporal.reglas", DefinicionVersion: 1, DefinicionHuellaSHA256: strings.Repeat("c", 64),
			MotivoAutorizacion: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_seguimiento_ct", CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)},
			EvaluadaEn: ahora.Add(-time.Second), ValidaHasta: ahora.Add(4 * time.Minute)}
	}
	estado, err := repo.ConsultarIncorporacionAcreditada(ctx, org, expA)
	if err != nil || estado.GINPIX != nil || estado.Centro != nil {
		t.Fatalf("estado inicial: %+v %v", estado, err)
	}

	// ------------------------------------------------ confirmación de GINPIX
	fecha := time.Date(2027, 2, 10, 0, 0, 0, 0, time.UTC)
	g := ports.MaterialConfirmacionGINPIX{OrganizacionRef: org, ExpedienteRef: expA, ActorRef: "per_ct124_go", PerfilRef: "prf_ct124_go", VersionEsperada: 7,
		ClaveIdempotencia: "44444444-4444-4444-8444-444444444444", Datos: domain.DatosConfirmacionGINPIX{Numero: "GX-2027-0077", ConfirmadaEn: fecha}}
	sellos := sellosPrueba(t, ports.OperacionConfirmarGINPIX, "ginpix-1")
	prep, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionConfirmarGINPIX, g, sellos, refsPrueba("ginpix-1"))
	if err != nil || prep.Confirmada || prep.IncorporacionRef == "" {
		t.Fatalf("preparar GINPIX: %+v %v", prep, err)
	}
	instante := time.Now().UTC().Truncate(time.Microsecond)
	siguiente, err := prep.Expediente.ConfirmarGINPIX(7, g.Datos, domain.DatosActuacion{AccionClave: domain.AccionConfirmarGINPIX, ActorRef: g.ActorRef,
		UnidadRef: prep.Expediente.Asignacion.UnidadRef, ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante, FaseDestino: domain.FaseNombramiento,
		EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{g.Datos.DocumentoGINPIX()}})
	if err != nil {
		t.Fatal(err)
	}
	p := politica()
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionConfirmarGINPIX, Material: g, Preparacion: prep, Siguiente: siguiente,
		Politica: p, InstanteEfecto: instante, Accion: domain.AccionConfirmarGINPIX, Finalidad: ports.FinalidadConfirmarGINPIX,
		Audiencia: ports.AudienciaConsumoConfirmacionGINPIXV1, Contexto: ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosPrueba(org, expA),
			Atributos: map[string]string{"version_expediente": "7", "ginpix_numero": "GX-2027-0077", "ginpix_confirmada_en": "2027-02-10",
				"observaciones_huella_sha256": huellaPrueba(""), "incorporacion_ref": prep.IncorporacionRef, "politica_ref": p.DefinicionRef,
				"politica_version": "1", "politica_huella_sha256": p.DefinicionHuellaSHA256, "ambito_idempotencia_hmac": prep.AmbitoIdempotenciaHMAC,
				"huella_peticion_hmac": prep.HuellaPeticionHMAC}}}
	orden.Autorizacion = exportacionPrueba(t, orden, expA, g.ActorRef, g.PerfilRef)
	recibo, err := repo.ConfirmarOperacionSeguimiento(ctx, orden)
	if err != nil || recibo.VersionResultante != 8 || recibo.GINPIXNumero != "GX-2027-0077" || !recibo.RegistradaEn.Equal(instante) {
		t.Fatalf("confirmar GINPIX: %+v %v", recibo, err)
	}
	otra, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionConfirmarGINPIX, g, sellos, refsPrueba("ginpix-otra"))
	if err != nil || !otra.Confirmada || otra.Recibo == nil || otra.Recibo.ReciboRef != recibo.ReciboRef {
		t.Fatalf("recuperación de GINPIX: %+v %v", otra, err)
	}
	segunda := g
	segunda.VersionEsperada, segunda.ClaveIdempotencia = 8, "55555555-5555-4555-8555-555555555555"
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionConfirmarGINPIX, segunda, sellosPrueba(t, ports.OperacionConfirmarGINPIX, "ginpix-2"),
		refsPrueba("ginpix-2")); !errors.Is(err, ports.ErrGINPIXYaConfirmado) {
		t.Fatalf("segunda confirmación: %v", err)
	}
	estado, err = repo.ConsultarIncorporacionAcreditada(ctx, org, expA)
	if err != nil || estado.GINPIX == nil || estado.GINPIX.Numero != "GX-2027-0077" || estado.GINPIX.ConfirmadaEn != "2027-02-10" ||
		estado.GINPIX.ReciboRef != recibo.ReciboRef {
		t.Fatalf("estado con GINPIX: %+v %v", estado, err)
	}

	// ------------------------------------------------ cierre con otro número: rechazado
	otroNumero := ports.MaterialCierreExpediente{OrganizacionRef: org, ExpedienteRef: expA, ActorRef: "per_ct124_go", PerfilRef: "prf_ct124_go", VersionEsperada: 8,
		ClaveIdempotencia: "66666666-6666-4666-8666-666666666666", Datos: domain.DatosCierreExpediente{Condiciones: []string{"cese_registrado", "ginpix_confirmado"},
			GINPIXNumero: "GX-OTRO", GINPIXConfirmadaEn: &fecha}}
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCerrarExpediente, otroNumero, sellosPrueba(t, ports.OperacionCerrarExpediente, "cierre-otro"),
		refsPrueba("cierre-otro")); !errors.Is(err, ports.ErrCierreSinCese) {
		// Sin cese el cierre se rechaza antes que el GINPIX.
		t.Fatalf("cierre sin cese: %v", err)
	}

	// ------------------------------------------------ centro
	// El ensayo lee como superusuario la petición ratificada del centro 520.
	var actor domain.ActorPeticionCentro
	peticion := os.Getenv("VEC_CT124_PETICION")
	if err := json.Unmarshal([]byte(os.Getenv("VEC_CT124_ACTOR")), &actor); err != nil || actor.Validar() != nil || peticion == "" {
		t.Fatalf("actor del centro: %+v %v", actor, err)
	}
	centro, err := NuevoRepositorioIncorporacionCentroPostgreSQL(pool, proveedorCentroPrueba{t: t, actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	filas, err := centro.ListarIncorporacionesCentro(ctx, ports.ConsultaIncorporacionesCentro{Modo: "bandeja", OrganizacionRef: org, Actor: actor})
	if err != nil || len(filas) != 1 || filas[0].ExpedienteRef != expB || !filas[0].AdmiteConfirmacion() || filas[0].ModalidadClave != "sustitucion" {
		t.Fatalf("bandeja del centro: %+v %v", filas, err)
	}
	m := ports.MaterialIncorporacionCentro{Operacion: ports.OperacionConfirmarIncorporacionCentro, ClaveIdempotencia: "77777777-7777-4777-8777-777777777777",
		OrganizacionRef: org, Actor: actor, PeticionRef: peticion, ExpedienteRef: expB, FechaIncorporacion: "2026-09-01",
		Documento: ports.DocumentoIncorporacionCentro{Tipo: "toma_posesion", Referencia: "registro:centro-520:2026/77", SHA256: strings.Repeat("b", 64)},
		Regla: ports.ReglaDocumentoIncorporacion{Referencia: "vec.contratacion_temporal.reglas:1:c21.acreditacion_incorporacion", HuellaSHA256: strings.Repeat("a", 64),
			ModalidadClave: "sustitucion"}}
	rc, err := centro.ConfirmarIncorporacionCentro(ctx, m)
	if err != nil || rc.EstadoLocal != "registrado" || rc.DocumentoTipo != "toma_posesion" {
		t.Fatalf("confirmación del centro: %+v %v", rc, err)
	}
	rc2, err := centro.ConfirmarIncorporacionCentro(ctx, m)
	if err != nil || rc2.EstadoLocal != "replay_confirmado" || rc2.ReciboRef != rc.ReciboRef {
		t.Fatalf("repetición del centro: %+v %v", rc2, err)
	}
	cambiado := m
	cambiado.FechaIncorporacion = "2026-09-02"
	if _, err := centro.ConfirmarIncorporacionCentro(ctx, cambiado); !errors.Is(err, ports.ErrClaveIncorporacionCentroUsada) {
		t.Fatalf("clave del centro con otro contenido: %v", err)
	}
	otraClave := m
	otraClave.ClaveIdempotencia = "88888888-8888-4888-8888-888888888888"
	if _, err := centro.ConfirmarIncorporacionCentro(ctx, otraClave); !errors.Is(err, ports.ErrIncorporacionCentroNoAdmitida) {
		t.Fatalf("segunda confirmación del centro: %v", err)
	}
	estado, err = repo.ConsultarIncorporacionAcreditada(ctx, org, expB)
	if err != nil || estado.Centro == nil || estado.Centro.FechaIncorporacion != "2026-09-01" || estado.Centro.DocumentoSHA256 != strings.Repeat("b", 64) {
		t.Fatalf("RRHH ve la confirmación del centro: %+v %v", estado, err)
	}
}

// proveedorCentroPrueba compone el material con la huella de contexto que
// calcula el dominio VEC: la prueba coteja la canonización Go frente a CT124.
type proveedorCentroPrueba struct {
	t     *testing.T
	actor domain.ActorPeticionCentro
}

func (p proveedorCentroPrueba) AutorizarIncorporacionCentro(_ context.Context, accion string, recurso vd.RecursoAutorizable) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.t.Helper()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	decisionRef := "decision:go:centro:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	decision, _ := json.Marshal(map[string]string{"decision_ref": decisionRef, "accion": accion, "finalidad": ports.FinalidadIncorporacionCentro,
		"modulo_id": ports.ModuloContratacion, "tipo_recurso": recurso.Tipo, "recurso_ref": recurso.Referencia, "principal_id": p.actor.ActorRef,
		"perfil_activo_ref": p.actor.PerfilRef, "contexto_recurso_huella_sha256": huella})
	hd := sha256.Sum256(decision)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, hex.EncodeToString(hd[:]), strings.Repeat("e", 64), "contexto:go",
		strings.Repeat("f", 64), accion, recurso.Referencia, huella, audienciaPeticionCentro, ahora, ahora.Add(4*time.Second))
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	publica, _, _ := ed25519.GenerateKey(rand.Reader)
	spki, _ := x509.MarshalPKIXPublicKey(publica)
	return vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("c", 600)), resumen, decision, []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
}
