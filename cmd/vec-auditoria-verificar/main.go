package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"

	"vec-diputacion-granada/internal/vec/auditoria"
)

var errJSONInvalido = errors.New("vec auditoria: JSON invalido")

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
	if leerJSONEstricto(f, *maxBytes, &checkpoint) != nil {
		return responderFallo(salida, "checkpoint_invalido", "checkpoint", 2)
	}
	contenido, err := io.ReadAll(io.LimitReader(entrada, *maxBytes+1))
	if err != nil || len(contenido) == 0 || int64(len(contenido)) > *maxBytes {
		return responderFallo(salida, "documento_invalido", "entrada", 2)
	}
	var cabecera struct {
		Esquema string `json:"esquema"`
	}
	if json.Unmarshal(contenido, &cabecera) != nil {
		return responderFallo(salida, "documento_invalido", "entrada", 2)
	}
	var informe auditoria.InformeVerificacion
	switch cabecera.Esquema {
	case auditoria.EsquemaVerificacion:
		var documento auditoria.DocumentoVerificacion
		if decodificarJSONEstricto(contenido, &documento) != nil {
			return responderFallo(salida, "documento_invalido", "entrada", 2)
		}
		informe = auditoria.VerificarCadenaV3(documento, checkpoint, *maxRegistros)
	case auditoria.EsquemaVerificacionMixta:
		var documento auditoria.DocumentoVerificacionMixta
		if decodificarJSONEstricto(contenido, &documento) != nil {
			return responderFallo(salida, "documento_invalido", "entrada", 2)
		}
		informe = auditoria.VerificarCadenaMixtaV2(documento, checkpoint, *maxRegistros)
	case auditoria.EsquemaVerificacionPreperfil:
		var documento auditoria.DocumentoVerificacionMixta
		if decodificarJSONEstrictoPreperfil(contenido, &documento) != nil {
			return responderFallo(salida, "documento_invalido", "entrada", 2)
		}
		informe := auditoria.VerificarCadenaMixtaV3(documento, checkpoint, *maxRegistros)
		codigo := 0
		if informe.Estado != "verificada" {
			codigo = 1
		}
		return responder(salida, informe, codigo)
	case auditoria.EsquemaVerificacionFuentesIniciales:
		var documento auditoria.DocumentoVerificacionMixta
		if decodificarJSONEstricto(contenido, &documento) != nil {
			return responderFallo(salida, "documento_invalido", "entrada", 2)
		}
		informe := auditoria.VerificarCadenaFuentesInicialesV1(documento, checkpoint, *maxRegistros)
		codigo := 0
		if informe.Estado != "verificada" {
			codigo = 1
		}
		return responder(salida, informe, codigo)
	case auditoria.EsquemaVerificacionUnidadInicial:
		var documento auditoria.DocumentoVerificacionMixta
		if decodificarJSONEstricto(contenido, &documento) != nil {
			return responderFallo(salida, "documento_invalido", "entrada", 2)
		}
		informe := auditoria.VerificarCadenaUnidadInicialV1(documento, checkpoint, *maxRegistros)
		codigo := 0
		if informe.Estado != "verificada" {
			codigo = 1
		}
		return responder(salida, informe, codigo)
	default:
		return responderFallo(salida, "documento_invalido", "entrada", 2)
	}
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

func responder(salida io.Writer, informe any, codigo int) int {
	if salida == nil || json.NewEncoder(salida).Encode(informe) != nil {
		return 4
	}
	return codigo
}

func leerJSONEstricto(r io.Reader, limite int64, destino any) error {
	b, err := io.ReadAll(io.LimitReader(r, limite+1))
	if err != nil {
		return err
	}
	if int64(len(b)) > limite || len(b) == 0 {
		return errJSONInvalido
	}
	return decodificarJSONEstricto(b, destino)
}

func decodificarJSONEstricto(b []byte, destino any) error {
	// encoding/json accepts repeated object keys by default. Reject them
	// before decoding so a supplied manifest has one unambiguous meaning.
	tokens := json.NewDecoder(bytes.NewReader(b))
	tokens.UseNumber()
	if err := valorJSONUnico(tokens, 0); err != nil {
		return err
	}
	if _, err := tokens.Token(); err != io.EOF {
		return errJSONInvalido
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	if decoder.Decode(destino) != nil {
		return errJSONInvalido
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errJSONInvalido
	}
	// All fields are mandatory, including zero values in an empty genesis.
	var objeto map[string]json.RawMessage
	if json.Unmarshal(b, &objeto) != nil {
		return errJSONInvalido
	}
	switch destino.(type) {
	case *auditoria.CoberturaCadena:
		if !clavesCoberturaExactas(objeto) {
			return errJSONInvalido
		}
		return nil
	case *auditoria.DocumentoVerificacion:
		var manifiesto map[string]json.RawMessage
		var registros []map[string]json.RawMessage
		if !clavesExactas(objeto, "esquema", "manifiesto", "registros") ||
			json.Unmarshal(objeto["manifiesto"], &manifiesto) != nil || !clavesCoberturaExactas(manifiesto) ||
			json.Unmarshal(objeto["registros"], &registros) != nil {
			return errJSONInvalido
		}
		for _, registro := range registros {
			if !clavesExactas(registro, "auditoria_ref", "secuencia", "decision_ref", "efecto_ref",
				"huella_efecto_sha256", "anterior_sha256", "huella_sha256", "consumo_huella_sha256") {
				return errJSONInvalido
			}
		}
		return nil
	case *auditoria.DocumentoVerificacionMixta:
		if destino.(*auditoria.DocumentoVerificacionMixta).Esquema == auditoria.EsquemaVerificacionPreperfil {
			return clavesDocumentoPreperfil(objeto)
		}
		if destino.(*auditoria.DocumentoVerificacionMixta).Esquema == auditoria.EsquemaVerificacionFuentesIniciales {
			return clavesDocumentoFuentesIniciales(objeto)
		}
		if destino.(*auditoria.DocumentoVerificacionMixta).Esquema == auditoria.EsquemaVerificacionUnidadInicial {
			return clavesDocumentoUnidadInicial(objeto)
		}
		return clavesDocumentoMixto(objeto, false)
	default:
		return errJSONInvalido
	}
}

func clavesIntentoExactas(c map[string]json.RawMessage) bool {
	return clavesExactas(c, "auditoria_ref", "secuencia", "anterior_sha256", "huella_sha256",
		"registrada_en", "intento_ref", "intento_material_sha256", "actor_ref", "perfil_activo_ref",
		"registro_contexto_ref", "contexto_sha256", "procedencia_sha256", "autenticacion_ref", "sesion_ref",
		"autenticacion_sha256", "accion", "modulo_id", "recurso_ref", "finalidad_ref", "resultado",
		"motivo_ref", "proceso", "canal", "correlacion_ref", "vinculo_sha256", "contexto_canonico_base64")
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

func valorJSONUnico(d *json.Decoder, profundidad int) error {
	if profundidad > 16 {
		return errJSONInvalido
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	if t == nil {
		return errJSONInvalido
	}
	delim, compuesto := t.(json.Delim)
	if !compuesto {
		return nil
	}
	if delim != '{' && delim != '[' {
		return errJSONInvalido
	}
	claves := map[string]struct{}{}
	for d.More() {
		if delim == '{' {
			clave, err := d.Token()
			texto, ok := clave.(string)
			if err != nil {
				return err
			}
			if !ok {
				return errJSONInvalido
			}
			if _, existe := claves[texto]; existe {
				return errJSONInvalido
			}
			claves[texto] = struct{}{}
		}
		if err := valorJSONUnico(d, profundidad+1); err != nil {
			return err
		}
	}
	cierre, err := d.Token()
	if err != nil {
		return err
	}
	if !((delim == '{' && cierre == json.Delim('}')) ||
		(delim == '[' && cierre == json.Delim(']'))) {
		return errJSONInvalido
	}
	return nil
}
