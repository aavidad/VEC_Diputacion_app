package ejecucioncopias

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	cs04 "vec-diputacion-granada/internal/modules/administracion/application/capturacopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	capturap "vec-diputacion-granada/internal/modules/administracion/ports/capturacopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
)

var ErrCapturaEnsemble = errors.New("copias_ejecucion_captura_no_comprobable")

// MedidorOrigen observa el contenido y los objetos persistentes de la fuente.
// Se invoca dentro de la exclusión CS04, antes de generar el volcado lógico.
// La composición debe usar el lector CS06; no admite una evidencia de JSON.
type MedidorOrigen interface {
	Medir(context.Context, copias.Inventario) (copias.Evidencia, error)
}

// MaterialCapturado traduce los archivos privados producidos por CS04/05 al
// manifiesto: conserva los tar físicos y demuestra sus artefactos originales.
// Registrar sólo admite archivos de la captura ya cerrada, nunca la fuente viva.
type MaterialCapturado interface {
	Registrar(context.Context, string, []copias.Artefacto, copias.Inventario) ([]copias.Artefacto, error)
}

type CapturaCS04 struct {
	Servicio         cs04.Servicio
	Medidor          MedidorOrigen
	Material         MaterialCapturado
	DuracionVentana  time.Duration
	Proteccion       copias.Proteccion
	InventarioActual p.Inventario
	Mantenimiento    MantenimientoExterno
	mu               sync.Mutex
}

type logicoMedido struct {
	captura capturap.CapturadorLogico
	medidor MedidorOrigen
	origen  copias.Evidencia
}

func (l *logicoMedido) Capturar(ctx context.Context, i copias.Inventario) ([]copias.Artefacto, error) {
	e, err := l.medidor.Medir(ctx, i)
	if err != nil {
		return nil, ErrCapturaEnsemble
	}
	l.origen = e
	return l.captura.Capturar(ctx, i)
}

func (v *CapturaCS04) Capturar(ctx context.Context, solicitud p.Peticion, lectura p.Lectura) (p.Captura, error) {
	if ctx == nil || v == nil || v.Medidor == nil || v.Material == nil || v.DuracionVentana <= 0 || v.Servicio.Logico == nil || v.Servicio.Componentes == nil || v.Servicio.Ahora == nil {
		return p.Captura{}, ErrCapturaEnsemble
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	servicio := v.Servicio
	logico := &logicoMedido{captura: servicio.Logico, medidor: v.Medidor}
	servicio.Logico = logico
	inicio := servicio.Ahora().UTC()
	parcial, err := servicio.Capturar(ctx, capturap.Peticion{OrigenRef: lectura.Observado.PostgreSQL.ClusterRef, OperacionRef: solicitud.OperacionRef, Esperado: lectura.Esperado, InicioVentana: inicio, FinVentana: inicio.Add(v.DuracionVentana)})
	if err != nil || parcial.Estado != "captura_parcial_pendiente" || parcial.InventarioSHA256 != copias.HuellaInventario(lectura.Observado) {
		return p.Captura{}, ErrCapturaEnsemble
	}
	return v.convertir(ctx, solicitud, lectura, parcial, logico.origen)
}

func (v *CapturaCS04) convertir(ctx context.Context, solicitud p.Peticion, lectura p.Lectura, parcial capturap.Parcial, origen copias.Evidencia) (p.Captura, error) {
	componentes, err := v.Material.Registrar(ctx, solicitud.ConjuntoRef, parcial.Componentes, lectura.Observado)
	if err != nil {
		return p.Captura{}, ErrCapturaEnsemble
	}
	var total int64
	for _, a := range componentes {
		if a.TamanoBytes < 0 || total > (1<<63-1)-a.TamanoBytes {
			return p.Captura{}, ErrCapturaEnsemble
		}
		total += a.TamanoBytes
	}
	manifiesto := copias.Manifiesto{FormatoVersion: copias.FormatoVersion, ConjuntoRef: solicitud.ConjuntoRef, OperacionRef: solicitud.OperacionRef, SolicitanteRef: solicitud.ActorRef, MotivoRef: solicitud.MotivoRef, PoliticaRef: lectura.Politica.Ref, Inicio: parcial.Inicio, Fin: parcial.Fin, TamanoBytes: total, Inventario: lectura.Observado, Componentes: componentes, InventarioSHA256: parcial.InventarioSHA256, Consistencia: copias.Consistencia{Modo: "fisica_fria_y_logica", EscritoresExcluidos: true, ParadaLimpia: true, EvidenciaRef: "ventana:" + solicitud.OperacionRef, PuntoRecuperacionRef: lectura.VersionRef}, Proteccion: v.Proteccion, Verificacion: copias.Verificacion{Estado: "pendiente_verificacion"}}
	manifiesto.Proteccion.CifradoSHA256 = strings.Repeat("0", 64)
	// Este puente interpreta la terminación CS04/05, sin conceder validez CS06.
	return p.Captura{Manifiesto: manifiesto, Origen: origen}, nil
}
