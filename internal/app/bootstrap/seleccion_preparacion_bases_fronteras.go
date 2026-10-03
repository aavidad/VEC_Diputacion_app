package bootstrap

import (
	"context"
	"errors"
	"net/http"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	seleccionhttp "vec-diputacion-granada/internal/modules/seleccion/adapters/http"
)

const clavePoliticaPreparacionBasesHTTPV3 = "seleccion-preparacion-bases-v3"

// Check protege frente a peticiones cruzadas del navegador. La identidad,
// sesión y autorización se acreditan por sus autoridades independientes.
func nuevaFronteraPreparacionBasesHTTPV3(broker *proveedorPreparacionBasesV3, registrar func(*http.Request) error) (func(*http.Request) error, error) {
	if broker == nil || registrar == nil {
		return nil, errMontajePreparacionBasesV3
	}
	proteccion := http.NewCrossOriginProtection()
	denegar := func(r *http.Request) error {
		if r == nil || registrar(r) != nil {
			return bolsaports.ErrPreparacionBasesNoDisponible
		}
		return bolsaports.ErrPreparacionBasesDenegada
	}
	return func(r *http.Request) error {
		if r == nil {
			return bolsaports.ErrPreparacionBasesDenegada
		}
		if r.Method != http.MethodPost || r.URL.RawPath != "" || r.URL.RawQuery != "" ||
			(r.URL.Path != seleccionhttp.RutaGuardarPreparacionBases && r.URL.Path != seleccionhttp.RutaConsultarPreparacionBases) {
			return denegar(r)
		}
		if _, err := broker.indice(r.Context()); err != nil {
			if errors.Is(err, bolsaports.ErrPreparacionBasesDenegada) {
				return denegar(r)
			}
			return err
		}
		if proteccion.Check(r) != nil {
			return denegar(r)
		}
		return nil
	}, nil
}

func paresPreparacionBasesHTTPV3() []struct{ ruta, accion, frontera string } {
	return []struct{ ruta, accion, frontera string }{
		{seleccionhttp.RutaGuardarPreparacionBases, bolsaports.AccionGuardarPreparacionBases, "seleccion-preparacion-bases-guardar-v3"},
		{seleccionhttp.RutaConsultarPreparacionBases, bolsaports.AccionConsultarPreparacionBases, "seleccion-preparacion-bases-consultar-v3"},
	}
}

func fronterasPreparacionBasesHTTPV3(guardar, consultar string) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(guardar) || !perfilActivoSeguridadComunValido(consultar) || guardar == consultar {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	perfiles := []string{guardar, consultar}
	var salida []descriptorFronteraComunDesarrollo
	for i, p := range paresPreparacionBasesHTTPV3() {
		salida = append(salida, descriptorFronteraComunDesarrollo{Clave: p.frontera,
			Superficie: superficieInternaSeguridadComunDesarrollo, Metodo: http.MethodPost,
			Ruta: p.ruta, PerfilesActivosRef: []string{perfiles[i]},
			ClavePolitica: clavePoliticaPreparacionBasesHTTPV3, ClaveCapacidad: p.accion})
	}
	return salida, nil
}

func autorizacionesPreparacionBasesHTTPV3(politica politicaAutorizacionSolicitudLigadaV3Desarrollo) ([]descriptorAutorizacionComunDesarrollo, error) {
	if !politica.valida() {
		return nil, errAutorizacionComunDesarrolloNoDisponible
	}
	var salida []descriptorAutorizacionComunDesarrollo
	for _, p := range paresPreparacionBasesHTTPV3() {
		salida = append(salida, descriptorAutorizacionComunDesarrollo{Accion: p.accion,
			ClavePolitica: clavePoliticaPreparacionBasesHTTPV3, ClaveCapacidad: p.accion,
			Fronteras: []string{p.frontera}, Politica: politica})
	}
	return salida, nil
}

// Clasifica únicamente errores de la sesión compuesta sobre estas dos
// fronteras. La declaración o un contexto de otro catálogo no habilitan acceso.
func (p *proveedorSesionConsultaRRHHDesarrollo) sesionPreparacionBasesHTTPV3(ctx context.Context, ruta string) bool {
	if p == nil || p.soporte == nil || ctx == nil || p.fronteras.identidad == nil {
		return false
	}
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	f, existe := fronteraSeguridadComunDesdeContexto(ctx)
	if !ok || !existe || c.sello != p.soporte.sello || c.ruta != ruta || c.metodo != http.MethodPost ||
		f.metodo != http.MethodPost || f.ruta != ruta || f.superficie != superficieInternaSeguridadComunDesarrollo ||
		!p.fronteras.mismaInstancia(f.catalogo) || c.principal.ID != p.soporte.principalID ||
		c.principal.Attributes["certificate_sha256"] != p.soporte.certificadoSHA256 {
		return false
	}
	for _, par := range paresPreparacionBasesHTTPV3() {
		if ruta != par.ruta {
			continue
		}
		d, declarada := p.fronteras.resolver(http.MethodPost, ruta)
		return declarada && d.Clave == par.frontera && f.descriptor.Clave == d.Clave &&
			d.ClavePolitica == clavePoliticaPreparacionBasesHTTPV3 && d.ClaveCapacidad == par.accion &&
			f.descriptor.ClavePolitica == d.ClavePolitica && f.descriptor.ClaveCapacidad == d.ClaveCapacidad &&
			len(d.PerfilesActivosRef) == 1 && d.PerfilesActivosRef[0] == p.base.Contexto.PerfilActivoRef &&
			len(f.descriptor.PerfilesActivosRef) == 1 && f.descriptor.PerfilesActivosRef[0] == p.base.Contexto.PerfilActivoRef
	}
	return false
}
