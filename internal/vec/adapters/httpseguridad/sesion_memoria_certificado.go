package httpseguridad

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

type asientoTokenSesionMemoria struct {
	servicio          *ServicioPresentacionCertificado
	sesionRef         string
	audiencia         string
	certificadoSHA256 [sha256.Size]byte
	caSHA256          [sha256.Size]byte
	generation        uint64
	validoHasta       time.Time
}

// RegistroSesionMemoriaCertificado conserva solo la huella del token. Cada
// instancia nace vacía; reiniciar el servidor elimina todas las continuidades.
type RegistroSesionMemoriaCertificado struct {
	mu       sync.Mutex
	reloj    Reloj
	politica PoliticaSesionMemoriaCertificado
	asientos map[[sha256.Size]byte]asientoTokenSesionMemoria
	cerrado  bool
}

func (*RegistroSesionMemoriaCertificado) String() string {
	return "[REGISTRO-TOKEN-SESION-CONFIDENCIAL]"
}
func (r *RegistroSesionMemoriaCertificado) GoString() string { return r.String() }
func (r *RegistroSesionMemoriaCertificado) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, r.String())
}
func (r *RegistroSesionMemoriaCertificado) LogValue() slog.Value { return slog.StringValue(r.String()) }

// VinculoSesionMemoriaCertificado solo nace de un token verificado por este
// registro. Permite confirmar la sesión durable y revocar sin localizadores HTTP.
type VinculoSesionMemoriaCertificado struct {
	bloqueoSerializacionPresentacionCertificado
	dueno       *RegistroSesionMemoriaCertificado
	servicio    *ServicioPresentacionCertificado
	huellaToken [sha256.Size]byte
	sesionRef   string
	generation  uint64
	marcador    *marcaPeticionCertificadoActual
}

func (VinculoSesionMemoriaCertificado) String() string     { return "[VINCULO-TOKEN-SESION-CONFIDENCIAL]" }
func (v VinculoSesionMemoriaCertificado) GoString() string { return v.String() }
func (v VinculoSesionMemoriaCertificado) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, v.String())
}
func (v VinculoSesionMemoriaCertificado) LogValue() slog.Value { return slog.StringValue(v.String()) }
func (VinculoSesionMemoriaCertificado) MarshalJSON() ([]byte, error) {
	return nil, ErrTokenSesionMemoriaCertificadoNoValido
}

func NuevoRegistroSesionMemoriaCertificado(
	reloj Reloj, politica PoliticaSesionMemoriaCertificado,
) (*RegistroSesionMemoriaCertificado, error) {
	if interfazNulaPasarela(reloj) || politica.version == 0 ||
		!referenciaPoliticaSesionValida(politica.referencia) || politica.vida <= 0 ||
		politica.vida > maximoTecnicoVidaToken || politica.capacidad <= 0 ||
		politica.capacidad > maximoTecnicoTokens {
		return nil, ErrTokenSesionMemoriaCertificadoNoValido
	}
	return &RegistroSesionMemoriaCertificado{
		reloj: reloj, politica: politica,
		asientos: make(map[[sha256.Size]byte]asientoTokenSesionMemoria),
	}, nil
}

func (r *RegistroSesionMemoriaCertificado) ahora() (time.Time, error) {
	if r == nil || interfazNulaPasarela(r.reloj) {
		return time.Time{}, ErrTokenSesionMemoriaCertificadoNoValido
	}
	ahora := r.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ahora.IsZero() || ahora.Year() < 1 || ahora.Year() > 9999 {
		return time.Time{}, ErrTokenSesionMemoriaCertificadoNoValido
	}
	return ahora, nil
}

func pruebaTokenCertificadoActual(
	ctx context.Context, servicio *ServicioPresentacionCertificado,
	prueba PruebaCertificadoActual, canal CanalProxyAutenticado, ahora time.Time,
) bool {
	if ctx == nil || ctx.Err() != nil || servicio == nil || servicio.identidad == nil ||
		prueba.datos == nil || prueba.datos.servicio != servicio || prueba.datos.marcador == nil ||
		ctx.Value(clavePeticionCertificadoActual{}) != prueba.datos.marcador ||
		canal.validar(servicio.identidad) != nil ||
		subtle.ConstantTimeCompare([]byte(prueba.datos.canalRef), []byte(canal.ReferenciaVinculacion())) != 1 ||
		!ahora.Before(prueba.datos.certificadoHasta) || !ahora.Before(prueba.datos.caHasta) ||
		!ahora.Before(prueba.datos.crlSiguienteEn) ||
		!ahora.Before(servicio.identidad.configuracion.RetiradaPoliticaInternaEn) ||
		prueba.datos.certificadoSHA256 == "" || prueba.datos.caSHA256 == "" {
		return false
	}
	return true
}

func capsulaTokenCertificadoVinculada(
	ctx context.Context, servicio *ServicioPresentacionCertificado,
	prueba PruebaCertificadoActual, canal CanalProxyAutenticado,
	ahora time.Time, inicio bool,
) (*datosCapsulaPresentacionCertificado, error) {
	if !pruebaTokenCertificadoActual(ctx, servicio, prueba, canal, ahora) ||
		!prueba.datos.consumida.Load() {
		return nil, ErrTokenSesionMemoriaCertificadoNoValido
	}
	vinculada, ok := ctx.Value(claveCapsulaPresentacionCertificado{}).(capsulaPresentacionCertificadoVinculada)
	if !ok || vinculada.capsula.datos == nil || vinculada.capsula.datos.servicio != servicio ||
		vinculada.capsula.datos.marcador != prueba.datos.marcador ||
		vinculada.capsula.datos.inicioExplicito != inicio ||
		!vinculada.capsula.datos.consumida.Load() ||
		subtle.ConstantTimeCompare([]byte(vinculada.canalRef), []byte(canal.ReferenciaVinculacion())) != 1 {
		return nil, ErrTokenSesionMemoriaCertificadoNoValido
	}
	c := vinculada.capsula.datos
	if c.presentacion.CertificadoSHA256 != prueba.datos.certificadoSHA256 ||
		c.presentacion.CASHA256 != prueba.datos.caSHA256 ||
		c.estadoActual.audiencia != servicio.identidad.configuracion.Audiencia ||
		validarResultadoPresentacionCertificado(c.resultado, c.presentacion, c.estadoActual, inicio, ahora) != nil {
		return nil, ErrTokenSesionMemoriaCertificadoNoValido
	}
	if _, _, err := servicio.identidad.datosCapsulaPresentacion(ctx); err != nil {
		return nil, falloTokenSesionMemoriaCertificado{causa: err}
	}
	return c, nil
}

// EmitirDesdeInicio requiere la cápsula de una orden Iniciar confirmada y ya
// vinculada a esta petición. Devuelve el token una sola vez al consumidor
// confiable para su respuesta POST; nunca registra el token en PostgreSQL.
func (r *RegistroSesionMemoriaCertificado) EmitirDesdeInicio(
	ctx context.Context, servicio *ServicioPresentacionCertificado,
	prueba PruebaCertificadoActual, canal CanalProxyAutenticado,
) (string, VinculoSesionMemoriaCertificado, error) {
	var vacio VinculoSesionMemoriaCertificado
	ahora, err := r.ahora()
	if err != nil {
		return "", vacio, err
	}
	c, err := capsulaTokenCertificadoVinculada(ctx, servicio, prueba, canal, ahora, true)
	if err != nil {
		return "", vacio, err
	}
	if c.resultado.SesionOriginal.SesionRef == "" || c.resultado.Recibo.SesionGeneracion == 0 {
		return "", vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	vence := ahora.Add(r.politica.vida)
	for _, limite := range []time.Time{
		c.resultado.SesionOriginal.SesionValidaHasta, c.resultado.Recibo.ValidaHasta,
		prueba.datos.certificadoHasta, prueba.datos.caHasta, prueba.datos.crlSiguienteEn,
		servicio.identidad.configuracion.RetiradaPoliticaInternaEn,
	} {
		if limite.IsZero() || !ahora.Before(limite) {
			return "", vacio, ErrTokenSesionMemoriaCertificadoNoValido
		}
		if limite.Before(vence) {
			vence = limite
		}
	}
	if !c.tokenEmitido.CompareAndSwap(false, true) {
		return "", vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	asiento := asientoTokenSesionMemoria{
		servicio:          servicio,
		sesionRef:         c.resultado.SesionOriginal.SesionRef,
		audiencia:         c.estadoActual.audiencia,
		certificadoSHA256: sha256.Sum256([]byte(prueba.datos.certificadoSHA256)),
		caSHA256:          sha256.Sum256([]byte(prueba.datos.caSHA256)),
		generation:        c.resultado.Recibo.SesionGeneracion,
		validoHasta:       vence,
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado || ctx.Err() != nil {
		return "", vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	r.purgarCaducadosBloqueado(ahora)
	if len(r.asientos) >= r.politica.capacidad {
		return "", vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	var aleatorio [bytesTokenSesion]byte
	if _, err := rand.Read(aleatorio[:]); err != nil || aleatorio == ([bytesTokenSesion]byte{}) {
		clear(aleatorio[:])
		return "", vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	token := base64.RawURLEncoding.EncodeToString(aleatorio[:])
	huella := sha256.Sum256(aleatorio[:])
	clear(aleatorio[:])
	if _, repetido := r.asientos[huella]; repetido {
		return "", vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	r.asientos[huella] = asiento
	return token, VinculoSesionMemoriaCertificado{
		dueno: r, servicio: servicio, huellaToken: huella,
		sesionRef: asiento.sesionRef, generation: asiento.generation,
		marcador: prueba.datos.marcador,
	}, nil
}

func (r *RegistroSesionMemoriaCertificado) purgarCaducadosBloqueado(ahora time.Time) {
	for huella, asiento := range r.asientos {
		if !ahora.Before(asiento.validoHasta) {
			delete(r.asientos, huella)
		}
	}
}

func huellaTokenSesion(token string) ([sha256.Size]byte, error) {
	if len(token) != base64.RawURLEncoding.EncodedLen(bytesTokenSesion) {
		return [sha256.Size]byte{}, ErrTokenSesionMemoriaCertificadoNoValido
	}
	material, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil {
		clear(material)
		return [sha256.Size]byte{}, falloTokenSesionMemoriaCertificado{causa: err}
	}
	if len(material) != bytesTokenSesion || base64.RawURLEncoding.EncodeToString(material) != token {
		clear(material)
		return [sha256.Size]byte{}, ErrTokenSesionMemoriaCertificadoNoValido
	}
	huella := sha256.Sum256(material)
	clear(material)
	return huella, nil
}

// VerificarActual solo habilita la consulta durable posterior. No abre sesión,
// no obtiene actor y no renueva el token. La prueba procede del TLS de ahora.
func (r *RegistroSesionMemoriaCertificado) VerificarActual(
	ctx context.Context, token string, servicio *ServicioPresentacionCertificado,
	prueba PruebaCertificadoActual, canal CanalProxyAutenticado,
) (VinculoSesionMemoriaCertificado, error) {
	var vacio VinculoSesionMemoriaCertificado
	ahora, err := r.ahora()
	if err != nil || !pruebaTokenCertificadoActual(ctx, servicio, prueba, canal, ahora) ||
		prueba.datos.consumida.Load() {
		return vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	huella, err := huellaTokenSesion(token)
	if err != nil {
		return vacio, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado {
		return vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	asiento, existe := r.asientos[huella]
	if !existe {
		return vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	if !ahora.Before(asiento.validoHasta) {
		delete(r.asientos, huella)
		return vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	certificado := sha256.Sum256([]byte(prueba.datos.certificadoSHA256))
	ca := sha256.Sum256([]byte(prueba.datos.caSHA256))
	if asiento.servicio != servicio || asiento.audiencia != servicio.identidad.configuracion.Audiencia ||
		asiento.sesionRef == "" || asiento.generation == 0 ||
		subtle.ConstantTimeCompare(certificado[:], asiento.certificadoSHA256[:]) != 1 ||
		subtle.ConstantTimeCompare(ca[:], asiento.caSHA256[:]) != 1 {
		return vacio, ErrTokenSesionMemoriaCertificadoNoValido
	}
	return VinculoSesionMemoriaCertificado{
		dueno: r, servicio: servicio, huellaToken: huella,
		sesionRef: asiento.sesionRef, generation: asiento.generation,
		marcador: prueba.datos.marcador,
	}, nil
}

// ConfirmarReanudacion coteja el asiento efímero con el resultado de la
// reanudación SQL de la MISMA petición. Nunca prolonga la vida del token.
func (r *RegistroSesionMemoriaCertificado) ConfirmarReanudacion(
	ctx context.Context, vinculo VinculoSesionMemoriaCertificado,
) error {
	ahora, err := r.ahora()
	if err != nil || !vinculo.validoPara(r) || ctx == nil || ctx.Err() != nil ||
		ctx.Value(clavePeticionCertificadoActual{}) != vinculo.marcador {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	ligada, ok := ctx.Value(claveCapsulaPresentacionCertificado{}).(capsulaPresentacionCertificadoVinculada)
	if !ok || ligada.capsula.datos == nil || ligada.capsula.datos.servicio != vinculo.servicio ||
		ligada.capsula.datos.marcador != vinculo.marcador ||
		ligada.capsula.datos.inicioExplicito || !ligada.capsula.datos.consumida.Load() ||
		validarResultadoPresentacionCertificado(
			ligada.capsula.datos.resultado, ligada.capsula.datos.presentacion,
			ligada.capsula.datos.estadoActual, false, ahora,
		) != nil {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	if _, _, err := vinculo.servicio.identidad.datosCapsulaPresentacion(ctx); err != nil {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	c := ligada.capsula.datos
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	asiento, existe := r.asientos[vinculo.huellaToken]
	if !existe || !ahora.Before(asiento.validoHasta) {
		delete(r.asientos, vinculo.huellaToken)
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	certificado := sha256.Sum256([]byte(c.presentacion.CertificadoSHA256))
	ca := sha256.Sum256([]byte(c.presentacion.CASHA256))
	if asiento.servicio != vinculo.servicio || asiento.sesionRef != vinculo.sesionRef ||
		asiento.generation != vinculo.generation ||
		c.resultado.SesionOriginal.SesionRef != asiento.sesionRef ||
		c.resultado.Recibo.SesionGeneracion != asiento.generation ||
		c.estadoActual.audiencia != asiento.audiencia ||
		subtle.ConstantTimeCompare(certificado[:], asiento.certificadoSHA256[:]) != 1 ||
		subtle.ConstantTimeCompare(ca[:], asiento.caSHA256[:]) != 1 {
		delete(r.asientos, vinculo.huellaToken)
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	return nil
}

func (v VinculoSesionMemoriaCertificado) validoPara(r *RegistroSesionMemoriaCertificado) bool {
	return r != nil && v.dueno == r && v.servicio != nil && v.marcador != nil &&
		v.sesionRef != "" && v.generation > 0 && v.huellaToken != ([sha256.Size]byte{})
}

// Revocar elimina un token mediante su vínculo opaco confiable.
func (r *RegistroSesionMemoriaCertificado) Revocar(vinculo VinculoSesionMemoriaCertificado) error {
	if !vinculo.validoPara(r) {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	delete(r.asientos, vinculo.huellaToken)
	return nil
}

// RevocarSesion corta todas las continuidades locales de la sesión original.
func (r *RegistroSesionMemoriaCertificado) RevocarSesion(vinculo VinculoSesionMemoriaCertificado) error {
	return r.revocarCoincidentes(vinculo, false)
}

// RevocarGeneracion corta solo la generación de sesión identificada por un
// vínculo previamente emitido o verificado; no altera el registro durable.
func (r *RegistroSesionMemoriaCertificado) RevocarGeneracion(vinculo VinculoSesionMemoriaCertificado) error {
	return r.revocarCoincidentes(vinculo, true)
}

func (r *RegistroSesionMemoriaCertificado) revocarCoincidentes(
	vinculo VinculoSesionMemoriaCertificado, soloGeneracion bool,
) error {
	if !vinculo.validoPara(r) {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cerrado {
		return ErrTokenSesionMemoriaCertificadoNoValido
	}
	for huella, asiento := range r.asientos {
		if asiento.servicio == vinculo.servicio && asiento.sesionRef == vinculo.sesionRef &&
			(!soloGeneracion || asiento.generation == vinculo.generation) {
			delete(r.asientos, huella)
		}
	}
	return nil
}

func (r *RegistroSesionMemoriaCertificado) Cerrar() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cerrado = true
	clear(r.asientos)
}
