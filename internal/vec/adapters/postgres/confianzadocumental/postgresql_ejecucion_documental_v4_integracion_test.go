package confianzadocumental

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

const (
	variableDSNV4AdminPrueba     = "VEC_POSTGRES_TEST_ADMIN_DSN"
	variableDSNV4EmisorPrueba    = "VEC_POSTGRES_TEST_V4_EMISOR_DSN"
	variableDSNV4EjecucionPrueba = "VEC_POSTGRES_TEST_V4_EJECUCION_DSN"
)

type casoPostgreSQLDocumentalV4 struct {
	decision domain.DecisionAutorizacion
	vinculo  ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4
	cabecera domain.CabeceraAtestacionAutorizacionV1
	payload  []byte
	sobre    ports.SobreCriptograficoDocumentalCrudoV4
}

type fixturePostgreSQLDocumentalV4 struct {
	ancla         time.Time
	datosVinculo  domain.DatosVinculoAutenticacionActorV1
	rol           domain.VersionRol
	controlRol    domain.ControlVigenciaVersionRol
	asignacion    domain.AsignacionPerfil
	materialFirma materialFirmaCOSEPrueba
	raiz          RaizPublicaFijada
	configuracion ConfiguracionConfianzaFijada
	secretoHMAC   []byte
	claveHMACID   string
	emisorID      string
	casos         []casoPostgreSQLDocumentalV4
}

// TestIntegracionEjecucionDocumentalV4PostgreSQLReal es deliberadamente una
// prueba opt-in: el runner crea PostgreSQL real, aplica las migraciones y
// entrega tres identidades distintas. Ningun doble de repositorio participa.
func TestIntegracionEjecucionDocumentalV4PostgreSQLReal(t *testing.T) {
	dsnAdmin := os.Getenv(variableDSNV4AdminPrueba)
	dsnEmisor := os.Getenv(variableDSNV4EmisorPrueba)
	dsnEjecucion := os.Getenv(variableDSNV4EjecucionPrueba)
	if dsnAdmin == "" || dsnEmisor == "" || dsnEjecucion == "" {
		t.Skipf(
			"prueba PostgreSQL omitida: defina %s, %s y %s o ejecute deploy/postgresql/ejecucion_documental_v4/probar_integracion.sh",
			variableDSNV4AdminPrueba, variableDSNV4EmisorPrueba,
			variableDSNV4EjecucionPrueba,
		)
	}

	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	poolAdmin := abrirPoolPostgreSQLDocumentalV4Prueba(t, ctx, dsnAdmin, "administrador")
	defer poolAdmin.Close()
	poolEmisor := abrirPoolPostgreSQLDocumentalV4Prueba(t, ctx, dsnEmisor, "emisor")
	defer poolEmisor.Close()
	poolEjecucion := abrirPoolPostgreSQLDocumentalV4Prueba(t, ctx, dsnEjecucion, "ejecutor")
	defer poolEjecucion.Close()

	fixture := nuevaFixturePostgreSQLDocumentalV4(t)
	sembrarFixturePostgreSQLDocumentalV4(t, ctx, poolAdmin, fixture)
	verificarSeparacionCredencialesPostgreSQLDocumentalV4(
		t, ctx, poolEmisor, poolEjecucion,
	)

	rutaSocket := t.TempDir() + "/emisor-v4.sock"
	manejador, err := NuevoManejadorHTTPEmisorCapacidadDocumentalV4(ctx, poolEmisor)
	if err != nil {
		t.Fatalf("crear emisor aislado: %v", err)
	}
	cerrarEmisor := servirEmisorUnixPostgreSQLDocumentalV4Prueba(t, rutaSocket, manejador)
	defer cerrarEmisor()

	ejecutor, err := NuevoEjecutorDocumentalPostgreSQLV4(ctx, poolEjecucion, rutaSocket)
	if err != nil {
		t.Fatalf("crear ejecutor segregado: %v", err)
	}

	// Un HMAC alterado llega a PostgreSQL, pero no deja atestacion, nonce,
	// consumo, efecto, auditoria ni outbox. La identidad ejecutora no conoce el
	// secreto y la validacion autoritativa ocurre dentro de la funcion SQL.
	artefactosInvalidos := solicitarArtefactosPostgreSQLDocumentalV4Prueba(
		t, ctx, ejecutor, fixture.casos[0],
	)
	verificarPrecondicionesArtefactosPostgreSQLDocumentalV4Prueba(
		t, ctx, poolAdmin, artefactosInvalidos,
	)
	verificarEjecucionReversiblePostgreSQLDocumentalV4Prueba(
		t, ctx, poolEjecucion, artefactosInvalidos,
	)
	var capacidadAlterada capacidadEjecucionDocumentalV4JSON
	if err = json.Unmarshal(artefactosInvalidos.capacidad, &capacidadAlterada); err != nil {
		t.Fatalf("leer capacidad de prueba: %v", err)
	}
	if capacidadAlterada.MACSHA256[0] == '0' {
		capacidadAlterada.MACSHA256 = "1" + capacidadAlterada.MACSHA256[1:]
	} else {
		capacidadAlterada.MACSHA256 = "0" + capacidadAlterada.MACSHA256[1:]
	}
	artefactosInvalidos.capacidad, err = json.Marshal(capacidadAlterada)
	if err != nil {
		t.Fatalf("alterar capacidad: %v", err)
	}
	if _, err = ejecutor.repositorio.ejecutarArtefactosAtestados(ctx, artefactosInvalidos); err == nil {
		t.Fatal("PostgreSQL acepto una capacidad con HMAC alterado")
	}
	verificarFilasDecisionPostgreSQLDocumentalV4(
		t, ctx, poolAdmin, fixture.casos[0].decision.DecisionRef, 0,
	)

	// Dos solicitudes productivas compiten por la misma DecisionRef. Solo una
	// puede crear el efecto completo; la otra falla cerrada sin filas parciales.
	type resultadoConcurrente struct {
		resultado ports.ResultadoConectorEjecucionDocumentalAtestadaV4
		err       error
	}
	resultados := make(chan resultadoConcurrente, 2)
	var grupo sync.WaitGroup
	for indice := 0; indice < 2; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			resultado, errEjecucion := ejecutor.EjecutarDocumentalAtestadoV4(
				ctx, fixture.casos[0].vinculo, fixture.casos[0].cabecera,
				fixture.casos[0].sobre,
			)
			resultados <- resultadoConcurrente{resultado: resultado, err: errEjecucion}
		}()
	}
	grupo.Wait()
	close(resultados)
	var exitos []ports.ResultadoConectorEjecucionDocumentalAtestadaV4
	var fallos int
	erroresConcurrentes := make([]string, 0, 2)
	for resultado := range resultados {
		if resultado.err != nil {
			fallos++
			erroresConcurrentes = append(erroresConcurrentes, resultado.err.Error())
			continue
		}
		exitos = append(exitos, resultado.resultado)
	}
	if len(exitos) != 1 || fallos != 1 {
		t.Fatalf("consumo concurrente no fue exactamente-uno: exitos=%d fallos=%d errores=%q",
			len(exitos), fallos, erroresConcurrentes)
	}
	verificarEfectoCompletoPostgreSQLDocumentalV4(
		t, ctx, poolAdmin, fixture.casos[0],
		resultadoInternoDesdePuertoPostgreSQLDocumentalV4Prueba(t, exitos[0]),
	)

	// Incluso con un nonce nuevo y una capacidad nueva, la decision consumida
	// no se puede reutilizar.
	if _, err = ejecutor.EjecutarDocumentalAtestadoV4(
		ctx, fixture.casos[0].vinculo, fixture.casos[0].cabecera,
		fixture.casos[0].sobre,
	); err == nil {
		t.Fatal("se reutilizo una DecisionRef ya consumida")
	}
	verificarFilasDecisionPostgreSQLDocumentalV4(
		t, ctx, poolAdmin, fixture.casos[0].decision.DecisionRef, 1,
	)

	// La carrera entre uso y revocacion de la raiz es linealizable. Se obtiene
	// primero una capacidad valida y se hacen competir directamente la funcion
	// ejecutora y el avance monotono del puntero de confianza. El resultado
	// admisible es efecto completo antes de revocar o denegacion sin una sola
	// fila; nunca un efecto parcial posterior a la revocacion.
	artefactosCarrera := solicitarArtefactosPostgreSQLDocumentalV4Prueba(
		t, ctx, ejecutor, fixture.casos[1],
	)
	inicio := make(chan struct{})
	resultadoCarrera := make(chan resultadoConcurrente, 1)
	revocacion := make(chan error, 1)
	go func() {
		<-inicio
		resultado, errEjecucion := ejecutor.repositorio.ejecutarArtefactosAtestados(
			ctx, artefactosCarrera,
		)
		resultadoPuerto := ports.ResultadoConectorEjecucionDocumentalAtestadaV4{}
		if errEjecucion == nil {
			resultadoPuerto, errEjecucion = ports.NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
				resultado.OrdenRef, resultado.Estado, resultado.AuditoriaRef,
				resultado.EventoOutboxRef, resultado.RegistradaEn,
			)
		}
		resultadoCarrera <- resultadoConcurrente{resultado: resultadoPuerto, err: errEjecucion}
	}()
	go func() {
		<-inicio
		revocacion <- revocarRaizPostgreSQLDocumentalV4(ctx, poolAdmin, fixture)
	}()
	close(inicio)
	resultadoLineal := <-resultadoCarrera
	if err = <-revocacion; err != nil {
		t.Fatalf("revocar raiz concurrentemente: %v", err)
	}
	esperadasCarrera := 0
	if resultadoLineal.err == nil {
		esperadasCarrera = 1
		verificarEfectoCompletoPostgreSQLDocumentalV4(
			t, ctx, poolAdmin, fixture.casos[1],
			resultadoInternoDesdePuertoPostgreSQLDocumentalV4Prueba(t, resultadoLineal.resultado),
		)
	}
	verificarFilasDecisionPostgreSQLDocumentalV4(
		t, ctx, poolAdmin, fixture.casos[1].decision.DecisionRef, esperadasCarrera,
	)
	verificarRaizRevocadaPostgreSQLDocumentalV4(t, ctx, poolAdmin, fixture)

	// El proceso emisor ya estaba vivo y conserva su copia publica, pero una
	// nueva DecisionRef no produce efecto tras revocar la raiz durable: la
	// barrera SQL vuelve a resolver y bloquear el estado actual.
	if _, err = ejecutor.EjecutarDocumentalAtestadoV4(
		ctx, fixture.casos[2].vinculo, fixture.casos[2].cabecera,
		fixture.casos[2].sobre,
	); err == nil {
		t.Fatal("se ejecuto una decision despues de revocar la raiz")
	}
	verificarFilasDecisionPostgreSQLDocumentalV4(
		t, ctx, poolAdmin, fixture.casos[2].decision.DecisionRef, 0,
	)
}

func resultadoInternoDesdePuertoPostgreSQLDocumentalV4Prueba(
	t *testing.T,
	resultado ports.ResultadoConectorEjecucionDocumentalAtestadaV4,
) ResultadoEjecucionPlanDocumentalV4 {
	t.Helper()
	orden, errOrden := resultado.OrdenRef()
	estado, errEstado := resultado.Estado()
	auditoria, errAuditoria := resultado.AuditoriaRef()
	evento, errEvento := resultado.EventoOutboxRef()
	registradaEn, errRegistro := resultado.RegistradaEn()
	if errOrden != nil || errEstado != nil || errAuditoria != nil ||
		errEvento != nil || errRegistro != nil {
		t.Fatalf("resultado del puerto PostgreSQL V4 invalido")
	}
	return ResultadoEjecucionPlanDocumentalV4{
		OrdenRef: orden, Estado: estado, AuditoriaRef: auditoria,
		EventoOutboxRef: evento, RegistradaEn: registradaEn,
	}
}

func nuevaFixturePostgreSQLDocumentalV4(t *testing.T) fixturePostgreSQLDocumentalV4 {
	t.Helper()
	// El segundo exacto ejercita que la representacion canonica conserve los
	// seis ceros requeridos por PostgreSQL sin depender de RFC3339Nano ni de una
	// fraccion elegida artificialmente por la prueba.
	ancla := time.Now().UTC().Add(-time.Second).Truncate(time.Second)
	actor, vinculoAutenticacion, err := pruebasvec.NuevoContextoYVinculo(
		ancla,
		"per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate,
		domain.AuthAssuranceHigh,
	)
	if err != nil || actor.Validar() != nil {
		t.Fatalf("crear vinculo de identidad realista: %v", err)
	}
	datosVinculo, err := vinculoAutenticacion.Datos()
	if err != nil {
		t.Fatalf("leer vinculo de identidad: %v", err)
	}

	publicada := ancla.Add(-10 * time.Minute)
	rol := domain.VersionRol{
		RolID:   "ejecutor_documental_v4_integracion",
		Version: 1,
		Nombre:  "Ejecucion documental V4 de integracion",
		Estado:  domain.EstadoVersionRolPublicada,
		Concesiones: []domain.ConcesionRol{{
			Accion: ports.AccionEjecutarPlanDocumentalV4, ModuloID: "bolsa",
			TipoRecurso: "documento_bolsa", Finalidades: []string{"tramitar_bolsa"},
			GarantiaMinima:   domain.AuthAssuranceHigh,
			CamposPermitidos: []string{"documento.generado"},
		}},
		PublicadaPor: "seguridad:integracion", PublicadaEn: publicada,
	}
	control := domain.ControlVigenciaVersionRol{
		VersionRolRef: rol.Referencia(), Revision: 1,
		Estado:         domain.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: "seguridad:integracion", ActualizadoEn: publicada,
	}
	asignacion := domain.AsignacionPerfil{
		AsignacionID: "ejecutor_documental_v4_integracion", Version: 1,
		PerfilActivoRef: datosVinculo.PerfilActivoRef,
		PrincipalID:     datosVinculo.PrincipalID, VersionRolRef: rol.Referencia(),
		Estado: domain.EstadoAsignacionPerfilActiva,
		Ambitos: []domain.AmbitoPerfil{{
			Clave: "organizacion", Valores: []string{"diputacion_granada"},
		}},
		VigenteDesde: publicada, VigenteHasta: ancla.Add(time.Hour),
		EmitidaPor: "rrhh:integracion", EmitidaEn: publicada,
	}
	if rol.Validar() != nil || control.Validar() != nil || asignacion.Validar() != nil {
		t.Fatal("configuracion de autorizacion de integracion invalida")
	}

	material := generarMaterialFirmaCOSEPrueba(
		t, AlgoritmoCOSEDocumentalEdDSA, []byte("clave:pdp:integracion-v4"),
	)
	raiz, err := nuevaRaizPublicaFijadaAtestacionPDP(
		material.claveID, material.algoritmoDocumental, material.publica,
		suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
		audienciaDespliegueAtestacionPDPPrueba,
		EstadoConfianzaClaveDocumentalActiva,
		publicada, ancla.Add(2*time.Hour), time.Time{},
	)
	if err != nil {
		t.Fatalf("crear raiz de integracion: %v", err)
	}
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		"confianza:pdp:integracion-v4", publicada, ancla.Add(time.Hour), raiz,
	)
	if err != nil {
		t.Fatalf("crear configuracion de confianza: %v", err)
	}

	fixture := fixturePostgreSQLDocumentalV4{
		ancla: ancla, datosVinculo: datosVinculo, rol: rol, controlRol: control,
		asignacion: asignacion, materialFirma: material, raiz: raiz,
		configuracion: configuracion,
		secretoHMAC:   []byte("0123456789abcdef0123456789abcdef"),
		claveHMACID:   "capacidad:hmac:integracion-v4",
		emisorID:      "emisor:capacidad:integracion-v4",
	}
	for indice := 1; indice <= 3; indice++ {
		fixture.casos = append(fixture.casos, nuevoCasoPostgreSQLDocumentalV4(
			t, fixture, vinculoAutenticacion, indice,
		))
	}
	return fixture
}

func nuevoCasoPostgreSQLDocumentalV4(
	t *testing.T,
	fixture fixturePostgreSQLDocumentalV4,
	vinculoAutenticacion domain.VinculoAutenticacionActorV1,
	indice int,
) casoPostgreSQLDocumentalV4 {
	t.Helper()
	sufijo := strconv.Itoa(indice)
	recurso := domain.RecursoAutorizable{
		Referencia: "recurso:documental:v4:integracion:" + sufijo,
		ModuloID:   "bolsa", Tipo: "documento_bolsa",
		Ambitos: map[string]string{"organizacion": "diputacion_granada"},
		Atributos: map[string]string{
			ports.AtributoAutorizacionDocumentalEfectoRef:        "efecto:documental:v4:integracion:" + sufijo,
			ports.AtributoAutorizacionDocumentalHuellaPlanSHA256: strings.Repeat(strconv.Itoa(indice), 64),
		},
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella de recurso %d: %v", indice, err)
	}
	huellaRol, _ := fixture.rol.HuellaSHA256()
	huellaControl, _ := fixture.controlRol.HuellaSHA256()
	huellaAsignacion, _ := fixture.asignacion.HuellaSHA256()
	huellaCatalogo, err := domain.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatalf("huella de catalogo vacio: %v", err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:documental:v4:integracion:" + sufijo,
		Concedida:   true, Codigo: "concedida",
		PrincipalID:     fixture.datosVinculo.PrincipalID,
		PerfilActivoRef: fixture.datosVinculo.PerfilActivoRef,
		Accion:          ports.AccionEjecutarPlanDocumentalV4,
		RecursoRef:      recurso.Referencia, ModuloID: recurso.ModuloID,
		TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad: "tramitar_bolsa", CorrelacionRef: "correlacion:documental:v4:" + sufijo,
		VinculoAutenticacionActor: vinculoAutenticacion,
		AsignacionRef:             fixture.asignacion.Referencia(), AsignacionHuellaSHA256: huellaAsignacion,
		VersionRolRef: fixture.rol.Referencia(), VersionRolHuellaSHA256: huellaRol,
		ControlVigenciaVersionRolRef:          fixture.controlRol.VersionRolRef,
		ControlVigenciaVersionRolRevision:     fixture.controlRol.Revision,
		ControlVigenciaVersionRolHuellaSHA256: huellaControl,
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasRefs: []string{}, PoliticasEvaluadasHuellasSHA256: map[string]string{},
		PoliticasRefs: []string{}, PoliticasHuellasSHA256: map[string]string{},
		GarantiaMinima:   domain.AuthAssuranceHigh,
		CamposPermitidos: []string{"documento.generado"}, Obligaciones: []string{},
		EmitidaEn: fixture.ancla, ValidaHasta: fixture.ancla.Add(2 * time.Minute),
	}
	if err = decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision %d invalida: %v", indice, err)
	}
	verificadaEn := fixture.ancla.Add(time.Microsecond)
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(decision, verificadaEn)
	if err != nil {
		t.Fatalf("crear evidencia %d: %v", indice, err)
	}
	expectativa := ports.ExpectativaAutorizacionEjecucionDocumentalV4{
		DecisionEsperada: decision,
		PrincipalID:      decision.PrincipalID, PerfilActivoRef: decision.PerfilActivoRef,
		AutenticacionRef:          fixture.datosVinculo.AutenticacionRef,
		SesionRef:                 fixture.datosVinculo.SesionRef,
		ControlSesionRef:          fixture.datosVinculo.ControlSesionRef,
		ControlSesionRevision:     fixture.datosVinculo.ControlSesionRevision,
		ControlSesionHuellaSHA256: fixture.datosVinculo.ControlSesionHuellaSHA256,
		ContextoActorRef:          fixture.datosVinculo.ContextoActorRef,
		ContextoActorVersion:      fixture.datosVinculo.ContextoActorVersion,
		ContextoActorHuellaSHA256: fixture.datosVinculo.ContextoActorHuellaSHA256,
		Recurso:                   recurso, Finalidad: decision.Finalidad, CorrelacionRef: decision.CorrelacionRef,
		EfectoRef:                 recurso.Atributos[ports.AtributoAutorizacionDocumentalEfectoRef],
		HuellaPlanSHA256:          recurso.Atributos[ports.AtributoAutorizacionDocumentalHuellaPlanSHA256],
		CamposPermitidosEsperados: []string{"documento.generado"},
		ObligacionesEsperadas:     []string{}, CumplimientosObligacionesPorRef: map[string]string{},
	}
	vinculo, err := ports.NuevaSolicitudVinculadaAutorizacionEjecucionDocumentalV4(
		evidencia, expectativa, fixture.ancla.Add(2*time.Microsecond),
	)
	if err != nil {
		t.Fatalf("vincular decision %d: %v", indice, err)
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV1,
		Suite:          suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
		ClaveID:        string(fixture.materialFirma.claveID),
		Audiencia:      audienciaDespliegueAtestacionPDPPrueba,
	}
	payload, err := domain.SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		t.Fatalf("serializar decision firmada %d: %v", indice, err)
	}
	solicitudCOSE := nuevaSolicitudCOSEPrueba(
		t, payload, AudienciaCOSEAtestacionAutorizacionPDP,
	)
	sobre := firmarSobreCOSEPrueba(
		t, fixture.materialFirma, payload, solicitudCOSE, nil, nil,
	)
	return casoPostgreSQLDocumentalV4{
		decision: decision, vinculo: vinculo, cabecera: cabecera,
		payload: payload, sobre: sobre,
	}
}

func sembrarFixturePostgreSQLDocumentalV4(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture fixturePostgreSQLDocumentalV4,
) {
	t.Helper()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatalf("iniciar siembra V4: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	documentoRol := serializarJSONPostgreSQLDocumentalV4Prueba(t, fixture.rol)
	huellaRol, _ := fixture.rol.HuellaSHA256()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.version_rol
		(version_rol_ref, rol_id, version, huella_sha256, publicada_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb)`, fixture.rol.Referencia(), fixture.rol.RolID,
		fixture.rol.Version, huellaRol, fixture.rol.PublicadaEn, documentoRol)
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar rol")
	documentoControl := serializarJSONPostgreSQLDocumentalV4Prueba(t, fixture.controlRol)
	huellaControl, _ := fixture.controlRol.HuellaSHA256()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol
		(version_rol_ref, revision, estado, huella_sha256, actualizado_en, documento)
		VALUES ($1,$2::numeric,$3,$4,$5,$6::jsonb)`, fixture.controlRol.VersionRolRef,
		strconv.FormatUint(fixture.controlRol.Revision, 10), fixture.controlRol.Estado,
		huellaControl, fixture.controlRol.ActualizadoEn, documentoControl)
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar control de rol")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
		(version_rol_ref, revision, actualizada_en, actualizada_por, acto_ref)
		VALUES ($1,$2::numeric,$3,$4,$5)`, fixture.controlRol.VersionRolRef,
		strconv.FormatUint(fixture.controlRol.Revision, 10), fixture.controlRol.ActualizadoEn,
		"seguridad:integracion", "acto:integracion:control-rol")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar puntero de rol")

	documentoAsignacion := serializarJSONPostgreSQLDocumentalV4Prueba(t, fixture.asignacion)
	huellaAsignacion, _ := fixture.asignacion.HuellaSHA256()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.asignacion_perfil
		(asignacion_ref, asignacion_id, version, perfil_activo_ref, principal_id,
		 version_rol_ref, huella_sha256, emitida_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`, fixture.asignacion.Referencia(),
		fixture.asignacion.AsignacionID, fixture.asignacion.Version,
		fixture.asignacion.PerfilActivoRef, fixture.asignacion.PrincipalID,
		fixture.asignacion.VersionRolRef, huellaAsignacion, fixture.asignacion.EmitidaEn,
		documentoAsignacion)
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar asignacion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.asignacion_perfil_actual
		(perfil_activo_ref, asignacion_ref, actualizada_en, actualizada_por, acto_ref)
		VALUES ($1,$2,$3,$4,$5)`, fixture.asignacion.PerfilActivoRef,
		fixture.asignacion.Referencia(), fixture.asignacion.EmitidaEn,
		"rrhh:integracion", "acto:integracion:asignacion")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar puntero de asignacion")

	datos := fixture.datosVinculo
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
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar sesion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_sesion_v1
		(control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
		 sesion_revalidada_en, sesion_valida_hasta)
		VALUES ($1,$2::numeric,$3,'activa',$4,$5,$6)`, datos.ControlSesionRef,
		strconv.FormatUint(datos.ControlSesionRevision, 10), datos.SesionRef,
		datos.ControlSesionHuellaSHA256, datos.SesionRevalidadaEn, datos.SesionValidaHasta)
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar control de sesion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.control_sesion_actual_v1
		(sesion_ref, control_sesion_ref, revision, actualizada_en, acto_ref)
		VALUES ($1,$2,$3::numeric,$4,$5)`, datos.SesionRef, datos.ControlSesionRef,
		strconv.FormatUint(datos.ControlSesionRevision, 10), datos.SesionRevalidadaEn,
		"acto:integracion:control-sesion")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar puntero de sesion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.contexto_actor_v1
		(contexto_actor_ref, version, cuenta_ref, principal_id, perfil_activo_ref,
		 estado, huella_sha256, vigente_desde, vigente_hasta)
		VALUES ($1,$2::numeric,$3,$4,$5,'activo',$6,$7,$8)`, datos.ContextoActorRef,
		strconv.FormatUint(datos.ContextoActorVersion, 10), datos.CuentaRef,
		datos.PrincipalID, datos.PerfilActivoRef, datos.ContextoActorHuellaSHA256,
		fixture.ancla.Add(-time.Hour), fixture.ancla.Add(30*time.Minute))
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar contexto de actor")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_autorizacion.contexto_actor_actual_v1
		(cuenta_ref, perfil_activo_ref, contexto_actor_ref, version, actualizada_en, acto_ref)
		VALUES ($1,$2,$3,$4::numeric,$5,$6)`, datos.CuentaRef, datos.PerfilActivoRef,
		datos.ContextoActorRef, strconv.FormatUint(datos.ContextoActorVersion, 10),
		fixture.ancla, "acto:integracion:contexto-actor")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar puntero de actor")

	exigirSQLPostgreSQLDocumentalV4Prueba(
		t, tx.Commit(ctx), "confirmar siembra de autorizacion e identidad",
	)

	almacenAutorizacion, err := postgresvec.NuevoAlmacenAutorizacion(pool)
	if err != nil {
		t.Fatalf("crear almacen real de autorizacion para siembra V4: %v", err)
	}
	for _, caso := range fixture.casos {
		if err = almacenAutorizacion.RegistrarDecisionSiInstantaneaVigente(
			ctx, caso.decision,
		); err != nil {
			t.Fatalf("registrar decision %s mediante el adaptador real: %v",
				caso.decision.DecisionRef, err)
		}
	}

	tx, err = pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatalf("iniciar siembra de confianza documental V4: %v", err)
	}

	spki, err := x509.MarshalPKIXPublicKey(fixture.materialFirma.publica)
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "serializar SPKI")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_ejecucion_documental_v4.configuracion_confianza_version
		(revision, huella_configuracion_sha256, publicada_en, expira_en, estado,
		 revocada_en, acto_ref)
		VALUES ($1,$2,$3,$4,'activa',NULL,$5)`, fixture.configuracion.revision,
		fixture.configuracion.huellaSHA256, fixture.configuracion.publicadaEn,
		fixture.configuracion.expiraEn, "acto:integracion:configuracion")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar configuracion de confianza")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_ejecucion_documental_v4.configuracion_confianza_actual
		(control_id, revision, huella_configuracion_sha256, actualizada_en, acto_ref)
		VALUES (true,$1,$2,$3,$4)`, fixture.configuracion.revision,
		fixture.configuracion.huellaSHA256, fixture.ancla,
		"acto:integracion:configuracion-actual")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar puntero de configuracion")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_version
		(clave_id, version, revision_configuracion, huella_configuracion_sha256,
		 algoritmo_cose, suite, audiencia_cose, audiencia_despliegue,
		 clave_publica_spki, huella_clave_sha256, valida_desde, valida_hasta,
		 estado, revocada_en, acto_ref)
		VALUES ($1,1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'activa',NULL,$12)`,
		string(fixture.raiz.claveID), fixture.configuracion.revision,
		fixture.configuracion.huellaSHA256, fixture.raiz.algoritmo,
		fixture.raiz.suiteAtestacionPDP, fixture.raiz.audiencia,
		fixture.raiz.audienciaDespliegueAtestacionPDP, spki,
		fixture.raiz.huellaClaveSHA256, fixture.raiz.validaDesde,
		fixture.raiz.validaHasta, "acto:integracion:raiz")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar raiz")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_actual
		(clave_id, version, revision_configuracion, huella_configuracion_sha256,
		 actualizada_en, acto_ref)
		VALUES ($1,1,$2,$3,$4,$5)`, string(fixture.raiz.claveID),
		fixture.configuracion.revision, fixture.configuracion.huellaSHA256,
		fixture.ancla, "acto:integracion:raiz-actual")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar puntero de raiz")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_ejecucion_documental_v4.clave_capacidad_version
		(clave_id, version, secreto_hmac, huella_secreto_sha256, emisor_id,
		 valida_desde, valida_hasta, estado, revocada_en, acto_ref)
		VALUES ($1,1,$2,$3,$4,$5,$6,'activa',NULL,$7)`, fixture.claveHMACID,
		fixture.secretoHMAC, huellaBytesDocumentales(fixture.secretoHMAC), fixture.emisorID,
		fixture.ancla.Add(-10*time.Minute), fixture.ancla.Add(time.Hour),
		"acto:integracion:clave-capacidad")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar clave de capacidad")
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_ejecucion_documental_v4.clave_capacidad_actual
		(control_id, clave_id, version, actualizada_en, acto_ref)
		VALUES (true,$1,1,$2,$3)`, fixture.claveHMACID, fixture.ancla,
		"acto:integracion:clave-capacidad-actual")
	exigirSQLPostgreSQLDocumentalV4Prueba(t, err, "insertar puntero de capacidad")

	exigirSQLPostgreSQLDocumentalV4Prueba(
		t, tx.Commit(ctx), "confirmar siembra de confianza documental V4",
	)
}

func verificarSeparacionCredencialesPostgreSQLDocumentalV4(
	t *testing.T,
	ctx context.Context,
	poolEmisor, poolEjecucion *pgxpool.Pool,
) {
	t.Helper()
	var emisorLeeMaterial, emisorEjecuta bool
	err := poolEmisor.QueryRow(ctx, `
		SELECT has_function_privilege(
		           current_user,
		           'vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()',
		           'EXECUTE'
		       ),
		       has_function_privilege(
		           current_user,
		           'vec_ejecucion_documental_v4.ejecutar_plan_atestado(bytea,bytea,bytea,bytea,bytea,bytea,bytea,jsonb)',
		           'EXECUTE'
		       )`).Scan(&emisorLeeMaterial, &emisorEjecuta)
	if err != nil || !emisorLeeMaterial || emisorEjecuta {
		t.Fatalf("capacidades del emisor incorrectas: material=%t ejecutar=%t err=%v",
			emisorLeeMaterial, emisorEjecuta, err)
	}
	_, err = poolEmisor.Exec(ctx, `
		SELECT * FROM vec_ejecucion_documental_v4.ejecutar_plan_atestado(
			NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,NULL::bytea,
			NULL::bytea,NULL::bytea,NULL::jsonb
		)`)
	exigirPrivilegioDenegadoPostgreSQLDocumentalV4Prueba(t, err, "emisor ejecutando efecto")

	var ejecutorLeeMaterial, ejecutorEjecuta bool
	err = poolEjecucion.QueryRow(ctx, `
		SELECT has_function_privilege(
		           current_user,
		           'vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()',
		           'EXECUTE'
		       ),
		       has_function_privilege(
		           current_user,
		           'vec_ejecucion_documental_v4.ejecutar_plan_atestado(bytea,bytea,bytea,bytea,bytea,bytea,bytea,jsonb)',
		           'EXECUTE'
		       )`).Scan(&ejecutorLeeMaterial, &ejecutorEjecuta)
	if err != nil || ejecutorLeeMaterial || !ejecutorEjecuta {
		t.Fatalf("capacidades del ejecutor incorrectas: material=%t ejecutar=%t err=%v",
			ejecutorLeeMaterial, ejecutorEjecuta, err)
	}
	_, err = poolEjecucion.Exec(ctx,
		`SELECT * FROM vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()`)
	exigirPrivilegioDenegadoPostgreSQLDocumentalV4Prueba(t, err, "ejecutor leyendo secreto")

	for nombre, pool := range map[string]*pgxpool.Pool{
		"emisor": poolEmisor, "ejecutor": poolEjecucion,
	} {
		var tablasConPrivilegio int
		err = pool.QueryRow(ctx, `
			SELECT count(*)
			  FROM pg_catalog.pg_class AS clase
			  JOIN pg_catalog.pg_namespace AS espacio ON espacio.oid=clase.relnamespace
			 WHERE espacio.nspname='vec_ejecucion_documental_v4'
			   AND clase.relkind IN ('r','p')
			   AND (has_table_privilege(current_user,clase.oid,'SELECT')
			        OR has_table_privilege(current_user,clase.oid,'INSERT')
			        OR has_table_privilege(current_user,clase.oid,'UPDATE')
			        OR has_table_privilege(current_user,clase.oid,'DELETE'))`).Scan(&tablasConPrivilegio)
		if err != nil || tablasConPrivilegio != 0 {
			t.Fatalf("%s conserva privilegios directos sobre %d tablas: %v",
				nombre, tablasConPrivilegio, err)
		}
	}
}

func solicitarArtefactosPostgreSQLDocumentalV4Prueba(
	t *testing.T,
	ctx context.Context,
	ejecutor *EjecutorDocumentalPostgreSQLV4,
	caso casoPostgreSQLDocumentalV4,
) artefactosEjecucionDocumentalV4 {
	t.Helper()
	solicitud, err := caso.vinculo.PrepararSolicitudAplicacionEn(
		time.Now().UTC().Truncate(time.Microsecond),
	)
	if err != nil {
		t.Fatalf("preparar preimagen para emisor: %v", err)
	}
	preimagen, err := solicitud.PreimagenRecursoParaEvidenciaDurable()
	if err != nil {
		t.Fatalf("extraer preimagen: %v", err)
	}
	preimagenBytes, err := preimagen.SerializacionCanonicaParaPersistencia()
	if err != nil {
		t.Fatalf("serializar preimagen: %v", err)
	}
	sobreBytes, err := caso.sobre.COSESign1()
	if err != nil {
		t.Fatalf("leer COSE: %v", err)
	}
	artefactos, err := ejecutor.cliente.solicitar(
		ctx, caso.cabecera, sobreBytes, preimagenBytes,
	)
	if err != nil {
		t.Fatalf("obtener capacidad del emisor aislado: %v", err)
	}
	return artefactos
}

func verificarEfectoCompletoPostgreSQLDocumentalV4(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	caso casoPostgreSQLDocumentalV4,
	resultado ResultadoEjecucionPlanDocumentalV4,
) {
	t.Helper()
	verificarFilasDecisionPostgreSQLDocumentalV4(
		t, ctx, pool, caso.decision.DecisionRef, 1,
	)
	var estado, decisionRef, auditoriaRef, eventoRef string
	err := pool.QueryRow(ctx, `
		SELECT orden.estado, orden.decision_ref, auditoria.auditoria_ref,
		       evento.evento_ref
		  FROM vec_ejecucion_documental_v4.orden_generacion_documental AS orden
		  JOIN vec_ejecucion_documental_v4.auditoria AS auditoria
		    ON auditoria.decision_ref=orden.decision_ref
		  JOIN vec_ejecucion_documental_v4.evento_outbox AS evento
		    ON evento.decision_ref=orden.decision_ref
		 WHERE orden.decision_ref=$1`, caso.decision.DecisionRef).Scan(
		&estado, &decisionRef, &auditoriaRef, &eventoRef,
	)
	if err != nil || estado != estadoOrdenGeneracionPendienteV4 ||
		decisionRef != caso.decision.DecisionRef || resultado.OrdenRef == "" ||
		resultado.Estado != estado || resultado.AuditoriaRef != auditoriaRef ||
		resultado.EventoOutboxRef != eventoRef {
		t.Fatalf("efecto documental inconsistente: resultado=%+v estado=%s decision=%s auditoria=%s evento=%s err=%v",
			resultado, estado, decisionRef, auditoriaRef, eventoRef, err)
	}

	sobreEsperado, _ := caso.sobre.COSESign1()
	evidencia, err := ports.NuevaEvidenciaUsoDecisionAutorizacion(
		caso.decision, caso.decision.EmitidaEn.Add(time.Microsecond),
	)
	datosEvidencia, errDatos := evidencia.Datos()
	decisionCanonica, errCanonica := datosEvidencia.RepresentacionCanonica()
	if err != nil || errDatos != nil || errCanonica != nil {
		t.Fatalf("representar decision canonica: %v", err)
	}
	var payload, sobre, decisionGuardada []byte
	err = pool.QueryRow(ctx, `
		SELECT payload_vec_ad_1, sobre_cose_sign1, decision_canonica
		  FROM vec_ejecucion_documental_v4.atestacion_pdp
		 WHERE decision_ref=$1`, caso.decision.DecisionRef).Scan(
		&payload, &sobre, &decisionGuardada,
	)
	if err != nil || !bytes.Equal(payload, caso.payload) ||
		!bytes.Equal(sobre, sobreEsperado) || !bytes.Equal(decisionGuardada, decisionCanonica) {
		t.Fatalf("la atestacion no conserva los bytes exactos: %v", err)
	}
}

func verificarPrecondicionesArtefactosPostgreSQLDocumentalV4Prueba(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	artefactos artefactosEjecucionDocumentalV4,
) {
	t.Helper()
	var metadatos struct {
		Aplicacion json.RawMessage `json:"aplicacion"`
	}
	if err := json.Unmarshal(artefactos.metadatos, &metadatos); err != nil {
		t.Fatalf("leer aplicacion emitida: %v", err)
	}
	var autorizacionVigente bool
	err := pool.QueryRow(ctx, `
		SELECT vec_autorizacion.revalidar_decision_ejecucion_documental_v4(
			$1::jsonb, $2::bytea
		)`, string(metadatos.Aplicacion), artefactos.decisionCanonica).Scan(&autorizacionVigente)
	if err != nil || !autorizacionVigente {
		t.Fatalf("PostgreSQL no revalida la decision canonica emitida: vigente=%t err=%v",
			autorizacionVigente, err)
	}
	var hmacValido bool
	err = pool.QueryRow(ctx, `
		SELECT encode(public.hmac(
		           vec_ejecucion_documental_v4.preimagen_capacidad($1::jsonb),
		           clave.secreto_hmac, 'sha256'
		       ), 'hex') = $1::jsonb ->> 'mac_sha256'
		  FROM vec_ejecucion_documental_v4.clave_capacidad_actual AS actual
		  JOIN vec_ejecucion_documental_v4.clave_capacidad_version AS clave
		    ON clave.clave_id=actual.clave_id AND clave.version=actual.version
		 WHERE actual.control_id=true`, string(artefactos.capacidad)).Scan(&hmacValido)
	if err != nil || !hmacValido {
		t.Fatalf("PostgreSQL no reproduce el HMAC emitido: valido=%t err=%v", hmacValido, err)
	}
}

func verificarEjecucionReversiblePostgreSQLDocumentalV4Prueba(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	artefactos artefactosEjecucionDocumentalV4,
) {
	t.Helper()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		t.Fatalf("iniciar diagnostico reversible: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err = configurarTransaccionEjecucionDocumentalV4(ctx, tx); err != nil {
		t.Fatalf("configurar diagnostico reversible: %v", err)
	}
	var estadoResultado string
	var resultado ResultadoEjecucionPlanDocumentalV4
	err = tx.QueryRow(ctx, `
		SELECT resultado, orden_ref, estado_orden, auditoria_ref,
		       evento_outbox_ref, registrada_en
		  FROM vec_ejecucion_documental_v4.ejecutar_plan_atestado(
		       $1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::bytea,
		       $7::bytea,$8::jsonb
		  )`, artefactos.metadatos, artefactos.payload, artefactos.sobre,
		artefactos.evidencia, artefactos.preimagen, artefactos.decisionCanonica,
		artefactos.efecto, string(artefactos.capacidad)).Scan(
		&estadoResultado, &resultado.OrdenRef, &resultado.Estado,
		&resultado.AuditoriaRef, &resultado.EventoOutboxRef, &resultado.RegistradaEn,
	)
	resultado.RegistradaEn = resultado.RegistradaEn.UTC().Truncate(time.Microsecond)
	errValidacion := resultado.validarContraArtefactos(artefactos)
	if err != nil || estadoResultado != "ejecutada" || errValidacion != nil {
		var efecto efectoEjecucionDocumentalV4PostgreSQL
		errEfecto := json.Unmarshal(artefactos.efecto, &efecto)
		t.Fatalf("la funcion atomica rechazo artefactos validos antes de concurrencia: estado=%s orden=%q/%q estado_orden=%q/%q auditoria=%q/%q evento=%q/%q registrada=%s solicitada=%s localizacion=%q err=%v validacion=%v efecto=%v",
			estadoResultado, resultado.OrdenRef, efecto.OrdenRef,
			resultado.Estado, efecto.Estado, resultado.AuditoriaRef, efecto.AuditoriaRef,
			resultado.EventoOutboxRef, efecto.EventoOutboxRef,
			resultado.RegistradaEn.Format(time.RFC3339Nano),
			efecto.SolicitadaEn.Format(time.RFC3339Nano), resultado.RegistradaEn.Location().String(),
			err, errValidacion, errEfecto)
	}
	// No confirma: demuestra la ruta SQL completa sin consumir el caso que
	// inmediatamente despues se ejercita de forma concurrente.
	if err = tx.Rollback(ctx); err != nil {
		t.Fatalf("revertir diagnostico de ejecucion: %v", err)
	}
}

func verificarFilasDecisionPostgreSQLDocumentalV4(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	decisionRef string,
	esperadas int,
) {
	t.Helper()
	tablas := []string{
		"atestacion_pdp", "orden_generacion_documental", "consumo_decision_atomico",
		"consumo_capacidad", "auditoria", "evento_outbox",
	}
	for _, tabla := range tablas {
		var total int
		consulta := "SELECT count(*) FROM vec_ejecucion_documental_v4." + tabla +
			" WHERE decision_ref=$1"
		if err := pool.QueryRow(ctx, consulta, decisionRef).Scan(&total); err != nil || total != esperadas {
			t.Fatalf("atomicidad rota en %s para %s: total=%d esperado=%d err=%v",
				tabla, decisionRef, total, esperadas, err)
		}
	}
}

func revocarRaizPostgreSQLDocumentalV4(
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture fixturePostgreSQLDocumentalV4,
) error {
	spki, err := x509.MarshalPKIXPublicKey(fixture.materialFirma.publica)
	if err != nil {
		return err
	}
	instante := time.Now().UTC().Truncate(time.Microsecond)
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	_, err = tx.Exec(ctx, `
		INSERT INTO vec_ejecucion_documental_v4.raiz_confianza_version
		(clave_id, version, revision_configuracion, huella_configuracion_sha256,
		 algoritmo_cose, suite, audiencia_cose, audiencia_despliegue,
		 clave_publica_spki, huella_clave_sha256, valida_desde, valida_hasta,
		 estado, revocada_en, acto_ref)
		VALUES ($1,2,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'revocada',$12,$13)`,
		string(fixture.raiz.claveID), fixture.configuracion.revision,
		fixture.configuracion.huellaSHA256, fixture.raiz.algoritmo,
		fixture.raiz.suiteAtestacionPDP, fixture.raiz.audiencia,
		fixture.raiz.audienciaDespliegueAtestacionPDP, spki,
		fixture.raiz.huellaClaveSHA256, fixture.raiz.validaDesde,
		fixture.raiz.validaHasta, instante, "acto:integracion:revocar-raiz")
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		UPDATE vec_ejecucion_documental_v4.raiz_confianza_actual
		   SET version=2, actualizada_en=$1, acto_ref=$2
		 WHERE clave_id=$3`, instante, "acto:integracion:raiz-revocada",
		string(fixture.raiz.claveID))
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func verificarRaizRevocadaPostgreSQLDocumentalV4(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	fixture fixturePostgreSQLDocumentalV4,
) {
	t.Helper()
	var version int
	var estado string
	var revocada bool
	err := pool.QueryRow(ctx, `
		SELECT actual.version::integer, version.estado,
		       version.revocada_en IS NOT NULL
		  FROM vec_ejecucion_documental_v4.raiz_confianza_actual AS actual
		  JOIN vec_ejecucion_documental_v4.raiz_confianza_version AS version
		    ON version.clave_id=actual.clave_id AND version.version=actual.version
		 WHERE actual.clave_id=$1`, string(fixture.raiz.claveID)).Scan(
		&version, &estado, &revocada,
	)
	if err != nil || version != 2 || estado != "revocada" || !revocada {
		t.Fatalf("raiz no quedo revocada monotonicamente: version=%d estado=%s revocada=%t err=%v",
			version, estado, revocada, err)
	}
}

func abrirPoolPostgreSQLDocumentalV4Prueba(
	t *testing.T,
	ctx context.Context,
	dsn, nombre string,
) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("abrir pool %s: %v", nombre, err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("conectar pool %s: %v", nombre, err)
	}
	return pool
}

func servirEmisorUnixPostgreSQLDocumentalV4Prueba(
	t *testing.T,
	ruta string,
	manejador http.Handler,
) func() {
	t.Helper()
	escucha, err := net.Listen("unix", ruta)
	if err != nil {
		t.Fatalf("abrir socket Unix de prueba: %v", err)
	}
	servidor := &http.Server{
		Handler: manejador, ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second,
		IdleTimeout: 5 * time.Second,
	}
	finalizado := make(chan error, 1)
	go func() { finalizado <- servidor.Serve(escucha) }()
	return func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = servidor.Shutdown(ctx)
		errServidor := <-finalizado
		if errServidor != nil && !errors.Is(errServidor, http.ErrServerClosed) {
			t.Errorf("cerrar emisor Unix: %v", errServidor)
		}
	}
}

func serializarJSONPostgreSQLDocumentalV4Prueba(t *testing.T, valor any) []byte {
	t.Helper()
	contenido, err := json.Marshal(valor)
	if err != nil {
		t.Fatalf("serializar JSON de integracion: %v", err)
	}
	return contenido
}

func exigirSQLPostgreSQLDocumentalV4Prueba(t *testing.T, err error, operacion string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", operacion, err)
	}
}

func exigirPrivilegioDenegadoPostgreSQLDocumentalV4Prueba(
	t *testing.T,
	err error,
	operacion string,
) {
	t.Helper()
	var errorPG *pgconn.PgError
	if err == nil || !errors.As(err, &errorPG) || errorPG.Code != "42501" {
		t.Fatalf("%s no fue denegada por privilegios: %v", operacion, err)
	}
}
