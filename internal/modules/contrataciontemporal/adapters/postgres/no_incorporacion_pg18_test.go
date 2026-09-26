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

// Prueba de contrato Go↔SQL de la no incorporación de CT124. Solo se ejecuta
// dentro del ensayo desechable (probar_ct124_no_incorporacion_pg18.sh con
// VEC_CT124_GO=1), con migraciones, fixtures y dobles de las fachadas AD3:
// prueba la proyección y el contexto canónicos, la publicación a Bolsa y el
// antecedente de la continuación, no la criptografía V3.
func TestNoIncorporacionPostgreSQLContratoGoSQL(t *testing.T) {
	dsn := os.Getenv("VEC_CT124NI_PG_DSN")
	if dsn == "" {
		t.Skip("solo en el ensayo PostgreSQL 18 desechable de la no incorporación")
	}
	org, exp := os.Getenv("VEC_CT124NI_ORG"), os.Getenv("VEC_CT124NI_EXP")
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
	m := ports.MaterialNoIncorporacion{OrganizacionRef: org, ExpedienteRef: exp, ActorRef: "per_ct124_go", PerfilRef: "prf_ct124_go", VersionEsperada: 7,
		ClaveIdempotencia: "44444444-4444-4444-8444-444444444442", Datos: domain.DatosNoIncorporacion{MotivoClave: "no_presentado",
			ConsecuenciaClave: "b24.sancion.baja_llamamiento_directo", ResolucionRef: "resolucion:rrhh:2026/0142", ResolucionSHA256: strings.Repeat("b", 64),
			ResueltaPor: "per_ct124_segunda", SegundaPersona: true, FechaNotificacion: time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)}}
	sellos := sellosPrueba(t, ports.OperacionRegistrarNoIncorporacion, "ni-go")
	prep, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionRegistrarNoIncorporacion, m, sellos, refsPrueba("ni-go"))
	if err != nil || prep.Confirmada || prep.AceptacionRef == "" {
		t.Fatalf("preparar: %+v %v", prep, err)
	}
	instante := time.Now().UTC().Truncate(time.Microsecond)
	siguiente, err := prep.Expediente.RegistrarNoIncorporacion(7, m.Datos, domain.DatosActuacion{AccionClave: domain.AccionRegistrarNoIncorporacion,
		ActorRef: m.ActorRef, UnidadRef: prep.Expediente.Asignacion.UnidadRef, ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante,
		FaseDestino: domain.FaseFiscalizacion, EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{m.Datos.ResolucionRef}})
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	p := ports.PoliticaOperacionSeguimiento{DefinicionRef: "vec.contratacion_temporal.reglas", DefinicionVersion: 1, DefinicionHuellaSHA256: strings.Repeat("c", 64),
		MotivoAutorizacion: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_seguimiento_ct", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)},
		EvaluadaEn: ahora.Add(-time.Second), ValidaHasta: ahora.Add(4 * time.Minute)}
	orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionRegistrarNoIncorporacion, Material: m, Preparacion: prep, Siguiente: siguiente,
		Politica: p, InstanteEfecto: instante, Accion: domain.AccionRegistrarNoIncorporacion, Finalidad: ports.FinalidadRegistrarNoIncorporacion,
		Audiencia: ports.AudienciaConsumoNoIncorporacionV1, Contexto: ports.ContextoAutorizadoSeguimiento{Ambitos: ambitosPrueba(org, exp),
			Atributos: map[string]string{"version_expediente": "7", "motivo_clave": "no_presentado", "consecuencia_clave": "b24.sancion.baja_llamamiento_directo",
				"resolucion_ref": "resolucion:rrhh:2026/0142", "resolucion_sha256": strings.Repeat("b", 64), "resuelta_por": "per_ct124_segunda",
				"segunda_persona": "si", "fecha_notificacion": "2026-09-20", "observaciones_huella_sha256": huellaPrueba(""), "aceptacion_ref": prep.AceptacionRef,
				"politica_ref": p.DefinicionRef, "politica_version": "1", "politica_huella_sha256": p.DefinicionHuellaSHA256,
				"ambito_idempotencia_hmac": prep.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": prep.HuellaPeticionHMAC}}}
	orden.Autorizacion = exportacionPrueba(t, orden, exp, m.ActorRef, m.PerfilRef)
	recibo, err := repo.ConfirmarOperacionSeguimiento(ctx, orden)
	if err != nil || recibo.VersionResultante != 8 || recibo.FaseResultante != domain.FaseFiscalizacion || !recibo.RegistradaEn.Equal(instante) {
		t.Fatalf("confirmar: %+v %v", recibo, err)
	}
	otra, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionRegistrarNoIncorporacion, m, sellos, refsPrueba("ni-otra"))
	if err != nil || !otra.Confirmada || otra.Recibo == nil || otra.Recibo.ReciboRef != recibo.ReciboRef || otra.AceptacionRef != prep.AceptacionRef {
		t.Fatalf("recuperación: %+v %v", otra, err)
	}
	segunda := m
	segunda.VersionEsperada, segunda.ClaveIdempotencia = 8, "55555555-5555-4555-8555-555555555552"
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionRegistrarNoIncorporacion, segunda, sellosPrueba(t, ports.OperacionRegistrarNoIncorporacion, "ni-2"),
		refsPrueba("ni-2")); !errors.Is(err, ports.ErrNoIncorporacionExistente) {
		t.Fatalf("segunda no incorporación: %v", err)
	}
	estado, err := repo.ConsultarIncorporacionAcreditada(ctx, org, exp)
	if err != nil || estado.NoIncorporacion == nil || estado.NoIncorporacion.MotivoClave != "no_presentado" || estado.NoIncorporacion.ReciboRef != recibo.ReciboRef {
		t.Fatalf("estado: %+v %v", estado, err)
	}

	// ------------------------------------------------ publicación a Bolsa
	lector, err := NuevoLectorPublicacionContratosBolsaPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	eventos, err := lector.LeerNoIncorporacionesBolsa(ctx, ports.CursorPublicacionContratosBolsa{}, 10)
	if err != nil || len(eventos) != 1 || !strings.HasPrefix(eventos[0].EventoRef, "evento:ct:no-incorporacion-bolsa:") || eventos[0].OrigenRef != recibo.EventoRef {
		t.Fatalf("publicación: %+v %v", eventos, err)
	}
	// El relevo lo entrega a la bandeja de Bolsa en su propia prueba.
	publicado, _ := json.Marshal(map[string]any{"contenido": string(eventos[0].Contenido), "huella": eventos[0].HuellaSHA256,
		"posicion": eventos[0].OrigenPosicion, "creada": eventos[0].OrigenCreadaEn})
	if err := os.WriteFile(os.Getenv("VEC_CT124NI_EVENTO"), publicado, 0o600); err != nil {
		t.Fatal(err)
	}

	// ------------------------------------------------ antecedente de la continuación
	registro, err := NuevoRegistroContinuacionLlamamientoPostgreSQL(pool, proveedorContinuacionNoIncorporacionPrueba{t: t})
	if err != nil {
		t.Fatal(err)
	}
	// La intención deriva del ámbito de idempotencia (CT124).
	suma := sha256.Sum256([]byte("intencion\x1f" + prep.AmbitoIdempotenciaHMAC))
	intencion := "intencion:ct124:" + hex.EncodeToString(suma[:])
	s := ports.SolicitudContinuarLlamamiento{ClaveIdempotencia: "66666666-6666-4666-8666-666666666662", OrganizacionRef: org, ExpedienteRef: exp,
		ResolucionRef: prep.AceptacionRef, IntencionRef: intencion}
	a, err := registro.LeerAntecedente(ctx, s)
	if err != nil || !a.EsNoIncorporacion() || a.NoIncorporacion.VersionResultante != 8 || a.NoIncorporacion.ReciboRef != recibo.ReciboRef {
		t.Fatalf("antecedente: %+v %v", a, err)
	}
}

// proveedorContinuacionNoIncorporacionPrueba compone el material con la huella de contexto
// del dominio VEC para el permiso de continuación (doble de AD3 en SQL).
type proveedorContinuacionNoIncorporacionPrueba struct{ t *testing.T }

func (p proveedorContinuacionNoIncorporacionPrueba) AutorizarContinuacionLlamamiento(_ context.Context, m ports.MaterialContinuacionLlamamiento) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.t.Helper()
	recurso, err := RecursoContinuacionLlamamiento(m)
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	decisionRef := "decision:go:continuacion:" + strconv.FormatInt(time.Now().UnixNano(), 10)
	decision, _ := json.Marshal(map[string]string{"decision_ref": decisionRef, "accion": AccionContinuacionLlamamiento, "finalidad": "gestionar_contratacion_temporal",
		"modulo_id": ports.ModuloContratacion, "tipo_recurso": recurso.Tipo, "recurso_ref": recurso.Referencia, "principal_id": "per_ct124_go",
		"perfil_activo_ref": "prf_ct124_go", "contexto_recurso_huella_sha256": huella})
	hd := sha256.Sum256(decision)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vp.NuevoResumenCapacidadAtestacionAutorizacionV3(decisionRef, hex.EncodeToString(hd[:]), strings.Repeat("e", 64), "contexto:go",
		strings.Repeat("f", 64), AccionContinuacionLlamamiento, recurso.Referencia, huella, AudienciaRegistroComunicacionLlamamiento, ahora, ahora.Add(4*time.Second))
	if err != nil {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	publica, _, _ := ed25519.GenerateKey(rand.Reader)
	spki, _ := x509.MarshalPKIXPublicKey(publica)
	return vp.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("c", 600)), resumen, decision, []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
}
