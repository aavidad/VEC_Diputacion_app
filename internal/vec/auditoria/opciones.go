package auditoria

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

const RutaOpciones = "/api/vec/auditoria/opciones"

type LectorCatalogoVigente interface {
	CatalogoVigente(context.Context) (domain.CatalogoConfigurable, string, time.Time, error)
}

type Opciones struct {
	FinalidadRef     string                           `json:"finalidad_ref"`
	MotivoRef        string                           `json:"motivo_ref"`
	PermisoRequerido string                           `json:"permiso_requerido"`
	EsEjemplo        bool                             `json:"es_ejemplo"`
	Fuentes          []string                         `json:"fuentes"`
	Motivo           domain.ReferenciaEntradaCatalogo `json:"-"`
}

type ProveedorOpciones interface {
	Actuales(context.Context) (Opciones, error)
}

// OpcionesCatalogo lee la versión publicada en cada petición. La referencia
// del motivo procede de la entrada elegida y conserva la huella del catálogo.
// La validación PostgreSQL confirma además que esa publicación sigue vigente.
type OpcionesCatalogo struct {
	lector    LectorCatalogoVigente
	validador ports.ValidadorReferenciaMotivoAutorizacionV2
}

func NuevoProveedorOpcionesCatalogo(lector LectorCatalogoVigente, validador ports.ValidadorReferenciaMotivoAutorizacionV2) (*OpcionesCatalogo, error) {
	if dependenciaNula(lector) || dependenciaNula(validador) {
		return nil, ErrNoDisponible
	}
	return &OpcionesCatalogo{lector: lector, validador: validador}, nil
}

// Configuradas obtiene una referencia del catálogo para preparar su
// publicación controlada en la autoridad de motivos. No concede consulta.
func (p *OpcionesCatalogo) Configuradas(ctx context.Context) (Opciones, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || dependenciaNula(p.lector) {
		return Opciones{}, ErrNoDisponible
	}
	catalogo, huella, instante, err := p.lector.CatalogoVigente(ctx)
	if err != nil || catalogo.ID == "" || catalogo.ModuloID != "auditoria" || !patronHuella.MatchString(huella) {
		return Opciones{}, ErrNoDisponible
	}
	var elegidas []domain.EntradaCatalogoConfigurable
	for _, entrada := range catalogo.Entradas {
		if entrada.VigenteEn(instante) && entrada.Atributos["uso"] == "consulta_rrhh" {
			elegidas = append(elegidas, entrada)
		}
	}
	if len(elegidas) != 1 {
		return Opciones{}, ErrNoDisponible
	}
	e := elegidas[0]
	if e.Atributos["permiso"] != AccionConsultar || e.Atributos["tipo_recurso"] != TipoRecurso ||
		e.Atributos["ambito"] != "expediente_exacto" || e.Atributos["fuentes"] != "ct,bolsa" ||
		!referenciaExacta(e.Atributos["finalidad_ref"], 128) {
		return Opciones{}, ErrNoDisponible
	}
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: catalogo.ID, CatalogoVersion: catalogo.Version, CatalogoHuellaSHA256: huella, EntradaClave: e.Clave}
	if !domain.ReferenciaMotivoAutorizacionV2Valida(motivo) || !referenciaExacta(motivo.Referencia(), 128) {
		return Opciones{}, ErrNoDisponible
	}
	return Opciones{FinalidadRef: e.Atributos["finalidad_ref"], MotivoRef: motivo.Referencia(), PermisoRequerido: AccionConsultar, EsEjemplo: catalogo.FuenteRef == reglas.MarcaPaqueteEjemplo, Fuentes: []string{"ct", "bolsa"}, Motivo: motivo}, nil
}

func (p *OpcionesCatalogo) Actuales(ctx context.Context) (Opciones, error) {
	o, err := p.Configuradas(ctx)
	if err != nil || dependenciaNula(p.validador) {
		return Opciones{}, ErrNoDisponible
	}
	if err := p.validador.ValidarReferenciaMotivoAutorizacionV2(ctx, o.Motivo, time.Now().UTC().Truncate(time.Microsecond)); err != nil {
		return Opciones{}, ErrNoDisponible
	}
	return o, nil
}
