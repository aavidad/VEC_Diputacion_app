package bootstrap

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	gocose "github.com/veraison/go-cose"

	"vec-diputacion-granada/config"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	rolEjecucionPostgreSQLContratacionTemporalDesarrollo   = "vec_contratacion_temporal_ejecutor"
	rolGobiernoPostgreSQLContratacionTemporalDesarrollo    = "vec_autorizacion_atestada_v3_migrador"
	rolConfirmadorPostgreSQLContratacionTemporalDesarrollo = "vec_contratacion_temporal_confirmador_cobertura"
	rolLectorPostgreSQLContratacionTemporalDesarrollo      = "vec_contratacion_temporal_lector_resultado_cobertura"
	audienciaAtestacionContratacionTemporalDesarrollo      = "vec:desarrollo:contratacion-temporal:atestacion:v3"
	audienciaConsumoAltaContratacionTemporal               = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
)

var errPostgreSQLContratacionTemporalDesarrolloNoDisponible = errors.New(
	"bootstrap: PostgreSQL de contratacion temporal no disponible",
)

type materialAtestacionContratacionTemporalDesarrollo struct {
	claveID             string
	claveVersion        uint64
	privada             ed25519.PrivateKey
	raiz                confianzaatestacion.RaizPublicaAtestacionAutorizacionV3
	configuracion       confianzaatestacion.ConfiguracionConfianzaAtestacionAutorizacionV3
	configuracionRef    string
	configuracionOrden  uint64
	configuracionHuella string
	publicadaEn         time.Time
	expiraEn            time.Time
	validaDesde         time.Time
	validaHasta         time.Time
	spki                []byte
	spkiHuella          string
	claveHMACID         string
	claveHMACVersion    uint64
	claveHMAC           []byte
	claveHMACRevision   uint64
	claveHMACHuella     string
	claveHMACSecreto    string
	emisorID            string
	capacidad           confianzaatestacion.ClaveHMACCapacidadAtestacionV3
}

type dependenciasPostgreSQLContratacionTemporalDesarrollo struct {
	ejecucion       *pgxpool.Pool
	confirmador     *pgxpool.Pool
	lectorResultado *postgrescontratacion.PoolRecuperacionCoberturaO405PostgreSQL
	candidaturas    ports.ResolutorCandidaturaAlta
	transaccionAlta ports.TransaccionAltasCandidata
	cerrarUnaVez    func()
}

func (d *dependenciasPostgreSQLContratacionTemporalDesarrollo) cerrar() {
	if d == nil {
		return
	}
	if d.cerrarUnaVez != nil {
		d.cerrarUnaVez()
	}
}

func nuevasDependenciasPostgreSQLContratacionTemporalDesarrollo(
	cfg config.Config,
	derivador *derivadorIdentidadOperacionDesarrollo,
	soporte *soporteAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (dependenciasPostgreSQLContratacionTemporalDesarrollo, error) {
	vacias := dependenciasPostgreSQLContratacionTemporalDesarrollo{}
	if !cfg.DevelopmentEnabledByDoubleKey() ||
		derivador == nil || !derivador.valido() || soporte == nil {
		return vacias, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	configuracion := cfg.Normalize().ContratacionTemporalPostgreSQL
	dsnEjecucion, dsnGobierno, err := configuracion.DSNSeparados()
	if err != nil {
		return vacias, err
	}
	dsnConfirmador, dsnLectorResultado, err :=
		configuracion.DSNCoberturaSeparados()
	if err != nil {
		return vacias, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(
		derivador, reloj.Ahora(),
	)
	if err != nil {
		return vacias, err
	}
	defer material.borrarCopiasEfimeras()
	gobierno, usuarioGobierno, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(
		ctx, dsnGobierno, "vec-ct-desarrollo-gobierno",
		rolGobiernoPostgreSQLContratacionTemporalDesarrollo,
	)
	if err != nil {
		return vacias, err
	}
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(
		ctx, gobierno, material,
	); err != nil {
		gobierno.Close()
		return vacias, err
	}
	if err := publicarAutoridadPostgreSQLContratacionTemporalDesarrollo(
		ctx, gobierno, soporte,
	); err != nil {
		gobierno.Close()
		return vacias, err
	}
	gobierno.Close()
	ejecucion, usuarioEjecucion, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(
		ctx, dsnEjecucion, "vec-ct-desarrollo-ejecucion",
		rolEjecucionPostgreSQLContratacionTemporalDesarrollo,
	)
	if err != nil || usuarioEjecucion == usuarioGobierno {
		if ejecucion != nil {
			ejecucion.Close()
		}
		return vacias, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	dependencias := dependenciasPostgreSQLContratacionTemporalDesarrollo{
		ejecucion: ejecucion,
	}
	var cierre sync.Once
	dependencias.cerrarUnaVez = func() {
		cierre.Do(func() {
			if dependencias.lectorResultado != nil {
				dependencias.lectorResultado.Cerrar()
			}
			if dependencias.confirmador != nil {
				dependencias.confirmador.Close()
			}
			if dependencias.ejecucion != nil {
				dependencias.ejecucion.Close()
			}
		})
	}
	completa := false
	defer func() {
		if !completa {
			dependencias.cerrar()
		}
	}()
	resolver, err := postgrescontratacion.NuevoResolutorCandidaturaAltaPostgreSQL(ejecucion)
	if err != nil {
		return vacias, err
	}
	proveedor, err := nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(
		material, soporte, reloj,
	)
	if err != nil {
		return vacias, err
	}
	transaccion, err := postgrescontratacion.NuevaTransaccionAltasPostgreSQLCandidata(
		ejecucion, proveedor,
	)
	if err != nil {
		return vacias, err
	}
	confirmador, usuarioConfirmador, err :=
		abrirPoolPostgreSQLContratacionTemporalDesarrollo(
			ctx,
			dsnConfirmador,
			"vec-ct-desarrollo-confirmador",
			rolConfirmadorPostgreSQLContratacionTemporalDesarrollo,
		)
	if err != nil || usuarioConfirmador == usuarioEjecucion ||
		usuarioConfirmador == usuarioGobierno {
		if confirmador != nil {
			confirmador.Close()
		}
		return vacias, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	dependencias.confirmador = confirmador
	inspectorLector, usuarioLector, err :=
		abrirPoolPostgreSQLContratacionTemporalDesarrollo(
			ctx,
			dsnLectorResultado,
			"vec-ct-desarrollo-lector-preflight",
			rolLectorPostgreSQLContratacionTemporalDesarrollo,
		)
	if inspectorLector != nil {
		inspectorLector.Close()
	}
	if err != nil || usuarioLector == usuarioEjecucion ||
		usuarioLector == usuarioGobierno || usuarioLector == usuarioConfirmador {
		return vacias, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	lectorResultado, err :=
		postgrescontratacion.NuevoPoolRecuperacionCoberturaO405PostgreSQL(
			ctx,
			dsnLectorResultado,
		)
	if err != nil {
		return vacias, err
	}
	dependencias.lectorResultado = lectorResultado
	dependencias.candidaturas = resolver
	dependencias.transaccionAlta = transaccion
	completa = true
	return dependencias, nil
}

func abrirPoolPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	dsn string,
	aplicacion string,
	rolEsperado string,
) (*pgxpool.Pool, string, error) {
	configuracion, err := pgxpool.ParseConfig(dsn)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil ||
		validarTLSPostgreSQLBorradores(&configuracion.ConnConfig.Config, true) != nil {
		return nil, "", errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	configuracion.MaxConns = 4
	configuracion.MinConns = 0
	configuracion.ConnConfig.ConnectTimeout = 5 * time.Second
	if configuracion.ConnConfig.RuntimeParams == nil {
		configuracion.ConnConfig.RuntimeParams = make(map[string]string)
	}
	parametros := configuracion.ConnConfig.RuntimeParams
	parametros["application_name"] = aplicacion
	parametros["timezone"] = "UTC"
	parametros["search_path"] = "pg_catalog,pg_temp"
	parametros["default_transaction_isolation"] = "serializable"
	parametros["default_transaction_read_only"] = "off"
	parametros["statement_timeout"] = "15s"
	parametros["lock_timeout"] = "3s"
	parametros["idle_in_transaction_session_timeout"] = "20s"
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		return nil, "", errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	usuario, err := comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, rolEsperado,
	)
	if err != nil {
		pool.Close()
		return nil, "", err
	}
	return pool, usuario, nil
}

func comprobarIdentidadPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	consultador interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	rolEsperado string,
) (string, error) {
	if ctx == nil || consultador == nil ||
		(rolEsperado != rolEjecucionPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolGobiernoPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolConfirmadorPostgreSQLContratacionTemporalDesarrollo &&
			rolEsperado != rolLectorPostgreSQLContratacionTemporalDesarrollo) {
		return "", errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	var usuario string
	var valido bool
	err := consultador.QueryRow(ctx, `
		SELECT session_user::text,
		       session_user = current_user
		       AND identidad.rolcanlogin
		       AND identidad.rolinherit
		       AND NOT identidad.rolsuper
		       AND NOT identidad.rolcreatedb
		       AND NOT identidad.rolcreaterole
		       AND NOT identidad.rolreplication
		       AND NOT identidad.rolbypassrls
		       AND pg_catalog.pg_has_role(session_user, $1, 'MEMBER')
		  FROM pg_catalog.pg_roles AS identidad
		 WHERE identidad.rolname = session_user`, rolEsperado).Scan(&usuario, &valido)
	if err != nil || !valido || usuario == "" {
		return "", errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return usuario, nil
}

func nuevoMaterialAtestacionContratacionTemporalDesarrollo(
	derivador *derivadorIdentidadOperacionDesarrollo,
	ahora time.Time,
) (materialAtestacionContratacionTemporalDesarrollo, error) {
	vacio := materialAtestacionContratacionTemporalDesarrollo{}
	if derivador == nil || !derivador.valido() {
		return vacio, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	resultados, err := derivador.calcularHMAC(
		[]byte("vec.ct.desarrollo.atestacion-v3.ed25519.v1"),
		[]byte("vec.ct.desarrollo.atestacion-v3.capacidad-hmac.v1"),
	)
	if err != nil || len(resultados) == 0 {
		return vacio, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer borrarResultadosHMACIdempotenciaDesarrollo(resultados)
	activo := resultados[0]
	semilla := append([]byte(nil), activo.localizador[:]...)
	defer borrarBytes(semilla)
	privada := ed25519.NewKeyFromSeed(semilla)
	publica := privada.Public().(ed25519.PublicKey)
	claveID := "clave:atestacion:ct:desarrollo:v" + numeroDecimal(activo.generacion)
	claveHMACID := "clave:capacidad:ct:desarrollo:v" + numeroDecimal(activo.generacion)
	validaDesde := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	validaHasta := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	ahora = ahora.UTC().Truncate(time.Microsecond)
	if ahora.Before(validaDesde) || !ahora.Before(validaHasta) {
		return vacio, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	publicadaEn := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, time.UTC)
	expiraEn := publicadaEn.Add(24 * time.Hour)
	secuencia := uint64(ahora.Year()*10000 + int(ahora.Month())*100 + ahora.Day())
	configuracionRef := "confianza:atestacion:ct:desarrollo:" + publicadaEn.Format("2006-01-02")
	raiz, err := confianzaatestacion.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
		claveID, 1, publica, audienciaAtestacionContratacionTemporalDesarrollo,
		confianzaatestacion.EstadoClaveAtestacionAutorizacionV3Activa,
		validaDesde, validaHasta, time.Time{},
	)
	if err != nil {
		return vacio, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	configuracion, err := confianzaatestacion.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
		configuracionRef, secuencia, publicadaEn, expiraEn, raiz,
	)
	if err != nil {
		return vacio, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	huellaConfiguracion, err := configuracion.HuellaSHA256ParaGobierno()
	if err != nil {
		return vacio, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		return vacio, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	huellaSPKI := sha256.Sum256(spki)
	claveHMAC := append([]byte(nil), activo.huellaSolicitud[:]...)
	huellaSecreto := sha256.Sum256(claveHMAC)
	calculadorGobierno := sha256.New()
	_, _ = calculadorGobierno.Write([]byte("vec.ct.desarrollo.capacidad-v3.gobierno.v1\x00"))
	_, _ = calculadorGobierno.Write(claveHMAC)
	huellaGobierno := hex.EncodeToString(calculadorGobierno.Sum(nil))
	emisorID := "emisor:ct:desarrollo:v" + numeroDecimal(activo.generacion)
	capacidad, err := confianzaatestacion.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(
		claveHMACID, 1, claveHMAC, emisorID,
		audienciaConsumoAltaContratacionTemporal,
		confianzaatestacion.EstadoClaveHMACCapacidadAtestacionV3Emision,
		validaDesde, validaHasta, time.Time{}, 1, huellaGobierno,
	)
	if err != nil {
		borrarBytes(claveHMAC)
		return vacio, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return materialAtestacionContratacionTemporalDesarrollo{
		claveID: claveID, claveVersion: 1, privada: privada,
		raiz: raiz, configuracion: configuracion,
		configuracionRef: configuracionRef, configuracionOrden: secuencia,
		configuracionHuella: huellaConfiguracion,
		publicadaEn:         publicadaEn, expiraEn: expiraEn,
		validaDesde: validaDesde, validaHasta: validaHasta,
		spki: spki, spkiHuella: hex.EncodeToString(huellaSPKI[:]),
		claveHMACID: claveHMACID, claveHMACVersion: 1, claveHMAC: claveHMAC,
		claveHMACRevision: 1, claveHMACHuella: huellaGobierno,
		claveHMACSecreto: hex.EncodeToString(huellaSecreto[:]),
		emisorID:         emisorID, capacidad: capacidad,
	}, nil
}

func numeroDecimal(valor uint32) string {
	const digitos = "0123456789"
	if valor == 0 {
		return "0"
	}
	var buffer [10]byte
	indice := len(buffer)
	for valor > 0 {
		indice--
		buffer[indice] = digitos[valor%10]
		valor /= 10
	}
	return string(buffer[indice:])
}

func (m *materialAtestacionContratacionTemporalDesarrollo) borrarCopiasEfimeras() {
	if m == nil {
		return
	}
	borrarBytes(m.claveHMAC)
	borrarBytes(m.spki)
	borrarBytes(m.privada)
}

func publicarGobiernoAtestacionContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	material materialAtestacionContratacionTemporalDesarrollo,
) error {
	if ctx == nil || pool == nil || len(material.claveHMAC) == 0 || len(material.spki) == 0 {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	consultas := []struct {
		sql  string
		args []any
	}{
		{`SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario`, nil},
		{`INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
		  (clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,
		   huella_secreto_sha256,emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
		  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT DO NOTHING`,
			[]any{material.claveHMACID, material.claveHMACVersion, material.claveHMACRevision,
				material.claveHMACHuella, material.claveHMAC, material.claveHMACSecreto,
				material.emisorID, audienciaConsumoAltaContratacionTemporal,
				material.validaDesde, material.validaHasta, "acto:ct:desarrollo:clave-capacidad:v1"}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
		  (orden,clave_id,version,establecida_en,acto_ref)
		  VALUES (1,$1,$2,$3,$4) ON CONFLICT DO NOTHING`,
			[]any{material.claveHMACID, material.claveHMACVersion, material.validaDesde,
				"acto:ct:desarrollo:puntero-clave:v1"}},
		{`INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
		  (clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,
		   valida_hasta,suite,audiencia_despliegue,acto_ref)
		  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			[]any{material.claveID, material.claveVersion, material.spki, material.spkiHuella,
				material.validaDesde, material.validaHasta,
				confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
				audienciaAtestacionContratacionTemporalDesarrollo,
				"acto:ct:desarrollo:raiz-atestacion:v1"}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
		  (revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
		  VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
			[]any{material.configuracionRef, material.configuracionOrden,
				material.configuracionHuella, material.publicadaEn, material.expiraEn,
				"acto:ct:desarrollo:configuracion:" + material.publicadaEn.Format("2006-01-02")}},
		{`INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz
		  (configuracion_revision,raiz_clave_id,raiz_version)
		  VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
			[]any{material.configuracionRef, material.claveID, material.claveVersion}},
		{`INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
		  (orden,configuracion_revision,establecida_en,acto_ref)
		  VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
			[]any{material.configuracionOrden, material.configuracionRef, material.publicadaEn,
				"acto:ct:desarrollo:puntero-configuracion:" + material.publicadaEn.Format("2006-01-02")}},
	}
	for _, consulta := range consultas {
		if _, err := tx.Exec(ctx, consulta.sql, consulta.args...); err != nil {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
	}
	var coincide bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
		 WHERE clave_id=$1 AND version=$2 AND revision_gobierno=$3
		   AND huella_gobierno_sha256=$4 AND secreto_hmac=$5
		   AND huella_secreto_sha256=$6 AND emisor_id=$7
		   AND audiencia_consumo=$8 AND valida_desde=$9 AND valida_hasta=$10)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.raiz_confianza_version
		 WHERE clave_id=$11 AND version=$12 AND clave_publica_spki=$13
		   AND huella_spki_sha256=$14 AND valida_desde=$9 AND valida_hasta=$10
		   AND suite=$15 AND audiencia_despliegue=$16)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version
		 WHERE revision=$17 AND secuencia=$18 AND huella_configuracion_sha256=$19
		   AND publicada_en=$20 AND expira_en=$21)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_raiz
		 WHERE configuracion_revision=$17 AND raiz_clave_id=$11 AND raiz_version=$12)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual
		 WHERE orden=$18 AND configuracion_revision=$17)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_clave_emision
		 WHERE orden=1 AND clave_id=$1 AND version=$2)`,
		material.claveHMACID, material.claveHMACVersion, material.claveHMACRevision,
		material.claveHMACHuella, material.claveHMAC, material.claveHMACSecreto,
		material.emisorID, audienciaConsumoAltaContratacionTemporal,
		material.validaDesde, material.validaHasta,
		material.claveID, material.claveVersion, material.spki, material.spkiHuella,
		confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
		audienciaAtestacionContratacionTemporalDesarrollo,
		material.configuracionRef, material.configuracionOrden,
		material.configuracionHuella, material.publicadaEn, material.expiraEn,
	).Scan(&coincide)
	if err != nil || !coincide {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

type firmanteAtestacionAltaContratacionTemporalDesarrollo struct {
	claveID string
	privada ed25519.PrivateKey
	reloj   relojContratacionTemporalDesarrollo
}

func (f *firmanteAtestacionAltaContratacionTemporalDesarrollo) FirmarAtestacionAutorizacionV3(
	ctx context.Context,
	solicitud puertosvec.SolicitudFirmaAtestacionAutorizacionV3,
) (puertosvec.ResultadoFirmaAtestacionAutorizacionV3, error) {
	if ctx == nil || f == nil || len(f.privada) != ed25519.PrivateKeySize {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	cabecera, err := solicitud.Cabecera()
	if err != nil || cabecera.ClaveID != f.claveID ||
		cabecera.Audiencia != audienciaAtestacionContratacionTemporalDesarrollo {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	mensaje, err := solicitud.Mensaje()
	if err != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	defer borrarBytes(mensaje)
	aad, err := confianzaatestacion.AADExternoAtestacionAutorizacionV3(cabecera.Audiencia)
	if err != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	sobre := gocose.NewSign1Message()
	sobre.Headers.Protected.SetAlgorithm(gocose.AlgorithmEdDSA)
	sobre.Headers.Protected[gocose.HeaderLabelKeyID] = []byte(f.claveID)
	sobre.Payload = append([]byte(nil), mensaje...)
	firmante, err := gocose.NewSigner(gocose.AlgorithmEdDSA, f.privada)
	if err != nil || sobre.Sign(rand.Reader, aad, firmante) != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	sobre.Payload = nil
	sobre.Headers.RawProtected = nil
	sobre.Headers.RawUnprotected = nil
	firma, err := sobre.MarshalCBOR()
	if err != nil {
		return puertosvec.ResultadoFirmaAtestacionAutorizacionV3{},
			puertosvec.ErrFirmaAtestacionNoDisponible
	}
	defer borrarBytes(firma)
	huella := sha256.Sum256(mensaje)
	return puertosvec.NuevoResultadoFirmaAtestacionAutorizacionV3(
		solicitud, firma, "evidencia:firma:ct:desarrollo:"+hex.EncodeToString(huella[:8]),
		f.reloj.Ahora(),
	)
}

type proveedorMaterialAltaContratacionTemporalDesarrollo struct {
	atestador *aplicacionvec.ServicioAtestacionesAutorizacionV3
	confianza *confianzaatestacion.ServicioConfianzaAtestacionAutorizacionV3
	emisor    *confianzaatestacion.EmisorCapacidadesAtestacionAutorizacionV3
	raiz      confianzaatestacion.RaizPublicaAtestacionAutorizacionV3
	contexto  dominiovec.ResultadoContextoActorRegistradoV2
	motivo    dominiovec.ReferenciaEntradaCatalogo
}

func nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(
	material materialAtestacionContratacionTemporalDesarrollo,
	soporte *soporteAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	if soporte == nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	firmante := &firmanteAtestacionAltaContratacionTemporalDesarrollo{
		claveID: material.claveID,
		privada: append(ed25519.PrivateKey(nil), material.privada...),
		reloj:   reloj,
	}
	atestador, err := aplicacionvec.NuevoServicioAtestacionesAutorizacionV3(
		dominiovec.CabeceraAtestacionAutorizacionV3{
			FormatoVersion: dominiovec.VersionFormatoAtestacionAutorizacionV3,
			Suite:          confianzaatestacion.SuiteAtestacionAutorizacionV3COSEEdDSA,
			ClaveID:        material.claveID,
			Audiencia:      audienciaAtestacionContratacionTemporalDesarrollo,
		},
		firmante,
	)
	if err != nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	confianza, err := confianzaatestacion.NuevoServicioConfianzaAtestacionAutorizacionV3(
		material.configuracion, reloj,
	)
	if err != nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	emisor, err := confianzaatestacion.NuevoEmisorCapacidadesAtestacionAutorizacionV3(
		material.capacidad, reloj,
	)
	if err != nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	contexto, err := soporte.contexto.Resultado.Clonar()
	if err != nil {
		return nil, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return &proveedorMaterialAltaContratacionTemporalDesarrollo{
		atestador: atestador, confianza: confianza, emisor: emisor,
		raiz: material.raiz, contexto: contexto, motivo: soporte.motivo,
	}, nil
}

func (p *proveedorMaterialAltaContratacionTemporalDesarrollo) ProveerMaterialConfirmacionAlta(
	ctx context.Context,
	orden ports.OrdenConfirmarAltaCandidata,
) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if ctx == nil || p == nil || p.atestador == nil || p.confianza == nil || p.emisor == nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	datos, err := orden.Datos()
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	contexto, err := p.contexto.Clonar()
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	datosSolicitud, err := datos.SolicitudAutorizacionV3.Datos()
	ordenConcesion, errOrden := puertosvec.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(
		datos.SolicitudAutorizacionV3, datos.DecisionAutorizacionV3,
		p.motivo, contexto,
	)
	if err != nil || errOrden != nil || datosSolicitud.ReferenciaMotivo != p.motivo ||
		datosSolicitud.VinculoAutenticacionActor.ValidarPara(contexto) != nil ||
		datos.ConfirmacionRegistroV3.ValidarPara(ordenConcesion) != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	atestacion, err := p.atestador.Atestar(
		ctx, datos.DecisionAutorizacionV3, p.motivo, contexto,
	)
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	prueba, err := p.confianza.Verificar(
		ctx, datos.SolicitudAutorizacionV3, datos.DecisionAutorizacionV3,
		p.motivo, contexto, atestacion,
	)
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	capacidad, err := p.emisor.Emitir(
		ctx, datos.SolicitudAutorizacionV3, datos.DecisionAutorizacionV3,
		p.motivo, contexto, atestacion, prueba,
	)
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	material, err := confianzaatestacion.NuevoMaterialConsumoAutorizacionAtestadaV3(
		datos.SolicitudAutorizacionV3, datos.DecisionAutorizacionV3,
		p.motivo, contexto, atestacion, prueba, capacidad, p.raiz,
	)
	if err != nil {
		return vacio, ports.ErrPersistenciaNoDisponible
	}
	return material.ExportarMaterialParaConsumidor()
}
