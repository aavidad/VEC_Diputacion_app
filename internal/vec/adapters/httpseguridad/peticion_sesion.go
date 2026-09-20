package httpseguridad

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"
)

var (
	ErrAsercionPeticionAusente       = errors.New("asercion de peticion ausente")
	ErrAsercionPeticionNoValida      = errors.New("asercion de peticion no valida")
	ErrRegistroPeticionesAusente     = errors.New("registro de peticiones de sesion ausente")
	ErrSeudonimizadorPeticionAusente = errors.New("seudonimizador de peticiones de sesion ausente")
)

// AsercionPeticionVerificada es el resultado ya autenticado de una asercion
// por peticion. No contiene cuenta, persona, perfil, permiso ni concesion:
// esas autoridades se revalidan fuera de la asercion.
type AsercionPeticionVerificada struct {
	Emisor            string
	Audiencia         string
	Superficie        Superficie
	SesionID          string
	Metodo            string
	Destino           string
	CuerpoSHA256      string
	NonceSHA256       string
	EmitidaEn         time.Time
	ExpiraEn          time.Time
	CanalVinculadoRef string
}

// VerificadorAsercionPeticionProtegida comprueba firma, algoritmo y formato
// antes de que se compare la asercion con la peticion efectiva.
type VerificadorAsercionPeticionProtegida interface {
	VerificarPeticion(context.Context, []byte) (AsercionPeticionVerificada, error)
}

// SolicitudConsumoPeticionSesion es la entrada minima para el consumo
// antirrepeticion y la revalidacion durable de la sesion.
type SolicitudConsumoPeticionSesion struct {
	EsquemaHMAC       string
	DominioHMACRef    string
	ClaveHMACID       string
	ClaveHMACVersion  uint64
	SesionIDHMAC      [32]byte
	NonceSHA256       string
	CanalVinculadoRef string
	Superficie        Superficie
	Metodo            string
	Destino           string
	CuerpoSHA256      string
	EmitidaEn         time.Time
	ExpiraEn          time.Time
}

// SeudonimoSesionPeticion transporta al registro solo las coordenadas y el
// digest de sesion ya derivado; los identificadores de emisor y sesion no
// abandonan el servicio.
type SeudonimoSesionPeticion struct {
	EsquemaHMAC      string
	DominioHMACRef   string
	ClaveHMACID      string
	ClaveHMACVersion uint64
	SesionIDHMAC     [32]byte
}

// SeudonimizadorPeticionSesion selecciona el dominio de emisor configurado y
// deriva el HMAC de sesion fuera de PostgreSQL.
type SeudonimizadorPeticionSesion interface {
	SeudonimizarSesionPeticion(context.Context, string, string) (SeudonimoSesionPeticion, error)
}

// ConfirmacionPeticionSesion devuelve solo referencias canonicas obtenidas de
// la autoridad durable; nunca identifica la cuenta desde los claims.
type ConfirmacionPeticionSesion struct {
	SesionRef             string
	AutenticacionRef      string
	AsercionRef           string
	CuentaRef             string
	ControlSesionRef      string
	ControlSesionRevision uint64
	SesionValidaHasta     time.Time
}

// SolicitudRevocarSesionExacta identifica el control durable que puede
// revocarse; nunca admite una referencia derivada desde claims.
type SolicitudRevocarSesionExacta struct {
	SesionRef             string
	ControlSesionRef      string
	ControlSesionRevision uint64
}

type RevocadorSesionExacta interface {
	RevocarSesionExacta(context.Context, SolicitudRevocarSesionExacta) error
}

// RegistroPeticionesSesion consume el nonce y revalida la sesion en una unica
// operacion durable.
type RegistroPeticionesSesion interface {
	ConsumirYRevalidar(context.Context, SolicitudConsumoPeticionSesion) (ConfirmacionPeticionSesion, error)
}

// ServicioPeticionSesion consume aserciones por peticion sin crear otra
// sesion. La identidad posterior debe proceder exclusivamente de la
// confirmacion durable.
type ServicioPeticionSesion struct {
	autoridadIdentidad *ServicioIdentidad
	configuracion      ConfiguracionSuperficie
	verificador        VerificadorAsercionPeticionProtegida
	registro           RegistroPeticionesSesion
	seudonimizador     SeudonimizadorPeticionSesion
	reloj              Reloj
}

func NuevoServicioPeticionSesion(
	autoridadIdentidad *ServicioIdentidad,
	verificador VerificadorAsercionPeticionProtegida,
	registro RegistroPeticionesSesion,
	seudonimizador SeudonimizadorPeticionSesion,
	reloj Reloj,
) (*ServicioPeticionSesion, error) {
	if autoridadIdentidad == nil {
		return nil, ErrCanalProxyNoAutenticado
	}
	configuracion := autoridadIdentidad.configuracion
	if err := configuracion.Validar(); err != nil {
		return nil, err
	}
	if configuracion.Superficie == SuperficiePublicaAnonima {
		return nil, ErrConfiguracionSuperficie
	}
	if verificador == nil {
		return nil, ErrVerificadorAusente
	}
	if registro == nil {
		return nil, ErrRegistroPeticionesAusente
	}
	if seudonimizador == nil {
		return nil, ErrSeudonimizadorPeticionAusente
	}
	if reloj == nil {
		reloj = relojSistema{}
	}
	return &ServicioPeticionSesion{
		autoridadIdentidad: autoridadIdentidad,
		configuracion:      copiarYNormalizarConfiguracion(configuracion), verificador: verificador,
		registro: registro, seudonimizador: seudonimizador, reloj: reloj,
	}, nil
}

func (s *ServicioPeticionSesion) Resolver(
	ctx context.Context,
	sobre []byte,
	canal CanalProxyAutenticado,
	metodo, destino string,
	cuerpo []byte,
) (ConfirmacionPeticionSesion, error) {
	if s == nil || s.autoridadIdentidad == nil || s.verificador == nil || s.registro == nil || s.seudonimizador == nil || s.reloj == nil {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	if err := validarContexto(ctx); err != nil {
		return ConfirmacionPeticionSesion{}, err
	}
	if len(sobre) == 0 {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionAusente
	}
	if len(sobre) > longitudMaximaAsercionProtegida || canal.validar(s.autoridadIdentidad) != nil {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	verificada, err := s.verificador.VerificarPeticion(ctx, append([]byte(nil), sobre...))
	if err != nil {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	ahora := s.reloj.Ahora()
	if !peticionCoincide(verificada, s.configuracion, canal.ReferenciaVinculacion(), metodo, destino, cuerpo, ahora) {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	seudonimo, err := s.seudonimizador.SeudonimizarSesionPeticion(ctx, verificada.Emisor, verificada.SesionID)
	if err != nil || !seudonimoSesionPeticionValido(seudonimo) {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	confirmacion, err := s.registro.ConsumirYRevalidar(ctx, SolicitudConsumoPeticionSesion{
		EsquemaHMAC: seudonimo.EsquemaHMAC, DominioHMACRef: seudonimo.DominioHMACRef,
		ClaveHMACID: seudonimo.ClaveHMACID, ClaveHMACVersion: seudonimo.ClaveHMACVersion,
		SesionIDHMAC: seudonimo.SesionIDHMAC,
		NonceSHA256:  verificada.NonceSHA256, CanalVinculadoRef: verificada.CanalVinculadoRef,
		Superficie: verificada.Superficie, Metodo: verificada.Metodo, Destino: verificada.Destino,
		CuerpoSHA256: verificada.CuerpoSHA256, EmitidaEn: verificada.EmitidaEn, ExpiraEn: verificada.ExpiraEn,
	})
	ahoraConfirmacion := s.reloj.Ahora()
	if err != nil || ahoraConfirmacion.IsZero() || !ahoraConfirmacion.Before(verificada.ExpiraEn) ||
		!confirmacionPeticionValida(confirmacion, ahoraConfirmacion) {
		return ConfirmacionPeticionSesion{}, ErrAsercionPeticionNoValida
	}
	return confirmacion, nil
}

func seudonimoSesionPeticionValido(s SeudonimoSesionPeticion) bool {
	return s.EsquemaHMAC != "" && referenciaOpacaSesionValida(s.DominioHMACRef, "idh_") &&
		textoTecnicoPeticionValido(s.ClaveHMACID) && s.ClaveHMACVersion > 0 &&
		s.SesionIDHMAC != [32]byte{}
}

func textoTecnicoPeticionValido(valor string) bool {
	return valor != "" && len(valor) <= 128 && valor == strings.TrimSpace(valor) &&
		!strings.ContainsAny(valor, " \t\r\n")
}

func peticionCoincide(a AsercionPeticionVerificada, c ConfiguracionSuperficie, canal, metodo, destino string, cuerpo []byte, ahora time.Time) bool {
	if ahora.IsZero() || a.Emisor != c.EmisorIdentidad || a.Audiencia != c.Audiencia ||
		a.Superficie != c.Superficie || a.CanalVinculadoRef == "" ||
		subtle.ConstantTimeCompare([]byte(a.CanalVinculadoRef), []byte(canal)) != 1 ||
		!metodoHTTPValido(metodo) || a.Metodo != metodo || !destinoHTTPValido(destino) || a.Destino != destino ||
		!hexSHA256Valido(a.CuerpoSHA256) || !hexSHA256Valido(a.NonceSHA256) ||
		!instantePeticionCanonico(a.EmitidaEn) || !instantePeticionCanonico(a.ExpiraEn) ||
		a.EmitidaEn.After(ahora.Add(c.ToleranciaReloj)) || !ahora.Before(a.ExpiraEn) ||
		!a.ExpiraEn.After(a.EmitidaEn) || a.ExpiraEn.Sub(a.EmitidaEn) > c.DuracionMaximaAsercion ||
		canonicalizarIDSinError(a.SesionID, longitudMaximaID) == "" {
		return false
	}
	suma := sha256.Sum256(cuerpo)
	return subtle.ConstantTimeCompare([]byte(a.CuerpoSHA256), []byte(hex.EncodeToString(suma[:]))) == 1
}

func confirmacionPeticionValida(c ConfirmacionPeticionSesion, ahora time.Time) bool {
	return referenciaOpacaSesionValida(c.SesionRef, "ses_") &&
		referenciaOpacaSesionValida(c.AutenticacionRef, "aut_") &&
		referenciaOpacaSesionValida(c.AsercionRef, "ase_") &&
		referenciaOpacaSesionValida(c.CuentaRef, "cta_") &&
		referenciaOpacaSesionValida(c.ControlSesionRef, "cse_") && c.ControlSesionRevision > 0 &&
		instantePeticionCanonico(c.SesionValidaHasta) && ahora.Before(c.SesionValidaHasta)
}

func metodoHTTPValido(metodo string) bool {
	return metodo != "" && metodo == strings.ToUpper(metodo) && !strings.ContainsAny(metodo, " \t\r\n")
}

func destinoHTTPValido(destino string) bool {
	if destino == "" || strings.Contains(destino, "#") || strings.Contains(destino, "//") {
		return false
	}
	u, err := url.ParseRequestURI(destino)
	return err == nil && u.IsAbs() == false && u.Host == "" && u.Path == destino && u.RawQuery == "" && u.EscapedPath() == destino
}

func hexSHA256Valido(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}

func instantePeticionCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Equal(instante.Truncate(time.Microsecond))
}

func canonicalizarIDSinError(valor string, maximo int) string {
	resultado, err := canonicalizarID(valor, maximo, false)
	if err != nil {
		return ""
	}
	return resultado
}
