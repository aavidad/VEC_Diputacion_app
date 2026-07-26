package postgresimportacionconvoca

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

func TestIntegracionPostgreSQLIdempotenciaConcurrenciaYRecuperacion(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	repositorio := repositorioIntegracion(
		t, entorno, "politica:retencion:convoca:concurrencia", 365*24*time.Hour,
	)
	recuperador := recuperadorIntegracion(t, entorno)
	lote := loteIntegracion(
		huellaIntegracion("b"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 600,
	)
	assertContratoAltaSQLGo(t, ctx, entorno, lote)
	const trabajadores = 24
	inicio := make(chan struct{})
	errores := make(chan error, trabajadores)
	var nuevos atomic.Int32
	var grupo sync.WaitGroup
	for i := 0; i < trabajadores; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			_, reutilizada, err := repositorio.GuardarSiAusente(ctx, lote)
			if err == nil && !reutilizada {
				nuevos.Add(1)
			}
			errores <- err
		}()
	}
	close(inicio)
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("guardar concurrente: %v", err)
		}
	}
	if nuevos.Load() != 1 {
		t.Fatalf("CAS PostgreSQL tuvo %d ganadores", nuevos.Load())
	}
	recuperado, estado, existe, err := recuperador.RecuperarLote(
		ctx, lote.Acta.HuellaFicheroSHA256,
	)
	if err != nil || !existe || estado.EstadoStaging != aplicacion.EstadoStagingDisponible ||
		len(recuperado.Aceptadas) != len(lote.Aceptadas) ||
		recuperado.Aceptadas[0].Identidad.Nombre != "Persona" {
		t.Fatalf("recuperacion durable inesperada: existe=%v estado=%+v lote=%+v error=%v",
			existe, estado, recuperado, err)
	}
	var lotes, filas int
	err = entorno.admin.QueryRow(ctx, `
		SELECT (SELECT count(*) FROM vec_bolsa_importacion_convoca.lote
		         WHERE huella_fichero_sha256 = $1),
		       (SELECT count(*) FROM vec_bolsa_importacion_convoca.fila_staging
		         WHERE importacion_ref = $2)`,
		lote.Acta.HuellaFicheroSHA256, lote.Acta.ImportacionRef,
	).Scan(&lotes, &filas)
	if err != nil || lotes != 1 || filas != 600 {
		t.Fatalf("cardinalidad durable: lotes=%d filas=%d error=%v", lotes, filas, err)
	}
}

func TestIntegracionPostgreSQLRechazaActasConTiposJSONInvalidos(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	lote := loteIntegracion(
		huellaIntegracion("0"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 1,
	)
	actaValida, err := serializarActa(lote.Acta)
	if err != nil {
		t.Fatalf("serializar acta valida: %v", err)
	}
	protegido, err := entorno.protector.ProtegerStaging(
		ctx, SolicitudProteccionStaging{
			ImportacionRef:      lote.Acta.ImportacionRef,
			HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256,
			Esquema:             lote.Acta.Esquema,
			Filas:               lote.Aceptadas,
		},
	)
	if err != nil {
		t.Fatalf("proteger filas: %v", err)
	}
	defer borrarFilasProtegidas(protegido.Filas)
	filas, err := serializarFilasProtegidas(protegido.Filas)
	if err != nil {
		t.Fatalf("serializar filas protegidas: %v", err)
	}
	defer borrarBytes(actaValida, filas)

	casos := []struct {
		nombre string
		campo  string
		valor  any
	}{
		{nombre: "leidas nulas", campo: "filas_leidas", valor: nil},
		{nombre: "aceptadas nulas", campo: "filas_aceptadas", valor: nil},
		{nombre: "rechazadas nulas", campo: "filas_rechazadas", valor: nil},
		{nombre: "aceptadas como texto", campo: "filas_aceptadas", valor: "1"},
		{nombre: "aceptadas como decimal", campo: "filas_aceptadas", valor: 1.5},
		{nombre: "acta como booleano", campo: "acta_ref", valor: true},
		{nombre: "actor como booleano", campo: "actor_ref", valor: true},
		{nombre: "esquema como booleano", campo: "esquema", valor: true},
		{nombre: "custodia como booleano", campo: "fichero_custodiado_ref", valor: true},
		{nombre: "huella como booleano", campo: "huella_fichero_sha256", valor: true},
		{nombre: "importacion como booleano", campo: "importacion_ref", valor: true},
		{nombre: "instante como booleano", campo: "registrada_en", valor: true},
		{nombre: "actor con barra", campo: "actor_ref", valor: "actor/rrhh/prueba"},
		{nombre: "nombre con control", campo: "nombre_fichero", valor: "sintetico\n.xls"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var acta map[string]any
			if err := json.Unmarshal(actaValida, &acta); err != nil {
				t.Fatalf("interpretar acta valida: %v", err)
			}
			acta[caso.campo] = caso.valor
			actaInvalida, err := json.Marshal(acta)
			if err != nil {
				t.Fatalf("serializar acta invalida: %v", err)
			}
			defer borrarBytes(actaInvalida)
			_, err = entorno.ejecutor.Exec(ctx, `
				SELECT * FROM vec_bolsa_importacion_convoca.guardar_lote_v1(
				    $1::jsonb, $2::jsonb
				)`, json.RawMessage(actaInvalida), json.RawMessage(filas))
			if err == nil {
				t.Fatal("PostgreSQL acepto un acta con conteo nulo o mal tipado")
			}
			exigirSQLState(t, err, "B1701")
		})
	}
}

func TestIntegracionPostgreSQLRechazaTiposJSONProtegidosYPaginacionNula(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	lote := loteIntegracion(
		huellaIntegracion("1"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 1,
	)
	acta, err := serializarActa(lote.Acta)
	if err != nil {
		t.Fatalf("serializar acta valida: %v", err)
	}
	protegido, err := entorno.protector.ProtegerStaging(
		ctx, SolicitudProteccionStaging{
			ImportacionRef:      lote.Acta.ImportacionRef,
			HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256,
			Esquema:             lote.Acta.Esquema,
			Filas:               lote.Aceptadas,
		},
	)
	if err != nil {
		t.Fatalf("proteger filas: %v", err)
	}
	defer borrarFilasProtegidas(protegido.Filas)
	filasValidas, err := serializarFilasProtegidas(protegido.Filas)
	if err != nil {
		t.Fatalf("serializar filas protegidas: %v", err)
	}
	defer borrarBytes(acta, filasValidas)

	casos := []struct {
		nombre string
		campo  string
		valor  any
	}{
		{nombre: "esquema booleano", campo: "esquema_proteccion", valor: true},
		{nombre: "clave booleano", campo: "clave_ref", valor: true},
		{nombre: "clave derivacion booleano", campo: "clave_derivacion_ref", valor: true},
		{nombre: "clave atestacion booleano", campo: "clave_atestacion_ref", valor: true},
		{nombre: "huella cifrado booleano", campo: "huella_contenido_cifrado_sha256", valor: true},
		{nombre: "derivacion nula", campo: "derivacion_documento_hmac_sha256", valor: nil},
		{nombre: "atestacion nula", campo: "atestacion_fila_hmac_sha256", valor: nil},
		{nombre: "nonce booleano", campo: "nonce_hex", valor: true},
		{nombre: "cifrado booleano", campo: "contenido_cifrado_hex", valor: true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			var filas []map[string]any
			if err := json.Unmarshal(filasValidas, &filas); err != nil {
				t.Fatalf("interpretar filas validas: %v", err)
			}
			filas[0][caso.campo] = caso.valor
			filasInvalidas, err := json.Marshal(filas)
			if err != nil {
				t.Fatalf("serializar filas invalidas: %v", err)
			}
			defer borrarBytes(filasInvalidas)
			_, err = entorno.ejecutor.Exec(ctx, `
				SELECT * FROM vec_bolsa_importacion_convoca.guardar_lote_v1(
				    $1::jsonb, $2::jsonb
				)`, json.RawMessage(acta), json.RawMessage(filasInvalidas))
			if err == nil {
				t.Fatal("PostgreSQL acepto material protegido mal tipado")
			}
			exigirSQLState(t, err, "B1701")
		})
	}

	for _, consulta := range []string{
		`SELECT vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(
		    repeat('a', 64), NULL, 512
		)`,
		`SELECT vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(
		    repeat('a', 64), 2, NULL
		)`,
	} {
		_, err := entorno.recuperador.Exec(ctx, consulta)
		if err == nil {
			t.Fatal("PostgreSQL acepto paginacion nula")
		}
		exigirSQLState(t, err, "B1701")
	}
}

func TestIntegracionPostgreSQLRechazaMutacionesProtegidasCampoACampo(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	repositorio := repositorioIntegracion(
		t, entorno, "politica:retencion:convoca:integridad", 365*24*time.Hour,
	)
	lote := loteIntegracion(
		huellaIntegracion("7"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 1,
	)
	if _, _, err := repositorio.GuardarSiAusente(ctx, lote); err != nil {
		t.Fatalf("guardar lote para mutaciones: %v", err)
	}
	mutaciones := []struct {
		nombre           string
		asignacion       string
		rechazoEscritura bool
	}{
		{nombre: "numero", asignacion: `numero = 22`},
		{
			nombre: "esquema", rechazoEscritura: true,
			asignacion: `esquema_proteccion =
			    'vec.bolsa.importacion-convoca.proteccion-staging.v2'`,
		},
		{nombre: "clave_ref", asignacion: `clave_ref = 'kms:fixture:convoca:b1:v2'`},
		{
			nombre: "clave_derivacion_ref",
			asignacion: `clave_derivacion_ref =
			    'kms:fixture:convoca:derivacion-documento:v2'`,
		},
		{
			nombre: "clave_atestacion_ref",
			asignacion: `clave_atestacion_ref =
			    'kms:fixture:convoca:atestacion-fila:v2'`,
		},
		{nombre: "nonce", asignacion: `nonce = decode(repeat('ab', 12), 'hex')`},
		{
			nombre: "cifrado",
			asignacion: `contenido_cifrado = decode(repeat('cd', 16), 'hex'),
			    huella_contenido_cifrado_sha256 = encode(
			        sha256(decode(repeat('cd', 16), 'hex')), 'hex'
			    )`,
		},
		{
			nombre: "huella_cifrado", rechazoEscritura: true,
			asignacion: `huella_contenido_cifrado_sha256 = repeat('c', 64)`,
		},
		{
			nombre: "derivacion",
			asignacion: `derivacion_documento_hmac_sha256 =
			    decode(repeat('ab', 32), 'hex')`,
		},
		{
			nombre: "atestacion",
			asignacion: `atestacion_fila_hmac_sha256 =
			    decode(repeat('cd', 32), 'hex')`,
		},
	}
	for _, mutacion := range mutaciones {
		t.Run(mutacion.nombre, func(t *testing.T) {
			assertMutacionProtegidaRechazada(
				t, ctx, entorno, lote, mutacion.asignacion,
				mutacion.rechazoEscritura,
			)
		})
	}
}

func TestIntegracionPostgreSQLMismaHuellaActoresConcurrentesEnConflicto(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	repositorio := repositorioIntegracion(
		t, entorno, "politica:retencion:convoca:actores", 365*24*time.Hour,
	)
	loteA := loteIntegracion(
		huellaIntegracion("5"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 2,
	)
	loteA.Acta.ActorRef = "actor:rrhh:integracion:a"
	loteB := dominio.ClonarLote(loteA)
	loteB.Acta.ActorRef = "actor:rrhh:integracion:b"
	type respuesta struct {
		actor       string
		acta        dominio.ActaImportacion
		reutilizada bool
		err         error
	}
	inicio := make(chan struct{})
	respuestas := make(chan respuesta, 2)
	for _, lote := range []dominio.LoteValidado{loteA, loteB} {
		go func(valor dominio.LoteValidado) {
			<-inicio
			acta, reutilizada, err := repositorio.GuardarSiAusente(ctx, valor)
			respuestas <- respuesta{
				actor: valor.Acta.ActorRef, acta: acta,
				reutilizada: reutilizada, err: err,
			}
		}(lote)
	}
	close(inicio)
	var ganador string
	conflictos := 0
	for range 2 {
		respuesta := <-respuestas
		switch {
		case respuesta.err == nil && !respuesta.reutilizada:
			ganador = respuesta.actor
			if respuesta.acta.ActorRef != respuesta.actor {
				t.Fatalf("acta ganadora cambio actor: %+v", respuesta.acta)
			}
		case errors.Is(respuesta.err, aplicacion.ErrImportacionEnConflicto):
			conflictos++
		default:
			t.Fatalf("resultado concurrente inesperado: %+v", respuesta)
		}
	}
	if ganador == "" || conflictos != 1 {
		t.Fatalf("CAS por actor: ganador=%q conflictos=%d", ganador, conflictos)
	}
	recuperador := recuperadorIntegracion(t, entorno)
	estado, existe, err := recuperador.ConsultarEstado(ctx, loteA.Acta.HuellaFicheroSHA256)
	if err != nil || !existe || estado.Acta.ActorRef != ganador {
		t.Fatalf("actor durable no coincide con ganador: existe=%v actor=%q error=%v",
			existe, estado.Acta.ActorRef, err)
	}
}

func assertMutacionProtegidaRechazada(
	t *testing.T,
	ctx context.Context,
	entorno *entornoPostgreSQLIntegracion,
	lote dominio.LoteValidado,
	asignacion string,
	rechazoEscritura bool,
) {
	t.Helper()
	tx, err := entorno.admin.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		t.Fatalf("abrir mutacion protegida: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(
		ctx, `SET LOCAL session_replication_role = replica`,
	); err != nil {
		t.Fatalf("aislar mutacion protegida: %v", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE vec_bolsa_importacion_convoca.fila_staging
		   SET `+asignacion+`
		 WHERE importacion_ref = $1 AND numero = 2`,
		lote.Acta.ImportacionRef,
	)
	if rechazoEscritura {
		if err == nil {
			t.Fatal("almacenamiento acepto una mutacion incompatible")
		}
		exigirSQLState(t, err, "23514")
		return
	}
	if err != nil {
		t.Fatalf("preparar mutacion protegida: %v", err)
	}
	_, err = tx.Exec(ctx, `
		SELECT vec_bolsa_importacion_convoca.recuperar_lote_pagina_v1(
		    $1, 2, 512
		)`, lote.Acta.HuellaFicheroSHA256)
	if err == nil {
		t.Fatal("recuperacion acepto material protegido mutado")
	}
	exigirSQLState(t, err, "B1701")
}

func assertContratoAltaSQLGo(
	t *testing.T,
	ctx context.Context,
	entorno *entornoPostgreSQLIntegracion,
	lote dominio.LoteValidado,
) {
	t.Helper()
	acta, err := serializarActa(lote.Acta)
	if err != nil {
		t.Fatalf("serializar acta para cotejo SQL/Go: %v", err)
	}
	protegido, err := entorno.protector.ProtegerStaging(
		ctx, SolicitudProteccionStaging{
			ImportacionRef:      lote.Acta.ImportacionRef,
			HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256,
			Esquema:             lote.Acta.Esquema, Filas: lote.Aceptadas,
		},
	)
	if err != nil {
		t.Fatalf("proteger filas para cotejo SQL/Go: %v", err)
	}
	defer borrarFilasProtegidas(protegido.Filas)
	filas, err := serializarFilasProtegidas(protegido.Filas)
	if err != nil {
		t.Fatalf("serializar filas para cotejo SQL/Go: %v", err)
	}
	defer borrarBytes(acta, filas)
	var actaValida, filasValidas bool
	if err := entorno.admin.QueryRow(ctx, `
		SELECT vec_bolsa_importacion_convoca.acta_valida($1::jsonb),
		       vec_bolsa_importacion_convoca.filas_protegidas_validas(
		           $2::jsonb, $3
		       )`, json.RawMessage(acta), json.RawMessage(filas),
		lote.Acta.FilasAceptadas,
	).Scan(&actaValida, &filasValidas); err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			t.Fatalf("cotejar contratos SQL/Go: %v detalle=%q donde=%q interna=%q",
				err, pg.Detail, pg.Where, pg.InternalQuery)
		}
		t.Fatalf("cotejar contratos SQL/Go: %v", err)
	}
	if !actaValida || !filasValidas {
		var instante, actor, custodia, procedencia, claves, nombre bool
		var referencias, esquema, ruta, conteos, incidencias bool
		diagnostico := entorno.admin.QueryRow(ctx, `
			SELECT
			  vec_bolsa_importacion_convoca.instante_microsegundo_valido(
			      $1::jsonb->>'registrada_en'
			  ),
			  vec_bolsa_importacion_convoca.texto_opaco_valido(
			      $1::jsonb->>'actor_ref', 128
			  ),
			  vec_bolsa_importacion_convoca.texto_opaco_valido(
			      $1::jsonb->>'fichero_custodiado_ref', 512
			  ),
			  $1::jsonb->'procedencia' = jsonb_build_object(
			      'esquema', 'vec.bolsa.importacion-convoca.procedencia.v1',
			      'fuente', 'Convoca (exportacion enmascarada)',
			      'autoridad', 'no_autoritativa',
			      'habilita_actos_con_efectos', false,
			      'requiere_confirmacion_registro', true,
			      'uso_puntos_autobaremacion', 'historico_contraste'
			  ),
			  $1::jsonb - (ARRAY[
			      'acta_ref','importacion_ref','huella_fichero_sha256',
			      'fichero_custodiado_ref','nombre_fichero','actor_ref',
			      'registrada_en','esquema','filas_leidas',
			      'filas_aceptadas','filas_rechazadas','incidencias',
			      'procedencia'
			  ]::text[]) = '{}'::jsonb,
			  lower(right($1::jsonb->>'nombre_fichero', 4)) = '.xls',
			  vec_bolsa_importacion_convoca.huella_valida(
			      $1::jsonb->>'huella_fichero_sha256'
			  ) AND $1::jsonb->>'acta_ref' =
			      'acta:importacion-convoca:' ||
			      $1::jsonb->>'huella_fichero_sha256'
			    AND $1::jsonb->>'importacion_ref' =
			      'importacion:convoca:' ||
			      $1::jsonb->>'huella_fichero_sha256',
			  $1::jsonb->>'esquema' IN (
			      'convoca_resumen_persona_v1',
			      'convoca_detalle_merito_v1'
			  ),
			  strpos($1::jsonb->>'nombre_fichero', '/') = 0
			    AND strpos($1::jsonb->>'nombre_fichero', '\') = 0
			    AND $1::jsonb->>'nombre_fichero' =
			        btrim($1::jsonb->>'nombre_fichero'),
			  ($1::jsonb->>'filas_leidas')::integer >= 0
			    AND ($1::jsonb->>'filas_aceptadas')::integer
			      + ($1::jsonb->>'filas_rechazadas')::integer =
			        ($1::jsonb->>'filas_leidas')::integer,
			  jsonb_typeof($1::jsonb->'incidencias') = 'array'
			    AND (SELECT count(DISTINCT (value->>'fila')::integer)
			           FROM jsonb_array_elements(
			               $1::jsonb->'incidencias'
			           )) = ($1::jsonb->>'filas_rechazadas')::integer
			`, json.RawMessage(acta)).Scan(
			&instante, &actor, &custodia, &procedencia, &claves, &nombre,
			&referencias, &esquema, &ruta, &conteos, &incidencias,
		)
		t.Fatalf(
			"canon SQL/Go divergente: acta=%v filas=%v instante=%v actor=%v "+
				"custodia=%v procedencia=%v claves=%v nombre=%v refs=%v "+
				"esquema=%v ruta=%v conteos=%v incidencias=%v diag=%v json_acta=%s",
			actaValida, filasValidas, instante, actor, custodia, procedencia,
			claves, nombre, referencias, esquema, ruta, conteos, incidencias,
			diagnostico, acta,
		)
	}
	tx, err := entorno.admin.Begin(ctx)
	if err != nil {
		t.Fatalf("abrir diagnostico atomico: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var resultado []byte
	var reutilizada bool
	err = tx.QueryRow(ctx, `
		SELECT acta_canonica, reutilizada
		  FROM vec_bolsa_importacion_convoca.guardar_lote_v1(
		      $1::jsonb, $2::jsonb
		  )`, json.RawMessage(acta), json.RawMessage(filas),
	).Scan(&resultado, &reutilizada)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			t.Fatalf(
				"alta SQL atomica tras validar canon: %v detalle=%q donde=%q interna=%q",
				err, pg.Detail, pg.Where, pg.InternalQuery,
			)
		}
		t.Fatalf("alta SQL atomica tras validar canon: %v", err)
	}
	if reutilizada {
		t.Fatal("alta diagnostica vacia se marco reutilizada")
	}
}

func TestIntegracionPostgreSQLConciliacionIdempotenteYConflicto(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	repositorio := repositorioIntegracion(
		t, entorno, "politica:retencion:convoca:conciliacion", 365*24*time.Hour,
	)
	recuperador := recuperadorIntegracion(t, entorno)
	lote := loteIntegracion(
		huellaIntegracion("c"), time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC), 1,
	)
	if _, _, err := repositorio.GuardarSiAusente(ctx, lote); err != nil {
		t.Fatalf("guardar lote para conciliar: %v", err)
	}
	conciliador, err := NuevoRepositorioConciliacionesPostgreSQL(entorno.conciliador)
	if err != nil {
		t.Fatalf("construir conciliador: %v", err)
	}
	solicitud := aplicacion.SolicitudConciliacion{
		ImportacionRef:         lote.Acta.ImportacionRef,
		ConciliacionRef:        "conciliacion:convoca:integracion:c",
		RegistroCorporativoRef: "registro:corporativo:opaco:c",
		Resultado:              aplicacion.ResultadoConciliadoConfirmado,
		ActorRef:               "actor:rrhh:conciliador", MotivoCodigo: "confirmacion_registro",
	}
	primera, err := conciliador.Conciliar(ctx, solicitud)
	if err != nil || primera.Reutilizada {
		t.Fatalf("primera conciliacion: %+v, %v", primera, err)
	}
	repetida, err := conciliador.Conciliar(ctx, solicitud)
	if err != nil || !repetida.Reutilizada ||
		!repetida.RegistradaEn.Equal(primera.RegistradaEn) {
		t.Fatalf("replay conciliacion: %+v, %v", repetida, err)
	}
	conflicto := solicitud
	conflicto.ConciliacionRef = "conciliacion:convoca:integracion:c:otra"
	if _, err := conciliador.Conciliar(ctx, conflicto); !errors.Is(
		err, aplicacion.ErrConciliacionEnConflicto,
	) {
		t.Fatalf("segunda conciliacion incompatible aceptada: %v", err)
	}
	estado, existe, err := recuperador.ConsultarEstado(ctx, lote.Acta.HuellaFicheroSHA256)
	if err != nil || !existe ||
		estado.EstadoConciliacion != aplicacion.EstadoConciliacionConfirmada ||
		estado.Version != 2 {
		t.Fatalf("estado conciliado inesperado: %+v existe=%v error=%v", estado, existe, err)
	}
	var eventos, historia int
	var payloadMinimo, cadenaIntegra bool
	if err := entorno.admin.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM vec_bolsa_importacion_convoca.outbox
		    WHERE importacion_ref = $1),
		  (SELECT count(*) FROM vec_bolsa_importacion_convoca.historia_estado
		    WHERE importacion_ref = $1),
		  (SELECT payload ?& ARRAY[
		      'esquema','evento_ref','importacion_ref','acta_ref',
		      'fichero_custodiado_ref','huella_fichero_sha256',
		      'huella_acta_sha256','conciliacion_ref',
		      'registro_corporativo_ref','procedencia'
		  ]::text[] AND NOT payload ?| ARRAY[
		      'actor_ref','nombre_fichero','nombre','documento'
		  ]::text[]
		     FROM vec_bolsa_importacion_convoca.outbox
		    WHERE importacion_ref = $1),
		  vec_bolsa_importacion_convoca.historia_integra($1)`,
		lote.Acta.ImportacionRef,
	).Scan(&eventos, &historia, &payloadMinimo, &cadenaIntegra); err != nil ||
		eventos != 1 || historia != 2 || !payloadMinimo || !cadenaIntegra {
		t.Fatalf(
			"outbox/cadena inesperada: eventos=%d historia=%d payload=%v cadena=%v error=%v",
			eventos, historia, payloadMinimo, cadenaIntegra, err,
		)
	}
	if _, err := entorno.admin.Exec(ctx, `
		UPDATE vec_bolsa_importacion_convoca.historia_estado
		   SET actor_ref = actor_ref WHERE importacion_ref = $1`,
		lote.Acta.ImportacionRef,
	); err == nil {
		t.Fatal("historia permitio mutacion privilegiada")
	} else {
		exigirSQLState(t, err, "55000")
	}
	if _, err := entorno.admin.Exec(ctx, `
		UPDATE vec_bolsa_importacion_convoca.outbox
		   SET payload = payload WHERE importacion_ref = $1`,
		lote.Acta.ImportacionRef,
	); err == nil {
		t.Fatal("outbox permitio mutacion privilegiada")
	} else {
		exigirSQLState(t, err, "55000")
	}
}

func TestIntegracionPostgreSQLBloqueoExpurgoYReimportacion(t *testing.T) {
	entorno := abrirEntornoPostgreSQLIntegracion(t)
	ctx, cancelar := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelar()
	politica := "politica:retencion:convoca:expurgo"
	repositorio := repositorioIntegracion(t, entorno, politica, time.Hour)
	recuperador := recuperadorIntegracion(t, entorno)
	lote := loteIntegracion(
		huellaIntegracion("d"), time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), 2,
	)
	if _, _, err := repositorio.GuardarSiAusente(ctx, lote); err != nil {
		t.Fatalf("guardar lote vencido: %v", err)
	}
	conservador, err := NuevoRepositorioRetencionPostgreSQL(entorno.retencion)
	if err != nil {
		t.Fatalf("construir conservador: %v", err)
	}
	futuro := aplicacion.SolicitudExpurgoStaging{
		EjecucionRef: "expurgo:convoca:integracion:d:antes-plazo",
		ActorRef:     "actor:archivo:integracion", PoliticaRef: politica,
		PoliticaVersion: 1, Limite: 10,
	}
	if resultado, err := conservador.ExpurgarVencidos(ctx, futuro); err != nil ||
		resultado.Lotes != 0 || resultado.Filas != 0 {
		t.Fatalf("expurgo antes de plazo: %+v, %v", resultado, err)
	}
	politicaIncorrecta := futuro
	politicaIncorrecta.EjecucionRef = "expurgo:convoca:integracion:d:politica-inexistente"
	politicaIncorrecta.PoliticaVersion = 2
	if _, err := conservador.ExpurgarVencidos(
		ctx, politicaIncorrecta,
	); !errors.Is(err, ErrLoteNoConfiable) {
		t.Fatalf("expurgo acepto version de politica inexistente: %v", err)
	}
	forzarVencimientoIntegracion(t, ctx, entorno, lote.Acta.ImportacionRef)
	bloqueo := aplicacion.SolicitudCambioBloqueoRetencion{
		ImportacionRef: lote.Acta.ImportacionRef,
		DecisionRef:    "retencion:convoca:integracion:d:bloquear",
		ActorRef:       "actor:archivo:integracion", MotivoCodigo: "conservacion_cautelar",
		Bloqueado: true,
	}
	if resultado, err := conservador.CambiarBloqueo(ctx, bloqueo); err != nil ||
		resultado.Reutilizada {
		t.Fatalf("bloquear retencion: %+v, %v", resultado, err)
	}
	expurgoBloqueado := aplicacion.SolicitudExpurgoStaging{
		EjecucionRef: "expurgo:convoca:integracion:d:bloqueado",
		ActorRef:     "actor:archivo:integracion", PoliticaRef: politica,
		PoliticaVersion: 1, Limite: 10,
	}
	if resultado, err := conservador.ExpurgarVencidos(ctx, expurgoBloqueado); err != nil ||
		resultado.Lotes != 0 || resultado.Filas != 0 {
		t.Fatalf("expurgo atraveso bloqueo: %+v, %v", resultado, err)
	}
	desbloqueo := bloqueo
	desbloqueo.DecisionRef = "retencion:convoca:integracion:d:desbloquear"
	desbloqueo.MotivoCodigo = "fin_conservacion_cautelar"
	desbloqueo.Bloqueado = false
	if _, err := conservador.CambiarBloqueo(ctx, desbloqueo); err != nil {
		t.Fatalf("desbloquear retencion: %v", err)
	}
	expurgo := aplicacion.SolicitudExpurgoStaging{
		EjecucionRef: "expurgo:convoca:integracion:d:definitivo",
		ActorRef:     "actor:archivo:integracion", PoliticaRef: politica,
		PoliticaVersion: 1, Limite: 10,
	}
	resultado, err := conservador.ExpurgarVencidos(ctx, expurgo)
	if err != nil || resultado.Lotes != 1 || resultado.Filas != 2 ||
		resultado.Reutilizada {
		t.Fatalf("expurgo definitivo: %+v, %v", resultado, err)
	}
	replay, err := conservador.ExpurgarVencidos(ctx, expurgo)
	if err != nil || !replay.Reutilizada || replay.Lotes != 1 || replay.Filas != 2 {
		t.Fatalf("replay expurgo: %+v, %v", replay, err)
	}
	estado, existe, err := recuperador.ConsultarEstado(ctx, lote.Acta.HuellaFicheroSHA256)
	if err != nil || !existe || estado.EstadoStaging != aplicacion.EstadoStagingExpurgado {
		t.Fatalf("acta tras expurgo: %+v existe=%v error=%v", estado, existe, err)
	}
	if _, _, existe, err := recuperador.RecuperarLote(
		ctx, lote.Acta.HuellaFicheroSHA256,
	); !existe || !errors.Is(err, aplicacion.ErrStagingExpurgado) {
		t.Fatalf("staging expurgado recuperable: existe=%v error=%v", existe, err)
	}
	acta, reutilizada, err := repositorio.GuardarSiAusente(ctx, lote)
	if err != nil || !reutilizada || acta.ActaRef != lote.Acta.ActaRef {
		t.Fatalf("reimportacion tras expurgo: reutilizada=%v acta=%+v error=%v",
			reutilizada, acta, err)
	}
}
