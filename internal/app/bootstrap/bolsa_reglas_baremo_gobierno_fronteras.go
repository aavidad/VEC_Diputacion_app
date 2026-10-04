package bootstrap

import (
	"context"
	"net/http"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
)

const clavePoliticaGobiernoReglasBaremoHTTPV3 = "bolsa-gobierno-reglas-baremo-v3"

func paresGobiernoReglasBaremoHTTPV3() []struct{ ruta, accion, frontera string } {
	return []struct{ ruta, accion, frontera string }{
		{bolsahttp.RutaAltaGobiernoReglasBaremoV3, "bolsa.reglas_baremo.borrador.crear", "bolsa-reglas-baremo-alta-v3"},
		{bolsahttp.RutaConsultaGobiernoReglasBaremoV3, "bolsa.reglas_baremo.version.consultar", "bolsa-reglas-baremo-consulta-v3"},
		{bolsahttp.RutaRecuperarGobiernoReglasBaremoV3, "bolsa.reglas_baremo.recibo.consultar", "bolsa-reglas-baremo-recuperar-v3"},
	}
}

func fronterasGobiernoReglasBaremoHTTPV3(perfil string) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfil) {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	var salida []descriptorFronteraComunDesarrollo
	for _, p := range paresGobiernoReglasBaremoHTTPV3() {
		salida = append(salida, descriptorFronteraComunDesarrollo{Clave: p.frontera,
			Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost,
			Ruta: p.ruta, PerfilesActivosRef: []string{perfil},
			ClavePolitica: clavePoliticaGobiernoReglasBaremoHTTPV3, ClaveCapacidad: p.accion})
	}
	return salida, nil
}

func autorizacionesGobiernoReglasBaremoHTTPV3(politica politicaAutorizacionSolicitudLigadaV3Desarrollo) ([]descriptorAutorizacionComunDesarrollo, error) {
	if !politica.valida() {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	var salida []descriptorAutorizacionComunDesarrollo
	for _, p := range paresGobiernoReglasBaremoHTTPV3() {
		salida = append(salida, descriptorAutorizacionComunDesarrollo{Accion: p.accion,
			ClavePolitica: clavePoliticaGobiernoReglasBaremoHTTPV3, ClaveCapacidad: p.accion,
			Fronteras: []string{p.frontera}, Politica: politica})
	}
	return salida, nil
}

// Sólo clasifica el error de esta sesión; no concede ni reconstruye autoridad.
// La misma instancia de catálogo y la capacidad emitida por la raíz son
// obligatorias, también al propagar la cancelación de una petición.
func (p *proveedorSesionConsultaRRHHDesarrollo) sesionGobiernoReglasBaremoHTTPV3(ctx context.Context, ruta string) bool {
	if p == nil || p.soporte == nil || ctx == nil || p.fronteras.identidad == nil {
		return false
	}
	capacidad, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	f, existe := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || !existe || capacidad.sello != p.soporte.sello || capacidad.ruta != ruta ||
		capacidad.metodo != http.MethodPost || f.metodo != http.MethodPost || f.ruta != ruta ||
		f.superficie != superficieInternaSeguridadComunDesarrollo || !p.fronteras.mismaInstancia(f.catalogo) ||
		capacidad.principal.ID != p.soporte.principalID ||
		capacidad.principal.Attributes["certificate_sha256"] != p.soporte.certificadoSHA256 {
		return false
	}
	for _, par := range paresGobiernoReglasBaremoHTTPV3() {
		if ruta != par.ruta {
			continue
		}
		d, declarada := p.fronteras.resolver(http.MethodPost, ruta)
		return declarada && d.Clave == par.frontera && f.descriptor.Clave == d.Clave &&
			d.ClavePolitica == clavePoliticaGobiernoReglasBaremoHTTPV3 && d.ClaveCapacidad == par.accion &&
			f.descriptor.ClavePolitica == d.ClavePolitica && f.descriptor.ClaveCapacidad == d.ClaveCapacidad &&
			len(d.PerfilesActivosRef) == 1 && d.PerfilesActivosRef[0] == p.base.Contexto.PerfilActivoRef &&
			f.descriptor.admitePerfil(p.base.Contexto.PerfilActivoRef)
	}
	return false
}
