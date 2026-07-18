package recibomaterial

import "crypto/sha256"

// SeleccionPlan contiene la identidad publica de un plan material versionado.
type SeleccionPlan struct {
	Referencia string
	Version    uint32
}

func SeleccionPlanValida(s SeleccionPlan) bool {
	return AliasLogicoValido(s.Referencia, 512) && s.Version > 0
}

// HechosContexto conserva solo los hechos estables necesarios para ligar un plan.
type HechosContexto struct {
	ModuloID, AccionNegocio, AccionTecnica string
	RecursoRef, OperacionRef, CargaRef     string
	EfectoRef, Clasificacion               string
}

func HechosContextoValidos(h HechosContexto) bool {
	return AliasLogicoValido(h.ModuloID, 128) && AliasLogicoValido(h.AccionNegocio, 256) &&
		AliasLogicoValido(h.AccionTecnica, 128) && AliasLogicoValido(h.RecursoRef, 512) &&
		AliasLogicoValido(h.OperacionRef, 512) && AliasLogicoValido(h.CargaRef, 512) &&
		AliasLogicoValido(h.EfectoRef, 512) && AliasLogicoValido(h.Clasificacion, 256)
}

// VinculoPlan liga la seleccion al conector y al contexto de negocio exacto.
type VinculoPlan struct {
	Seleccion        SeleccionPlan
	ConectorLogicoID string
	Hechos           HechosContexto
}

func CanonicoVinculoPlan(v VinculoPlan) ([]byte, error) {
	if !SeleccionPlanValida(v.Seleccion) || !AliasLogicoValido(v.ConectorLogicoID, 128) ||
		!HechosContextoValidos(v.Hechos) || v.Hechos.AccionTecnica != AccionEscribir {
		return nil, ErrReciboNoValido
	}
	var canonico []byte
	canonico = AnexarTLV(canonico, 0, []byte("vec.almacen.vinculo-plan-material.v2"))
	canonico = AnexarTLV(canonico, 1, Uint16(EsquemaVersion))
	canonico = AnexarTLV(canonico, 2, []byte(v.Seleccion.Referencia))
	canonico = AnexarTLV(canonico, 3, Uint32(v.Seleccion.Version))
	canonico = AnexarTLV(canonico, 4, []byte(v.ConectorLogicoID))
	canonico = AnexarTLV(canonico, 5, []byte(v.Hechos.ModuloID))
	canonico = AnexarTLV(canonico, 6, []byte(v.Hechos.AccionNegocio))
	canonico = AnexarTLV(canonico, 7, []byte(v.Hechos.AccionTecnica))
	canonico = AnexarTLV(canonico, 8, []byte(v.Hechos.RecursoRef))
	canonico = AnexarTLV(canonico, 9, []byte(v.Hechos.OperacionRef))
	canonico = AnexarTLV(canonico, 10, []byte(v.Hechos.CargaRef))
	canonico = AnexarTLV(canonico, 11, []byte(v.Hechos.EfectoRef))
	canonico = AnexarTLV(canonico, 12, []byte(v.Hechos.Clasificacion))
	return canonico, nil
}

func HuellaVinculoPlan(v VinculoPlan) ([sha256.Size]byte, error) {
	canonico, err := CanonicoVinculoPlan(v)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(canonico), nil
}
