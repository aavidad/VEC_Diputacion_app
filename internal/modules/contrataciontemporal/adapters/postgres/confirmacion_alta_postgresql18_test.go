package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestCandidaturaAltaPostgreSQL18DeExtremoATerminal(t *testing.T) {
	if os.Getenv("VEC_CT_O2_R3B_INTEGRACION_PG") != "SI" {
		t.Skip("solo se ejecuta desde el runner PostgreSQL 18.4 de R3B")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	runtime := abrirPoolR3B(t, ctx, "VEC_CT_O2_R3B_RUNTIME_DSN")
	defer runtime.Close()
	admin := abrirPoolR3B(t, ctx, "VEC_CT_O2_R3B_ADMIN_DSN")
	defer admin.Close()

	t.Run("backfill conserva instante y raiz", func(t *testing.T) {
		var coincide bool
		err := admin.QueryRow(ctx, `SELECT c.instante_efecto = i.creada_en
			AND c.ambito_raiz_hmac = i.ambito_hmac
			AND c.huella_raiz_hmac = i.huella_peticion_hmac
			AND c.origen = 'backfill'
		  FROM vec_contratacion_temporal.candidatura_alta_tecnica c
		  JOIN vec_contratacion_temporal.identidad_reserva_alta i
		    ON i.ambito_hmac = c.ambito_raiz_hmac
		 WHERE i.expediente_ref = 'expediente:ct:r3b:backfill'`).Scan(&coincide)
		if err != nil || !coincide {
			t.Fatalf("backfill divergente: %v, %v", coincide, err)
		}
	})

	resolutor, err := NuevoResolutorCandidaturaAltaPostgreSQL(runtime)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, propuesta := solicitudR3BIntegracion(t, "estable", []int{2, 1})
	estadoAntes := estadoEfectosR3B(t, ctx, admin)
	primera, err := resolutor.ResolverCandidaturaAlta(ctx, solicitud)
	if err != nil {
		t.Fatal(err)
	}
	assertCandidaturaR3B(t, primera, propuesta)
	if despues := estadoEfectosR3B(t, ctx, admin); despues != estadoAntes {
		t.Fatalf("resolver creo efecto administrativo: antes=%s despues=%s", estadoAntes, despues)
	}
	probarEntradasInvalidasR3B(t, ctx, runtime, admin, solicitud, estadoAntes)

	t.Run("replay entre procesos conserva originales", func(t *testing.T) {
		otroPool := abrirPoolR3B(t, ctx, "VEC_CT_O2_R3B_RUNTIME_DSN")
		defer otroPool.Close()
		otro, err := NuevoResolutorCandidaturaAltaPostgreSQL(otroPool)
		if err != nil {
			t.Fatal(err)
		}
		recuperada, err := otro.ResolverCandidaturaAlta(ctx, solicitud)
		if err != nil {
			t.Fatal(err)
		}
		assertCandidaturaR3B(t, recuperada, propuesta)
	})

	t.Run("concurrencia converge", func(t *testing.T) {
		concurrente, esperado := solicitudR3BIntegracion(t, "concurrente", []int{2, 1})
		const sesiones = 8
		resultados := make(chan ports.CandidaturaAlta, sesiones)
		errores := make(chan error, sesiones)
		var grupo sync.WaitGroup
		for indice := 0; indice < sesiones; indice++ {
			grupo.Add(1)
			go func() {
				defer grupo.Done()
				resultado, err := resolutor.ResolverCandidaturaAlta(ctx, concurrente)
				if err != nil {
					errores <- err
					return
				}
				resultados <- resultado
			}()
		}
		grupo.Wait()
		close(resultados)
		close(errores)
		for err := range errores {
			t.Errorf("sesion concurrente: %v", err)
		}
		for resultado := range resultados {
			assertCandidaturaR3B(t, resultado, esperado)
		}
		var raices, aliases int
		if err := admin.QueryRow(ctx, `SELECT
			(SELECT count(*) FROM vec_contratacion_temporal.candidatura_alta_tecnica
			  WHERE expediente_ref=$1),
			(SELECT count(*) FROM vec_contratacion_temporal.candidatura_alta_alias a
			  JOIN vec_contratacion_temporal.candidatura_alta_tecnica c USING(ambito_raiz_hmac)
			 WHERE c.expediente_ref=$1)`, esperado.Referencias.ExpedienteRef).Scan(&raices, &aliases); err != nil || raices != 1 || aliases != 2 {
			t.Fatalf("convergencia durable invalida: %d/%d, %v", raices, aliases, err)
		}
	})

	t.Run("rotacion anade alias sin mutar raiz", func(t *testing.T) {
		rotarPoliticaR3B(t, ctx, admin)
		rotada, propuestaRotada := solicitudR3BIntegracion(t, "rotacion", []int{2, 1})
		original, err := resolutor.ResolverCandidaturaAlta(ctx, rotada)
		if err != nil {
			t.Fatal(err)
		}
		assertCandidaturaR3B(t, original, propuestaRotada)
		replay, propuestaNueva := solicitudR3BIntegracion(t, "rotacion", []int{3, 2, 1})
		propuestaNueva.ReservaRef = "reserva:alta:r3b:propuesta-descartada"
		propuestaNueva.Referencias.ExpedienteRef = "expediente:ct:r3b:propuesta-descartada"
		propuestaNueva.Referencias.NumeroVisible = "2026/R3B-DESC"
		propuestaNueva.Referencias.ReciboRef = "recibo:alta:r3b:propuesta-descartada"
		replay = reconstruirSolicitudR3B(t, replay, propuestaNueva)
		recuperada, err := resolutor.ResolverCandidaturaAlta(ctx, replay)
		if err != nil {
			t.Fatal(err)
		}
		assertCandidaturaR3B(t, recuperada, propuestaRotada)
		var aliases int
		if err := admin.QueryRow(ctx, `SELECT count(*)
		  FROM vec_contratacion_temporal.candidatura_alta_alias a
		  JOIN vec_contratacion_temporal.candidatura_alta_tecnica c USING(ambito_raiz_hmac)
		 WHERE c.expediente_ref=$1`, propuestaRotada.Referencias.ExpedienteRef).Scan(&aliases); err != nil || aliases != 3 {
			t.Fatalf("rotacion no anadio exactamente un alias: %d, %v", aliases, err)
		}
	})

	t.Run("conflictos no se convierten en idempotencia", func(t *testing.T) {
		base, _ := solicitudR3BIntegracion(t, "conflicto", []int{3, 2, 1})
		if _, err := resolutor.ResolverCandidaturaAlta(ctx, base); err != nil {
			t.Fatal(err)
		}
		datos, _ := base.Datos()
		identidad := datos
		identidad.OrganizacionRef = "organizacion:otra"
		propuestaIdentidad, _ := identidad.Propuesta.Datos()
		propuestaIdentidad.OrganizacionRef = identidad.OrganizacionRef
		identidad.Propuesta, _ = ports.NuevaCandidaturaAlta(propuestaIdentidad)
		solicitudIdentidad, _ := ports.NuevaSolicitudResolverCandidaturaAlta(identidad)
		if _, err := resolutor.ResolverCandidaturaAlta(ctx, solicitudIdentidad); !errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
			t.Fatalf("cruce de identidad aceptado: %v", err)
		}

		colision, datosColision := solicitudR3BIntegracion(t, "colision", []int{3, 2, 1})
		datosColision.Referencias.ExpedienteRef = propuesta.Referencias.ExpedienteRef
		colision = reconstruirSolicitudR3B(t, colision, datosColision)
		if _, err := resolutor.ResolverCandidaturaAlta(ctx, colision); !errors.Is(err, ports.ErrPersistenciaNoDisponible) || errors.Is(err, ports.ErrClaveIdempotenciaUsada) {
			t.Fatalf("colision aleatoria tratada como replay: %v", err)
		}
	})

	t.Run("rollback no deja candidatura", func(t *testing.T) {
		solicitudRollback, _ := solicitudR3BIntegracion(t, "rollback", []int{3, 2, 1})
		entrada, err := nuevaEntradaResolverCandidaturaAlta(solicitudRollback)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(context.Background())
		_, err = tx.Exec(ctx, `SELECT set_config('timezone','UTC',true),
			set_config('statement_timeout','15s',true),
			set_config('idle_in_transaction_session_timeout','20s',true)`)
		if err != nil {
			t.Fatal(err)
		}
		var estado string
		err = tx.QueryRow(ctx, `SELECT resultado FROM `+funcionResolverCandidaturaAlta+`(
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, entrada.ambitos, entrada.huellas,
			entrada.organizacionRef, entrada.actorRef, entrada.perfilRef,
			entrada.propuesta.ReservaRef, entrada.propuesta.Referencias.ExpedienteRef,
			entrada.propuesta.Referencias.NumeroVisible, entrada.propuesta.Referencias.ReciboRef,
			entrada.propuesta.InstanteEfecto).Scan(&estado)
		if err != nil || estado != "estabilizada" {
			t.Fatalf("resolucion previa a rollback: %s, %v", estado, err)
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
		var cantidad int
		if err := admin.QueryRow(ctx, `SELECT count(*) FROM
			vec_contratacion_temporal.candidatura_alta_tecnica WHERE expediente_ref=$1`,
			entrada.propuesta.Referencias.ExpedienteRef).Scan(&cantidad); err != nil || cantidad != 0 {
			t.Fatalf("rollback dejo candidatura: %d, %v", cantidad, err)
		}
	})

	t.Run("propietario no reescribe ni trunca historia", func(t *testing.T) {
		antes := huellaHistoriaCandidaturasR3B(t, ctx, admin)
		operaciones := []struct {
			nombre   string
			consulta string
		}{
			{"actualizar candidatura", `UPDATE vec_contratacion_temporal.candidatura_alta_tecnica SET origen=origen`},
			{"eliminar candidatura", `DELETE FROM vec_contratacion_temporal.candidatura_alta_tecnica`},
			{"actualizar alias", `UPDATE vec_contratacion_temporal.candidatura_alta_alias SET registrada_en=registrada_en`},
			{"eliminar alias", `DELETE FROM vec_contratacion_temporal.candidatura_alta_alias`},
			{"truncar candidatura", `TRUNCATE vec_contratacion_temporal.candidatura_alta_tecnica, vec_contratacion_temporal.candidatura_alta_alias`},
			{"truncar alias", `TRUNCATE vec_contratacion_temporal.candidatura_alta_alias`},
		}
		for _, operacion := range operaciones {
			t.Run(operacion.nombre, func(t *testing.T) {
				tx, err := admin.BeginTx(ctx, pgx.TxOptions{})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = tx.Exec(ctx, `SET LOCAL ROLE vec_contratacion_temporal_propietario`); err != nil {
					_ = tx.Rollback(context.Background())
					t.Fatal(err)
				}
				if _, err = tx.Exec(ctx, operacion.consulta); err == nil {
					_ = tx.Rollback(context.Background())
					t.Fatalf("el propietario pudo %s", operacion.nombre)
				}
				if err := tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
					t.Fatal(err)
				}
				if despues := huellaHistoriaCandidaturasR3B(t, ctx, admin); despues != antes {
					t.Fatalf("%s altero la historia: antes=%s despues=%s", operacion.nombre, antes, despues)
				}
			})
		}
		var disparadores int
		err := admin.QueryRow(ctx, `SELECT count(*)
		  FROM pg_catalog.pg_trigger t
		  JOIN pg_catalog.pg_class c ON c.oid=t.tgrelid
		  JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
		 WHERE n.nspname='vec_contratacion_temporal'
		   AND c.relname IN ('candidatura_alta_tecnica','candidatura_alta_alias')
		   AND t.tgname IN ('candidatura_alta_tecnica_no_truncar','candidatura_alta_alias_no_truncar')
		   AND NOT t.tgisinternal AND (t.tgtype & 32)=32 AND (t.tgtype & 1)=0`).Scan(&disparadores)
		if err != nil || disparadores != 2 {
			t.Fatalf("disparadores de truncado divergentes: %d, %v", disparadores, err)
		}
	})

	t.Run("ACL y RLS cierran tablas V1 y preparador", func(t *testing.T) {
		for _, consulta := range []string{
			"TABLE vec_contratacion_temporal.candidatura_alta_tecnica",
			"TABLE vec_contratacion_temporal.candidatura_alta_alias",
			"SELECT * FROM vec_contratacion_temporal.confirmar_alta_atestada_v1(''::bytea,''::bytea,''::bytea,''::bytea,1,1,''::bytea,''::bytea,''::bytea,''::bytea,''::bytea,''::bytea)",
			"SELECT * FROM vec_contratacion_temporal.preparar_alta_v2('{}'::jsonb)",
		} {
			if _, err := runtime.Exec(ctx, consulta); err == nil {
				t.Fatalf("ACL permitio %s", consulta)
			}
		}
		var privilegios string
		err := admin.QueryRow(ctx, `SELECT concat_ws('|',
			has_function_privilege('vec_ct_r3b_runtime',$1,'EXECUTE'),
			has_function_privilege('vec_ct_r3b_runtime',$2,'EXECUTE'),
			has_function_privilege('vec_ct_r3b_runtime',$3,'EXECUTE'))`,
			"vec_contratacion_temporal.resolver_candidatura_alta_tecnica_v1(text[],text[],text,text,text,text,text,text,text,timestamp with time zone)",
			"vec_contratacion_temporal.confirmar_alta_atestada_v2(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)",
			"vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)").Scan(&privilegios)
		if err != nil || privilegios != "t|t|f" {
			t.Fatalf("privilegios runtime divergentes: %q, %v", privilegios, err)
		}
		var rlsForzada bool
		var politicas int
		err = admin.QueryRow(ctx, `SELECT
			bool_and(c.relrowsecurity AND c.relforcerowsecurity),
			(SELECT count(*) FROM pg_catalog.pg_policies
			  WHERE schemaname = 'vec_contratacion_temporal'
			    AND policyname IN ('candidatura_alta_tecnica_propietario',
			                       'candidatura_alta_alias_propietario')
			    AND roles = ARRAY['vec_contratacion_temporal_propietario']::name[])
		  FROM pg_catalog.pg_class c
		  JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = 'vec_contratacion_temporal'
		   AND c.relname IN ('candidatura_alta_tecnica', 'candidatura_alta_alias')`).Scan(
			&rlsForzada, &politicas,
		)
		if err != nil || !rlsForzada || politicas != 2 {
			t.Fatalf("RLS divergente: forzada=%t politicas=%d error=%v", rlsForzada, politicas, err)
		}
	})
}

func probarEntradasInvalidasR3B(t *testing.T, ctx context.Context, runtime, admin *pgxpool.Pool, solicitud ports.SolicitudResolverCandidaturaAlta, efectosAntes string) {
	t.Helper()
	entrada, err := nuevaEntradaResolverCandidaturaAlta(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	base := []any{literalMatrizR3B(entrada.ambitos...), literalMatrizR3B(entrada.huellas...), entrada.organizacionRef, entrada.actorRef, entrada.perfilRef, entrada.propuesta.ReservaRef, entrada.propuesta.Referencias.ExpedienteRef, entrada.propuesta.Referencias.NumeroVisible, entrada.propuesta.Referencias.ReciboRef, entrada.propuesta.InstanteEfecto}
	type caso struct {
		nombre     string
		argumentos []any
	}
	casos := make([]caso, 0, 19)
	for indice, nombre := range []string{"ambitos nulos", "huellas nulas", "organizacion nula", "actor nulo", "perfil nulo", "reserva nula", "expediente nulo", "numero nulo", "recibo nulo", "instante nulo"} {
		casos = append(casos, caso{nombre, argumentosInvalidosR3B(base, map[int]any{indice: nil})})
	}
	ambitos, huellas := base[0].(string), base[1].(string)
	casos = append(casos,
		caso{"ambitos multidimensionales", argumentosInvalidosR3B(base, map[int]any{0: "{" + ambitos + "," + ambitos + "}"})},
		caso{"huellas multidimensionales", argumentosInvalidosR3B(base, map[int]any{1: "{" + huellas + "," + huellas + "}"})},
		caso{"ambitos con lower cero", argumentosInvalidosR3B(base, map[int]any{0: "[0:1]=" + ambitos})},
		caso{"huellas con lower cero", argumentosInvalidosR3B(base, map[int]any{1: "[0:1]=" + huellas})},
		caso{"matrices vacias", argumentosInvalidosR3B(base, map[int]any{0: "{}", 1: "{}"})},
		caso{"cardinalidad cinco", argumentosInvalidosR3B(base, map[int]any{0: literalMatrizR3B(entrada.ambitos[0], entrada.ambitos[1], entrada.ambitos[0], entrada.ambitos[1], entrada.ambitos[0]), 1: literalMatrizR3B(entrada.huellas[0], entrada.huellas[1], entrada.huellas[0], entrada.huellas[1], entrada.huellas[0])})},
		caso{"cardinalidades distintas", argumentosInvalidosR3B(base, map[int]any{1: literalMatrizR3B(entrada.huellas[0])})},
		caso{"elemento ambito nulo", argumentosInvalidosR3B(base, map[int]any{0: "{NULL," + entrada.ambitos[1] + "}"})},
		caso{"elemento huella nulo", argumentosInvalidosR3B(base, map[int]any{1: "{NULL," + entrada.huellas[1] + "}"})},
	)
	historiaAntes := huellaHistoriaCandidaturasR3B(t, ctx, admin)
	for _, prueba := range casos {
		t.Run("entrada invalida/"+prueba.nombre, func(t *testing.T) {
			tx, err := runtime.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec(ctx, `SELECT set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
				_ = tx.Rollback(context.Background())
				t.Fatal(err)
			}
			var resultado string
			err = tx.QueryRow(ctx, `SELECT resultado FROM `+funcionResolverCandidaturaAlta+`($1::text[],$2::text[],$3::text,$4::text,$5::text,$6::text,$7::text,$8::text,$9::text,$10::timestamptz)`, prueba.argumentos...).Scan(&resultado)
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "22023" || pgErr.Message != "candidatura de alta invalida" || resultado != "" {
				_ = tx.Rollback(context.Background())
				t.Fatalf("rechazo divergente: resultado=%q error=%v", resultado, err)
			}
			if err := tx.Rollback(ctx); err != nil {
				t.Fatal(err)
			}
			if historia := huellaHistoriaCandidaturasR3B(t, ctx, admin); historia != historiaAntes {
				t.Fatalf("entrada invalida muto candidaturas: antes=%s despues=%s", historiaAntes, historia)
			}
			if efectos := estadoEfectosR3B(t, ctx, admin); efectos != efectosAntes {
				t.Fatalf("entrada invalida creo efecto administrativo: antes=%s despues=%s", efectosAntes, efectos)
			}
		})
	}
}

func literalMatrizR3B(valores ...string) string { return "{" + strings.Join(valores, ",") + "}" }

func argumentosInvalidosR3B(base []any, cambios map[int]any) []any {
	resultado := append([]any(nil), base...)
	for indice, valor := range cambios {
		resultado[indice] = valor
	}
	return resultado
}

type entradaPublicaR3B struct {
	Ahora                  string                           `json:"ahora"`
	DecisionPlantillaB64   string                           `json:"decision_plantilla_b64"`
	MotivoB64              string                           `json:"motivo_b64"`
	ContextoB64            string                           `json:"contexto_b64"`
	ManifiestoB64          string                           `json:"manifiesto_b64"`
	ManifiestoHuellaSHA256 string                           `json:"manifiesto_huella_sha256"`
	AutoridadEfectiva      string                           `json:"autoridad_efectiva"`
	ResueltoEn             string                           `json:"resuelto_en"`
	AltaB64                string                           `json:"alta_b64"`
	SellosB64              string                           `json:"sellos_b64"`
	EfectoHuellaSHA256     string                           `json:"efecto_huella_sha256"`
	Politicas              []dominiovec.PoliticaRestrictiva `json:"politicas"`
	RevisionCatalogo       uint64                           `json:"revision_catalogo"`
	HuellaCatalogoSHA256   string                           `json:"huella_catalogo_sha256"`
}

type decisionPlantillaPublicaR3B struct {
	DecisionRef               string                                      `json:"decision_ref"`
	Accion                    string                                      `json:"accion"`
	RecursoRef                string                                      `json:"recurso_ref"`
	ModuloID                  string                                      `json:"modulo_id"`
	TipoRecurso               string                                      `json:"tipo_recurso"`
	Finalidad                 string                                      `json:"finalidad"`
	CorrelacionRef            string                                      `json:"correlacion_ref"`
	VinculoAutenticacionActor dominiovec.DatosVinculoAutenticacionActorV2 `json:"vinculo_autenticacion_actor"`
}

type motivoPublicoR3B struct {
	Referencia dominiovec.ReferenciaEntradaCatalogo `json:"referencia"`
}

type bundlePublicoR3B struct {
	CapacidadB64        string          `json:"capacidad_b64"`
	DecisionB64         string          `json:"decision_b64"`
	MotivoB64           string          `json:"motivo_b64"`
	ContextoB64         string          `json:"contexto_b64"`
	PayloadB64          string          `json:"payload_b64"`
	COSEB64             string          `json:"cose_b64"`
	EvidenciaB64        string          `json:"evidencia_b64"`
	SPKIB64             string          `json:"spki_b64"`
	AltaB64             string          `json:"alta_b64"`
	SellosB64           string          `json:"sellos_b64"`
	PersonaVersion      uint64          `json:"persona_version"`
	PerfilVersion       uint64          `json:"perfil_version"`
	VersionRolDocumento json.RawMessage `json:"version_rol_documento"`
	ControlRolDocumento json.RawMessage `json:"control_rol_documento"`
	AsignacionDocumento json.RawMessage `json:"asignacion_documento"`
}

type capacidadPublicaR3B struct {
	DecisionRef          string `json:"decision_ref"`
	HuellaDecisionSHA256 string `json:"huella_decision_sha256"`
	HuellaMotivoSHA256   string `json:"huella_motivo_sha256"`
	ContextoRef          string `json:"contexto_ref"`
	HuellaContextoSHA256 string `json:"huella_contexto_sha256"`
	Operacion            string `json:"operacion"`
	EfectoRef            string `json:"efecto_ref"`
	HuellaEfectoSHA256   string `json:"huella_efecto_sha256"`
	AudienciaConsumo     string `json:"audiencia_consumo"`
	EmitidaEn            string `json:"emitida_en"`
	ExpiraEn             string `json:"expira_en"`
}

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
	segundaTransaccion, err := NuevaTransaccionAltasPostgreSQLCandidata(segundoPool, proveedor)
	if err != nil {
		t.Fatal(err)
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

func huellaHistoriaCandidaturasR3B(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
) string {
	t.Helper()
	var huella string
	err := admin.QueryRow(ctx, `SELECT concat_ws('|',
		(SELECT count(*) FROM vec_contratacion_temporal.candidatura_alta_tecnica),
		(SELECT md5(coalesce(string_agg(to_jsonb(c)::text,E'\n' ORDER BY c.ambito_raiz_hmac),''))
		   FROM vec_contratacion_temporal.candidatura_alta_tecnica c),
		(SELECT count(*) FROM vec_contratacion_temporal.candidatura_alta_alias),
		(SELECT md5(coalesce(string_agg(to_jsonb(a)::text,E'\n' ORDER BY a.ambito_hmac),''))
		   FROM vec_contratacion_temporal.candidatura_alta_alias a))`).Scan(&huella)
	if err != nil {
		t.Fatal(err)
	}
	return huella
}

func abrirPoolR3B(t *testing.T, ctx context.Context, variable string) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv(variable)
	if dsn == "" || strings.Contains(strings.ToLower(dsn), "password") {
		t.Fatalf("%s ausente o contiene credencial", variable)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	return pool
}

func solicitudR3BIntegracion(
	t *testing.T,
	sufijo string,
	generaciones []int,
) (ports.SolicitudResolverCandidaturaAlta, ports.DatosCandidaturaAlta) {
	t.Helper()
	ambitosValores := make([]string, len(generaciones))
	huellasValores := make([]string, len(generaciones))
	for indice, generacion := range generaciones {
		ambitosValores[indice] = selloR3BIntegracion("ambito-idempotencia", generacion, sufijo)
		huellasValores[indice] = selloR3BIntegracion("huella-peticion", generacion, sufijo)
	}
	ambitos, err := ports.NuevaColeccionSellosHMAC(ambitosValores[0], ambitosValores[1:])
	if err != nil {
		t.Fatal(err)
	}
	huellas, err := ports.NuevaColeccionSellosHMAC(huellasValores[0], huellasValores[1:])
	if err != nil {
		t.Fatal(err)
	}
	datos := ports.DatosCandidaturaAlta{
		ReservaRef: "reserva:alta:r3b:" + sufijo,
		Referencias: ports.ReferenciasAlta{
			ExpedienteRef: "expediente:ct:r3b:" + sufijo,
			NumeroVisible: "2026/" + strings.ToUpper(sufijo),
			ReciboRef:     "recibo:alta:r3b:" + sufijo,
		},
		AmbitoIdempotenciaHMAC: ambitosValores[0], HuellaPeticionHMAC: huellasValores[0],
		OrganizacionRef: "organizacion:dipgra", ActorRef: "actor:rrhh:r3b",
		PerfilRef:      "perfil:rrhh:r3b",
		InstanteEfecto: time.Date(2026, 8, 31, 12, 0, 0, 123456000, time.UTC),
	}
	candidatura, err := ports.NuevaCandidaturaAlta(datos)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudResolverCandidaturaAlta(
		ports.DatosSolicitudResolverCandidaturaAlta{
			AmbitosIdempotenciaHMAC: ambitos, HuellasPeticionHMAC: huellas,
			OrganizacionRef: datos.OrganizacionRef, ActorRef: datos.ActorRef,
			PerfilRef: datos.PerfilRef, Propuesta: candidatura,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, datos
}

func selloR3BIntegracion(dominio string, generacion int, sufijo string) string {
	semilla := fmt.Sprintf("r3b:%s:%d:%s", dominio, generacion, sufijo)
	huella := huellaTextoR3B(semilla)
	return fmt.Sprintf("hmac-sha256:vec.contratacion-temporal.%s/v%d:%s", dominio, generacion, huella)
}

func huellaTextoR3B(texto string) string {
	const hexadecimal = "0123456789abcdef"
	resultado := make([]byte, 64)
	for indice := range resultado {
		resultado[indice] = hexadecimal[(int(texto[indice%len(texto)])+indice)%len(hexadecimal)]
	}
	return string(resultado)
}

func reconstruirSolicitudR3B(
	t *testing.T,
	base ports.SolicitudResolverCandidaturaAlta,
	propuesta ports.DatosCandidaturaAlta,
) ports.SolicitudResolverCandidaturaAlta {
	t.Helper()
	datos, err := base.Datos()
	if err != nil {
		t.Fatal(err)
	}
	datos.Propuesta, err = ports.NuevaCandidaturaAlta(propuesta)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := ports.NuevaSolicitudResolverCandidaturaAlta(datos)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func assertCandidaturaR3B(
	t *testing.T,
	obtenida ports.CandidaturaAlta,
	esperada ports.DatosCandidaturaAlta,
) {
	t.Helper()
	datos, err := obtenida.Datos()
	if err != nil || datos != esperada {
		t.Fatalf("candidatura divergente: %+v, %v", datos, err)
	}
}

func estadoEfectosR3B(t *testing.T, ctx context.Context, admin *pgxpool.Pool) string {
	t.Helper()
	var estado string
	err := admin.QueryRow(ctx, `SELECT concat_ws('|',
		(SELECT count(*) FROM vec_contratacion_temporal.identidad_reserva_alta),
		(SELECT count(*) FROM vec_contratacion_temporal.reserva_alta_version),
		(SELECT count(*) FROM vec_contratacion_temporal.expediente_alta),
		(SELECT count(*) FROM vec_contratacion_temporal.actuacion_alta),
		(SELECT count(*) FROM vec_contratacion_temporal.auditoria_alta),
		(SELECT count(*) FROM vec_contratacion_temporal.outbox_alta),
		(SELECT count(*) FROM vec_autorizacion_atestada_v3.consumo_decision_v3))`).Scan(&estado)
	if err != nil {
		t.Fatal(err)
	}
	return estado
}

func rotarPoliticaR3B(t *testing.T, ctx context.Context, admin *pgxpool.Pool) {
	t.Helper()
	tx, err := admin.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(ctx, `SET LOCAL ROLE vec_contratacion_temporal_propietario;
		DELETE FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
		INSERT INTO vec_contratacion_temporal.politica_generaciones_hmac_alta
		(generacion,posicion,estado,registrada_en) VALUES
		(3,0,'activa',date_trunc('microseconds',clock_timestamp())),
		(2,1,'retenida',date_trunc('microseconds',clock_timestamp())),
		(1,2,'retenida',date_trunc('microseconds',clock_timestamp()))`)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
