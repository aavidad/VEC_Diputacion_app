package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var errFechaCivilExpedienteSQL = errors.New("fecha civil de expediente no válida")

// rehidratarFechasExpedienteSQL adapta únicamente la representación civil que
// el alta v3 guarda en los dos periodos de la solicitud. El contenido original
// se conserva para cualquier comprobación de huella o postimagen histórica.
func rehidratarFechasExpedienteSQL(contenido []byte, ruta ...string) ([]byte, error) {
	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &raiz); err != nil || raiz == nil {
		return nil, errFechaCivilExpedienteSQL
	}
	resultado, cambiado, err := rehidratarObjetoExpedienteSQL(raiz, ruta)
	if err != nil {
		return nil, err
	}
	if !cambiado {
		return contenido, nil
	}
	// Al volver a codificar una copia, dos claves iguales colapsarían en un
	// mapa. Rechazarlas antes impide que desaparezca un campo que el decoder
	// estricto habría detectado en los bytes originales.
	if validarJSONSinDuplicadosDecisionCoberturaO404E(contenido, 128) != nil {
		return nil, errFechaCivilExpedienteSQL
	}
	return json.Marshal(resultado)
}

func rehidratarObjetoExpedienteSQL(objeto map[string]json.RawMessage, ruta []string) (map[string]json.RawMessage, bool, error) {
	if len(ruta) != 0 {
		clave := ruta[0]
		var hijo map[string]json.RawMessage
		if json.Unmarshal(objeto[clave], &hijo) != nil || hijo == nil {
			return nil, false, errFechaCivilExpedienteSQL
		}
		cambio, cambiado, err := rehidratarObjetoExpedienteSQL(hijo, ruta[1:])
		if err != nil || !cambiado {
			return objeto, cambiado, err
		}
		objeto[clave], err = json.Marshal(cambio)
		return objeto, err == nil, err
	}
	var solicitud map[string]json.RawMessage
	if json.Unmarshal(objeto["solicitud"], &solicitud) != nil || solicitud == nil {
		return nil, false, errFechaCivilExpedienteSQL
	}
	cambiado, err := rehidratarPeriodoExpedienteSQL(solicitud)
	if err != nil {
		return nil, false, err
	}
	if necesidadRaw, existe := solicitud["necesidad"]; existe && string(necesidadRaw) != "null" {
		var necesidad map[string]json.RawMessage
		if json.Unmarshal(necesidadRaw, &necesidad) != nil || necesidad == nil {
			return nil, false, errFechaCivilExpedienteSQL
		}
		cambioNecesidad, err := rehidratarPeriodoExpedienteSQL(necesidad)
		if err != nil {
			return nil, false, err
		}
		if cambioNecesidad {
			solicitud["necesidad"], err = json.Marshal(necesidad)
			if err != nil {
				return nil, false, err
			}
			cambiado = true
		}
	}
	cambioRC, err := rehidratarRCAusenteExpedienteSQL(solicitud)
	if err != nil {
		return nil, false, err
	}
	cambiado = cambiado || cambioRC
	if !cambiado {
		return objeto, false, nil
	}
	objeto["solicitud"], err = json.Marshal(solicitud)
	return objeto, err == nil, err
}

// El canon de alta escribe un RC ausente como fecha vacía e importe 0 EUR.
// En el dominio, la ausencia es Fecha e Importe cero. Sólo se adapta esa
// representación exacta; los datos de un RC existente permanecen intactos.
func rehidratarRCAusenteExpedienteSQL(solicitud map[string]json.RawMessage) (bool, error) {
	var rc map[string]json.RawMessage
	if json.Unmarshal(solicitud["rc"], &rc) != nil || rc == nil {
		return false, errFechaCivilExpedienteSQL
	}
	if !bytes.Equal(bytes.TrimSpace(rc["existe"]), []byte("false")) {
		return false, nil
	}
	cambiado := false
	if string(rc["fecha"]) == `""` {
		delete(rc, "fecha")
		cambiado = true
	}
	if bruto, tiene := rc["importe"]; tiene {
		var campos map[string]json.RawMessage
		if json.Unmarshal(bruto, &campos) == nil && len(campos) == 2 &&
			bytes.Equal(bytes.TrimSpace(campos["centimos"]), []byte("0")) &&
			bytes.Equal(bytes.TrimSpace(campos["moneda"]), []byte(`"EUR"`)) {
			delete(rc, "importe")
			cambiado = true
		}
	}
	if cambiado {
		var err error
		solicitud["rc"], err = json.Marshal(rc)
		if err != nil {
			return false, err
		}
	}
	return cambiado, nil
}

func rehidratarPeriodoExpedienteSQL(objeto map[string]json.RawMessage) (bool, error) {
	var periodo map[string]json.RawMessage
	if json.Unmarshal(objeto["periodo"], &periodo) != nil || periodo == nil {
		return false, errFechaCivilExpedienteSQL
	}
	cambiado := false
	for _, campo := range [...]string{"inicio", "fin"} {
		crudo, existe := periodo[campo]
		if !existe || string(crudo) == "null" {
			continue
		}
		var fecha string
		if json.Unmarshal(crudo, &fecha) != nil {
			return false, errFechaCivilExpedienteSQL
		}
		if len(fecha) != len("2006-01-02") {
			continue // RFC3339 histórico; el decodificador estricto lo valida.
		}
		civil, err := time.Parse("2006-01-02", fecha)
		if err != nil || civil.Format("2006-01-02") != fecha {
			return false, errFechaCivilExpedienteSQL
		}
		periodo[campo], err = json.Marshal(civil.UTC().Format(time.RFC3339))
		if err != nil {
			return false, err
		}
		cambiado = true
	}
	if cambiado {
		var err error
		objeto["periodo"], err = json.Marshal(periodo)
		if err != nil {
			return false, err
		}
	}
	return cambiado, nil
}

func decodificarExpedienteSQL(contenido []byte, destino *domain.Expediente) error {
	adaptado, err := rehidratarFechasExpedienteSQL(contenido)
	if err != nil {
		return err
	}
	return decodificarExpedienteEstrictoSQL(adaptado, destino)
}

// La respuesta SQL puede no incluir expediente (conflicto o recibo ya
// confirmado). Si lo incluye, se adapta esa proyección antes del decoder
// estricto sin alterar el cuerpo recibido ni otros campos de la respuesta.
func decodificarConExpedienteSQL(contenido []byte, destino any, campos ...string) error {
	var raiz map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &raiz); err != nil || raiz == nil {
		return errFechaCivilExpedienteSQL
	}
	adaptado := contenido
	for _, campo := range campos {
		crudo, existe := raiz[campo]
		if !existe || string(crudo) == "null" {
			continue
		}
		var err error
		adaptado, err = rehidratarFechasExpedienteSQL(adaptado, campo)
		if err != nil {
			return err
		}
	}
	return decodificarExpedienteEstrictoSQL(adaptado, destino)
}

func decodificarExpedienteEstrictoSQL(contenido []byte, destino any) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.UseNumber()
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if err := decodificador.Decode(&struct{}{}); err != io.EOF {
		return errFechaCivilExpedienteSQL
	}
	return nil
}
