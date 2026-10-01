package contrastecopias

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

// RuntimePlantillaFisica pertenece al ejecutor de plataforma. Su constructor
// autentica el conjunto y verifica que el runtime aislado procede del componente
// físico, operación y ventana observados. Las dos operaciones técnicas son el
// único canal de escritura; no habilitan SQL libre ni modifican la base origen.
type RuntimePlantillaFisica interface {
	EjecutorPostgreSQL
	ExclusionObservada
	ObservarProcedenciaFisica(context.Context) (ProcedenciaPlantillaFisica, error)
	// Si CREATE confirma o puede haber confirmado y se pierde su respuesta, el
	// runtime concilia su registro de propiedad y retira solo su clon antes de
	// devolver error. Un nombre preexistente nunca se considera creado aquí.
	CrearClonPlantilla(context.Context, string, string) error
	RetirarClonPlantilla(context.Context, string) error
}

// Las dos huellas del DTO son hex SHA256 minúsculo sin prefijo sha256:.
type ProcedenciaPlantillaFisica struct {
	OperacionRef, ConjuntoRef, ArtefactoFisicoSHA256, ImagenSHA256, SelloExclusion string
}
type ConfiguracionPlantillaFisica struct {
	NombreClon, BaseControl string
	Lector                  Configuracion
}
type FuentePlantillaFisica struct {
	config  ConfiguracionPlantillaFisica
	runtime RuntimePlantillaFisica
	gate    chan struct{}
}

var errPlantilla = errors.New("evidencia_plantilla_fisica_no_disponible")

func NuevoFuentePlantillaFisica(config ConfiguracionPlantillaFisica, runtime RuntimePlantillaFisica) (*FuentePlantillaFisica, error) {
	if runtime == nil || !baseAdmitida.MatchString(config.NombreClon) || !baseAdmitida.MatchString(config.BaseControl) || config.NombreClon == config.BaseControl || !contieneBase(config.Lector.BasesInventariadas, config.BaseControl) || contieneBase(config.Lector.BasesInventariadas, config.NombreClon) || config.Lector.DSN != "" {
		return nil, errPlantilla
	}
	lector, e := Nuevo(config.Lector)
	if e != nil {
		return nil, errPlantilla
	}
	config.Lector = lector.limites
	return &FuentePlantillaFisica{config: config, runtime: runtime, gate: make(chan struct{}, 1)}, nil
}
func procedenciaAdmitida(p ProcedenciaPlantillaFisica, sello string) bool {
	return p.OperacionRef != "" && len(p.OperacionRef) <= 256 && p.ConjuntoRef != "" && len(p.ConjuntoRef) <= 256 && selloValido.MatchString(p.ArtefactoFisicoSHA256) && selloValido.MatchString(p.ImagenSHA256) && p.SelloExclusion == sello
}
func basesIguales(a, b []baseObservada) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// CapturarBase solo opera sobre la copia física aislada del mismo punto. El hash
// físico es procedencia: la igualdad del contenido siempre se obtiene por SQL.
// El clon técnico nunca es parte del inventario final y no se retira si existía.
func (f *FuentePlantillaFisica) CapturarBase(ctx context.Context, req SolicitudBaseNoConectable) (out EvidenciaBaseNoConectable, err error) {
	ctx, cancel := context.WithTimeout(ctx, f.config.Lector.TiempoMaximo)
	defer cancel()
	select {
	case f.gate <- struct{}{}:
		defer func() { <-f.gate }()
	case <-ctx.Done():
		return out, errPlantilla
	}
	if !contieneBase(f.config.Lector.BasesInventariadas, req.Nombre) || req.Nombre == f.config.NombreClon || req.VersionPostgreSQL != f.config.Lector.VersionPostgreSQL || !selloValido.MatchString(req.PropiedadesSHA256) || !selloValido.MatchString(req.SelloExclusion) || req.MaxFilas < 1 || req.MaxBytes < 1 || req.MaxObjetos < 8 {
		return out, errPlantilla
	}
	seal, e := observarSello(ctx, f.runtime, req.SelloExclusion)
	if e != nil {
		return out, errPlantilla
	}
	provenance, e := f.runtime.ObservarProcedenciaFisica(ctx)
	if e != nil || !procedenciaAdmitida(provenance, seal) {
		return out, errPlantilla
	}
	cfg := f.config.Lector
	if req.MaxFilas < cfg.MaxFilas {
		cfg.MaxFilas = req.MaxFilas
	}
	if req.MaxBytes < cfg.MaxBytes {
		cfg.MaxBytes = req.MaxBytes
	}
	if req.MaxObjetos < cfg.MaxObjetos {
		cfg.MaxObjetos = req.MaxObjetos
	}
	full := &Lector{limites: cfg}
	pool := &presupuestoCaptura{objetos: 8}
	control := &transporteEjecutor{exec: f.runtime, base: f.config.BaseControl, maxBytes: cfg.MaxBytes, timeout: cfg.TiempoMaximo.Milliseconds()}
	before, e := full.leerBases(ctx, control, pool)
	if e != nil || !full.coincideAmbito(before) {
		return out, errPlantilla
	}
	found := false
	for _, b := range before {
		if b.Nombre == f.config.NombreClon {
			return out, errPlantilla
		}
		if b.Nombre == req.Nombre {
			found = !b.Conectable && shaBytes(b.Propiedades) == req.PropiedadesSHA256
		}
	}
	if !found {
		return out, errPlantilla
	}
	if e = f.runtime.CrearClonPlantilla(ctx, req.Nombre, f.config.NombreClon); e != nil {
		return out, errPlantilla
	}
	created := true
	// La limpieza tiene plazo independiente del contexto agotado. Solo puede
	// retirar el nombre creado y confirmado en esta invocación del canal propio.
	cleanup := func() error {
		if !created {
			return nil
		}
		timeout := f.config.Lector.TiempoMaximo
		if timeout > 5*time.Second {
			timeout = 5 * time.Second
		}
		cleanCtx, stop := context.WithTimeout(context.Background(), timeout)
		defer stop()
		if e := f.runtime.RetirarClonPlantilla(cleanCtx, f.config.NombreClon); e != nil {
			return errPlantilla
		}
		created = false
		return nil
	}
	defer func() {
		if e := cleanup(); e != nil {
			out = EvidenciaBaseNoConectable{}
			err = errPlantilla
		}
	}()
	afterCreate, e := full.leerBases(ctx, control, pool)
	if e != nil {
		return out, errPlantilla
	}
	original := make([]baseObservada, 0, len(afterCreate))
	own := false
	for _, b := range afterCreate {
		if b.Nombre == f.config.NombreClon {
			own = b.Conectable
			continue
		}
		original = append(original, b)
	}
	if !own || !basesIguales(before, original) {
		return out, errPlantilla
	}
	if _, e = observarSello(ctx, f.runtime, seal); e != nil {
		return out, errPlantilla
	}
	childCfg := cfg
	childCfg.BasesInventariadas = nil
	childCfg.ReferenciasObjetosGrandes = nil
	for _, r := range cfg.ReferenciasObjetosGrandes {
		if r.Base == req.Nombre {
			r.Base = ""
			childCfg.ReferenciasObjetosGrandes = append(childCfg.ReferenciasObjetosGrandes, r)
		}
	}
	child := &Lector{limites: childCfg}
	clone := &transporteEjecutor{exec: f.runtime, base: f.config.NombreClon, maxBytes: cfg.MaxBytes, timeout: cfg.TiempoMaximo.Milliseconds()}
	snapshot, e := child.capturarSQLAcotada(ctx, clone, f.runtime, true, pool, req.Nombre)
	if e != nil || !snapshot.Completo || len(domain.Validar(snapshot)) != 0 {
		return out, errPlantilla
	}
	if e = cleanup(); e != nil {
		return out, errPlantilla
	}
	after, e := full.leerBases(ctx, control, pool)
	if e != nil || !basesIguales(before, after) {
		return out, errPlantilla
	}
	if _, e = observarSello(ctx, f.runtime, seal); e != nil {
		return out, errPlantilla
	}
	final, e := f.runtime.ObservarProcedenciaFisica(ctx)
	if e != nil || final != provenance {
		return out, errPlantilla
	}
	encoded, e := json.Marshal(snapshot)
	if e != nil {
		return out, errPlantilla
	}
	// El recibo sella el material físico autenticado, su ventana y la observación
	// lógica real. No compara páginas de PostgreSQL ni declara una fábrica vacía.
	receipt, e := json.Marshal(struct {
		Procedencia                 ProcedenciaPlantillaFisica
		Base, Propiedades, Snapshot string
	}{provenance, req.Nombre, req.PropiedadesSHA256, shaBytes(encoded)})
	if e != nil {
		return out, errPlantilla
	}
	if pool.bytes < int64(len(encoded)) || pool.bytes > req.MaxBytes || pool.filas < 1 || pool.filas > req.MaxFilas {
		return out, errPlantilla
	}
	out = EvidenciaBaseNoConectable{Snapshot: snapshot, Nombre: req.Nombre, PropiedadesSHA256: req.PropiedadesSHA256, SelloExclusion: seal, InicializacionSHA256: shaBytes(receipt), SnapshotSHA256: shaBytes(encoded), FilasLeidas: pool.filas, BytesLeidos: pool.bytes}
	return out, nil
}
