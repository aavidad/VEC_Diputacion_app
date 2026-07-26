package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func sembrarAnalisisO3PreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture fixturePreparacionDecisionCoberturaDurable,
) {
	t.Helper()
	agregado, err := json.Marshal(fixture.expediente)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(tx)
	if _, err := tx.Exec(ctx, `
		SET LOCAL session_replication_role = 'replica';
		SET LOCAL ROLE vec_contratacion_temporal_propietario`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
		  ambito_hmac,reserva_ref,expediente_ref,numero_visible,recibo_ref,
		  huella_peticion_hmac,organizacion_ref,actor_ref,perfil_ref,creada_en
		) VALUES (
		  'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'
		    || pg_catalog.repeat('1',64),
		  'reserva_alta_o404c_01',$1,$2,'recibo_alta_o404c_01',
		  'hmac-sha256:vec.contratacion-temporal.huella-peticion/v1:'
		    || pg_catalog.repeat('2',64),
		  $3,'actor_o404c_01','perfil_o404c_01',$4
		)`,
		fixture.expediente.Referencia,
		fixture.expediente.NumeroVisible,
		fixture.expediente.OrganizacionRef,
		fixture.base.Add(-2*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO vec_contratacion_temporal.expediente_alta (
		  expediente_ref,reserva_ref,numero_visible,organizacion_ref,
		  actor_ref,perfil_ref,decision_ref,efecto_ref,
		  huella_efecto_sha256,creada_en,confirmacion_ref
		) VALUES (
		  $1,'reserva_alta_o404c_01',$2,$3,
		  'actor_o404c_01','perfil_o404c_01',
		  'decision_alta_o404c_01','efecto_alta_o404c_01',
		  pg_catalog.repeat('3',64),$4,
		  'cnf_ct_0123456789abcdef0123456789abcdef'
		)`,
		fixture.expediente.Referencia,
		fixture.expediente.NumeroVisible,
		fixture.expediente.OrganizacionRef,
		fixture.base.Add(-2*time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		WITH datos AS (
		  SELECT $1::jsonb AS agregado,
		         pg_catalog.convert_to(
		           pg_catalog.repeat('p',128),'UTF8'
		         ) AS prueba
		)
		INSERT INTO
		  vec_contratacion_temporal.expediente_version_integral (
		    expediente_ref,version,agregado_json,
		    agregado_json_huella_sha256,prueba_canonica,
		    prueba_huella_sha256,flujo_ref,flujo_version,
		    flujo_huella_sha256,fase_clave,estado,origen_version,
		    operacion_ref,registrada_en
		  )
		SELECT $2,$3,agregado,
		       pg_catalog.encode(
		         pg_catalog.sha256(
		           pg_catalog.convert_to(agregado::text,'UTF8')
		         ),'hex'
		       ),
		       prueba,pg_catalog.encode(pg_catalog.sha256(prueba),'hex'),
		       agregado #>> '{flujo,definicion_ref}',
		       (agregado #>> '{flujo,version}')::numeric,
		       agregado #>> '{flujo,huella_sha256}',
		       agregado ->> 'fase_actual',agregado ->> 'estado_actual',
		       'analisis_o3','operacion_analisis_o404c_01',$4
		  FROM datos`,
		agregado,
		fixture.expediente.Referencia,
		fixture.expediente.Version,
		fixture.base.Add(-time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO vec_contratacion_temporal.expediente_integral_actual (
		  expediente_ref,version,actualizada_en,operacion_ref
		) VALUES ($1,$2,$3,'operacion_analisis_o404c_01')`,
		fixture.expediente.Referencia,
		fixture.expediente.Version,
		fixture.base.Add(-time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var analisisValido bool
	var huella string
	lectura, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(lectura)
	if _, err := lectura.Exec(
		ctx,
		"SET LOCAL ROLE vec_contratacion_temporal_propietario",
	); err != nil {
		t.Fatal(err)
	}
	if err := lectura.QueryRow(ctx, `
		SELECT vec_contratacion_temporal.analisis_rrhh_valido_v3(
		         agregado_json -> 'analisis'
		       ),
		       vec_contratacion_temporal.huella_analisis_derivado_v2(
		         agregado_json -> 'analisis'
		       )
		  FROM vec_contratacion_temporal.expediente_version_integral
		 WHERE expediente_ref=$1 AND version=$2`,
		fixture.expediente.Referencia,
		fixture.expediente.Version,
	).Scan(&analisisValido, &huella); err != nil ||
		!analisisValido || len(huella) != 64 {
		t.Fatalf(
			"precondición O3 sintética inválida: valida=%t huella=%q err=%v",
			analisisValido,
			huella,
			err,
		)
	}
	if err := lectura.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func probarReservaCercadaPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	preparador *PreparadorOperacionDecisionCoberturaDurablePostgreSQL,
	fixture fixturePreparacionDecisionCoberturaDurable,
) {
	t.Helper()
	identidad := fixture.identidad(
		t,
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a51",
		fixture.expediente.OrganizacionRef,
	)
	par := parHMACDecisionCoberturaPrueba(1, "1", "2")
	_, primera := solicitudPreparacionDecisionCoberturaPrueba(
		t,
		identidad,
		par,
	)
	preparacion, err :=
		preparador.ReservarOReapropiarOperacionDecisionCobertura(
			ctx,
			primera,
		)
	if err != nil {
		t.Fatalf("reservar primera operación C: %v", err)
	}
	estado, err := preparacion.EstadoPara(primera)
	if err != nil ||
		estado != cobertura.PreparacionOperacionDecisionCoberturaPropietaria {
		t.Fatalf("primera reserva no propietaria: %q / %v", estado, err)
	}
	datos, err := preparacion.DatosPropietariaPara(primera)
	if err != nil || datos.RevisionCercado != 1 ||
		datos.RevisionCercadoAnterior != 0 {
		t.Fatalf("fence inicial inválido: %#v / %v", datos, err)
	}
	_, segunda := solicitudPreparacionDecisionCoberturaPrueba(
		t,
		identidad,
		par,
	)
	ocupada, err :=
		preparador.ReservarOReapropiarOperacionDecisionCobertura(
			ctx,
			segunda,
		)
	if err != nil {
		t.Fatal(err)
	}
	estado, err = ocupada.EstadoPara(segunda)
	if err != nil ||
		estado != cobertura.PreparacionOperacionDecisionCoberturaOcupada {
		t.Fatalf("reserva simultánea no ocupada: %q / %v", estado, err)
	}
	time.Sleep(5200 * time.Millisecond)
	type resultado struct {
		solicitud   cobertura.SolicitudReservarOperacionDecisionCobertura
		preparacion cobertura.PreparacionOperacionDecisionCobertura
		err         error
	}
	resultados := make(chan resultado, 2)
	solicitudes := make(
		[]cobertura.SolicitudReservarOperacionDecisionCobertura,
		2,
	)
	for indice := range solicitudes {
		_, solicitudes[indice] =
			solicitudPreparacionDecisionCoberturaPrueba(
				t,
				identidad,
				par,
			)
	}
	var grupo sync.WaitGroup
	grupo.Add(2)
	for _, solicitud := range solicitudes {
		go func(
			solicitud cobertura.SolicitudReservarOperacionDecisionCobertura,
		) {
			defer grupo.Done()
			preparacion, err :=
				preparador.ReservarOReapropiarOperacionDecisionCobertura(
					ctx,
					solicitud,
				)
			resultados <- resultado{
				solicitud: solicitud, preparacion: preparacion, err: err,
			}
		}(solicitud)
	}
	grupo.Wait()
	close(resultados)
	var propietarias, ocupadas int
	for resultado := range resultados {
		if resultado.err != nil {
			t.Fatal(resultado.err)
		}
		estado, err := resultado.preparacion.EstadoPara(resultado.solicitud)
		if err != nil {
			t.Fatal(err)
		}
		switch estado {
		case cobertura.PreparacionOperacionDecisionCoberturaPropietaria:
			propietarias++
			datos, err := resultado.preparacion.DatosPropietariaPara(
				resultado.solicitud,
			)
			if err != nil || datos.RevisionCercado != 2 ||
				datos.RevisionCercadoAnterior != 1 {
				t.Fatalf(
					"reapropiación sin fence creciente: %#v / %v",
					datos,
					err,
				)
			}
		case cobertura.PreparacionOperacionDecisionCoberturaOcupada:
			ocupadas++
		default:
			t.Fatalf("estado concurrente inesperado: %q", estado)
		}
	}
	if propietarias != 1 || ocupadas != 1 {
		t.Fatalf(
			"CAS concurrente divergente: propietarias=%d ocupadas=%d",
			propietarias,
			ocupadas,
		)
	}
}

func probarReplayRotadoPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	preparador *PreparadorOperacionDecisionCoberturaDurablePostgreSQL,
	fixture fixturePreparacionDecisionCoberturaDurable,
) cobertura.SolicitudConsultarOperacionDecisionCoberturaConfirmada {
	t.Helper()
	identidad := fixture.identidad(
		t,
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a52",
		fixture.expediente.OrganizacionRef,
	)
	original := parHMACDecisionCoberturaPrueba(1, "3", "4")
	consultaOriginal, solicitudOriginal :=
		solicitudPreparacionDecisionCoberturaPrueba(
			t,
			identidad,
			original,
		)
	preparacion, err :=
		preparador.ReservarOReapropiarOperacionDecisionCobertura(
			ctx,
			solicitudOriginal,
		)
	if err != nil {
		t.Fatal(err)
	}
	if estado, err := preparacion.EstadoPara(solicitudOriginal); err != nil ||
		estado != cobertura.PreparacionOperacionDecisionCoberturaPropietaria {
		t.Fatalf("reserva para terminal concedido inválida: %q / %v", estado, err)
	}
	confirmarTerminalPreparacionDecisionCobertura(
		t,
		ctx,
		admin,
		original,
		"concedida",
	)
	contenidoTerminal := cargarTerminalPreparacionDecisionCobertura(
		t,
		ctx,
		admin,
		original,
	)
	if _, err := restaurarPreparacionTerminalDecisionCoberturaDurable(
		consultaOriginal,
		contenidoTerminal,
	); err != nil {
		t.Fatalf(
			"carga terminal concedida no rehidratable: %v / %s",
			err,
			contenidoTerminal,
		)
	}
	ambitosTerminal, err :=
		codificarAmbitosConsultaDecisionCoberturaDurable(consultaOriginal)
	if err != nil {
		t.Fatal(err)
	}
	if _, existe, err := preparador.consultarEnTransaccion(
		ctx,
		consultaOriginal,
		ambitosTerminal,
	); err != nil || !existe {
		t.Fatalf(
			"lector terminal SQL directo: existe=%t err=%T %v",
			existe,
			err,
			err,
		)
	}
	confirmada, existe, err :=
		preparador.ConsultarOperacionDecisionCoberturaConfirmada(
			ctx,
			consultaOriginal,
		)
	if err != nil || !existe {
		t.Fatalf("replay concedido original ausente: %t / %v", existe, err)
	}
	recibo, err := confirmada.ReciboConfirmadoPara(consultaOriginal)
	if err != nil || !recibo.ConcedidaVEC || recibo.Aplicada == nil ||
		recibo.DenegadaVEC != nil {
		t.Fatalf("recibo concedido inválido: %#v / %v", recibo, err)
	}
	activo := parHMACDecisionCoberturaPrueba(2, "5", "6")
	consultaRotada, solicitudRotada :=
		solicitudPreparacionDecisionCoberturaPrueba(
			t,
			identidad,
			activo,
			original,
		)
	for intento := 1; intento <= 2; intento++ {
		if intento == 2 {
			_, solicitudRotada =
				solicitudPreparacionDecisionCoberturaPrueba(
					t,
					identidad,
					activo,
					original,
				)
		}
		replay, err :=
			preparador.ReservarOReapropiarOperacionDecisionCobertura(
				ctx,
				solicitudRotada,
			)
		if err != nil {
			t.Fatalf(
				"preparar terminal rotado, intento %d: %v",
				intento,
				err,
			)
		}
		if estado, err := replay.EstadoPara(solicitudRotada); err != nil ||
			estado != cobertura.PreparacionOperacionDecisionCoberturaConfirmada {
			t.Fatalf(
				"preparar terminal rotado no confirmó: %q / %v",
				estado,
				err,
			)
		}
	}
	confirmada, existe, err =
		preparador.ConsultarOperacionDecisionCoberturaConfirmada(
			ctx,
			consultaRotada,
		)
	if err != nil || !existe {
		t.Fatalf(
			"lector terminal tras alias activo: existe=%t err=%v",
			existe,
			err,
		)
	}
	recibo, err = confirmada.ReciboConfirmadoPara(consultaRotada)
	if err != nil ||
		recibo.AmbitoIdempotenciaHMAC != original.ambitoHMAC() ||
		recibo.HuellaSemanticaHMAC != original.semanticaHMAC() {
		t.Fatalf("lector no devolvió el par terminal autoritativo: %#v / %v", recibo, err)
	}
	var aliases int
	lectura, err := admin.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(lectura)
	if _, err := lectura.Exec(
		ctx,
		"SET LOCAL ROLE vec_contratacion_temporal_propietario",
	); err != nil {
		t.Fatal(err)
	}
	if err := lectura.QueryRow(ctx, `
		SELECT count(*)
		  FROM vec_contratacion_temporal.alias_operacion_decision_cobertura
		 WHERE ambito_raiz_hmac=$1`,
		original.ambitoHMAC(),
	).Scan(&aliases); err != nil || aliases != 2 {
		t.Fatalf("rotación de alias no durable: aliases=%d err=%v", aliases, err)
	}
	if err := lectura.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return consultaRotada
}

func cargarTerminalPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	par parHMACPreparacionDecisionCoberturaPrueba,
) string {
	t.Helper()
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer revertirTransaccion(tx)
	if _, err := tx.Exec(
		ctx,
		"SET LOCAL ROLE vec_contratacion_temporal_propietario",
	); err != nil {
		t.Fatal(err)
	}
	var contenido string
	if err := tx.QueryRow(ctx, `
		SELECT coalesce(
		  vec_contratacion_temporal.o404c_carga_terminal_v1($1)::text,
		  ''
		)`,
		par.ambitoHMAC(),
	).Scan(&contenido); err != nil {
		t.Fatal(err)
	}
	if contenido == "" {
		t.Fatal("carga terminal SQL nula")
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return contenido
}

func probarReplayDenegadoPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	preparador *PreparadorOperacionDecisionCoberturaDurablePostgreSQL,
	fixture fixturePreparacionDecisionCoberturaDurable,
) {
	t.Helper()
	identidad := fixture.identidad(
		t,
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a53",
		fixture.expediente.OrganizacionRef,
	)
	par := parHMACDecisionCoberturaPrueba(1, "7", "8")
	consulta, solicitud := solicitudPreparacionDecisionCoberturaPrueba(
		t,
		identidad,
		par,
	)
	if _, err :=
		preparador.ReservarOReapropiarOperacionDecisionCobertura(
			ctx,
			solicitud,
		); err != nil {
		t.Fatal(err)
	}
	confirmarTerminalPreparacionDecisionCobertura(
		t,
		ctx,
		admin,
		par,
		"denegada",
	)
	confirmada, existe, err :=
		preparador.ConsultarOperacionDecisionCoberturaConfirmada(
			ctx,
			consulta,
		)
	if err != nil || !existe {
		t.Fatalf("replay denegado ausente: %t / %v", existe, err)
	}
	recibo, err := confirmada.ReciboConfirmadoPara(consulta)
	if err != nil || recibo.ConcedidaVEC || recibo.Aplicada != nil ||
		recibo.DenegadaVEC == nil {
		t.Fatalf("recibo denegado mezcló ramas: %#v / %v", recibo, err)
	}
}

func probarColisionPreparacionDecisionCobertura(
	t *testing.T,
	ctx context.Context,
	preparador *PreparadorOperacionDecisionCoberturaDurablePostgreSQL,
	fixture fixturePreparacionDecisionCoberturaDurable,
) {
	t.Helper()
	original := fixture.identidad(
		t,
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a54",
		fixture.expediente.OrganizacionRef,
	)
	par := parHMACDecisionCoberturaPrueba(1, "9", "a")
	_, solicitud := solicitudPreparacionDecisionCoberturaPrueba(
		t,
		original,
		par,
	)
	if _, err :=
		preparador.ReservarOReapropiarOperacionDecisionCobertura(
			ctx,
			solicitud,
		); err != nil {
		t.Fatal(err)
	}
	colision := fixture.identidad(
		t,
		"018f3b2a-7c4d-4e5f-8a9b-0c1d2e3f4a55",
		"organizacion_ajena_o404c",
	)
	_, solicitudColision := solicitudPreparacionDecisionCoberturaPrueba(
		t,
		colision,
		par,
	)
	_, err :=
		preparador.ReservarOReapropiarOperacionDecisionCobertura(
			ctx,
			solicitudColision,
		)
	if !errors.Is(err, cobertura.ErrClaveOperacionDecisionCoberturaUsada) ||
		!errors.Is(
			err,
			cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
		) {
		t.Fatalf("colisión no cerrada: %v", err)
	}
}
