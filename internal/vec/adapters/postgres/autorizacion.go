// Package postgres contiene adaptadores duraderos del nucleo para PostgreSQL.
// No ejecuta migraciones ni abre conexiones: composicion y ciclo de vida del
// pool pertenecen al borde de infraestructura.
package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// AlmacenAutorizacion implementa la fuente coherente y el registro CAS de
// decisiones. La cuenta del pool no recibe permisos sobre tablas: solo puede
// ejecutar las funciones cerradas de su grupo tecnico NOLOGIN. La composicion
// debe crear una instancia y un pool distintos para fuente y registro; ninguna
// identidad de ejecucion hereda ambos grupos.
type AlmacenAutorizacion struct {
	pool iniciadorTransacciones
}

type iniciadorTransacciones interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// NuevoAlmacenAutorizacion no toma un DSN para evitar que el adaptador lo
// conserve o lo incluya accidentalmente en errores. El llamador crea y prueba
// el pool con su gestor de secretos y conserva su ciclo de vida.
func NuevoAlmacenAutorizacion(pool *pgxpool.Pool) (*AlmacenAutorizacion, error) {
	return nuevoAlmacenAutorizacion(pool)
}

func nuevoAlmacenAutorizacion(pool iniciadorTransacciones) (*AlmacenAutorizacion, error) {
	if valorNuloPostgreSQL(pool) {
		return nil, domain.ErrConfiguracionAccesoInvalida
	}
	return &AlmacenAutorizacion{pool: pool}, nil
}

func (a *AlmacenAutorizacion) ObtenerInstantaneaAutorizacion(
	ctx context.Context,
	principalID, perfilActivoRef string,
) (domain.InstantaneaAutorizacion, error) {
	if a == nil || valorNuloPostgreSQL(a.pool) {
		return domain.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return domain.InstantaneaAutorizacion{}, err
	}
	// Una entrada no canonica nunca se corrige ni se usa para explorar filas.
	// Se responde igual que ante un perfil ajeno o inexistente.
	if !identificadorPostgreSQLSeguro(principalID, 512) ||
		!identificadorPostgreSQLSeguro(perfilActivoRef, 512) {
		return domain.InstantaneaAutorizacion{}, ports.ErrAsignacionPerfilNoEncontrada
	}

	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return domain.InstantaneaAutorizacion{}, errorFuenteAutorizacion(ctx)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionAutorizacion(ctx, tx); err != nil {
		return domain.InstantaneaAutorizacion{}, errorFuenteAutorizacion(ctx)
	}

	var documentoAsignacion, documentoRol, documentoControl, documentosPoliticas []byte
	var revisionCatalogoTexto, huellaCatalogo string
	err = tx.QueryRow(ctx, `
		SELECT documento_asignacion, documento_rol, documento_control_rol,
		       revision_catalogo, huella_catalogo, documentos_politicas
		FROM vec_autorizacion.obtener_instantanea($1, $2)`,
		principalID, perfilActivoRef,
	).Scan(
		&documentoAsignacion, &documentoRol, &documentoControl,
		&revisionCatalogoTexto, &huellaCatalogo, &documentosPoliticas,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.InstantaneaAutorizacion{}, ports.ErrAsignacionPerfilNoEncontrada
	}
	if err != nil {
		return domain.InstantaneaAutorizacion{}, errorFuenteAutorizacion(ctx)
	}

	var asignacion domain.AsignacionPerfil
	var versionRol domain.VersionRol
	var controlRol domain.ControlVigenciaVersionRol
	var politicas []domain.PoliticaRestrictiva
	if decodificarDocumentoPostgreSQL(documentoAsignacion, &asignacion) != nil ||
		decodificarDocumentoPostgreSQL(documentoRol, &versionRol) != nil ||
		decodificarDocumentoPostgreSQL(documentoControl, &controlRol) != nil ||
		decodificarDocumentoPostgreSQL(documentosPoliticas, &politicas) != nil ||
		asignacion.PrincipalID != principalID || asignacion.PerfilActivoRef != perfilActivoRef {
		return domain.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	revisionCatalogo, err := parsearRevisionAutorizacion(revisionCatalogoTexto)
	if err != nil {
		return domain.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	instantanea := domain.InstantaneaAutorizacion{
		AsignacionPerfil:              asignacion,
		VersionRol:                    versionRol,
		ControlVigenciaVersionRol:     controlRol,
		Politicas:                     politicas,
		RevisionCatalogoPoliticas:     revisionCatalogo,
		CatalogoPoliticasHuellaSHA256: huellaCatalogo,
	}
	if err = instantanea.Validar(); err != nil {
		return domain.InstantaneaAutorizacion{}, ports.ErrFuenteAutorizacionNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.InstantaneaAutorizacion{}, errorFuenteAutorizacion(ctx)
	}
	return instantanea, nil
}

func (a *AlmacenAutorizacion) RegistrarDecisionSiInstantaneaVigente(
	ctx context.Context,
	decision domain.DecisionAutorizacion,
) error {
	if a == nil || valorNuloPostgreSQL(a.pool) {
		return ports.ErrRegistroDecisionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := decision.ValidarEvidenciaInstantanea(); err != nil || decision.TieneSolicitudLigadaV2() {
		if err == nil {
			err = domain.ErrDecisionAutorizacionInvalida
		}
		return err
	}
	// Esta tabla materializa exclusivamente concesiones ejecutables. Las
	// denegaciones requieren el registro de auditoria probatoria separado; no
	// se convierten en capacidades ni se simula que han quedado persistidas.
	if !decision.Concedida || decision.Codigo != "concedida" {
		return ports.ErrRegistroDecisionNoDisponible
	}
	documento, err := serializarDecisionPostgreSQL(decision)
	if err != nil {
		return ports.ErrRegistroDecisionNoDisponible
	}

	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return errorRegistroAutorizacion(ctx, err)
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionAutorizacion(ctx, tx); err != nil {
		return errorRegistroAutorizacion(ctx, err)
	}
	var registrada bool
	err = tx.QueryRow(ctx, `
		SELECT vec_autorizacion.registrar_decision_si_vigente($1::jsonb)`,
		documento,
	).Scan(&registrada)
	if err != nil {
		return errorRegistroAutorizacion(ctx, err)
	}
	if !registrada {
		return ports.ErrInstantaneaAutorizacionObsoleta
	}
	if err = tx.Commit(ctx); err != nil {
		return errorRegistroAutorizacion(ctx, err)
	}
	return nil
}

func serializarDecisionPostgreSQL(decision domain.DecisionAutorizacion) ([]byte, error) {
	documento, err := ports.RepresentacionCanonicaDecisionAutorizacionReforzadaV1(decision)
	if err != nil {
		return nil, err
	}
	var objeto map[string]json.RawMessage
	if err = json.Unmarshal(documento, &objeto); err != nil {
		return nil, err
	}
	if len(objeto) != 30 || string(objeto["esquema"]) != `"`+ports.EsquemaHuellaDecisionAutorizacionReforzadaV1+`"` {
		return nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}

	evaluadasRefs, evaluadasHuellas, err := materializarManifiestoDecisionPostgreSQL(
		objeto["politicas_evaluadas"],
	)
	if err != nil {
		return nil, err
	}
	aplicablesRefs, aplicablesHuellas, err := materializarManifiestoDecisionPostgreSQL(
		objeto["politicas_aplicables"],
	)
	if err != nil {
		return nil, err
	}
	delete(objeto, "esquema")
	delete(objeto, "politicas_evaluadas")
	delete(objeto, "politicas_aplicables")
	objeto["politicas_evaluadas_refs"] = evaluadasRefs
	objeto["politicas_evaluadas_huellas_sha256"] = evaluadasHuellas
	objeto["politicas_refs"] = aplicablesRefs
	objeto["politicas_huellas_sha256"] = aplicablesHuellas
	if len(objeto) != 31 {
		return nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return json.Marshal(objeto)
}

type entradaManifiestoDecisionPostgreSQL struct {
	Referencia   string `json:"referencia"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func materializarManifiestoDecisionPostgreSQL(
	canonico json.RawMessage,
) (json.RawMessage, json.RawMessage, error) {
	var entradas []entradaManifiestoDecisionPostgreSQL
	if len(canonico) == 0 || json.Unmarshal(canonico, &entradas) != nil || entradas == nil {
		return nil, nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	referencias := make([]string, 0, len(entradas))
	huellas := make(map[string]string, len(entradas))
	for _, entrada := range entradas {
		if _, duplicada := huellas[entrada.Referencia]; duplicada {
			return nil, nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
		}
		referencias = append(referencias, entrada.Referencia)
		huellas[entrada.Referencia] = entrada.HuellaSHA256
	}
	referenciasJSON, errReferencias := json.Marshal(referencias)
	huellasJSON, errHuellas := json.Marshal(huellas)
	if errReferencias != nil || errHuellas != nil {
		return nil, nil, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return referenciasJSON, huellasJSON, nil
}

func configurarTransaccionAutorizacion(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '8s', true),
		       set_config('idle_in_transaction_session_timeout', '10s', true)`)
	return err
}

func decodificarDocumentoPostgreSQL(documento []byte, destino any) error {
	if len(documento) == 0 || valorNuloPostgreSQL(destino) {
		return ports.ErrFuenteAutorizacionNoDisponible
	}
	decodificador := json.NewDecoder(bytes.NewReader(documento))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return ports.ErrFuenteAutorizacionNoDisponible
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return ports.ErrFuenteAutorizacionNoDisponible
	}
	return nil
}

func parsearRevisionAutorizacion(valor string) (uint64, error) {
	// El parser local evita que una revision fuera del contrato propague un
	// mensaje del controlador o una normalizacion permisiva.
	var revision uint64
	if valor == "" || valor != strings.TrimSpace(valor) {
		return 0, ports.ErrFuenteAutorizacionNoDisponible
	}
	for _, digito := range valor {
		if digito < '0' || digito > '9' {
			return 0, ports.ErrFuenteAutorizacionNoDisponible
		}
		if revision > (^uint64(0)-uint64(digito-'0'))/10 {
			return 0, ports.ErrFuenteAutorizacionNoDisponible
		}
		revision = revision*10 + uint64(digito-'0')
	}
	if revision == 0 {
		return 0, ports.ErrFuenteAutorizacionNoDisponible
	}
	return revision, nil
}

func errorFuenteAutorizacion(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ports.ErrFuenteAutorizacionNoDisponible
}

func errorRegistroAutorizacion(ctx context.Context, err error) error {
	if contextoErr := ctx.Err(); contextoErr != nil {
		return contextoErr
	}
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) {
		switch errorPG.Code {
		case "23505":
			return ports.ErrVersionAutorizacionYaExiste
		case "40001", "40P01", "55P03":
			return ports.ErrInstantaneaAutorizacionObsoleta
		}
	}
	return ports.ErrRegistroDecisionNoDisponible
}

func revertirTransaccionPostgreSQL(tx pgx.Tx) {
	if tx == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	_ = tx.Rollback(ctx)
}

func identificadorPostgreSQLSeguro(valor string, maximo int) bool {
	if maximo < 1 || valor == "" || valor != strings.TrimSpace(valor) ||
		len(valor) > maximo || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return true
}

func valorNuloPostgreSQL(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

var _ ports.FuenteAutorizacion = (*AlmacenAutorizacion)(nil)
var _ ports.RegistroDecisionesAutorizacion = (*AlmacenAutorizacion)(nil)
