package bootstrap

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	usuariosimagen "vec-diputacion-granada/internal/modules/usuarios/adapters/imagen"
	usuariosseguridad "vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
	"vec-diputacion-granada/internal/shared/i18n"
)

// nuevasDependenciasAdicionalesPortalExterno carga exclusivamente las
// audiencias externas. Las claves de correos las entrega una fuente propia
// del proceso; nunca se leen del material interno ni del gobierno V3.
func nuevasDependenciasAdicionalesPortalExterno(ctx context.Context, cfg config.Config, preflight *pgxpool.Pool,
	fuentesCorreos ...usuariosseguridad.FuenteClavesCorreos,
) (*dependenciasCorreosUsuariosDesarrollo, *dependenciasImagenUsuariosDesarrollo, error) {
	activoCorreos, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosCorreosDesarrollo)
	if err != nil {
		return nil, nil, ErrUsuariosPortalExternoNoDisponible
	}
	activaImagen, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosImagenDesarrollo)
	if err != nil || len(fuentesCorreos) > 1 || (activoCorreos && (len(fuentesCorreos) != 1 || fuentesCorreos[0] == nil)) {
		return nil, nil, ErrUsuariosPortalExternoNoDisponible
	}
	if !activoCorreos && !activaImagen {
		return nil, nil, nil
	}
	if ctx == nil || preflight == nil {
		return nil, nil, ErrUsuariosPortalExternoNoDisponible
	}
	var correos *dependenciasCorreosUsuariosDesarrollo
	if activoCorreos {
		proveedores, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight, "usuarios_correos", relojContratacionTemporalDesarrollo{})
		if err != nil {
			return nil, nil, ErrUsuariosPortalExternoNoDisponible
		}
		materiales, ok := materialesCorreosPortalExterno(proveedores)
		if !ok {
			return nil, nil, ErrUsuariosPortalExternoNoDisponible
		}
		cripto, err := usuariosseguridad.NuevoAdaptadorCorreos(fuentesCorreos[0], time.Now)
		if err != nil {
			return nil, nil, ErrUsuariosPortalExternoNoDisponible
		}
		smtp, err := nuevoEnviadorCorreoLlamamientoDesarrollo(cfg)
		if err != nil || smtp == nil {
			return nil, nil, ErrUsuariosPortalExternoNoDisponible
		}
		catalogo, err := i18n.Load()
		if err != nil {
			return nil, nil, ErrUsuariosPortalExternoNoDisponible
		}
		transporte, err := nuevoTransporteCorreosPropiosDesarrollo(smtp, catalogo, cfg.SMTPFrom)
		if err != nil {
			return nil, nil, ErrUsuariosPortalExternoNoDisponible
		}
		correos = &dependenciasCorreosUsuariosDesarrollo{materiales: materiales, cripto: cripto, transporte: transporte}
	}
	var imagen *dependenciasImagenUsuariosDesarrollo
	if activaImagen {
		proveedores, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight, "usuarios_imagen", relojContratacionTemporalDesarrollo{})
		if err != nil {
			return nil, nil, ErrUsuariosPortalExternoNoDisponible
		}
		materiales, ok := materialesImagenPortalExterno(proveedores)
		if !ok {
			return nil, nil, ErrUsuariosPortalExternoNoDisponible
		}
		imagen = &dependenciasImagenUsuariosDesarrollo{materiales: materiales, transformador: usuariosimagen.Nuevo(decodificacionesFotoSimultaneas)}
	}
	return correos, imagen, nil
}

func materialesCorreosPortalExterno(proveedores map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo) (proveedoresMaterialCorreosUsuarios, bool) {
	var materiales proveedoresMaterialCorreosUsuarios
	audiencias := audienciasCorreosUsuariosDesarrollo()[len(accionesCorreosUsuarios):]
	if len(proveedores) != len(audiencias) {
		return materiales, false
	}
	for i, audiencia := range audiencias {
		p := proveedores[audiencia]
		if p == nil {
			return proveedoresMaterialCorreosUsuarios{}, false
		}
		indice := len(accionesCorreosUsuarios) + i
		if indice >= len(materiales.lote) {
			return proveedoresMaterialCorreosUsuarios{}, false
		}
		materiales.lote[indice] = p
	}
	return materiales, true
}

func materialesImagenPortalExterno(proveedores map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo) (proveedoresMaterialImagenUsuarios, bool) {
	var materiales proveedoresMaterialImagenUsuarios
	audiencias := audienciasImagenUsuariosDesarrollo()[len(accionesImagenUsuarios):]
	if len(proveedores) != len(audiencias) {
		return materiales, false
	}
	for i, audiencia := range audiencias {
		p := proveedores[audiencia]
		if p == nil {
			return proveedoresMaterialImagenUsuarios{}, false
		}
		indice := len(accionesImagenUsuarios) + i
		if indice >= len(materiales) {
			return proveedoresMaterialImagenUsuarios{}, false
		}
		materiales[indice] = p
	}
	return materiales, true
}
