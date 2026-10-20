package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
)

// seleccionMaterialUsuariosDesarrollo declara únicamente las audiencias que
// consumen Usuarios y Aspirantes. Su preflight usa el gobierno y la identidad
// que consumirán esas rutas; no selecciona perfiles ni acciones de CT.
type seleccionMaterialUsuariosDesarrollo struct {
	preferencias, correos, imagen, aspirantes bool
	catalogo                                  catalogoMaterialAutorizacionComunDesarrollo
	descriptores                              []descriptorMaterialConsumidorV3Desarrollo
	etapa                                     string
}

func seleccionarMaterialUsuariosDesarrollo(cfg config.Config, derivador *derivadorIdentidadOperacionDesarrollo,
	gobierno *pgxpool.Pool,
) (seleccionMaterialUsuariosDesarrollo, error) {
	vacia := seleccionMaterialUsuariosDesarrollo{}
	vacia.etapa = "selector_usuarios_preferencias"
	preferencias, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil {
		return vacia, err
	}
	var descriptores []descriptorMaterialConsumidorV3Desarrollo
	if preferencias {
		vacia.etapa = "preflight_sql_usuarios_preferencias"
		usuarios, err := descriptoresMaterialPreferenciasTrasPreflight(func() error {
			return preflightSQLPreferenciasUsuariosDesarrollo(cfg, derivador, gobierno)
		})
		if err != nil {
			return vacia, err
		}
		descriptores = append(descriptores, usuarios...)
	}
	vacia.etapa = "preflight_sql_usuarios_correos"
	correos, dsCorreos, err := seleccionCorreosUsuariosDesarrollo(cfg, preferencias)
	if err != nil {
		return vacia, err
	}
	descriptores = append(descriptores, dsCorreos...)
	vacia.etapa = "preflight_sql_usuarios_imagen"
	imagen, dsImagen, err := seleccionImagenUsuariosDesarrollo(cfg, preferencias)
	if err != nil {
		return vacia, err
	}
	descriptores = append(descriptores, dsImagen...)
	vacia.etapa = "preflight_sql_aspirantes"
	aspirantes, dsAspirantes, err := seleccionAspirantesDesarrollo(cfg, preferencias)
	if err != nil {
		return vacia, err
	}
	descriptores = append(descriptores, dsAspirantes...)
	vacia.etapa = "catalogo_material_usuarios"
	catalogo, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptores)
	if err != nil {
		return vacia, err
	}
	return seleccionMaterialUsuariosDesarrollo{preferencias: preferencias, correos: correos, imagen: imagen,
		aspirantes: aspirantes, catalogo: catalogo, descriptores: descriptores}, nil
}

type materialUsuariosDesarrollo struct {
	preferencias proveedoresMaterialPreferenciasUsuarios
	correos      proveedoresMaterialCorreosUsuarios
	imagen       proveedoresMaterialImagenUsuarios
	aspirantes   proveedoresMaterialAspirantes
	etapa        string
}

// publicarMaterialUsuariosDesarrollo publica cada audiencia nominal de las
// rutas comunes. La clave funcional de CT no pertenece a este catálogo.
func publicarMaterialUsuariosDesarrollo(ctx context.Context, gobierno *pgxpool.Pool,
	base materialAtestacionContratacionTemporalDesarrollo, reloj relojContratacionTemporalDesarrollo,
	seleccion seleccionMaterialUsuariosDesarrollo,
) (materialUsuariosDesarrollo, error) {
	vacio := materialUsuariosDesarrollo{}
	if seleccion.preferencias {
		vacio.etapa = "material_usuarios_preferencias"
		lote, err := publicarMaterialPreferenciasUsuariosEnLote(ctx, gobierno, base, reloj, seleccion.catalogo)
		if err != nil {
			return vacio, err
		}
		vacio.preferencias = lote
	}
	if seleccion.correos {
		vacio.etapa = "material_usuarios_correos"
		correos, err := publicarMaterialCorreosUsuariosEnLote(ctx, gobierno, base, reloj, seleccion.catalogo)
		if err != nil {
			return vacio, err
		}
		vacio.correos = correos
	}
	if seleccion.imagen {
		vacio.etapa = "material_usuarios_imagen"
		imagen, err := publicarMaterialImagenUsuariosEnLote(ctx, gobierno, base, reloj, seleccion.catalogo)
		if err != nil {
			return vacio, err
		}
		vacio.imagen = imagen
	}
	if seleccion.aspirantes {
		vacio.etapa = "material_aspirantes"
		aspirantes, err := publicarMaterialAspirantesEnLote(ctx, gobierno, base, reloj, seleccion.catalogo)
		if err != nil {
			return vacio, err
		}
		vacio.aspirantes = aspirantes
	}
	vacio.etapa = ""
	return vacio, nil
}
