package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
)

// VEC_USUARIOS_CORREOS_ENABLED añade «Mis correos» (5.08b) a la composición
// de preferencias. Exige preferencias activas, SMTP configurado y AD3-107.
const envUsuariosCorreosDesarrollo = "VEC_USUARIOS_CORREOS_ENABLED"

// accionesCorreosUsuarios fija el orden de los doce proveedores: seis
// acciones por superficie, primero la interna y después la externa.
var accionesCorreosUsuarios = [6]struct{ accion, segmento string }{
	{usuariosports.AccionConsultarCorreos, "consultar"},
	{usuariosports.AccionAnadirCorreo, "anadir"},
	{usuariosports.AccionReenviarCorreo, "reenviar"},
	{usuariosports.AccionVerificarCorreo, "verificar"},
	{usuariosports.AccionActivarCorreo, "activar"},
	{usuariosports.AccionRetirarCorreo, "retirar"},
}

// audienciasCorreosUsuariosDesarrollo devuelve las doce audiencias nominales
// que AD3-107 añade a clave_capacidad_version.
func audienciasCorreosUsuariosDesarrollo() []string {
	return []string{
		usuariosports.AudienciaConsultarCorreosInterna, usuariosports.AudienciaAnadirCorreoInterna,
		usuariosports.AudienciaReenviarCorreoInterna, usuariosports.AudienciaVerificarCorreoInterna,
		usuariosports.AudienciaActivarCorreoInterna, usuariosports.AudienciaRetirarCorreoInterna,
		usuariosports.AudienciaConsultarCorreosExterna, usuariosports.AudienciaAnadirCorreoExterna,
		usuariosports.AudienciaReenviarCorreoExterna, usuariosports.AudienciaVerificarCorreoExterna,
		usuariosports.AudienciaActivarCorreoExterna, usuariosports.AudienciaRetirarCorreoExterna,
	}
}

// Cada efecto conserva su audiencia y clave HMAC derivada del gobierno V3
// común, igual que las cuatro de preferencias.
func descriptoresMaterialCorreosUsuariosDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	audiencias := audienciasCorreosUsuariosDesarrollo()
	d := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(audiencias))
	for i, audiencia := range audiencias {
		superficie := "interna"
		if i >= len(accionesCorreosUsuarios) {
			superficie = "externa"
		}
		segmento := accionesCorreosUsuarios[i%len(accionesCorreosUsuarios)].segmento
		d = append(d, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia:        audiencia,
			Dominio:          "vec.usuarios.correos." + segmento + "." + superficie + ".desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:usuarios-correos-" + segmento + "-" + superficie + ":",
			ProveedorNominal: "proveedor-material-usuarios-correos-" + segmento + "-" + superficie,
		})
	}
	return d
}

// seleccionCorreosUsuariosDesarrollo decide si se publican las doce
// audiencias. «Mis correos» exige preferencias, con cuya identidad y pools
// se compone, y comprueba su SQL antes de publicar ninguna clave.
func seleccionCorreosUsuariosDesarrollo(cfg config.Config, preferenciasActivas bool) (bool, []descriptorMaterialConsumidorV3Desarrollo, error) {
	activo, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosCorreosDesarrollo)
	if err != nil || !activo {
		return false, nil, err
	}
	if !preferenciasActivas {
		return false, nil, errComposicionUsuariosCorreos
	}
	if err := preflightSQLCorreosUsuariosDesarrollo(cfg); err != nil {
		return false, nil, err
	}
	descriptores := descriptoresMaterialCorreosUsuariosDesarrollo()
	avisos, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaAvisosMisCorreosDesarrollo)
	if err != nil {
		return false, nil, err
	}
	if avisos && !portalProcesoSeparado(cfg) {
		// B59 publica una audiencia más en el mismo lote; su SQL se comprueba
		// antes de publicar nada, igual que la de «Mis correos».
		if err := preflightSQLCorreoAvisosDesarrollo(cfg); err != nil {
			return false, nil, err
		}
		descriptores = append(descriptores, descriptorMaterialCorreoAvisosDesarrollo())
	}
	return true, descriptores, nil
}

// proveedoresMaterialCorreosUsuarios lleva los doce proveedores de «Mis
// correos» y, sólo si B59 está activo, el de la lectura para avisos.
type proveedoresMaterialCorreosUsuarios struct {
	lote   [12]*proveedorMaterialAltaContratacionTemporalDesarrollo
	avisos *proveedorMaterialAltaContratacionTemporalDesarrollo
}

// Las doce audiencias (trece con B59) forman un único corte: si una publicación falla, la
// transacción de gobierno revierte todas.
func publicarMaterialCorreosUsuariosEnLote(ctx context.Context, gobierno *pgxpool.Pool, base materialAtestacionContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo, catalogo catalogoMaterialAutorizacionComunDesarrollo,
) (proveedoresMaterialCorreosUsuarios, error) {
	vacios := proveedoresMaterialCorreosUsuarios{}
	if ctx == nil || gobierno == nil {
		return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	descriptores := descriptoresMaterialCorreosUsuariosDesarrollo()
	if len(descriptores) != len(vacios.lote) {
		return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	// La audiencia de avisos (B59) se publica si el catálogo la declara, es
	// decir, si la selección la pidió y su SQL pasó el preflight.
	if avisos, ok := catalogo.descriptorPara(usuariosports.AudienciaCorreoAvisosLlamamientoInterna); ok {
		if avisos != descriptorMaterialCorreoAvisosDesarrollo() {
			return vacios, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
		}
		descriptores = append(descriptores, avisos)
	}
	preparados := make([]materialAtestacionContratacionTemporalDesarrollo, len(descriptores))
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
	var resultado proveedoresMaterialCorreosUsuarios
	for i := range preparados {
		p, err := nuevoProveedorMaterialAutorizacionBaseDesarrollo(preparados[i], reloj)
		if err != nil {
			return vacios, err
		}
		if i < len(resultado.lote) {
			resultado.lote[i] = p
		} else {
			resultado.avisos = p
		}
	}
	return resultado, nil
}
