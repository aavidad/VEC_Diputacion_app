package administracion

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

type claveConexionPerfiles struct{}

type conexionPerfiles struct {
	aceptadaEn time.Time
	conexion   *tls.Conn
}

// NuevoContextoConexionPerfiles conserva el inicio de la autenticación TLS
// hasta la petición. La hora de otra petición no rejuvenece la credencial.
func NuevoContextoConexionPerfiles(reloj httpseguridad.Reloj) (func(context.Context, net.Conn) context.Context, error) {
	if reloj == nil {
		return nil, ErrConfiguracion
	}
	return func(ctx context.Context, c net.Conn) context.Context {
		conexion, ok := c.(*tls.Conn)
		if ctx == nil || !ok || conexion == nil || ctx.Value(claveConexionPerfiles{}) != nil {
			return ctx
		}
		return context.WithValue(ctx, claveConexionPerfiles{}, conexionPerfiles{
			aceptadaEn: reloj.Ahora().UTC().Truncate(time.Microsecond), conexion: conexion,
		})
	}, nil
}

func autenticacionConexionPerfiles(ctx context.Context, r *http.Request) (time.Time, error) {
	if ctx == nil || r == nil || r.TLS == nil {
		return time.Time{}, errAccesoDenegado
	}
	guardada, ok := ctx.Value(claveConexionPerfiles{}).(conexionPerfiles)
	if !ok || guardada.conexion == nil || guardada.aceptadaEn.IsZero() {
		return time.Time{}, errAccesoDenegado
	}
	actual := guardada.conexion.ConnectionState()
	if !actual.HandshakeComplete || actual.DidResume || !r.TLS.HandshakeComplete || r.TLS.DidResume {
		return time.Time{}, errAccesoDenegado
	}
	a, err := materialConexionPerfiles(actual)
	if err != nil {
		return time.Time{}, errAccesoDenegado
	}
	b, err := materialConexionPerfiles(*r.TLS)
	if err != nil || subtle.ConstantTimeCompare(a[:], b[:]) != 1 {
		return time.Time{}, errAccesoDenegado
	}
	return guardada.aceptadaEn, nil
}

func materialConexionPerfiles(estado tls.ConnectionState) (huella [32]byte, err error) {
	defer func() {
		if recover() != nil {
			huella, err = [32]byte{}, errAccesoDenegado
		}
	}()
	material, err := estado.ExportKeyingMaterial("VEC-ADMIN-perfiles-conexion-v1", nil, 32)
	if err != nil {
		return huella, errAccesoDenegado
	}
	huella = sha256.Sum256(material)
	for i := range material {
		material[i] = 0
	}
	return huella, nil
}
