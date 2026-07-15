package confianzadocumental

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type iniciadorConsultasConfianzaDocumentalV4 interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// EjecutorDocumentalPostgreSQLV4 es la composicion del proceso web. Solo
// contiene la credencial ejecutora y un cliente concreto de socket Unix. No
// carga raices, no conoce el secreto HMAC y no puede emitir capacidades.
type EjecutorDocumentalPostgreSQLV4 struct {
	repositorio *repositorioPostgreSQLEjecucionDocumentalV4
	cliente     *clienteEmisorCapacidadUnixV4
}

// NuevoEjecutorDocumentalPostgreSQLV4 exige un pool concreto con la identidad
// ejecutor_atestado y la ruta absoluta al socket del verificador aislado. No
// acepta una interfaz emisora, una raiz, una clave ni un repositorio externo.
func NuevoEjecutorDocumentalPostgreSQLV4(
	ctx context.Context,
	pool *pgxpool.Pool,
	rutaSocketEmisor string,
) (*EjecutorDocumentalPostgreSQLV4, error) {
	if ctx == nil || pool == nil {
		return nil, ErrConfiguracionConfianzaDocumentalInvalida
	}
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(ErrConfiguracionConfianzaDocumentalInvalida, err)
	}
	repositorio, err := nuevoRepositorioPostgreSQLEjecucionDocumentalV4(pool)
	if err != nil {
		return nil, ErrConfiguracionConfianzaDocumentalInvalida
	}
	cliente, err := nuevoClienteEmisorCapacidadUnixV4(rutaSocketEmisor)
	if err != nil {
		return nil, ErrConfiguracionConfianzaDocumentalInvalida
	}
	ejecutor := &EjecutorDocumentalPostgreSQLV4{
		repositorio: repositorio, cliente: cliente,
	}
	if ejecutor.validar() != nil {
		return nil, ErrConfiguracionConfianzaDocumentalInvalida
	}
	return ejecutor, nil
}

// EjecutarDocumentalAtestadoV4 implementa el puerto neutral del nucleo. El
// resultado que cruza la frontera no contiene el paquete, la capacidad ni
// material criptografico reutilizable.
func (e *EjecutorDocumentalPostgreSQLV4) EjecutarDocumentalAtestadoV4(
	ctx context.Context,
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (ports.ResultadoConectorEjecucionDocumentalAtestadaV4, error) {
	resultado, err := e.ejecutar(ctx, vinculo, cabecera, sobre)
	if err != nil {
		return ports.ResultadoConectorEjecucionDocumentalAtestadaV4{}, err
	}
	confirmacion, err := ports.NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
		resultado.OrdenRef, resultado.Estado, resultado.AuditoriaRef,
		resultado.EventoOutboxRef, resultado.RegistradaEn,
	)
	if err != nil {
		return ports.ResultadoConectorEjecucionDocumentalAtestadaV4{},
			errorAutoridadInternaEjecucionDocumentalV4()
	}
	return confirmacion, nil
}

// ejecutar extrae unicamente la preimagen no autoritativa de la solicitud
// opaca, solicita al proceso aislado que verifique COSE y emita una capacidad,
// y presenta el paquete a PostgreSQL. El HMAC se valida y consume dentro del
// mismo COMMIT que el efecto, la auditoria y el outbox.
func (e *EjecutorDocumentalPostgreSQLV4) ejecutar(
	ctx context.Context,
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (ResultadoEjecucionPlanDocumentalV4, error) {
	if e.validar() != nil || ctx == nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorAutoridadInternaEjecucionDocumentalV4()
	}
	instante := time.Now().UTC().Truncate(time.Microsecond)
	if vinculo.ValidarEn(instante) != nil || cabecera.Validar() != nil ||
		sobre.ValidarSintaxis() != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorCapacidadDocumentalV4(nil)
	}
	solicitudAplicacion, err := vinculo.PrepararSolicitudAplicacionEn(instante)
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorCapacidadDocumentalV4(err)
	}
	preimagen, err := solicitudAplicacion.PreimagenRecursoParaEvidenciaDurable()
	if err != nil {
		return ResultadoEjecucionPlanDocumentalV4{}, errorCapacidadDocumentalV4(err)
	}
	preimagenBytes, errPreimagen := preimagen.SerializacionCanonicaParaPersistencia()
	sobreBytes, errSobre := sobre.COSESign1()
	if errPreimagen != nil || errSobre != nil {
		return ResultadoEjecucionPlanDocumentalV4{},
			errorCapacidadDocumentalV4(errors.Join(errPreimagen, errSobre))
	}
	artefactos, err := e.cliente.solicitar(ctx, cabecera, sobreBytes, preimagenBytes)
	if err != nil || !bytes.Equal(artefactos.preimagen, preimagenBytes) ||
		!bytes.Equal(artefactos.sobre, sobreBytes) {
		return ResultadoEjecucionPlanDocumentalV4{}, errorCapacidadDocumentalV4(err)
	}
	return e.repositorio.ejecutarArtefactosAtestados(ctx, artefactos)
}

func (e *EjecutorDocumentalPostgreSQLV4) validar() error {
	if e == nil || e.repositorio == nil || e.cliente == nil ||
		interfazPostgreSQLDocumentalNula(e.repositorio.pool) || e.cliente.cliente == nil {
		return ErrConfiguracionConfianzaDocumentalInvalida
	}
	return nil
}

type filaConfianzaPostgreSQLV4 struct {
	revision                 string
	huellaConfiguracion      string
	configuracionPublicadaEn time.Time
	configuracionExpiraEn    time.Time
	configuracionEstado      string
	configuracionRevocadaEn  pgtype.Timestamptz
	claveID                  string
	algoritmo                string
	suite                    string
	audienciaCOSE            string
	audienciaDespliegue      string
	clavePublicaSPKI         []byte
	huellaClave              string
	raizValidaDesde          time.Time
	raizValidaHasta          time.Time
	raizEstado               string
	raizRevocadaEn           pgtype.Timestamptz
}

func cargarConfiguracionConfianzaPostgreSQLV4(
	ctx context.Context,
	pool *pgxpool.Pool,
) (ConfiguracionConfianzaFijada, error) {
	if ctx == nil || pool == nil {
		return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err = configurarTransaccionEjecucionDocumentalV4(ctx, tx); err != nil {
		return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	filas, err := tx.Query(ctx, `
		SELECT revision, huella_configuracion_sha256,
		       configuracion_publicada_en, configuracion_expira_en,
		       configuracion_estado, configuracion_revocada_en,
		       clave_id, algoritmo_cose, suite, audiencia_cose,
		       audiencia_despliegue, clave_publica_spki,
		       huella_clave_sha256, raiz_valida_desde, raiz_valida_hasta,
		       raiz_estado, raiz_revocada_en
		  FROM vec_ejecucion_documental_v4.obtener_confianza_actual()`)
	if err != nil {
		return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	defer filas.Close()
	var base *filaConfianzaPostgreSQLV4
	raices := make([]RaizPublicaFijada, 0, 4)
	for filas.Next() {
		var fila filaConfianzaPostgreSQLV4
		if err = filas.Scan(
			&fila.revision, &fila.huellaConfiguracion,
			&fila.configuracionPublicadaEn, &fila.configuracionExpiraEn,
			&fila.configuracionEstado, &fila.configuracionRevocadaEn,
			&fila.claveID, &fila.algoritmo, &fila.suite,
			&fila.audienciaCOSE, &fila.audienciaDespliegue,
			&fila.clavePublicaSPKI, &fila.huellaClave,
			&fila.raizValidaDesde, &fila.raizValidaHasta,
			&fila.raizEstado, &fila.raizRevocadaEn,
		); err != nil {
			return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
		}
		fila.normalizarInstantes()
		if fila.validarBase() != nil {
			return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
		}
		if base == nil {
			copia := fila
			base = &copia
		} else if !fila.mismaConfiguracion(*base) {
			return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
		}
		clave, errClave := x509.ParsePKIXPublicKey(fila.clavePublicaSPKI)
		if errClave != nil {
			return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
		}
		raiz, errRaiz := nuevaRaizPublicaFijadaAtestacionPDP(
			[]byte(fila.claveID), AlgoritmoCOSEDocumental(fila.algoritmo), clave,
			fila.suite, fila.audienciaDespliegue,
			EstadoConfianzaClaveDocumental(fila.raizEstado),
			fila.raizValidaDesde, fila.raizValidaHasta, time.Time{},
		)
		if errRaiz != nil || raiz.huellaClaveSHA256 != fila.huellaClave {
			return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
		}
		raices = append(raices, raiz)
	}
	if err = filas.Err(); err != nil || base == nil || len(raices) == 0 {
		return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		base.revision, base.configuracionPublicadaEn,
		base.configuracionExpiraEn, raices...,
	)
	if err != nil || configuracion.huellaSHA256 != base.huellaConfiguracion {
		return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	if err = tx.Commit(ctx); err != nil {
		return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	return configuracion, nil
}

// cargarMaterialEmisorCapacidadPostgreSQLV4 solo se usa al arrancar el
// proceso verificador aislado. La funcion SQL correspondiente no esta
// concedida a la identidad ejecutora del portal.
func cargarMaterialEmisorCapacidadPostgreSQLV4(
	ctx context.Context,
	pool iniciadorConsultasConfianzaDocumentalV4,
) (ConfiguracionConfianzaFijada, materialEmisorCapacidadDocumentalV4, error) {
	if ctx == nil || interfazPostgreSQLDocumentalNula(pool) {
		return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
			ErrConfiguracionConfianzaDocumentalInvalida
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
			ErrConfiguracionConfianzaDocumentalInvalida
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err = configurarTransaccionEjecucionDocumentalV4(ctx, tx); err != nil {
		return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
			ErrConfiguracionConfianzaDocumentalInvalida
	}
	filas, err := tx.Query(ctx, `
		SELECT revision, huella_configuracion_sha256,
		       configuracion_publicada_en, configuracion_expira_en,
		       configuracion_estado, configuracion_revocada_en,
		       clave_id, algoritmo_cose, suite, audiencia_cose,
		       audiencia_despliegue, clave_publica_spki,
		       huella_clave_sha256, raiz_valida_desde, raiz_valida_hasta,
		       raiz_estado, raiz_revocada_en,
		       capacidad_clave_id, capacidad_clave_version::text,
		       capacidad_secreto, capacidad_emisor_id,
		       capacidad_valida_desde, capacidad_valida_hasta,
		       capacidad_estado, capacidad_revocada_en
		  FROM vec_ejecucion_documental_v4.obtener_material_emisor_capacidad()`)
	if err != nil {
		return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
			ErrConfiguracionConfianzaDocumentalInvalida
	}
	defer filas.Close()
	var base *filaConfianzaPostgreSQLV4
	var materialBase *materialEmisorCapacidadDocumentalV4
	raices := make([]RaizPublicaFijada, 0, 4)
	for filas.Next() {
		var fila filaConfianzaPostgreSQLV4
		var versionTexto string
		var material materialEmisorCapacidadDocumentalV4
		var revocada pgtype.Timestamptz
		if err = filas.Scan(
			&fila.revision, &fila.huellaConfiguracion,
			&fila.configuracionPublicadaEn, &fila.configuracionExpiraEn,
			&fila.configuracionEstado, &fila.configuracionRevocadaEn,
			&fila.claveID, &fila.algoritmo, &fila.suite,
			&fila.audienciaCOSE, &fila.audienciaDespliegue,
			&fila.clavePublicaSPKI, &fila.huellaClave,
			&fila.raizValidaDesde, &fila.raizValidaHasta,
			&fila.raizEstado, &fila.raizRevocadaEn,
			&material.claveID, &versionTexto, &material.secreto,
			&material.emisorID, &material.validaDesde, &material.validaHasta,
			&material.estado, &revocada,
		); err != nil {
			return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
				ErrConfiguracionConfianzaDocumentalInvalida
		}
		material.version, err = strconv.ParseUint(versionTexto, 10, 64)
		if err != nil {
			return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
				ErrConfiguracionConfianzaDocumentalInvalida
		}
		fila.normalizarInstantes()
		material.validaDesde = material.validaDesde.UTC().Truncate(time.Microsecond)
		material.validaHasta = material.validaHasta.UTC().Truncate(time.Microsecond)
		if revocada.Valid {
			material.revocadaEn = revocada.Time.UTC().Truncate(time.Microsecond)
		}
		if fila.validarBase() != nil || material.validarEn(
			time.Now().UTC().Truncate(time.Microsecond),
		) != nil {
			return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
				ErrConfiguracionConfianzaDocumentalInvalida
		}
		if base == nil {
			copia := fila
			base = &copia
			copiaMaterial := material
			copiaMaterial.secreto = append([]byte(nil), material.secreto...)
			materialBase = &copiaMaterial
		} else if !fila.mismaConfiguracion(*base) ||
			!mismoMaterialEmisorCapacidadV4(material, *materialBase) {
			return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
				ErrConfiguracionConfianzaDocumentalInvalida
		}
		clave, errClave := x509.ParsePKIXPublicKey(fila.clavePublicaSPKI)
		if errClave != nil {
			return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
				ErrConfiguracionConfianzaDocumentalInvalida
		}
		raiz, errRaiz := nuevaRaizPublicaFijadaAtestacionPDP(
			[]byte(fila.claveID), AlgoritmoCOSEDocumental(fila.algoritmo), clave,
			fila.suite, fila.audienciaDespliegue,
			EstadoConfianzaClaveDocumental(fila.raizEstado), fila.raizValidaDesde,
			fila.raizValidaHasta, time.Time{},
		)
		if errRaiz != nil || raiz.huellaClaveSHA256 != fila.huellaClave {
			return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
				ErrConfiguracionConfianzaDocumentalInvalida
		}
		raices = append(raices, raiz)
	}
	if err = filas.Err(); err != nil || base == nil || materialBase == nil || len(raices) == 0 {
		return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
			ErrConfiguracionConfianzaDocumentalInvalida
	}
	configuracion, err := nuevaConfiguracionConfianzaFijada(
		base.revision, base.configuracionPublicadaEn, base.configuracionExpiraEn,
		raices...,
	)
	if err != nil || configuracion.huellaSHA256 != base.huellaConfiguracion {
		return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
			ErrConfiguracionConfianzaDocumentalInvalida
	}
	if err = tx.Commit(ctx); err != nil {
		return ConfiguracionConfianzaFijada{}, materialEmisorCapacidadDocumentalV4{},
			ErrConfiguracionConfianzaDocumentalInvalida
	}
	return configuracion, *materialBase, nil
}

func mismoMaterialEmisorCapacidadV4(
	primero, segundo materialEmisorCapacidadDocumentalV4,
) bool {
	return primero.claveID == segundo.claveID && primero.version == segundo.version &&
		bytes.Equal(primero.secreto, segundo.secreto) && primero.emisorID == segundo.emisorID &&
		primero.validaDesde.Equal(segundo.validaDesde) &&
		primero.validaHasta.Equal(segundo.validaHasta) && primero.estado == segundo.estado &&
		primero.revocadaEn.Equal(segundo.revocadaEn)
}

func (f *filaConfianzaPostgreSQLV4) normalizarInstantes() {
	f.configuracionPublicadaEn = f.configuracionPublicadaEn.UTC().Truncate(time.Microsecond)
	f.configuracionExpiraEn = f.configuracionExpiraEn.UTC().Truncate(time.Microsecond)
	f.raizValidaDesde = f.raizValidaDesde.UTC().Truncate(time.Microsecond)
	f.raizValidaHasta = f.raizValidaHasta.UTC().Truncate(time.Microsecond)
}

func (f filaConfianzaPostgreSQLV4) validarBase() error {
	if !referenciaConfiguracionDocumentalValida(f.revision) ||
		!huellaSHA256DocumentalValida(f.huellaConfiguracion) ||
		!instanteCanonicoDocumental(f.configuracionPublicadaEn) ||
		!instanteCanonicoDocumental(f.configuracionExpiraEn) ||
		f.configuracionEstado != "activa" || f.configuracionRevocadaEn.Valid ||
		!referenciaDurableAtestacionPDPValida(f.claveID) ||
		f.algoritmo != string(AlgoritmoCOSEDocumentalEdDSA) ||
		f.suite != suiteAtestacionAutorizacionPDPCOSEEdDSAV1 ||
		f.audienciaCOSE != string(AudienciaCOSEAtestacionAutorizacionPDP) ||
		!audienciaDespliegueAtestacionPDPValida(f.audienciaDespliegue) ||
		len(f.clavePublicaSPKI) == 0 ||
		huellaBytesDocumentales(f.clavePublicaSPKI) != f.huellaClave ||
		!instanteCanonicoDocumental(f.raizValidaDesde) ||
		!instanteCanonicoDocumental(f.raizValidaHasta) ||
		f.raizEstado != string(EstadoConfianzaClaveDocumentalActiva) ||
		f.raizRevocadaEn.Valid {
		return ErrConfiguracionConfianzaDocumentalInvalida
	}
	return nil
}

func (f filaConfianzaPostgreSQLV4) mismaConfiguracion(otra filaConfianzaPostgreSQLV4) bool {
	return f.revision == otra.revision &&
		f.huellaConfiguracion == otra.huellaConfiguracion &&
		f.configuracionPublicadaEn.Equal(otra.configuracionPublicadaEn) &&
		f.configuracionExpiraEn.Equal(otra.configuracionExpiraEn) &&
		f.configuracionEstado == otra.configuracionEstado &&
		f.configuracionRevocadaEn.Valid == otra.configuracionRevocadaEn.Valid
}

func (*EjecutorDocumentalPostgreSQLV4) String() string {
	return "[EJECUTOR-DOCUMENTAL-POSTGRESQL-V4-SELLADO]"
}
func (e *EjecutorDocumentalPostgreSQLV4) GoString() string { return e.String() }
func (e *EjecutorDocumentalPostgreSQLV4) LogValue() slog.Value {
	return slog.StringValue(e.String())
}
func (e *EjecutorDocumentalPostgreSQLV4) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, e.String())
}
func (*EjecutorDocumentalPostgreSQLV4) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadEjecucionDocumentalV4Prohibida
}

var _ ports.ConectorEjecucionDocumentalAtestadaV4 = (*EjecutorDocumentalPostgreSQLV4)(nil)
