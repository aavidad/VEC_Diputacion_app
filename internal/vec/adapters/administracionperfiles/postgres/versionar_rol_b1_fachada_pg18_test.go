package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// TestVersionarRolBolsaFachadaYAdaptadorRealesPostgreSQL18 recorre con el
// adaptador Go real y las fachadas SQL reales la propuesta del ADMIN 1, su
// replay, el cierre del ADMIN 2 y su replay. El decodificador del adaptador
// recibe exactamente lo que devuelve PostgreSQL (por ejemplo, instantes
// «+00:00» de jsonb), que los dobles en memoria no reproducen: así se cazó el
// 503 «sql_respuesta_invalida» del clon.
//
// Sólo corre contra una base desechable tras los actos ADMIN (dos ADMIN v9 y el
// catálogo B1 cargado). El DSN debe ser de un superusuario de esa copia. Todo
// ocurre en una transacción que se revierte; cada operación del adaptador va en
// un punto de guardado dentro de ella.
func TestVersionarRolBolsaFachadaYAdaptadorRealesPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_AUT62_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18 desechable tras los actos ADMIN con AUT62/AUT63")
	}
	dsn := os.Getenv("VEC_AUT62_PG_DSN")
	if dsn == "" {
		t.Fatal("falta VEC_AUT62_PG_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conexión: %v", err)
	}
	defer pool.Close()
	plan, intencion := planB1RealConAsignaciones(ctx, t, pool)
	admins := adminsV9Reales(ctx, t, pool)
	fuente, err := nuevaFuenteCatalogoAcciones(ctx, pool)
	if err != nil {
		t.Fatal("fuente:", err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	// Sólo dentro de esta transacción, que se revierte: la emisión atestada V3
	// real necesita claves del conjunto que una base desechable no tiene.
	for _, s := range []string{consumoV3SustituidoSQL("consumir_version_rol_bolsa_v3_atestada"),
		acreditacionSustituidaSQL("acreditar_version_rol_bolsa_v1"), revalidarV3SustituidaEnTransaccion} {
		if _, err := tx.Exec(ctx, s); err != nil {
			t.Fatal("sustitución V3:", err)
		}
	}
	emisor := &emisorV3SustituidoPrueba{t: t}
	a := &AutoridadVersionarRolBolsa{pool: poolSobreTxPrueba{tx: tx}, catalogo: fuente, emisor: emisor,
		reloj: relojSistemaPrueba{}}

	hp, err := plan.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	proponente := admins[0]
	solicitud := domain.SolicitudPropuestaVersionarRolBolsa{OperacionRef: "propuesta_admin:" + hexAleatorio(t),
		Actor: proponente.actor, Evidencia: proponente.evidencia, InstantaneaAutorizacion: proponente.instantanea,
		Intencion: intencion, HuellaPlanEsperada: hp, CorrelacionRef: "correlacion_" + hexAleatorio(t)}
	orden := domain.OrdenPropuestaVersionarRolBolsa{Solicitud: solicitud,
		Material: domain.MaterialPropuestaVersionarRolBolsa{OperacionRef: solicitud.OperacionRef,
			ProponentePersonaRef: proponente.actor.PersonaRef, PerfilActivoRef: proponente.actor.PerfilActivoRef,
			AsignacionPerfilRef: proponente.instantanea.AsignacionPerfil.Referencia(), Plan: plan}}
	if err := orden.Validar(); err != nil {
		t.Fatal("orden de prueba:", err)
	}
	emisor.motivo = plan.Motivo
	propuesta, err := a.ProponerVersionarRolBolsa(ctx, orden)
	if err != nil {
		t.Fatalf("propuesta: %v", err)
	}
	if propuesta.Replay || propuesta.Propuesta.CaducaEn.Location() != time.UTC {
		t.Fatalf("propuesta: replay=%v caduca_en=%v", propuesta.Replay, propuesta.Propuesta.CaducaEn)
	}
	repetida, err := a.ProponerVersionarRolBolsa(ctx, orden)
	if err != nil || !repetida.Replay || repetida.Propuesta.HuellaSHA256 != propuesta.Propuesta.HuellaSHA256 ||
		!repetida.Propuesta.CaducaEn.Equal(propuesta.Propuesta.CaducaEn) {
		t.Fatalf("replay de la propuesta: %+v %v", repetida.Replay, err)
	}

	aprobador := admins[1]
	cierre := domain.SolicitudCierreVersionarRolBolsa{OperacionRef: "cierre_admin:" + hexAleatorio(t),
		PropuestaRef: orden.Material.OperacionRef, PropuestaHuellaSHA256: propuesta.Propuesta.HuellaSHA256,
		Aprobador: aprobador.actor, Evidencia: aprobador.evidencia, InstantaneaAutorizacion: aprobador.instantanea,
		Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: plan.Motivo,
		CorrelacionRef: "correlacion_" + hexAleatorio(t)}
	if err := cierre.Validar(); err != nil {
		t.Fatal("cierre de prueba:", err)
	}
	cerrado, err := a.CerrarVersionarRolBolsa(ctx, cierre)
	if err != nil || cerrado.Recibo == nil {
		t.Fatalf("cierre: %v", err)
	}
	if cerrado.Recibo.VersionRol.Referencia() != plan.VersionRolObjetivoRef ||
		len(cerrado.Recibo.Asignaciones) != len(plan.Asignaciones) {
		t.Fatalf("recibo: versión %q, %d asignaciones", cerrado.Recibo.VersionRol.Referencia(),
			len(cerrado.Recibo.Asignaciones))
	}
	otra, err := a.CerrarVersionarRolBolsa(ctx, cierre)
	if err != nil || otra.Recibo == nil || otra.Recibo.ReciboRef != cerrado.Recibo.ReciboRef ||
		!otra.ConfirmadoEn.Equal(cerrado.ConfirmadoEn) {
		t.Fatalf("replay del cierre: %v", err)
	}
}

// emisorV3SustituidoPrueba entrega el material V3 para AUT63 sin firmarlo; la
// sustitución SQL de la prueba lo interpreta.
type emisorV3SustituidoPrueba struct {
	t      *testing.T
	motivo domain.ReferenciaEntradaCatalogo
}

func (e *emisorV3SustituidoPrueba) EmitirVersionarRolBolsa(_ context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion,
	efecto Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Helper()
	recurso, err := RecursoVersionarRolBolsa(efecto, instantanea.AsignacionPerfil)
	if err != nil {
		e.t.Fatal(err)
	}
	h, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		e.t.Fatal(err)
	}
	return exportacionV3SustituidaPrueba(e.t, actor, evidencia, instantanea.AsignacionPerfil, efecto,
		recurso.Referencia, h, e.motivo)
}
