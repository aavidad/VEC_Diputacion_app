package postgres

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	variableDSNBolsaPostgreSQLE2E      = "VEC_BOLSA_POSTGRES_E2E_DSN"
	variableDSNAdminBolsaPostgreSQLE2E = "VEC_BOLSA_POSTGRES_E2E_ADMIN_DSN"
	variableFaseBolsaPostgreSQLE2E     = "VEC_BOLSA_POSTGRES_E2E_FASE"
	variableEstadoBolsaPostgreSQLE2E   = "VEC_BOLSA_POSTGRES_E2E_ESTADO"

	fasePrevalidarFalloBolsaPostgreSQLE2E       = "prevalidar_fallo"
	faseConfirmarBolsaPostgreSQLE2E             = "confirmar"
	faseRecuperarBolsaPostgreSQLE2E             = "recuperar"
	etapaConfirmadaPendienteReplayPostgreSQLE2E = "confirmada_pendiente_replay"
	etapaConfirmadaReproducidaPostgreSQLE2E     = "confirmada_reproducida"

	baremacionRefBolsaPostgreSQLE2E = "baremacion:e2e:postgresql:v3:real"
	procesoRefBolsaPostgreSQLE2E    = "proceso:e2e:postgresql:v3:real"
	solicitudRefBolsaPostgreSQLE2E  = "solicitud:e2e:postgresql:v3:real"
	sujetoRefBolsaPostgreSQLE2E     = "sujeto:e2e:postgresql:v3:real"
	correlacionBolsaPostgreSQLE2E   = "correlacion:e2e:postgresql:v3:real"
	finalidadBolsaPostgreSQLE2E     = "gestion_bolsa"
)

var claveHMACBolsaPostgreSQLE2E = sha256.Sum256([]byte(
	"clave controlada del E2E real Go a PostgreSQL para Bolsa V3",
))

type relojBolsaPostgreSQLE2E struct{ ahora time.Time }

func (r relojBolsaPostgreSQLE2E) Ahora() time.Time { return r.ahora }

func instanteAutoritativoBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) time.Time {
	t.Helper()
	var instante time.Time
	if err := pool.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&instante); err != nil {
		t.Fatalf("obtener reloj autoritativo PostgreSQL para E2E: %v", err)
	}
	// Fuerza de forma determinista al menos un cero fraccional final. El
	// contrato canonico exige seis digitos: RFC3339Nano no puede volver a
	// ocultar este defecto de serializacion por probabilidad.
	instante = instante.UTC().Truncate(10 * time.Microsecond)
	if instante.IsZero() {
		t.Fatal("PostgreSQL devolvio un instante autoritativo nulo")
	}
	return instante
}

type generadorDecisionBolsaPostgreSQLE2E string

func (g generadorDecisionBolsaPostgreSQLE2E) NuevaReferenciaDecisionAutorizacion() (string, error) {
	return string(g), nil
}

type revalidadorActorBolsaPostgreSQLE2E struct {
	resultado dominiovec.AutenticacionRevalidadaV1
}

func (r revalidadorActorBolsaPostgreSQLE2E) RevalidarAutenticacionActorV1(
	context.Context,
	dominiovec.SolicitudRevalidacionAutenticacionActorV1,
) (dominiovec.AutenticacionRevalidadaV1, error) {
	return r.resultado, nil
}

type registroDenegacionesBolsaPostgreSQLE2E struct{}

func (registroDenegacionesBolsaPostgreSQLE2E) RegistrarDenegacionAutorizacion(
	context.Context,
	dominiovec.DecisionAutorizacion,
) error {
	return nil
}

type registroConcesionesNuloBolsaPostgreSQLE2E struct{}

func (registroConcesionesNuloBolsaPostgreSQLE2E) RegistrarDecisionSiInstantaneaVigente(
	context.Context,
	dominiovec.DecisionAutorizacion,
) error {
	return nil
}

type observadorSellosBolsaPostgreSQLE2E struct {
	mu     sync.Mutex
	sellos []string
}

func (o *observadorSellosBolsaPostgreSQLE2E) registrar(sello string) {
	if o == nil {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.sellos = append(o.sellos, sello)
}

func (o *observadorSellosBolsaPostgreSQLE2E) contieneEnOrden(
	primero string,
	segundo string,
) bool {
	if o == nil || primero == "" || segundo == "" || primero == segundo {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	primeroVisto := false
	for _, sello := range o.sellos {
		if sello == primero {
			primeroVisto = true
		}
		if sello == segundo && primeroVisto {
			return true
		}
	}
	return false
}

func (o *observadorSellosBolsaPostgreSQLE2E) contieneTodos(esperados ...string) bool {
	if o == nil {
		return false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, esperado := range esperados {
		encontrado := false
		for _, sello := range o.sellos {
			if sello == esperado {
				encontrado = true
				break
			}
		}
		if !encontrado {
			return false
		}
	}
	return true
}

type protectorSellosBolsaPostgreSQLE2E struct {
	fallaSelloHistorico string
	observador          *observadorSellosBolsaPostgreSQLE2E
}

func (p protectorSellosBolsaPostgreSQLE2E) VerificarSelloBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudVerificarSelloBaremacion,
) error {
	if ctx == nil {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	p.observador.registrar(solicitud.SelloHMAC)
	if p.fallaSelloHistorico != "" && solicitud.SelloHMAC == p.fallaSelloHistorico {
		return puertosbolsa.ErrVerificacionSelloBaremacionNoDisponible
	}
	if solicitud.Validar() != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	esperado, err := selloHMACBolsaPostgreSQLE2E(
		solicitud.Finalidad,
		solicitud.RepresentacionCanonica,
	)
	if err != nil {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	recibido, err := hex.DecodeString(strings.TrimPrefix(
		solicitud.SelloHMAC,
		"hmac-sha256:e2e_postgresql_v3:",
	))
	esperadoBinario, errorEsperado := hex.DecodeString(strings.TrimPrefix(
		esperado,
		"hmac-sha256:e2e_postgresql_v3:",
	))
	defer borrarBytesE2E(recibido, esperadoBinario)
	if err != nil || errorEsperado != nil || len(recibido) != sha256.Size ||
		subtle.ConstantTimeCompare(recibido, esperadoBinario) != 1 {
		return puertosbolsa.ErrSelloBaremacionNoAutentico
	}
	return nil
}

func selloHMACBolsaPostgreSQLE2E(
	finalidad puertosbolsa.FinalidadSelloBaremacion,
	representacion puertosbolsa.CargaProtegida,
) (string, error) {
	material, err := (puertosbolsa.SolicitudSellarSelloBaremacion{
		Finalidad: finalidad, RepresentacionCanonica: representacion,
	}).MaterialCanonicoHMAC()
	if err != nil {
		return "", err
	}
	contenido := material.Revelar()
	defer borrarBytesE2E(contenido)
	mac := hmac.New(sha256.New, claveHMACBolsaPostgreSQLE2E[:])
	if _, err = mac.Write(contenido); err != nil {
		return "", err
	}
	return "hmac-sha256:e2e_postgresql_v3:" + hex.EncodeToString(mac.Sum(nil)), nil
}

func borrarBytesE2E(valores ...[]byte) {
	for _, valor := range valores {
		for indice := range valor {
			valor[indice] = 0
		}
	}
}

type estadoBolsaPostgreSQLE2E struct {
	Esquema                         string                                      `json:"esquema"`
	Etapa                           string                                      `json:"etapa"`
	Ancla                           time.Time                                   `json:"ancla"`
	InstanteUso                     time.Time                                   `json:"instante_uso"`
	HuellaTokenSHA256               string                                      `json:"huella_token_sha256"`
	VersionBase                     puertosbolsa.ReferenciaVersionBaremacion    `json:"version_base"`
	AgregadoObjetivo                dominiobolsa.BaremacionMerito               `json:"agregado_objetivo"`
	ManifiestoHistorico             puertosbolsa.ManifiestoProbatorioBaremacion `json:"manifiesto_historico"`
	Manifiesto                      puertosbolsa.ManifiestoProbatorioBaremacion `json:"manifiesto"`
	Trazabilidad                    puertosbolsa.TrazabilidadCambioBaremacion   `json:"trazabilidad"`
	AutorizacionReservaRef          string                                      `json:"autorizacion_reserva_ref"`
	AutorizacionPrevalidacionRef    string                                      `json:"autorizacion_prevalidacion_ref"`
	AutorizacionConfirmacionRef     string                                      `json:"autorizacion_confirmacion_ref"`
	ConfirmadaEn                    time.Time                                   `json:"confirmada_en"`
	HuellaSolicitudReservaHMAC      string                                      `json:"huella_solicitud_reserva_hmac"`
	HuellaSolicitudConfirmacionHMAC string                                      `json:"huella_solicitud_confirmacion_hmac"`
	VersionConfirmada               *puertosbolsa.ReferenciaVersionBaremacion   `json:"version_confirmada,omitempty"`
	AuditoriaRef                    string                                      `json:"auditoria_ref,omitempty"`
	HuellaAuditoriaSHA256           string                                      `json:"huella_auditoria_sha256,omitempty"`
	EventoOutboxRef                 string                                      `json:"evento_outbox_ref,omitempty"`
	HuellaEventoOutboxSHA256        string                                      `json:"huella_evento_outbox_sha256,omitempty"`
	ConfirmadaFiableEn              time.Time                                   `json:"confirmada_fiable_en,omitempty"`
}

func TestIntegracionBaremacionPostgreSQLV3Real(t *testing.T) {
	dsn := os.Getenv(variableDSNBolsaPostgreSQLE2E)
	dsnAdmin := os.Getenv(variableDSNAdminBolsaPostgreSQLE2E)
	fase := os.Getenv(variableFaseBolsaPostgreSQLE2E)
	rutaEstado := os.Getenv(variableEstadoBolsaPostgreSQLE2E)
	if dsn == "" && dsnAdmin == "" && fase == "" && rutaEstado == "" {
		t.Skipf(
			"E2E PostgreSQL V3 omitido: defina %s, %s, %s y %s o use deploy/postgresql/bolsa_baremacion/probar_integracion_v3.sh",
			variableDSNBolsaPostgreSQLE2E,
			variableDSNAdminBolsaPostgreSQLE2E,
			variableFaseBolsaPostgreSQLE2E,
			variableEstadoBolsaPostgreSQLE2E,
		)
	}
	if dsn == "" || dsnAdmin == "" || fase == "" || rutaEstado == "" {
		t.Fatal("configuracion E2E PostgreSQL V3 parcial: se requieren las cuatro variables")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("abrir pool ejecutor: %v", err)
	}
	defer pool.Close()
	poolAdmin, err := pgxpool.New(ctx, dsnAdmin)
	if err != nil {
		t.Fatalf("abrir pool administrativo: %v", err)
	}
	defer poolAdmin.Close()
	if err = pool.Ping(ctx); err != nil {
		t.Fatalf("conectar ejecutor: %v", err)
	}
	if err = poolAdmin.Ping(ctx); err != nil {
		t.Fatalf("conectar administrador: %v", err)
	}

	switch fase {
	case fasePrevalidarFalloBolsaPostgreSQLE2E:
		ejecutarFaseInicialBolsaPostgreSQLE2E(t, ctx, pool, poolAdmin, rutaEstado)
	case faseConfirmarBolsaPostgreSQLE2E:
		ejecutarFaseConfirmacionBolsaPostgreSQLE2E(t, ctx, pool, poolAdmin, rutaEstado)
	case faseRecuperarBolsaPostgreSQLE2E:
		ejecutarFaseRecuperacionBolsaPostgreSQLE2E(t, ctx, pool, poolAdmin, rutaEstado)
	default:
		t.Fatalf("fase E2E desconocida %q", fase)
	}
}

type entornoAutorizacionBolsaPostgreSQLE2E struct {
	admin    *pgxpool.Pool
	fuente   *postgresvec.AlmacenAutorizacion
	ancla    time.Time
	actor    dominiovec.ContextoActor
	vinculo  dominiovec.VinculoAutenticacionActorV1
	vinculoB puertosbolsa.VinculoAutenticacionBaremacion
}

func nuevoEntornoAutorizacionBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	ejecutor *pgxpool.Pool,
	admin *pgxpool.Pool,
	ancla time.Time,
	sembrar bool,
) entornoAutorizacionBolsaPostgreSQLE2E {
	t.Helper()
	ancla = ancla.UTC().Truncate(time.Microsecond)
	const sufijo = "e2epostgresqlv3real0123456789"
	cuenta := dominiovec.CuentaAutenticadaContextoActor{
		CuentaRef: "cta_" + sufijo,
		Metodo:    dominiovec.AuthMethodCertificate,
		Garantia:  dominiovec.AuthAssuranceHigh,
	}
	instantanea := dominiovec.InstantaneaContextoActor{
		VinculoRef:      "vca_" + sufijo,
		VinculoVersion:  1,
		CuentaRef:       cuenta.CuentaRef,
		PersonaRef:      "per_" + sufijo,
		PersonaVersion:  1,
		PerfilActivoRef: "prf_" + sufijo,
		PerfilVersion:   1,
		Estado:          dominiovec.EstadoVinculoContextoActorActivo,
		VigenteDesde:    ancla.Add(-10 * time.Minute),
		VigenteHasta:    ancla.Add(30 * time.Minute),
		Vinculos:        []dominiovec.VinculoReferenciaContextoActor{},
	}
	actor, err := dominiovec.NuevoContextoActor(cuenta, instantanea, ancla.Add(-2*time.Minute))
	if err != nil {
		t.Fatalf("crear ContextoActor E2E: %v", err)
	}
	autenticacion := dominiovec.AutenticacionRevalidadaV1{
		AutenticacionRef:             "aut_" + sufijo,
		AutenticacionHuellaSHA256:    huellaSHA256E2E("autenticacion autoritativa E2E PostgreSQL V3"),
		AsercionRef:                  "ase_" + sufijo,
		SesionRef:                    "ses_" + sufijo,
		ControlSesionRef:             "cse_" + sufijo,
		ControlSesionRevision:        1,
		ControlSesionHuellaSHA256:    huellaSHA256E2E("control de sesion E2E PostgreSQL V3 revision 1"),
		CuentaRef:                    cuenta.CuentaRef,
		CuentaOrdinariaRef:           cuenta.CuentaRef,
		Superficie:                   dominiovec.SuperficieAutenticacionInternaCorporativaV1,
		MetodoObservado:              cuenta.Metodo,
		GarantiaObservada:            cuenta.Garantia,
		PoliticaGarantiaRef:          "pga_" + sufijo,
		PoliticaGarantiaHuellaSHA256: huellaSHA256E2E("politica de garantia E2E PostgreSQL V3"),
		AutenticacionVerificadaEn:    ancla.Add(-5 * time.Minute),
		SesionEmitidaEn:              ancla.Add(-4 * time.Minute),
		SesionRevalidadaEn:           ancla.Add(-3 * time.Minute),
		SesionValidaHasta:            ancla.Add(30 * time.Minute),
	}
	vinculo, err := dominiovec.CrearVinculoAutenticacionActorV1(
		ctx,
		revalidadorActorBolsaPostgreSQLE2E{resultado: autenticacion},
		dominiovec.SolicitudRevalidacionAutenticacionActorV1{
			AutenticacionRef: autenticacion.AutenticacionRef,
			SesionRef:        autenticacion.SesionRef,
		},
		actor,
		ancla,
	)
	if err != nil {
		t.Fatalf("crear vinculo autenticacion-actor E2E: %v", err)
	}
	datos, err := vinculo.Datos()
	if err != nil {
		t.Fatalf("proyectar vinculo E2E: %v", err)
	}
	if sembrar {
		prepararFuenteAutorizacionRestringidaBolsaPostgreSQLE2E(t, ctx, ejecutor, admin)
	}
	almacen, err := postgresvec.NuevoAlmacenAutorizacion(ejecutor)
	if err != nil {
		t.Fatalf("crear almacen real de autorizacion: %v", err)
	}
	entorno := entornoAutorizacionBolsaPostgreSQLE2E{
		admin: admin, fuente: almacen, ancla: ancla, actor: actor, vinculo: vinculo,
		vinculoB: puertosbolsa.VinculoAutenticacionBaremacion{
			SujetoRef: sujetoRefBolsaPostgreSQLE2E,
			Metodo:    datos.MetodoObservado, Garantia: datos.GarantiaObservada,
			AutenticacionRef: datos.AutenticacionRef, SesionRef: datos.SesionRef,
			SesionEmitidaEn: datos.SesionEmitidaEn, SesionValidaHasta: datos.SesionValidaHasta,
			VinculoAutenticacionActor: vinculo,
		},
	}
	if sembrar {
		sembrarAutorizacionBolsaPostgreSQLE2E(t, ctx, admin, entorno, datos)
	}
	return entorno
}

func prepararFuenteAutorizacionRestringidaBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	ejecutor *pgxpool.Pool,
	admin *pgxpool.Pool,
) {
	t.Helper()
	var identidad string
	var superusuario, crearRol, crearBD, replicacion, omitirRLS bool
	err := ejecutor.QueryRow(ctx, `
		SELECT current_user, rolsuper, rolcreaterole, rolcreatedb,
		       rolreplication, rolbypassrls
		  FROM pg_catalog.pg_roles
		 WHERE rolname=current_user`,
	).Scan(&identidad, &superusuario, &crearRol, &crearBD, &replicacion, &omitirRLS)
	if err != nil || identidad == "" || superusuario || crearRol || crearBD || replicacion || omitirRLS {
		t.Fatalf(
			"el ejecutor E2E no es una identidad restringida sin BYPASSRLS: identidad=%q error=%v",
			identidad, err,
		)
	}
	consulta := "GRANT vec_autorizacion_fuente TO " + pgx.Identifier{identidad}.Sanitize()
	if _, err = admin.Exec(ctx, consulta); err != nil {
		t.Fatalf("sembrar pertenencia de solo lectura a la fuente PDP: %v", err)
	}
	var puedeLeer, puedeRegistrar bool
	err = ejecutor.QueryRow(ctx, `
		SELECT
		  has_function_privilege(current_user,
		    'vec_autorizacion.obtener_instantanea(text,text)', 'EXECUTE'),
		  has_function_privilege(current_user,
		    'vec_autorizacion.registrar_decision_si_vigente(jsonb)', 'EXECUTE')`,
	).Scan(&puedeLeer, &puedeRegistrar)
	if err != nil || !puedeLeer || puedeRegistrar {
		t.Fatalf(
			"ACL PDP del ejecutor E2E no conserva separacion fuente/registro: lectura=%t registro=%t error=%v",
			puedeLeer, puedeRegistrar, err,
		)
	}
}

func accionesDurablesBolsaPostgreSQLE2E() []puertosbolsa.AccionOperacionBaremacion {
	acciones := []puertosbolsa.AccionOperacionBaremacion{
		puertosbolsa.AccionReservarAltaBaremacion,
		puertosbolsa.AccionConfirmarAltaBaremacion,
		puertosbolsa.AccionReservarDecisionBaremacion,
		puertosbolsa.AccionPrevalidarArchivoProbatorioBaremacion,
		puertosbolsa.AccionConfirmarDecisionBaremacion,
		puertosbolsa.AccionConsultarBaremacionVigente,
		puertosbolsa.AccionConsultarVersionBaremacion,
		puertosbolsa.AccionConsultarEvidenciaTransaccionBaremacion,
	}
	sort.Slice(acciones, func(i, j int) bool { return acciones[i] < acciones[j] })
	return acciones
}

func configuracionRBACBolsaPostgreSQLE2E(
	ancla time.Time,
	actor dominiovec.ContextoActor,
) (dominiovec.VersionRol, dominiovec.ControlVigenciaVersionRol, dominiovec.AsignacionPerfil) {
	concesiones := make([]dominiovec.ConcesionRol, 0, len(accionesDurablesBolsaPostgreSQLE2E()))
	for _, accion := range accionesDurablesBolsaPostgreSQLE2E() {
		clase, _ := puertosbolsa.ClaseRecursoRequeridaOperacionBaremacion(accion)
		campos, _ := puertosbolsa.CamposRequeridosOperacionBaremacion(accion)
		concesiones = append(concesiones, dominiovec.ConcesionRol{
			Accion: string(accion), ModuloID: "bolsa", TipoRecurso: string(clase),
			Finalidades:      []string{finalidadBolsaPostgreSQLE2E},
			GarantiaMinima:   dominiovec.AuthAssuranceHigh,
			CamposPermitidos: campos, Obligaciones: []string{},
		})
	}
	publicada := ancla.Add(-6 * time.Minute)
	rol := dominiovec.VersionRol{
		RolID: "bolsa_e2e_postgresql_v3_real", Version: 1,
		Nombre: "Bolsa E2E PostgreSQL V3 real", Estado: dominiovec.EstadoVersionRolPublicada,
		Concesiones: concesiones, PublicadaPor: "seguridad:e2e:postgresql:v3", PublicadaEn: publicada,
	}
	control := dominiovec.ControlVigenciaVersionRol{
		VersionRolRef: rol.Referencia(), Revision: 1,
		Estado:         dominiovec.EstadoControlVigenciaVersionRolHabilitada,
		ActualizadoPor: "seguridad:e2e:postgresql:v3", ActualizadoEn: publicada,
	}
	asignacion := dominiovec.AsignacionPerfil{
		AsignacionID: "bolsa_e2e_postgresql_v3_real", Version: 1,
		PerfilActivoRef: actor.PerfilActivoRef, PrincipalID: actor.Principal.ID,
		VersionRolRef: rol.Referencia(), Estado: dominiovec.EstadoAsignacionPerfilActiva,
		Ambitos: []dominiovec.AmbitoPerfil{{
			Clave: "sujeto_ref", Valores: []string{sujetoRefBolsaPostgreSQLE2E},
		}},
		VigenteDesde: publicada, VigenteHasta: ancla.Add(30 * time.Minute),
		EmitidaPor: "rrhh:e2e:postgresql:v3", EmitidaEn: publicada,
	}
	return rol, control, asignacion
}

func sembrarAutorizacionBolsaPostgreSQLE2E(
	t *testing.T,
	ctx context.Context,
	admin *pgxpool.Pool,
	entorno entornoAutorizacionBolsaPostgreSQLE2E,
	datos dominiovec.DatosVinculoAutenticacionActorV1,
) {
	t.Helper()
	rol, control, asignacion := configuracionRBACBolsaPostgreSQLE2E(entorno.ancla, entorno.actor)
	if err := rol.Validar(); err != nil {
		t.Fatalf("rol E2E invalido: %v", err)
	}
	if err := control.Validar(); err != nil {
		t.Fatalf("control de rol E2E invalido: %v", err)
	}
	if err := asignacion.Validar(); err != nil {
		t.Fatalf("asignacion E2E invalida: %v", err)
	}
	documentoRol := JSONE2E(t, rol)
	documentoControl := JSONE2E(t, control)
	documentoAsignacion := JSONE2E(t, asignacion)
	huellaRol, _ := rol.HuellaSHA256()
	huellaControl, _ := control.HuellaSHA256()
	huellaAsignacion, _ := asignacion.HuellaSHA256()
	tx, err := admin.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		t.Fatalf("iniciar siembra E2E: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.version_rol
		(version_rol_ref, rol_id, version, huella_sha256, publicada_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6::jsonb)`,
		rol.Referencia(), rol.RolID, rol.Version, huellaRol, rol.PublicadaEn, documentoRol)
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol
		(version_rol_ref, revision, estado, huella_sha256, actualizado_en, documento)
		VALUES ($1,$2::numeric,$3,$4,$5,$6::jsonb)`,
		control.VersionRolRef, strconv.FormatUint(control.Revision, 10), control.Estado,
		huellaControl, control.ActualizadoEn, documentoControl)
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.control_vigencia_version_rol_actual
		(version_rol_ref, revision, actualizada_en, actualizada_por, acto_ref)
		VALUES ($1,$2::numeric,$3,$4,$5)`,
		control.VersionRolRef, strconv.FormatUint(control.Revision, 10), control.ActualizadoEn,
		"seguridad:e2e:postgresql:v3", "acto:control:rol:e2e:postgresql:v3")
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.asignacion_perfil
		(asignacion_ref, asignacion_id, version, perfil_activo_ref, principal_id,
		 version_rol_ref, huella_sha256, emitida_en, documento)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)`,
		asignacion.Referencia(), asignacion.AsignacionID, asignacion.Version,
		asignacion.PerfilActivoRef, asignacion.PrincipalID, asignacion.VersionRolRef,
		huellaAsignacion, asignacion.EmitidaEn, documentoAsignacion)
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.asignacion_perfil_actual
		(perfil_activo_ref, asignacion_ref, actualizada_en, actualizada_por, acto_ref)
		VALUES ($1,$2,$3,$4,$5)`, asignacion.PerfilActivoRef, asignacion.Referencia(),
		asignacion.EmitidaEn, "rrhh:e2e:postgresql:v3", "acto:asignacion:e2e:postgresql:v3")
	ejecutarE2E(t, ctx, tx, `
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
		datos.PoliticaGarantiaHuellaSHA256, datos.AutenticacionVerificadaEn, datos.SesionEmitidaEn)
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.control_sesion_v1
		(control_sesion_ref, revision, sesion_ref, estado, huella_sha256,
		 sesion_revalidada_en, sesion_valida_hasta)
		VALUES ($1,$2::numeric,$3,'activa',$4,$5,$6)`,
		datos.ControlSesionRef, strconv.FormatUint(datos.ControlSesionRevision, 10),
		datos.SesionRef, datos.ControlSesionHuellaSHA256, datos.SesionRevalidadaEn,
		datos.SesionValidaHasta)
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.control_sesion_actual_v1
		(sesion_ref, control_sesion_ref, revision, actualizada_en, acto_ref)
		VALUES ($1,$2,$3::numeric,$4,$5)`, datos.SesionRef, datos.ControlSesionRef,
		strconv.FormatUint(datos.ControlSesionRevision, 10), datos.SesionRevalidadaEn,
		"acto:control:sesion:e2e:postgresql:v3")
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.contexto_actor_v1
		(contexto_actor_ref, version, cuenta_ref, principal_id, perfil_activo_ref,
		 estado, huella_sha256, vigente_desde, vigente_hasta)
		VALUES ($1,$2::numeric,$3,$4,$5,'activo',$6,$7,$8)`,
		datos.ContextoActorRef, strconv.FormatUint(datos.ContextoActorVersion, 10),
		datos.CuentaRef, datos.PrincipalID, datos.PerfilActivoRef,
		datos.ContextoActorHuellaSHA256, entorno.actor.Instantanea.VigenteDesde,
		entorno.actor.Instantanea.VigenteHasta)
	ejecutarE2E(t, ctx, tx, `
		INSERT INTO vec_autorizacion.contexto_actor_actual_v1
		(cuenta_ref, perfil_activo_ref, contexto_actor_ref, version, actualizada_en, acto_ref)
		VALUES ($1,$2,$3,$4::numeric,$5,$6)`, datos.CuentaRef, datos.PerfilActivoRef,
		datos.ContextoActorRef, strconv.FormatUint(datos.ContextoActorVersion, 10),
		entorno.actor.ResueltoEn, "acto:contexto:actor:e2e:postgresql:v3")
	if err = tx.Commit(ctx); err != nil {
		t.Fatalf("confirmar siembra E2E: %v", err)
	}
}
