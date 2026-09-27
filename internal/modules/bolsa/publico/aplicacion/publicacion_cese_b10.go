package aplicacion

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
)

var ErrPublicacionCeseB10NoDisponible = errors.New("bolsa: publicacion de cese B10 no disponible")

var patronOrigenCeseB10 = regexp.MustCompile(`^evento:ct:contrato-bolsa:[a-f0-9]{64}$`)

// CesePendienteB10 procede exclusivamente del feed durable de Bolsa B49. El
// evento no forma parte del material que se entrega a la base pública.
type CesePendienteB10 struct {
	OrigenPosicion int64
	OrigenRef      string
	EventoRef      string
	BolsaRef       string
	Fase           string
}

type FeedPublicacionCeseB10 interface {
	SiguientePublicacionCeseB10(context.Context) (CesePendienteB10, bool, error)
	ConfirmarPublicacionCeseB10(context.Context, CesePendienteB10, string) (bool, error)
}

// InstantaneaCeseB10 debe proceder de una lectura fresca y consistente de
// Bolsa. La fuente incluye el cese indicado y todas las participaciones del
// candidato afectado antes de descartar las referencias internas. Debe
// conservar exactamente el mismo material al reintentar ese evento; V2
// conserva su definición externa y B10 solo contiene orden, documento
// enmascarado y estado.
type InstantaneaCeseB10 struct {
	OrigenPosicionIncluida int64
	OrigenRefIncluido      string
	ProyeccionV2           []byte
	ManifiestoV2           []byte
	BolsasV1               []byte
}

type FuenteInstantaneaCeseB10 interface {
	PrepararInstantaneaCeseB10(context.Context, CesePendienteB10) (InstantaneaCeseB10, error)
}

// DestinoPublicacionCeseB10 corresponde a la transacción V3 de la base
// pública separada. Solo devuelve nil después del COMMIT confirmado.
type DestinoPublicacionCeseB10 func(context.Context, []byte, []byte, string) error

type ReciboPublicacionCeseB10 struct {
	OrigenPosicion int64
	OrigenRef      string
	Fase           string
	AnclaSHA256    string
	Reutilizada    bool
}

// PublicarSiguienteCeseB10 hace un paso recuperable: feed B49, instantánea,
// publicación atómica V3 y confirmación B49, en ese orden. Si cae después del
// COMMIT público y antes de confirmar B49, el mismo evento permanece pendiente
// y se verifica el replay exacto antes de avanzar. La fuente debe conservar
// la misma instantánea mientras ese evento esté pendiente.
func PublicarSiguienteCeseB10(
	ctx context.Context,
	feed FeedPublicacionCeseB10,
	instante FuenteInstantaneaCeseB10,
	publicar DestinoPublicacionCeseB10,
) (ReciboPublicacionCeseB10, bool, error) {
	if ctx == nil || feed == nil || instante == nil || publicar == nil {
		return ReciboPublicacionCeseB10{}, false, ErrPublicacionCeseB10NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ReciboPublicacionCeseB10{}, false, err
	}
	evento, pendiente, err := feed.SiguientePublicacionCeseB10(ctx)
	if err != nil {
		return ReciboPublicacionCeseB10{}, false, ErrPublicacionCeseB10NoDisponible
	}
	if !pendiente {
		return ReciboPublicacionCeseB10{}, false, nil
	}
	if evento.OrigenPosicion < 0 || evento.OrigenRef == "" ||
		strings.TrimSpace(evento.BolsaRef) != evento.BolsaRef || evento.BolsaRef == "" ||
		(evento.Fase != "cese" && evento.Fase != "vencimiento") ||
		!patronOrigenCeseB10.MatchString(evento.EventoRef) {
		return ReciboPublicacionCeseB10{}, false, ErrPublicacionCeseB10NoDisponible
	}
	snapshot, err := instante.PrepararInstantaneaCeseB10(ctx, evento)
	if err != nil || snapshot.OrigenPosicionIncluida != evento.OrigenPosicion || snapshot.OrigenRefIncluido != evento.OrigenRef {
		return ReciboPublicacionCeseB10{}, false, ErrPublicacionCeseB10NoDisponible
	}
	material, err := canonico.PrepararMaterialPublicacionV3(snapshot.ProyeccionV2, snapshot.ManifiestoV2, snapshot.BolsasV1)
	if err != nil {
		return ReciboPublicacionCeseB10{}, false, ErrPublicacionCeseB10NoDisponible
	}
	if err := publicar(ctx, material.ProyeccionV2, material.BolsasV1, material.AnclaManifiestoSHA256); err != nil {
		return ReciboPublicacionCeseB10{}, false, ErrPublicacionCeseB10NoDisponible
	}
	reutilizada, err := feed.ConfirmarPublicacionCeseB10(ctx, evento, material.AnclaManifiestoSHA256)
	if err != nil {
		return ReciboPublicacionCeseB10{}, false, ErrPublicacionCeseB10NoDisponible
	}
	return ReciboPublicacionCeseB10{
		OrigenPosicion: evento.OrigenPosicion, OrigenRef: evento.OrigenRef, Fase: evento.Fase,
		AnclaSHA256: material.AnclaManifiestoSHA256, Reutilizada: reutilizada,
	}, true, nil
}
