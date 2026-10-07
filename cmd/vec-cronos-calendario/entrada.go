package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	calapp "vec-diputacion-granada/internal/modules/calendarios/application"
	cal "vec-diputacion-granada/internal/modules/calendarios/domain"
	calports "vec-diputacion-granada/internal/modules/calendarios/ports"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
)

const maximoEntrada = 2 << 20

var errEntrada = errors.New("entrada_invalida")

type versionSeleccionada struct {
	Calendario cal.VersionConDias                 `json:"calendario"`
	Fuente     cronosports.FuenteEnsayoCalendario `json:"fuente"`
}

type entrada struct {
	Demostracion  bool                                  `json:"demostracion"`
	Idioma        string                                `json:"idioma"`
	PersonaNombre string                                `json:"persona_nombre"`
	Solicitud     cronosports.SolicitudEnsayoCalendario `json:"solicitud"`
	Calendarios   struct {
		ConocidoEn time.Time             `json:"conocido_en"`
		Versiones  []versionSeleccionada `json:"versiones"`
	} `json:"calendarios"`
}

func leerEntrada(r io.Reader) (entrada, []byte, error) {
	var e entrada
	b, err := io.ReadAll(io.LimitReader(r, maximoEntrada+1))
	if err != nil || len(b) > maximoEntrada {
		return e, nil, errEntrada
	}
	// encoding/json acepta claves duplicadas; el ensayo las rechaza para que
	// ninguna versión o referencia pueda sustituirse al decodificar.
	d := json.NewDecoder(bytes.NewReader(b))
	if comprobarJSON(d, 0) != nil {
		return e, nil, errEntrada
	}
	if _, err = d.Token(); err != io.EOF {
		return e, nil, errEntrada
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(&e); err != nil {
		return e, nil, errEntrada
	}
	if !e.Demostracion || e.PersonaNombre == "" || len(e.PersonaNombre) > 160 || !e.Calendarios.ConocidoEn.Equal(e.Solicitud.ConocidoEn) || len(e.Calendarios.Versiones) > 128 {
		return e, nil, errEntrada
	}
	return e, b, nil
}

func comprobarJSON(d *json.Decoder, profundidad int) error {
	if profundidad > 32 {
		return errEntrada
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		vistos := map[string]bool{}
		for d.More() {
			t, err = d.Token()
			if err != nil {
				return err
			}
			k, ok := t.(string)
			if !ok || !claveCanonicaJSON(k) || vistos[k] {
				return errEntrada
			}
			vistos[k] = true
			if err = comprobarJSON(d, profundidad+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err = comprobarJSON(d, profundidad+1); err != nil {
				return err
			}
		}
	default:
		return errEntrada
	}
	_, err = d.Token()
	return err
}

// encoding/json también reconoce alias con mayúsculas y ciertos caracteres
// Unicode. Solo admitimos la grafía ASCII de las claves declaradas en los DTO.
func claveCanonicaJSON(k string) bool {
	if k == "" {
		return false
	}
	for _, c := range k {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// El repositorio de ensayo contiene versiones ya seleccionadas por el fixture
// para un instante exacto. No decide cuál es la última versión histórica.
type repositorioEnsayo struct {
	conocido  time.Time
	versiones []cal.VersionConDias
}

func (r *repositorioEnsayo) VersionesVigentes(ctx context.Context, q calports.ConsultaVersiones) ([]cal.VersionConDias, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !q.ConocidoEn.Equal(r.conocido) {
		return nil, errEntrada
	}
	resultado := []cal.VersionConDias{}
	for _, a := range q.Ambitos {
		for _, v := range r.versiones {
			if v.Version.Anio == q.Anio && v.Version.Ambito == a {
				resultado = append(resultado, v)
			}
		}
	}
	return resultado, nil
}

func (r *repositorioEnsayo) CentrosConCalendario(ctx context.Context, anio int, conocido time.Time) ([]cal.VersionCalendario, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !conocido.Equal(r.conocido) {
		return nil, errEntrada
	}
	resultado := []cal.VersionCalendario{}
	for _, v := range r.versiones {
		if v.Version.Anio == anio && v.Version.Ambito.Tipo == cal.AmbitoCentro {
			resultado = append(resultado, v.Version)
		}
	}
	return resultado, nil
}

type relojEnsayo struct{ conocido time.Time }

func (r relojEnsayo) Ahora() time.Time { return r.conocido }

func consultaEnsayo(e entrada) (*calapp.Servicio, error) {
	r := &repositorioEnsayo{conocido: e.Calendarios.ConocidoEn.UTC()}
	vistas := map[string]bool{}
	for _, v := range e.Calendarios.Versiones {
		clave := v.Calendario.Version.ID
		// Dos versiones del mismo ámbito/año tampoco se admiten: no elegimos
		// silenciosamente entre ellas, incluso si una no llega a consultarse.
		for _, previa := range r.versiones {
			if previa.Version.Ambito == v.Calendario.Version.Ambito && previa.Version.Anio == v.Calendario.Version.Anio {
				return nil, errEntrada
			}
		}
		if vistas[clave] || v.Calendario.Validar() != nil || !v.Calendario.Version.Procedencia.Sintetica || v.Calendario.Version.ConocidoDesde.After(r.conocido) || !fuenteValida(v.Fuente) {
			return nil, errEntrada
		}
		vistas[clave] = true
		r.versiones = append(r.versiones, v.Calendario)
	}
	return calapp.NuevoServicio(r, relojEnsayo{conocido: r.conocido})
}
