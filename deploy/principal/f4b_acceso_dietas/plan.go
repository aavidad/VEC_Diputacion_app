package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	dominio "vec-diputacion-granada/internal/vec/domain"
)

// F4b sustituye a F4 (nunca aplicado en la principal): mismas reglas V3 sobre
// la preimagen P6, con actor y acto propios para no confundir su historia.
const actorF4 = "administracion:f4b:acceso-dietas"
const actoF4 = "acto:f4b:acceso-dietas:20260925"
const actorP6 = "administracion:p6:retirada-dietas-r1d"
const actoP6 = "acto:p6:revocacion-dietas-r1d:20260923"
const actorRetiradaF4 = "administracion:f4b:retirada-acceso-dietas"
const actoRetiradaF4 = "acto:f4b:retirada-acceso-dietas:20260925"

type fila struct {
	Ref       string                   `json:"asignacion_ref"`
	Huella    string                   `json:"huella_sha256"`
	Documento dominio.AsignacionPerfil `json:"documento"`
}

type preimagen struct {
	Original    fila   `json:"original"`
	Revocada    fila   `json:"revocada"`
	PunteroPor  string `json:"actualizada_por"`
	PunteroActo string `json:"acto_ref"`
}

type plan struct {
	OriginalRef    string                   `json:"original_ref"`
	OriginalHuella string                   `json:"original_huella"`
	AnteriorRef    string                   `json:"anterior_ref"`
	AnteriorHuella string                   `json:"anterior_huella"`
	NuevaRef       string                   `json:"nueva_ref"`
	NuevaHuella    string                   `json:"nueva_huella"`
	Documento      dominio.AsignacionPerfil `json:"documento"`
	Actor          string                   `json:"actor"`
	Acto           string                   `json:"acto"`
}

type preimagenRetirada struct {
	Activa      fila   `json:"activa"`
	PunteroPor  string `json:"actualizada_por"`
	PunteroActo string `json:"acto_ref"`
}

type planRetirada struct {
	AnteriorRef    string                   `json:"anterior_ref"`
	AnteriorHuella string                   `json:"anterior_huella"`
	NuevaRef       string                   `json:"nueva_ref"`
	NuevaHuella    string                   `json:"nueva_huella"`
	Documento      dominio.AsignacionPerfil `json:"documento"`
	Actor          string                   `json:"actor"`
	Acto           string                   `json:"acto"`
}

func construir(p preimagen, ahora time.Time) (plan, error) {
	o, r := p.Original.Documento, p.Revocada.Documento
	for _, f := range []fila{p.Original, p.Revocada} {
		if f.Documento.Validar() != nil || f.Documento.Referencia() != f.Ref {
			return plan{}, errors.New("preimagen de asignacion incompatible")
		}
		h, err := f.Documento.HuellaSHA256()
		if err != nil || h != f.Huella {
			return plan{}, errors.New("huella de preimagen incompatible")
		}
	}
	if o.Version != 1 || r.Version != 2 ||
		o.Estado != dominio.EstadoAsignacionPerfilActiva ||
		r.Estado != dominio.EstadoAsignacionPerfilRevocada ||
		o.VersionRolRef != "rol:dietas_r1d_provisional:v1" ||
		r.VersionRolRef != o.VersionRolRef ||
		r.AsignacionID != o.AsignacionID ||
		r.PerfilActivoRef != o.PerfilActivoRef || r.PrincipalID != o.PrincipalID ||
		p.PunteroPor != actorP6 || p.PunteroActo != actoP6 ||
		r.RevocadaPor != actorP6 || r.RevocacionRef != actoP6 ||
		r.RevocadaEn.Before(o.EmitidaEn) ||
		!mismoContenidoP6(o, r) {
		return plan{}, errors.New("preimagen no corresponde a retirada P6 exacta")
	}
	ahora = ahora.UTC().Truncate(time.Microsecond)
	if !ahora.After(r.RevocadaEn) || !ahora.Before(r.VigenteHasta) {
		return plan{}, errors.New("vigencia F4b agotada o reloj incompatible")
	}
	b := r
	b.Version = 3
	b.Estado = dominio.EstadoAsignacionPerfilActiva
	b.VigenteDesde = ahora
	b.EmitidaPor = actorF4
	b.EmitidaEn = ahora
	b.RevocadaPor = ""
	b.RevocadaEn = time.Time{}
	b.RevocacionRef = ""
	if b.Validar() != nil {
		return plan{}, errors.New("nueva asignacion rechazada por el dominio")
	}
	h, err := b.HuellaSHA256()
	if err != nil {
		return plan{}, err
	}
	return plan{p.Original.Ref, p.Original.Huella, p.Revocada.Ref, p.Revocada.Huella,
		b.Referencia(), h, b, actorF4, actoF4}, nil
}

func mismoContenidoP6(o, r dominio.AsignacionPerfil) bool {
	a, errA := json.Marshal(o.Ambitos)
	b, errB := json.Marshal(r.Ambitos)
	return errA == nil && errB == nil && string(a) == string(b) &&
		o.VigenteDesde.Equal(r.VigenteDesde) && o.VigenteHasta.Equal(r.VigenteHasta) &&
		o.EmitidaPor == r.EmitidaPor && o.EmitidaEn.Equal(r.EmitidaEn)
}

func construirRetirada(p preimagenRetirada, ahora time.Time) (planRetirada, error) {
	a := p.Activa.Documento
	if a.Validar() != nil || a.Referencia() != p.Activa.Ref ||
		a.Version != 3 || a.Estado != dominio.EstadoAsignacionPerfilActiva ||
		a.VersionRolRef != "rol:dietas_r1d_provisional:v1" ||
		a.EmitidaPor != actorF4 || p.PunteroPor != actorF4 || p.PunteroActo != actoF4 {
		return planRetirada{}, errors.New("preimagen de retirada F4b incompatible")
	}
	h, err := a.HuellaSHA256()
	if err != nil || h != p.Activa.Huella {
		return planRetirada{}, errors.New("huella de retirada F4b incompatible")
	}
	ahora = ahora.UTC().Truncate(time.Microsecond)
	if !ahora.After(a.EmitidaEn) {
		return planRetirada{}, errors.New("reloj de retirada F4b incompatible")
	}
	b := a
	b.Version = 4
	b.Estado = dominio.EstadoAsignacionPerfilRevocada
	b.RevocadaPor = actorRetiradaF4
	b.RevocadaEn = ahora
	b.RevocacionRef = actoRetiradaF4
	if b.Validar() != nil {
		return planRetirada{}, errors.New("retirada rechazada por el dominio")
	}
	hb, err := b.HuellaSHA256()
	if err != nil {
		return planRetirada{}, err
	}
	return planRetirada{p.Activa.Ref, h, b.Referencia(), hb, b, actorRetiradaF4, actoRetiradaF4}, nil
}

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--retirar" {
		var p preimagenRetirada
		d := json.NewDecoder(os.Stdin)
		d.DisallowUnknownFields()
		if err := d.Decode(&p); err != nil {
			fail(err)
		}
		resultado, err := construirRetirada(p, time.Now().UTC())
		if err != nil {
			fail(err)
		}
		contenido, err := json.Marshal(resultado)
		if err != nil {
			fail(err)
		}
		fmt.Printf("\\set f4b_retirada_plan_b64 %s\n", base64.StdEncoding.EncodeToString(contenido))
		return
	}
	if len(os.Args) != 1 {
		fail(errors.New("modo de plan F4b desconocido"))
	}
	var p preimagen
	d := json.NewDecoder(os.Stdin)
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		fail(err)
	}
	resultado, err := construir(p, time.Now().UTC())
	if err != nil {
		fail(err)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		fail(err)
	}
	fmt.Printf("\\set f4b_plan_b64 %s\n", base64.StdEncoding.EncodeToString(contenido))
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
