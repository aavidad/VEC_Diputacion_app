package httpinterno

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	app "vec-diputacion-granada/internal/modules/bolsa/application/gobiernoreglasbaremo"
	reglas "vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

type motivoGobiernoHTTPV3 struct {
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      uint64 `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
	EntradaClave         string `json:"entrada_clave"`
}

func (m motivoGobiernoHTTPV3) dominio() (reglas.MotivoCatalogadoReglasBaremo, vd.ReferenciaEntradaCatalogo, error) {
	var motivo reglas.MotivoCatalogadoReglasBaremo
	var referencia vd.ReferenciaEntradaCatalogo
	if m.CatalogoVersion == 0 || m.CatalogoVersion > 1<<31-1 {
		return motivo, referencia, app.ErrGobiernoV3PeticionInvalida
	}
	catalogo, err := reglas.NuevaReferenciaVersionada(m.CatalogoID, m.CatalogoVersion, m.CatalogoHuellaSHA256)
	if err != nil {
		return motivo, referencia, app.ErrGobiernoV3PeticionInvalida
	}
	motivo, err = reglas.NuevoMotivoCatalogadoReglasBaremo(catalogo, m.EntradaClave)
	referencia = vd.ReferenciaEntradaCatalogo{CatalogoID: m.CatalogoID, CatalogoVersion: int(m.CatalogoVersion), CatalogoHuellaSHA256: m.CatalogoHuellaSHA256, EntradaClave: m.EntradaClave}
	if err != nil || !vd.ReferenciaMotivoAutorizacionV2Valida(referencia) {
		return motivo, referencia, app.ErrGobiernoV3PeticionInvalida
	}
	// La forma no acredita la referencia: el PDP consulta su validador de
	// motivos actual antes de emitir material atestado para la operación.
	return motivo, referencia, nil
}

type selectorGobiernoHTTPV3 struct {
	Referencia            string `json:"referencia"`
	Version               uint64 `json:"version"`
	ConvocatoriaRef       string `json:"convocatoria_ref"`
	ExpedienteRef         string `json:"expediente_ref"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
	Revision              uint64 `json:"revision"`
	HuellaEstadoSHA256    string `json:"huella_estado_sha256"`
}

func (s selectorGobiernoHTTPV3) dominio() (ports.SelectorGobiernoReglasV3, error) {
	var vacio ports.SelectorGobiernoReglasV3
	id, err := reglas.NuevaIdentidadConjuntoReglasBaremo(s.Referencia, s.Version, s.ConvocatoriaRef, s.ExpedienteRef)
	if err != nil {
		return vacio, app.ErrGobiernoV3PeticionInvalida
	}
	contenido, err := reglas.NuevaReferenciaVersionada(s.Referencia, s.Version, s.HuellaContenidoSHA256)
	if err != nil {
		return vacio, app.ErrGobiernoV3PeticionInvalida
	}
	estado, err := reglas.NuevoVinculoEstadoReglasBaremo(contenido, s.Revision, s.HuellaEstadoSHA256)
	if err != nil || s.Revision != 1 {
		return vacio, app.ErrGobiernoV3PeticionInvalida
	}
	return ports.SelectorGobiernoReglasV3{Identidad: id, Estado: estado}, nil
}

type entradaGobiernoHTTPV3 struct {
	Reglas                json.RawMessage
	Selector              selectorGobiernoHTTPV3
	Motivo                motivoGobiernoHTTPV3
	ClaveOperacion        string
	HuellaSolicitudSHA256 string
}

func leerEntradaGobiernoReglasV3(r io.Reader, ruta string) (entradaGobiernoHTTPV3, error) {
	contenido, err := io.ReadAll(r)
	if err != nil {
		return entradaGobiernoHTTPV3{}, errors.Join(app.ErrGobiernoV3PeticionInvalida, err)
	}
	if !utf8.Valid(contenido) || validarJSONSinDuplicados(contenido) != nil {
		return entradaGobiernoHTTPV3{}, app.ErrGobiernoV3PeticionInvalida
	}
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.DisallowUnknownFields()
	var entrada entradaGobiernoHTTPV3
	switch ruta {
	case RutaAltaGobiernoReglasBaremoV3:
		var e struct {
			Reglas         json.RawMessage      `json:"reglas"`
			Motivo         motivoGobiernoHTTPV3 `json:"motivo"`
			ClaveOperacion string               `json:"clave_operacion"`
		}
		err = d.Decode(&e)
		entrada.Reglas, entrada.Motivo, entrada.ClaveOperacion = e.Reglas, e.Motivo, e.ClaveOperacion
	case RutaConsultaGobiernoReglasBaremoV3:
		var e struct {
			Selector selectorGobiernoHTTPV3 `json:"selector"`
			Motivo   motivoGobiernoHTTPV3   `json:"motivo"`
		}
		err = d.Decode(&e)
		entrada.Selector, entrada.Motivo = e.Selector, e.Motivo
	case RutaRecuperarGobiernoReglasBaremoV3:
		var e struct {
			Selector              selectorGobiernoHTTPV3 `json:"selector"`
			Motivo                motivoGobiernoHTTPV3   `json:"motivo"`
			ClaveOperacion        string                 `json:"clave_operacion"`
			HuellaSolicitudSHA256 string                 `json:"huella_solicitud_sha256"`
		}
		err = d.Decode(&e)
		entrada.Selector, entrada.Motivo, entrada.ClaveOperacion, entrada.HuellaSolicitudSHA256 = e.Selector, e.Motivo, e.ClaveOperacion, e.HuellaSolicitudSHA256
	default:
		return entrada, app.ErrGobiernoV3PeticionInvalida
	}
	if err != nil {
		return entradaGobiernoHTTPV3{}, errors.Join(app.ErrGobiernoV3PeticionInvalida, err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return entradaGobiernoHTTPV3{}, errors.Join(app.ErrGobiernoV3PeticionInvalida, err)
	}
	return entrada, nil
}

func (e entradaGobiernoHTTPV3) alta() (app.PeticionAltaBorradorV3, error) {
	var vacio app.PeticionAltaBorradorV3
	canon, err := reglas.CanonicalizarConjuntoReglasBaremoJSON(e.Reglas)
	if err != nil {
		return vacio, app.ErrGobiernoV3PeticionInvalida
	}
	h := sha256.Sum256(canon)
	conjunto, err := reglas.RestaurarConjuntoReglasBaremoConHuellaSHA256(canon, hex.EncodeToString(h[:]))
	if err != nil {
		return vacio, app.ErrGobiernoV3PeticionInvalida
	}
	motivo, _, err := e.Motivo.dominio()
	if err != nil {
		return vacio, err
	}
	return app.PeticionAltaBorradorV3{Conjunto: conjunto, Motivo: motivo, ClaveOperacion: e.ClaveOperacion}, nil
}

func (e entradaGobiernoHTTPV3) consulta() (app.PeticionConsultaExactaV3, error) {
	var vacio app.PeticionConsultaExactaV3
	s, err := e.Selector.dominio()
	if err != nil {
		return vacio, err
	}
	_, motivo, err := e.Motivo.dominio()
	if err != nil {
		return vacio, err
	}
	return app.PeticionConsultaExactaV3{Selector: s, Motivo: motivo}, nil
}

func (e entradaGobiernoHTTPV3) recuperacion() (app.PeticionRecuperarReciboV3, error) {
	p, err := e.consulta()
	if err != nil {
		return app.PeticionRecuperarReciboV3{}, err
	}
	return app.PeticionRecuperarReciboV3{Selector: p.Selector, Motivo: p.Motivo, ClaveOperacion: e.ClaveOperacion, HuellaSolicitudSHA256: e.HuellaSolicitudSHA256}, nil
}
