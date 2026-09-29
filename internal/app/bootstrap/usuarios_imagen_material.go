package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
)

// VEC_USUARIOS_IMAGEN_ENABLED añade «Mi imagen» (5.08c) a la composición de
// preferencias. Exige preferencias activas y AD3-108, Documentos 000007 y
// Usuarios 000006/000007 instalados.
const envUsuariosImagenDesarrollo = "VEC_USUARIOS_IMAGEN_ENABLED"

// accionesImagenUsuarios fija el orden de los cuatro proveedores: dos
// acciones por superficie, primero la interna y después la externa.
var accionesImagenUsuarios = [2]struct{ accion, segmento string }{
	{usuariosports.AccionConsultarImagen, "consultar"},
	{usuariosports.AccionActualizarImagen, "actualizar"},
}

func audienciasImagenUsuariosDesarrollo() []string {
	return []string{
		usuariosports.AudienciaConsultarImagenInterna, usuariosports.AudienciaActualizarImagenInterna,
		usuariosports.AudienciaConsultarImagenExterna, usuariosports.AudienciaActualizarImagenExterna,
	}
}

func descriptoresMaterialImagenUsuariosDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	audiencias := audienciasImagenUsuariosDesarrollo()
	d := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(audiencias))
	for i, audiencia := range audiencias {
		superficie := "interna"
		if i >= len(accionesImagenUsuarios) {
			superficie = "externa"
		}
		segmento := accionesImagenUsuarios[i%len(accionesImagenUsuarios)].segmento
		d = append(d, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia:        audiencia,
			Dominio:          "vec.usuarios.imagen." + segmento + "." + superficie + ".desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:usuarios-imagen-" + segmento + "-" + superficie + ":",
			ProveedorNominal: "proveedor-material-usuarios-imagen-" + segmento + "-" + superficie,
		})
	}
	return d
}

// seleccionImagenUsuariosDesarrollo decide si se publican las cuatro
// audiencias. Comprueba su SQL con los LOGIN de cada superficie antes.
func seleccionImagenUsuariosDesarrollo(cfg config.Config, preferenciasActivas bool) (bool, []descriptorMaterialConsumidorV3Desarrollo, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosImagenDesarrollo)
	if err != nil || !activo {
		return false, nil, err
	}
	if !preferenciasActivas {
		return false, nil, errComposicionUsuariosImagen
	}
	if err := preflightSQLImagenUsuariosDesarrollo(cfg); err != nil {
		return false, nil, err
	}
	return true, descriptoresMaterialImagenUsuariosDesarrollo(), nil
}

type proveedoresMaterialImagenUsuarios [4]*proveedorMaterialAltaContratacionTemporalDesarrollo

// Las cuatro audiencias forman un único corte: si una publicación falla, la
// transacción de gobierno revierte todas.
func publicarMaterialImagenUsuariosEnLote(ctx context.Context, gobierno *pgxpool.Pool, base materialAtestacionContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo,
) (proveedoresMaterialImagenUsuarios, error) {
	vacios := proveedoresMaterialImagenUsuarios{}
	if ctx == nil || gobierno == nil {
		return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	descriptores := descriptoresMaterialImagenUsuariosDesarrollo()
	if len(descriptores) != len(vacios) {
		return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	var preparados [4]materialAtestacionContratacionTemporalDesarrollo
	defer func() {
		for i := range preparados {
			borrarBytes(preparados[i].claveHMAC)
		}
	}()
	for i, d := range descriptores {
		declarado, ok := catalogo.descriptorPara(d.Audiencia)
		if !ok || declarado != d {
			return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
		}
		derivado, err := derivarMaterialConsumidorV3Desarrollo(base, d)
		if err != nil {
			return vacios, err
		}
		preparados[i] = derivado
	}
	err := ejecutarTransaccionGobiernoCTDesarrollo(ctx, gobierno, func(tx pgx.Tx) error {
		for i := range preparados {
			if err := publicarGobiernoAtestacionCTEnTxDesarrollo(ctx, tx, &preparados[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return vacios, err
	}
	var resultado proveedoresMaterialImagenUsuarios
	for i := range preparados {
		p, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(preparados[i], reloj)
		if err != nil {
			return vacios, err
		}
		resultado[i] = p
	}
	return resultado, nil
}
