package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func main() { os.Exit(ejecutar(os.Args[1:], os.Stdin, os.Stdout)) }

func ejecutar(args []string, entrada io.Reader, salida io.Writer) int {
	fs := flag.NewFlagSet("vec-auditoria-verificar", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	checkpointRuta := fs.String("checkpoint", "", "")
	maxBytes := fs.Int64("max-bytes", 0, "")
	maxRegistros := fs.Uint64("max-registros", 0, "")
	if fs.Parse(args) != nil || fs.NArg() != 0 || *checkpointRuta == "" ||
		*maxBytes <= 0 || *maxBytes > 1<<30 || *maxRegistros == 0 || entrada == nil {
		return responderFallo(salida, "argumentos_invalidos", "argumentos", 2)
	}
	// The path is explicitly selected by the local operator. The command has
	// no server endpoint, writes no files and never prints the path or content.
	f, err := os.Open(*checkpointRuta) // #nosec G304 -- explicit local CLI input, read-only.
	if err != nil {
		return responderFallo(salida, "checkpoint_no_disponible", "checkpoint", 2)
	}
	defer f.Close()
	var checkpoint auditoria.CoberturaCadena
	if !leerJSONEstricto(f, *maxBytes, &checkpoint) {
		return responderFallo(salida, "checkpoint_invalido", "checkpoint", 2)
	}
	var documento auditoria.DocumentoVerificacion
	if !leerJSONEstricto(entrada, *maxBytes, &documento) {
		return responderFallo(salida, "documento_invalido", "entrada", 2)
	}
	informe := auditoria.VerificarCadenaV3(documento, checkpoint, *maxRegistros)
	codigo := 0
	if informe.Estado != "verificada" {
		codigo = 1
	}
	return responder(salida, informe, codigo)
}

func responderFallo(salida io.Writer, codigo, clave string, salidaCodigo int) int {
	return responder(salida, auditoria.InformeVerificacion{
		Esquema: auditoria.EsquemaVerificacion, Estado: "rechazada", AutenticidadCheckpoint: "no_comprobada",
		Fallo: &auditoria.FalloVerificacion{Codigo: codigo, Clave: clave,
			Esperado: "entrada_valida_dentro_limite", Obtenido: "no_admitida"},
	}, salidaCodigo)
}

func responder(salida io.Writer, informe auditoria.InformeVerificacion, codigo int) int {
	if salida == nil || json.NewEncoder(salida).Encode(informe) != nil {
		return 4
	}
	return codigo
}

func leerJSONEstricto(r io.Reader, limite int64, destino any) bool {
	b, err := io.ReadAll(io.LimitReader(r, limite+1))
	if err != nil || int64(len(b)) > limite || len(b) == 0 {
		return false
	}
	// encoding/json accepts repeated object keys by default. Reject them
	// before decoding so a supplied manifest has one unambiguous meaning.
	tokens := json.NewDecoder(bytes.NewReader(b))
	tokens.UseNumber()
	if !valorJSONUnico(tokens, 0) {
		return false
	}
	if _, err = tokens.Token(); err != io.EOF {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(destino) != nil {
		return false
	}
	if decoder.Decode(new(any)) != io.EOF {
		return false
	}
	// All fields are mandatory, including zero values in an empty genesis.
	var objeto map[string]json.RawMessage
	if json.Unmarshal(b, &objeto) != nil {
		return false
	}
	switch destino.(type) {
	case *auditoria.CoberturaCadena:
		return clavesCoberturaExactas(objeto)
	case *auditoria.DocumentoVerificacion:
		var manifiesto map[string]json.RawMessage
		var registros []map[string]json.RawMessage
		if !clavesExactas(objeto, "esquema", "manifiesto", "registros") ||
			json.Unmarshal(objeto["manifiesto"], &manifiesto) != nil || !clavesCoberturaExactas(manifiesto) ||
			json.Unmarshal(objeto["registros"], &registros) != nil {
			return false
		}
		for _, registro := range registros {
			if !clavesExactas(registro, "auditoria_ref", "secuencia", "decision_ref", "efecto_ref",
				"huella_efecto_sha256", "anterior_sha256", "huella_sha256", "consumo_huella_sha256") {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func clavesCoberturaExactas(objeto map[string]json.RawMessage) bool {
	return clavesExactas(objeto, "cadena_id", "primera_secuencia", "ultima_secuencia",
		"anterior_sha256", "cabeza_sha256", "registros")
}

func clavesExactas(objeto map[string]json.RawMessage, claves ...string) bool {
	if len(objeto) != len(claves) {
		return false
	}
	for _, clave := range claves {
		if _, existe := objeto[clave]; !existe {
			return false
		}
	}
	return true
}

func valorJSONUnico(d *json.Decoder, profundidad int) bool {
	if profundidad > 16 {
		return false
	}
	t, err := d.Token()
	if err != nil || t == nil {
		return false
	}
	delim, compuesto := t.(json.Delim)
	if !compuesto {
		return true
	}
	if delim != '{' && delim != '[' {
		return false
	}
	claves := map[string]struct{}{}
	for d.More() {
		if delim == '{' {
			clave, err := d.Token()
			texto, ok := clave.(string)
			if err != nil || !ok {
				return false
			}
			if _, existe := claves[texto]; existe {
				return false
			}
			claves[texto] = struct{}{}
		}
		if !valorJSONUnico(d, profundidad+1) {
			return false
		}
	}
	cierre, err := d.Token()
	return err == nil && ((delim == '{' && cierre == json.Delim('}')) ||
		(delim == '[' && cierre == json.Delim(']')))
}
