package bootstrap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
)

type publicacionGobiernoPortalExterno struct {
	Configuracion struct {
		Revision    string    `json:"revision"`
		Secuencia   uint64    `json:"secuencia"`
		Huella      string    `json:"huella_configuracion_sha256"`
		PublicadaEn time.Time `json:"publicada_en"`
		ExpiraEn    time.Time `json:"expira_en"`
	} `json:"configuracion"`
	Raiz struct {
		ClaveID     string    `json:"clave_id"`
		Version     uint64    `json:"version"`
		SPKI        string    `json:"clave_publica_spki_hex"`
		Huella      string    `json:"huella_spki_sha256"`
		ValidaDesde time.Time `json:"valida_desde"`
		ValidaHasta time.Time `json:"valida_hasta"`
		Suite       string    `json:"suite"`
		Audiencia   string    `json:"audiencia_despliegue"`
	} `json:"raiz"`
	Claves []clavePublicacionGobiernoPortalExterno `json:"claves"`
}

type clavePublicacionGobiernoPortalExterno struct {
	ClaveID        string    `json:"clave_id"`
	Version        uint64    `json:"version"`
	Revision       uint64    `json:"revision_gobierno"`
	HuellaGobierno string    `json:"huella_gobierno_sha256"`
	Secreto        string    `json:"secreto_hmac_hex,omitempty"`
	HuellaSecreto  string    `json:"huella_secreto_sha256"`
	EmisorID       string    `json:"emisor_id"`
	Audiencia      string    `json:"audiencia_consumo"`
	ValidaDesde    time.Time `json:"valida_desde"`
	ValidaHasta    time.Time `json:"valida_hasta"`
	Orden          uint64    `json:"orden"`
}

// Se prepara y publica bajo el mismo cerrojo y la misma autoridad Gobierno.
// Sin aprobación solo se devuelve el resumen público para el operador; no se
// publica una configuración, clave, raíz o permiso al arrancar el servidor.
func prepararPublicacionV3PortalExterno(ctx context.Context, pool *pgxpool.Pool, materiales map[string][]materialAtestacionContratacionTemporalDesarrollo, o OpcionesPreparacionPortalExterno) (ResumenPreparacionPortalExterno, error) {
	var resumen ResumenPreparacionPortalExterno
	if ctx == nil || pool == nil || (o.HuellaAprobacionSHA256 == "") != (o.PreimagenSHA256 == "") {
		return resumen, ErrPreparacionPortalExternoInvalida
	}
	err := ejecutarTransaccionGobiernoCTDesarrolloUnaVez(ctx, pool, func(tx pgx.Tx) error {
		var texto string
		if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.leer_estado_publicacion_externa_v1()::text`).Scan(&texto); err != nil || len(texto) > tamanoMaximoInventarioV3Externo {
			return ErrPreparacionPortalExternoInvalida
		}
		var estado struct {
			Preimagen string                            `json:"preimagen_sha256"`
			Secuencia uint64                            `json:"configuracion_secuencia_siguiente"`
			Version   uint64                            `json:"clave_version_siguiente"`
			Revision  uint64                            `json:"revision_gobierno_siguiente"`
			Orden     uint64                            `json:"clave_orden_siguiente"`
			Actual    *publicacionGobiernoPortalExterno `json:"publicacion_externa_actual"`
		}
		if json.Unmarshal([]byte(texto), &estado) != nil || estado.Secuencia < 1 || estado.Version < 1 || estado.Revision < 1 || estado.Orden < 1 {
			return ErrPreparacionPortalExternoInvalida
		}
		if estado.Actual != nil {
			primera := materiales[consumidoresPortalExternoV3[0]]
			if len(primera) == 0 {
				return ErrPreparacionPortalExternoInvalida
			}
			if primera[0].spkiHuella != estado.Actual.Raiz.Huella && o.RotarRaizPreimagenSHA256 != estado.Actual.Raiz.Huella {
				return ErrPreparacionPortalExternoInvalida
			}
		}
		publicacion, err := componerPublicacionGobiernoPortalExterno(ctx, tx, materiales, estado.Actual, estado.Secuencia, estado.Version, estado.Revision, estado.Orden)
		if err != nil {
			return err
		}
		contenido, err := json.Marshal(publicacion)
		if err != nil {
			return ErrPreparacionPortalExternoInvalida
		}
		defer clear(contenido)
		var propuesta struct {
			Huella    string `json:"huella_aprobacion_sha256"`
			Preimagen string `json:"preimagen_sha256"`
		}
		if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.preparar_publicacion_externa_v1($1::jsonb)::text`, string(contenido)).Scan(&texto); err != nil || json.Unmarshal([]byte(texto), &propuesta) != nil || propuesta.Huella == "" || propuesta.Preimagen == "" {
			return ErrPreparacionPortalExternoInvalida
		}
		resumen.Configuracion = publicacion.Configuracion.Revision
		resumen.HuellaAprobacionSHA256, resumen.PreimagenSHA256 = propuesta.Huella, propuesta.Preimagen
		if o.HuellaAprobacionSHA256 == "" {
			resumen.PendientePublicacion = true
			return nil
		}
		if o.HuellaAprobacionSHA256 != propuesta.Huella {
			return ErrPreparacionPortalExternoInvalida
		}
		if err := tx.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.publicar_confianza_externa_v1($1::jsonb,$2::text,$3::text)::text`, string(contenido), o.HuellaAprobacionSHA256, o.PreimagenSHA256).Scan(&texto); err != nil {
			return ErrPreparacionPortalExternoInvalida
		}
		return nil
	})
	return resumen, err
}

// Las coordenadas se recuperan del material exacto, nunca por coincidencia
// de audiencia con una clave interna. Los números nuevos se asignan en la
// transacción gobernada y una clave histórica ya sustituida no se revive.
func componerPublicacionGobiernoPortalExterno(ctx context.Context, tx pgx.Tx, materiales map[string][]materialAtestacionContratacionTemporalDesarrollo, actual *publicacionGobiernoPortalExterno, secuencia, version, revision, orden uint64) (publicacionGobiernoPortalExterno, error) {
	var p publicacionGobiernoPortalExterno
	var base *materialAtestacionContratacionTemporalDesarrollo
	misma := actual != nil
	for _, consumidor := range consumidoresPortalExternoV3 {
		lista := materiales[consumidor]
		if len(lista) != len(audienciasConsumidorPortalExternoV3(consumidor)) {
			return p, ErrPreparacionPortalExternoInvalida
		}
		for i := range lista {
			m := &lista[i]
			if base == nil {
				base = m
			}
			var claveVersion, claveRevision uint64
			err := tx.QueryRow(ctx, `SELECT version,revision_gobierno FROM vec_autorizacion_atestada_v3.clave_capacidad_version
 WHERE clave_id=$1 AND huella_gobierno_sha256=$2 AND secreto_hmac=$3 AND huella_secreto_sha256=$4
 AND emisor_id=$5 AND audiencia_consumo=$6 AND valida_desde=$7 AND valida_hasta=$8`, m.claveHMACID, m.claveHMACHuella, m.claveHMAC, m.claveHMACSecreto, m.emisorID, m.audienciaConsumo, m.validaDesde, m.validaHasta).Scan(&claveVersion, &claveRevision)
			if errors.Is(err, pgx.ErrNoRows) {
				m.claveHMACVersion, m.claveHMACRevision, m.claveHMACOrden = version, revision, orden
				version++
				revision++
				orden++
				misma = false
			} else if err != nil {
				return p, ErrPreparacionPortalExternoInvalida
			} else {
				var vigente *clavePublicacionGobiernoPortalExterno
				if actual != nil {
					for j := range actual.Claves {
						c := &actual.Claves[j]
						if c.ClaveID == m.claveHMACID && c.Version == claveVersion && c.Revision == claveRevision && c.Audiencia == m.audienciaConsumo && c.HuellaSecreto == m.claveHMACSecreto {
							vigente = c
						}
					}
				}
				if vigente == nil {
					return p, ErrPreparacionPortalExternoInvalida
				}
				m.claveHMACVersion, m.claveHMACRevision, m.claveHMACOrden = claveVersion, claveRevision, vigente.Orden
			}
			p.Claves = append(p.Claves, clavePublicacionGobiernoPortalExterno{m.claveHMACID, m.claveHMACVersion, m.claveHMACRevision, m.claveHMACHuella, hex.EncodeToString(m.claveHMAC), m.claveHMACSecreto, m.emisorID, m.audienciaConsumo, m.validaDesde, m.validaHasta, m.claveHMACOrden})
		}
	}
	if base == nil {
		return p, ErrPreparacionPortalExternoInvalida
	}
	var raizVersion uint64
	err := tx.QueryRow(ctx, `SELECT version FROM vec_autorizacion_atestada_v3.raiz_confianza_version WHERE clave_id=$1 AND clave_publica_spki=$2 AND huella_spki_sha256=$3 AND valida_desde=$4 AND valida_hasta=$5 AND audiencia_despliegue=$6 AND suite=$7`, base.claveID, base.spki, base.spkiHuella, base.validaDesde, base.validaHasta, audienciaAtestacionContratacionTemporalDesarrollo, confianza.SuiteAtestacionAutorizacionV3COSEEdDSA).Scan(&raizVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		raizVersion = 1
		misma = false
	} else if err != nil {
		return p, ErrPreparacionPortalExternoInvalida
	}
	if actual != nil && (actual.Raiz.ClaveID != base.claveID || actual.Raiz.Version != raizVersion || !actual.Configuracion.PublicadaEn.Equal(base.publicadaEn) || !actual.Configuracion.ExpiraEn.Equal(base.expiraEn)) {
		misma = false
	}
	p.Raiz.ClaveID, p.Raiz.Version = base.claveID, raizVersion
	p.Raiz.SPKI, p.Raiz.Huella = hex.EncodeToString(base.spki), base.spkiHuella
	p.Raiz.ValidaDesde, p.Raiz.ValidaHasta = base.validaDesde, base.validaHasta
	p.Raiz.Suite, p.Raiz.Audiencia = confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, audienciaAtestacionContratacionTemporalDesarrollo
	p.Configuracion.Revision = "confianza:atestacion:externo:r" + numeroDecimal64(secuencia)
	p.Configuracion.Secuencia, p.Configuracion.PublicadaEn, p.Configuracion.ExpiraEn = secuencia, base.publicadaEn, base.expiraEn
	if misma {
		p.Configuracion = actual.Configuracion
	}
	for _, consumidor := range consumidoresPortalExternoV3 {
		for i := range materiales[consumidor] {
			m := &materiales[consumidor][i]
			m.claveVersion = raizVersion
			m.configuracionRef, m.configuracionOrden = p.Configuracion.Revision, p.Configuracion.Secuencia
			m.publicadaEn, m.expiraEn = p.Configuracion.PublicadaEn, p.Configuracion.ExpiraEn
			if err := reconstruirClavesGobiernoPostgreSQLContratacionTemporalDesarrollo(m); err != nil {
				return p, ErrPreparacionPortalExternoInvalida
			}
			m.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(m.configuracionRef, m.configuracionOrden, m.publicadaEn, m.expiraEn, m.raiz)
			if err != nil {
				return p, ErrPreparacionPortalExternoInvalida
			}
			m.configuracionHuella, err = m.configuracion.HuellaSHA256ParaGobierno()
			if err != nil {
				return p, ErrPreparacionPortalExternoInvalida
			}
			p.Configuracion.Huella = m.configuracionHuella
		}
	}
	return p, nil
}
