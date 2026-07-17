package calculoexperienciaoficial

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
)

// ClaveEfectoV1 identifica un único efecto oficial por sus entradas
// semánticas. Excluye actor, sesión, autorizaciones, correlaciones, tiempos,
// auditoría y cualquier dato personal directo.
type ClaveEfectoV1 struct {
	datos DatosClaveEfectoV1
}

func NuevaClaveEfectoV1(datos DatosClaveEfectoV1) (ClaveEfectoV1, error) {
	if err := validarDatosClave(datos); err != nil {
		return ClaveEfectoV1{}, err
	}
	return ClaveEfectoV1{datos: clonarDatosClave(datos)}, nil
}

func (c ClaveEfectoV1) Validar() error { return validarDatosClave(c.datos) }

func (c ClaveEfectoV1) Datos() (DatosClaveEfectoV1, error) {
	if err := c.Validar(); err != nil {
		return DatosClaveEfectoV1{}, err
	}
	return clonarDatosClave(c.datos), nil
}

func (c ClaveEfectoV1) SujetoPseudonimizado() ReferenciaExactaV1 {
	return c.datos.SujetoPseudonimizado
}
func (c ClaveEfectoV1) Convocatoria() ReferenciaExactaV1 { return c.datos.Convocatoria }
func (c ClaveEfectoV1) Reglas() VinculoReglasV1          { return c.datos.Reglas }
func (c ClaveEfectoV1) Entrada() VinculoEntradaV1        { return c.datos.Entrada }
func (c ClaveEfectoV1) Motor() VinculoMotorV1            { return c.datos.Motor }
func (c ClaveEfectoV1) HuellaPlanSHA256() string         { return c.datos.HuellaPlanSHA256 }
func (c ClaveEfectoV1) Causa() CausaGobernadaV1          { return c.datos.Causa }
func (c ClaveEfectoV1) Tipo() TipoEfectoV1               { return c.datos.Tipo }

func (c ClaveEfectoV1) Predecesor() (VinculoPredecesorV1, bool) {
	if c.datos.Predecesor == nil {
		return VinculoPredecesorV1{}, false
	}
	return *c.datos.Predecesor, true
}

func (c ClaveEfectoV1) RepresentacionCanonica() ([]byte, error) {
	if err := c.Validar(); err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(materializarClave(c))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesRepresentacionV1 {
		return nil, nuevoError("clave.representacion_canonica", CodigoFueraDeLimites)
	}
	return contenido, nil
}

func (c ClaveEfectoV1) MarshalJSON() ([]byte, error) { return c.RepresentacionCanonica() }

// UnmarshalJSON impide crear un valor cero aparente saltándose la
// restauración estricta, que es la única frontera de entrada admitida.
func (*ClaveEfectoV1) UnmarshalJSON([]byte) error { return ErrEntradaNoPermitida }

func (c ClaveEfectoV1) HuellaSHA256() (string, error) {
	contenido, err := c.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	return sha256Hex(contenido), nil
}

func RestaurarClaveEfectoV1(contenido []byte) (ClaveEfectoV1, error) {
	var material materialClaveEfectoV1
	if err := decodificarJSONEstricto(contenido, &material); err != nil {
		return ClaveEfectoV1{}, err
	}
	if material.Esquema != esquemaClaveEfectoV1 {
		return ClaveEfectoV1{}, nuevoError("clave.esquema", CodigoEsquemaIncompatible)
	}
	clave, err := NuevaClaveEfectoV1(datosDesdeMaterialClave(material))
	if err != nil {
		return ClaveEfectoV1{}, err
	}
	canonico, err := clave.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ClaveEfectoV1{}, nuevoError("clave.representacion_canonica", CodigoValorNoCanonico)
	}
	return clave, nil
}

func RestaurarClaveEfectoV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ClaveEfectoV1, error) {
	if !huellaSHA256Valida(huellaEsperada) {
		return ClaveEfectoV1{}, nuevoError("clave.huella_esperada_sha256", CodigoValorNoCanonico)
	}
	clave, err := RestaurarClaveEfectoV1(contenido)
	if err != nil {
		return ClaveEfectoV1{}, err
	}
	huella, err := clave.HuellaSHA256()
	if err != nil || subtle.ConstantTimeCompare([]byte(huella), []byte(huellaEsperada)) != 1 {
		return ClaveEfectoV1{}, nuevoError("clave.huella_esperada_sha256", CodigoHuellaNoCoincide)
	}
	return clave, nil
}

// CalcularIndiceHMACSHA256 deriva el índice durable sin exponer el pseudónimo
// ni permitir que una clave elegida por el cliente controle la idempotencia.
func CalcularIndiceHMACSHA256(clave ClaveEfectoV1, secretoServidor []byte) (string, error) {
	if len(secretoServidor) < minimoBytesSecretoHMACV1 ||
		len(secretoServidor) > maximoBytesSecretoHMACV1 {
		return "", nuevoError("indice_hmac.secreto_servidor", CodigoSecretoInvalido)
	}
	contenido, err := clave.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secretoServidor)
	_, _ = mac.Write([]byte(dominioIndiceHMACV1))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write(contenido)
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func materializarClave(clave ClaveEfectoV1) materialClaveEfectoV1 {
	datos := clave.datos
	return materialClaveEfectoV1{
		Esquema: esquemaClaveEfectoV1, SujetoPseudonimizado: datos.SujetoPseudonimizado,
		Convocatoria: datos.Convocatoria, Reglas: datos.Reglas, Entrada: datos.Entrada,
		Motor: datos.Motor, HuellaPlanSHA256: datos.HuellaPlanSHA256, Causa: datos.Causa,
		Tipo: datos.Tipo, Predecesor: clonarPredecesor(datos.Predecesor),
	}
}

func datosDesdeMaterialClave(material materialClaveEfectoV1) DatosClaveEfectoV1 {
	return DatosClaveEfectoV1{
		SujetoPseudonimizado: material.SujetoPseudonimizado,
		Convocatoria:         material.Convocatoria, Reglas: material.Reglas, Entrada: material.Entrada,
		Motor: material.Motor, HuellaPlanSHA256: material.HuellaPlanSHA256,
		Causa: material.Causa, Tipo: material.Tipo, Predecesor: clonarPredecesor(material.Predecesor),
	}
}

func clonarDatosClave(datos DatosClaveEfectoV1) DatosClaveEfectoV1 {
	datos.Predecesor = clonarPredecesor(datos.Predecesor)
	return datos
}

func clonarPredecesor(origen *VinculoPredecesorV1) *VinculoPredecesorV1 {
	if origen == nil {
		return nil
	}
	copia := *origen
	return &copia
}

func sha256Hex(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}
