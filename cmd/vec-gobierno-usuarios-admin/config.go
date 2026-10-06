package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/vec/auditoria"
)

const noSeguirEnlaces = syscall.O_NOFOLLOW | syscall.O_CLOEXEC | syscall.O_NONBLOCK

// configuracionPrivada procede de un JSON propio 0600 fuera de Git. Cada fase
// exige sólo sus campos; los demás pueden ir vacíos para usar un único fichero.
type configuracionPrivada struct {
	DirectorioMaterial    string `json:"directorio_material"`
	RutaConfiguracionHMAC string `json:"ruta_configuracion_hmac"`
	ArchivoSemillaRaiz    string `json:"archivo_semilla_raiz"`
	DSNLectura            string `json:"dsn_lectura"`
	DSNOperador           string `json:"dsn_operador"`
	Salida                string `json:"salida"`
	HorasValidezClaves    int    `json:"horas_validez_claves"`
	// ConjuntoCapacidades 0 (o ausente) publica las dos claves de usuarios con
	// AD188; 1 publica el conjunto 1 de AD198 (usuarios y lote ordinario); 2, el
	// conjunto 2 de AD202 (además, el gobierno del plan nominal de firma); 3, el
	// conjunto 3 de AD204 (además, la publicación de cargos competenciales).
	ConjuntoCapacidades uint64 `json:"conjunto_capacidades,omitempty"`
}

var errConfiguracion = errors.New("configuracion")

func (c configuracionPrivada) validar(fase string) error {
	if _, ok := bootstrap.AudienciasConjuntoCapacidadesAdmin(c.ConjuntoCapacidades); !ok ||
		!rutaAbsoluta(c.Salida) || c.HorasValidezClaves < 0 || c.HorasValidezClaves > 24 {
		return errConfiguracion
	}
	switch fase {
	case "preparar":
		if !rutaAbsoluta(c.DirectorioMaterial) || !rutaAbsoluta(c.RutaConfiguracionHMAC) || !rutaAbsoluta(c.ArchivoSemillaRaiz) || c.HorasValidezClaves < 1 || dsnValido(c.DSNLectura) != nil {
			return errConfiguracion
		}
	case "aplicar":
		if dsnValido(c.DSNOperador) != nil {
			return errConfiguracion
		}
		// El LOGIN técnico es exclusivo: no puede ser el mismo usuario de lectura.
		if c.DSNLectura != "" && mismoUsuario(c.DSNLectura, c.DSNOperador) {
			return errConfiguracion
		}
	case "verificar":
		if dsnValido(c.DSNLectura) != nil {
			return errConfiguracion
		}
	default:
		return errConfiguracion
	}
	return nil
}

func rutaAbsoluta(r string) bool { return r != "" && filepath.IsAbs(r) && filepath.Clean(r) == r }

// dsnValido exige usuario y base, sin SET ROLE ni opciones libres, y un canal
// local (socket o bucle) o TLS con verificación del servidor.
func dsnValido(dsn string) error {
	if dsn == "" {
		return errConfiguracion
	}
	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return errConfiguracion
	}
	cc := &pc.ConnConfig.Config
	if cc.User == "" || cc.Database == "" || !destinoValido(cc.Host, cc.TLSConfig) {
		return errConfiguracion
	}
	for _, f := range cc.Fallbacks {
		if f == nil || !destinoValido(f.Host, f.TLSConfig) {
			return errConfiguracion
		}
	}
	// Solo se admite application_name: cualquier otro parámetro de sesión
	// (role en cualquier grafía, options, search_path, default_transaction_*)
	// podría cambiar la identidad efectiva o el comportamiento de la sesión.
	for clave := range cc.RuntimeParams {
		if clave != "application_name" {
			return errConfiguracion
		}
	}
	return nil
}

func destinoValido(host string, t *tls.Config) bool {
	if strings.HasPrefix(host, "/") {
		return socketPrivado(host)
	}
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
	return host != "" && t != nil && !t.InsecureSkipVerify && t.ServerName != ""
}

// socketPrivado exige un directorio de socket real, sin enlace y sin escritura
// para cualquiera: en una carpeta como /tmp otro usuario podría suplantar el
// socket de PostgreSQL y recibir la contraseña del DSN.
func socketPrivado(dir string) bool {
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir {
		return false
	}
	i, err := os.Lstat(dir)
	return err == nil && i.IsDir() && i.Mode().Perm()&0o002 == 0
}

func mismoUsuario(a, b string) bool {
	pa, ea := pgconn.ParseConfig(a)
	pb, eb := pgconn.ParseConfig(b)
	return ea != nil || eb != nil || pa.User == pb.User
}

// operaciones separa la consola de PostgreSQL para poder probarla sin base.
type operaciones struct {
	preparar  func(context.Context, string, time.Duration, bootstrap.MaterialOrigenGobiernoUsuariosAdmin, *os.Root) (bootstrap.PreparacionGobiernoUsuariosAdmin, error)
	aplicar   func(context.Context, string, time.Duration, *os.Root, string) (bootstrap.ConfirmacionGobiernoUsuariosAdmin, error)
	verificar func(context.Context, string, time.Duration, *os.Root, string) (auditoria.InformeVerificacion, error)
}

func (o operaciones) incompletas() bool {
	return o.preparar == nil || o.aplicar == nil || o.verificar == nil
}

type relojSistema struct{}

func (relojSistema) Ahora() time.Time { return time.Now().UTC() }

func operacionesPG() operaciones {
	return operaciones{
		preparar: func(ctx context.Context, dsn string, limite time.Duration, o bootstrap.MaterialOrigenGobiernoUsuariosAdmin, r *os.Root) (bootstrap.PreparacionGobiernoUsuariosAdmin, error) {
			pool, err := abrirPool(ctx, dsn, limite)
			if err != nil {
				return bootstrap.PreparacionGobiernoUsuariosAdmin{}, err
			}
			defer pool.Close()
			return bootstrap.PrepararGobiernoUsuariosAdmin(ctx, pool, o, r, relojSistema{})
		},
		aplicar: func(ctx context.Context, dsn string, limite time.Duration, r *os.Root, acuse string) (bootstrap.ConfirmacionGobiernoUsuariosAdmin, error) {
			pool, err := abrirPool(ctx, dsn, limite)
			if err != nil {
				return bootstrap.ConfirmacionGobiernoUsuariosAdmin{}, err
			}
			defer pool.Close()
			return bootstrap.AplicarGobiernoUsuariosAdminPreparado(ctx, pool, r, acuse, relojSistema{})
		},
		verificar: func(ctx context.Context, dsn string, limite time.Duration, r *os.Root, nombre string) (auditoria.InformeVerificacion, error) {
			pool, err := abrirPool(ctx, dsn, limite)
			if err != nil {
				return auditoria.InformeVerificacion{}, err
			}
			defer pool.Close()
			return bootstrap.VerificarCadenaGobiernoUsuariosAdmin(ctx, pool, r, nombre)
		},
	}
}

// abrirPool crea un pool mínimo; la conexión real se abre en la primera
// consulta y sus errores nunca se muestran (pueden contener el DSN).
func abrirPool(ctx context.Context, dsn string, limite time.Duration) (*pgxpool.Pool, error) {
	if dsnValido(dsn) != nil {
		return nil, errConfiguracion
	}
	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errConfiguracion
	}
	pc.MaxConns = 2
	pc.ConnConfig.ConnectTimeout = limite
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, errConfiguracion
	}
	return pool, nil
}

// decodificarEstricto rechaza claves repetidas, campos desconocidos y datos
// tras el primer documento JSON.
func decodificarEstricto(b []byte, destino any) error {
	if err := clavesUnicas(b); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	var sobra any
	if !errors.Is(d.Decode(&sobra), io.EOF) {
		return errors.New("json_extra")
	}
	return nil
}

// El decodificador de Go acepta claves repetidas; este recorrido las deniega.
func clavesUnicas(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var recorrer func() error
	recorrer = func() error {
		t, err := d.Token()
		if err != nil {
			return err
		}
		switch t {
		case json.Delim('{'):
			vistas := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return err
				}
				clave, ok := k.(string)
				if !ok || vistas[clave] {
					return errors.New("json_clave_repetida")
				}
				vistas[clave] = true
				if err := recorrer(); err != nil {
					return err
				}
			}
			_, err := d.Token()
			return err
		case json.Delim('['):
			for d.More() {
				if err := recorrer(); err != nil {
					return err
				}
			}
			_, err := d.Token()
			return err
		default:
			return nil
		}
	}
	return recorrer()
}
