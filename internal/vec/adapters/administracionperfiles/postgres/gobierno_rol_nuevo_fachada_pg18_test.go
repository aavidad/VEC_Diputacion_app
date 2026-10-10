package postgres

import (
	"context"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// TestGobiernoRolNuevoFachadaYAdaptadorRealesPostgreSQL18 recorre con el
// adaptador Go real y las fachadas SQL reales de AUT60 la propuesta de un
// RolID nuevo por el ADMIN 1, su replay, el cierre del ADMIN 2 y su replay. El
// decodificador recibe lo que devuelve PostgreSQL (instantes «+00:00» de
// jsonb), que los dobles en memoria no reproducen.
//
// Sólo corre contra una base desechable tras los actos ADMIN (dos ADMIN v9 con
// las concesiones de gobierno y un catálogo de acciones con una entrada
// ordinaria). El DSN debe ser de un superusuario de esa copia. Todo ocurre en
// una transacción que se revierte, con sólo el consumo V3 sustituido.
func TestGobiernoRolNuevoFachadaYAdaptadorRealesPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_AUT60_PG_DESECHABLE") != "si" {
		t.Skip("requiere PostgreSQL 18 desechable tras los actos ADMIN con AUT60")
	}
	dsn := os.Getenv("VEC_AUT60_PG_DSN")
	if dsn == "" {
		t.Fatal("falta VEC_AUT60_PG_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("conexión: %v", err)
	}
	defer pool.Close()
	admins := adminsV9Reales(ctx, t, pool)
	fuente, err := nuevaFuenteCatalogoAcciones(ctx, pool)
	if err != nil {
		t.Fatal("fuente:", err)
	}
	proponente, aprobador := admins[0], admins[1]
	intencion, plan := planRolNuevoReal(ctx, t, pool, fuente, proponente.actor.PersonaRef)

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	for _, s := range []string{consumoV3SustituidoSQL("consumir_gobierno_rol_nuevo_v3_atestada"),
		acreditacionSustituidaSQL("acreditar_gobierno_rol_nuevo_v1"), revalidarV3SustituidaEnTransaccion} {
		if _, err := tx.Exec(ctx, s); err != nil {
			t.Fatal("sustitución V3:", err)
		}
	}
	emisor := &emisorGobiernoV3SustituidoPrueba{t: t, motivo: plan.Motivo}
	a := &AutoridadGobiernoRolNuevo{pool: poolSobreTxPrueba{tx: tx}, fuente: fuente, emisor: emisor,
		reloj: relojSistemaPrueba{}}

	hp, err := plan.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	solicitud := domain.SolicitudPropuestaGobiernoPerfil{OperacionRef: "propuesta_admin:" + hexAleatorio(t),
		Actor: proponente.actor, Evidencia: proponente.evidencia, InstantaneaAutorizacion: proponente.instantanea,
		Intencion: intencion, HuellaPlanEsperada: hp, CorrelacionRef: "correlacion_" + hexAleatorio(t)}
	orden := domain.OrdenPropuestaGobiernoPerfil{Solicitud: solicitud,
		Material: domain.MaterialPropuestaGobiernoPerfil{OperacionRef: solicitud.OperacionRef,
			ProponentePersonaRef: proponente.actor.PersonaRef, PerfilActivoRef: proponente.actor.PerfilActivoRef,
			AsignacionPerfilRef: proponente.instantanea.AsignacionPerfil.Referencia(), Plan: plan}}
	if err := orden.Validar(); err != nil {
		t.Fatal("orden de prueba:", err)
	}
	propuesta, err := a.ProponerGobiernoRolNuevoRecuperable(ctx, orden)
	if err != nil {
		t.Fatalf("propuesta: %v", err)
	}
	if propuesta.Replay || propuesta.Propuesta.CaducaEn.Location() != time.UTC {
		t.Fatalf("propuesta: replay=%v caduca_en=%v", propuesta.Replay, propuesta.Propuesta.CaducaEn)
	}
	repetida, err := a.ProponerGobiernoRolNuevoRecuperable(ctx, orden)
	if err != nil || !repetida.Replay || repetida.Propuesta.HuellaSHA256 != propuesta.Propuesta.HuellaSHA256 {
		t.Fatalf("replay de la propuesta: %v %v", repetida.Replay, err)
	}

	cierre := domain.SolicitudCierreGobiernoPerfil{OperacionRef: "cierre_admin:" + hexAleatorio(t),
		PropuestaRef: orden.Material.OperacionRef, PropuestaHuellaSHA256: propuesta.Propuesta.HuellaSHA256,
		ProponentePersonaRef: proponente.actor.PersonaRef, VersionRolObjetivoRef: plan.VersionRolObjetivoRef,
		Aprobador: aprobador.actor, Evidencia: aprobador.evidencia, InstantaneaAutorizacion: aprobador.instantanea,
		Decision: domain.DecisionAprobarPropuestaPerfil, Motivo: plan.Motivo,
		CorrelacionRef: "correlacion_" + hexAleatorio(t)}
	if err := cierre.Validar(); err != nil {
		t.Fatal("cierre de prueba:", err)
	}
	cerrado, err := a.CerrarGobiernoPerfil(ctx, cierre)
	if err != nil || cerrado.Recibo == nil || cerrado.ConfirmadoEn.Location() != time.UTC {
		t.Fatalf("cierre: %v", err)
	}
	otra, err := a.CerrarGobiernoPerfil(ctx, cierre)
	if err != nil || !otra.ConfirmadoEn.Equal(cerrado.ConfirmadoEn) {
		t.Fatalf("replay del cierre: %v", err)
	}
}

// planRolNuevoReal prepara con el dominio Go, sobre la cabeza real del
// catálogo, la creación de un RolID nuevo con su única entrada ordinaria.
func planRolNuevoReal(ctx context.Context, t *testing.T, pool *pgxpool.Pool, fuente *FuenteCatalogoAcciones,
	actorPersonaRef string) (domain.SolicitudPlanGobiernoPerfil, domain.PlanGobiernoPerfil) {
	t.Helper()
	var ref, huella string
	var version int
	if err := pool.QueryRow(ctx, `SELECT catalogo_ref,version,huella_sha256
	 FROM vec_autorizacion.cabeza_catalogo_acciones_admin_v1 ORDER BY catalogo_ref LIMIT 1`).Scan(&ref, &version, &huella); err != nil {
		t.Fatal("cabeza del catálogo:", err)
	}
	catalogo, err := fuente.ObtenerCatalogoAccionesAdministracionV1(ctx, ref, version, huella)
	if err != nil {
		t.Fatal("catálogo:", err)
	}
	var entrada *domain.EntradaAccionAdministracionV1
	for i := range catalogo.Entradas {
		if catalogo.Entradas[i].ClaseControl == string(domain.ClaseControlPerfilOrdinario) {
			entrada = &catalogo.Entradas[i]
			break
		}
	}
	if entrada == nil {
		t.Fatal("el catálogo no tiene una entrada ordinaria")
	}
	he, err := entrada.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	intencion := domain.SolicitudPlanGobiernoPerfil{Operacion: domain.OperacionCrearPerfilGobernado,
		Publicacion: &domain.PropuestaPerfilAdministracionV1{CatalogoRef: catalogo.Referencia,
			CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: huella,
			RolPropuesto: domain.VersionRol{RolID: "prueba_utc_" + hexAleatorio(t)[:12], Version: 1,
				Nombre: "Prueba de fechas UTC", Estado: domain.EstadoVersionRolPublicada,
				Concesiones: []domain.ConcesionRol{entrada.Concesion}},
			Selecciones: []domain.SeleccionAccionAdministracionV1{{EntradaRef: entrada.Referencia,
				EntradaVersion: entrada.Version, EntradaHuellaSHA256: he}}},
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos.admin.prueba", CatalogoVersion: 1,
			CatalogoHuellaSHA256: hex.EncodeToString(make([]byte, 31)) + "01",
			EntradaClave:         "motivo_" + hex.EncodeToString([]byte("prueba-aut60-utc"))}}
	plan, _, err := domain.PrepararPlanGobiernoRolNuevoDesdeCatalogo(catalogo, intencion,
		time.Now().UTC().Truncate(time.Microsecond), actorPersonaRef)
	if err != nil {
		t.Fatal("plan:", err)
	}
	return intencion, plan
}

// emisorGobiernoV3SustituidoPrueba entrega el material V3 para AUT60 sin
// firmarlo; la sustitución SQL de la prueba lo interpreta.
type emisorGobiernoV3SustituidoPrueba struct {
	t      *testing.T
	motivo domain.ReferenciaEntradaCatalogo
}

func (e *emisorGobiernoV3SustituidoPrueba) EmitirGobiernoRolNuevo(_ context.Context, actor domain.ContextoActor,
	evidencia domain.EvidenciaSesionAdministracionPerfiles, instantanea domain.InstantaneaAutorizacion,
	efecto Efecto) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	e.t.Helper()
	recurso, err := RecursoGobiernoRolNuevo(efecto, instantanea.AsignacionPerfil)
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
