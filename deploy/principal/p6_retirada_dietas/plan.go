package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	dominio "vec-diputacion-granada/internal/vec/domain"
)

const actor = "administracion:p6:retirada-dietas-r1d"
const acto = "acto:p6:revocacion-dietas-r1d:20260923"

type preimagen struct {
	AsignacionRef string                   `json:"asignacion_ref"`
	HuellaSHA256  string                   `json:"huella_sha256"`
	Asignacion    dominio.AsignacionPerfil `json:"documento"`
}

type plan struct {
	AnteriorRef    string                   `json:"anterior_ref"`
	AnteriorHuella string                   `json:"anterior_huella"`
	NuevaRef       string                   `json:"nueva_ref"`
	NuevaHuella    string                   `json:"nueva_huella"`
	Documento      dominio.AsignacionPerfil `json:"documento"`
	Actor          string                   `json:"actor"`
	Acto           string                   `json:"acto"`
}

func construir(p preimagen, ahora time.Time) (plan, error) {
	a := p.Asignacion
	if a.Validar() != nil || a.Referencia() != p.AsignacionRef ||
		a.Estado != dominio.EstadoAsignacionPerfilActiva ||
		!strings.HasPrefix(a.AsignacionID, "dietas_r1d_") ||
		a.VersionRolRef != "rol:dietas_r1d_provisional:v1" {
		return plan{}, errors.New("preimagen Dietas incompatible")
	}
	h, err := a.HuellaSHA256()
	if err != nil || h != p.HuellaSHA256 {
		return plan{}, errors.New("huella de preimagen incompatible con Go")
	}
	if a.Version >= int(^uint(0)>>1) || ahora.Before(a.EmitidaEn) {
		return plan{}, errors.New("version o instante de revocacion invalido")
	}
	b := a
	b.Version++
	b.Estado = dominio.EstadoAsignacionPerfilRevocada
	b.RevocadaPor = actor
	b.RevocadaEn = ahora.UTC().Truncate(time.Microsecond)
	b.RevocacionRef = acto
	if b.Validar() != nil {
		return plan{}, errors.New("revocacion rechazada por Validar de Go")
	}
	hb, err := b.HuellaSHA256()
	if err != nil {
		return plan{}, err
	}
	return plan{p.AsignacionRef, h, b.Referencia(), hb, b, actor, acto}, nil
}

func main() {
	var p preimagen
	dec := json.NewDecoder(os.Stdin)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		fail(err)
	}
	res, err := construir(p, time.Now().UTC())
	if err != nil {
		fail(err)
	}
	contenido, err := json.Marshal(res)
	if err != nil {
		fail(err)
	}
	fmt.Printf("\\set p6_plan_b64 %s\n", base64.StdEncoding.EncodeToString(contenido))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
