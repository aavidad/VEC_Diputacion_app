package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	gobiernoconvocatorias "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoconvocatorias"
)

const funcionVerificarReciboBorradorPostgreSQL = "vec_bolsa_convocatorias.verificar_recibo_borrador_v1"

const duracionMaximaVerificacionCriptograficaPostgreSQL = 2 * time.Second

var _ gobiernoconvocatorias.VerificadorReciboBorrador = (*VerificadorReciboBorradorPostgreSQL)(nil)

// VerificadorCriptograficoReciboBorrador es una dependencia del adaptador de
// lectura, no un puerto nuevo del núcleo. Debe conservar únicamente material
// público de verificación y nunca claves de firma ni credenciales A/B.
type VerificadorCriptograficoReciboBorrador interface {
	gobiernoconvocatorias.DescriptorAutoridadBorrador
	VerificarEvidenciasRecibo(
		context.Context,
		gobiernoconvocatorias.ProyeccionReciboBorrador,
	) error
}

// VerificadorReciboBorradorPostgreSQL requiere un pool exclusivo cuyo LOGIN
// solo pertenezca al rol verificador. La función SQL vuelve a leer tras el
// COMMIT; después, una autoridad criptográfica independiente verifica A/B.
type VerificadorReciboBorradorPostgreSQL struct {
	pool         iniciadorTransacciones
	criptografia VerificadorCriptograficoReciboBorrador
	identidad    gobiernoconvocatorias.IdentidadAutoridadBorrador
	vinculo      gobiernoconvocatorias.VinculoVerificadorReciboBorrador
}

func NuevoVerificadorReciboBorradorPostgreSQL(
	pool *pgxpool.Pool,
	criptografia VerificadorCriptograficoReciboBorrador,
	identidad gobiernoconvocatorias.IdentidadAutoridadBorrador,
) (*VerificadorReciboBorradorPostgreSQL, error) {
	return nuevoVerificadorReciboBorradorPostgreSQL(pool, criptografia, identidad)
}

func nuevoVerificadorReciboBorradorPostgreSQL(
	pool iniciadorTransacciones,
	criptografia VerificadorCriptograficoReciboBorrador,
	identidad gobiernoconvocatorias.IdentidadAutoridadBorrador,
) (*VerificadorReciboBorradorPostgreSQL, error) {
	if valorNulo(pool) || valorNulo(criptografia) ||
		!identidadAutoridadBorradorPostgreSQLValida(identidad) ||
		!autoridadesBorradorPostgreSQLSeparadas(identidad, criptografia.IdentidadAutoridadBorrador()) {
		return nil, gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	vinculo, err := gobiernoconvocatorias.NuevoVinculoVerificadorReciboBorrador(
		identidad, criptografia.IdentidadAutoridadBorrador(),
	)
	if err != nil {
		return nil, gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	return &VerificadorReciboBorradorPostgreSQL{
		pool: pool, criptografia: criptografia, identidad: identidad, vinculo: vinculo,
	}, nil
}

func (v *VerificadorReciboBorradorPostgreSQL) IdentidadAutoridadBorrador() gobiernoconvocatorias.IdentidadAutoridadBorrador {
	if v == nil {
		return gobiernoconvocatorias.IdentidadAutoridadBorrador{}
	}
	return v.identidad
}

func (v *VerificadorReciboBorradorPostgreSQL) VinculoVerificadorReciboBorrador() gobiernoconvocatorias.VinculoVerificadorReciboBorrador {
	if v == nil {
		return gobiernoconvocatorias.VinculoVerificadorReciboBorrador{}
	}
	return v.vinculo
}

func (v *VerificadorReciboBorradorPostgreSQL) VerificarReciboBorrador(
	ctx context.Context,
	recibo gobiernoconvocatorias.ProyeccionReciboBorrador,
) error {
	if ctx == nil || v == nil || valorNulo(v.pool) || valorNulo(v.criptografia) {
		return gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	if _, _, valido := identidadesVerificadorReciboPostgreSQL(v); !valido {
		return gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !reciboAcreditaVinculoVerificadorPostgreSQL(v.vinculo, recibo) {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	if _, _, _, err := recibo.EvidenciasKMSParaVerificacion(); err != nil {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	reciboJSON, err := serializarReciboBorradorPostgreSQL(recibo)
	if err != nil {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	defer borrarBytesDiarioPostgreSQL(reciboJSON)
	tx, err := iniciarTransaccionBorradorPostgreSQL(ctx, v.pool, pgx.ReadOnly)
	if err != nil {
		return errorConfirmacionBorradorPostgreSQL(ctx, err)
	}
	defer revertir(tx)
	fila, err := releerReciboBorradorPostgreSQL(ctx, tx, recibo, reciboJSON)
	defer fila.borrar()
	if err != nil {
		return errorConfirmacionBorradorPostgreSQL(ctx, err)
	}
	durable, err := fila.validarYRestaurar(recibo)
	if err != nil {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return confirmarRelecturaYVerificarCriptografiaPostgreSQL(
		ctx, tx, v.criptografia, durable, duracionMaximaVerificacionCriptograficaPostgreSQL,
	)
}

func reciboAcreditaVinculoVerificadorPostgreSQL(
	vinculo gobiernoconvocatorias.VinculoVerificadorReciboBorrador,
	recibo gobiernoconvocatorias.ProyeccionReciboBorrador,
) bool {
	referencia, err := vinculo.ReferenciaParaAcreditacion()
	return err == nil && referencia != "" && recibo.AcreditacionKMS.VerificadorRef == referencia
}

func confirmarRelecturaYVerificarCriptografiaPostgreSQL(
	ctx context.Context,
	tx pgx.Tx,
	criptografia VerificadorCriptograficoReciboBorrador,
	recibo gobiernoconvocatorias.ProyeccionReciboBorrador,
	duracionMaxima time.Duration,
) error {
	if ctx == nil || valorNulo(tx) || valorNulo(criptografia) || duracionMaxima <= 0 ||
		duracionMaxima > duracionMaximaVerificacionCriptograficaPostgreSQL {
		return gobiernoconvocatorias.ErrServicioBorradoresInvalido
	}
	if err := tx.Commit(ctx); err != nil {
		return errorConfirmacionBorradorPostgreSQL(ctx, err)
	}
	ctxCriptografia, cancelar := context.WithTimeout(ctx, duracionMaxima)
	defer cancelar()
	err := criptografia.VerificarEvidenciasRecibo(ctxCriptografia, recibo)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if ctxCriptografia.Err() != nil {
		return gobiernoconvocatorias.ErrOperacionBorradorEnCurso
	}
	if err != nil {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return nil
}

func identidadesVerificadorReciboPostgreSQL(
	v *VerificadorReciboBorradorPostgreSQL,
) (
	identidadDB gobiernoconvocatorias.IdentidadAutoridadBorrador,
	identidadCriptografica gobiernoconvocatorias.IdentidadAutoridadBorrador,
	valido bool,
) {
	if v == nil || valorNulo(v.pool) || valorNulo(v.criptografia) ||
		!identidadAutoridadBorradorPostgreSQLValida(v.identidad) {
		return identidadDB, identidadCriptografica, false
	}
	identidadCriptografica = v.criptografia.IdentidadAutoridadBorrador()
	if !identidadAutoridadBorradorPostgreSQLValida(identidadCriptografica) ||
		!autoridadesBorradorPostgreSQLSeparadas(v.identidad, identidadCriptografica) {
		return identidadDB, gobiernoconvocatorias.IdentidadAutoridadBorrador{}, false
	}
	vinculoEsperado, err := gobiernoconvocatorias.NuevoVinculoVerificadorReciboBorrador(
		v.identidad, identidadCriptografica,
	)
	referenciaEsperada, errEsperada := vinculoEsperado.ReferenciaParaAcreditacion()
	referenciaReal, errReal := v.vinculo.ReferenciaParaAcreditacion()
	if err != nil || errEsperada != nil || errReal != nil || referenciaReal != referenciaEsperada {
		return identidadDB, gobiernoconvocatorias.IdentidadAutoridadBorrador{}, false
	}
	return v.identidad, identidadCriptografica, true
}

type filaVerificacionReciboBorradorPostgreSQL struct {
	reciboRef, transaccionRef, convocatoriaID  string
	secuencia, revision                        pgtype.Int8
	recibo, reciboCanonico                     []byte
	huellaRecibo                               string
	cuerpoCanonico                             []byte
	huellaCuerpo                               string
	acreditacion, acreditacionCanonica         []byte
	huellaAcreditacion                         string
	preimagenAtestacion, preimagenRevalidacion []byte
	verificadaEn                               pgtype.Timestamptz
}

func (f *filaVerificacionReciboBorradorPostgreSQL) borrar() {
	borrarBytesDiarioPostgreSQL(
		f.recibo, f.reciboCanonico, f.cuerpoCanonico, f.acreditacion,
		f.acreditacionCanonica, f.preimagenAtestacion, f.preimagenRevalidacion,
	)
}

func releerReciboBorradorPostgreSQL(
	ctx context.Context, tx pgx.Tx,
	recibo gobiernoconvocatorias.ProyeccionReciboBorrador,
	reciboJSON []byte,
) (filaVerificacionReciboBorradorPostgreSQL, error) {
	var f filaVerificacionReciboBorradorPostgreSQL
	err := tx.QueryRow(ctx, `
		SELECT recibo_ref, transaccion_ref, convocatoria_id, secuencia,
		       revision, recibo, recibo_canonico, huella_recibo_sha256,
		       cuerpo_recibo_canonico, huella_cuerpo_recibo_sha256,
		       acreditacion, acreditacion_canonica,
		       huella_acreditacion_sha256, preimagen_atestacion_kms,
		       preimagen_revalidacion_kms, verificada_en
		FROM `+funcionVerificarReciboBorradorPostgreSQL+`(
			$1::text, $2::text,
			encode(sha256(convert_to($3::jsonb::text, 'UTF8')), 'hex')
		)`,
		recibo.ReciboRef, recibo.TransaccionRef, reciboJSON,
	).Scan(
		&f.reciboRef, &f.transaccionRef, &f.convocatoriaID, &f.secuencia,
		&f.revision, &f.recibo, &f.reciboCanonico, &f.huellaRecibo,
		&f.cuerpoCanonico, &f.huellaCuerpo, &f.acreditacion,
		&f.acreditacionCanonica, &f.huellaAcreditacion,
		&f.preimagenAtestacion, &f.preimagenRevalidacion, &f.verificadaEn,
	)
	return f, err
}

func (f filaVerificacionReciboBorradorPostgreSQL) validarYRestaurar(
	esperado gobiernoconvocatorias.ProyeccionReciboBorrador,
) (gobiernoconvocatorias.ProyeccionReciboBorrador, error) {
	secuencia, errSecuencia := enteroPositivoConfirmacionPostgreSQL(f.secuencia)
	revision, errRevision := enteroPositivoConfirmacionPostgreSQL(f.revision)
	verificada, errVerificada := restaurarInstanteObligatorioDiarioPostgreSQL(f.verificadaEn)
	if errSecuencia != nil || errRevision != nil || secuencia > math.MaxInt64 || revision > math.MaxInt64 ||
		!metadatosVersionReciboCoinciden(f.convocatoriaID, secuencia, revision, esperado) ||
		f.reciboRef != esperado.ReciboRef ||
		f.transaccionRef != esperado.TransaccionRef || len(f.recibo) == 0 ||
		!bytes.Equal(f.recibo, f.reciboCanonico) || huellaSHA256BytesPostgreSQL(f.reciboCanonico) != f.huellaRecibo ||
		huellaSHA256BytesPostgreSQL(f.cuerpoCanonico) != f.huellaCuerpo ||
		huellaSHA256BytesPostgreSQL(f.acreditacionCanonica) != f.huellaAcreditacion ||
		errVerificada != nil || verificada.Before(esperado.ConfirmadaEn) {
		return gobiernoconvocatorias.ProyeccionReciboBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	durable, err := restaurarReciboBorradorPostgreSQL(f.recibo)
	if err != nil || durable == nil || *durable != esperado {
		return gobiernoconvocatorias.ProyeccionReciboBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	acreditacion, err := restaurarAcreditacionDesdeJSONBorradorPostgreSQL(f.acreditacion)
	if err != nil || acreditacion != durable.AcreditacionKMS ||
		f.huellaCuerpo != durable.AcreditacionKMS.HuellaCuerpoReciboSHA256 ||
		f.huellaAcreditacion != durable.AcreditacionKMS.HuellaAcreditacionSHA256 ||
		validarPreimagenesReciboBorradorPostgreSQL(*durable, f.preimagenAtestacion, f.preimagenRevalidacion) != nil {
		return gobiernoconvocatorias.ProyeccionReciboBorrador{}, gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return *durable, nil
}

func metadatosVersionReciboCoinciden(
	convocatoriaID string, secuencia, revision uint64,
	recibo gobiernoconvocatorias.ProyeccionReciboBorrador,
) bool {
	return convocatoriaID != "" && recibo.EstadoPrincipal.Revision > 0 &&
		convocatoriaID+"#"+strconv.FormatUint(secuencia, 10) == recibo.EstadoPrincipal.Referencia &&
		revision == uint64(recibo.EstadoPrincipal.Revision)
}

func restaurarAcreditacionDesdeJSONBorradorPostgreSQL(
	contenido []byte,
) (gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador, error) {
	var persistida acreditacionKMSReciboPostgreSQL
	if err := decodificarJSONCerradoDiarioPostgreSQL(contenido, &persistida); err != nil {
		return gobiernoconvocatorias.AcreditacionKMSConfirmacionBorrador{}, err
	}
	return restaurarAcreditacionKMSReciboPostgreSQL(persistida)
}

func validarPreimagenesReciboBorradorPostgreSQL(
	recibo gobiernoconvocatorias.ProyeccionReciboBorrador,
	preimagenAtestacion, preimagenRevalidacion []byte,
) error {
	atestacion, solicitud, revalidacion, err := recibo.EvidenciasKMSParaVerificacion()
	if err != nil {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	preimagenA, _, _, _, firmaA, errA := atestacion.DatosParaVerificacionFirma()
	preimagenB, _, _, _, firmaB, errB := revalidacion.DatosParaVerificacionFirma(solicitud)
	defer borrarBytesDiarioPostgreSQL(preimagenA, preimagenB, firmaA, firmaB)
	if errors.Join(errA, errB) != nil || !bytes.Equal(preimagenA, preimagenAtestacion) ||
		!bytes.Equal(preimagenB, preimagenRevalidacion) {
		return gobiernoconvocatorias.ErrResultadoBorradorInseguro
	}
	return nil
}

func huellaSHA256BytesPostgreSQL(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}
