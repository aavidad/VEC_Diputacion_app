package postgres

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

const (
	variableDSNRevocador          = "VEC_POSTGRES_TEST_IDENTIDAD_REVOCADOR_DSN"
	variableDSNRevalidacionAjena  = "VEC_POSTGRES_TEST_IDENTIDAD_REVALIDACION_AJENA_DSN"
	variableDSNRevalidacionAcceso = "VEC_POSTGRES_TEST_IDENTIDAD_REVALIDACION_ACCESO_DSN"
	variableDSNAdmin              = "VEC_POSTGRES_TEST_IDENTIDAD_ADMIN_DSN"
)

func TestIntegracionRevalidadorAutenticacionActorPostgreSQL18(t *testing.T) {
	dsnRegistro := os.Getenv(variableDSNRegistro)
	dsnRevalidacion := os.Getenv(variableDSNRevalidacion)
	dsnProvisionador := os.Getenv(variableDSNProvisionador)
	dsnRevocador := os.Getenv(variableDSNRevocador)
	dsnMixto := os.Getenv(variableDSNMixto)
	dsnRevalidacionAjena := os.Getenv(variableDSNRevalidacionAjena)
	dsnRevalidacionAcceso := os.Getenv(variableDSNRevalidacionAcceso)
	dsnAdmin := os.Getenv(variableDSNAdmin)
	if dsnRegistro == "" || dsnRevalidacion == "" ||
		dsnProvisionador == "" || dsnRevocador == "" || dsnMixto == "" ||
		dsnRevalidacionAjena == "" || dsnRevalidacionAcceso == "" || dsnAdmin == "" {
		t.Skip("ejecute deploy/postgresql/identidad_sesiones_v1/probar_integracion.sh")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	poolRegistro := abrirPoolIntegracion(t, ctx, dsnRegistro)
	defer poolRegistro.Close()
	poolRevalidacion := abrirPoolIntegracion(t, ctx, dsnRevalidacion)
	defer poolRevalidacion.Close()
	poolProvisionador := abrirPoolIntegracion(t, ctx, dsnProvisionador)
	defer poolProvisionador.Close()
	poolRevocador := abrirPoolIntegracion(t, ctx, dsnRevocador)
	defer poolRevocador.Close()
	poolMixto := abrirPoolIntegracion(t, ctx, dsnMixto)
	defer poolMixto.Close()
	poolRevalidacionAjena := abrirPoolIntegracion(t, ctx, dsnRevalidacionAjena)
	defer poolRevalidacionAjena.Close()
	poolRevalidacionAcceso := abrirPoolIntegracion(t, ctx, dsnRevalidacionAcceso)
	defer poolRevalidacionAcceso.Close()
	poolAdmin := abrirPoolIntegracion(t, ctx, dsnAdmin)
	defer poolAdmin.Close()

	for nombre, poolInvalido := range map[string]*pgxpool.Pool{
		"registro":                        poolRegistro,
		"provisionador":                   poolProvisionador,
		"revocador":                       poolRevocador,
		"mixto":                           poolMixto,
		"revalidador con rol ajeno":       poolRevalidacionAjena,
		"revalidador con rol propietario": poolRevalidacionAcceso,
	} {
		revalidadorInvalido, err := NuevoRevalidadorAutenticacionActorPostgreSQL(
			ctx, poolInvalido,
		)
		if revalidadorInvalido != nil ||
			!errors.Is(err, ErrRevalidadorAutenticacionActorNoDisponible) {
			t.Fatalf("el constructor acepto el pool %s", nombre)
		}
	}
	revalidador, err := NuevoRevalidadorAutenticacionActorPostgreSQL(
		ctx, poolRevalidacion,
	)
	if err != nil {
		t.Fatal("acreditar el pool exclusivo de revalidacion rica")
	}
	probarRechazoMembresiaIndirectaIntegracion(
		t, ctx, poolAdmin, poolRevalidacion,
	)
	probarRechazoPerfilAcreditadoAdicionalIntegracion(
		t, ctx, poolAdmin, poolRevalidacion,
	)

	seudonimosBase := SeudonimosAlta{
		Esquema:          EsquemaHMACSHA256V1,
		EspacioIdentidad: espacioIdentidadPrueba,
		DominioRef:       dominioHMACPrueba,
		ClaveID:          "clave-hsm-revalidacion-rica",
		ClaveVersion:     11,
		AsercionIDHMAC:   [32]byte{0xd1},
		SesionIDHMAC:     [32]byte{0xd2},
		SujetoIDHMAC:     [32]byte{0xd3},
		CuentaIDHMAC:     [32]byte{0xd4},
	}
	cuentaRef := provisionarCuentaIntegracionRevalidador(
		t, ctx, poolProvisionador, seudonimosBase,
	)

	altaBase, confirmacionBase := registrarSesionIntegracionRevalidador(
		t, ctx, poolRegistro, poolRevalidacion, seudonimosBase,
		"base", '6', 4*time.Minute,
	)
	resultado := revalidarIntegracionActor(
		t, ctx, revalidador, confirmacionBase.AutenticacionRef,
		confirmacionBase.SesionRef,
	)
	comprobarProyeccionIntegracionRevalidador(
		t, resultado, altaBase, confirmacionBase, cuentaRef,
	)
	probarLigaduraAsercionIntegracionRevalidador(
		t, ctx, poolAdmin, confirmacionBase,
	)

	seudonimosSegunda := seudonimosBase
	seudonimosSegunda.AsercionIDHMAC = [32]byte{0xd5}
	seudonimosSegunda.SesionIDHMAC = [32]byte{0xd6}
	_, confirmacionSegunda := registrarSesionIntegracionRevalidador(
		t, ctx, poolRegistro, poolRevalidacion, seudonimosSegunda,
		"segunda", '7', 4*time.Minute,
	)
	for _, cruzada := range []domain.SolicitudRevalidacionAutenticacionActorV1{
		{
			AutenticacionRef: confirmacionBase.AutenticacionRef,
			SesionRef:        confirmacionSegunda.SesionRef,
		},
		{
			AutenticacionRef: confirmacionSegunda.AutenticacionRef,
			SesionRef:        confirmacionBase.SesionRef,
		},
		{
			AutenticacionRef: referencia("aut_", "x"),
			SesionRef:        referencia("ses_", "y"),
		},
	} {
		if _, err = revalidador.RevalidarAutenticacionActorV1(ctx, cruzada); !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
			t.Fatal("una referencia ausente o cruzada produjo autenticacion")
		}
	}

	var revisionRevocada string
	err = poolRevocador.QueryRow(ctx, `
		SELECT vec_identidad_sesiones_v1.revocar_sesion_v1($1,$2,$3,$4)`,
		confirmacionBase.SesionRef,
		confirmacionBase.ControlSesionRef,
		"1",
		referencia("opr_", "R"),
	).Scan(&revisionRevocada)
	if err != nil || revisionRevocada != "2" {
		t.Fatal("revocar la sesion desde su capacidad exclusiva")
	}
	if _, err = revalidador.RevalidarAutenticacionActorV1(
		ctx,
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: confirmacionBase.AutenticacionRef,
			SesionRef:        confirmacionBase.SesionRef,
		},
	); !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
		t.Fatal("una sesion revocada produjo autenticacion rica")
	}

	var revisionCuenta string
	err = poolRevocador.QueryRow(ctx, `
		SELECT vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1($1,$2,$3,$4)`,
		cuentaRef, "1", "inactiva", referencia("opr_", "I"),
	).Scan(&revisionCuenta)
	if err != nil || revisionCuenta != "2" {
		t.Fatal("inactivar la cuenta desde su capacidad exclusiva")
	}
	if _, err = revalidador.RevalidarAutenticacionActorV1(
		ctx,
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: confirmacionSegunda.AutenticacionRef,
			SesionRef:        confirmacionSegunda.SesionRef,
		},
	); !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
		t.Fatal("una cuenta inactiva produjo autenticacion rica")
	}
	err = poolRevocador.QueryRow(ctx, `
		SELECT vec_identidad_sesiones_v1.cambiar_estado_cuenta_v1($1,$2,$3,$4)`,
		cuentaRef, "2", "activa", referencia("opr_", "A"),
	).Scan(&revisionCuenta)
	if err != nil || revisionCuenta != "3" {
		t.Fatal("reactivar la cuenta desde su capacidad exclusiva")
	}
	if _, err = revalidador.RevalidarAutenticacionActorV1(
		ctx,
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: confirmacionSegunda.AutenticacionRef,
			SesionRef:        confirmacionSegunda.SesionRef,
		},
	); !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
		t.Fatal("reactivar la cuenta resucito una sesion anterior")
	}

	probarExpiracionIntegracionRevalidador(
		t, ctx, poolRegistro, poolRevalidacion, poolProvisionador, revalidador,
	)
	probarEsperaYRevocacionIntegracionRevalidador(
		t, ctx, poolRegistro, poolRevalidacion, poolProvisionador,
		poolAdmin, revalidador,
	)

	var restriccionesExactas int
	if err = poolAdmin.QueryRow(ctx, `
		SELECT count(*)
		  FROM pg_catalog.pg_constraint AS restriccion
		  JOIN pg_catalog.pg_class AS tabla ON tabla.oid = restriccion.conrelid
		  JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid = tabla.relnamespace
		 WHERE espacio.nspname = 'vec_autorizacion'
		   AND tabla.relname = 'sesion_autenticacion_v1'
		   AND restriccion.contype = 'u'
		   AND pg_catalog.pg_get_constraintdef(restriccion.oid) =
		       'UNIQUE (autenticacion_ref, sesion_ref)'`).Scan(&restriccionesExactas); err != nil ||
		restriccionesExactas != 1 {
		t.Fatal("la cardinalidad exacta autenticacion/sesion no esta protegida")
	}
}

func probarRechazoMembresiaIndirectaIntegracion(
	t *testing.T,
	ctx context.Context,
	poolAdmin, poolRevalidacion *pgxpool.Pool,
) {
	t.Helper()
	const rolIndirecto = "vec_identidad_rol_indirecto_prueba"
	if _, err := poolAdmin.Exec(ctx, `
		CREATE ROLE vec_identidad_rol_indirecto_prueba NOLOGIN NOSUPERUSER
			NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
		GRANT vec_identidad_rol_indirecto_prueba
			TO vec_identidad_sesiones_v1_revalidador
			WITH ADMIN FALSE, INHERIT TRUE, SET FALSE`); err != nil {
		t.Fatal("crear membresia indirecta heredable de prueba")
	}
	pendienteLimpieza := true
	limpiar := func() {
		if !pendienteLimpieza {
			return
		}
		_, _ = poolAdmin.Exec(context.Background(), `
			REVOKE vec_identidad_rol_indirecto_prueba
				FROM vec_identidad_sesiones_v1_revalidador;
			DROP ROLE IF EXISTS vec_identidad_rol_indirecto_prueba`)
		pendienteLimpieza = false
	}
	defer limpiar()

	var autoridadHeredada bool
	if err := poolRevalidacion.QueryRow(ctx, `
		SELECT pg_catalog.pg_has_role(
			session_user, $1::text, 'USAGE'
		)`, rolIndirecto).Scan(&autoridadHeredada); err != nil || !autoridadHeredada {
		t.Fatal("PostgreSQL no acredito la autoridad indirecta heredada de prueba")
	}
	revalidador, err := NuevoRevalidadorAutenticacionActorPostgreSQL(
		ctx, poolRevalidacion,
	)
	if revalidador != nil ||
		!errors.Is(err, ErrRevalidadorAutenticacionActorNoDisponible) {
		t.Fatal("el constructor acepto autoridad indirecta heredable")
	}
	limpiar()
	if _, err = NuevoRevalidadorAutenticacionActorPostgreSQL(
		ctx, poolRevalidacion,
	); err != nil {
		t.Fatal("la acreditacion exclusiva no se restauro tras retirar la cadena")
	}
}

type mutacionPerfilAcreditadoIntegracion struct {
	nombre   string
	aplicar  string
	revertir string
}

func probarRechazoPerfilAcreditadoAdicionalIntegracion(
	t *testing.T,
	ctx context.Context,
	poolAdmin, poolRevalidacion *pgxpool.Pool,
) {
	t.Helper()
	var login, base string
	if err := poolRevalidacion.QueryRow(ctx, `SELECT session_user::text`).Scan(&login); err != nil {
		t.Fatal("obtener el LOGIN sometido a acreditacion")
	}
	if err := poolAdmin.QueryRow(ctx, `SELECT current_database()::text`).Scan(&base); err != nil {
		t.Fatal("obtener la base sometida a acreditacion")
	}

	loginSQL := pgx.Identifier{login}.Sanitize()
	baseSQL := pgx.Identifier{base}.Sanitize()
	grupoSQL := pgx.Identifier{capacidadRevalidar}.Sanitize()
	const rolAjeno = "vec_identidad_rol_ajeno_prueba"
	rolAjenoSQL := pgx.Identifier{rolAjeno}.Sanitize()
	esquemaLoginSQL := pgx.Identifier{
		"vec_identidad_objeto_login_acreditacion_prueba",
	}.Sanitize()
	esquemaGrupoSQL := pgx.Identifier{
		"vec_identidad_objeto_grupo_acreditacion_prueba",
	}.Sanitize()
	funcionRevalidacion := "vec_identidad_sesiones_v1." +
		"revalidar_autenticacion_actor_v1(text,text)"

	mutaciones := []mutacionPerfilAcreditadoIntegracion{
		{
			nombre: "EXECUTE directo al LOGIN",
			aplicar: "GRANT EXECUTE ON FUNCTION " + funcionRevalidacion +
				" TO " + loginSQL,
			revertir: "REVOKE EXECUTE ON FUNCTION " + funcionRevalidacion +
				" FROM " + loginSQL,
		},
		{
			nombre:   "privilegio adicional al grupo esperado",
			aplicar:  "GRANT TEMPORARY ON DATABASE " + baseSQL + " TO " + grupoSQL,
			revertir: "REVOKE TEMPORARY ON DATABASE " + baseSQL + " FROM " + grupoSQL,
		},
		{
			nombre: "objeto propiedad del LOGIN",
			aplicar: "CREATE SCHEMA " + esquemaLoginSQL +
				" AUTHORIZATION " + loginSQL,
			revertir: "DROP SCHEMA IF EXISTS " + esquemaLoginSQL + " CASCADE",
		},
		{
			nombre: "objeto propiedad del grupo esperado",
			aplicar: "CREATE SCHEMA " + esquemaGrupoSQL +
				" AUTHORIZATION " + grupoSQL,
			revertir: "DROP SCHEMA IF EXISTS " + esquemaGrupoSQL + " CASCADE",
		},
		{
			nombre: "ACL predeterminada directa al LOGIN",
			aplicar: "ALTER DEFAULT PRIVILEGES FOR ROLE " + rolAjenoSQL +
				" GRANT SELECT ON TABLES TO " + loginSQL,
			revertir: "ALTER DEFAULT PRIVILEGES FOR ROLE " + rolAjenoSQL +
				" REVOKE SELECT ON TABLES FROM " + loginSQL,
		},
		{
			nombre: "ACL predeterminada directa al grupo esperado",
			aplicar: "ALTER DEFAULT PRIVILEGES FOR ROLE " + rolAjenoSQL +
				" GRANT SELECT ON TABLES TO " + grupoSQL,
			revertir: "ALTER DEFAULT PRIVILEGES FOR ROLE " + rolAjenoSQL +
				" REVOKE SELECT ON TABLES FROM " + grupoSQL,
		},
		{
			nombre: "configuracion persistente del LOGIN",
			aplicar: "ALTER ROLE " + loginSQL +
				" SET application_name TO 'perfil_login_inseguro_prueba'",
			revertir: "ALTER ROLE " + loginSQL + " RESET application_name",
		},
		{
			nombre: "configuracion persistente del grupo esperado",
			aplicar: "ALTER ROLE " + grupoSQL +
				" SET application_name TO 'perfil_grupo_inseguro_prueba'",
			revertir: "ALTER ROLE " + grupoSQL + " RESET application_name",
		},
	}

	for _, mutacion := range mutaciones {
		mutacion := mutacion
		t.Run(mutacion.nombre, func(t *testing.T) {
			if _, err := poolAdmin.Exec(ctx, mutacion.aplicar); err != nil {
				t.Fatal("aplicar la mutacion del perfil acreditado")
			}
			pendienteLimpieza := true
			limpiar := func(ctxLimpieza context.Context) error {
				if !pendienteLimpieza {
					return nil
				}
				_, err := poolAdmin.Exec(ctxLimpieza, mutacion.revertir)
				if err == nil {
					pendienteLimpieza = false
				}
				return err
			}
			defer func() {
				if pendienteLimpieza {
					_ = limpiar(context.Background())
				}
			}()

			revalidador, err := NuevoRevalidadorAutenticacionActorPostgreSQL(
				ctx, poolRevalidacion,
			)
			if revalidador != nil ||
				!errors.Is(err, ErrRevalidadorAutenticacionActorNoDisponible) {
				t.Fatal("el constructor acepto autoridad fuera del manifiesto")
			}
			if err = limpiar(ctx); err != nil {
				t.Fatal("retirar la mutacion del perfil acreditado")
			}
			if _, err = NuevoRevalidadorAutenticacionActorPostgreSQL(
				ctx, poolRevalidacion,
			); err != nil {
				t.Fatal("el perfil exacto no se restauro tras retirar la mutacion")
			}
		})
	}
}

func provisionarCuentaIntegracionRevalidador(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	seudonimos SeudonimosAlta,
) string {
	t.Helper()
	var cuentaRef string
	rellenoOperacion := string(rune('V' + int(seudonimos.ClaveVersion-11)))
	err := pool.QueryRow(ctx, `
		SELECT cuenta_ref
		  FROM vec_identidad_sesiones_v1.provisionar_cuenta_v1(
		       $1,$2,$3,$4,$5,$6,$7,$8,$9
		  )`,
		referencia("opr_", rellenoOperacion),
		seudonimos.Esquema,
		seudonimos.DominioRef,
		seudonimos.ClaveID,
		int64(seudonimos.ClaveVersion),
		seudonimos.CuentaIDHMAC[:],
		seudonimos.SujetoIDHMAC[:],
		false,
		nil,
	).Scan(&cuentaRef)
	if err != nil || !referenciaTecnicaValida(cuentaRef, "cta_") {
		t.Fatal("provisionar cuenta para revalidacion rica")
	}
	return cuentaRef
}

func registrarSesionIntegracionRevalidador(
	t *testing.T,
	ctx context.Context,
	poolRegistro, poolRevalidacion *pgxpool.Pool,
	seudonimos SeudonimosAlta,
	sufijo string,
	huella rune,
	ttl time.Duration,
) (httpseguridad.AltaSesionAtomica, httpseguridad.ConfirmacionAltaSesion) {
	t.Helper()
	adaptador, err := NuevoRegistroSesionesPostgreSQL(
		ctx,
		poolRegistro,
		poolRevalidacion,
		seudonimizadorIntegracion{resultado: seudonimos},
		espacioIdentidadPrueba,
		dominioHMACPrueba,
	)
	if err != nil {
		t.Fatal("componer registro para revalidacion rica")
	}
	emitida := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Second)
	alta := httpseguridad.AltaSesionAtomica{
		EspacioIdentidad:             espacioIdentidadPrueba,
		AsercionID:                   "asercion-rica-" + sufijo,
		SesionID:                     "sesion-rica-" + sufijo,
		SujetoID:                     "sujeto-rica",
		CuentaID:                     "cuenta-rica",
		Superficie:                   httpseguridad.SuperficieInternaCorporativa,
		MetodoObservado:              domain.AuthMethodKerberos,
		GarantiaObservada:            domain.AuthAssuranceHigh,
		AutenticacionHuellaSHA256:    strings.Repeat(string(huella), 64),
		AutenticacionVerificadaEn:    emitida.Add(-time.Second),
		SesionEmitidaEn:              emitida,
		AsercionExpiraEn:             emitida.Add(ttl),
		PoliticaGarantiaRef:          referencia("pga_", "r"),
		PoliticaGarantiaHuellaSHA256: strings.Repeat("f", 64),
	}
	confirmacion, err := adaptador.ConsumirAsercionYRegistrar(ctx, alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil {
		t.Fatal("registrar sesion para revalidacion rica")
	}
	return alta, confirmacion
}

func revalidarIntegracionActor(
	t *testing.T,
	ctx context.Context,
	revalidador *RevalidadorAutenticacionActorPostgreSQL,
	autenticacionRef, sesionRef string,
) domain.AutenticacionRevalidadaV1 {
	t.Helper()
	resultado, err := revalidador.RevalidarAutenticacionActorV1(
		ctx,
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacionRef,
			SesionRef:        sesionRef,
		},
	)
	if err != nil || resultado.Validar() != nil {
		t.Fatal("revalidar autenticacion rica")
	}
	return resultado
}

func comprobarProyeccionIntegracionRevalidador(
	t *testing.T,
	resultado domain.AutenticacionRevalidadaV1,
	alta httpseguridad.AltaSesionAtomica,
	confirmacion httpseguridad.ConfirmacionAltaSesion,
	cuentaRef string,
) {
	t.Helper()
	if resultado.AutenticacionRef != confirmacion.AutenticacionRef ||
		resultado.AutenticacionHuellaSHA256 != alta.AutenticacionHuellaSHA256 ||
		resultado.AsercionRef != confirmacion.AsercionRef ||
		resultado.SesionRef != confirmacion.SesionRef ||
		resultado.ControlSesionRef != confirmacion.ControlSesionRef ||
		resultado.ControlSesionRevision != confirmacion.ControlSesionRevision ||
		resultado.ControlSesionHuellaSHA256 != confirmacion.ControlSesionHuellaSHA256 ||
		resultado.CuentaRef != cuentaRef || resultado.CuentaOrdinariaRef != cuentaRef ||
		resultado.CuentaPrivilegiada ||
		resultado.Superficie != domain.SuperficieAutenticacionInternaCorporativaV1 ||
		resultado.MetodoObservado != alta.MetodoObservado ||
		resultado.GarantiaObservada != alta.GarantiaObservada ||
		resultado.PoliticaGarantiaRef != alta.PoliticaGarantiaRef ||
		resultado.PoliticaGarantiaHuellaSHA256 != alta.PoliticaGarantiaHuellaSHA256 ||
		!resultado.AutenticacionVerificadaEn.Equal(alta.AutenticacionVerificadaEn) ||
		!resultado.SesionEmitidaEn.Equal(alta.SesionEmitidaEn) ||
		!resultado.SesionValidaHasta.Equal(confirmacion.SesionValidaHasta) ||
		!resultado.SesionRevalidadaEn.Equal(confirmacion.SesionRevalidadaEn) {
		t.Fatal("la proyeccion rica no coincide exactamente con el ledger")
	}
}

func probarLigaduraAsercionIntegracionRevalidador(
	t *testing.T,
	ctx context.Context,
	poolAdmin *pgxpool.Pool,
	confirmacion httpseguridad.ConfirmacionAltaSesion,
) {
	t.Helper()
	tx, err := poolAdmin.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatal("abrir transaccion adversarial de asercion")
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(ctx, `
		ALTER TABLE vec_identidad_sesiones_v1.consumo_asercion
		DISABLE TRIGGER consumo_asercion_inmutable`); err != nil {
		t.Fatal("aislar la corrupcion adversarial de asercion")
	}
	if _, err = tx.Exec(ctx, `
		UPDATE vec_identidad_sesiones_v1.consumo_asercion
		   SET asercion_ref = $1
		 WHERE sesion_ref = $2`,
		referencia("ase_", "z"),
		confirmacion.SesionRef,
	); err != nil {
		t.Fatal("simular una asercion desligada")
	}
	var filas int
	if err = tx.QueryRow(ctx, `
		SELECT count(*)
		  FROM vec_identidad_sesiones_v1.revalidar_autenticacion_actor_v1($1,$2)`,
		confirmacion.AutenticacionRef,
		confirmacion.SesionRef,
	).Scan(&filas); err != nil || filas != 0 {
		t.Fatal("la revalidacion rica acepto una asercion desligada")
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal("revertir la corrupcion adversarial de asercion")
	}
}

func probarExpiracionIntegracionRevalidador(
	t *testing.T,
	ctx context.Context,
	poolRegistro, poolRevalidacion, poolProvisionador *pgxpool.Pool,
	revalidador *RevalidadorAutenticacionActorPostgreSQL,
) {
	t.Helper()
	seudonimos := SeudonimosAlta{
		Esquema: EsquemaHMACSHA256V1, EspacioIdentidad: espacioIdentidadPrueba,
		DominioRef: dominioHMACPrueba, ClaveID: "clave-hsm-expiracion", ClaveVersion: 12,
		AsercionIDHMAC: [32]byte{0xe1}, SesionIDHMAC: [32]byte{0xe2},
		SujetoIDHMAC: [32]byte{0xe3}, CuentaIDHMAC: [32]byte{0xe4},
	}
	provisionarCuentaIntegracionRevalidador(t, ctx, poolProvisionador, seudonimos)
	_, confirmacion := registrarSesionIntegracionRevalidador(
		t, ctx, poolRegistro, poolRevalidacion, seudonimos,
		"expiracion", '8', 1500*time.Millisecond,
	)
	espera := time.Until(confirmacion.SesionValidaHasta) + 20*time.Millisecond
	if espera > 0 {
		temporizador := time.NewTimer(espera)
		select {
		case <-ctx.Done():
			temporizador.Stop()
			t.Fatal("el contexto vencio antes de probar expiracion")
		case <-temporizador.C:
		}
	}
	_, err := revalidador.RevalidarAutenticacionActorV1(
		ctx,
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: confirmacion.AutenticacionRef,
			SesionRef:        confirmacion.SesionRef,
		},
	)
	if !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
		t.Fatal("el extremo half-open de expiracion fue aceptado")
	}
}

func probarEsperaYRevocacionIntegracionRevalidador(
	t *testing.T,
	ctx context.Context,
	poolRegistro, poolRevalidacion, poolProvisionador, poolAdmin *pgxpool.Pool,
	revalidador *RevalidadorAutenticacionActorPostgreSQL,
) {
	t.Helper()
	seudonimos := SeudonimosAlta{
		Esquema: EsquemaHMACSHA256V1, EspacioIdentidad: espacioIdentidadPrueba,
		DominioRef: dominioHMACPrueba, ClaveID: "clave-hsm-espera", ClaveVersion: 13,
		AsercionIDHMAC: [32]byte{0xf1}, SesionIDHMAC: [32]byte{0xf2},
		SujetoIDHMAC: [32]byte{0xf3}, CuentaIDHMAC: [32]byte{0xf4},
	}
	provisionarCuentaIntegracionRevalidador(t, ctx, poolProvisionador, seudonimos)
	_, confirmacion := registrarSesionIntegracionRevalidador(
		t, ctx, poolRegistro, poolRevalidacion, seudonimos,
		"espera", '9', 4*time.Minute,
	)
	tx, err := poolAdmin.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatal("abrir transaccion retenedora")
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var revision string
	if err = tx.QueryRow(ctx, `
		SELECT revision::text
		  FROM vec_autorizacion.control_sesion_actual_v1
		 WHERE sesion_ref = $1
		 FOR UPDATE`, confirmacion.SesionRef).Scan(&revision); err != nil || revision != "1" {
		t.Fatal("retener el puntero de control")
	}

	resultado := make(chan error, 1)
	go func() {
		_, errRevalidacion := revalidador.RevalidarAutenticacionActorV1(
			ctx,
			domain.SolicitudRevalidacionAutenticacionActorV1{
				AutenticacionRef: confirmacion.AutenticacionRef,
				SesionRef:        confirmacion.SesionRef,
			},
		)
		resultado <- errRevalidacion
	}()
	var usuarioRevalidador string
	if err = poolRevalidacion.QueryRow(ctx, `SELECT session_user::text`).Scan(
		&usuarioRevalidador,
	); err != nil {
		t.Fatal("identificar el LOGIN revalidador")
	}
	bloqueada := false
	for !bloqueada && ctx.Err() == nil {
		if err = poolAdmin.QueryRow(ctx, `
			SELECT EXISTS (
			    SELECT 1 FROM pg_catalog.pg_stat_activity
			     WHERE usename = $1
			       AND state = 'active'
			       AND wait_event_type = 'Lock'
			       AND query LIKE '%revalidar_autenticacion_actor_v1%'
			)`, usuarioRevalidador).Scan(&bloqueada); err != nil {
			t.Fatal("observar la espera de revalidacion")
		}
	}
	if !bloqueada {
		t.Fatal("la revalidacion no espero el puntero actual")
	}
	if err = tx.QueryRow(ctx, `
		SELECT vec_identidad_sesiones_v1.revocar_sesion_v1($1,$2,$3,$4)`,
		confirmacion.SesionRef,
		confirmacion.ControlSesionRef,
		"1",
		referencia("opr_", "B"),
	).Scan(&revision); err != nil || revision != "2" {
		t.Fatal("revocar mientras la revalidacion esperaba")
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal("liberar el puntero revocado")
	}
	select {
	case err = <-resultado:
		if !errors.Is(err, domain.ErrAutenticacionRevalidadaInvalida) {
			t.Fatal("la revalidacion acepto la revision anterior tras esperar")
		}
	case <-ctx.Done():
		t.Fatal("la revalidacion no termino tras liberar el bloqueo")
	}
}
