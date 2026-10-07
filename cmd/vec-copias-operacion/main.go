// vec-copias-operacion simulates declared progress from synthetic JSON only.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	"vec-diputacion-granada/internal/shared/i18n"
)

const maxBytes = 1 << 20
const maxPasos = 128

type entrada struct {
	Sintetica bool                        `json:"sintetica"`
	Solicitud operacionescopias.Solicitud `json:"solicitud"`
	Historia  []operacionescopias.Evento  `json:"historia,omitempty"`
	Comandos  []operacionescopias.Comando `json:"comandos"`
}

type paso struct {
	Secuencia     uint64                   `json:"secuencia"`
	VersionPrevia uint64                   `json:"version_previa"`
	Version       uint64                   `json:"version"`
	Estado        operacionescopias.Estado `json:"estado"`
	SelloSHA256   string                   `json:"sello_sha256"`
	Replay        bool                     `json:"replay"`
}

func main() {
	// Bound blocked stdin and filesystem reads too; no subprocesses are started.
	timer := time.AfterFunc(30*time.Second, func() { os.Exit(2) })
	code := ejecutar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	timer.Stop()
	os.Exit(code)
}

func ejecutar(args []string, in io.Reader, out, diagnostico io.Writer) int {
	f := flag.NewFlagSet("vec-copias-operacion", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	textos := f.String("textos", "", "")
	if f.Parse(args) != nil || f.NArg() != 0 || *textos == "" {
		return fallo(diagnostico, nil, "argumentos_invalidos")
	}
	catalogo, err := cargarCatalogo(*textos)
	if err != nil {
		return fallo(diagnostico, nil, "catalogo_invalido")
	}
	var e entrada
	if err := decodificar(in, &e); err != nil || !e.Sintetica || len(e.Comandos)+len(e.Historia) > maxPasos {
		return fallo(diagnostico, catalogo, "entrada_invalida")
	}
	o, err := operacionescopias.Reconstruir(e.Solicitud, e.Historia)
	if err != nil {
		return fallo(diagnostico, catalogo, codigo(err))
	}
	pasos := make([]paso, 0, len(e.Comandos))
	for _, c := range e.Comandos {
		next, recibo, replay, err := o.Aplicar(c)
		if errors.Is(err, operacionescopias.ErrVersion) {
			_ = json.NewEncoder(diagnostico).Encode(struct {
				Error    string `json:"error"`
				Clave    string `json:"clave"`
				Esperado uint64 `json:"esperado"`
				Obtenido uint64 `json:"obtenido"`
				Mensaje  string `json:"mensaje"`
			}{codigo(err), "version_esperada", o.Version(), c.VersionEsperada, catalogo.T(i18n.DefaultLocale, codigo(err))})
			return 2
		}
		if err != nil {
			return fallo(diagnostico, catalogo, codigo(err))
		}
		o = next
		pasos = append(pasos, paso{recibo.Secuencia, recibo.VersionPrevia, recibo.Version, recibo.Estado, recibo.SelloSHA256, replay})
	}
	r := struct {
		Alcance              string                   `json:"alcance"`
		Autenticidad         string                   `json:"autenticidad"`
		HabilitaCopia        bool                     `json:"habilita_copia"`
		HabilitaRestauracion bool                     `json:"habilita_restauracion"`
		RegistroDurable      bool                     `json:"registro_durable"`
		Estado               operacionescopias.Estado `json:"estado"`
		Version              uint64                   `json:"version"`
		Reconciliacion       string                   `json:"reconciliacion"`
		Mensaje              string                   `json:"mensaje"`
		Aviso                string                   `json:"aviso"`
		Pasos                []paso                   `json:"pasos"`
	}{Alcance: "simulacion_offline", Autenticidad: "no_comprobada", Estado: o.Estado(), Version: o.Version(), Reconciliacion: o.Reconciliar(), Mensaje: catalogo.T(i18n.DefaultLocale, o.Reconciliar()), Aviso: catalogo.T(i18n.DefaultLocale, "aviso_simulacion"), Pasos: pasos}
	if json.NewEncoder(out).Encode(r) != nil {
		return fallo(diagnostico, catalogo, "salida_fallida")
	}
	return 0
}

func cargarCatalogo(ruta string) (*i18n.Catalog, error) {
	// #nosec G304 -- Explicit local catalogue selected by CLI operator; no HTTP
	// input, server file access or error/path reflection in the output.
	f, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxBytes {
		return nil, errors.New("catalogo_invalido")
	}
	var textos map[string]string
	if err := decodificar(f, &textos); err != nil {
		return nil, err
	}
	for _, key := range []string{"aviso_simulacion", "revalidar_antes_de_captura", "conciliar_captura_sin_repetir", "revalidar_antes_de_ensayos", "conciliar_ensayos_pendientes", "autenticar_evidencias_antes_de_uso", "revisar_ensayo_fallido", "historia_invalida", "entrada_invalida", "operacion_entrada_invalida", "operacion_conflicto_idempotencia", "operacion_conflicto_version", "operacion_vinculo_distinto", "operacion_transicion_invalida", "operacion_historia_invalida", "salida_fallida"} {
		if textos[key] == "" {
			return nil, errors.New("catalogo_invalido")
		}
	}
	return i18n.New(i18n.DefaultLocale, map[string]map[string]string{i18n.DefaultLocale: textos})
}

func decodificar(r io.Reader, target any) error {
	b, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil || len(b) > maxBytes {
		return errors.New("entrada_invalida")
	}
	if err := comprobarClaves(json.NewDecoder(bytes.NewReader(b))); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("entrada_invalida")
	}
	return nil
}

// Every key in these closed DTOs and their catalogue uses lowercase ASCII,
// digits or underscore. Check raw keys before encoding/json can fold aliases
// or silently replace repeated values. Each object has its own duplicate set.
func comprobarClaves(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return errors.New("entrada_invalida")
	}
	switch token {
	case json.Delim('{'):
		vistos := make(map[string]bool)
		for d.More() {
			token, err := d.Token()
			clave, ok := token.(string)
			if err != nil || !ok || clave == "" || vistos[clave] {
				return errors.New("entrada_invalida")
			}
			for _, c := range clave {
				if !(c >= 'a' && c <= 'z') && !(c >= '0' && c <= '9') && c != '_' {
					return errors.New("entrada_invalida")
				}
			}
			vistos[clave] = true
			if err := comprobarClaves(d); err != nil {
				return err
			}
		}
		if token, err := d.Token(); err != nil || token != json.Delim('}') {
			return errors.New("entrada_invalida")
		}
	case json.Delim('['):
		for d.More() {
			if err := comprobarClaves(d); err != nil {
				return err
			}
		}
		if token, err := d.Token(); err != nil || token != json.Delim(']') {
			return errors.New("entrada_invalida")
		}
	}
	return nil
}

func codigo(err error) string {
	switch {
	case errors.Is(err, operacionescopias.ErrEntrada):
		return "operacion_entrada_invalida"
	case errors.Is(err, operacionescopias.ErrConflicto):
		return "operacion_conflicto_idempotencia"
	case errors.Is(err, operacionescopias.ErrVersion):
		return "operacion_conflicto_version"
	case errors.Is(err, operacionescopias.ErrVinculo):
		return "operacion_vinculo_distinto"
	case errors.Is(err, operacionescopias.ErrTransicion):
		return "operacion_transicion_invalida"
	case errors.Is(err, operacionescopias.ErrHistoria):
		return "operacion_historia_invalida"
	default:
		return "entrada_invalida"
	}
}

func fallo(out io.Writer, c *i18n.Catalog, key string) int {
	r := map[string]string{"error": key}
	if c != nil {
		r["mensaje"] = c.T(i18n.DefaultLocale, key)
	}
	_ = json.NewEncoder(out).Encode(r)
	return 2
}
