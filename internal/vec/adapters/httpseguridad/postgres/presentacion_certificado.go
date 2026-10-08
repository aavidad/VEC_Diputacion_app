package postgres

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

const (
	grupoPresentadorCertificado = "vec_identidad_sesiones_v1_presentador"
	firmaInicioPresentacion     = "vec_identidad_sesiones_v1.iniciar_y_consumir_presentacion_certificado_v1(text,text,text,text,bigint,bytea,bytea,bytea,bytea,bytea,boolean,text,text,text,text,timestamptz,timestamptz,timestamptz,text,text,bytea,bytea,bytea,text,text,timestamptz,timestamptz,timestamptz,timestamptz,timestamptz,timestamptz,text,text,text)"
	firmaReanudarPresentacion   = "vec_identidad_sesiones_v1.reanudar_y_consumir_presentacion_v1(text,text,text,text,bigint,text,bytea,bytea,bytea,bytea,bytea,text,text,timestamptz,timestamptz,timestamptz,timestamptz,timestamptz,timestamptz,text,text,text,text,text)"
	consultaInicioPresentacion  = `SELECT * FROM vec_identidad_sesiones_v1.iniciar_y_consumir_presentacion_certificado_v1(
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34)`
	consultaReanudarPresentacion = `SELECT * FROM vec_identidad_sesiones_v1.reanudar_y_consumir_presentacion_v1(
		$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
		$21,$22,$23,$24)`
)

// RegistroPresentacionesCertificadoPostgreSQL recibe un LOGIN técnico dedicado
// con exactamente dos EXECUTE. No comparte el pool de registrar/revalidar IS2
// ni puede leer o mutar directamente las tablas de sesiones.
type RegistroPresentacionesCertificadoPostgreSQL struct {
	pool           iniciadorTransacciones
	seudonimizador SeudonimizadorPresentacionCertificado
	coordenadas    CoordenadasPresentacionCertificado
}

func NuevoRegistroPresentacionesCertificadoPostgreSQL(
	ctx context.Context,
	pool *pgxpool.Pool,
	seudonimizador SeudonimizadorPresentacionCertificado,
	coordenadas CoordenadasPresentacionCertificado,
) (*RegistroPresentacionesCertificadoPostgreSQL, error) {
	if ctx == nil || pool == nil || valorNulo(seudonimizador) || !coordenadas.valida() {
		return nil, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	manifiesto := manifiestoCapacidadIdentidad{grupo: grupoPresentadorCertificado, funciones: []string{firmaInicioPresentacion, firmaReanudarPresentacion}}
	if _, err := acreditarManifiesto(ctx, pool, manifiesto); err != nil {
		return nil, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	return nuevoRegistroPresentacionesCertificadoPostgreSQL(pool, seudonimizador, coordenadas)
}

func nuevoRegistroPresentacionesCertificadoPostgreSQL(
	pool iniciadorTransacciones,
	seudonimizador SeudonimizadorPresentacionCertificado,
	coordenadas CoordenadasPresentacionCertificado,
) (*RegistroPresentacionesCertificadoPostgreSQL, error) {
	if valorNulo(pool) || valorNulo(seudonimizador) || !coordenadas.valida() {
		return nil, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	return &RegistroPresentacionesCertificadoPostgreSQL{pool: pool, seudonimizador: seudonimizador, coordenadas: coordenadas}, nil
}

func (r *RegistroPresentacionesCertificadoPostgreSQL) IniciarYConsumirPresentacion(
	ctx context.Context, orden httpseguridad.OrdenInicioCertificado,
) (httpseguridad.ResultadoRegistroPresentacionCertificado, error) {
	var vacio httpseguridad.ResultadoRegistroPresentacionCertificado
	alta, d, err := orden.Datos()
	if err != nil || !altaCoincidePresentacion(alta, d) {
		return vacio, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	s, err := r.seudonimos(ctx, d)
	if err != nil {
		return vacio, err
	}
	version, err := versionClavePresentacionSQL(s.ClaveVersion)
	if err != nil {
		return vacio, err
	}
	var ordinaria any
	if alta.CuentaPrivilegiada {
		return vacio, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	argumentos := []any{
		d.OperacionRef, s.Esquema, s.DominioRef, s.ClaveID, version,
		s.AsercionIDHMAC[:], s.SesionIDHMAC[:], s.SujetoIDHMAC[:], s.CuentaIDHMAC[:], ordinaria,
		alta.CuentaPrivilegiada, string(alta.Superficie), string(alta.MetodoObservado), string(alta.GarantiaObservada),
		alta.AutenticacionHuellaSHA256, alta.AutenticacionVerificadaEn, alta.SesionEmitidaEn,
		alta.AsercionExpiraEn, alta.PoliticaGarantiaRef, alta.PoliticaGarantiaHuellaSHA256,
		s.CertificadoDERHMAC[:], s.CAHMAC[:], s.NonceHMAC[:], d.CanalSHA256, d.AsercionActualHuellaSHA256,
		d.AsercionActualEmitidaEn, d.AsercionActualExpiraEn, d.CertificadoValidoHasta, d.CAValidaHasta,
		d.CRLSiguienteActualizacion, d.CertificadoVerificadoEn, d.PoliticaGarantiaRef, d.PoliticaGarantiaHuellaSHA256,
		d.ACRVerificado,
	}
	return r.ejecutar(ctx, consultaInicioPresentacion, argumentos, d)
}

func (r *RegistroPresentacionesCertificadoPostgreSQL) ReanudarYConsumirPresentacion(
	ctx context.Context, orden httpseguridad.OrdenReanudacionCertificado,
) (httpseguridad.ResultadoRegistroPresentacionCertificado, error) {
	var vacio httpseguridad.ResultadoRegistroPresentacionCertificado
	d, err := orden.Datos()
	if err != nil {
		return vacio, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	s, err := r.seudonimos(ctx, d)
	if err != nil {
		return vacio, err
	}
	version, err := versionClavePresentacionSQL(s.ClaveVersion)
	if err != nil {
		return vacio, err
	}
	argumentos := []any{
		d.OperacionRef, s.Esquema, s.DominioRef, s.ClaveID, version, string(d.Superficie),
		s.CertificadoDERHMAC[:], s.CAHMAC[:], s.SujetoIDHMAC[:], s.CuentaIDHMAC[:], s.NonceHMAC[:],
		d.CanalSHA256, d.AsercionActualHuellaSHA256, d.AsercionActualEmitidaEn, d.AsercionActualExpiraEn,
		d.CertificadoValidoHasta, d.CAValidaHasta, d.CRLSiguienteActualizacion, d.CertificadoVerificadoEn,
		d.PoliticaGarantiaRef, d.PoliticaGarantiaHuellaSHA256, string(d.MetodoObservado), string(d.GarantiaObservada),
		d.ACRVerificado,
	}
	return r.ejecutar(ctx, consultaReanudarPresentacion, argumentos, d)
}

func (r *RegistroPresentacionesCertificadoPostgreSQL) seudonimos(
	ctx context.Context, d httpseguridad.DatosPresentacionCertificado,
) (SeudonimosPresentacionCertificado, error) {
	if r == nil || ctx == nil || valorNulo(r.pool) || valorNulo(r.seudonimizador) || !r.coordenadas.valida() {
		return SeudonimosPresentacionCertificado{}, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	if err := ctx.Err(); err != nil {
		return SeudonimosPresentacionCertificado{}, err
	}
	s, err := r.seudonimizador.SeudonimizarPresentacionCertificado(ctx, IdentificadoresPresentacionCertificado{
		EspacioIdentidad: r.coordenadas.EspacioIdentidad, AsercionID: d.AsercionID,
		SesionIDAfirmada: d.SesionIDAfirmada, SujetoID: d.SujetoID, CuentaID: d.CuentaID,
		CertificadoSHA256: d.CertificadoSHA256, CASHA256: d.CASHA256, NonceID: d.AsercionID,
	})
	if err != nil || !s.validarPara(r.coordenadas, d) {
		return SeudonimosPresentacionCertificado{}, errorPresentacionPostgres(ctx)
	}
	if err := ctx.Err(); err != nil {
		return SeudonimosPresentacionCertificado{}, err
	}
	return s, nil
}

func altaCoincidePresentacion(a httpseguridad.AltaSesionAtomica, d httpseguridad.DatosPresentacionCertificado) bool {
	return a.Validar() == nil && !a.CuentaPrivilegiada && a.AsercionID == d.AsercionID &&
		a.SesionID == d.SesionIDAfirmada && a.SujetoID == d.SujetoID && a.CuentaID == d.CuentaID &&
		a.Superficie == d.Superficie && a.MetodoObservado == d.MetodoObservado &&
		a.GarantiaObservada == d.GarantiaObservada &&
		a.AutenticacionHuellaSHA256 == d.AsercionActualHuellaSHA256 &&
		a.SesionEmitidaEn.Equal(d.AsercionActualEmitidaEn) &&
		a.AsercionExpiraEn.Equal(d.AsercionActualExpiraEn) &&
		a.PoliticaGarantiaRef == d.PoliticaGarantiaRef &&
		a.PoliticaGarantiaHuellaSHA256 == d.PoliticaGarantiaHuellaSHA256
}

func versionClavePresentacionSQL(version uint64) (int64, error) {
	valor, err := strconv.ParseInt(strconv.FormatUint(version, 10), 10, 64)
	if err != nil || valor <= 0 {
		return 0, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	return valor, nil
}

func (r *RegistroPresentacionesCertificadoPostgreSQL) ejecutar(
	ctx context.Context, sql string, args []any, d httpseguridad.DatosPresentacionCertificado,
) (httpseguridad.ResultadoRegistroPresentacionCertificado, error) {
	var vacio httpseguridad.ResultadoRegistroPresentacionCertificado
	if ctx == nil || r == nil || valorNulo(r.pool) {
		return vacio, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	tx, err := r.pool.BeginTx(ctx, opcionesTransaccion())
	if err != nil {
		return vacio, errorPresentacionPostgres(ctx)
	}
	defer revertir(tx)
	if err = prepararTransaccion(ctx, tx); err != nil {
		return vacio, errorPresentacionPostgres(ctx)
	}
	resultado, err := consultarResultadoPresentacion(ctx, tx, sql, args)
	if err != nil || !resultadoValidoParaDatos(resultado, d) {
		return vacio, errorPresentacionPostgres(ctx)
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, errorPresentacionPostgres(ctx)
	}
	return resultado, nil
}

func consultarResultadoPresentacion(ctx context.Context, tx pgx.Tx, sql string, args []any) (httpseguridad.ResultadoRegistroPresentacionCertificado, error) {
	var r httpseguridad.ResultadoRegistroPresentacionCertificado
	var revision, superficie, metodo, garantia string
	var generacionPresentacion, generacionSesion int64
	err := tx.QueryRow(ctx, sql, args...).Scan(
		&r.SesionOriginal.AutenticacionRef, &r.SesionOriginal.AutenticacionHuellaSHA256,
		&r.SesionOriginal.AsercionRef, &r.SesionOriginal.SesionRef, &r.SesionOriginal.ControlSesionRef,
		&revision, &r.SesionOriginal.ControlSesionHuellaSHA256, &r.SesionOriginal.CuentaRef,
		&r.SesionOriginal.CuentaOrdinariaRef, &r.SesionOriginal.CuentaPrivilegiada,
		&superficie, &metodo, &garantia, &r.SesionOriginal.PoliticaGarantiaRef,
		&r.SesionOriginal.PoliticaGarantiaHuellaSHA256,
		&r.SesionOriginal.AutenticacionVerificadaEn, &r.SesionOriginal.SesionEmitidaEn,
		&r.SesionOriginal.SesionValidaHasta, &r.SesionOriginal.SesionRevalidadaEn,
		&r.Recibo.PresentacionRef, &generacionPresentacion, &r.Recibo.HuellaSHA256,
		&r.Recibo.RegistradaEn, &r.Recibo.ValidaHasta, &r.Recibo.CanalSHA256,
		&r.Recibo.AsercionActualHuellaSHA256, &r.Recibo.OperacionRef, &generacionSesion,
		&r.Recibo.ModoInicio,
	)
	if err != nil {
		return httpseguridad.ResultadoRegistroPresentacionCertificado{}, err
	}
	if generacionPresentacion <= 0 || generacionSesion <= 0 {
		return httpseguridad.ResultadoRegistroPresentacionCertificado{}, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	r.Recibo.Generacion = uint64(generacionPresentacion)
	r.Recibo.SesionGeneracion = uint64(generacionSesion)
	r.SesionOriginal.ControlSesionRevision, err = strconv.ParseUint(revision, 10, 64)
	if err != nil || r.SesionOriginal.ControlSesionRevision == 0 {
		return httpseguridad.ResultadoRegistroPresentacionCertificado{}, httpseguridad.ErrPresentacionCertificadoNoValida
	}
	r.SesionOriginal.Superficie = domain.SuperficieAutenticacionActorV1(superficie)
	r.SesionOriginal.MetodoObservado = domain.AuthMethod(metodo)
	r.SesionOriginal.GarantiaObservada = domain.AuthAssurance(garantia)
	r.SesionOriginal.AutenticacionVerificadaEn = instanteUTCPostgreSQL(r.SesionOriginal.AutenticacionVerificadaEn)
	r.SesionOriginal.SesionEmitidaEn = instanteUTCPostgreSQL(r.SesionOriginal.SesionEmitidaEn)
	r.SesionOriginal.SesionValidaHasta = instanteUTCPostgreSQL(r.SesionOriginal.SesionValidaHasta)
	r.SesionOriginal.SesionRevalidadaEn = instanteUTCPostgreSQL(r.SesionOriginal.SesionRevalidadaEn)
	r.Recibo.RegistradaEn = instanteUTCPostgreSQL(r.Recibo.RegistradaEn)
	r.Recibo.ValidaHasta = instanteUTCPostgreSQL(r.Recibo.ValidaHasta)
	return r, nil
}

func resultadoValidoParaDatos(r httpseguridad.ResultadoRegistroPresentacionCertificado, d httpseguridad.DatosPresentacionCertificado) bool {
	return r.SesionOriginal.Validar() == nil && r.Recibo.OperacionRef == d.OperacionRef &&
		r.Recibo.CanalSHA256 == d.CanalSHA256 &&
		r.Recibo.AsercionActualHuellaSHA256 == d.AsercionActualHuellaSHA256 &&
		r.Recibo.SesionGeneracion > 0 && r.Recibo.Generacion > 0 &&
		r.Recibo.RegistradaEn.Location() == time.UTC && r.Recibo.ValidaHasta.Location() == time.UTC
}

func errorPresentacionPostgres(ctx context.Context) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return httpseguridad.ErrPresentacionCertificadoNoValida
}

var _ httpseguridad.RegistroPresentacionesCertificado = (*RegistroPresentacionesCertificadoPostgreSQL)(nil)
