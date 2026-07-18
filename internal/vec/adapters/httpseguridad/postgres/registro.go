package postgres

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

const (
	consultaRegistrarSesion = `
		SELECT autenticacion_ref, asercion_ref, sesion_ref,
		       control_sesion_ref, control_sesion_revision_texto,
		       control_sesion_estado, control_sesion_huella_sha256,
		       cuenta_ref, cuenta_ordinaria_ref,
		       sesion_revalidada_en, sesion_valida_hasta
		FROM vec_identidad_sesiones_v1.registrar_sesion_v1(
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19,$20
		)`
	consultaReconciliarSesion = `
		SELECT autenticacion_ref, asercion_ref, sesion_ref,
		       control_sesion_ref, control_sesion_revision_texto,
		       control_sesion_estado, control_sesion_huella_sha256,
		       cuenta_ref, cuenta_ordinaria_ref,
		       sesion_revalidada_en, sesion_valida_hasta
		FROM vec_identidad_sesiones_v1.reconciliar_registro_sesion_v1(
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19,$20
		)`
	consultaRevalidarSesion = `
		SELECT vec_identidad_sesiones_v1.revalidar_sesion_y_cuentas_v1(
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19,$20
		)`
)

type iniciadorTransacciones interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// RegistroSesionesPostgreSQL usa pools con LOGIN distintos para mutacion y
// revalidacion. La composicion conserva su ciclo de vida y sus secretos.
type RegistroSesionesPostgreSQL struct {
	registro         iniciadorTransacciones
	revalidacion     iniciadorTransacciones
	seudonimizador   SeudonimizadorAlta
	espacioIdentidad string
	dominioHMACRef   string
	aleatorio        io.Reader
}

// NuevoRegistroSesionesPostgreSQL no acepta un DSN. Acredita contra PostgreSQL
// que cada pool usa un LOGIN distinto, sin SET ROLE y ligado exclusivamente a
// la capacidad tecnica correspondiente.
func NuevoRegistroSesionesPostgreSQL(
	ctx context.Context,
	poolRegistro, poolRevalidacion *pgxpool.Pool,
	seudonimizador SeudonimizadorAlta,
	espacioIdentidad string,
	dominioHMACRef string,
) (*RegistroSesionesPostgreSQL, error) {
	if ctx == nil || poolRegistro == nil || poolRevalidacion == nil ||
		poolRegistro == poolRevalidacion {
		return nil, httpseguridad.ErrRegistroSesionesAusente
	}
	usuarioRegistro, err := acreditarCapacidadPool(ctx, poolRegistro, capacidadRegistrar)
	if err != nil {
		return nil, err
	}
	usuarioRevalidacion, err := acreditarCapacidadPool(
		ctx,
		poolRevalidacion,
		capacidadRevalidar,
	)
	if err != nil || usuarioRegistro == usuarioRevalidacion {
		return nil, httpseguridad.ErrRegistroSesionesAusente
	}
	return nuevoRegistroSesionesPostgreSQL(
		poolRegistro, poolRevalidacion, seudonimizador,
		espacioIdentidad, dominioHMACRef, rand.Reader,
	)
}

func nuevoRegistroSesionesPostgreSQL(
	registro, revalidacion iniciadorTransacciones,
	seudonimizador SeudonimizadorAlta,
	espacioIdentidad string,
	dominioHMACRef string,
	aleatorio io.Reader,
) (*RegistroSesionesPostgreSQL, error) {
	if valorNulo(registro) || valorNulo(revalidacion) || valorNulo(seudonimizador) ||
		valorNulo(aleatorio) || mismaInstancia(registro, revalidacion) ||
		!espacioIdentidadValido(espacioIdentidad) ||
		!referenciaTecnicaValida(dominioHMACRef, "idh_") {
		return nil, httpseguridad.ErrRegistroSesionesAusente
	}
	return &RegistroSesionesPostgreSQL{
		registro: registro, revalidacion: revalidacion,
		seudonimizador: seudonimizador, espacioIdentidad: espacioIdentidad,
		dominioHMACRef: dominioHMACRef,
		aleatorio:      aleatorio,
	}, nil
}

func (r *RegistroSesionesPostgreSQL) ConsumirAsercionYRegistrar(
	ctx context.Context,
	alta httpseguridad.AltaSesionAtomica,
) (httpseguridad.ConfirmacionAltaSesion, error) {
	if r == nil || valorNulo(ctx) || alta.Validar() != nil ||
		valorNulo(r.registro) || valorNulo(r.seudonimizador) || valorNulo(r.aleatorio) {
		return httpseguridad.ConfirmacionAltaSesion{}, httpseguridad.ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, err
	}
	if alta.EspacioIdentidad != r.espacioIdentidad {
		return httpseguridad.ConfirmacionAltaSesion{}, httpseguridad.ErrSesionNoValida
	}
	seudonimos, err := r.seudonimizador.SeudonimizarAlta(ctx, IdentificadoresAlta{
		EspacioIdentidad: alta.EspacioIdentidad,
		AsercionID:       alta.AsercionID, SesionID: alta.SesionID, SujetoID: alta.SujetoID,
		CuentaID: alta.CuentaID, CuentaOrdinariaID: alta.CuentaOrdinariaID,
	})
	if err != nil || !seudonimos.valida(
		r.espacioIdentidad, r.dominioHMACRef, alta.CuentaPrivilegiada,
	) {
		return httpseguridad.ConfirmacionAltaSesion{}, errorSesionSaneado(ctx)
	}
	operacionRef, err := nuevaReferenciaOperacion(r.aleatorio)
	if err != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, errorSesionSaneado(ctx)
	}
	argumentos := argumentosAlta(operacionRef, seudonimos, alta)
	respuesta, err := r.ejecutarAlta(ctx, consultaRegistrarSesion, argumentos)
	if err != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, err
	}
	confirmacion := respuesta.confirmacion(alta)
	if err = confirmacion.ValidarPara(alta); err != nil {
		return httpseguridad.ConfirmacionAltaSesion{}, httpseguridad.ErrSesionNoValida
	}
	return confirmacion, nil
}

func (r *RegistroSesionesPostgreSQL) ComprobarSesionYCuentaActivas(
	ctx context.Context,
	consulta httpseguridad.ConsultaSesionActiva,
) error {
	if r == nil || valorNulo(ctx) || valorNulo(r.revalidacion) || consulta.Validar() != nil {
		return httpseguridad.ErrSesionNoValida
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := r.revalidacion.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return errorSesionSaneado(ctx)
	}
	defer revertir(tx)
	if err = prepararTransaccion(ctx, tx); err != nil {
		return errorSesionSaneado(ctx)
	}
	var activa bool
	err = tx.QueryRow(ctx, consultaRevalidarSesion, argumentosConsulta(consulta)...).Scan(&activa)
	if err != nil || !activa {
		return errorSesionSaneado(ctx)
	}
	if err = tx.Commit(ctx); err != nil {
		return errorSesionSaneado(ctx)
	}
	return nil
}

type respuestaAlta struct {
	autenticacionRef, asercionRef, sesionRef, controlRef string
	revisionTexto, estado, huellaControl                 string
	cuentaRef, cuentaOrdinariaRef                        string
	revalidadaEn, validaHasta                            time.Time
}

func (r *RegistroSesionesPostgreSQL) ejecutarAlta(
	ctx context.Context,
	consulta string,
	argumentos []any,
) (respuestaAlta, error) {
	tx, err := r.registro.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	defer revertir(tx)
	if err = prepararTransaccion(ctx, tx); err != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	respuesta, err := consultarRespuestaAlta(ctx, tx, consulta, argumentos)
	if err != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	if err = tx.Commit(ctx); err == nil {
		return respuesta, nil
	}
	// Un COMMIT incierto no se reintenta. Se consulta exclusivamente la
	// operacion CSPRNG de esta invocacion y se cotejan todos sus campos.
	ctxReconciliacion, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	txReconciliacion, errReconciliacion := r.registro.BeginTx(
		ctxReconciliacion, opcionesTransaccion(),
	)
	if errReconciliacion != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	defer revertir(txReconciliacion)
	if prepararTransaccion(ctxReconciliacion, txReconciliacion) != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	reconciliada, errReconciliacion := consultarRespuestaAlta(
		ctxReconciliacion, txReconciliacion, consultaReconciliarSesion, argumentos,
	)
	if errReconciliacion != nil || !respuestasIguales(respuesta, reconciliada) ||
		txReconciliacion.Commit(ctxReconciliacion) != nil {
		return respuestaAlta{}, errorSesionSaneado(ctx)
	}
	return reconciliada, nil
}

func consultarRespuestaAlta(
	ctx context.Context,
	tx pgx.Tx,
	consulta string,
	argumentos []any,
) (respuestaAlta, error) {
	var respuesta respuestaAlta
	err := tx.QueryRow(ctx, consulta, argumentos...).Scan(
		&respuesta.autenticacionRef, &respuesta.asercionRef, &respuesta.sesionRef,
		&respuesta.controlRef, &respuesta.revisionTexto, &respuesta.estado,
		&respuesta.huellaControl, &respuesta.cuentaRef, &respuesta.cuentaOrdinariaRef,
		&respuesta.revalidadaEn, &respuesta.validaHasta,
	)
	if err == nil {
		respuesta.revalidadaEn = respuesta.revalidadaEn.UTC().Truncate(time.Microsecond)
		respuesta.validaHasta = respuesta.validaHasta.UTC().Truncate(time.Microsecond)
	}
	return respuesta, err
}

func (r respuestaAlta) confirmacion(alta httpseguridad.AltaSesionAtomica) httpseguridad.ConfirmacionAltaSesion {
	confirmacion := r.confirmacionSinEco()
	confirmacion.AltaConfirmada = alta
	return confirmacion
}

func (r respuestaAlta) confirmacionSinEco() httpseguridad.ConfirmacionAltaSesion {
	revision, _ := strconv.ParseUint(r.revisionTexto, 10, 64)
	return httpseguridad.ConfirmacionAltaSesion{
		AutenticacionRef: r.autenticacionRef, AsercionRef: r.asercionRef,
		SesionRef: r.sesionRef, ControlSesionRef: r.controlRef,
		ControlSesionRevision:     revision,
		ControlSesionEstado:       httpseguridad.EstadoControlSesion(r.estado),
		ControlSesionHuellaSHA256: r.huellaControl,
		CuentaRef:                 r.cuentaRef, CuentaOrdinariaRef: r.cuentaOrdinariaRef,
		SesionRevalidadaEn: r.revalidadaEn, SesionValidaHasta: r.validaHasta,
	}
}

func respuestasIguales(a, b respuestaAlta) bool {
	return a.autenticacionRef == b.autenticacionRef && a.asercionRef == b.asercionRef &&
		a.sesionRef == b.sesionRef && a.controlRef == b.controlRef &&
		a.revisionTexto == b.revisionTexto && a.estado == b.estado &&
		a.huellaControl == b.huellaControl && a.cuentaRef == b.cuentaRef &&
		a.cuentaOrdinariaRef == b.cuentaOrdinariaRef &&
		a.revalidadaEn.Equal(b.revalidadaEn) && a.validaHasta.Equal(b.validaHasta)
}

func argumentosAlta(
	operacionRef string,
	s SeudonimosAlta,
	a httpseguridad.AltaSesionAtomica,
) []any {
	var ordinaria any
	if a.CuentaPrivilegiada {
		ordinaria = s.CuentaOrdinariaIDHMAC[:]
	}
	return []any{
		operacionRef, s.Esquema, s.DominioRef, s.ClaveID, int64(s.ClaveVersion),
		s.AsercionIDHMAC[:], s.SesionIDHMAC[:], s.SujetoIDHMAC[:], s.CuentaIDHMAC[:], ordinaria,
		a.CuentaPrivilegiada, string(a.Superficie), string(a.MetodoObservado),
		string(a.GarantiaObservada), a.AutenticacionHuellaSHA256,
		a.AutenticacionVerificadaEn, a.SesionEmitidaEn, a.AsercionExpiraEn,
		a.PoliticaGarantiaRef, a.PoliticaGarantiaHuellaSHA256,
	}
}

func argumentosConsulta(c httpseguridad.ConsultaSesionActiva) []any {
	return []any{
		c.AutenticacionRef, c.AutenticacionHuellaSHA256, c.AsercionRef, c.SesionRef,
		c.CuentaRef, c.CuentaOrdinariaRef, c.CuentaPrivilegiada, string(c.Superficie),
		string(c.MetodoObservado), string(c.GarantiaObservada), c.PoliticaGarantiaRef,
		c.PoliticaGarantiaHuellaSHA256, c.AutenticacionVerificadaEn, c.SesionEmitidaEn,
		c.ControlSesionRef, strconv.FormatUint(c.ControlSesionRevision, 10),
		string(c.ControlSesionEstado), c.ControlSesionHuellaSHA256,
		c.SesionRevalidadaEn, c.SesionValidaHasta,
	}
}

func opcionesTransaccion() pgx.TxOptions {
	return pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite}
}

func prepararTransaccion(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '8s', true),
		       set_config('idle_in_transaction_session_timeout', '10s', true)`)
	return err
}

func nuevaReferenciaOperacion(aleatorio io.Reader) (string, error) {
	material := make([]byte, 18)
	if _, err := io.ReadFull(aleatorio, material); err != nil {
		return "", err
	}
	return "opr_" + base64.RawURLEncoding.EncodeToString(material), nil
}

func errorSesionSaneado(ctx context.Context) error {
	if !valorNulo(ctx) {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return httpseguridad.ErrSesionNoValida
}

func revertir(tx pgx.Tx) {
	if tx == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	_ = tx.Rollback(ctx)
}

func valorNulo(valor any) bool {
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

func mismaInstancia(a, b any) bool {
	if valorNulo(a) || valorNulo(b) {
		return false
	}
	va, vb := reflect.ValueOf(a), reflect.ValueOf(b)
	return va.Type() == vb.Type() && va.Kind() == reflect.Pointer && va.Pointer() == vb.Pointer()
}

var _ httpseguridad.RegistroSesiones = (*RegistroSesionesPostgreSQL)(nil)
