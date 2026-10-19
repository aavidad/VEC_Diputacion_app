package ports

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrEstadoComprobacionOfrecimientoInvalido = errors.New("bolsa: estado de comprobacion de ofrecimiento invalido")

var referenciaComprobacionOfrecimiento = regexp.MustCompile(`^comprobacion:[0-9a-f]{64}$`)
var numeroVersionComprobacionOfrecimiento = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

// EstadoComprobacionOfrecimiento es la proyección mínima de B133 para CT.
// Un puntero nil en la entrada indica una lectura histórica sin esta proyección;
// version_esperada=0 indica que la lectura actual confirma ausencia de acto.
type EstadoComprobacionOfrecimiento struct {
	VersionEsperada        uint64    `json:"version_esperada"`
	ComprobacionRef        string    `json:"comprobacion_ref,omitempty"`
	ReciboRef              string    `json:"recibo_ref,omitempty"`
	RegistradaEn           time.Time `json:"registrada_en,omitempty"`
	VersionPoliticaOfertas uint64    `json:"version_politica_ofertas,omitempty"`
	Resultado              string    `json:"resultado,omitempty"`
	MotivoClave            string    `json:"motivo_clave,omitempty"`
	Vigente                *bool     `json:"vigente,omitempty"`
}

func (e EstadoComprobacionOfrecimiento) Validar() error {
	if e.VersionEsperada == 0 {
		if e.ComprobacionRef == "" && e.ReciboRef == "" && e.RegistradaEn.IsZero() &&
			e.VersionPoliticaOfertas == 0 && e.Resultado == "" && e.MotivoClave == "" && e.Vigente == nil {
			return nil
		}
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	motivoValido := e.Resultado == "no_aplicable" && e.MotivoClave == "sin_supuesto_articulo_9" ||
		e.Resultado == "pendiente" && (e.MotivoClave == "antecedentes_incompletos" ||
			e.MotivoClave == "criterios_invalidos" || e.MotivoClave == "guarda_contratacion_pendiente" ||
			e.MotivoClave == "descuento_pendiente_criterio_rrhh")
	_, desfase := e.RegistradaEn.Zone()
	if e.VersionEsperada > 999999999 || !referenciaComprobacionOfrecimiento.MatchString(e.ComprobacionRef) ||
		e.ReciboRef != "recibo:"+e.ComprobacionRef || e.VersionPoliticaOfertas == 0 ||
		e.VersionPoliticaOfertas > 9007199254740991 ||
		!motivoValido || e.Vigente == nil ||
		e.RegistradaEn.IsZero() || desfase != 0 ||
		e.RegistradaEn.Year() < 1 || e.RegistradaEn.Year() > 9999 || e.RegistradaEn.Nanosecond()%1000 != 0 {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	return nil
}

func (e EstadoComprobacionOfrecimiento) MarshalJSON() ([]byte, error) {
	if e.Validar() != nil {
		return nil, ErrEstadoComprobacionOfrecimientoInvalido
	}
	if e.VersionEsperada == 0 {
		return []byte(`{"version_esperada":0}`), nil
	}
	type sinMetodo EstadoComprobacionOfrecimiento
	return json.Marshal(sinMetodo(e))
}

func (e *EstadoComprobacionOfrecimiento) UnmarshalJSON(raw []byte) error {
	if e == nil || len(raw) > 2048 {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	campos, err := camposObjetoComprobacionOfrecimiento(raw)
	if err != nil {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	version, ok := campos["version_esperada"]
	if !ok || !numeroVersionComprobacionOfrecimiento.Match(version) {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	n, err := strconv.ParseUint(string(version), 10, 64)
	if err != nil || n > 999999999 || (n == 0 && len(campos) != 1) || (n > 0 && len(campos) != 8) {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	type sinMetodo EstadoComprobacionOfrecimiento
	var decodificado sinMetodo
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&decodificado) != nil || !errors.Is(d.Decode(new(any)), io.EOF) ||
		EstadoComprobacionOfrecimiento(decodificado).Validar() != nil {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	for _, clave := range []string{"comprobacion_ref", "recibo_ref", "registrada_en", "version_politica_ofertas", "resultado", "motivo_clave", "vigente"} {
		if n > 0 && (len(campos[clave]) == 0 || bytes.Equal(campos[clave], []byte("null"))) {
			return ErrEstadoComprobacionOfrecimientoInvalido
		}
	}
	*e = EstadoComprobacionOfrecimiento(decodificado)
	return nil
}

// La entrada histórica no traía comprobacion_ofrecimiento. Un null explícito
// tampoco acredita ausencia: solo el objeto {"version_esperada":0} lo hace.
func (e *EntradaOrigenLlamamientoCT) UnmarshalJSON(raw []byte) error {
	campos, err := camposObjetoComprobacionOfrecimiento(raw)
	if err != nil {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	for clave := range campos {
		if strings.EqualFold(clave, "comprobacion_ofrecimiento") && clave != "comprobacion_ofrecimiento" {
			return ErrEstadoComprobacionOfrecimientoInvalido
		}
	}
	if v, existe := campos["comprobacion_ofrecimiento"]; existe && bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	type sinMetodo EntradaOrigenLlamamientoCT
	var decodificada sinMetodo
	if json.Unmarshal(raw, &decodificada) != nil {
		return ErrEstadoComprobacionOfrecimientoInvalido
	}
	*e = EntradaOrigenLlamamientoCT(decodificada)
	return nil
}

func camposObjetoComprobacionOfrecimiento(raw []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	inicio, err := d.Token()
	if err != nil || inicio != json.Delim('{') {
		return nil, ErrEstadoComprobacionOfrecimientoInvalido
	}
	campos := make(map[string]json.RawMessage)
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return nil, ErrEstadoComprobacionOfrecimientoInvalido
		}
		clave, ok := token.(string)
		if !ok {
			return nil, ErrEstadoComprobacionOfrecimientoInvalido
		}
		if _, repetido := campos[clave]; repetido {
			return nil, ErrEstadoComprobacionOfrecimientoInvalido
		}
		var valor json.RawMessage
		if d.Decode(&valor) != nil {
			return nil, ErrEstadoComprobacionOfrecimientoInvalido
		}
		campos[clave] = valor
	}
	if _, err := d.Token(); err != nil || !errors.Is(d.Decode(new(any)), io.EOF) {
		return nil, ErrEstadoComprobacionOfrecimientoInvalido
	}
	return campos, nil
}
