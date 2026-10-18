package mibolsa

import (
	"context"
	"errors"
	"slices"
	"time"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// ServicioPreferenciasModalidades liga la selección al catálogo gobernado
// capturado por el servidor antes de pedir la decisión nominal.
type ServicioPreferenciasModalidades struct {
	repositorio bolsa.EscrituraPreferenciasModalidades
	catalogo    bolsa.CatalogoPreferenciasModalidades
	autoridad   autoridadPreferencias
}

func NuevoServicioPreferenciasModalidades(repositorio bolsa.EscrituraPreferenciasModalidades,
	catalogo bolsa.CatalogoPreferenciasModalidades, autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	proveedor bolsa.ProveedorMaterialPreferenciasBolsas, reloj puertosvec.Reloj,
) (*ServicioPreferenciasModalidades, error) {
	return nuevoServicioPreferenciasModalidades(repositorio, catalogo, autorizador, proveedor, reloj,
		dominiovec.SuperficieAutenticacionExternaPersonalV1, bolsa.AudienciaOrdenarPreferenciasBolsas)
}

func NuevoServicioPreferenciasModalidadesInterna(repositorio bolsa.EscrituraPreferenciasModalidades,
	catalogo bolsa.CatalogoPreferenciasModalidades, autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	proveedor bolsa.ProveedorMaterialPreferenciasBolsas, reloj puertosvec.Reloj,
) (*ServicioPreferenciasModalidades, error) {
	return nuevoServicioPreferenciasModalidades(repositorio, catalogo, autorizador, proveedor, reloj,
		dominiovec.SuperficieAutenticacionInternaCorporativaV1, bolsa.AudienciaOrdenarPreferenciasBolsasInterna)
}

func nuevoServicioPreferenciasModalidades(repositorio bolsa.EscrituraPreferenciasModalidades,
	catalogo bolsa.CatalogoPreferenciasModalidades, autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	proveedor bolsa.ProveedorMaterialPreferenciasBolsas, reloj puertosvec.Reloj,
	superficie dominiovec.SuperficieAutenticacionActorV1, audiencia string,
) (*ServicioPreferenciasModalidades, error) {
	if nula(repositorio) || nula(catalogo) || nula(autorizador) || nula(proveedor) || nula(reloj) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &ServicioPreferenciasModalidades{repositorio: repositorio, catalogo: catalogo,
		autoridad: autoridadPreferencias{autorizador, proveedor, reloj, superficie, audiencia}}, nil
}

func (s *ServicioPreferenciasModalidades) GuardarModalidades(ctx context.Context, orden Orden,
	seleccion []string, versionEsperada uint64, clave string,
) (bolsa.ReciboPreferenciasModalidades, error) {
	vacio := bolsa.ReciboPreferenciasModalidades{}
	if ctx == nil || s == nil || nula(s.repositorio) || nula(s.catalogo) ||
		nula(s.autoridad.autorizador) || nula(s.autoridad.proveedor) || nula(s.autoridad.reloj) {
		return vacio, bolsa.ErrPreferenciasNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if versionEsperada >= 2147483647 || !clavePreferenciasBolsas.MatchString(clave) {
		return vacio, bolsa.ErrOrdenPreferenciasInvalido
	}
	seleccion = slices.Clone(seleccion)
	slices.Sort(seleccion)
	actor, candidato, err := s.autoridad.identificar(orden)
	if err != nil {
		return vacio, err
	}
	disponibles, err := s.catalogo.ConsultarModalidadesDisponibles(ctx)
	if err != nil {
		return vacio, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	disponibles = slices.Clone(disponibles)
	huella, err := bolsaapp.HuellaSolicitudPreferenciasModalidades(candidato, versionEsperada, seleccion, disponibles)
	if err != nil {
		return vacio, err
	}
	preparado, err := s.autoridad.preparar(ctx, orden, actor, candidato, map[string]string{
		"propiedad": "candidato", "tipo_preferencia": "modalidades", "material_sha256": huella,
	})
	if err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	recibo, err := s.repositorio.GuardarPreferenciasModalidades(ctx, bolsa.GuardarPreferenciasModalidades{
		SolicitudPreferenciasBolsas: bolsa.SolicitudPreferenciasBolsas{CandidatoRef: candidato, ConsultadaEn: preparado.consultada},
		Material:                    preparado.material, Modalidades: slices.Clone(seleccion),
		Disponibles: slices.Clone(disponibles), VersionEsperada: versionEsperada, ClaveIdempotencia: clave,
	})
	if err != nil {
		return vacio, err
	}
	retorno := s.autoridad.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if recibo.ReciboRef == "" || recibo.Version != versionEsperada+1 || recibo.RegistradoEn.IsZero() ||
		recibo.RegistradoEn.After(retorno) || (recibo.Modalidades == nil) != (seleccion == nil) ||
		!slices.Equal(recibo.Modalidades, seleccion) {
		return vacio, bolsa.ErrPreferenciasNoDisponibles
	}
	return recibo, nil
}
