// Package seguridad implementa los puertos criptográficos de Mis correos.
// Las claves proceden de una fuente inyectada; este paquete no configura KMS.
package seguridad

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

var ErrCorreosCriptoNoDisponible = errors.New("usuarios correos: criptografia no disponible")

// ClaveCorreo representa material ya obtenido por el KMS inyectado. Una clave
// revocada se rechaza incluso si su periodo de retención aún no ha terminado.
type ClaveCorreo struct {
	Ref          string
	Material     [32]byte
	RetenerHasta time.Time
	Revocada     bool
}

// ClavesCorreos mantiene ámbitos criptográficos independientes. Igualdad debe
// permanecer estable hasta reindexar de forma transaccional las huellas SQL.
type ClavesCorreos struct {
	CifradoActivo, Igualdad, SemanticaActiva, CodigoActivo ClaveCorreo
	CifradoRetenidas, SemanticaRetenidas, CodigoRetenidas  []ClaveCorreo
}

type FuenteClavesCorreos interface {
	CargarClavesCorreos(context.Context) (ClavesCorreos, error)
}

type AdaptadorCorreos struct {
	fuente FuenteClavesCorreos
	ahora  func() time.Time
}

func NuevoAdaptadorCorreos(fuente FuenteClavesCorreos, ahora func() time.Time) (*AdaptadorCorreos, error) {
	if fuente == nil || ahora == nil {
		return nil, ErrCorreosCriptoNoDisponible
	}
	return &AdaptadorCorreos{fuente: fuente, ahora: ahora}, nil
}

func (a *AdaptadorCorreos) claves(ctx context.Context) (ClavesCorreos, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || a.fuente == nil || a.ahora == nil {
		return ClavesCorreos{}, ErrCorreosCriptoNoDisponible
	}
	c, err := a.fuente.CargarClavesCorreos(ctx)
	if err != nil || ctx.Err() != nil || !clavesValidas(c) {
		return ClavesCorreos{}, ErrCorreosCriptoNoDisponible
	}
	return c, nil
}

func clavesValidas(c ClavesCorreos) bool {
	if len(c.SemanticaRetenidas) > 8 || len(c.CifradoRetenidas) > 8 || len(c.CodigoRetenidas) > 8 {
		return false
	}
	grupos := [][]ClaveCorreo{{c.CifradoActivo}, {c.Igualdad}, {c.SemanticaActiva}, {c.CodigoActivo}, c.CifradoRetenidas, c.SemanticaRetenidas, c.CodigoRetenidas}
	vistos := make(map[string]bool)
	materiales := make(map[[32]byte]bool)
	for _, grupo := range grupos {
		for _, k := range grupo {
			if !refValida(k.Ref) || k.Material == ([32]byte{}) || vistos[k.Ref] || materiales[k.Material] {
				return false
			}
			vistos[k.Ref], materiales[k.Material] = true, true
		}
	}
	for _, grupo := range [][]ClaveCorreo{c.CifradoRetenidas, c.CodigoRetenidas} {
		for _, k := range grupo {
			if k.RetenerHasta.IsZero() {
				return false
			}
		}
	}
	return true
}

func claveUsable(k ClaveCorreo, ahora time.Time) bool {
	return !k.Revocada && (k.RetenerHasta.IsZero() || ahora.Before(k.RetenerHasta))
}

func buscarClave(activa ClaveCorreo, retenidas []ClaveCorreo, ref string, ahora time.Time) (ClaveCorreo, bool) {
	if activa.Ref == ref && claveUsable(activa, ahora) {
		return activa, true
	}
	for _, k := range retenidas {
		if k.Ref == ref && claveUsable(k, ahora) {
			return k, true
		}
	}
	return ClaveCorreo{}, false
}

func refValida(s string) bool {
	if len(s) < 8 || len(s) > 128 {
		return false
	}
	for _, b := range []byte(s) {
		if b != ':' && b != '-' && b != '_' && (b < '0' || b > '9') && (b < 'a' || b > 'z') && (b < 'A' || b > 'Z') {
			return false
		}
	}
	return true
}

func referenciasValidas(persona, correo string) bool { return refValida(persona) && refValida(correo) }

// codificarCampos usa longitudes binarias y evita preimágenes ambiguas.
func codificarCampos(campos ...[]byte) []byte {
	var b []byte
	for _, campo := range campos {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(campo)))
		b = append(b, n[:]...)
		b = append(b, campo...)
	}
	return b
}

func versionBytes(version uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], version)
	return b[:]
}

func aadDireccion(persona, correo string, version uint64) []byte {
	return codificarCampos([]byte("vec.usuarios.correos.direccion.aead.v1"), []byte(persona), []byte(correo), versionBytes(version))
}

func mac(clave [32]byte, campos ...[]byte) []byte {
	h := hmac.New(sha256.New, clave[:])
	_, _ = h.Write(codificarCampos(campos...))
	return h.Sum(nil)
}

func (a *AdaptadorCorreos) CifrarDireccionCorreo(ctx context.Context, persona, correo string, version uint64, claro []byte) (ports.SobreDireccionCorreo, error) {
	if !referenciasValidas(persona, correo) || version == 0 || !domain.DireccionCorreoValida(string(claro)) {
		return ports.SobreDireccionCorreo{}, ErrCorreosCriptoNoDisponible
	}
	c, err := a.claves(ctx)
	if err != nil || !claveUsable(c.CifradoActivo, a.ahora()) || !claveUsable(c.Igualdad, a.ahora()) {
		return ports.SobreDireccionCorreo{}, ErrCorreosCriptoNoDisponible
	}
	bloque, err := aes.NewCipher(c.CifradoActivo.Material[:])
	if err != nil {
		return ports.SobreDireccionCorreo{}, ErrCorreosCriptoNoDisponible
	}
	g, err := cipher.NewGCM(bloque)
	if err != nil {
		return ports.SobreDireccionCorreo{}, ErrCorreosCriptoNoDisponible
	}
	nonce := make([]byte, g.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return ports.SobreDireccionCorreo{}, ErrCorreosCriptoNoDisponible
	}
	cifrado := g.Seal(nil, nonce, claro, aadDireccion(persona, correo, version))
	if ctx.Err() != nil {
		return ports.SobreDireccionCorreo{}, ErrCorreosCriptoNoDisponible
	}
	igualdad := mac(c.Igualdad.Material, []byte("vec.usuarios.correos.igualdad.v1"), []byte(strings.ToLower(string(claro))))
	return ports.SobreDireccionCorreo{CorreoRef: correo, Version: version, ClaveRef: c.CifradoActivo.Ref, Nonce: nonce, Cifrado: cifrado, HuellaIgualdad: igualdad}, nil
}

// ConDireccionCorreoDescifrada limita la exposición del claro a un callback.
func (a *AdaptadorCorreos) ConDireccionCorreoDescifrada(ctx context.Context, persona string, sobre ports.SobreDireccionCorreo, usar func([]byte) error) error {
	if usar == nil || !referenciasValidas(persona, sobre.CorreoRef) || sobre.Version == 0 || len(sobre.Nonce) != 12 || len(sobre.Cifrado) < 19 || len(sobre.Cifrado) > 270 {
		return ErrCorreosCriptoNoDisponible
	}
	c, err := a.claves(ctx)
	if err != nil {
		return err
	}
	k, ok := buscarClave(c.CifradoActivo, c.CifradoRetenidas, sobre.ClaveRef, a.ahora())
	if !ok {
		return ErrCorreosCriptoNoDisponible
	}
	bloque, err := aes.NewCipher(k.Material[:])
	if err != nil {
		return ErrCorreosCriptoNoDisponible
	}
	g, err := cipher.NewGCM(bloque)
	if err != nil {
		return ErrCorreosCriptoNoDisponible
	}
	claro, err := g.Open(nil, sobre.Nonce, sobre.Cifrado, aadDireccion(persona, sobre.CorreoRef, sobre.Version))
	if err != nil || ctx.Err() != nil {
		return ErrCorreosCriptoNoDisponible
	}
	defer borrar(claro)
	if !domain.DireccionCorreoValida(string(claro)) || usar(claro) != nil || ctx.Err() != nil {
		return ErrCorreosCriptoNoDisponible
	}
	return nil
}

func borrar(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

type preimagenSemantica struct {
	Esquema, Persona, Accion, CorreoRef, Direccion, Codigo, SustitutoRef string
	Version                                                              uint64
}

func (a *AdaptadorCorreos) SellarHuellaCorreo(ctx context.Context, preimagen []byte) (ports.HuellasSemanticasCorreo, error) {
	if len(preimagen) == 0 || len(preimagen) > 2048 {
		return ports.HuellasSemanticasCorreo{}, ErrCorreosCriptoNoDisponible
	}
	var p preimagenSemantica
	if json.Unmarshal(preimagen, &p) != nil || p.Esquema != "usuarios.correos.peticion.v2" || !refValida(p.Persona) || p.Version > 1<<63-1 || !accionCorreoValida(p.Accion) {
		return ports.HuellasSemanticasCorreo{}, ErrCorreosCriptoNoDisponible
	}
	canonico, err := json.Marshal(p)
	if err != nil || !bytes.Equal(canonico, preimagen) || !preimagenCorreoValida(p) {
		return ports.HuellasSemanticasCorreo{}, ErrCorreosCriptoNoDisponible
	}
	c, err := a.claves(ctx)
	if err != nil || !claveUsable(c.SemanticaActiva, a.ahora()) {
		return ports.HuellasSemanticasCorreo{}, ErrCorreosCriptoNoDisponible
	}
	sellar := func(k ClaveCorreo) ports.HuellaSemanticaCorreo {
		return ports.HuellaSemanticaCorreo{ClaveRef: k.Ref, Valor: hex.EncodeToString(mac(k.Material, []byte("vec.usuarios.correos.peticion.v1"), preimagen))}
	}
	h := ports.HuellasSemanticasCorreo{Activa: sellar(c.SemanticaActiva)}
	for _, k := range c.SemanticaRetenidas {
		if claveUsable(k, a.ahora()) {
			h.Retenidas = append(h.Retenidas, sellar(k))
		}
	}
	if ctx.Err() != nil {
		return ports.HuellasSemanticasCorreo{}, ErrCorreosCriptoNoDisponible
	}
	return h, nil
}

func preimagenCorreoValida(p preimagenSemantica) bool {
	switch p.Accion {
	case ports.AccionAnadirCorreo:
		return p.CorreoRef == "" && domain.DireccionCorreoValida(p.Direccion) && p.Codigo == "" && p.SustitutoRef == ""
	case ports.AccionReenviarCorreo, ports.AccionActivarCorreo:
		return refValida(p.CorreoRef) && p.Direccion == "" && p.Codigo == "" && p.SustitutoRef == ""
	case ports.AccionVerificarCorreo:
		return refValida(p.CorreoRef) && p.Direccion == "" && len(p.Codigo) >= 16 && len(p.Codigo) <= 128 && p.SustitutoRef == ""
	case ports.AccionRetirarCorreo:
		return refValida(p.CorreoRef) && p.Direccion == "" && p.Codigo == "" && (p.SustitutoRef == "" || (refValida(p.SustitutoRef) && p.SustitutoRef != p.CorreoRef))
	}
	return false
}

func accionCorreoValida(s string) bool {
	switch s {
	case ports.AccionAnadirCorreo, ports.AccionReenviarCorreo, ports.AccionVerificarCorreo, ports.AccionActivarCorreo, ports.AccionRetirarCorreo:
		return true
	}
	return false
}

func vencimientoValido(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Nanosecond()%1000 == 0
}

func (a *AdaptadorCorreos) PrepararDesafioCorreo(ctx context.Context, persona, correo string, vence time.Time) (ports.ReservaDesafio, error) {
	if a == nil || a.ahora == nil || !referenciasValidas(persona, correo) || !vencimientoValido(vence) || !a.ahora().Before(vence) {
		return ports.ReservaDesafio{}, ErrCorreosCriptoNoDisponible
	}
	c, err := a.claves(ctx)
	if err != nil || !claveUsable(c.CodigoActivo, a.ahora()) {
		return ports.ReservaDesafio{}, ErrCorreosCriptoNoDisponible
	}
	desafio := make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, desafio); err != nil {
		return ports.ReservaDesafio{}, ErrCorreosCriptoNoDisponible
	}
	ref := "desafio:" + hex.EncodeToString(desafio)
	_, huella := derivarCodigo(c.CodigoActivo.Material, persona, correo, desafio, vence)
	if ctx.Err() != nil {
		borrar(desafio)
		return ports.ReservaDesafio{}, ErrCorreosCriptoNoDisponible
	}
	return ports.ReservaDesafio{DesafioRef: ref, Desafio: desafio, HuellaCodigo: huella, ClaveRef: c.CodigoActivo.Ref, VenceUTC: vence}, nil
}

func derivarCodigo(clave [32]byte, persona, correo string, desafio []byte, vence time.Time) (string, []byte) {
	instant := []byte(vence.Format("2006-01-02T15:04:05.000000Z"))
	bytesCodigo := mac(clave, []byte("vec.usuarios.correos.codigo.v1"), []byte(persona), []byte(correo), desafio, instant)
	// 16 bytes efectivos: 128 bits y 26 caracteres para introducir a mano.
	codigo := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytesCodigo[:16])
	huella := mac(clave, []byte("vec.usuarios.correos.huella-codigo.v1"), []byte(persona), []byte(correo), desafio, instant, []byte(codigo))
	borrar(bytesCodigo)
	return codigo, huella
}

func desafioDesdeRef(ref string) ([]byte, bool) {
	if !strings.HasPrefix(ref, "desafio:") || len(ref) != len("desafio:")+64 {
		return nil, false
	}
	b, err := hex.DecodeString(strings.TrimPrefix(ref, "desafio:"))
	return b, err == nil && len(b) == 32
}

func (a *AdaptadorCorreos) ComprobarCodigoCorreo(ctx context.Context, m ports.MetadatosDesafioCorreo, codigo string) (bool, error) {
	desafio, ok := desafioDesdeRef(m.DesafioRef)
	if a == nil || a.ahora == nil || !ok || !referenciasValidas(m.PersonaRef, m.CorreoRef) || !vencimientoValido(m.VenceUTC) || !a.ahora().Before(m.VenceUTC) || len(m.HuellaCodigo) != 32 || len(codigo) != 26 {
		return false, ErrCorreosCriptoNoDisponible
	}
	c, err := a.claves(ctx)
	if err != nil {
		return false, err
	}
	k, ok := buscarClave(c.CodigoActivo, c.CodigoRetenidas, m.ClaveRef, a.ahora())
	if !ok {
		return false, ErrCorreosCriptoNoDisponible
	}
	esperado, huella := derivarCodigo(k.Material, m.PersonaRef, m.CorreoRef, desafio, m.VenceUTC)
	acierto := subtle.ConstantTimeCompare([]byte(codigo), []byte(esperado)) & subtle.ConstantTimeCompare(m.HuellaCodigo, huella)
	if ctx.Err() != nil {
		return false, ErrCorreosCriptoNoDisponible
	}
	return acierto == 1, nil
}

func (a *AdaptadorCorreos) DerivarCodigoCorreo(ctx context.Context, d ports.DespachoVerificacionCorreo) (string, error) {
	desafio, ok := desafioDesdeRef(d.DesafioRef)
	if a == nil || a.ahora == nil || !ok || subtle.ConstantTimeCompare(desafio, d.Desafio) != 1 || !referenciasValidas(d.PersonaRef, d.CorreoRef) || !vencimientoValido(d.VenceUTC) || !a.ahora().Before(d.VenceUTC) || !refValida(d.OutboxRef) {
		return "", ErrCorreosCriptoNoDisponible
	}
	c, err := a.claves(ctx)
	if err != nil {
		return "", err
	}
	k, ok := buscarClave(c.CodigoActivo, c.CodigoRetenidas, d.ClaveRef, a.ahora())
	if !ok {
		return "", ErrCorreosCriptoNoDisponible
	}
	codigo, _ := derivarCodigo(k.Material, d.PersonaRef, d.CorreoRef, desafio, d.VenceUTC)
	if ctx.Err() != nil {
		return "", ErrCorreosCriptoNoDisponible
	}
	return codigo, nil
}

var _ ports.ProtectorDireccionCorreo = (*AdaptadorCorreos)(nil)
var _ ports.SelladorHuellaCorreos = (*AdaptadorCorreos)(nil)
var _ ports.PreparadorDesafioCorreo = (*AdaptadorCorreos)(nil)
var _ ports.ValidadorCodigoCorreo = (*AdaptadorCorreos)(nil)
var _ ports.DerivadorCodigoDespachoCorreo = (*AdaptadorCorreos)(nil)
