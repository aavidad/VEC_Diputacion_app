package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"time"

	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

const rutaMotivoAlegacionesBolsa = "identidad/bolsa-alegaciones.json"

type configuracionMotivoAlegacionesBolsa struct {
	Version            int                            `json:"version"`
	MotivoPresentacion core.ReferenciaEntradaCatalogo `json:"motivo_presentacion"`
}

func leerMotivoAlegacionesBolsa(directorio string) (core.ReferenciaEntradaCatalogo, error) {
	vacio := core.ReferenciaEntradaCatalogo{}
	b, err := leerFicheroMaterialSeguro(filepath.Join(directorio, rutaMotivoAlegacionesBolsa), 16<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacio, errMontajeInscripcionBolsa
	}
	defer borrarBytes(b)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var c configuracionMotivoAlegacionesBolsa
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoPresentacion) ||
		c.MotivoPresentacion.CatalogoID == motivoPresentarInscripcionBolsaDesarrollo().CatalogoID {
		return vacio, errMontajeInscripcionBolsa
	}
	return c.MotivoPresentacion, nil
}

type validadoresMotivoInscripcionAlegaciones struct {
	inscripcion         puertosvec.ValidadorReferenciaMotivoAutorizacionV2
	alegaciones         puertosvec.ValidadorReferenciaMotivoAutorizacionV2
	catalogoInscripcion string
	catalogoAlegaciones string
}

func (v validadoresMotivoInscripcionAlegaciones) ValidarReferenciaMotivoAutorizacionV2(
	ctx context.Context, motivo core.ReferenciaEntradaCatalogo, instante time.Time) error {
	switch motivo.CatalogoID {
	case v.catalogoInscripcion:
		return v.inscripcion.ValidarReferenciaMotivoAutorizacionV2(ctx, motivo, instante)
	case v.catalogoAlegaciones:
		return v.alegaciones.ValidarReferenciaMotivoAutorizacionV2(ctx, motivo, instante)
	default:
		return core.ErrSolicitudAutorizacionInvalida
	}
}

func nuevosValidadoresMotivoInscripcionAlegaciones(pool *pgxpool.Pool, referencia core.ReferenciaEntradaCatalogo) (
	puertosvec.ValidadorReferenciaMotivoAutorizacionV2, error) {
	inscripcion := motivoPresentarInscripcionBolsaDesarrollo()
	if pool == nil || !core.ReferenciaMotivoAutorizacionV2Valida(referencia) || referencia.CatalogoID == inscripcion.CatalogoID {
		return nil, errMontajeInscripcionBolsa
	}
	primero, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2Externo(pool, inscripcion.CatalogoID)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	segundo, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2Externo(pool, referencia.CatalogoID)
	if err != nil {
		return nil, errMontajeInscripcionBolsa
	}
	return validadoresMotivoInscripcionAlegaciones{primero, segundo, inscripcion.CatalogoID, referencia.CatalogoID}, nil
}
