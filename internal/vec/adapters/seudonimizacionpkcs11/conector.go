// Package seudonimizacionpkcs11 seudonimiza identificadores de sesiones con una
// clave HMAC-SHA-256 que permanece dentro de un token PKCS#11 provisionado.
package seudonimizacionpkcs11

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"unicode/utf8"

	"github.com/miekg/pkcs11"

	registro "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

// Configuracion identifica una clave existente. No incluye material de clave
// ni permite crear o sustituir objetos del token durante el arranque.
type Configuracion struct {
	Modulo           string
	TokenLabel       string
	TokenSerial      string
	ObjetoID         []byte
	ClaveID          string
	ClaveVersion     uint64
	DominioRef       string
	EspacioIdentidad string
	PINFichero       string
}

// Conector serializa las operaciones sobre una sesion PKCS#11. Nunca lee
// CKA_VALUE ni conserva el PIN despues del login.
type Conector struct {
	mu      sync.Mutex
	ctx     *pkcs11.Ctx
	slot    uint
	sesion  pkcs11.SessionHandle
	clave   pkcs11.ObjectHandle
	config  Configuracion
	cerrado bool
}

var _ registro.SeudonimizadorAlta = (*Conector)(nil)

// Abrir valida el proveedor y la clave antes de entregar el conector. Cualquier
// dependencia ausente impide arrancar la frontera de identidad.
func Abrir(config Configuracion) (_ *Conector, err error) {
	if err := validarConfiguracion(config); err != nil {
		return nil, err
	}
	ctx := pkcs11.New(config.Modulo)
	if ctx == nil {
		return nil, errors.New("modulo PKCS#11 no disponible")
	}
	if err := ctx.Initialize(); err != nil {
		ctx.Destroy()
		return nil, errors.New("no se pudo inicializar PKCS#11")
	}
	conector := &Conector{ctx: ctx, config: config}
	conector.config.ObjetoID = append([]byte(nil), config.ObjetoID...)
	defer func() {
		if err != nil {
			_ = conector.Cerrar()
		}
	}()

	slots, fallo := ctx.GetSlotList(true)
	if fallo != nil {
		return nil, errors.New("no se pudieron consultar los tokens PKCS#11")
	}
	var coincidencias []uint
	for _, slot := range slots {
		info, fallo := ctx.GetTokenInfo(slot)
		if fallo != nil {
			return nil, errors.New("token PKCS#11 inaccesible")
		}
		if strings.TrimSpace(info.Label) == config.TokenLabel &&
			strings.TrimSpace(info.SerialNumber) == config.TokenSerial {
			coincidencias = append(coincidencias, slot)
		}
	}
	if len(coincidencias) != 1 {
		return nil, errors.New("token PKCS#11 ausente o ambiguo")
	}
	conector.slot = coincidencias[0]
	if fallo := verificarMecanismo(ctx, conector.slot); fallo != nil {
		return nil, fallo
	}
	conector.sesion, fallo = ctx.OpenSession(conector.slot, pkcs11.CKF_SERIAL_SESSION)
	if fallo != nil {
		return nil, errors.New("no se pudo abrir sesion PKCS#11")
	}
	pin, fallo := leerPIN(config.PINFichero)
	if fallo != nil {
		return nil, fallo
	}
	fallo = ctx.Login(conector.sesion, pkcs11.CKU_USER, string(pin))
	for i := range pin {
		pin[i] = 0
	}
	if fallo != nil {
		return nil, errors.New("login PKCS#11 rechazado")
	}
	conector.clave, fallo = buscarClave(ctx, conector.sesion, config.ObjetoID)
	if fallo != nil {
		return nil, fallo
	}
	if fallo = verificarAtributos(ctx, conector.sesion, conector.clave); fallo != nil {
		return nil, fallo
	}
	primera, fallo := conector.firmar([]byte("vec.pkcs11.preflight.hmac-sha256.v1"))
	if fallo != nil {
		return nil, errors.New("preflight HMAC PKCS#11 fallido")
	}
	segunda, fallo := conector.firmar([]byte("vec.pkcs11.preflight.hmac-sha256.v1"))
	if fallo != nil || primera != segunda || primera == [32]byte{} {
		return nil, errors.New("preflight HMAC PKCS#11 no determinista")
	}
	return conector, nil
}

func validarConfiguracion(c Configuracion) error {
	if c.Modulo == "" || c.PINFichero == "" ||
		!textoTecnico(c.TokenLabel, 32) || !textoTecnico(c.TokenSerial, 32) ||
		len(c.ObjetoID) == 0 || len(c.ObjetoID) > 64 ||
		!textoTecnico(c.ClaveID, 128) || c.ClaveVersion == 0 ||
		c.ClaveVersion > 1<<63-1 || !dominioValido(c.DominioRef) ||
		!espacioValido(c.EspacioIdentidad) {
		return errors.New("configuracion PKCS#11 invalida")
	}
	return nil
}

func textoTecnico(s string, max int) bool {
	if s == "" || len(s) > max || s != strings.TrimSpace(s) || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if r <= 0x20 || r >= 0x7f {
			return false
		}
	}
	return true
}

func dominioValido(s string) bool {
	if !strings.HasPrefix(s, "idh_") || len(s) < 26 || len(s) > 132 {
		return false
	}
	for _, r := range s[4:] {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func espacioValido(s string) bool {
	if !textoTecnico(s, 512) {
		return false
	}
	u, err := url.Parse(s)
	return err == nil && strings.EqualFold(u.Scheme, "https") &&
		u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
}

func leerPIN(ruta string) ([]byte, error) {
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() < 1 || info.Size() > 256 {
		return nil, errors.New("fichero PIN PKCS#11 invalido")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) && stat.Uid != 0 {
		return nil, errors.New("propietario de fichero PIN PKCS#11 invalido")
	}
	f, err := os.Open(ruta)
	if err != nil {
		return nil, errors.New("no se pudo abrir fichero PIN PKCS#11")
	}
	defer f.Close()
	actual, err := f.Stat()
	if err != nil || !os.SameFile(info, actual) {
		return nil, errors.New("fichero PIN PKCS#11 cambiado")
	}
	pin := make([]byte, info.Size())
	if _, err = io.ReadFull(f, pin); err != nil {
		return nil, errors.New("no se pudo leer PIN PKCS#11")
	}
	if len(pin) > 0 && pin[len(pin)-1] == '\n' {
		pin[len(pin)-1] = 0
		pin = pin[:len(pin)-1]
	}
	if len(pin) == 0 || len(pin) > 256 {
		return nil, errors.New("PIN PKCS#11 invalido")
	}
	for _, caracter := range pin {
		if caracter < 0x21 || caracter > 0x7e {
			clear(pin)
			return nil, errors.New("PIN PKCS#11 invalido")
		}
	}
	return pin, nil
}

func verificarMecanismo(ctx *pkcs11.Ctx, slot uint) error {
	mecanismos, err := ctx.GetMechanismList(slot)
	if err != nil {
		return errors.New("no se pudieron consultar mecanismos PKCS#11")
	}
	for _, m := range mecanismos {
		if m.Mechanism == pkcs11.CKM_SHA256_HMAC {
			info, err := ctx.GetMechanismInfo(slot, []*pkcs11.Mechanism{m})
			if err == nil && info.Flags&pkcs11.CKF_SIGN != 0 {
				return nil
			}
		}
	}
	return errors.New("mecanismo HMAC-SHA-256 PKCS#11 ausente")
}

func buscarClave(ctx *pkcs11.Ctx, sesion pkcs11.SessionHandle, id []byte) (pkcs11.ObjectHandle, error) {
	plantilla := []*pkcs11.Attribute{
		pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_SECRET_KEY),
		pkcs11.NewAttribute(pkcs11.CKA_ID, id),
	}
	if err := ctx.FindObjectsInit(sesion, plantilla); err != nil {
		return 0, errors.New("busqueda de clave PKCS#11 fallida")
	}
	objetos, _, err := ctx.FindObjects(sesion, 2)
	finalErr := ctx.FindObjectsFinal(sesion)
	if err != nil || finalErr != nil || len(objetos) != 1 {
		return 0, errors.New("clave PKCS#11 ausente o ambigua")
	}
	return objetos[0], nil
}

func verificarAtributos(ctx *pkcs11.Ctx, sesion pkcs11.SessionHandle, clave pkcs11.ObjectHandle) error {
	condiciones := []struct {
		atributo uint
		valor    byte
	}{
		{pkcs11.CKA_TOKEN, 1}, {pkcs11.CKA_PRIVATE, 1}, {pkcs11.CKA_SIGN, 1},
		{pkcs11.CKA_SENSITIVE, 1}, {pkcs11.CKA_ALWAYS_SENSITIVE, 1},
		{pkcs11.CKA_EXTRACTABLE, 0}, {pkcs11.CKA_NEVER_EXTRACTABLE, 1},
	}
	for _, condicion := range condiciones {
		attrs, err := ctx.GetAttributeValue(sesion, clave, []*pkcs11.Attribute{{Type: condicion.atributo}})
		if err != nil || len(attrs) != 1 || len(attrs[0].Value) != 1 || attrs[0].Value[0] != condicion.valor {
			return errors.New("atributos de clave PKCS#11 incompatibles")
		}
	}
	tipo, err := ctx.GetAttributeValue(sesion, clave, []*pkcs11.Attribute{{Type: pkcs11.CKA_KEY_TYPE}})
	if err != nil || len(tipo) != 1 || !ulongIgual(tipo[0].Value, pkcs11.CKK_GENERIC_SECRET) {
		return errors.New("tipo de clave PKCS#11 incompatible")
	}
	longitud, err := ctx.GetAttributeValue(sesion, clave, []*pkcs11.Attribute{{Type: pkcs11.CKA_VALUE_LEN}})
	if err != nil || len(longitud) != 1 || ulong(longitud[0].Value) < 32 {
		return errors.New("longitud de clave PKCS#11 insuficiente")
	}
	return nil
}

func ulongIgual(b []byte, n uint) bool { return ulong(b) == uint64(n) }

func ulong(b []byte) uint64 {
	switch len(b) {
	case 4:
		return uint64(binary.NativeEndian.Uint32(b))
	case 8:
		return binary.NativeEndian.Uint64(b)
	default:
		return 0
	}
}

func (c *Conector) firmar(mensaje []byte) ([32]byte, error) {
	var resultado [32]byte
	if err := c.ctx.SignInit(c.sesion, []*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_SHA256_HMAC, nil)}, c.clave); err != nil {
		return resultado, err
	}
	firma, err := c.ctx.Sign(c.sesion, mensaje)
	if err != nil || len(firma) != len(resultado) {
		return resultado, errors.New("firma HMAC PKCS#11 invalida")
	}
	copy(resultado[:], firma)
	return resultado, nil
}

// SeudonimizarAlta aplica separacion de dominio y de los cinco propositos.
// Las cadenas fuente solo cruzan el limite del modulo como mensaje de HMAC.
func (c *Conector) SeudonimizarAlta(ctx context.Context, entrada registro.IdentificadoresAlta) (registro.SeudonimosAlta, error) {
	var salida registro.SeudonimosAlta
	if err := ctx.Err(); err != nil {
		return salida, err
	}
	if entrada.EspacioIdentidad != c.config.EspacioIdentidad ||
		!identificadorValido(entrada.AsercionID) || !identificadorValido(entrada.SesionID) ||
		!identificadorValido(entrada.SujetoID) || !identificadorValido(entrada.CuentaID) ||
		entrada.CuentaOrdinariaID != "" && !identificadorValido(entrada.CuentaOrdinariaID) {
		return salida, errors.New("identificadores de alta invalidos")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cerrado {
		return salida, errors.New("conector PKCS#11 cerrado")
	}
	info, err := c.ctx.GetTokenInfo(c.slot)
	if err != nil || strings.TrimSpace(info.Label) != c.config.TokenLabel ||
		strings.TrimSpace(info.SerialNumber) != c.config.TokenSerial {
		return salida, errors.New("token PKCS#11 retirado")
	}
	salida = registro.SeudonimosAlta{
		Esquema:          registro.EsquemaHMACSHA256V1,
		EspacioIdentidad: c.config.EspacioIdentidad,
		DominioRef:       c.config.DominioRef,
		ClaveID:          c.config.ClaveID,
		ClaveVersion:     c.config.ClaveVersion,
	}
	valores := []struct {
		proposito string
		valor     string
		destino   *[32]byte
	}{
		{"asercion", entrada.AsercionID, &salida.AsercionIDHMAC},
		{"sesion", entrada.SesionID, &salida.SesionIDHMAC},
		{"sujeto", entrada.SujetoID, &salida.SujetoIDHMAC},
		{"cuenta", entrada.CuentaID, &salida.CuentaIDHMAC},
		{"cuenta_ordinaria", entrada.CuentaOrdinariaID, &salida.CuentaOrdinariaIDHMAC},
	}
	for _, valor := range valores {
		if valor.valor == "" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return registro.SeudonimosAlta{}, err
		}
		mensaje := mensajeCanonico(c.config, valor.proposito, valor.valor)
		firma, err := c.firmar(mensaje)
		clear(mensaje)
		if err != nil {
			return registro.SeudonimosAlta{}, errors.New("HMAC PKCS#11 fallido")
		}
		*valor.destino = firma
	}
	return salida, nil
}

func identificadorValido(s string) bool {
	return s != "" && len(s) <= 4096 && utf8.ValidString(s)
}

func mensajeCanonico(c Configuracion, proposito, valor string) []byte {
	partes := []string{
		registro.EsquemaHMACSHA256V1, c.DominioRef, c.EspacioIdentidad,
		fmt.Sprintf("%d", c.ClaveVersion), proposito, valor,
	}
	mensaje := make([]byte, 0, 128+len(valor))
	var longitud [4]byte
	for _, parte := range partes {
		binary.BigEndian.PutUint32(longitud[:], uint32(len(parte)))
		mensaje = append(mensaje, longitud[:]...)
		mensaje = append(mensaje, parte...)
	}
	return mensaje
}

// Cerrar libera sesion, login y proveedor. Se invoca al terminar el servidor.
func (c *Conector) Cerrar() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cerrado {
		return nil
	}
	c.cerrado = true
	if c.ctx == nil {
		return nil
	}
	if c.sesion != 0 {
		_ = c.ctx.Logout(c.sesion)
		_ = c.ctx.CloseSession(c.sesion)
	}
	err := c.ctx.Finalize()
	c.ctx.Destroy()
	c.ctx = nil
	if err != nil {
		return errors.New("cierre PKCS#11 fallido")
	}
	return nil
}
