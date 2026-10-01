package ejecucioncopias

import (
	"context"
	"errors"
	"sync"

	lector "vec-diputacion-granada/internal/modules/administracion/adapters/contrastecopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	aislado "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

var ErrEnsayoEnsemble = errors.New("copias_ejecucion_ensayo_no_comprobable")

type ResultadoArranque struct{ Ref, SaludRef, ConsultaRef, DespachosRef string }

// ArrancadorConjunto usa el runner propio CS06 y sondas reales del binario VEC
// archivado. Sus referencias proceden de observaciones, nunca de configuración.
type ArrancadorConjunto interface {
	Arrancar(context.Context, aislado.Entorno, copias.Manifiesto) (ResultadoArranque, error)
}
type InspectorFicheros interface {
	MedirFicheros(context.Context, aislado.Entorno, copias.Manifiesto) ([]copias.Artefacto, error)
}

type ObservadorEnsemble struct {
	Lector     *lector.Lector
	Base       string
	Conjunto   p.Conjunto
	Modo       p.ModoEnsayo
	Arrancador ArrancadorConjunto
	Ficheros   InspectorFicheros
	mu         sync.Mutex
	evidencia  copias.Evidencia
	completo   bool
}

func (o *ObservadorEnsemble) Observar(ctx context.Context, e aislado.Entorno) (aislado.Observacion, error) {
	fallida := aislado.Observacion{ContrasteEstado: "no_comprobable", ArranqueEstado: "no_comprobable"}
	if o == nil || o.Lector == nil || o.Base == "" || e.PostgreSQL == nil || e.Archivado == nil || o.Arrancador == nil || o.Ficheros == nil || !e.Aislamiento.RedSinSalida || !e.Aislamiento.VolumenesPropios || !e.Aislamiento.ConfiguracionOrigenExcluida || !e.Aislamiento.RuntimeInspeccionado {
		return fallida, ErrEnsayoEnsemble
	}
	snapshot, err := o.Lector.CapturarEjecutor(ctx, e.PostgreSQL, o.Base, e.PostgreSQL)
	if err != nil {
		return fallida, ErrEnsayoEnsemble
	}
	archivos, err := o.Ficheros.MedirFicheros(ctx, e, o.Conjunto.Manifiesto)
	if err != nil {
		return fallida, ErrEnsayoEnsemble
	}
	arranque, err := o.Arrancador.Arrancar(ctx, e, o.Conjunto.Manifiesto)
	if err != nil || arranque.Ref == "" || arranque.SaludRef == "" || arranque.ConsultaRef == "" || arranque.DespachosRef == "" {
		return fallida, ErrEnsayoEnsemble
	}
	evidencia, err := evidenciaSnapshot(snapshot, archivos, "ensayo:"+string(o.Modo)+":"+o.Conjunto.Ref, arranque.Ref)
	if err != nil || !mismoDatos(o.Conjunto.Origen, evidencia) {
		return fallida, ErrEnsayoEnsemble
	}
	o.mu.Lock()
	o.evidencia = evidencia
	o.completo = true
	o.mu.Unlock()
	return aislado.Observacion{ContrasteEstado: "igual", ArranqueEstado: "comprobado", EvidenciaSHA256: huellaCanonica(evidencia), SaludComprobada: true, ConsultaAutorizadaComprobada: true, DespachosBloqueados: true}, nil
}
func (o *ObservadorEnsemble) resultado() (p.Ensayo, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if !o.completo {
		return p.Ensayo{}, ErrEnsayoEnsemble
	}
	return p.Ensayo{Modo: o.Modo, Evidencia: o.evidencia, VerificadorVersion: "ensemble:v1", ArranqueRef: o.evidencia.ArranqueRef}, nil
}

// MedidorCS06 usa el mismo lector y canonicalización en la fuente y restaurados.
// Su ejecutor sólo existe dentro de la ventana CS04 de escritores excluidos.
type MedidorCS06 struct {
	Lector    *lector.Lector
	Ejecutor  lector.EjecutorPostgreSQL
	Exclusion lector.ExclusionObservada
	Base      string
	Ficheros  FicherosOrigen
}
type FicherosOrigen interface {
	MedirOrigen(context.Context, copias.Inventario) ([]copias.Artefacto, error)
}

func (m MedidorCS06) Medir(ctx context.Context, i copias.Inventario) (copias.Evidencia, error) {
	if m.Lector == nil || m.Ejecutor == nil || m.Exclusion == nil || m.Ficheros == nil || len(i.PostgreSQL.Bases) != 1 {
		return copias.Evidencia{}, ErrEvidencia
	}
	snapshot, err := m.Lector.CapturarEjecutor(ctx, m.Ejecutor, m.Base, m.Exclusion)
	if err != nil {
		return copias.Evidencia{}, ErrEvidencia
	}
	f, err := m.Ficheros.MedirOrigen(ctx, i)
	if err != nil {
		return copias.Evidencia{}, ErrEvidencia
	}
	return evidenciaSnapshot(snapshot, f, "origen:"+i.Ref, "")
}
