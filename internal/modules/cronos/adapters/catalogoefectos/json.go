package catalogoefectos

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"regexp"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const LimiteJSON = 1 << 20

type Catalogo struct {
	VersionEsquema int
	Demostracion   bool
	PaqueteRef     string
	Politica       domain.PoliticaEfectosPermisoSaldo
	Textos         map[string]map[string]string
}

type catalogoJSON struct {
	VersionEsquema int                          `json:"version_esquema"`
	Demostracion   bool                         `json:"demostracion"`
	PaqueteRef     string                       `json:"paquete_ref"`
	Politica       politicaJSON                 `json:"politica"`
	Textos         map[string]map[string]string `json:"textos"`
}

type politicaJSON struct {
	Referencia string                          `json:"referencia"`
	Version    int64                           `json:"version"`
	Reglas     []ports.ReglaEfectoPermisoSaldo `json:"reglas"`
}

var claveJSON = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// DecodificarEstricto verifica los bytes exactos antes de decodificar. Rechaza
// claves duplicadas, mayúsculas, campos desconocidos, exceso de tamaño y cola.
func DecodificarEstricto(b []byte, esperado string, destino any) error {
	if len(b) == 0 || len(b) > LimiteJSON {
		return domain.ErrEfectoPermisoSaldoInvalido
	}
	h := sha256.Sum256(b)
	if !domain.HuellaEfectosValida(esperado) || hex.EncodeToString(h[:]) != esperado {
		return domain.ErrEfectoPermisoSaldoInvalido
	}
	d := json.NewDecoder(bytes.NewReader(b))
	tokens := 0
	var recorrer func(int) error
	recorrer = func(profundidad int) error {
		tokens++
		if profundidad > 16 || tokens > 30000 {
			return domain.ErrEfectoPermisoSaldoInvalido
		}
		t, err := d.Token()
		if err != nil {
			return domain.ErrEfectoPermisoSaldoInvalido
		}
		delim, ok := t.(json.Delim)
		if profundidad == 0 && (!ok || delim != '{') {
			return domain.ErrEfectoPermisoSaldoInvalido
		}
		if !ok {
			return nil
		}
		if delim != '{' && delim != '[' {
			return domain.ErrEfectoPermisoSaldoInvalido
		}
		vistos := map[string]bool{}
		for d.More() {
			if delim == '{' {
				k, err := d.Token()
				clave, ok := k.(string)
				if err != nil || !ok || !claveJSON.MatchString(clave) || vistos[clave] {
					return domain.ErrEfectoPermisoSaldoInvalido
				}
				vistos[clave] = true
			}
			if err := recorrer(profundidad + 1); err != nil {
				return err
			}
		}
		cierre, err := d.Token()
		if err != nil || (delim == '{' && cierre != json.Delim('}')) || (delim == '[' && cierre != json.Delim(']')) {
			return domain.ErrEfectoPermisoSaldoInvalido
		}
		return nil
	}
	if err := recorrer(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return domain.ErrEfectoPermisoSaldoInvalido
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(destino) != nil {
		return domain.ErrEfectoPermisoSaldoInvalido
	}
	return nil
}

func Cargar(b []byte, esperado string) (Catalogo, error) {
	var w catalogoJSON
	if DecodificarEstricto(b, esperado, &w) != nil {
		return Catalogo{}, domain.ErrEfectoPermisoSaldoInvalido
	}
	c := Catalogo{VersionEsquema: w.VersionEsquema, Demostracion: w.Demostracion, PaqueteRef: w.PaqueteRef, Textos: w.Textos, Politica: domain.PoliticaEfectosPermisoSaldo{Referencia: w.Politica.Referencia, Version: w.Politica.Version, SHA256: esperado}}
	if w.Politica.Reglas != nil {
		c.Politica.Reglas = make([]domain.ReglaEfectoPermisoSaldo, len(w.Politica.Reglas))
		for i, r := range w.Politica.Reglas {
			c.Politica.Reglas[i] = domain.ReglaEfectoPermisoSaldo(r)
		}
	}
	if c.VersionEsquema != 1 || !c.Demostracion || c.PaqueteRef != "paquete:ejemplo:vec:v1" || c.Politica.Validar() != nil || len(c.Textos) < 1 || len(c.Textos) > 10 {
		return Catalogo{}, domain.ErrEfectoPermisoSaldoInvalido
	}
	for _, textos := range c.Textos {
		if len(textos) != 13 {
			return Catalogo{}, domain.ErrEfectoPermisoSaldoInvalido
		}
		for _, clave := range []string{"aviso", "fuera_vigencia", "concesion_incompleta", "sin_permisos", "permiso_computado", "programacion_ausente", "hechos_incompletos", "permisos_coincidentes", "concesion_no_acreditada", "permiso_parcial", "trabajo_concurrente", "regla_no_disponible", "entrada_invalida"} {
			if textos[clave] == "" || len(textos[clave]) > 1024 {
				return Catalogo{}, domain.ErrEfectoPermisoSaldoInvalido
			}
		}
	}
	return c, nil
}
