package canonico

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

const FormatoFechaAvisoExterno = "2006-01-02T15:04:05.000000Z"

var referenciaAvisoExterno = regexp.MustCompile(`^[A-Za-z0-9:._-]{1,256}$`)

// MaterialAvisoExterno devuelve la única preimagen admitida por Go y SQL.
func MaterialAvisoExterno(e ports.EventoAvisoExterno) ([]byte, error) {
	for _, v := range []string{e.EventoRef, e.ProductorRef, e.CorrelacionRef, e.PlantillaRef, e.PlantillaVersion} {
		if !referenciaAvisoExterno.MatchString(v) {
			return nil, ports.ErrAvisoExternoInvalido
		}
	}
	t, err := time.Parse(FormatoFechaAvisoExterno, e.OcurridoEn)
	if err != nil || t.Year() < 1 || t.Format(FormatoFechaAvisoExterno) != e.OcurridoEn || e.TipoVersionado != ports.TipoAvisoLlamamientoExternoV1 ||
		!patronCandidatoAvisos.MatchString(e.DestinatarioExternoRef) || !patronLlamamientoAvisos.MatchString(e.ComunicacionRef) ||
		(e.RecursoPublicoRef != "" && !referenciaAvisoExterno.MatchString(e.RecursoPublicoRef)) {
		return nil, ports.ErrAvisoExternoInvalido
	}
	b, err := json.Marshal(e)
	if err != nil {
		return nil, ports.ErrAvisoExternoInvalido
	}
	return b, nil
}
func HuellaAvisoExterno(e ports.EventoAvisoExterno) (string, error) {
	b, err := MaterialAvisoExterno(e)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

// LeerAvisoExterno rechaza claves desconocidas, duplicadas, orden cambiado y
// tipos no string comprobando la preimagen completa, además de la estructura.
func LeerAvisoExterno(b []byte) (ports.EventoAvisoExterno, error) {
	var e ports.EventoAvisoExterno
	if len(b) == 0 || len(b) > 4096 {
		return e, ports.ErrAvisoExternoInvalido
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&e) != nil || d.Decode(new(any)) != io.EOF {
		return ports.EventoAvisoExterno{}, ports.ErrAvisoExternoInvalido
	}
	canon, err := MaterialAvisoExterno(e)
	if err != nil || !bytes.Equal(canon, b) {
		return ports.EventoAvisoExterno{}, ports.ErrAvisoExternoInvalido
	}
	return e, nil
}
