package domain

import (
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
)

// B4 (Peticion.pdf p.1, «Candidatos: correo electrónico y dos teléfonos»; pliego
// SE15/2020 §1 c.2): datos de contacto de una participación en bolsa. Son datos
// personales: el agregado solo viaja cifrado fuera del proceso (sobre AEAD ligado
// a la participación y a la versión) y las lecturas ordinarias reciben la forma
// enmascarada. Versionado append-only: cada registro es una versión nueva.

var ErrDatosContactoParticipacionInvalidos = errors.New("bolsa: datos de contacto de participacion invalidos")

const (
	longitudMaximaCorreoContacto   = 254
	longitudMaximaTelefonoContacto = 16
)

type DatosContactoParticipacion struct {
	ParticipacionRef string
	Correo           string
	Telefono1        string
	Telefono2        string
}

// canonicoDatosContacto es la única forma serializada del agregado; se cifra
// entera y nunca se persiste en claro.
type canonicoDatosContacto struct {
	Esquema   string `json:"esquema"`
	Correo    string `json:"correo"`
	Telefono1 string `json:"telefono_1"`
	Telefono2 string `json:"telefono_2"`
}

const EsquemaDatosContactoParticipacion = "vec.bolsa.datos-contacto-participacion.v1"

// Normalizar aplica las normalizaciones admitidas antes de validar: recorte,
// correo en minúsculas en el dominio y teléfonos sin espacios ni separadores.
func (d DatosContactoParticipacion) Normalizar() DatosContactoParticipacion {
	d.ParticipacionRef = strings.TrimSpace(d.ParticipacionRef)
	d.Correo = normalizarCorreoContacto(d.Correo)
	d.Telefono1 = normalizarTelefonoContacto(d.Telefono1)
	d.Telefono2 = normalizarTelefonoContacto(d.Telefono2)
	return d
}

func (d DatosContactoParticipacion) Validar() error {
	if !referenciaLlamamientoOpacaValida(d.ParticipacionRef) {
		return ErrDatosContactoParticipacionInvalidos
	}
	if d.Correo == "" && d.Telefono1 == "" {
		return ErrDatosContactoParticipacionInvalidos
	}
	if d.Correo != "" && !correoContactoValido(d.Correo) {
		return ErrDatosContactoParticipacionInvalidos
	}
	if d.Telefono1 != "" && !telefonoContactoValido(d.Telefono1) {
		return ErrDatosContactoParticipacionInvalidos
	}
	if d.Telefono2 != "" && (d.Telefono1 == "" || !telefonoContactoValido(d.Telefono2) || d.Telefono2 == d.Telefono1) {
		return ErrDatosContactoParticipacionInvalidos
	}
	return nil
}

// Canonico devuelve los bytes que se cifran. La referencia de participación no
// va dentro: forma parte de los datos asociados del sobre.
func (d DatosContactoParticipacion) Canonico() ([]byte, error) {
	if err := d.Validar(); err != nil {
		return nil, err
	}
	return json.Marshal(canonicoDatosContacto{Esquema: EsquemaDatosContactoParticipacion, Correo: d.Correo, Telefono1: d.Telefono1, Telefono2: d.Telefono2})
}

// DatosContactoParticipacionDesdeCanonico reconstruye el agregado tras descifrar.
func DatosContactoParticipacionDesdeCanonico(participacionRef string, claro []byte) (DatosContactoParticipacion, error) {
	var c canonicoDatosContacto
	if err := json.Unmarshal(claro, &c); err != nil || c.Esquema != EsquemaDatosContactoParticipacion {
		return DatosContactoParticipacion{}, ErrDatosContactoParticipacionInvalidos
	}
	d := DatosContactoParticipacion{ParticipacionRef: participacionRef, Correo: c.Correo, Telefono1: c.Telefono1, Telefono2: c.Telefono2}
	if err := d.Validar(); err != nil {
		return DatosContactoParticipacion{}, err
	}
	return d, nil
}

// Igual compara dos agregados ya normalizados (replay idempotente).
func (d DatosContactoParticipacion) Igual(otro DatosContactoParticipacion) bool {
	return d.ParticipacionRef == otro.ParticipacionRef && d.Correo == otro.Correo && d.Telefono1 == otro.Telefono1 && d.Telefono2 == otro.Telefono2
}

// DatosContactoEnmascarados es la forma que ven las lecturas sin permiso de
// contacto: conserva lo justo para reconocer el dato («a***@dominio.es»,
// «***1234»).
type DatosContactoEnmascarados struct {
	Correo    string `json:"correo"`
	Telefono1 string `json:"telefono_1"`
	Telefono2 string `json:"telefono_2"`
}

func (d DatosContactoParticipacion) Enmascarados() DatosContactoEnmascarados {
	return DatosContactoEnmascarados{Correo: enmascararCorreoContacto(d.Correo), Telefono1: enmascararTelefonoContacto(d.Telefono1), Telefono2: enmascararTelefonoContacto(d.Telefono2)}
}

func (DatosContactoParticipacion) String() string {
	return "bolsa.DatosContactoParticipacion{redactado}"
}
func (DatosContactoParticipacion) GoString() string {
	return "bolsa.DatosContactoParticipacion{redactado}"
}

func normalizarCorreoContacto(valor string) string {
	valor = strings.TrimSpace(valor)
	arroba := strings.LastIndex(valor, "@")
	if arroba <= 0 {
		return valor
	}
	return valor[:arroba] + "@" + strings.ToLower(valor[arroba+1:])
}

func correoContactoValido(valor string) bool {
	if valor == "" || len(valor) > longitudMaximaCorreoContacto || strings.TrimSpace(valor) != valor || strings.ContainsAny(valor, " <>\"\t\r\n") {
		return false
	}
	direccion, err := mail.ParseAddress(valor)
	if err != nil || direccion.Address != valor {
		return false
	}
	dominio := valor[strings.LastIndex(valor, "@")+1:]
	return strings.Contains(dominio, ".") && !strings.HasPrefix(dominio, ".") && !strings.HasSuffix(dominio, ".")
}

func normalizarTelefonoContacto(valor string) string {
	valor = strings.TrimSpace(valor)
	var b strings.Builder
	for i, r := range valor {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
		default:
			return valor
		}
	}
	return b.String()
}

// telefonoContactoValido admite números españoles de nueve cifras (móvil o
// fijo) y números internacionales con prefijo «+» de 9 a 15 cifras.
func telefonoContactoValido(valor string) bool {
	if valor == "" || len(valor) > longitudMaximaTelefonoContacto {
		return false
	}
	digitos := valor
	if strings.HasPrefix(valor, "+") {
		digitos = valor[1:]
		if len(digitos) < 9 || len(digitos) > 15 {
			return false
		}
	} else if len(digitos) != 9 || !strings.ContainsAny(digitos[:1], "6789") {
		return false
	}
	for _, r := range digitos {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func enmascararCorreoContacto(valor string) string {
	if valor == "" {
		return ""
	}
	arroba := strings.LastIndex(valor, "@")
	if arroba <= 0 {
		return "***"
	}
	return valor[:1] + "***@" + valor[arroba+1:]
}

func enmascararTelefonoContacto(valor string) string {
	if valor == "" {
		return ""
	}
	if len(valor) <= 4 {
		return "***"
	}
	return "***" + valor[len(valor)-4:]
}
