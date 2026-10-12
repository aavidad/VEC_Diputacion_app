package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
)

// Los POST se limitan a consultas sin efecto de negocio. La auditoría de
// acceso corresponde al servidor real y no se sustituye en el banco.
var consultasPOST = map[string]bool{
	"/api/vec/contratacion-temporal/cuadro/consultas":      true,
	"/api/vec/contratacion-temporal/expedientes/consultas": true,
}

type casoLectura struct {
	Nombre        string          `json:"nombre"`
	Ruta          string          `json:"ruta"`
	Metodo        string          `json:"metodo"`
	Cuerpo        json.RawMessage `json:"cuerpo,omitempty"`
	PresupuestoMS float64         `json:"presupuesto_ms,omitempty"`
}

func validarDestino(base string) error {
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "https" || u.User != nil || u.ForceQuery || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("destino esperado=origen HTTPS local obtenido=no válido")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("destino esperado=loopback obtenido=no local")
	}
	return nil
}

func validarCaso(c *casoLectura) error {
	u, err := url.Parse(c.Ruta)
	if err != nil || !strings.HasPrefix(c.Ruta, "/") || strings.HasPrefix(c.Ruta, "//") || u.IsAbs() || u.Host != "" || u.Fragment != "" || strings.ContainsAny(c.Ruta, "\r\n") {
		return fmt.Errorf("ruta esperada=relativa al origen obtenido=no válida")
	}
	if c.Nombre == "" {
		c.Nombre = c.Ruta
	}
	if strings.ContainsAny(c.Nombre, "\r\n\t|") {
		return fmt.Errorf("nombre contiene separadores")
	}
	if c.Metodo == "" {
		c.Metodo = "GET"
	}
	if c.PresupuestoMS < 0 {
		return fmt.Errorf("presupuesto esperado>=0 obtenido=%v", c.PresupuestoMS)
	}
	switch c.Metodo {
	case "GET":
		if len(c.Cuerpo) != 0 {
			return fmt.Errorf("GET esperado=sin cuerpo obtenido=con cuerpo")
		}
	case "POST":
		if !consultasPOST[u.Path] || u.RawQuery != "" || !json.Valid(c.Cuerpo) {
			return fmt.Errorf("POST esperado=consulta CT con JSON obtenido=no permitido")
		}
	default:
		return fmt.Errorf("método esperado=GET o POST de consulta obtenido=%s", c.Metodo)
	}
	return nil
}

func cargarCasos(archivo string, rutas []string) ([]casoLectura, error) {
	var casos []casoLectura
	if archivo != "" {
		if len(rutas) != 0 {
			return nil, fmt.Errorf("casos y rutas son alternativas")
		}
		f, err := os.Open(archivo)
		if err != nil {
			return nil, fmt.Errorf("no se pudo abrir el fichero de casos")
		}
		dec := json.NewDecoder(io.LimitReader(f, 1<<20))
		dec.DisallowUnknownFields()
		err = dec.Decode(&casos)
		if err == nil {
			var extra any
			if dec.Decode(&extra) != io.EOF {
				err = fmt.Errorf("contenido adicional")
			}
		}
		cierre := f.Close()
		if err != nil || cierre != nil {
			return nil, fmt.Errorf("fichero de casos esperado=JSON válido hasta 1 MiB obtenido=no válido")
		}
	} else {
		for _, ruta := range rutas {
			casos = append(casos, casoLectura{Ruta: ruta})
		}
	}
	if len(casos) == 0 {
		return nil, fmt.Errorf("casos esperado>=1 obtenido=0")
	}
	nombres := map[string]bool{}
	for i := range casos {
		if err := validarCaso(&casos[i]); err != nil {
			return nil, fmt.Errorf("caso %d: %w", i+1, err)
		}
		if nombres[casos[i].Nombre] {
			return nil, fmt.Errorf("nombre de caso duplicado")
		}
		nombres[casos[i].Nombre] = true
	}
	return casos, nil
}
