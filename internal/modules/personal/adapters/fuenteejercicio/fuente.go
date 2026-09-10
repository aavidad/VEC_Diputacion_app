// Package fuenteejercicio resuelve vínculos sintéticos inmutables para el
// ejercicio local de Personal. No autoriza ni ejecuta altas.
package fuenteejercicio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"

	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	esquemaFuenteEjercicio = "vec.personal.fuente-ejercicio.v1"
	maximoBytesFuente      = 256 << 10
	maximoRegistros        = 256
	maximaProfundidadJSON  = 16
	maximoEnteroSeguro     = uint64(9_007_199_254_740_991)
)

var (
	ErrFuenteInvalida      = errors.New("personal: fuente de ejercicio inválida")
	ErrVinculoNoDisponible = errors.New("personal: vínculo de ejercicio no disponible")
	patronHuellaSHA256     = regexp.MustCompile(`^[a-f0-9]{64}$`)
	patronFechaCivil       = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// TernaEsperada procede de composición confiable y liga exactamente los bytes
// de la fuente. No admite una fuente remota ni una versión implícita.
type TernaEsperada struct {
	Referencia   string
	Version      uint64
	HuellaSHA256 string
}

// VinculoEjercicio es el mínimo dato sintético que un efecto futuro podría
// consumir. No representa permiso, relación jurídica, ocupación ni recibo.
type VinculoEjercicio struct {
	PersonaSinteticaRef     string
	CentroRef               string
	PuestoRef               string
	PlazaRef                string
	FuenteRPT               ctports.ReferenciaVersionadaPersonalRPT
	Desde                   string
	Hasta                   string
	AntecedenteEjercicioRef string
}

type documentoFuente struct {
	Esquema    string              `json:"esquema"`
	Referencia string              `json:"referencia"`
	Version    uint64              `json:"version"`
	FuenteRPT  fuenteRPTDocumento  `json:"fuente_rpt"`
	Registros  []registroDocumento `json:"registros"`
}

// fuenteRPTDocumento es la fuente gobernada de compatibilidades de ejercicio.
// Su huella se calcula sobre este material, sin una huella autorreferenciada.
type fuenteRPTDocumento struct {
	Referencia string      `json:"referencia"`
	Version    uint64      `json:"version"`
	Parejas    []parejaRPT `json:"parejas"`
}

type parejaRPT struct {
	PuestoRef string `json:"puesto_ref"`
	PlazaRef  string `json:"plaza_ref"`
	CentroRef string `json:"centro_ref"`
}

type registroDocumento struct {
	ExpedienteRef           string                                  `json:"expediente_ref"`
	VersionExpediente       uint64                                  `json:"version_expediente"`
	PersonaSinteticaRef     string                                  `json:"persona_sintetica_ref"`
	CentroRef               string                                  `json:"centro_ref"`
	PuestoRef               string                                  `json:"puesto_ref"`
	PlazaRef                string                                  `json:"plaza_ref"`
	FuenteRPT               ctports.ReferenciaVersionadaPersonalRPT `json:"fuente_rpt"`
	Desde                   string                                  `json:"desde"`
	Hasta                   string                                  `json:"hasta"`
	AntecedenteEjercicioRef string                                  `json:"antecedente_ejercicio_ref,omitempty"`
}

// Fuente es una instantánea validada y de solo lectura.
type Fuente struct {
	registros map[string]VinculoEjercicio
}

// NuevaFuenteEjercicio valida la huella de los bytes exactos antes de tratar
// su contenido. La terna se fija por composición, nunca por navegador.
func NuevaFuenteEjercicio(contenido []byte, esperada TernaEsperada) (*Fuente, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesFuente || !ternaValida(esperada) {
		return nil, ErrFuenteInvalida
	}
	suma := sha256.Sum256(contenido)
	if subtle.ConstantTimeCompare([]byte(hex.EncodeToString(suma[:])), []byte(esperada.HuellaSHA256)) != 1 {
		return nil, ErrFuenteInvalida
	}
	if err := validarJSONEstricto(contenido); err != nil {
		return nil, ErrFuenteInvalida
	}
	if err := validarClavesExactas(contenido); err != nil {
		return nil, ErrFuenteInvalida
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var documento documentoFuente
	if err := decodificador.Decode(&documento); err != nil || exigirFinJSON(decodificador) != nil {
		return nil, ErrFuenteInvalida
	}
	if documento.Esquema != esquemaFuenteEjercicio || documento.Referencia != esperada.Referencia ||
		documento.Version != esperada.Version || documento.Version > maximoEnteroSeguro ||
		!fuenteRPTValida(documento.FuenteRPT) || len(documento.Registros) == 0 || len(documento.Registros) > maximoRegistros {
		return nil, ErrFuenteInvalida
	}
	huellaRPT, err := huellaRPT(documento.FuenteRPT)
	if err != nil {
		return nil, ErrFuenteInvalida
	}
	fuente := &Fuente{registros: make(map[string]VinculoEjercicio, len(documento.Registros))}
	for _, registro := range documento.Registros {
		if !registroValido(registro) || registro.FuenteRPT.Referencia != documento.FuenteRPT.Referencia ||
			registro.FuenteRPT.Version != documento.FuenteRPT.Version || registro.FuenteRPT.HuellaSHA256 != huellaRPT ||
			!parejaCompatible(documento.FuenteRPT, registro.PuestoRef, registro.PlazaRef, registro.CentroRef) {
			return nil, ErrFuenteInvalida
		}
		clave := claveRegistro(registro.ExpedienteRef, registro.VersionExpediente)
		if _, existe := fuente.registros[clave]; existe {
			return nil, ErrFuenteInvalida
		}
		fuente.registros[clave] = VinculoEjercicio{
			PersonaSinteticaRef: registro.PersonaSinteticaRef, CentroRef: registro.CentroRef,
			PuestoRef: registro.PuestoRef, PlazaRef: registro.PlazaRef, FuenteRPT: registro.FuenteRPT,
			Desde: registro.Desde, Hasta: registro.Hasta, AntecedenteEjercicioRef: registro.AntecedenteEjercicioRef,
		}
	}
	return fuente, nil
}

// Resolver coteja un comando CT ya válido con el único vínculo exacto. La
// CapacidadRef se valida como sintaxis del contrato CT, pero no se interpreta.
func (f *Fuente) Resolver(ctx context.Context, solicitud ctports.SolicitudAltaPersonalRPT) (VinculoEjercicio, error) {
	if ctx == nil || f == nil {
		return VinculoEjercicio{}, ErrVinculoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return VinculoEjercicio{}, err
	}
	if err := solicitud.Validar(); err != nil {
		return VinculoEjercicio{}, ErrVinculoNoDisponible
	}
	vinculo, existe := f.registros[claveRegistro(solicitud.ExpedienteRef, solicitud.VersionExpediente)]
	if !existe || vinculo.FuenteRPT != solicitud.FuenteRPT || vinculo.PuestoRef != solicitud.PuestoRef || vinculo.PlazaRef != solicitud.PlazaRef {
		return VinculoEjercicio{}, ErrVinculoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return VinculoEjercicio{}, err
	}
	return vinculo, nil
}

func ternaValida(terna TernaEsperada) bool {
	return ctdomain.ReferenciaOpacaValida(terna.Referencia) && terna.Version > 0 && terna.Version <= maximoEnteroSeguro && patronHuellaSHA256.MatchString(terna.HuellaSHA256)
}

func registroValido(registro registroDocumento) bool {
	if !ctdomain.ReferenciaOpacaValida(registro.ExpedienteRef) || registro.VersionExpediente == 0 || registro.VersionExpediente > maximoEnteroSeguro ||
		!ctdomain.ReferenciaOpacaValida(registro.PersonaSinteticaRef) ||
		!ctdomain.ReferenciaOpacaValida(registro.CentroRef) || !ctdomain.ReferenciaOpacaValida(registro.PuestoRef) ||
		!ctdomain.ReferenciaOpacaValida(registro.PlazaRef) || registro.FuenteRPT.Validar() != nil ||
		!fechaCivilValida(registro.Desde) || !fechaCivilValida(registro.Hasta) || registro.Hasta < registro.Desde ||
		(registro.AntecedenteEjercicioRef != "" && !ctdomain.ReferenciaOpacaValida(registro.AntecedenteEjercicioRef)) {
		return false
	}
	return len(registro.PersonaSinteticaRef) >= len("persona:ejercicio:") && registro.PersonaSinteticaRef[:len("persona:ejercicio:")] == "persona:ejercicio:"
}

func fuenteRPTValida(fuente fuenteRPTDocumento) bool {
	if !ctdomain.ReferenciaOpacaValida(fuente.Referencia) || fuente.Version == 0 || fuente.Version > maximoEnteroSeguro ||
		len(fuente.Parejas) == 0 || len(fuente.Parejas) > maximoRegistros {
		return false
	}
	vistas := make(map[string]struct{}, len(fuente.Parejas))
	for _, pareja := range fuente.Parejas {
		if !ctdomain.ReferenciaOpacaValida(pareja.PuestoRef) || !ctdomain.ReferenciaOpacaValida(pareja.PlazaRef) ||
			!ctdomain.ReferenciaOpacaValida(pareja.CentroRef) {
			return false
		}
		clave := pareja.PuestoRef + "\x00" + pareja.PlazaRef + "\x00" + pareja.CentroRef
		if _, existe := vistas[clave]; existe {
			return false
		}
		vistas[clave] = struct{}{}
	}
	return true
}

func parejaCompatible(fuente fuenteRPTDocumento, puestoRef, plazaRef, centroRef string) bool {
	for _, pareja := range fuente.Parejas {
		if pareja.PuestoRef == puestoRef && pareja.PlazaRef == plazaRef && pareja.CentroRef == centroRef {
			return true
		}
	}
	return false
}

func huellaRPT(fuente fuenteRPTDocumento) (string, error) {
	if !fuenteRPTValida(fuente) {
		return "", ErrFuenteInvalida
	}
	constructor := constructorCanonico{}
	constructor.campo("rpt_ref", fuente.Referencia)
	constructor.entero("rpt_version", fuente.Version)
	constructor.entero("parejas", uint64(len(fuente.Parejas)))
	for _, pareja := range fuente.Parejas {
		constructor.campo("puesto_ref", pareja.PuestoRef)
		constructor.campo("plaza_ref", pareja.PlazaRef)
		constructor.campo("centro_ref", pareja.CentroRef)
	}
	suma := sha256.Sum256(constructor.Bytes())
	return hex.EncodeToString(suma[:]), nil
}

type constructorCanonico struct{ bytes.Buffer }

func (c *constructorCanonico) campo(nombre, valor string) {
	c.WriteString(strconv.Itoa(len(nombre)))
	c.WriteByte(':')
	c.WriteString(nombre)
	c.WriteString(strconv.Itoa(len(valor)))
	c.WriteByte(':')
	c.WriteString(valor)
}

func (c *constructorCanonico) entero(nombre string, valor uint64) {
	c.campo(nombre, strconv.FormatUint(valor, 10))
}

func fechaCivilValida(valor string) bool {
	if !patronFechaCivil.MatchString(valor) {
		return false
	}
	fecha, err := time.Parse("2006-01-02", valor)
	return err == nil && fecha.Format("2006-01-02") == valor
}

func claveRegistro(expedienteRef string, version uint64) string {
	return strconv.Itoa(len(expedienteRef)) + ":" + expedienteRef + ":" + strconv.FormatUint(version, 10)
}

func exigirFinJSON(decodificador *json.Decoder) error {
	var extra any
	if err := decodificador.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("json adicional")
	}
	return nil
}

func validarJSONEstricto(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	if err := validarValorJSON(decodificador, 0); err != nil {
		return err
	}
	return exigirFinJSON(decodificador)
}

func validarClavesExactas(contenido []byte) error {
	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &raiz); err != nil || !clavesPermitidas(raiz, "esquema", "referencia", "version", "fuente_rpt", "registros") {
		return errors.New("claves de fuente")
	}
	var fuente map[string]json.RawMessage
	if err := json.Unmarshal(raiz["fuente_rpt"], &fuente); err != nil || !clavesPermitidas(fuente, "referencia", "version", "parejas") {
		return errors.New("claves de rpt")
	}
	var parejas []json.RawMessage
	if err := json.Unmarshal(fuente["parejas"], &parejas); err != nil {
		return errors.New("parejas rpt")
	}
	for _, valor := range parejas {
		var pareja map[string]json.RawMessage
		if err := json.Unmarshal(valor, &pareja); err != nil || !clavesPermitidas(pareja, "puesto_ref", "plaza_ref", "centro_ref") {
			return errors.New("claves de pareja")
		}
	}
	var registros []json.RawMessage
	if err := json.Unmarshal(raiz["registros"], &registros); err != nil {
		return errors.New("registros")
	}
	for _, valor := range registros {
		var registro map[string]json.RawMessage
		if err := json.Unmarshal(valor, &registro); err != nil || !clavesPermitidas(registro,
			"expediente_ref", "version_expediente", "persona_sintetica_ref", "centro_ref", "puesto_ref", "plaza_ref", "fuente_rpt", "desde", "hasta", "antecedente_ejercicio_ref") {
			return errors.New("claves de registro")
		}
		if antecedente, existe := registro["antecedente_ejercicio_ref"]; existe && bytes.Equal(bytes.TrimSpace(antecedente), []byte("null")) {
			return errors.New("antecedente nulo")
		}
		var terna map[string]json.RawMessage
		if err := json.Unmarshal(registro["fuente_rpt"], &terna); err != nil || !clavesPermitidas(terna, "referencia", "version", "huella_sha256") {
			return errors.New("claves de terna")
		}
	}
	return nil
}

func clavesPermitidas(campos map[string]json.RawMessage, permitidas ...string) bool {
	if campos == nil {
		return false
	}
	permitidasPorNombre := make(map[string]struct{}, len(permitidas))
	for _, clave := range permitidas {
		permitidasPorNombre[clave] = struct{}{}
	}
	for clave := range campos {
		if _, existe := permitidasPorNombre[clave]; !existe {
			return false
		}
	}
	return true
}

func validarValorJSON(decodificador *json.Decoder, profundidad int) error {
	if profundidad > maximaProfundidadJSON {
		return errors.New("json profundo")
	}
	token, err := decodificador.Token()
	if err != nil {
		return err
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := make(map[string]struct{})
		for decodificador.More() {
			tokenClave, err := decodificador.Token()
			clave, ok := tokenClave.(string)
			if err != nil || !ok {
				return errors.New("clave json")
			}
			if _, duplicada := claves[clave]; duplicada {
				return errors.New("clave json duplicada")
			}
			claves[clave] = struct{}{}
			if err := validarValorJSON(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		_, err = decodificador.Token()
		return err
	case '[':
		for decodificador.More() {
			if err := validarValorJSON(decodificador, profundidad+1); err != nil {
				return err
			}
		}
		_, err = decodificador.Token()
		return err
	default:
		return errors.New("delimitador json")
	}
}
