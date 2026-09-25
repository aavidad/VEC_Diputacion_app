package ports

import (
	"errors"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrEfectoAutorizacionBorradorInvalido = errors.New("dietas: efecto de autorizacion de borrador invalido")

const (
	EsquemaEfectoAutorizacionBorradorV1  = "vec.dietas.borrador-operacion.v1"
	EsquemaEfectoAutorizacionDocumentoV2 = "vec.dietas.borrador-operacion.v2"
	ModuloDietas                         = "dietas"
	TipoRecursoComisionBorrador          = "comision_borrador"
)

type EfectoAutorizacionBorrador struct {
	Material []byte
	Recurso  vecdomain.RecursoAutorizable
}
