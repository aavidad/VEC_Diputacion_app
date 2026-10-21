package bootstrap

import (
	"context"
	"errors"
	"os"
	"strings"

	pgbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	appbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	bp "vec-diputacion-granada/internal/modules/bolsa/ports"
	plantillashttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La raíz fija la fuente constituida antes de abrir HTTP. Nunca cambia durante
// las peticiones ni guarda identidad entre descargas.
type montajeIdentidadDocumentalCT struct {
	consultar          ct.FuenteIdentidadDocumentoRRHH
	selectorDocumental selectorConfiableIdentidadDocumentalCT
	configurar         func(*fuenteConstituidaRRHHDesarrollo)
	cerrar             func()
}

func (m *montajeIdentidadDocumentalCT) Consultar(ctx context.Context, expediente string) (*ct.IdentidadDocumentoRRHH, error) {
	if ctx == nil {
		return nil, ct.ErrBorradorRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if m == nil || m.consultar == nil {
		return nil, ct.ErrBorradorRRHHNoDisponible
	}
	return m.consultar(ctx, expediente)
}

func (m *montajeIdentidadDocumentalCT) Cerrar() {
	if m != nil && m.cerrar != nil {
		m.cerrar()
	}
}

func nuevoMontajeIdentidadDocumentalCT(ctx context.Context, alta *dependenciasAltaContratacionTemporalDesarrollo,
	consultas dependenciasConsultasRRHHDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (*montajeIdentidadDocumentalCT, error) {
	m := &montajeIdentidadDocumentalCT{}
	ruta := strings.TrimSpace(os.Getenv(envIdentidadDocumentalCTFile))
	if ruta == "" {
		return m, nil
	}
	if alta == nil || alta.soporte == nil || alta.postgresql.gobierno == nil ||
		alta.postgresql.registroAutorizacion == nil || consultas.identidad == nil || derivador == nil {
		return m, errIdentidadDocumentalCTNoDisponible
	}
	cfg, raiz, err := leerConfiguracionIdentidadDocumentalCT(ruta)
	if err != nil {
		return m, err
	}
	defer raiz.Close()
	pools, cerrar, err := abrirPoolsIdentidadDocumentalCT(ctx, raiz, cfg)
	if err != nil {
		return m, err
	}
	completo := false
	defer func() {
		if !completo {
			cerrar()
		}
	}()
	sesion, ok := consultas.sesion.(ct.SesionSelectorIdentidadDocumentalRRHH)
	if !ok {
		return m, errIdentidadDocumentalCTNoDisponible
	}
	emisorDetalle, err := nuevoEmisorMaterialRenovableCTDesarrollo(consultas.autoridad, consultas.materialDetalle)
	if err != nil {
		return m, err
	}
	emisor, err := ct.NuevoEmisorMaterialConsultaRRHH(consultas.motivos, seg.GeneradorReferenciasCriptograficas{},
		reloj, consultas.emisorCuadro, emisorDetalle)
	if err != nil {
		return m, err
	}
	selector, err := appct.NuevoServicioSelectorIdentidadDocumentalRRHH(consultas.autoridad, emisor, sesion, reloj)
	if err != nil {
		return m, err
	}
	selectorConfiable := func(ctx context.Context, expediente string) (bp.SelectorAceptacionNativaCT, error) {
		canal, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
		if ok && canal.ruta == plantillashttp.RutaBorradores {
			if m.selectorDocumental == nil {
				return bp.SelectorAceptacionNativaCT{}, ct.ErrAutorizacionDenegada
			}
			return m.selectorDocumental(ctx, expediente)
		}
		s, e := selector.Consultar(ctx, expediente)
		if e != nil {
			return bp.SelectorAceptacionNativaCT{}, e
		}
		return bp.SelectorAceptacionNativaCT(s), nil
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(derivador, reloj.Ahora())
	if err != nil {
		return m, err
	}
	defer material.borrarCopiasEfimeras()
	material.fuenteConfianza = alta.postgresql.proveedorMaterial.fuenteConfianza
	descriptor := descriptorMaterialConsumidorV3Desarrollo{
		Audiencia: bp.AudienciaIdentidadDocumentoAceptacionCT,
		Dominio:   "vec.ct.identidad-documental.capacidad-v3", Prefijo: "clave:capacidad:ct-identidad-documental:",
		ProveedorNominal: "proveedor-material-ct-identidad-documental",
	}
	proveedor, err := nuevoProveedorMaterialConsumidorConDescriptorDesarrollo(ctx, alta.postgresql.gobierno,
		material, alta.soporte, reloj, descriptor)
	if err != nil {
		return m, err
	}
	lector := func(ctx context.Context, perfil string) (instantaneaPublicadaDesarrollo, bool, error) {
		return leerInstantaneaPublicadaPostgreSQLDesarrollo(ctx, alta.postgresql.gobierno, perfil)
	}
	autoridad, err := nuevoProveedorIdentidadDocumentalCT(cfg, alta.soporte, consultas.identidad,
		lector, selectorConfiable, pools, alta.postgresql.registroAutorizacion, proveedor, reloj)
	if err != nil {
		return m, err
	}
	permiso, err := appbolsa.NuevoProveedorNominalIdentidadDocumentoAceptacionCT(autoridad, cfg.Motivo, vecports.ReferenciaCorrelacionAutorizacionV2DePeticion, reloj.Ahora)
	if err != nil {
		return m, err
	}
	repo, err := pgbolsa.NuevoRepositorioIdentidadDocumentoAceptacionCTPostgreSQL(pools["bolsa"], reloj.Ahora)
	if err != nil {
		return m, err
	}
	m.configurar = func(f *fuenteConstituidaRRHHDesarrollo) {
		if f == nil || f.recuperador == nil {
			return
		}
		servicio, e := appbolsa.NuevoServicioIdentidadDocumentoAceptacionCT(permiso, repo, f.recuperador, reloj.Ahora)
		if e != nil {
			registrarFalloMontajeCTDesarrollo("identidad_documental_fuente", e)
			return
		}
		m.consultar = func(ctx context.Context, expediente string) (*ct.IdentidadDocumentoRRHH, error) {
			p, e := autoridad.Preparar(ctx, expediente)
			if e != nil {
				return nil, e
			}
			r, e := servicio.ConsultarIdentidadDocumentoAceptacionCT(p.contexto,
				bp.SolicitudIdentidadDocumentoAceptacionCT{Selector: p.selector, ActorConfiable: p.actor})
			if e != nil {
				return nil, errors.Join(ct.ErrBorradorRRHHNoDisponible, e)
			}
			if r.Identidad == nil {
				return nil, ct.ErrBorradorRRHHNoDisponible
			}
			i := r.Identidad
			return &ct.IdentidadDocumentoRRHH{Nombre: i.Nombre, Apellidos: i.Apellidos,
				DocumentoEnmascarado: i.DocumentoEnmascarado, VersionFuente: i.VersionFuente}, nil
		}
	}
	m.cerrar = cerrar
	completo = true
	return m, nil
}
