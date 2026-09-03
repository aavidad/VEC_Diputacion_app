package postgres

import (
	"context"
	"crypto/hmac"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	funcionConfirmarAltaV2      = "vec_contratacion_temporal.confirmar_alta_atestada_v2"
	maximoIntentosConfirmarAlta = 3
	limiteReconciliacionAlta    = 5 * time.Second
)

var _ ports.TransaccionAltasCandidata = (*TransaccionAltasPostgreSQLCandidata)(nil)

type TransaccionAltasPostgreSQLCandidata struct {
	pool      iniciadorTransacciones
	proveedor ports.ProveedorMaterialConfirmacionAlta
}

func NuevaTransaccionAltasPostgreSQLCandidata(
	pool *pgxpool.Pool,
	proveedor ports.ProveedorMaterialConfirmacionAlta,
) (*TransaccionAltasPostgreSQLCandidata, error) {
	return nuevaTransaccionAltasPostgreSQLCandidata(pool, proveedor)
}

func nuevaTransaccionAltasPostgreSQLCandidata(
	pool iniciadorTransacciones,
	proveedor ports.ProveedorMaterialConfirmacionAlta,
) (*TransaccionAltasPostgreSQLCandidata, error) {
	if dependenciaNula(pool) || dependenciaNula(proveedor) {
		return nil, ports.ErrPersistenciaNoDisponible
	}
	return &TransaccionAltasPostgreSQLCandidata{pool: pool, proveedor: proveedor}, nil
}

type entradasConfirmarAlta struct {
	capacidad       []byte
	decision        []byte
	motivo          []byte
	contextoActor   []byte
	personaVersion  int64
	perfilVersion   int64
	payloadVECAD3   []byte
	sobreCOSESign1  []byte
	evidencia       []byte
	raizPublicaSPKI []byte
	alta            []byte
	sellos          []byte
}

func (e *entradasConfirmarAlta) borrar() {
	if e == nil {
		return
	}
	for _, contenido := range [][]byte{
		e.capacidad, e.decision, e.motivo, e.contextoActor,
		e.payloadVECAD3, e.sobreCOSESign1, e.evidencia,
		e.raizPublicaSPKI, e.alta, e.sellos,
	} {
		borrarBytes(contenido)
	}
}

func (t *TransaccionAltasPostgreSQLCandidata) ConfirmarAltaCandidata(
	ctx context.Context,
	orden ports.OrdenConfirmarAltaCandidata,
) (ports.ReciboAlta, error) {
	if ctx == nil || t == nil || dependenciaNula(t.pool) || dependenciaNula(t.proveedor) {
		return ports.ReciboAlta{}, ports.ErrOrdenAltaInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	evidencia, err := orden.Datos()
	if err != nil {
		return ports.ReciboAlta{}, ports.ErrOrdenAltaInvalida
	}
	material, err := t.proveedor.ProveerMaterialConfirmacionAlta(ctx, orden)
	if err != nil {
		return ports.ReciboAlta{}, errorDependencia(ctx)
	}
	entradas, err := prepararEntradasConfirmarAlta(evidencia, material)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	defer entradas.borrar()
	return t.confirmarConEntradas(ctx, evidencia, entradas)
}

func (t *TransaccionAltasPostgreSQLCandidata) confirmarConEntradas(
	ctx context.Context,
	evidencia ports.EvidenciaOrdenConfirmarAltaCandidata,
	entradas entradasConfirmarAlta,
) (ports.ReciboAlta, error) {
	for intento := 1; intento <= maximoIntentosConfirmarAlta; intento++ {
		recibo, causa := t.confirmarEnTransaccion(ctx, evidencia, entradas)
		if causa == nil {
			return recibo, nil
		}
		if errorPostgreSQLReintentable(causa) && intento < maximoIntentosConfirmarAlta {
			continue
		}
		if errorConfirmacionAmbiguo(causa) {
			return t.reconciliarConfirmacion(evidencia, entradas)
		}
		return ports.ReciboAlta{}, normalizarErrorConfirmacion(ctx, causa)
	}
	return ports.ReciboAlta{}, ports.ErrPersistenciaNoDisponible
}

func prepararEntradasConfirmarAlta(
	evidencia ports.EvidenciaOrdenConfirmarAltaCandidata,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (entradasConfirmarAlta, error) {
	if material.ValidarEstructura() != nil {
		return entradasConfirmarAlta{}, ports.ErrPersistenciaNoDisponible
	}
	alta, sellos, huellaAlta, err := canonConfirmacionAlta(evidencia)
	if err != nil {
		return entradasConfirmarAlta{}, err
	}
	solicitud, errSolicitud := evidencia.SolicitudAutorizacionV3.Datos()
	confirmacion, errConfirmacion := evidencia.ConfirmacionRegistroV3.Datos()
	vinculo, errVinculo := solicitud.VinculoAutenticacionActor.Datos()
	ambitoActivo, _, errActivo := ports.ParActivoColeccionesHMACAlta(
		evidencia.AmbitosIdempotenciaHMAC,
		evidencia.HuellasPeticionHMAC,
	)
	huellaRecurso, errRecurso := solicitud.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if errSolicitud != nil || errConfirmacion != nil || errVinculo != nil || errActivo != nil ||
		errRecurso != nil || resumen.ValidarEstructura() != nil ||
		solicitud.Recurso.Atributos[ports.AtributoHuellaEfectoAltaSHA256] != huellaAlta ||
		resumen.DecisionRef() != confirmacion.DecisionRef ||
		resumen.DecisionHuellaSHA256() != confirmacion.DecisionHuellaSHA256 ||
		resumen.ContextoRef() != vinculo.RegistroContextoRef ||
		resumen.ContextoHuellaSHA256() != vinculo.ContextoActorHuellaSHA256 ||
		resumen.Operacion() != ports.AccionCrearSolicitud ||
		resumen.EfectoRef() != ambitoActivo ||
		resumen.EfectoHuellaSHA256() != huellaRecurso ||
		resumen.AudienciaConsumo() != audienciaConfirmarAltaV1 ||
		!capacidadBreveContenidaEnConcesion(resumen, confirmacion) {
		borrarBytes(alta)
		borrarBytes(sellos)
		return entradasConfirmarAlta{}, ports.ErrPersistenciaNoDisponible
	}
	return entradasConfirmarAlta{
		capacidad: material.CapacidadCanonica(), decision: material.DecisionCanonica(),
		motivo: material.MotivoCanonico(), contextoActor: material.ContextoActorCanonico(),
		personaVersion: int64(material.PersonaVersion()),
		perfilVersion:  int64(material.PerfilVersion()),
		payloadVECAD3:  material.PayloadVECAD3(), sobreCOSESign1: material.SobreCOSESign1(),
		evidencia: material.EvidenciaVerificacion(), raizPublicaSPKI: material.RaizPublicaSPKI(),
		alta: alta, sellos: sellos,
	}, nil
}

func capacidadBreveContenidaEnConcesion(
	resumen puertosvec.ResumenCapacidadAtestacionAutorizacionV3,
	confirmacion puertosvec.DatosConfirmacionRegistroConcesionAutorizacionLigadaV3,
) bool {
	emitidaEn := resumen.EmitidaEn()
	return !emitidaEn.Before(confirmacion.EmitidaEn) &&
		!emitidaEn.Before(confirmacion.RegistradaEn) &&
		emitidaEn.Before(confirmacion.ValidaHasta) &&
		!resumen.ExpiraEn().After(confirmacion.ValidaHasta)
}

type filaConfirmacionAlta struct {
	expedienteRef string
	numeroVisible string
	version       int64
	reciboRef     string
	auditoriaRef  string
	eventoRef     string
	confirmadaEn  time.Time
	huellaRecibo  string
}

func (t *TransaccionAltasPostgreSQLCandidata) confirmarEnTransaccion(
	ctx context.Context,
	evidencia ports.EvidenciaOrdenConfirmarAltaCandidata,
	entradas entradasConfirmarAlta,
) (ports.ReciboAlta, error) {
	tx, err := iniciarTransaccionAltaCandidata(ctx, t.pool)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	defer revertirTransaccion(tx)
	fila := filaConfirmacionAlta{}
	err = tx.QueryRow(ctx, consultaConfirmarAltaV2(), argumentosConfirmarAlta(entradas)...).Scan(
		&fila.expedienteRef, &fila.numeroVisible, &fila.version,
		&fila.reciboRef, &fila.auditoriaRef, &fila.eventoRef,
		&fila.confirmadaEn, &fila.huellaRecibo,
	)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	recibo, err := fila.restaurar(evidencia)
	if err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboAlta{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ReciboAlta{}, marcarEnvioPosibleConfirmacion(err)
	}
	return recibo, nil
}

func consultaConfirmarAltaV2() string {
	return `SELECT expediente_ref, numero_visible, version, recibo_ref,
	              auditoria_ref, evento_ref, confirmada_en,
	              recibo_huella_sha256
	         FROM ` + funcionConfirmarAltaV2 + `(
	              $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`
}

func argumentosConfirmarAlta(e entradasConfirmarAlta) []any {
	return []any{
		e.capacidad, e.decision, e.motivo, e.contextoActor,
		e.personaVersion, e.perfilVersion, e.payloadVECAD3,
		e.sobreCOSESign1, e.evidencia, e.raizPublicaSPKI,
		e.alta, e.sellos,
	}
}

func (f filaConfirmacionAlta) restaurar(
	evidencia ports.EvidenciaOrdenConfirmarAltaCandidata,
) (ports.ReciboAlta, error) {
	if f.version < 1 {
		return ports.ReciboAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	recibo := ports.ReciboAlta{
		ExpedienteRef: f.expedienteRef, NumeroVisible: f.numeroVisible,
		Version: uint64(f.version), ReciboRef: f.reciboRef,
		AuditoriaRef: f.auditoriaRef, EventoRef: f.eventoRef,
		ConfirmadaEn: f.confirmadaEn.UTC(),
	}
	if recibo.ValidarPara(evidencia.Expediente) != nil ||
		!hmac.Equal([]byte(huellaReciboAlta(recibo)), []byte(f.huellaRecibo)) {
		return ports.ReciboAlta{}, ports.ErrResultadoAltaNoConfiable
	}
	return recibo, nil
}

func (t *TransaccionAltasPostgreSQLCandidata) reconciliarConfirmacion(
	evidencia ports.EvidenciaOrdenConfirmarAltaCandidata,
	entradas entradasConfirmarAlta,
) (ports.ReciboAlta, error) {
	ctx, cancelar := context.WithTimeout(context.Background(), limiteReconciliacionAlta)
	defer cancelar()
	recibo, err := t.confirmarEnTransaccion(ctx, evidencia, entradas)
	if err == nil {
		return recibo, nil
	}
	if errorConfirmacionAmbiguo(err) {
		return ports.ReciboAlta{}, ports.ErrResultadoAltaIndeterminado
	}
	return ports.ReciboAlta{}, normalizarErrorConfirmacion(ctx, err)
}

func errorConfirmacionAmbiguo(err error) bool {
	if err == nil || errors.Is(err, pgx.ErrTxCommitRollback) ||
		errors.Is(err, ports.ErrResultadoAltaNoConfiable) ||
		errors.Is(err, ports.ErrResultadoAltaIndeterminado) ||
		errors.Is(err, ports.ErrOrdenAltaInvalida) ||
		errors.Is(err, ports.ErrPersistenciaNoDisponible) ||
		errorPostgreSQLReintentable(err) || pgconn.SafeToRetry(err) {
		return false
	}
	var posible errorEnvioPosibleConfirmacion
	return errors.As(err, &posible)
}

type errorEnvioPosibleConfirmacion struct {
	causa error
}

func (e errorEnvioPosibleConfirmacion) Error() string {
	return "confirmacion de alta con resultado de transporte incierto"
}

func (e errorEnvioPosibleConfirmacion) Unwrap() error {
	return e.causa
}

func marcarEnvioPosibleConfirmacion(err error) error {
	if err == nil || pgconn.SafeToRetry(err) || errors.Is(err, pgx.ErrTxCommitRollback) ||
		errorPostgreSQLReintentable(err) {
		return err
	}
	var postgres *pgconn.PgError
	if errors.As(err, &postgres) {
		if postgres.Code == "08007" {
			return errorEnvioPosibleConfirmacion{causa: err}
		}
		return err
	}
	return errorEnvioPosibleConfirmacion{causa: err}
}

func normalizarErrorConfirmacion(ctx context.Context, causa error) error {
	if errors.Is(causa, ports.ErrResultadoAltaNoConfiable) ||
		errors.Is(causa, ports.ErrResultadoAltaIndeterminado) {
		return causa
	}
	if ctx != nil && ctx.Err() != nil && !errorConfirmacionAmbiguo(causa) {
		return ctx.Err()
	}
	return ports.ErrPersistenciaNoDisponible
}
