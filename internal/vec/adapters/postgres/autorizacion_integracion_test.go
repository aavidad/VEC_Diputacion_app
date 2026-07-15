package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	variableDSNPostgreSQLPruebasFuente   = "VEC_POSTGRES_TEST_FUENTE_DSN"
	variableDSNPostgreSQLPruebasRegistro = "VEC_POSTGRES_TEST_REGISTRO_DSN"
	variableDSNPostgreSQLPruebasAdmin    = "VEC_POSTGRES_TEST_ADMIN_DSN"
)

type fixtureAutorizacionPostgreSQL struct {
	prefijo    string
	rol        domain.VersionRol
	controlRol domain.ControlVigenciaVersionRol
	asignacion domain.AsignacionPerfil
	politica   domain.PoliticaRestrictiva
	ahora      time.Time
}

type relojPostgreSQLPrueba struct{ ahora time.Time }

func (r relojPostgreSQLPrueba) Ahora() time.Time { return r.ahora }

type generadorPostgreSQLPrueba string

func (g generadorPostgreSQLPrueba) NuevaReferenciaDecisionAutorizacion() (string, error) {
	return string(g), nil
}

type revalidadorAutenticacionActorPostgreSQLIntegracion struct {
	resultado domain.AutenticacionRevalidadaV1
}

func (r revalidadorAutenticacionActorPostgreSQLIntegracion) RevalidarAutenticacionActorV1(
	context.Context,
	domain.SolicitudRevalidacionAutenticacionActorV1,
) (domain.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type registroDenegacionesPostgreSQLPrueba struct{}

func (registroDenegacionesPostgreSQLPrueba) RegistrarDenegacionAutorizacion(
	context.Context,
	domain.DecisionAutorizacion,
) error {
	return nil
}

type registroPostgreSQLConMutacion struct {
	destino  ports.RegistroDecisionesAutorizacion
	mutar    func(context.Context) error
	unaVez   sync.Once
	mutarErr error
}

func (r *registroPostgreSQLConMutacion) RegistrarDecisionSiInstantaneaVigente(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error {
	r.unaVez.Do(func() { r.mutarErr = r.mutar(ctx) })
	if r.mutarErr != nil {
		return r.mutarErr
	}
	return r.destino.RegistrarDecisionSiInstantaneaVigente(ctx, decision)
}

func TestIntegracionAutorizacionPostgreSQL(t *testing.T) {
	dsnFuente := os.Getenv(variableDSNPostgreSQLPruebasFuente)
	dsnRegistro := os.Getenv(variableDSNPostgreSQLPruebasRegistro)
	dsnAdmin := os.Getenv(variableDSNPostgreSQLPruebasAdmin)
	if dsnFuente == "" || dsnRegistro == "" || dsnAdmin == "" {
		t.Skipf("prueba PostgreSQL omitida: defina %s, %s y %s o ejecute deploy/postgresql/autorizacion/probar_integracion.sh",
			variableDSNPostgreSQLPruebasFuente, variableDSNPostgreSQLPruebasRegistro,
			variableDSNPostgreSQLPruebasAdmin)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	poolFuente, err := pgxpool.New(ctx, dsnFuente)
	if err != nil {
		t.Fatal("abrir pool fuente")
	}
	defer poolFuente.Close()
	poolRegistro, err := pgxpool.New(ctx, dsnRegistro)
	if err != nil {
		t.Fatal("abrir pool registro")
	}
	defer poolRegistro.Close()
	poolAdmin, err := pgxpool.New(ctx, dsnAdmin)
	if err != nil {
		t.Fatal("abrir pool administrativo")
	}
	defer poolAdmin.Close()
	if err = poolFuente.Ping(ctx); err != nil {
		t.Fatal("conectar fuente")
	}
	if err = poolRegistro.Ping(ctx); err != nil {
		t.Fatal("conectar registro")
	}
	if err = poolAdmin.Ping(ctx); err != nil {
		t.Fatal("conectar administrador")
	}
	almacenFuente, err := NuevoAlmacenAutorizacion(poolFuente)
	if err != nil {
		t.Fatalf("crear adaptador fuente: %v", err)
	}
	almacenRegistro, err := NuevoAlmacenAutorizacion(poolRegistro)
	if err != nil {
		t.Fatalf("crear adaptador registro: %v", err)
	}

	t.Run("fuente y registro tienen identidades y capacidades separadas", func(t *testing.T) {
		verificarPrivilegiosFuentePostgreSQL(t, ctx, poolFuente)
		verificarPrivilegiosRegistroPostgreSQL(t, ctx, poolRegistro)
	})

	ahora := time.Now().UTC().Truncate(time.Microsecond)
	fixture := nuevaFixtureAutorizacionPostgreSQL("principal", ahora)
	sembrarFixtureAutorizacionPostgreSQL(t, ctx, poolAdmin, fixture)

	t.Run("instantanea coherente decision append-only y copias independientes", func(t *testing.T) {
		instantanea, err := almacenFuente.ObtenerInstantaneaAutorizacion(
			ctx, fixture.asignacion.PrincipalID, fixture.asignacion.PerfilActivoRef,
		)
		if err != nil {
			t.Fatalf("obtener instantanea: %v", err)
		}
		if instantanea.VersionRol.Referencia() != fixture.rol.Referencia() ||
			instantanea.ControlVigenciaVersionRol.Revision != fixture.controlRol.Revision {
			t.Fatalf("instantanea distinta: %#v", instantanea)
		}
		instantanea.AsignacionPerfil.Ambitos[0].Valores[0] = "mutado"
		instantanea.Politicas[0].Restricciones[0].ValoresPermitidos[0] = "mutado"
		otra, err := almacenFuente.ObtenerInstantaneaAutorizacion(
			ctx, fixture.asignacion.PrincipalID, fixture.asignacion.PerfilActivoRef,
		)
		if err != nil {
			t.Fatalf("releer instantanea: %v", err)
		}
		if otra.AsignacionPerfil.Ambitos[0].Valores[0] != "granada" ||
			otra.Politicas[0].Restricciones[0].ValoresPermitidos[0] != "presentado" {
			t.Fatal("la lectura comparte memoria mutable")
		}

		autorizador := nuevoAutorizadorPostgreSQLPrueba(
			t, almacenFuente, almacenRegistro, ahora, "decision:postgres:principal",
		)
		decision, err := autorizador.Exigir(ctx, solicitudPostgreSQLPrueba(t, fixture))
		if err != nil || !decision.Concedida {
			t.Fatalf("autorizar y registrar: decision=%#v err=%v", decision, err)
		}
		guardada := obtenerDocumentoDecisionAdministrativamentePostgreSQL(t, ctx, poolAdmin, decision.DecisionRef)
		esperada, err := serializarDecisionPostgreSQL(decision)
		if err != nil || !documentosJSONPostgreSQLIguales(guardada, esperada) {
			t.Fatalf("decision durable distinta: error=%v", err)
		}
		var copiaLocal map[string]json.RawMessage
		if err = json.Unmarshal(guardada, &copiaLocal); err != nil {
			t.Fatalf("leer copia local: %v", err)
		}
		copiaLocal["codigo"] = json.RawMessage(`"mutado-localmente"`)
		guardadaOtra := obtenerDocumentoDecisionAdministrativamentePostgreSQL(t, ctx, poolAdmin, decision.DecisionRef)
		if !documentosJSONPostgreSQLIguales(guardadaOtra, esperada) {
			t.Fatal("una mutacion de la copia local altero la decision durable")
		}
		if err = almacenRegistro.RegistrarDecisionSiInstantaneaVigente(ctx, decision); !errors.Is(err, ports.ErrVersionAutorizacionYaExiste) {
			t.Fatalf("duplicado no rechazado exactamente: %v", err)
		}
		if err = intentarActualizarDecisionDirectamente(ctx, poolRegistro, decision.DecisionRef); err == nil {
			t.Fatal("la cuenta de registro pudo actualizar una decision")
		}
		verificarDecisionesMalformadasPostgreSQL(t, ctx, poolRegistro, decision)
	})

	t.Run("un perfil ajeno no se distingue de uno inexistente", func(t *testing.T) {
		_, err := almacenFuente.ObtenerInstantaneaAutorizacion(ctx, "persona:ajena", fixture.asignacion.PerfilActivoRef)
		if !errors.Is(err, ports.ErrAsignacionPerfilNoEncontrada) {
			t.Fatalf("resultado revelador: %v", err)
		}
	})

	t.Run("CAS rechaza cambio de asignacion", func(t *testing.T) {
		caso := nuevaFixtureAutorizacionPostgreSQL("asignacion", ahora)
		sembrarFixtureAutorizacionPostgreSQL(t, ctx, poolAdmin, caso)
		registro := &registroPostgreSQLConMutacion{
			destino: almacenRegistro,
			mutar: func(ctx context.Context) error {
				return avanzarAsignacionPostgreSQL(ctx, poolAdmin, caso, ahora.Add(time.Second))
			},
		}
		autorizador := nuevoAutorizadorPostgreSQLPrueba(
			t, almacenFuente, registro, ahora, "decision:postgres:obsoleta:asignacion",
		)
		_, err := autorizador.Exigir(ctx, solicitudPostgreSQLPrueba(t, caso))
		if !errors.Is(err, ports.ErrInstantaneaAutorizacionObsoleta) {
			t.Fatalf("CAS no detecto asignacion nueva: %v", err)
		}
	})

	t.Run("CAS rechaza retirada de la version exacta del rol", func(t *testing.T) {
		caso := nuevaFixtureAutorizacionPostgreSQL("control", ahora)
		sembrarFixtureAutorizacionPostgreSQL(t, ctx, poolAdmin, caso)
		registro := &registroPostgreSQLConMutacion{
			destino: almacenRegistro,
			mutar: func(ctx context.Context) error {
				return retirarVersionRolPostgreSQL(ctx, poolAdmin, caso, ahora.Add(time.Second))
			},
		}
		autorizador := nuevoAutorizadorPostgreSQLPrueba(
			t, almacenFuente, registro, ahora, "decision:postgres:obsoleta:control",
		)
		_, err := autorizador.Exigir(ctx, solicitudPostgreSQLPrueba(t, caso))
		if !errors.Is(err, ports.ErrInstantaneaAutorizacionObsoleta) {
			t.Fatalf("CAS no detecto retirada de rol: %v", err)
		}
	})

	t.Run("CAS rechaza revision huella y conjunto completo de politicas", func(t *testing.T) {
		caso := nuevaFixtureAutorizacionPostgreSQL("catalogo", ahora)
		sembrarFixtureAutorizacionPostgreSQL(t, ctx, poolAdmin, caso)
		registro := &registroPostgreSQLConMutacion{
			destino: almacenRegistro,
			mutar: func(ctx context.Context) error {
				return anadirPoliticaPostgreSQL(ctx, poolAdmin, caso.prefijo+"-posterior", ahora)
			},
		}
		autorizador := nuevoAutorizadorPostgreSQLPrueba(
			t, almacenFuente, registro, ahora, "decision:postgres:obsoleta:catalogo",
		)
		_, err := autorizador.Exigir(ctx, solicitudPostgreSQLPrueba(t, caso))
		if !errors.Is(err, ports.ErrInstantaneaAutorizacionObsoleta) {
			t.Fatalf("CAS no detecto catalogo nuevo: %v", err)
		}
	})

	t.Run("barrera durable rechaza cualquier comodin positivo", func(t *testing.T) {
		verificarComodinesRechazadosPostgreSQL(t, ctx, poolAdmin, fixture, ahora)
	})

	t.Run("barrera durable rechaza JSON ausente nulo o de tipo incorrecto", func(t *testing.T) {
		verificarJSONConfiguracionInvalidoPostgreSQL(t, ctx, poolAdmin, fixture)
	})
}

func nuevaFixtureAutorizacionPostgreSQL(prefijo string, ahora time.Time) fixtureAutorizacionPostgreSQL {
	publicada := ahora.Add(-2 * time.Hour)
	rol := domain.VersionRol{
		RolID:   "rol_" + prefijo,
		Version: 1,
		Nombre:  "Rol " + prefijo,
		Estado:  domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion:           "bolsa.merito.revisar",
			ModuloID:         "bolsa",
			TipoRecurso:      "merito",
			Finalidades:      []string{"gestion_bolsa"},
			GarantiaMinima:   domain.AuthAssuranceHigh,
			CamposPermitidos: []string{"descripcion", "estado"},
			Obligaciones:     []string{"trazar_acceso"},
		}},
		PublicadaPor: "seguridad:pruebas",
		PublicadaEn:  publicada,
	}
	control := domain.ControlVigenciaVersionRol{
		VersionRolRef: rol.Referencia(), Revision: 1,
		Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: "seguridad:pruebas", ActualizadoEn: publicada,
	}
	asignacion := domain.AsignacionPerfil{
		AsignacionID: "asignacion_" + prefijo, Version: 1,
		PerfilActivoRef: "prf_0123456789abcdefghijkl_" + prefijo,
		PrincipalID:     "per_0123456789abcdefghijkl_" + prefijo,
		VersionRolRef:   rol.Referencia(), Estado: domain.EstadoAsignacionPerfilActiva,
		Ambitos:      []domain.AmbitoPerfil{{Clave: "provincia", Valores: []string{"granada"}}},
		VigenteDesde: publicada.Add(time.Hour), VigenteHasta: ahora.Add(24 * time.Hour),
		EmitidaPor: "rrhh:pruebas", EmitidaEn: publicada,
	}
	politica := domain.PoliticaRestrictiva{
		PoliticaID: "politica_" + prefijo, Version: 1, Nombre: "Politica " + prefijo,
		Estado: domain.EstadoPoliticaRestrictivaPublicada, Efecto: domain.EfectoPoliticaRestringir,
		Acciones: []string{"bolsa.merito.revisar"}, Modulos: []string{"bolsa"},
		TiposRecurso: []string{"merito"}, FinalidadesPermitidas: []string{"gestion_bolsa"},
		GarantiaMinima: domain.AuthAssuranceHigh,
		Restricciones: []domain.RestriccionAtributoRecurso{{
			Clave: "estado", ValoresPermitidos: []string{"presentado"},
		}},
		RestringeCampos: true, CamposPermitidos: []string{"descripcion", "estado"},
		Obligaciones: []string{"registrar_revision"},
		VigenteDesde: publicada.Add(time.Hour), VigenteHasta: ahora.Add(24 * time.Hour),
		PublicadaPor: "seguridad:pruebas", PublicadaEn: publicada,
	}
	return fixtureAutorizacionPostgreSQL{
		prefijo: prefijo, rol: rol, controlRol: control,
		asignacion: asignacion, politica: politica, ahora: ahora,
	}
}

func solicitudPostgreSQLPrueba(t *testing.T, f fixtureAutorizacionPostgreSQL) domain.SolicitudAutorizacion {
	t.Helper()
	actor, vinculo, err := contextoYVinculoAutenticacionPostgreSQLPrueba(f)
	if err != nil {
		t.Fatalf("crear vinculo autenticacion-actor sintetico: %v", err)
	}
	return domain.SolicitudAutorizacion{
		Principal:                 actor.Principal,
		PerfilActivoRef:           actor.PerfilActivoRef,
		ContextoActor:             actor,
		VinculoAutenticacionActor: vinculo,
		Accion:                    "bolsa.merito.revisar",
		Recurso: domain.RecursoAutorizable{
			Referencia: "merito:" + f.prefijo, ModuloID: "bolsa", Tipo: "merito",
			Ambitos:   map[string]string{"provincia": "granada"},
			Atributos: map[string]string{"estado": "presentado"},
		},
		Finalidad: "gestion_bolsa", CorrelacionRef: "corr:" + f.prefijo,
		Motivo: "prueba de integracion",
	}
}

func contextoYVinculoAutenticacionPostgreSQLPrueba(
	f fixtureAutorizacionPostgreSQL,
) (domain.ContextoActor, domain.VinculoAutenticacionActorV1, error) {
	sufijo := "0123456789abcdefghijkl_" + f.prefijo
	cuenta := domain.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_" + sufijo,
		Metodo:    domain.AuthMethodCertificate,
		Garantia:  domain.AuthAssuranceHigh,
	}
	instantanea := domain.InstantaneaContextoActor{
		VinculoRef:      "vca_" + sufijo,
		VinculoVersion:  5,
		CuentaRef:       cuenta.CuentaRef,
		PersonaRef:      f.asignacion.PrincipalID,
		PersonaVersion:  3,
		PerfilActivoRef: f.asignacion.PerfilActivoRef,
		PerfilVersion:   4,
		Estado:          domain.EstadoVinculoContextoActorActivo,
		VigenteDesde:    f.ahora.Add(-time.Hour),
		VigenteHasta:    f.ahora.Add(30 * time.Minute),
	}
	actor, err := domain.NuevoContextoActor(cuenta, instantanea, f.ahora.Add(-2*time.Minute))
	if err != nil {
		return domain.ContextoActor{}, domain.VinculoAutenticacionActorV1{}, err
	}
	autenticacion := domain.AutenticacionRevalidadaV1{
		AutenticacionRef:             "aut_" + sufijo,
		AutenticacionHuellaSHA256:    strings.Repeat("1", 64),
		AsercionRef:                  "ase_" + sufijo,
		SesionRef:                    "ses_" + sufijo,
		ControlSesionRef:             "cse_" + sufijo,
		ControlSesionRevision:        7,
		ControlSesionHuellaSHA256:    strings.Repeat("2", 64),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   domain.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_" + sufijo,
		PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64),
		AutenticacionVerificadaEn:    f.ahora.Add(-5 * time.Minute),
		SesionEmitidaEn:              f.ahora.Add(-4 * time.Minute),
		SesionRevalidadaEn:           f.ahora.Add(-3 * time.Minute),
		SesionValidaHasta:            f.ahora.Add(10 * time.Minute),
	}
	vinculo, err := domain.CrearVinculoAutenticacionActorV1(
		context.Background(),
		revalidadorAutenticacionActorPostgreSQLIntegracion{resultado: autenticacion},
		domain.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		actor,
		f.ahora,
	)
	if err != nil {
		return domain.ContextoActor{}, domain.VinculoAutenticacionActorV1{}, err
	}
	return actor, vinculo, nil
}

func nuevoAutorizadorPostgreSQLPrueba(
	t *testing.T,
	puertoFuente ports.FuenteAutorizacion,
	puertoRegistro ports.RegistroDecisionesAutorizacion,
	ahora time.Time,
	decisionRef string,
) *application.ServicioAutorizacion {
	t.Helper()
	servicio, err := application.NuevoServicioAutorizacion(
		puertoFuente, puertoRegistro, registroDenegacionesPostgreSQLPrueba{}, relojPostgreSQLPrueba{ahora: ahora},
		generadorPostgreSQLPrueba(decisionRef),
		application.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		t.Fatalf("crear autorizador: %v", err)
	}
	return servicio
}

func sembrarFixtureAutorizacionPostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	f fixtureAutorizacionPostgreSQL,
) {
	t.Helper()
	if err := validarFixtureAutorizacionPostgreSQL(f); err != nil {
		t.Fatalf("fixture invalida: %v", err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatal("iniciar siembra")
	}
	defer revertirTransaccionPostgreSQL(tx)
	documentoRol := documentoJSONPostgreSQLPrueba(t, f.rol)
	huellaRol, _ := f.rol.HuellaSHA256()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.version_rol
		(version_rol_ref, rol_id, version, huella_sha256, publicada_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb)`,
		f.rol.Referencia(), f.rol.RolID, f.rol.Version, huellaRol, f.rol.PublicadaEn, documentoRol)
	fallarSQLPostgreSQLPrueba(t, err, "insertar rol")
	documentoControl := documentoJSONPostgreSQLPrueba(t, f.controlRol)
	huellaControl, _ := f.controlRol.HuellaSHA256()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol
		(version_rol_ref, revision, estado, huella_sha256, actualizado_en, documento)
		VALUES ($1,$2::numeric,$3,$4,$5,$6::jsonb)`,
		f.controlRol.VersionRolRef, strconv.FormatUint(f.controlRol.Revision, 10),
		f.controlRol.Estado, huellaControl, f.controlRol.ActualizadoEn, documentoControl)
	fallarSQLPostgreSQLPrueba(t, err, "insertar control de rol")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
		(version_rol_ref, revision, actualizada_en, actualizada_por, acto_ref)
		VALUES ($1,$2::numeric,$3,$4,$5)`, f.controlRol.VersionRolRef,
		strconv.FormatUint(f.controlRol.Revision, 10), f.controlRol.ActualizadoEn,
		"seguridad:pruebas", "acto:control:"+f.prefijo)
	fallarSQLPostgreSQLPrueba(t, err, "insertar puntero de control")

	documentoAsignacion := documentoJSONPostgreSQLPrueba(t, f.asignacion)
	huellaAsignacion, _ := f.asignacion.HuellaSHA256()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.asignacion_perfil
		(asignacion_ref, asignacion_id, version, perfil_activo_ref, principal_id,
		 version_rol_ref, huella_sha256, emitida_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`,
		f.asignacion.Referencia(), f.asignacion.AsignacionID, f.asignacion.Version,
		f.asignacion.PerfilActivoRef, f.asignacion.PrincipalID, f.asignacion.VersionRolRef,
		huellaAsignacion, f.asignacion.EmitidaEn, documentoAsignacion)
	fallarSQLPostgreSQLPrueba(t, err, "insertar asignacion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.asignacion_perfil_actual
		(perfil_activo_ref, asignacion_ref, actualizada_en, actualizada_por, acto_ref)
		VALUES ($1,$2,$3,$4,$5)`, f.asignacion.PerfilActivoRef, f.asignacion.Referencia(),
		f.asignacion.EmitidaEn, "rrhh:pruebas", "acto:asignacion:"+f.prefijo)
	fallarSQLPostgreSQLPrueba(t, err, "insertar puntero de asignacion")

	insertarIdentidadAutenticacionActorPostgreSQL(t, ctx, tx, f)
	insertarPoliticaPostgreSQL(t, ctx, tx, f.politica)
	actualizarControlCatalogoPostgreSQL(t, ctx, tx, "acto:politica:"+f.prefijo)
	fallarSQLPostgreSQLPrueba(t, tx.Commit(ctx), "confirmar fixture")
}

func insertarIdentidadAutenticacionActorPostgreSQL(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	f fixtureAutorizacionPostgreSQL,
) {
	t.Helper()
	actor, vinculo, err := contextoYVinculoAutenticacionPostgreSQLPrueba(f)
	if err != nil {
		t.Fatalf("crear identidad sintetica: %v", err)
	}
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatalf("proyectar vinculo sintetico: %v", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.sesion_autenticacion_v1
		(sesion_ref, autenticacion_ref, autenticacion_huella_sha256, asercion_ref,
		 cuenta_ref, cuenta_ordinaria_ref, cuenta_privilegiada, superficie,
		 metodo_observado, garantia_observada, politica_garantia_ref,
		 politica_garantia_huella_sha256, autenticacion_verificada_en, sesion_emitida_en)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		datos.SesionRef, datos.AutenticacionRef, datos.AutenticacionHuellaSHA256,
		datos.AsercionRef, datos.CuentaRef, datos.CuentaOrdinariaRef,
		datos.CuentaPrivilegiada, datos.Superficie, datos.MetodoObservado,
		datos.GarantiaObservada, datos.PoliticaGarantiaRef,
		datos.PoliticaGarantiaHuellaSHA256, datos.AutenticacionVerificadaEn,
		datos.SesionEmitidaEn)
	fallarSQLPostgreSQLPrueba(t, err, "insertar sesion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_sesion_v1
		(control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
		 sesion_revalidada_en, sesion_valida_hasta)
		VALUES ($1,$2::numeric,$3,'activa',$4,$5,$6)`, datos.ControlSesionRef,
		strconv.FormatUint(datos.ControlSesionRevision, 10), datos.SesionRef,
		datos.ControlSesionHuellaSHA256, datos.SesionRevalidadaEn, datos.SesionValidaHasta)
	fallarSQLPostgreSQLPrueba(t, err, "insertar control de sesion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_sesion_actual_v1
		(sesion_ref, control_sesion_ref, revision, actualizada_en, acto_ref)
		VALUES ($1,$2,$3::numeric,$4,$5)`, datos.SesionRef, datos.ControlSesionRef,
		strconv.FormatUint(datos.ControlSesionRevision, 10), datos.SesionRevalidadaEn,
		"acto:control-sesion:"+f.prefijo)
	fallarSQLPostgreSQLPrueba(t, err, "insertar puntero de sesion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.contexto_actor_v1
		(contexto_actor_ref, version, cuenta_ref, principal_id, perfil_activo_ref,
		 estado, huella_sha256, vigente_desde, vigente_hasta)
		VALUES ($1,$2::numeric,$3,$4,$5,'activo',$6,$7,$8)`, datos.ContextoActorRef,
		strconv.FormatUint(datos.ContextoActorVersion, 10), datos.CuentaRef,
		datos.PrincipalID, datos.PerfilActivoRef, datos.ContextoActorHuellaSHA256,
		actor.Instantanea.VigenteDesde, actor.Instantanea.VigenteHasta)
	fallarSQLPostgreSQLPrueba(t, err, "insertar contexto de actor")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.contexto_actor_actual_v1
		(cuenta_ref, perfil_activo_ref, contexto_actor_ref, version, actualizada_en, acto_ref)
		VALUES ($1,$2,$3,$4::numeric,$5,$6)`, datos.CuentaRef, datos.PerfilActivoRef,
		datos.ContextoActorRef, strconv.FormatUint(datos.ContextoActorVersion, 10),
		actor.ResueltoEn, "acto:contexto-actor:"+f.prefijo)
	fallarSQLPostgreSQLPrueba(t, err, "insertar puntero de actor")
}

func validarFixtureAutorizacionPostgreSQL(f fixtureAutorizacionPostgreSQL) error {
	if err := f.rol.Validar(); err != nil {
		return err
	}
	if err := f.controlRol.Validar(); err != nil {
		return err
	}
	if err := f.asignacion.Validar(); err != nil {
		return err
	}
	return f.politica.Validar()
}

func insertarPoliticaPostgreSQL(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	politica domain.PoliticaRestrictiva,
) {
	t.Helper()
	documento := documentoJSONPostgreSQLPrueba(t, politica)
	huella, err := politica.HuellaSHA256()
	if err != nil {
		t.Fatalf("huella politica: %v", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.politica_restrictiva
		(politica_ref, politica_id, version, huella_sha256, publicada_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, politica.Referencia(), politica.PoliticaID,
		politica.Version, huella, politica.PublicadaEn, documento)
	fallarSQLPostgreSQLPrueba(t, err, "insertar politica")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.politica_restrictiva_actual
		(politica_id, politica_ref, actualizada_en, actualizada_por, acto_ref)
		VALUES ($1,$2,$3,$4,$5)`, politica.PoliticaID, politica.Referencia(),
		politica.PublicadaEn, "seguridad:pruebas", "acto:"+politica.PoliticaID)
	fallarSQLPostgreSQLPrueba(t, err, "insertar puntero de politica")
}

func actualizarControlCatalogoPostgreSQL(
	t *testing.T,
	ctx context.Context,
	tx pgx.Tx,
	actoRef string,
) {
	t.Helper()
	filas, err := tx.Query(ctx, `
		SELECT politica.documento
		FROM vec_autorizacion.politica_restrictiva_actual AS actual
		JOIN vec_autorizacion.politica_restrictiva AS politica
		  ON politica.politica_id=actual.politica_id AND politica.politica_ref=actual.politica_ref
		ORDER BY politica.politica_ref`)
	fallarSQLPostgreSQLPrueba(t, err, "leer catalogo")
	politicas := make([]domain.PoliticaRestrictiva, 0)
	for filas.Next() {
		var documento []byte
		fallarSQLPostgreSQLPrueba(t, filas.Scan(&documento), "leer politica")
		var politica domain.PoliticaRestrictiva
		if err = json.Unmarshal(documento, &politica); err != nil {
			t.Fatalf("decodificar politica: %v", err)
		}
		politicas = append(politicas, politica)
	}
	fallarSQLPostgreSQLPrueba(t, filas.Err(), "recorrer catalogo")
	filas.Close()
	huella, err := domain.HuellaCatalogoPoliticasAutorizacion(politicas)
	if err != nil {
		t.Fatalf("huella catalogo: %v", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE vec_autorizacion.control_catalogo_politicas
		SET revision=revision+1, huella_sha256=$1, actualizado_en=clock_timestamp(),
		    actualizado_por=$2, acto_ref=$3
		WHERE control_id=true`, huella, "seguridad:pruebas", actoRef)
	fallarSQLPostgreSQLPrueba(t, err, "actualizar control de catalogo")
}

func avanzarAsignacionPostgreSQL(
	ctx context.Context,
	pool *pgxpool.Pool,
	f fixtureAutorizacionPostgreSQL,
	instante time.Time,
) error {
	asignacion := f.asignacion
	asignacion.Version = 2
	asignacion.EmitidaEn = instante
	asignacion.VigenteDesde = instante
	asignacion.VigenteHasta = instante.Add(24 * time.Hour)
	if err := asignacion.Validar(); err != nil {
		return err
	}
	documento, _ := json.Marshal(asignacion)
	huella, _ := asignacion.HuellaSHA256()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer revertirTransaccionPostgreSQL(tx)
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.asignacion_perfil
		(asignacion_ref, asignacion_id, version, perfil_activo_ref, principal_id,
		 version_rol_ref, huella_sha256, emitida_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`, asignacion.Referencia(),
		asignacion.AsignacionID, asignacion.Version, asignacion.PerfilActivoRef,
		asignacion.PrincipalID, asignacion.VersionRolRef, huella, asignacion.EmitidaEn, documento)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE vec_autorizacion.asignacion_perfil_actual
		SET asignacion_ref=$1, actualizada_en=$2, actualizada_por=$3, acto_ref=$4
		WHERE perfil_activo_ref=$5`, asignacion.Referencia(), instante,
		"rrhh:pruebas", "acto:asignacion:v2:"+f.prefijo, asignacion.PerfilActivoRef)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func retirarVersionRolPostgreSQL(
	ctx context.Context,
	pool *pgxpool.Pool,
	f fixtureAutorizacionPostgreSQL,
	instante time.Time,
) error {
	control := domain.ControlVigenciaVersionRol{
		VersionRolRef: f.rol.Referencia(), Revision: 2,
		Estado:         domain.EstadoControlVigenciaVersionRolRetirada,
		ActualizadoPor: "seguridad:pruebas", ActualizadoEn: instante,
		ActoRef: "acto:retirada:" + f.prefijo, MotivoCodigo: "revision_seguridad",
	}
	if err := control.Validar(); err != nil {
		return err
	}
	documento, _ := json.Marshal(control)
	huella, _ := control.HuellaSHA256()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer revertirTransaccionPostgreSQL(tx)
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol
		(version_rol_ref, revision, estado, huella_sha256, actualizado_en, documento)
		VALUES ($1,$2::numeric,$3,$4,$5,$6::jsonb)`, control.VersionRolRef,
		strconv.FormatUint(control.Revision, 10), control.Estado, huella, control.ActualizadoEn, documento)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE vec_autorizacion.control_vigencia_version_rol_actual
		SET revision=$1::numeric, actualizada_en=$2, actualizada_por=$3, acto_ref=$4
		WHERE version_rol_ref=$5`, strconv.FormatUint(control.Revision, 10), instante,
		control.ActualizadoPor, control.ActoRef, control.VersionRolRef)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func anadirPoliticaPostgreSQL(
	ctx context.Context,
	pool *pgxpool.Pool,
	prefijo string,
	ahora time.Time,
) error {
	fixture := nuevaFixtureAutorizacionPostgreSQL(prefijo, ahora)
	politica := fixture.politica
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer revertirTransaccionPostgreSQL(tx)
	documento, _ := json.Marshal(politica)
	huella, _ := politica.HuellaSHA256()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.politica_restrictiva
		(politica_ref, politica_id, version, huella_sha256, publicada_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, politica.Referencia(), politica.PoliticaID,
		politica.Version, huella, politica.PublicadaEn, documento)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.politica_restrictiva_actual
		(politica_id, politica_ref, actualizada_en, actualizada_por, acto_ref)
		VALUES ($1,$2,$3,$4,$5)`, politica.PoliticaID, politica.Referencia(),
		politica.PublicadaEn, "seguridad:pruebas", "acto:"+politica.PoliticaID)
	if err != nil {
		return err
	}
	filas, err := tx.Query(ctx, `
		SELECT politica.documento FROM vec_autorizacion.politica_restrictiva_actual actual
		JOIN vec_autorizacion.politica_restrictiva politica
		ON politica.politica_id=actual.politica_id AND politica.politica_ref=actual.politica_ref
		ORDER BY politica.politica_ref`)
	if err != nil {
		return err
	}
	politicas := make([]domain.PoliticaRestrictiva, 0)
	for filas.Next() {
		var contenido []byte
		if err = filas.Scan(&contenido); err != nil {
			filas.Close()
			return err
		}
		var actual domain.PoliticaRestrictiva
		if err = json.Unmarshal(contenido, &actual); err != nil {
			filas.Close()
			return err
		}
		politicas = append(politicas, actual)
	}
	if err = filas.Err(); err != nil {
		filas.Close()
		return err
	}
	filas.Close()
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(politicas)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE vec_autorizacion.control_catalogo_politicas
		SET revision=revision+1, huella_sha256=$1, actualizado_en=clock_timestamp(),
		actualizado_por=$2, acto_ref=$3 WHERE control_id=true`,
		huellaCatalogo, "seguridad:pruebas", "acto:catalogo:"+prefijo)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func verificarPrivilegiosFuentePostgreSQL(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	verificarAtributosRuntimePostgreSQL(t, ctx, pool, "fuente")
	verificarSinPrivilegiosTablasPostgreSQL(t, ctx, pool, "fuente")

	var ejecutarFuente, ejecutarRegistro, existeFuncionLectura bool
	err := pool.QueryRow(ctx, `
		SELECT has_function_privilege(current_user, 'vec_autorizacion.obtener_instantanea(text,text)', 'EXECUTE'),
		       has_function_privilege(current_user, 'vec_autorizacion.registrar_decision_si_vigente(jsonb)', 'EXECUTE'),
		       pg_catalog.to_regprocedure('vec_autorizacion.obtener_decision(text)') IS NOT NULL`).Scan(
		&ejecutarFuente, &ejecutarRegistro, &existeFuncionLectura,
	)
	if err != nil || !ejecutarFuente || ejecutarRegistro || existeFuncionLectura {
		t.Fatalf("capacidades de fuente incorrectas: err=%v fuente=%t registro=%t lectura=%t",
			err, ejecutarFuente, ejecutarRegistro, existeFuncionLectura)
	}
	_, err = pool.Exec(ctx, `SELECT vec_autorizacion.registrar_decision_si_vigente('{}'::jsonb)`)
	exigirDenegacionPrivilegioPostgreSQL(t, err, "la identidad fuente ejecuto el registro")
}

func verificarPrivilegiosRegistroPostgreSQL(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	verificarAtributosRuntimePostgreSQL(t, ctx, pool, "registro")
	verificarSinPrivilegiosTablasPostgreSQL(t, ctx, pool, "registro")

	var ejecutarFuente, ejecutarRegistro, existeFuncionLectura bool
	err := pool.QueryRow(ctx, `
		SELECT has_function_privilege(current_user, 'vec_autorizacion.obtener_instantanea(text,text)', 'EXECUTE'),
		       has_function_privilege(current_user, 'vec_autorizacion.registrar_decision_si_vigente(jsonb)', 'EXECUTE'),
		       pg_catalog.to_regprocedure('vec_autorizacion.obtener_decision(text)') IS NOT NULL`).Scan(
		&ejecutarFuente, &ejecutarRegistro, &existeFuncionLectura,
	)
	if err != nil || ejecutarFuente || !ejecutarRegistro || existeFuncionLectura {
		t.Fatalf("capacidades de registro incorrectas: err=%v fuente=%t registro=%t lectura=%t",
			err, ejecutarFuente, ejecutarRegistro, existeFuncionLectura)
	}
	_, err = pool.Exec(ctx, `SELECT * FROM vec_autorizacion.obtener_instantanea('persona:no-existe', 'perfil:no-existe')`)
	exigirDenegacionPrivilegioPostgreSQL(t, err, "la identidad de registro leyo mediante una funcion")
}

func verificarAtributosRuntimePostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	identidad string,
) {
	t.Helper()
	var superusuario, crearRol, crearBD, replicacion, omitirRLS bool
	err := pool.QueryRow(ctx, `
		SELECT rolsuper, rolcreaterole, rolcreatedb, rolreplication, rolbypassrls
		FROM pg_catalog.pg_roles WHERE rolname=current_user`).Scan(
		&superusuario, &crearRol, &crearBD, &replicacion, &omitirRLS,
	)
	if err != nil || superusuario || crearRol || crearBD || replicacion || omitirRLS {
		t.Fatalf("atributos %s inseguros: %v %t %t %t %t %t",
			identidad, err, superusuario, crearRol, crearBD, replicacion, omitirRLS)
	}
}

func verificarSinPrivilegiosTablasPostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	identidad string,
) {
	t.Helper()
	var tablasConPrivilegio int
	err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM pg_catalog.pg_class AS clase
		JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid=clase.relnamespace
		WHERE espacio.nspname='vec_autorizacion' AND clase.relkind IN ('r', 'p')
		  AND (
		      has_table_privilege(current_user, clase.oid, 'SELECT')
		      OR has_table_privilege(current_user, clase.oid, 'INSERT')
		      OR has_table_privilege(current_user, clase.oid, 'UPDATE')
		      OR has_table_privilege(current_user, clase.oid, 'DELETE')
		      OR has_table_privilege(current_user, clase.oid, 'TRUNCATE')
		      OR has_table_privilege(current_user, clase.oid, 'REFERENCES')
		      OR has_table_privilege(current_user, clase.oid, 'TRIGGER')
		  )`).Scan(&tablasConPrivilegio)
	if err != nil || tablasConPrivilegio != 0 {
		t.Fatalf("la identidad %s tiene privilegios directos sobre %d tablas: %v",
			identidad, tablasConPrivilegio, err)
	}
	var total int
	err = pool.QueryRow(ctx, `SELECT count(*) FROM vec_autorizacion.asignacion_perfil`).Scan(&total)
	if err == nil {
		t.Fatalf("la identidad %s leyo directamente %d filas", identidad, total)
	}
	exigirDenegacionPrivilegioPostgreSQL(t, err, "lectura directa de "+identidad)
}

func exigirDenegacionPrivilegioPostgreSQL(t *testing.T, err error, operacion string) {
	t.Helper()
	var errorPG *pgconn.PgError
	if err == nil || !errors.As(err, &errorPG) || errorPG.Code != "42501" {
		t.Fatalf("%s no fue denegada por privilegios: %v", operacion, err)
	}
}

func obtenerDocumentoDecisionAdministrativamentePostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	referencia string,
) []byte {
	t.Helper()
	var documento []byte
	err := pool.QueryRow(ctx, `
		SELECT documento FROM vec_autorizacion.decision_autorizacion WHERE decision_ref=$1`,
		referencia,
	).Scan(&documento)
	if err != nil {
		t.Fatalf("leer decision con identidad administrativa: %v", err)
	}
	var objeto map[string]json.RawMessage
	if decodificarDocumentoPostgreSQL(documento, &objeto) != nil || len(objeto) != 31 {
		t.Fatal("documento administrativo de decision no integro")
	}
	var referenciaGuardada string
	if json.Unmarshal(objeto["decision_ref"], &referenciaGuardada) != nil ||
		referenciaGuardada != referencia {
		t.Fatal("referencia durable de decision no coincide")
	}
	var vinculo domain.DatosVinculoAutenticacionActorV1
	if decodificarDocumentoPostgreSQL(objeto["vinculo_autenticacion_actor"], &vinculo) != nil ||
		vinculo.Validar() != nil {
		t.Fatal("vinculo autenticacion-actor durable no integro")
	}
	return append([]byte(nil), documento...)
}

func documentosJSONPostgreSQLIguales(primero, segundo []byte) bool {
	var a, b any
	return json.Unmarshal(primero, &a) == nil && json.Unmarshal(segundo, &b) == nil &&
		reflect.DeepEqual(a, b)
}

func intentarActualizarDecisionDirectamente(ctx context.Context, pool *pgxpool.Pool, referencia string) error {
	_, err := pool.Exec(ctx, `UPDATE vec_autorizacion.decision_autorizacion SET codigo='alterada' WHERE decision_ref=$1`, referencia)
	return err
}

func verificarComodinesRechazadosPostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	f fixtureAutorizacionPostgreSQL,
	ahora time.Time,
) {
	t.Helper()
	casosRol := []struct {
		nombre string
		mutar  func(*domain.ConcesionRol)
	}{
		{"accion parcial", func(c *domain.ConcesionRol) { c.Accion = "bolsa.*" }},
		{"modulo parcial", func(c *domain.ConcesionRol) { c.ModuloID = "bol*sa" }},
		{"tipo parcial", func(c *domain.ConcesionRol) { c.TipoRecurso = "merito:*" }},
		{"finalidad parcial", func(c *domain.ConcesionRol) { c.Finalidades[0] = "gestion_*" }},
		{"campo parcial", func(c *domain.ConcesionRol) { c.CamposPermitidos[0] = "estado:*" }},
		{"obligacion parcial", func(c *domain.ConcesionRol) { c.Obligaciones[0] = "trazar_*" }},
	}
	for indice, caso := range casosRol {
		var rol domain.VersionRol
		clonarJSONPostgreSQLPrueba(t, f.rol, &rol)
		rol.RolID = fmt.Sprintf("rol_comodin_sql_%d", indice)
		caso.mutar(&rol.Concesiones[0])
		documentoRol := documentoJSONPostgreSQLPrueba(t, rol)
		_, err := pool.Exec(ctx, `
			INSERT INTO vec_autorizacion.version_rol
			(version_rol_ref, rol_id, version, huella_sha256, publicada_en, documento)
			VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, rol.Referencia(), rol.RolID, rol.Version,
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", rol.PublicadaEn, documentoRol)
		exigirViolacionConstraintPostgreSQL(t, err, caso.nombre)
	}

	casosAmbito := []domain.AmbitoPerfil{
		{Clave: "global", Valores: []string{"granada"}},
		{Clave: "provin*cia", Valores: []string{"granada"}},
		{Clave: "provincia", Valores: []string{"gran*ada"}},
	}
	for indice, ambito := range casosAmbito {
		var asignacion domain.AsignacionPerfil
		clonarJSONPostgreSQLPrueba(t, f.asignacion, &asignacion)
		asignacion.AsignacionID = fmt.Sprintf("asignacion_comodin_sql_%d", indice)
		asignacion.PerfilActivoRef = fmt.Sprintf("perfil:comodin:sql:%d", indice)
		asignacion.PrincipalID = fmt.Sprintf("persona:comodin:sql:%d", indice)
		asignacion.EmitidaEn = ahora.Add(-time.Hour)
		asignacion.VigenteDesde = ahora.Add(-time.Hour)
		asignacion.VigenteHasta = ahora.Add(time.Hour)
		asignacion.Ambitos = []domain.AmbitoPerfil{ambito}
		documentoAsignacion := documentoJSONPostgreSQLPrueba(t, asignacion)
		_, err := pool.Exec(ctx, `
			INSERT INTO vec_autorizacion.asignacion_perfil
			(asignacion_ref, asignacion_id, version, perfil_activo_ref, principal_id,
			 version_rol_ref, huella_sha256, emitida_en, documento)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`, asignacion.Referencia(),
			asignacion.AsignacionID, asignacion.Version, asignacion.PerfilActivoRef,
			asignacion.PrincipalID, asignacion.VersionRolRef,
			"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			asignacion.EmitidaEn, documentoAsignacion)
		exigirViolacionConstraintPostgreSQL(t, err, "ambito positivo no exacto")
	}
}

func verificarJSONConfiguracionInvalidoPostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	f fixtureAutorizacionPostgreSQL,
) {
	t.Helper()
	casos := []struct {
		nombre string
		valor  any
	}{
		{"ausente", nil},
		{"nulo", nil},
		{"tipo incorrecto", map[string]any{"objeto": true}},
	}
	for indice, caso := range casos {
		var documento map[string]any
		clonarJSONPostgreSQLPrueba(t, f.rol, &documento)
		rolID := fmt.Sprintf("rol_json_invalido_%d", indice)
		documento["rol_id"] = rolID
		switch caso.nombre {
		case "ausente":
			delete(documento, "version")
		case "nulo":
			documento["version"] = caso.valor
		case "tipo incorrecto":
			documento["version"] = caso.valor
		}
		contenido := documentoJSONPostgreSQLPrueba(t, documento)
		_, err := pool.Exec(ctx, `
			INSERT INTO vec_autorizacion.version_rol
			(version_rol_ref, rol_id, version, huella_sha256, publicada_en, documento)
			VALUES ($1,$2,1,$3,$4,$5::jsonb)`, "rol:"+rolID+":v1", rolID,
			"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			f.rol.PublicadaEn, contenido)
		if err == nil {
			t.Fatalf("JSON %s aceptado", caso.nombre)
		}
	}
	for indice, instante := range []string{
		"2026-07-15T10:00:00.1234567Z",
		"2026-07-15T12:00:00+02:00",
	} {
		var documento map[string]any
		clonarJSONPostgreSQLPrueba(t, f.rol, &documento)
		rolID := fmt.Sprintf("rol_instante_invalido_%d", indice)
		documento["rol_id"] = rolID
		documento["publicada_en"] = instante
		contenido := documentoJSONPostgreSQLPrueba(t, documento)
		_, err := pool.Exec(ctx, `
			INSERT INTO vec_autorizacion.version_rol
			(version_rol_ref, rol_id, version, huella_sha256, publicada_en, documento)
			VALUES ($1,$2,1,$3,$4,$5::jsonb)`, "rol:"+rolID+":v1", rolID,
			"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
			f.rol.PublicadaEn, contenido)
		if err == nil {
			t.Fatalf("instante no canonico aceptado: %s", instante)
		}
	}

	var asignacion map[string]any
	clonarJSONPostgreSQLPrueba(t, f.asignacion, &asignacion)
	delete(asignacion, "principal_id")
	asignacion["asignacion_id"] = "asignacion_json_sin_principal"
	asignacion["perfil_activo_ref"] = "perfil:json:sin-principal"
	contenido := documentoJSONPostgreSQLPrueba(t, asignacion)
	_, err := pool.Exec(ctx, `
		INSERT INTO vec_autorizacion.asignacion_perfil
		(asignacion_ref, asignacion_id, version, perfil_activo_ref, principal_id,
		 version_rol_ref, huella_sha256, emitida_en, documento)
		VALUES ($1,$2,1,$3,$4,$5,$6,$7,$8::jsonb)`,
		"asignacion:asignacion_json_sin_principal:v1", "asignacion_json_sin_principal",
		"perfil:json:sin-principal", "persona:json:sin-principal", f.rol.Referencia(),
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		f.asignacion.EmitidaEn, contenido)
	if err == nil {
		t.Fatal("asignacion sin principal en documento aceptada")
	}
}

func verificarDecisionesMalformadasPostgreSQL(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	decision domain.DecisionAutorizacion,
) {
	t.Helper()
	base, err := serializarDecisionPostgreSQL(decision)
	if err != nil {
		t.Fatalf("serializar decision base: %v", err)
	}
	casos := []struct {
		nombre       string
		excedeLimite bool
		mutar        func(map[string]any)
	}{
		{"clave obligatoria ausente", false, func(documento map[string]any) { delete(documento, "accion") }},
		{"clave adicional", false, func(documento map[string]any) { documento["inesperada"] = true }},
		{"booleano falso", false, func(documento map[string]any) { documento["concedida"] = false }},
		{"coercion booleana", false, func(documento map[string]any) { documento["concedida"] = "yes" }},
		{"nulo", false, func(documento map[string]any) { documento["modulo_id"] = nil }},
		{"texto como objeto", false, func(documento map[string]any) {
			documento["tipo_recurso"] = map[string]any{"tipo": "merito"}
		}},
		{"array como objeto", false, func(documento map[string]any) {
			documento["politicas_refs"] = map[string]any{"politica": true}
		}},
		{"valor de manifiesto no textual", false, func(documento map[string]any) {
			manifesto := documento["politicas_evaluadas_huellas_sha256"].(map[string]any)
			for referencia := range manifesto {
				manifesto[referencia] = json.RawMessage(strings.Repeat("1", 64))
				break
			}
		}},
		{"comodin positivo", false, func(documento map[string]any) { documento["finalidad"] = "gestion_*" }},
		{"exceso de 512 KiB", true, func(documento map[string]any) {
			documento["campos_permitidos"] = listaGrandeDecisionPostgreSQL("campo")
			documento["obligaciones"] = listaGrandeDecisionPostgreSQL("obligacion")
		}},
	}
	for indice, caso := range casos {
		var documento map[string]any
		if err = json.Unmarshal(base, &documento); err != nil {
			t.Fatalf("decodificar decision base: %v", err)
		}
		documento["decision_ref"] = fmt.Sprintf("decision:malformada:%d", indice)
		caso.mutar(documento)
		contenido := documentoJSONPostgreSQLPrueba(t, documento)
		if caso.excedeLimite {
			var tamano int
			if err = pool.QueryRow(ctx, `SELECT pg_column_size($1::jsonb)`, contenido).Scan(&tamano); err != nil {
				t.Fatalf("medir decision sobredimensionada: %v", err)
			}
			if tamano <= 512*1024 {
				t.Fatalf("fixture de exceso insuficiente: %d bytes", tamano)
			}
		}
		var registrada bool
		err = pool.QueryRow(ctx, `SELECT vec_autorizacion.registrar_decision_si_vigente($1::jsonb)`, contenido).Scan(&registrada)
		if err != nil || registrada {
			t.Fatalf("decision %q no fue denegada limpiamente: registrada=%t err=%v",
				caso.nombre, registrada, err)
		}
	}
}

func listaGrandeDecisionPostgreSQL(prefijo string) []string {
	valores := make([]string, 512)
	for indice := range valores {
		inicio := fmt.Sprintf("%s_%03d_", prefijo, indice)
		valores[indice] = inicio + strings.Repeat("x", 512-len(inicio))
	}
	return valores
}

func clonarJSONPostgreSQLPrueba(t *testing.T, origen, destino any) {
	t.Helper()
	contenido := documentoJSONPostgreSQLPrueba(t, origen)
	if err := json.Unmarshal(contenido, destino); err != nil {
		t.Fatalf("clonar JSON: %v", err)
	}
}

func exigirViolacionConstraintPostgreSQL(t *testing.T, err error, operacion string) {
	t.Helper()
	var errorPG *pgconn.PgError
	if err == nil || !errors.As(err, &errorPG) || errorPG.Code != "23514" {
		t.Fatalf("%s no fue rechazada por constraint: %v", operacion, err)
	}
}

func documentoJSONPostgreSQLPrueba(t *testing.T, valor any) []byte {
	t.Helper()
	documento, err := json.Marshal(valor)
	if err != nil {
		t.Fatalf("codificar documento: %v", err)
	}
	return documento
}

func fallarSQLPostgreSQLPrueba(t *testing.T, err error, operacion string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", operacion, err)
	}
}

func Example_integracionAutorizacionPostgreSQL() {
	fmt.Println("deploy/postgresql/autorizacion/probar_integracion.sh")
	// Output: deploy/postgresql/autorizacion/probar_integracion.sh
}
