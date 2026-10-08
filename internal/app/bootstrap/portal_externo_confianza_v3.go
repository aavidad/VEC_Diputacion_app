package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"

	postgresqlcompartido "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/shared/telemetria"
)

// LOGIN nominal y grupo del preflight externo (AD3-112). Las funciones SQL
// rechazan cualquier otro login; aquí se comprueba antes para fallar pronto.
const (
	loginPreflightV3PortalExterno = "vec_externo_preflight_v3_desarrollo"
	rolPreflightV3PortalExterno   = "vec_autorizacion_atestada_v3_preflight_externo"
)

// abrirPoolPreflightV3PortalExterno abre la única conexión con la que el
// proceso externo mira el gobierno V3: solo lectura, TLS verificado y el
// LOGIN nominal con una única membresía en el grupo de preflight externo.
func abrirPoolPreflightV3PortalExterno(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if ctx == nil || dsn == "" {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User != loginPreflightV3PortalExterno ||
		validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	postgresqlcompartido.FijarTamanoPool(c, dsn, 4)
	c.MinConns = 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{
		"application_name": "vec-portal-externo-preflight-v3", "timezone": "UTC", "search_path": "pg_catalog",
		"statement_timeout": "10s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "15s",
		"default_transaction_read_only": "on",
	} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	telemetria.Instrumentar(c) // consultas por petición en el registro de acceso
	pool, err := postgresqlcompartido.NuevoPoolConPreflightTEMP(ctx, c)
	if err != nil {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	var valido bool
	err = pool.QueryRow(ctx, `SELECT session_user=current_user AND session_user=$1
 AND r.rolcanlogin AND r.rolinherit AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
            WHERE m.member=r.oid AND g.rolname=$2 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`, loginPreflightV3PortalExterno, rolPreflightV3PortalExterno).Scan(&valido)
	if err != nil || !valido {
		pool.Close()
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	return pool, nil
}

// leerConfiguracionV3PortalExterno pide al gobierno la configuración vigente
// partiendo de la previa (AD3-112 comprueba claves, raíz y anti-retroceso).
func leerConfiguracionV3PortalExterno(ctx context.Context, pool *pgxpool.Pool, inv inventarioV3PortalExterno, consumidor string, previa configuracionInventarioV3PortalExterno) (configuracionInventarioV3PortalExterno, error) {
	var vacia configuracionInventarioV3PortalExterno
	if pool == nil || !consumidorPortalExternoValido(consumidor) {
		return vacia, ErrMaterialV3PortalExternoInvalido
	}
	material, err := materialJSONV3PortalExterno(inv, consumidor, previa)
	if err != nil {
		return vacia, err
	}
	var respuesta string
	if err := pool.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.leer_configuracion_externa_v1($1,$2::jsonb)::text`,
		consumidor, string(material)).Scan(&respuesta); err != nil || len(respuesta) > 4096 {
		return vacia, ErrMaterialV3PortalExternoInvalido
	}
	var leida struct {
		Revision     string    `json:"revision"`
		Secuencia    uint64    `json:"secuencia"`
		HuellaSHA256 string    `json:"huella_configuracion_sha256"`
		PublicadaEn  time.Time `json:"publicada_en"`
		ExpiraEn     time.Time `json:"expira_en"`
	}
	d := json.NewDecoder(bytes.NewReader([]byte(respuesta)))
	d.DisallowUnknownFields()
	var sobra any
	if d.Decode(&leida) != nil || !errors.Is(d.Decode(&sobra), io.EOF) || leida.Revision == "" ||
		leida.Secuencia < previa.Secuencia || !leida.PublicadaEn.Before(leida.ExpiraEn) {
		return vacia, ErrMaterialV3PortalExternoInvalido
	}
	return configuracionInventarioV3PortalExterno{
		Revision: leida.Revision, Secuencia: leida.Secuencia, HuellaSHA256: leida.HuellaSHA256,
		PublicadaEn: leida.PublicadaEn.UTC(), ExpiraEn: leida.ExpiraEn.UTC(),
	}, nil
}

// comprobarMaterialV3PortalExterno exige que las claves del inventario sean
// las vigentes de su audiencia con la configuración y la raíz publicadas.
func comprobarMaterialV3PortalExterno(ctx context.Context, pool *pgxpool.Pool, inv inventarioV3PortalExterno, consumidor string, vigente configuracionInventarioV3PortalExterno) error {
	material, err := materialJSONV3PortalExterno(inv, consumidor, vigente)
	if err != nil || pool == nil {
		return ErrMaterialV3PortalExternoInvalido
	}
	var aceptado bool
	if err := pool.QueryRow(ctx, `SELECT vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1($1,$2::jsonb)`,
		consumidor, string(material)).Scan(&aceptado); err != nil || !aceptado {
		return ErrMaterialV3PortalExternoInvalido
	}
	return nil
}

// nuevaFuenteConfianzaPortalExterno reutiliza la fuente renovable de la
// composición de desarrollo con dos diferencias: lee con el rol de preflight
// externo y nunca publica. Si la configuración vence y el lado interno aún no
// ha publicado la del día, la autorización falla cerrada hasta que lo haga.
func nuevaFuenteConfianzaPortalExterno(pool *pgxpool.Pool, inv inventarioV3PortalExterno, consumidor string, m materialAtestacionContratacionTemporalDesarrollo, reloj relojConfianzaCTDesarrollo) (*fuenteConfianzaRenovableCTDesarrollo, error) {
	if pool == nil || reloj == nil {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	servicio, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(m.configuracion, reloj)
	if err != nil {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	publica := materialAtestacionContratacionTemporalDesarrollo{
		claveID: m.claveID, claveVersion: m.claveVersion, raiz: m.raiz,
		configuracion: m.configuracion, configuracionRef: m.configuracionRef,
		configuracionOrden: m.configuracionOrden, configuracionHuella: m.configuracionHuella,
		publicadaEn: m.publicadaEn, expiraEn: m.expiraEn, validaDesde: m.validaDesde,
		validaHasta: m.validaHasta, spki: append([]byte(nil), m.spki...), spkiHuella: m.spkiHuella,
	}
	f := &fuenteConfianzaRenovableCTDesarrollo{reloj: reloj, material: publica, actual: servicio,
		leer: func(ctx context.Context, anterior materialAtestacionContratacionTemporalDesarrollo, _ time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
			previa := configuracionInventarioV3PortalExterno{
				Revision: anterior.configuracionRef, Secuencia: anterior.configuracionOrden,
				HuellaSHA256: anterior.configuracionHuella, PublicadaEn: anterior.publicadaEn, ExpiraEn: anterior.expiraEn,
			}
			vigente, err := leerConfiguracionV3PortalExterno(ctx, pool, inv, consumidor, previa)
			if err != nil {
				return materialAtestacionContratacionTemporalDesarrollo{}, err
			}
			actual := anterior
			actual.configuracionRef, actual.configuracionOrden = vigente.Revision, vigente.Secuencia
			actual.configuracionHuella, actual.publicadaEn, actual.expiraEn = vigente.HuellaSHA256, vigente.PublicadaEn, vigente.ExpiraEn
			actual.configuracion, err = confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
				actual.configuracionRef, actual.configuracionOrden, actual.publicadaEn, actual.expiraEn, actual.raiz)
			if err != nil {
				return materialAtestacionContratacionTemporalDesarrollo{}, err
			}
			// La configuración reconstruida con la raíz propia debe tener la
			// huella que publicó el gobierno.
			huella, err := actual.configuracion.HuellaSHA256ParaGobierno()
			if err != nil || huella != vigente.HuellaSHA256 {
				return materialAtestacionContratacionTemporalDesarrollo{}, ErrMaterialV3PortalExternoInvalido
			}
			return actual, nil
		},
		// El proceso externo no publica: renovar solo deja paso a la lectura.
		renovar: func(_ context.Context, anterior materialAtestacionContratacionTemporalDesarrollo, _ time.Time) (materialAtestacionContratacionTemporalDesarrollo, error) {
			return anterior, nil
		}}
	f.lector, err = f.nuevoLector()
	if err != nil {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	return f, nil
}

// nuevosProveedoresV3PortalExterno devuelve, por audiencia, el proveedor de
// material de un consumidor: coteja el inventario con el gobierno vigente y
// comparte una sola fuente de confianza entre sus audiencias.
func nuevosProveedoresV3PortalExterno(ctx context.Context, directorio string, pool *pgxpool.Pool, consumidor string, reloj relojContratacionTemporalDesarrollo) (map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo, error) {
	if ctx == nil || pool == nil || !consumidorPortalExternoValido(consumidor) {
		return nil, ErrMaterialV3PortalExternoInvalido
	}
	inv, err := leerInventarioV3PortalExterno(directorio)
	if err != nil {
		return nil, err
	}
	vigente, err := leerConfiguracionV3PortalExterno(ctx, pool, inv, consumidor, inv.Configuracion)
	if err != nil {
		return nil, err
	}
	if err := comprobarMaterialV3PortalExterno(ctx, pool, inv, consumidor, vigente); err != nil {
		return nil, err
	}
	materiales, err := materialesConsumidorV3PortalExterno(directorio, inv, consumidor, vigente)
	if err != nil {
		return nil, err
	}
	defer borrarMaterialesV3PortalExterno(materiales)
	fuente, err := nuevaFuenteConfianzaPortalExterno(pool, inv, consumidor, materiales[0], reloj)
	if err != nil {
		return nil, err
	}
	proveedores := make(map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo, len(materiales))
	for i := range materiales {
		materiales[i].fuenteConfianza = fuente
		p, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(materiales[i], reloj)
		if err != nil {
			return nil, ErrMaterialV3PortalExternoInvalido
		}
		proveedores[materiales[i].audienciaConsumo] = p
	}
	return proveedores, nil
}
