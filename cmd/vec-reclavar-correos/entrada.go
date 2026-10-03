package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"syscall"
)

var refPersona = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
var refOperacion = regexp.MustCompile(`^[A-Za-z0-9:._-]{16,128}$`)
var digestValido = regexp.MustCompile(`^[0-9a-f]{64}$`)

type objetivo struct {
	Persona   string `json:"persona_ref"`
	Preimagen string `json:"preimagen_sha256"`
}

type plan struct {
	Esquema        string          `json:"esquema"`
	Base           string          `json:"base"`
	Sistema        string          `json:"sistema_postgresql"`
	Conexion       destinoConexion `json:"conexion"`
	ConexionHuella string          `json:"conexion_sha256"`
	Lote           string          `json:"lote_ref"`
	Aprobacion     string          `json:"aprobacion_ref"`
	Personas       []objetivo      `json:"personas"`
}

// Sólo descriptores heredados de ficheros privados. No rutas, claves o DSN
// en argumentos, variables del proceso, mensajes o JSON de salida.
func leerDescriptor(fd int, limite int64) ([]byte, error) {
	if fd < 3 || fd > 1024 {
		return nil, errEntrada
	}
	f := os.NewFile(uintptr(fd), "material-privado") // #nosec G115 -- intervalo 3..1024 comprobado.
	if f == nil {
		return nil, errEntrada
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 || st.Size() <= 0 || st.Size() > limite {
		return nil, errEntrada
	}
	meta, ok := st.Sys().(*syscall.Stat_t)
	if !ok || uint64(meta.Uid) != uint64(os.Geteuid()) || meta.Nlink != 1 {
		return nil, errEntrada
	} // #nosec G115 -- uid del SO no negativo.
	b, err := io.ReadAll(io.LimitReader(f, limite+1))
	if err != nil || int64(len(b)) != st.Size() {
		clear(b)
		return nil, errEntrada
	}
	return b, nil
}

func cargarPlan(b []byte, modo string) (plan, error) {
	if err := validarJSONSinDuplicados(b); err != nil {
		return plan{}, err
	}
	var p plan
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil || d.Decode(new(any)) != io.EOF || p.Esquema != "usuarios.correos.reclaveado.plan.v1" || !regexp.MustCompile(`^[0-9]{1,20}$`).MatchString(p.Sistema) || p.Base == "" || len(p.Base) > 128 || p.ConexionHuella != huellaConexion(p.Base, p.Conexion) || !refOperacion.MatchString(p.Lote) || !refOperacion.MatchString(p.Aprobacion) || len(p.Personas) == 0 || len(p.Personas) > 10000 {
		return plan{}, errEntrada
	}
	vistas := map[string]bool{}
	for _, o := range p.Personas {
		if !refPersona.MatchString(o.Persona) || vistas[o.Persona] || (modo != "inventario" && !digestValido.MatchString(o.Preimagen)) || (o.Preimagen != "" && !digestValido.MatchString(o.Preimagen)) {
			return plan{}, errEntrada
		}
		vistas[o.Persona] = true
	}
	return p, nil
}

func validarJSONSinDuplicados(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var valor func(int) error
	valor = func(profundidad int) error {
		if profundidad > 32 {
			return errEntrada
		}
		t, err := d.Token()
		if err != nil {
			return errEntrada
		}
		inicio, es := t.(json.Delim)
		if !es {
			return nil
		}
		vistos := map[string]bool{}
		for d.More() {
			if inicio == '{' {
				k, err := d.Token()
				if err != nil {
					return errEntrada
				}
				nombre, ok := k.(string)
				if !ok || vistos[nombre] {
					return errEntrada
				}
				vistos[nombre] = true
			}
			if err := valor(profundidad + 1); err != nil {
				return err
			}
		}
		fin, err := d.Token()
		if err != nil {
			return errEntrada
		}
		if (inicio == '{' && fin == json.Delim('}')) || (inicio == '[' && fin == json.Delim(']')) {
			return nil
		}
		return errEntrada
	}
	if err := valor(0); err != nil {
		return err
	}
	_, err := d.Token()
	if err != io.EOF {
		return errEntrada
	}
	return nil
}
