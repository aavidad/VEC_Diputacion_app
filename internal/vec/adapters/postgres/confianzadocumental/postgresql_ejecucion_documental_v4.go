package confianzadocumental

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type iniciadorTransaccionesEjecucionDocumentalV4 interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// repositorioPostgreSQLEjecucionDocumentalV4 vive dentro del conector y recibe
// la prueba privada directamente. El nucleo solo observa el puerto neutral; no
// se publica el DTO transaccional ni una fabrica verificadora inyectable.
type repositorioPostgreSQLEjecucionDocumentalV4 struct {
	pool iniciadorTransaccionesEjecucionDocumentalV4
}

func nuevoRepositorioPostgreSQLEjecucionDocumentalV4(
	pool iniciadorTransaccionesEjecucionDocumentalV4,
) (*repositorioPostgreSQLEjecucionDocumentalV4, error) {
	if interfazPostgreSQLDocumentalNula(pool) {
		return nil, ErrConfiguracionConfianzaDocumentalInvalida
	}
	return &repositorioPostgreSQLEjecucionDocumentalV4{pool: pool}, nil
}

// configurarRepositorioPostgreSQLEjecucionDocumentalV4 solo se invoca desde
// el ensamblado de confianza (y pruebas del mismo paquete). No existe un setter
// exportado que permita cambiar el repositorio durante una peticion.
func (s *Servicio) configurarRepositorioPostgreSQLEjecucionDocumentalV4(
	pool *pgxpool.Pool,
) error {
	if s == nil || s.repositorioEjecucionV4 != nil {
		return ErrConfiguracionConfianzaDocumentalInvalida
	}
	repositorio, err := nuevoRepositorioPostgreSQLEjecucionDocumentalV4(pool)
	if err != nil {
		return err
	}
	s.repositorioEjecucionV4 = repositorio
	return nil
}

func (r *repositorioPostgreSQLEjecucionDocumentalV4) ejecutarPlanAtestado(
	ctx context.Context,
	solicitud solicitudEjecucionDocumentalAtestadaV4,
) (ResultadoEjecucionPlanDocumentalV4, error) {
	// La ruta local antigua carece de la capacidad emitida por el proceso
	// aislado. Permanece solo para satisfacer el contrato privado de pruebas y
	// falla cerrada; nunca se transforma una prueba Go en autoridad SQL.
	return ResultadoEjecucionPlanDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
}

func (r *repositorioPostgreSQLEjecucionDocumentalV4) ejecutarArtefactosAtestados(
	ctx context.Context,
	artefactos artefactosEjecucionDocumentalV4,
) (ResultadoEjecucionPlanDocumentalV4, error) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	if r == nil || interfazPostgreSQLDocumentalNula(r.pool) || ctx == nil ||
		artefactos.validarEn(instante) != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorCapacidadDocumentalV4(nil)
	}
	if err := ctx.Err(); err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, denegarEjecucionDocumentalV4(err)
	}
	// La transaccion solo comienza despues de validar estructura, limites,
	// caducidad y enlaces de huella. PostgreSQL verifica ademas el HMAC con un
	// secreto que esta identidad ejecutora no puede leer.
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorPostgreSQLEjecucionDocumentalV4(ctx, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err = configurarTransaccionEjecucionDocumentalV4(ctx, tx); err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorPostgreSQLEjecucionDocumentalV4(ctx, err)
	}

	var estadoResultado string
	var resultado ResultadoEjecucionPlanDocumentalV4
	err = tx.QueryRow(ctx, `
		SELECT resultado, orden_ref, estado_orden, auditoria_ref,
		       evento_outbox_ref, registrada_en
		  FROM vec_ejecucion_documental_v4.ejecutar_plan_atestado(
		       $1::bytea, $2::bytea, $3::bytea, $4::bytea,
		       $5::bytea, $6::bytea, $7::bytea, $8::jsonb
		  )`,
		artefactos.metadatos, artefactos.payload, artefactos.sobre,
		artefactos.evidencia, artefactos.preimagen, artefactos.decisionCanonica,
		artefactos.efecto, string(artefactos.capacidad),
	).Scan(
		&estadoResultado, &resultado.OrdenRef, &resultado.Estado,
		&resultado.AuditoriaRef, &resultado.EventoOutboxRef,
		&resultado.RegistradaEn,
	)
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorPostgreSQLEjecucionDocumentalV4(ctx, err)
	}
	// pgx conserva por defecto la localizacion del proceso al decodificar un
	// timestamptz. PostgreSQL ya fijo la precision a seis decimales y el
	// instante es absoluto; se normaliza a UTC antes de aplicar el contrato
	// canonico para que la zona del host no convierta un efecto valido en un
	// falso negativo y provoque su ROLLBACK.
	resultado.RegistradaEn = resultado.RegistradaEn.UTC().Truncate(time.Microsecond)
	if estadoResultado != "ejecutada" ||
		resultado.validarContraArtefactos(artefactos) != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorCapacidadDocumentalV4(nil)
	}
	if err = tx.Commit(ctx); err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorPostgreSQLEjecucionDocumentalV4(ctx, err)
	}
	return resultado, nil
}

func (r ResultadoEjecucionPlanDocumentalV4) validarContraArtefactos(
	artefactos artefactosEjecucionDocumentalV4,
) error {
	var efecto efectoEjecucionDocumentalV4PostgreSQL
	if decodificarJSONExactoDocumentalV4(artefactos.efecto, &efecto) != nil ||
		r.OrdenRef != efecto.OrdenRef || r.Estado != estadoOrdenGeneracionPendienteV4 ||
		r.AuditoriaRef != efecto.AuditoriaRef ||
		r.EventoOutboxRef != efecto.EventoOutboxRef ||
		!instanteCanonicoDocumental(r.RegistradaEn) ||
		r.RegistradaEn.Before(efecto.SolicitadaEn) {
		return errCapacidadEjecucionDocumentalV4Invalida
	}
	return nil
}

type aplicacionEjecucionDocumentalV4PostgreSQL struct {
	Esquema                         string    `json:"esquema"`
	DecisionRef                     string    `json:"decision_ref"`
	HuellaPlanSHA256                string    `json:"huella_plan_sha256"`
	EfectoRef                       string    `json:"efecto_ref"`
	EsquemaHuellaDecision           string    `json:"esquema_huella_decision"`
	HuellaDecisionSHA256            string    `json:"huella_decision_sha256"`
	PerfilActivoRef                 string    `json:"perfil_activo_ref"`
	ContextoActorHuellaSHA256       string    `json:"contexto_actor_huella_sha256"`
	Accion                          string    `json:"accion"`
	RecursoRef                      string    `json:"recurso_ref"`
	ModuloID                        string    `json:"modulo_id"`
	TipoRecurso                     string    `json:"tipo_recurso"`
	HuellaRecursoSHA256             string    `json:"huella_recurso_sha256"`
	HuellaAmbitosSHA256             string    `json:"huella_ambitos_sha256"`
	Finalidad                       string    `json:"finalidad"`
	CorrelacionRef                  string    `json:"correlacion_ref"`
	HuellaCamposPermitidosSHA256    string    `json:"huella_campos_permitidos_sha256"`
	HuellaObligacionesSHA256        string    `json:"huella_obligaciones_sha256"`
	HuellaCumplimientosSHA256       string    `json:"huella_cumplimientos_sha256"`
	VerificadaEn                    time.Time `json:"verificada_en"`
	VinculadaEn                     time.Time `json:"vinculada_en"`
	SolicitadaEn                    time.Time `json:"solicitada_en"`
	ValidaHasta                     time.Time `json:"valida_hasta"`
	HuellaSolicitudVinculadaSHA256  string    `json:"huella_solicitud_vinculada_sha256"`
	HuellaSolicitudAplicacionSHA256 string    `json:"huella_solicitud_aplicacion_sha256"`
}

type metadatosAtestacionEjecucionDocumentalV4PostgreSQL struct {
	Aplicacion                aplicacionEjecucionDocumentalV4PostgreSQL `json:"aplicacion"`
	HuellaPreimagenSHA256     string                                    `json:"huella_preimagen_sha256"`
	FormatoVECADVersion       uint16                                    `json:"formato_vec_ad_version"`
	Suite                     string                                    `json:"suite"`
	ClaveID                   string                                    `json:"clave_id"`
	AudienciaDespliegue       string                                    `json:"audiencia_despliegue"`
	AlgoritmoCOSE             string                                    `json:"algoritmo_cose"`
	AudienciaCOSE             string                                    `json:"audiencia_cose"`
	EstadoConfianza           string                                    `json:"estado_confianza"`
	HuellaClaveSHA256         string                                    `json:"huella_clave_sha256"`
	HuellaPayloadSHA256       string                                    `json:"huella_payload_sha256"`
	HuellaSobreSHA256         string                                    `json:"huella_sobre_sha256"`
	VerificadaEn              time.Time                                 `json:"verificada_en"`
	RaizValidaDesde           time.Time                                 `json:"raiz_valida_desde"`
	RaizValidaHasta           time.Time                                 `json:"raiz_valida_hasta"`
	RevisionConfianza         string                                    `json:"revision_confianza"`
	HuellaConfiguracionSHA256 string                                    `json:"huella_configuracion_sha256"`
	ConfiguracionPublicadaEn  time.Time                                 `json:"configuracion_publicada_en"`
	ConfiguracionExpiraEn     time.Time                                 `json:"configuracion_expira_en"`
	HuellaEvidenciaSHA256     string                                    `json:"huella_evidencia_sha256"`
}

type efectoEjecucionDocumentalV4PostgreSQL struct {
	Esquema           string    `json:"esquema"`
	OrdenRef          string    `json:"orden_ref"`
	Estado            string    `json:"estado"`
	DecisionRef       string    `json:"decision_ref"`
	EfectoRef         string    `json:"efecto_ref"`
	HuellaPlanSHA256  string    `json:"huella_plan_sha256"`
	HuellaDecision    string    `json:"huella_decision_sha256"`
	HuellaAplicacion  string    `json:"huella_aplicacion_sha256"`
	HuellaOrdenSHA256 string    `json:"huella_orden_sha256"`
	AuditoriaRef      string    `json:"auditoria_ref"`
	EventoOutboxRef   string    `json:"evento_outbox_ref"`
	CorrelacionRef    string    `json:"correlacion_ref"`
	SolicitadaEn      time.Time `json:"solicitada_en"`
}

func serializarEjecucionDocumentalAtestadaV4(
	s solicitudEjecucionDocumentalAtestadaV4,
) (metadatos, preimagen, efecto []byte, err error) {
	if s.validar() != nil {
		return nil, nil, nil, errorAutoridadInternaEjecucionDocumentalV4()
	}
	d := clonarDatosRegistroAtestacionPDPDocumentalV4(*s.prueba.datos)
	preimagen, err = d.PreimagenRecurso.SerializacionCanonicaParaPersistencia()
	if err != nil {
		return nil, nil, nil, errorAutoridadInternaEjecucionDocumentalV4()
	}
	huellaPreimagen, err := d.PreimagenRecurso.HuellaSHA256()
	if err != nil {
		return nil, nil, nil, errorAutoridadInternaEjecucionDocumentalV4()
	}
	p := d.ProyeccionAplicacion
	aplicacion := aplicacionEjecucionDocumentalV4PostgreSQL{
		Esquema: p.Esquema, DecisionRef: p.Clave.DecisionRef,
		HuellaPlanSHA256: p.Clave.HuellaPlanSHA256, EfectoRef: p.Clave.EfectoRef,
		EsquemaHuellaDecision: p.EsquemaHuellaDecision, HuellaDecisionSHA256: p.HuellaDecisionSHA256,
		PerfilActivoRef: p.PerfilActivoRef, ContextoActorHuellaSHA256: p.ContextoActorHuellaSHA256,
		Accion: p.Accion, RecursoRef: p.RecursoRef, ModuloID: p.ModuloID,
		TipoRecurso: p.TipoRecurso, HuellaRecursoSHA256: p.HuellaRecursoSHA256,
		HuellaAmbitosSHA256: p.HuellaAmbitosSHA256, Finalidad: p.Finalidad,
		CorrelacionRef:               p.CorrelacionRef,
		HuellaCamposPermitidosSHA256: p.HuellaCamposPermitidosSHA256,
		HuellaObligacionesSHA256:     p.HuellaObligacionesSHA256,
		HuellaCumplimientosSHA256:    p.HuellaCumplimientosSHA256,
		VerificadaEn:                 p.VerificadaEn, VinculadaEn: p.VinculadaEn,
		SolicitadaEn: p.SolicitadaEn, ValidaHasta: p.ValidaHasta,
		HuellaSolicitudVinculadaSHA256:  p.HuellaSolicitudVinculadaSHA256,
		HuellaSolicitudAplicacionSHA256: p.HuellaSolicitudAplicacionSHA256,
	}
	metadatos, err = json.Marshal(metadatosAtestacionEjecucionDocumentalV4PostgreSQL{
		Aplicacion: aplicacion, HuellaPreimagenSHA256: huellaPreimagen,
		FormatoVECADVersion: d.FormatoVECADVersion, Suite: d.Suite,
		ClaveID: d.ClaveID, AudienciaDespliegue: d.AudienciaDespliegue,
		AlgoritmoCOSE: d.AlgoritmoCOSE, AudienciaCOSE: d.AudienciaCOSE,
		EstadoConfianza: d.EstadoConfianza, HuellaClaveSHA256: d.HuellaClaveSHA256,
		HuellaPayloadSHA256: d.HuellaPayloadSHA256, HuellaSobreSHA256: d.HuellaSobreSHA256,
		VerificadaEn: d.VerificadaEn, RaizValidaDesde: d.RaizValidaDesde,
		RaizValidaHasta: d.RaizValidaHasta, RevisionConfianza: d.RevisionConfianza,
		HuellaConfiguracionSHA256: d.HuellaConfiguracion,
		ConfiguracionPublicadaEn:  d.ConfiguracionPublicadaEn,
		ConfiguracionExpiraEn:     d.ConfiguracionExpiraEn,
		HuellaEvidenciaSHA256:     d.HuellaEvidenciaSHA256,
	})
	if err != nil {
		return nil, nil, nil, errorAutoridadInternaEjecucionDocumentalV4()
	}
	o := s.orden
	efecto, err = json.Marshal(efectoEjecucionDocumentalV4PostgreSQL{
		Esquema: o.Esquema, OrdenRef: o.OrdenRef, Estado: o.Estado,
		DecisionRef: o.DecisionRef, EfectoRef: o.EfectoRef,
		HuellaPlanSHA256: o.HuellaPlanSHA256, HuellaDecision: o.HuellaDecision,
		HuellaAplicacion: o.HuellaAplicacion, HuellaOrdenSHA256: o.HuellaOrdenSHA256,
		AuditoriaRef: o.AuditoriaRef, EventoOutboxRef: o.EventoOutboxRef,
		CorrelacionRef: o.CorrelacionRef, SolicitadaEn: o.SolicitadaEn,
	})
	if err != nil {
		return nil, nil, nil, errorAutoridadInternaEjecucionDocumentalV4()
	}
	return metadatos, preimagen, efecto, nil
}

func configurarTransaccionEjecucionDocumentalV4(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '8s', true),
		       set_config('idle_in_transaction_session_timeout', '10s', true)`)
	return err
}

func errorPostgreSQLEjecucionDocumentalV4(ctx context.Context, err error) error {
	if ctx != nil {
		if contextoErr := ctx.Err(); contextoErr != nil {
			return denegarEjecucionDocumentalV4(contextoErr)
		}
	}
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) {
		switch errorPG.Code {
		case "40001", "40P01", "55P03":
			return errorAutoridadInternaEjecucionDocumentalV4()
		}
	}
	return errorAutoridadInternaEjecucionDocumentalV4()
}

func interfazPostgreSQLDocumentalNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

var _ repositorioEjecucionDocumentalV4 = (*repositorioPostgreSQLEjecucionDocumentalV4)(nil)
