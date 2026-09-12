// Package postgres persiste la configuración SMTP administrativa.
package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrConfiguracionCorreoNoDisponible = errors.New("administracion: persistencia de configuracion de correo no disponible")

const referenciaConfiguracionCorreo = "configuracion:smtp:diputacion"

// ProtectorSecretoCorreo representa una clave privada fuera de PostgreSQL.
// El adaptador nunca acepta texto cifrado creado por HTTP ni implementa
// criptografía: la composición debe aportar un protector KMS/HSM apto.
type ProtectorSecretoCorreo interface {
	CifrarSecretoCorreo(context.Context, []byte, []byte) (SobreSecretoCorreo, error)
	ConSecretoCorreoDescifrado(context.Context, SobreSecretoCorreo, []byte, func([]byte) error) error
}

// SobreSecretoCorreo contiene únicamente material cifrado y metadatos no
// secretos. La versión de secreto es independiente de la de configuración:
// una actualización sin secreto conserva el AAD original descifrable.
type SobreSecretoCorreo struct {
	Version  uint64
	ClaveRef string
	Nonce    []byte
	Cifrado  []byte
}

func (s SobreSecretoCorreo) valido() bool {
	return s.Version > 0 && s.ClaveRef != "" && len(s.ClaveRef) <= 512 &&
		len(s.Nonce) >= 12 && len(s.Nonce) <= 64 && len(s.Cifrado) >= 16 && len(s.Cifrado) <= 32768
}

// AutorizadorUsoSecretoCorreo pertenece a la composición del transporte. La
// interfaz administrativa y la UI no reciben esta capacidad.
type AutorizadorUsoSecretoCorreo interface {
	AutorizarUsoSecretoCorreo(context.Context) error
}

type iniciadorConfiguracionCorreo interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type RegistroConfiguracionCorreoPostgreSQL struct {
	pool      iniciadorConfiguracionCorreo
	protector ProtectorSecretoCorreo
}

var _ adminports.RegistroConfiguracionCorreo = (*RegistroConfiguracionCorreoPostgreSQL)(nil)
var _ adminports.PreparadorConfiguracionCorreo = (*RegistroConfiguracionCorreoPostgreSQL)(nil)

func NuevoRegistroConfiguracionCorreoPostgreSQL(pool *pgxpool.Pool, protector ProtectorSecretoCorreo) (*RegistroConfiguracionCorreoPostgreSQL, error) {
	if dependenciaNula(pool) || dependenciaNula(protector) {
		return nil, ErrConfiguracionCorreoNoDisponible
	}
	return &RegistroConfiguracionCorreoPostgreSQL{pool: pool, protector: protector}, nil
}

func (r *RegistroConfiguracionCorreoPostgreSQL) LeerConfiguracionCorreo(ctx context.Context) (admindomain.VistaConfiguracionCorreo, error) {
	if err := r.valido(ctx); err != nil {
		return admindomain.VistaConfiguracionCorreo{}, err
	}
	tx, err := r.iniciar(ctx, pgx.ReadOnly)
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var datos []byte
	err = tx.QueryRow(ctx, `SELECT vec_administracion.leer_configuracion_correo_v1()::text`).Scan(&datos)
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	defer borrar(datos)
	vista, err := vistaDesdeJSON(datos)
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return admindomain.VistaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	return vista, nil
}

func (r *RegistroConfiguracionCorreoPostgreSQL) PrepararConfiguracionCorreo(ctx context.Context, entrada admindomain.ActualizacionConfiguracionCorreo, auditoria vecdomain.AuditEntry) (adminports.PreparacionConfiguracionCorreo, error) {
	if r.valido(ctx) != nil || entrada.Validar() != nil || auditoriaInvalida(auditoria) {
		return adminports.PreparacionConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
	}
	// Se construye un nuevo sobre sólo cuando la sustitución fue explícita. Un
	// nil nunca borra ni intenta descifrar el secreto ya persistido.
	sobre := SobreSecretoCorreo{}
	sustituir := entrada.SecretoNuevo != nil
	var err error
	if sustituir {
		if entrada.VersionEsperada == ^uint64(0) {
			return adminports.PreparacionConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
		}
		versionSecreto := entrada.VersionEsperada + 1
		aad, e := aadConfiguracionCorreo(versionSecreto)
		if e != nil {
			return adminports.PreparacionConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
		}
		defer borrar(aad)
		e = entrada.SecretoNuevo.Consumir(func(claro []byte) error {
			defer borrar(claro)
			sobre, err = r.protector.CifrarSecretoCorreo(ctx, claro, aad)
			return err
		})
		if e != nil || err != nil || !sobre.valido() || sobre.Version != versionSecreto {
			return adminports.PreparacionConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
		}
	}
	huellaAAD := ""
	if sustituir {
		aad, e := aadConfiguracionCorreo(sobre.Version)
		if e != nil {
			return adminports.PreparacionConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
		}
		h := sha256.Sum256(aad)
		borrar(aad)
		huellaAAD = hex.EncodeToString(h[:])
	}
	entradaPreparada := entrada
	// El valor claro ya fue consumido y borrado por el protector. Ninguna fase
	// posterior recibe siquiera el wrapper que podría retenerlo.
	entradaPreparada.SecretoNuevo = nil
	preparacion := adminports.PreparacionConfiguracionCorreo{Entrada: entradaPreparada, SobreNuevo: adminports.SobreSecretoConfiguracionCorreo{Version: sobre.Version, ClaveRef: sobre.ClaveRef, Nonce: append([]byte(nil), sobre.Nonce...), Cifrado: append([]byte(nil), sobre.Cifrado...)}, Sustituir: sustituir, HuellaAADSHA256: huellaAAD}
	payload, e := adminapp.PayloadNegocioConfiguracionCorreo(preparacion, auditoria)
	if e != nil {
		return adminports.PreparacionConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
	}
	preparacion.PayloadNegocio = payload
	return preparacion, nil
}

// GuardarConfiguracionCorreo es la frontera durable del flujo
// Preparar→Autorizar→Guardar. No acepta texto claro ni reconstruye la orden
// desde HTTP; la composición debe entregar exactamente el payload firmado.
func (r *RegistroConfiguracionCorreoPostgreSQL) GuardarConfiguracionCorreo(ctx context.Context, orden adminports.OrdenConfiguracionCorreoAutorizada) (admindomain.VistaConfiguracionCorreo, error) {
	if r.valido(ctx) != nil || orden.Preparacion.Entrada.Validar() != nil || len(orden.Preparacion.PayloadNegocio) == 0 || len(orden.Preparacion.PayloadNegocio) > 65536 || orden.Material.ValidarEstructura() != nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
	}
	defer borrar(orden.Preparacion.PayloadNegocio)
	tx, err := r.iniciar(ctx, pgx.ReadWrite)
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var respuesta []byte
	err = tx.QueryRow(ctx, `SELECT vec_administracion.guardar_configuracion_correo_v1($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)::text`,
		orden.Preparacion.PayloadNegocio, orden.Material.CapacidadCanonica(), orden.Material.DecisionCanonica(), orden.Material.MotivoCanonico(), orden.Material.ContextoActorCanonico(), int64(orden.Material.PersonaVersion()), int64(orden.Material.PerfilVersion()), orden.Material.PayloadVECAD3(), orden.Material.SobreCOSESign1(), orden.Material.EvidenciaVerificacion(), orden.Material.RaizPublicaSPKI(),
	).Scan(&respuesta)
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	defer borrar(respuesta)
	vista, err := vistaDesdeJSON(respuesta)
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return admindomain.VistaConfiguracionCorreo{}, normalizarError(ctx, err)
	}
	return vista, nil
}

// UsarSecretoCorreo es la única salida protegida para un transporte compuesto
// y autorizado. El callback no puede devolver el secreto y recibe un buffer
// que se borra al terminar.
func (r *RegistroConfiguracionCorreoPostgreSQL) UsarSecretoCorreo(ctx context.Context, autorizador AutorizadorUsoSecretoCorreo, usar func([]byte) error) error {
	if r.valido(ctx) != nil || dependenciaNula(autorizador) || autorizador.AutorizarUsoSecretoCorreo(ctx) != nil || usar == nil {
		return ErrConfiguracionCorreoNoDisponible
	}
	tx, err := r.iniciar(ctx, pgx.ReadOnly)
	if err != nil {
		return normalizarError(ctx, err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var version int64
	var claveRef string
	var nonce, cifrado []byte
	err = tx.QueryRow(ctx, `SELECT version_secreto,clave_ref,nonce,secreto_cifrado FROM vec_administracion.sobre_configuracion_correo_actual_v1()`).Scan(&version, &claveRef, &nonce, &cifrado)
	if err != nil || version <= 0 {
		return ErrConfiguracionCorreoNoDisponible
	}
	sobre := SobreSecretoCorreo{Version: uint64(version), ClaveRef: claveRef, Nonce: append([]byte(nil), nonce...), Cifrado: append([]byte(nil), cifrado...)}
	defer borrar(sobre.Nonce)
	defer borrar(sobre.Cifrado)
	aad, err := aadConfiguracionCorreo(sobre.Version)
	if err != nil {
		return ErrConfiguracionCorreoNoDisponible
	}
	defer borrar(aad)
	if err = r.protector.ConSecretoCorreoDescifrado(ctx, sobre, aad, func(claro []byte) error { defer borrar(claro); return usar(claro) }); err != nil {
		return ErrConfiguracionCorreoNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return normalizarError(ctx, err)
	}
	return nil
}

func (r *RegistroConfiguracionCorreoPostgreSQL) valido(ctx context.Context) error {
	if ctx == nil || ctx.Err() != nil || r == nil || dependenciaNula(r.pool) || dependenciaNula(r.protector) {
		return ErrConfiguracionCorreoNoDisponible
	}
	return nil
}
func (r *RegistroConfiguracionCorreoPostgreSQL) iniciar(ctx context.Context, acceso pgx.TxAccessMode) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: acceso})
	if err != nil {
		return nil, err
	}
	for _, s := range [...]string{"SET LOCAL search_path = pg_catalog", "SET LOCAL row_security = on", "SET LOCAL TIME ZONE 'UTC'", "SET LOCAL lock_timeout = '3s'", "SET LOCAL statement_timeout = '15s'", "SET LOCAL idle_in_transaction_session_timeout = '20s'"} {
		if _, err = tx.Exec(ctx, s); err != nil {
			_ = tx.Rollback(context.Background())
			return nil, err
		}
	}
	return tx, nil
}
func aadConfiguracionCorreo(version uint64) ([]byte, error) {
	return json.Marshal(struct {
		Esquema, Referencia string
		Version             uint64
	}{"vec.administracion.configuracion-correo.secreto.v1", referenciaConfiguracionCorreo, version})
}

func sobreJSON(s SobreSecretoCorreo, presente bool) []byte {
	if !presente {
		return nil
	}
	aad, err := aadConfiguracionCorreo(s.Version)
	if err != nil {
		return nil
	}
	defer borrar(aad)
	huella := sha256.Sum256(aad)
	b, _ := json.Marshal(struct {
		Version   uint64 `json:"version"`
		ClaveRef  string `json:"clave_ref"`
		Nonce     []byte `json:"nonce"`
		Cifrado   []byte `json:"cifrado"`
		HuellaAAD string `json:"huella_aad_sha256"`
	}{s.Version, s.ClaveRef, s.Nonce, s.Cifrado, hex.EncodeToString(huella[:])})
	return b
}
func vistaDesdeJSON(datos []byte) (admindomain.VistaConfiguracionCorreo, error) {
	var v admindomain.VistaConfiguracionCorreo
	if len(datos) > 8192 || json.Unmarshal(datos, &v) != nil || v.Validar() != nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrConfiguracionCorreoNoDisponible
	}
	return v, nil
}
func auditoriaInvalida(a vecdomain.AuditEntry) bool {
	return a.ActorID == "" || a.Action == "" || a.ModuleID == "" || a.SubjectRef != referenciaConfiguracionCorreo || a.Result != "accepted" || a.OccurredAt.IsZero()
}
func dependenciaNula(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return x.IsNil()
	}
	return false
}
func borrar(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
func normalizarError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ErrConfiguracionCorreoNoDisponible
	}
	var p *pgconn.PgError
	if errors.As(err, &p) && (p.Code == "P0002" || p.Code == "40001" || p.Code == "P0409") {
		return adminports.ErrConfiguracionCorreoConflicto
	}
	return ErrConfiguracionCorreoNoDisponible
}
