package calculoexperienciaoficial

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
)

// IntencionResultadoV1 liga la clave semántica completa con el resultado
// exacto que se pretende confirmar. No porta autoridad ni contexto de sesión.
type IntencionResultadoV1 struct {
	clave                 ClaveEfectoV1
	huellaResultadoSHA256 string
	estado                EstadoResultadoV1
	fase                  FaseResultadoV1
}

func NuevaIntencionResultadoV1(
	clave ClaveEfectoV1,
	huellaResultadoSHA256 string,
	estado EstadoResultadoV1,
	fase FaseResultadoV1,
) (IntencionResultadoV1, error) {
	intencion := IntencionResultadoV1{
		clave: clonarClave(clave), huellaResultadoSHA256: huellaResultadoSHA256,
		estado: estado, fase: fase,
	}
	if err := intencion.Validar(); err != nil {
		return IntencionResultadoV1{}, err
	}
	return intencion, nil
}

func (i IntencionResultadoV1) Validar() error {
	if err := i.clave.Validar(); err != nil {
		return err
	}
	if !huellaSHA256Valida(i.huellaResultadoSHA256) {
		return nuevoError("intencion.huella_resultado_sha256", CodigoValorNoCanonico)
	}
	if !estadoYFaseValidos(i.estado, i.fase) {
		return nuevoError("intencion.estado_fase", CodigoEstadoIncompatible)
	}
	return nil
}

func (i IntencionResultadoV1) Clave() ClaveEfectoV1 { return clonarClave(i.clave) }
func (i IntencionResultadoV1) HuellaResultadoSHA256() string {
	return i.huellaResultadoSHA256
}
func (i IntencionResultadoV1) Estado() EstadoResultadoV1 { return i.estado }
func (i IntencionResultadoV1) Fase() FaseResultadoV1     { return i.fase }

func (i IntencionResultadoV1) RepresentacionCanonica() ([]byte, error) {
	if err := i.Validar(); err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(materializarIntencion(i))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesRepresentacionV1 {
		return nil, nuevoError("intencion.representacion_canonica", CodigoFueraDeLimites)
	}
	return contenido, nil
}

func (i IntencionResultadoV1) MarshalJSON() ([]byte, error) {
	return i.RepresentacionCanonica()
}

func (*IntencionResultadoV1) UnmarshalJSON([]byte) error { return ErrEntradaNoPermitida }

func (i IntencionResultadoV1) HuellaSHA256() (string, error) {
	contenido, err := i.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	return sha256Hex(contenido), nil
}

func (i IntencionResultadoV1) ValidarPara(
	clave ClaveEfectoV1,
	huellaResultadoSHA256 string,
	estado EstadoResultadoV1,
	fase FaseResultadoV1,
) error {
	esperada, err := NuevaIntencionResultadoV1(clave, huellaResultadoSHA256, estado, fase)
	if err != nil {
		return err
	}
	real, errReal := i.HuellaSHA256()
	objetivo, errObjetivo := esperada.HuellaSHA256()
	if errReal != nil || errObjetivo != nil ||
		subtle.ConstantTimeCompare([]byte(real), []byte(objetivo)) != 1 {
		return nuevoError("intencion", CodigoHuellaNoCoincide)
	}
	return nil
}

func RestaurarIntencionResultadoV1(contenido []byte) (IntencionResultadoV1, error) {
	var material materialIntencionResultadoV1
	if err := decodificarJSONEstricto(contenido, &material); err != nil {
		return IntencionResultadoV1{}, err
	}
	if material.Esquema != esquemaIntencionV1 {
		return IntencionResultadoV1{}, nuevoError("intencion.esquema", CodigoEsquemaIncompatible)
	}
	if material.Clave.Esquema != esquemaClaveEfectoV1 {
		return IntencionResultadoV1{}, nuevoError("intencion.clave.esquema", CodigoEsquemaIncompatible)
	}
	clave, err := NuevaClaveEfectoV1(datosDesdeMaterialClave(material.Clave))
	if err != nil {
		return IntencionResultadoV1{}, err
	}
	intencion, err := NuevaIntencionResultadoV1(
		clave, material.HuellaResultadoSHA256, material.Estado, material.Fase,
	)
	if err != nil {
		return IntencionResultadoV1{}, err
	}
	canonico, err := intencion.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return IntencionResultadoV1{},
			nuevoError("intencion.representacion_canonica", CodigoValorNoCanonico)
	}
	return intencion, nil
}

func RestaurarIntencionResultadoV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (IntencionResultadoV1, error) {
	if !huellaSHA256Valida(huellaEsperada) {
		return IntencionResultadoV1{},
			nuevoError("intencion.huella_esperada_sha256", CodigoValorNoCanonico)
	}
	intencion, err := RestaurarIntencionResultadoV1(contenido)
	if err != nil {
		return IntencionResultadoV1{}, err
	}
	huella, err := intencion.HuellaSHA256()
	if err != nil || subtle.ConstantTimeCompare([]byte(huella), []byte(huellaEsperada)) != 1 {
		return IntencionResultadoV1{},
			nuevoError("intencion.huella_esperada_sha256", CodigoHuellaNoCoincide)
	}
	return intencion, nil
}

func materializarIntencion(i IntencionResultadoV1) materialIntencionResultadoV1 {
	return materialIntencionResultadoV1{
		Esquema: esquemaIntencionV1, Clave: materializarClave(i.clave),
		HuellaResultadoSHA256: i.huellaResultadoSHA256, Estado: i.estado, Fase: i.fase,
	}
}

func clonarClave(clave ClaveEfectoV1) ClaveEfectoV1 {
	return ClaveEfectoV1{datos: clonarDatosClave(clave.datos)}
}
