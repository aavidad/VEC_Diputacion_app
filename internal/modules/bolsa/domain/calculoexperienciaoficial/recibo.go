package calculoexperienciaoficial

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
)

// ReciboV1 es el comprobante mínimo e inmutable del efecto confirmado. No
// duplica la intención ni incorpora actor, autoridad, auditoría o tiempos.
type ReciboV1 struct {
	referencia              string
	generacionClaveHMAC     uint32
	indiceEfectoHMACSHA256  string
	huellaClaveEfectoSHA256 string
	huellaIntencionSHA256   string
	huellaResultadoSHA256   string
	estado                  EstadoResultadoV1
	fase                    FaseResultadoV1
}

func NuevoReciboV1(
	referencia string,
	generacionClaveHMAC uint32,
	indiceEfectoHMACSHA256 string,
	intencion IntencionResultadoV1,
) (ReciboV1, error) {
	if err := intencion.Validar(); err != nil {
		return ReciboV1{}, err
	}
	huellaClave, errClave := intencion.clave.HuellaSHA256()
	huellaIntencion, errIntencion := intencion.HuellaSHA256()
	if errClave != nil || errIntencion != nil {
		return ReciboV1{}, nuevoError("recibo.intencion", CodigoValorInvalido)
	}
	recibo := ReciboV1{
		referencia: referencia, generacionClaveHMAC: generacionClaveHMAC,
		indiceEfectoHMACSHA256:  indiceEfectoHMACSHA256,
		huellaClaveEfectoSHA256: huellaClave, huellaIntencionSHA256: huellaIntencion,
		huellaResultadoSHA256: intencion.huellaResultadoSHA256,
		estado:                intencion.estado, fase: intencion.fase,
	}
	if err := recibo.Validar(); err != nil {
		return ReciboV1{}, err
	}
	return recibo, nil
}

func (r ReciboV1) Validar() error {
	if !referenciaOpacaValida(r.referencia) || r.generacionClaveHMAC == 0 ||
		!huellaSHA256Valida(r.indiceEfectoHMACSHA256) ||
		!huellaSHA256Valida(r.huellaClaveEfectoSHA256) ||
		!huellaSHA256Valida(r.huellaIntencionSHA256) ||
		!huellaSHA256Valida(r.huellaResultadoSHA256) {
		return nuevoError("recibo", CodigoValorNoCanonico)
	}
	if !estadoYFaseValidos(r.estado, r.fase) {
		return nuevoError("recibo.estado_fase", CodigoEstadoIncompatible)
	}
	return nil
}

func (r ReciboV1) Referencia() string              { return r.referencia }
func (r ReciboV1) GeneracionClaveHMAC() uint32     { return r.generacionClaveHMAC }
func (r ReciboV1) IndiceHMACSHA256() string        { return r.indiceEfectoHMACSHA256 }
func (r ReciboV1) HuellaClaveEfectoSHA256() string { return r.huellaClaveEfectoSHA256 }
func (r ReciboV1) HuellaIntencionSHA256() string   { return r.huellaIntencionSHA256 }
func (r ReciboV1) HuellaResultadoSHA256() string   { return r.huellaResultadoSHA256 }
func (r ReciboV1) Estado() EstadoResultadoV1       { return r.estado }
func (r ReciboV1) Fase() FaseResultadoV1           { return r.fase }

func (r ReciboV1) ValidarPara(
	indiceEfectoHMACSHA256 string,
	intencion IntencionResultadoV1,
) error {
	if err := r.Validar(); err != nil {
		return err
	}
	if err := intencion.Validar(); err != nil {
		return err
	}
	huellaClave, errClave := intencion.clave.HuellaSHA256()
	huellaIntencion, errIntencion := intencion.HuellaSHA256()
	if errClave != nil || errIntencion != nil ||
		!textoConstanteIgual(r.indiceEfectoHMACSHA256, indiceEfectoHMACSHA256) ||
		!textoConstanteIgual(r.huellaClaveEfectoSHA256, huellaClave) ||
		!textoConstanteIgual(r.huellaIntencionSHA256, huellaIntencion) ||
		!textoConstanteIgual(r.huellaResultadoSHA256, intencion.huellaResultadoSHA256) ||
		r.estado != intencion.estado || r.fase != intencion.fase {
		return nuevoError("recibo.intencion", CodigoHuellaNoCoincide)
	}
	return nil
}

func (r ReciboV1) RepresentacionCanonica() ([]byte, error) {
	if err := r.Validar(); err != nil {
		return nil, err
	}
	contenido, err := json.Marshal(materializarRecibo(r))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoBytesRepresentacionV1 {
		return nil, nuevoError("recibo.representacion_canonica", CodigoFueraDeLimites)
	}
	return contenido, nil
}

func (r ReciboV1) MarshalJSON() ([]byte, error) { return r.RepresentacionCanonica() }

func (*ReciboV1) UnmarshalJSON([]byte) error { return ErrEntradaNoPermitida }

func (r ReciboV1) HuellaSHA256() (string, error) {
	contenido, err := r.RepresentacionCanonica()
	if err != nil {
		return "", err
	}
	return sha256Hex(contenido), nil
}

func (r ReciboV1) VinculoPredecesor() (VinculoPredecesorV1, error) {
	huella, err := r.HuellaSHA256()
	if err != nil {
		return VinculoPredecesorV1{}, err
	}
	return VinculoPredecesorV1{
		ReferenciaRecibo: r.referencia, HuellaReciboSHA256: huella,
	}, nil
}

func RestaurarReciboV1(contenido []byte) (ReciboV1, error) {
	var material materialReciboV1
	if err := decodificarJSONEstricto(contenido, &material); err != nil {
		return ReciboV1{}, err
	}
	if material.Esquema != esquemaReciboV1 {
		return ReciboV1{}, nuevoError("recibo.esquema", CodigoEsquemaIncompatible)
	}
	recibo := reciboDesdeMaterial(material)
	if err := recibo.Validar(); err != nil {
		return ReciboV1{}, err
	}
	canonico, err := recibo.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ReciboV1{}, nuevoError("recibo.representacion_canonica", CodigoValorNoCanonico)
	}
	return recibo, nil
}

func RestaurarReciboV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ReciboV1, error) {
	if !huellaSHA256Valida(huellaEsperada) {
		return ReciboV1{}, nuevoError("recibo.huella_esperada_sha256", CodigoValorNoCanonico)
	}
	recibo, err := RestaurarReciboV1(contenido)
	if err != nil {
		return ReciboV1{}, err
	}
	huella, err := recibo.HuellaSHA256()
	if err != nil || !textoConstanteIgual(huella, huellaEsperada) {
		return ReciboV1{}, nuevoError("recibo.huella_esperada_sha256", CodigoHuellaNoCoincide)
	}
	return recibo, nil
}

func materializarRecibo(r ReciboV1) materialReciboV1 {
	return materialReciboV1{
		Esquema: esquemaReciboV1, Referencia: r.referencia,
		GeneracionClaveHMAC:     r.generacionClaveHMAC,
		IndiceEfectoHMACSHA256:  r.indiceEfectoHMACSHA256,
		HuellaClaveEfectoSHA256: r.huellaClaveEfectoSHA256,
		HuellaIntencionSHA256:   r.huellaIntencionSHA256,
		HuellaResultadoSHA256:   r.huellaResultadoSHA256, Estado: r.estado, Fase: r.fase,
	}
}

func reciboDesdeMaterial(m materialReciboV1) ReciboV1 {
	return ReciboV1{
		referencia: m.Referencia, generacionClaveHMAC: m.GeneracionClaveHMAC,
		indiceEfectoHMACSHA256:  m.IndiceEfectoHMACSHA256,
		huellaClaveEfectoSHA256: m.HuellaClaveEfectoSHA256,
		huellaIntencionSHA256:   m.HuellaIntencionSHA256,
		huellaResultadoSHA256:   m.HuellaResultadoSHA256, estado: m.Estado, fase: m.Fase,
	}
}

func textoConstanteIgual(izquierda, derecha string) bool {
	return subtle.ConstantTimeCompare([]byte(izquierda), []byte(derecha)) == 1
}
