package interna

import (
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	"vec-diputacion-granada/internal/app/composicion/internagobierno"
	inc "vec-diputacion-granada/internal/app/incorporacionejercicio"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/adapters/seudonimizacionpkcs11"
)

type relojGobiernoInterno struct{}

func (relojGobiernoInterno) Ahora() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

// El recurso posee solo las conexiones que no pertenecen a los once pools CT.
// Su cierre es único incluso si el servidor falla antes de escuchar.
type recursoGobiernoInterno struct {
	pools       PoolsIdentidadInterna
	hmac        *seudonimizacionpkcs11.Conector
	ct          *internactproveedores.Proveedores
	personalB2  interface{ Cerrar() }
	propiedad   atomic.Bool
	unaVez      sync.Once
	errorCierre error
}

func (r *recursoGobiernoInterno) reclamarPropiedad() bool {
	return r != nil && r.propiedad.CompareAndSwap(false, true)
}

func (r *recursoGobiernoInterno) cerrar() error {
	if r == nil || !r.propiedad.Load() {
		return ErrDependenciasProductivasNoDisponibles
	}
	return r.cerrarSinPropiedad()
}

func (r *recursoGobiernoInterno) cerrarSinPropiedad() error {
	if r == nil {
		return nil
	}
	r.unaVez.Do(func() {
		if r.personalB2 != nil {
			r.personalB2.Cerrar()
		}
		if r.ct != nil {
			r.ct.Cerrar()
		}
		if r.hmac != nil {
			if r.hmac.Cerrar() != nil {
				r.errorCierre = ErrDependenciasProductivasNoDisponibles
			}
		}
		r.pools.Cerrar()
	})
	return r.errorCierre
}

// cargarProveedoresGobernados sólo conecta autoridades ya publicadas. No
// instala SQL, no crea persona/cuenta/perfil ni genera claves al arrancar.
func cargarProveedoresGobernados(ctx context.Context, cfg Configuracion) (proveedoresConsultaSeguimiento, error) {
	var vacio proveedoresConsultaSeguimiento
	if ctx == nil || ctx.Err() != nil || cfg.Validar() != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	directorio := os.Getenv(EnvMaterialInterno)
	if directorio == "" {
		return vacio, &ErrorDependenciasFaltantes{
			faltantes: append([]Dependencia(nil), dependenciasConsultaSeguimiento[:]...),
		}
	}
	if cfg.RetiradaPoliticaInternaEn.IsZero() {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	materialCT, err := cargarMaterialPoolsSeguimiento(directorio)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	configuracionV2, err := AbrirPoolsSeguimiento(ctx, materialCT)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	exito := false
	recursos := &recursoGobiernoInterno{}
	defer func() {
		if !exito {
			_ = recursos.cerrarSinPropiedad()
			cerrarPoolsSeguimiento(poolsConsultaSeguimiento(configuracionV2))
		}
	}()
	recursos.pools, err = abrirPoolsIdentidadInterna(ctx, directorio, materialCT)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	hmacConfig, err := cargarConfiguracionHMACIdentidad(directorio)
	if err != nil || hmacConfig.EspacioIdentidad != cfg.EmisorIdentidad {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	recursos.hmac, err = seudonimizacionpkcs11.Abrir(hmacConfig)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	identidadPG, err := internagobierno.NuevoIdentidadPostgreSQL(ctx,
		recursos.pools.Registro, recursos.pools.Revalidacion,
		recursos.hmac, hmacConfig.EspacioIdentidad, hmacConfig.DominioRef)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	materialIdentidad, err := cargarMaterialIdentidadCertificado(directorio)
	if err != nil || preflightPoliticaCertificado(ctx, recursos.pools.Revalidacion,
		materialIdentidad, cfg.RetiradaPoliticaInternaEn) != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	identidad, extractor, err := montarServicioIdentidadCertificado(
		cfg, materialIdentidad.claveID, materialIdentidad.firmante,
		materialIdentidad.rutaCertificados,
		materialIdentidad.politicaRef, materialIdentidad.huellaPolitica,
		identidadPG.Registro,
	)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	reloj := relojGobiernoInterno{}
	contextoPG, err := internagobierno.NuevoContextoPostgreSQL(ctx,
		recursos.pools.Revalidacion, recursos.pools.Contexto, reloj)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	contextos, err := cargarContextosNominalesIdentidad(directorio)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	materialV3, err := internactproveedores.CargarMaterial(directorio)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	defer materialV3.Cerrar()
	politica := inc.PoliticaConsultaDesarrollo{
		Tipo:         httpseguridad.PoliticaInternaDesarrolloCertificadoPersonal,
		Referencia:   materialIdentidad.politicaRef,
		HuellaSHA256: strings.TrimPrefix(materialIdentidad.huellaPolitica, "sha256:"),
		RetiradaEn:   cfg.RetiradaPoliticaInternaEn,
	}
	fuenteF1, err := internagobierno.NuevaFuenteF1(internagobierno.ConfiguracionFuenteF1{
		Identidad: identidad, Revalidador: identidadPG.Revalidador,
		Resolutor: contextoPG.Resolutor, Reloj: reloj,
		PorCuenta: contextos, MotivoAlta: materialV3.MotivoAlta,
		MotivoLectura: materialV3.MotivoLectura, Politica: politica,
	})
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	recursos.ct = &internactproveedores.Proveedores{}
	*recursos.ct, err = internactproveedores.Construir(ctx,
		internactproveedores.Configuracion{Material: materialV3, Fuente: fuenteF1, Reloj: reloj})
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	autoridadRuta, err := internagobierno.NuevaAutoridadRutaSeguimiento(fuenteF1)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	auditoria, err := internagobierno.NuevoRegistradorAuditoriaPostgreSQL(ctx, recursos.pools.Auditoria)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	configuracionV2.FuenteAutoridad = fuenteF1
	configuracionV2.Revalidador = identidadPG.Revalidador
	configuracionV2.Resolutor = contextoPG.Resolutor
	configuracionV2.Cadena = recursos.ct.Cadena
	configuracionV2.Correlador = seguridad.GeneradorReferenciasCriptograficas{}
	configuracionV2.Preparacion.Detalle = recursos.ct.Detalle
	configuracionV2.Preparacion.Planes = recursos.ct.Planes
	configuracionV2.Preparacion.TernaPlanes = recursos.ct.TernaPlanes
	configuracionV2.Preparacion.FuentePersonal = recursos.ct.FuentePersonal
	configuracionV2.Preparacion.TernaPersonal = recursos.ct.TernaPersonal
	configuracionV2.Preparacion.Reloj = reloj
	configuracionV2.PoliticaConsultaDesarrollo = &politica
	if err := acreditarPoolPersonalB2(ctx, configuracionV2.AltaPersonal, materialCT.AltaPersonal.Login); err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	materialPersonalB2, err := internactproveedores.CargarMaterialPersonalB2(directorio)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	defer materialPersonalB2.Cerrar()
	proveedorPersonalB2, err := internactproveedores.ConstruirPersonalB2(ctx,
		internactproveedores.ConfiguracionPersonalB2{
			Material: materialPersonalB2, Base: recursos.ct, Fuente: fuenteF1, Reloj: reloj,
		})
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	recursos.personalB2 = proveedorPersonalB2
	repositorioPersonalB2, err := personalpg.NuevoRepositorioRegistroEmpleadoB2PostgreSQL(configuracionV2.AltaPersonal)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	consultaPersonalB2, err := personalapp.NuevoServicioRegistroEmpleadoB2(proveedorPersonalB2, repositorioPersonalB2)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	actosPersonalB2, err := personalapp.NuevoServicioActosRegistroEmpleadoB2(proveedorPersonalB2, repositorioPersonalB2)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	autoridadPersonalB2 := internactproveedores.AutoridadContextoRegistroEmpleadoB2{Fuente: fuenteF1}
	auditorPersonalB2 := internactproveedores.AuditorDenegacionRegistroEmpleadoB2{Registrador: auditoria}
	fichaPersonalB2, err := httpapi.NewHandlerFichaEmpleadoB2(autoridadPersonalB2, consultaPersonalB2, auditorPersonalB2)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	vacantesPersonalB2, err := httpapi.NewHandlerVacantesEmpleadoB2(autoridadPersonalB2, consultaPersonalB2, auditorPersonalB2)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	altaPersonalB2, err := httpapi.NewHandlerAltaEmpleadoB2(autoridadPersonalB2, actosPersonalB2, auditorPersonalB2)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	hechoPersonalB2, err := httpapi.NewHandlerHechosEmpleadoB2(autoridadPersonalB2, actosPersonalB2, auditorPersonalB2)
	if err != nil {
		return vacio, ErrDependenciasProductivasNoDisponibles
	}
	salida := proveedoresConsultaSeguimiento{
		identidad: identidad, extractor: extractor,
		autoridadRutas: autoridadRuta, auditoriaRutas: auditoria,
		vincularPersonalB2: fuenteF1.VincularContextoPersonalB2,
		fichaPersonalB2:    fichaPersonalB2, vacantesPersonalB2: vacantesPersonalB2,
		altaPersonalB2: altaPersonalB2, hechoPersonalB2: hechoPersonalB2,
		configuracionV2: configuracionV2,
		recursos:        []recursoCerrableAplicacionInterna{recursos},
	}
	exito = true
	return salida, nil
}
