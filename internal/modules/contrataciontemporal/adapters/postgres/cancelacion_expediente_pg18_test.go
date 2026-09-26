package postgres

import (
	"context"
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
)

// Prueba de contrato Go↔SQL de CT122. Solo se ejecuta dentro del ensayo
// desechable (probar_ct122_cancelacion_pg18.sh con VEC_CT122_GO=1), que
// instala las migraciones, la fixture y el doble de la fachada AD3-87: prueba
// la transacción CT, no la criptografía V3.
func TestCancelacionPostgreSQLContratoGoSQL(t *testing.T) {
	dsn := os.Getenv("VEC_CT122_PG_DSN")
	if dsn == "" {
		t.Skip("solo en el ensayo PostgreSQL 18 desechable de CT122")
	}
	org := os.Getenv("VEC_CT122_ORG")
	expAsignacion, expSolicitud, expOtra, expFiscalizado := os.Getenv("VEC_CT122_EXP_ASIGNACION"), os.Getenv("VEC_CT122_EXP_SOLICITUD"),
		os.Getenv("VEC_CT122_EXP_OTRA"), os.Getenv("VEC_CT122_EXP_FISCALIZADO")
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
	fases := []domain.ClaveFase{"solicitud", "asignacion_unidad", "informe_juridico"}
	material := func(exp string, version uint64, clave string, canal domain.CanalCancelacion, motivo domain.ClaveCatalogo, obs string) ports.MaterialCancelacion {
		return ports.MaterialCancelacion{OrganizacionRef: org, ExpedienteRef: exp, ActorRef: "per_antonio_reyes_alvarez", PerfilRef: "prf_rrhh_ct122",
			VersionEsperada: version, ClaveIdempotencia: clave, Datos: domain.DatosCancelacion{MotivoClave: motivo, Observaciones: obs, Canal: canal,
				FasesAdmitidas: append([]domain.ClaveFase(nil), fases...)}}
	}
	confirmar := func(m ports.MaterialCancelacion, semilla string) (ports.PreparacionOperacionSeguimiento, ports.ReciboOperacionSeguimiento, error) {
		sellos := sellosPrueba(t, ports.OperacionCancelarExpediente, semilla)
		prep, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCancelarExpediente, m, sellos, refsPrueba(semilla))
		if err != nil {
			return prep, ports.ReciboOperacionSeguimiento{}, err
		}
		instante := time.Now().UTC().Truncate(time.Microsecond)
		siguiente, err := prep.Expediente.Cancelar(m.VersionEsperada, m.Datos, domain.DatosActuacion{AccionClave: domain.AccionCancelarExpediente,
			ActorRef: m.ActorRef, UnidadRef: prep.Expediente.UnidadActual(), ReciboRef: prep.Referencias.ReciboRef, RealizadaEn: instante,
			FaseDestino: prep.Expediente.FaseActual, EstadoDestino: domain.EstadoCancelado, Observaciones: m.Datos.Observaciones})
		if err != nil {
			t.Fatalf("cancelar en dominio: %v", err)
		}
		ahora := time.Now().UTC().Truncate(time.Microsecond)
		p := ports.PoliticaOperacionSeguimiento{DefinicionRef: "motivos_cancelacion_contratacion_temporal", DefinicionVersion: 1,
			DefinicionHuellaSHA256: strings.Repeat("c", 64), MotivoAutorizacion: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cancelacion_ct", CatalogoVersion: 1,
				CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("2", 32)},
			EvaluadaEn: ahora.Add(-time.Second), ValidaHasta: ahora.Add(4 * time.Minute)}
		ambitos := map[string]string{"organizacion_ref": org, "expediente_ref": m.ExpedienteRef, "fase_previa": string(prep.Expediente.FaseActual), "estado_previo": "en_curso"}
		if m.Datos.Canal == domain.CanalCancelacionCentro {
			ambitos["centro_ref"] = prep.Expediente.Solicitud.CentroRef
		}
		orden := ports.OrdenConfirmarOperacionSeguimiento{Operacion: ports.OperacionCancelarExpediente, Material: m, Preparacion: prep, Siguiente: siguiente,
			Politica: p, InstanteEfecto: instante, Accion: domain.AccionCancelarExpediente, Finalidad: ports.FinalidadCancelarExpediente,
			Audiencia: ports.AudienciaConsumoCancelacionV1, Contexto: ports.ContextoAutorizadoSeguimiento{Ambitos: ambitos, Atributos: map[string]string{
				"version_expediente": strconv.FormatUint(m.VersionEsperada, 10), "canal": string(m.Datos.Canal),
				"motivo_clave": string(m.Datos.MotivoClave), "fases_admitidas": "solicitud,asignacion_unidad,informe_juridico",
				"observaciones_huella_sha256": huellaPrueba(m.Datos.Observaciones), "politica_ref": p.DefinicionRef, "politica_version": "1",
				"politica_huella_sha256": p.DefinicionHuellaSHA256, "ambito_idempotencia_hmac": prep.AmbitoIdempotenciaHMAC,
				"huella_peticion_hmac": prep.HuellaPeticionHMAC}}}
		orden.Autorizacion = exportacionPrueba(t, orden, m.ExpedienteRef, m.ActorRef, m.PerfilRef)
		recibo, err := repo.ConfirmarOperacionSeguimiento(ctx, orden)
		if err == nil && !recibo.RegistradaEn.Equal(instante) {
			t.Fatalf("instante del recibo: %v", recibo.RegistradaEn)
		}
		return prep, recibo, err
	}

	// RRHH cancela un expediente en asignación de unidad, con observación.
	m := material(expAsignacion, 3, "44444444-4444-4444-8444-444444444444", domain.CanalCancelacionRRHH, "necesidad_desaparecida",
		"El centro ya no necesita el refuerzo")
	_, recibo, err := confirmar(m, "rrhh-1")
	if err != nil || recibo.VersionResultante != 4 || recibo.EstadoResultante != domain.EstadoCancelado ||
		recibo.FaseResultante != "asignacion_unidad" || recibo.MotivoClave != "necesidad_desaparecida" {
		t.Fatalf("confirmar cancelación: %+v %v", recibo, err)
	}
	// La misma intención recupera el recibo original sin otra escritura.
	otra, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCancelarExpediente, m, sellosPrueba(t, ports.OperacionCancelarExpediente, "rrhh-1"), refsPrueba("rrhh-otra"))
	if err != nil || !otra.Confirmada || otra.Recibo == nil || otra.Recibo.ReciboRef != recibo.ReciboRef || otra.Expediente.EstadoActual != domain.EstadoCancelado {
		t.Fatalf("recuperación: %+v %v", otra, err)
	}
	cambiada := m
	cambiada.Datos.MotivoClave = "error_solicitud"
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCancelarExpediente, cambiada, sellosPrueba(t, ports.OperacionCancelarExpediente, "rrhh-1"), refsPrueba("rrhh-c")); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
		t.Fatalf("clave reutilizada: %v", err)
	}
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCancelarExpediente, m, sellosPrueba(t, ports.OperacionCancelarExpediente, "rrhh-2"), refsPrueba("rrhh-2")); !errors.Is(err, ports.ErrCancelacionYaRegistrada) {
		t.Fatalf("segunda cancelación: %v", err)
	}
	estado, err := repo.ConsultarEstadoCancelacion(ctx, org, expAsignacion)
	if err != nil || estado.Cancelacion == nil || estado.Cancelacion.MotivoClave != "necesidad_desaparecida" || estado.Cancelacion.Canal != domain.CanalCancelacionRRHH ||
		estado.Cancelacion.FasePrevia != "asignacion_unidad" || estado.Cancelacion.ReciboRef != recibo.ReciboRef ||
		estado.Cancelacion.Observaciones != "El centro ya no necesita el refuerzo" {
		t.Fatalf("estado de la cancelación: %+v %v", estado, err)
	}

	// El centro cancela su petición en solicitud: el ámbito lleva su centro.
	_, reciboCentro, err := confirmar(material(expSolicitud, 1, "55555555-5555-4555-8555-555555555555", domain.CanalCancelacionCentro, "desistimiento_centro", ""), "centro-1")
	if err != nil || reciboCentro.FaseResultante != "solicitud" || reciboCentro.EstadoResultante != domain.EstadoCancelado {
		t.Fatalf("cancelación del centro: %+v %v", reciboCentro, err)
	}
	if e, err := repo.ConsultarEstadoCancelacion(ctx, org, expOtra); err != nil || e.Cancelacion != nil {
		t.Fatalf("expediente sin cancelar: %+v %v", e, err)
	}

	// La regla no admite la fase del expediente.
	soloInforme := material(expOtra, 1, "66666666-6666-4666-8666-666666666666", domain.CanalCancelacionRRHH, "error_solicitud", "")
	soloInforme.Datos.FasesAdmitidas = []domain.ClaveFase{"informe_juridico"}
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCancelarExpediente, soloInforme, sellosPrueba(t, ports.OperacionCancelarExpediente, "fase"), refsPrueba("fase")); !errors.Is(err, ports.ErrCancelacionNoAdmitida) {
		t.Fatalf("fase no admitida: %v", err)
	}
	// Tras la fiscalización no se puede cancelar, aunque la regla admita su fase.
	fiscalizado := material(expFiscalizado, 7, "77777777-7777-4777-8777-777777777777", domain.CanalCancelacionRRHH, "error_solicitud", "")
	fiscalizado.Datos.FasesAdmitidas = []domain.ClaveFase{"nombramiento", "fiscalizacion"}
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCancelarExpediente, fiscalizado, sellosPrueba(t, ports.OperacionCancelarExpediente, "fisc"), refsPrueba("fisc")); !errors.Is(err, ports.ErrCancelacionTrasFiscalizacion) {
		t.Fatalf("tras fiscalización: %v", err)
	}
	// Versión desfasada.
	if _, err := repo.PrepararOperacionSeguimiento(ctx, ports.OperacionCancelarExpediente, material(expOtra, 9, "88888888-8888-4888-8888-888888888888", domain.CanalCancelacionRRHH, "error_solicitud", ""), sellosPrueba(t, ports.OperacionCancelarExpediente, "ver"), refsPrueba("ver")); !errors.Is(err, domain.ErrVersionEnConflicto) {
		t.Fatalf("versión desfasada: %v", err)
	}
}
