package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type autoridadPublicaR3B struct {
	autenticacion dominiovec.AutenticacionRevalidadaV1
	contexto      dominiovec.ResultadoContextoActorRegistradoV2
	ahora         time.Time
	correlacion   string
}

func (a autoridadPublicaR3B) RevalidarAutenticacionActorV1(context.Context, dominiovec.SolicitudRevalidacionAutenticacionActorV1) (dominiovec.AutenticacionRevalidadaV1, error) {
	return a.autenticacion, nil
}
func (a autoridadPublicaR3B) ResolverContextoActorRegistradoV2(context.Context, dominiovec.SolicitudContextoActor) (dominiovec.ResultadoContextoActorRegistradoV2, error) {
	return a.contexto, nil
}
func (a autoridadPublicaR3B) Ahora() time.Time { return a.ahora }
func (a autoridadPublicaR3B) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return a.correlacion, nil
}
func (a autoridadPublicaR3B) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(context.Context, puertosvec.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	return a.ahora, nil
}

type proveedorPublicoR3B struct {
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (p proveedorPublicoR3B) ProveerMaterialConfirmacionAlta(ctx context.Context, orden ports.OrdenConfirmarAltaCandidata) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if ctx == nil || ctx.Err() != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrPersistenciaNoDisponible
	}
	if _, err := orden.Datos(); err != nil || p.material.ValidarEstructura() != nil {
		return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrOrdenAltaInvalida
	}
	return p.material, nil
}

func TestConfirmacionAltaPublicaPostgreSQL18DesdeDosPools(t *testing.T) {
	if os.Getenv("VEC_CT_O2_R3B_INTEGRACION_PG") != "SI" {
		t.Skip("solo se ejecuta desde el runner PostgreSQL 18.4 de R3B")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	primerPool := abrirPoolR3B(t, ctx, "VEC_CT_O2_R3B_RUNTIME_DSN")
	defer primerPool.Close()
	segundoPool := abrirPoolR3B(t, ctx, "VEC_CT_O2_R3B_RUNTIME_DSN")
	defer segundoPool.Close()
	admin := abrirPoolR3B(t, ctx, "VEC_CT_O2_R3B_ADMIN_DSN")
	defer admin.Close()

	var entrada entradaPublicaR3B
	var bundle bundlePublicoR3B
	leerJSONPublicoR3B(t, os.Getenv("VEC_CT_O2_R3B_VECTOR_ENTRADA"), &entrada)
	leerJSONPublicoR3B(t, os.Getenv("VEC_CT_O2_R3B_VECTOR_BUNDLE"), &bundle)
	alta := decodificarPublicoR3B(t, bundle.AltaB64)
	sellosJSON := decodificarPublicoR3B(t, bundle.SellosB64)
	var efecto efectoAltaCanonico
	var sellos sellosAltaCanonicos
	if json.Unmarshal(alta, &efecto) != nil || json.Unmarshal(sellosJSON, &sellos) != nil {
		t.Fatal("vector publico de alta invalido")
	}
	expediente := expedientePublicoR3B(t, efecto)
	ambitos, huellas := coleccionesPublicasR3B(t, sellos)
	propuesta, err := ports.NuevaCandidaturaAlta(ports.DatosCandidaturaAlta{
		ReservaRef: efecto.ReservaRef,
		Referencias: ports.ReferenciasAlta{ExpedienteRef: efecto.ExpedienteRef,
			NumeroVisible: efecto.NumeroVisible, ReciboRef: efecto.ReciboRef},
		AmbitoIdempotenciaHMAC: sellos.Activo.AmbitoHMAC,
		HuellaPeticionHMAC:     sellos.Activo.HuellaHMAC,
		OrganizacionRef:        efecto.OrganizacionRef, ActorRef: efecto.ActorRef,
		PerfilRef: efecto.PerfilRef, InstanteEfecto: expediente.CreadoEn,
	})
	if err != nil {
		t.Fatal(err)
	}
	solicitudCandidatura, err := ports.NuevaSolicitudResolverCandidaturaAlta(ports.DatosSolicitudResolverCandidaturaAlta{
		AmbitosIdempotenciaHMAC: ambitos, HuellasPeticionHMAC: huellas,
		OrganizacionRef: efecto.OrganizacionRef, ActorRef: efecto.ActorRef,
		PerfilRef: efecto.PerfilRef, Propuesta: propuesta,
	})
	if err != nil {
		t.Fatal(err)
	}
	resolutor, _ := NuevoResolutorCandidaturaAltaPostgreSQL(primerPool)
	candidatura, err := resolutor.ResolverCandidaturaAlta(ctx, solicitudCandidatura)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, decision, confirmacion, material := materialPublicoR3B(t, entrada, bundle, efecto, sellos)
	orden, err := ports.NuevaOrdenConfirmarAltaCandidata(ports.DatosOrdenConfirmarAltaCandidata{
		Expediente: expediente, SolicitudAutorizacionV3: solicitud,
		DecisionAutorizacionV3: decision, ConfirmacionRegistroV3: confirmacion,
		AmbitosIdempotenciaHMAC: ambitos, HuellasPeticionHMAC: huellas,
		Candidatura: candidatura,
	})
	if err != nil {
		t.Fatal(err)
	}
	evidencia, _ := orden.Datos()
	entradas, err := prepararEntradasConfirmarAlta(evidencia, material)
	if err != nil {
		t.Fatal(err)
	}
	defer entradas.borrar()
	if len(argumentosConfirmarAlta(entradas)) != 12 ||
		!bytes.Equal(entradas.alta, alta) || !bytes.Equal(entradas.sellos, sellosJSON) {
		t.Fatal("las doce entradas publicas no conservan alta y sellos exactos")
	}
	proveedor := proveedorPublicoR3B{material: material}
	primeraTransaccion, err := NuevaTransaccionAltasPostgreSQLCandidata(primerPool, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	primerRecibo, err := primeraTransaccion.ConfirmarAltaCandidata(ctx, orden)
	if err != nil {
		t.Fatal(err)
	}
	estado := estadoConfirmacionPublicaR3B(t, ctx, admin, efecto.ExpedienteRef)
	estadoGlobal := estadoEfectosR3B(t, ctx, admin)
	segundaTransaccion, err := NuevaTransaccionAltasPostgreSQLCandidata(segundoPool, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	var huellaReciboOriginal string
	if err := admin.QueryRow(ctx, `SELECT recibo_huella_sha256
		FROM vec_contratacion_temporal.confirmacion_agregado_alta
		WHERE expediente_ref=$1`, efecto.ExpedienteRef).Scan(&huellaReciboOriginal); err != nil {
		t.Fatal(err)
	}
	huellaReciboIncoherente := strings.Repeat("f", 64)
	if huellaReciboIncoherente == huellaReciboOriginal {
		huellaReciboIncoherente = strings.Repeat("e", 64)
	}
	cambiarHuellaReciboPublicoR3B(
		t, ctx, admin, efecto.ExpedienteRef, huellaReciboIncoherente,
	)
	defer func() {
		cambiarHuellaReciboPublicoR3B(
			t, context.Background(), admin, efecto.ExpedienteRef, huellaReciboOriginal,
		)
	}()
	reciboRechazado, err := segundaTransaccion.ConfirmarAltaCandidata(ctx, orden)
	if !errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
		reciboRechazado != (ports.ReciboAlta{}) {
		t.Fatalf("otro fallo posterior al consumo fue reconciliado: %+v, %v", reciboRechazado, err)
	}
	cambiarHuellaReciboPublicoR3B(
		t, ctx, admin, efecto.ExpedienteRef, huellaReciboOriginal,
	)
	if despues, globalDespues := estadoConfirmacionPublicaR3B(t, ctx, admin, efecto.ExpedienteRef),
		estadoEfectosR3B(t, ctx, admin); despues != estado || globalDespues != estadoGlobal {
		t.Fatalf("fallo distinto muto o duplico el efecto: antes=%s/%s despues=%s/%s",
			estado, estadoGlobal, despues, globalDespues)
	}
	segundoRecibo, err := segundaTransaccion.ConfirmarAltaCandidata(ctx, orden)
	if err != nil || segundoRecibo != primerRecibo ||
		estadoConfirmacionPublicaR3B(t, ctx, admin, efecto.ExpedienteRef) != estado {
		t.Fatalf("replay publico muto el efecto: recibos=%+v/%+v err=%v", primerRecibo, segundoRecibo, err)
	}
	var reciboSQL ports.ReciboAlta
	var version int64
	var huella string
	err = admin.QueryRow(ctx, `SELECT expediente_ref,numero_visible,version_expediente,
		recibo_ref,auditoria_ref,evento_ref,confirmada_en,recibo_huella_sha256
		FROM vec_contratacion_temporal.confirmacion_agregado_alta WHERE expediente_ref=$1`,
		efecto.ExpedienteRef).Scan(&reciboSQL.ExpedienteRef, &reciboSQL.NumeroVisible, &version,
		&reciboSQL.ReciboRef, &reciboSQL.AuditoriaRef, &reciboSQL.EventoRef,
		&reciboSQL.ConfirmadaEn, &huella)
	reciboSQL.Version = uint64(version)
	reciboSQL.ConfirmadaEn = reciboSQL.ConfirmadaEn.UTC()
	if err != nil || reciboSQL != primerRecibo || huella != huellaReciboAlta(primerRecibo) {
		t.Fatalf("recibo o huella interna divergente: SQL=%+v Go=%+v huella=%s err=%v",
			reciboSQL, primerRecibo, huella, err)
	}
}

func cambiarHuellaReciboPublicoR3B(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	expedienteRef string,
	huella string,
) {
	t.Helper()
	tx, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role = replica`); err == nil {
		_, err = tx.Exec(ctx, `UPDATE vec_contratacion_temporal.confirmacion_agregado_alta
			SET recibo_huella_sha256=$2 WHERE expediente_ref=$1`, expedienteRef, huella)
	}
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
