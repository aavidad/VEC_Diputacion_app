package mibolsa

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"time"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var clavePreferenciasBolsas = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

// ServicioPreferenciasBolsas prepara solo el acto propio de escritura. Las
// lecturas esperan la autoridad nominal común de sesión, permiso y auditoría.
type ServicioPreferenciasBolsas struct {
	repositorio bolsa.EscrituraPreferenciasBolsas
	autorizador puertosvec.AutorizadorSolicitudLigadaV3
	proveedor   bolsa.ProveedorMaterialPreferenciasBolsas
	reloj       puertosvec.Reloj
}

func NuevoServicioPreferenciasBolsas(repositorio bolsa.EscrituraPreferenciasBolsas,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	proveedor bolsa.ProveedorMaterialPreferenciasBolsas, reloj puertosvec.Reloj,
) (*ServicioPreferenciasBolsas, error) {
	if nula(repositorio) || nula(autorizador) || nula(proveedor) || nula(reloj) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &ServicioPreferenciasBolsas{repositorio: repositorio, autorizador: autorizador, proveedor: proveedor, reloj: reloj}, nil
}

// Ordenar exige V3 antes del efecto y vuelve a cotejar el material atestado.
// PostgreSQL comprueba el conjunto propio y el CAS al consumir esa decisión.
func (s *ServicioPreferenciasBolsas) Ordenar(ctx context.Context, orden Orden, referencias []string,
	versionEsperada uint64, clave string,
) (bolsa.ReciboPreferenciasBolsas, error) {
	if ctx == nil || s == nil || nula(s.repositorio) || nula(s.autorizador) || nula(s.proveedor) || nula(s.reloj) {
		return bolsa.ReciboPreferenciasBolsas{}, bolsa.ErrPreferenciasNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	if versionEsperada >= 2147483647 || !clavePreferenciasBolsas.MatchString(clave) ||
		bolsaapp.ValidarOrdenPreferencias(referencias, referencias) != nil {
		return bolsa.ReciboPreferenciasBolsas{}, bolsa.ErrOrdenPreferenciasInvalido
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	actor, candidato, err := validarOrden(orden, ahora)
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:" + candidato,
		ModuloID: bolsa.ModuloMiBolsa, Tipo: bolsa.TipoRecursoMiBolsa,
		Ambitos:   map[string]string{"candidato_ref": candidato},
		Atributos: map[string]string{"propiedad": "candidato"}}
	if recurso.Validar() != nil {
		return bolsa.ReciboPreferenciasBolsas{}, denegar(nil)
	}
	nominal, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: orden.Vinculo, ReferenciaMotivo: orden.Motivo,
		Accion: bolsa.AccionOrdenarPreferenciasBolsas, Recurso: recurso,
		Finalidad: bolsa.FinalidadPreferenciasBolsas, Correlacion: orden.Correlacion,
	})
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, denegar(err)
	}
	decision, confirmacion, err := s.autorizador.ExigirSolicitudLigadaV3(ctx, nominal, actor)
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, denegar(err)
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !decisionExacta(nominal, decision, confirmacion, actor, ahora, nil) {
		return bolsa.ReciboPreferenciasBolsas{}, denegar(nil)
	}
	exportador, err := s.proveedor.EmitirMaterialPreferenciasBolsas(ctx, nominal, actor, decision, confirmacion)
	if err != nil || nula(exportador) {
		return bolsa.ReciboPreferenciasBolsas{}, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if _, _, err = validarOrden(orden, ahora); err != nil ||
		!materialExacto(material, nominal, decision, confirmacion, actor, ahora,
			bolsa.AccionOrdenarPreferenciasBolsas, bolsa.AudienciaOrdenarPreferenciasBolsas) {
		return bolsa.ReciboPreferenciasBolsas{}, denegar(err)
	}
	if err := ctx.Err(); err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	recibo, err := s.repositorio.OrdenarPreferenciasBolsas(ctx, bolsa.OrdenarPreferenciasBolsas{
		SolicitudPreferenciasBolsas: bolsa.SolicitudPreferenciasBolsas{CandidatoRef: candidato, ConsultadaEn: ahora},
		Material:                    material, Orden: slices.Clone(referencias), VersionEsperada: versionEsperada,
		ClaveIdempotencia: clave,
	})
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	if recibo.ReciboRef == "" || recibo.Version != versionEsperada+1 || recibo.RegistradoEn.IsZero() ||
		recibo.RegistradoEn.After(ahora) || !slices.Equal(recibo.Orden, referencias) {
		return bolsa.ReciboPreferenciasBolsas{}, bolsa.ErrPreferenciasNoDisponibles
	}
	return recibo, nil
}
