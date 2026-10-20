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
	superficie  dominiovec.SuperficieAutenticacionActorV1
	audiencia   string
}

// autoridadPreferencias conserva el mismo cotejo nominal para las dos
// preferencias. El recurso solo cambia por los atributos ligados al efecto.
type autoridadPreferencias struct {
	autorizador puertosvec.AutorizadorSolicitudLigadaV3
	proveedor   bolsa.ProveedorMaterialPreferenciasBolsas
	reloj       puertosvec.Reloj
	superficie  dominiovec.SuperficieAutenticacionActorV1
	audiencia   string
}

type materialPreferencias struct {
	material   puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	consultada time.Time
}

func (a autoridadPreferencias) identificar(orden Orden) (dominiovec.ResultadoContextoActorRegistradoV2, string, error) {
	return validarOrdenPreferencias(orden, a.reloj.Ahora().UTC().Truncate(time.Microsecond), a.superficie)
}

func (a autoridadPreferencias) preparar(ctx context.Context, orden Orden,
	actor dominiovec.ResultadoContextoActorRegistradoV2, candidato string,
	atributos map[string]string,
) (materialPreferencias, error) {
	recurso := dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:" + candidato,
		ModuloID: bolsa.ModuloMiBolsa, Tipo: bolsa.TipoRecursoMiBolsa,
		Ambitos: map[string]string{"candidato_ref": candidato}, Atributos: atributos}
	if recurso.Validar() != nil {
		return materialPreferencias{}, denegar(nil)
	}
	nominal, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: orden.Vinculo, ReferenciaMotivo: orden.Motivo,
		Accion: bolsa.AccionOrdenarPreferenciasBolsas, Recurso: recurso,
		Finalidad: bolsa.FinalidadPreferenciasBolsas, Correlacion: orden.Correlacion,
	})
	if err != nil {
		return materialPreferencias{}, denegar(err)
	}
	decision, confirmacion, err := a.autorizador.ExigirSolicitudLigadaV3(ctx, nominal, actor)
	if err != nil {
		return materialPreferencias{}, denegar(err)
	}
	ahora := a.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !decisionExacta(nominal, decision, confirmacion, actor, ahora, nil) {
		return materialPreferencias{}, denegar(nil)
	}
	exportador, err := a.proveedor.EmitirMaterialPreferenciasBolsas(ctx, nominal, actor, decision, confirmacion)
	if err != nil || nula(exportador) {
		return materialPreferencias{}, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return materialPreferencias{}, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	ahora = a.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if _, actual, err := validarOrdenPreferencias(orden, ahora, a.superficie); err != nil ||
		actual != candidato || !materialExacto(material, nominal, decision, confirmacion, actor, ahora,
		bolsa.AccionOrdenarPreferenciasBolsas, a.audiencia) {
		return materialPreferencias{}, denegar(err)
	}
	return materialPreferencias{material: material, consultada: ahora}, nil
}

func NuevoServicioPreferenciasBolsas(repositorio bolsa.EscrituraPreferenciasBolsas,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	proveedor bolsa.ProveedorMaterialPreferenciasBolsas, reloj puertosvec.Reloj,
) (*ServicioPreferenciasBolsas, error) {
	return nuevoServicioPreferenciasBolsasSuperficie(repositorio, autorizador, proveedor, reloj, dominiovec.SuperficieAutenticacionExternaPersonalV1)
}

func NuevoServicioPreferenciasBolsasInterna(repositorio bolsa.EscrituraPreferenciasBolsas,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	proveedor bolsa.ProveedorMaterialPreferenciasBolsas, reloj puertosvec.Reloj,
) (*ServicioPreferenciasBolsas, error) {
	return nuevoServicioPreferenciasBolsasSuperficie(repositorio, autorizador, proveedor, reloj, dominiovec.SuperficieAutenticacionInternaCorporativaV1)
}

func nuevoServicioPreferenciasBolsasSuperficie(repositorio bolsa.EscrituraPreferenciasBolsas,
	autorizador puertosvec.AutorizadorSolicitudLigadaV3,
	proveedor bolsa.ProveedorMaterialPreferenciasBolsas, reloj puertosvec.Reloj,
	superficie dominiovec.SuperficieAutenticacionActorV1,
) (*ServicioPreferenciasBolsas, error) {
	if nula(repositorio) || nula(autorizador) || nula(proveedor) || nula(reloj) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	audiencia := bolsa.AudienciaOrdenarPreferenciasBolsas
	if superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 {
		audiencia = bolsa.AudienciaOrdenarPreferenciasBolsasInterna
	}
	return &ServicioPreferenciasBolsas{repositorio: repositorio, autorizador: autorizador, proveedor: proveedor, reloj: reloj, superficie: superficie, audiencia: audiencia}, nil
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
	autoridad := autoridadPreferencias{s.autorizador, s.proveedor, s.reloj, s.superficie, s.audiencia}
	actor, candidato, err := autoridad.identificar(orden)
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	preparado, err := autoridad.preparar(ctx, orden, actor, candidato, map[string]string{"propiedad": "candidato"})
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	if err := ctx.Err(); err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	recibo, err := s.repositorio.OrdenarPreferenciasBolsas(ctx, bolsa.OrdenarPreferenciasBolsas{
		SolicitudPreferenciasBolsas: bolsa.SolicitudPreferenciasBolsas{CandidatoRef: candidato, ConsultadaEn: preparado.consultada},
		Material:                    preparado.material, Orden: slices.Clone(referencias), VersionEsperada: versionEsperada,
		ClaveIdempotencia: clave,
	})
	if err != nil {
		return bolsa.ReciboPreferenciasBolsas{}, err
	}
	retorno := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if recibo.ReciboRef == "" || recibo.Version != versionEsperada+1 || recibo.RegistradoEn.IsZero() ||
		recibo.RegistradoEn.After(retorno) || !slices.Equal(recibo.Orden, referencias) {
		return bolsa.ReciboPreferenciasBolsas{}, bolsa.ErrPreferenciasNoDisponibles
	}
	return recibo, nil
}
