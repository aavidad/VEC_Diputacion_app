package mibolsa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"slices"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	claveIdempotenciaPortal = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)
	justificanteRefPortal   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{0,255}$`)
	huellaSHA256Portal      = regexp.MustCompile(`^[a-f0-9]{64}$`)
	bolsaRefPortal          = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`)
	causaPortal             = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
)

// Portal ejecuta las acciones propias del candidato: autoriza la acción
// exacta sobre su propio recurso, resuelve las reglas del catálogo y deja
// que PostgreSQL compruebe, consuma y conserve en una sola transacción.
type Portal struct {
	registro    puertosbolsa.RegistroPortalCandidato
	autorizador puertosvec.AutorizadorSolicitudLigadaV3
	proveedor   puertosbolsa.ProveedorMaterialPortalCandidato
	reglas      puertosbolsa.ReglasPortalCandidato
	reloj       puertosvec.Reloj
}

// ComandoRespuestaPortal es lo único que aporta la persona al responder.
type ComandoRespuestaPortal struct {
	Bolsa, Respuesta, Causa           string
	JustificanteRef, JustificanteHash string
	Clave                             string
}

func NuevoPortal(registro puertosbolsa.RegistroPortalCandidato, autorizador puertosvec.AutorizadorSolicitudLigadaV3, proveedor puertosbolsa.ProveedorMaterialPortalCandidato, reglas puertosbolsa.ReglasPortalCandidato, reloj puertosvec.Reloj) (*Portal, error) {
	if nula(registro) || nula(autorizador) || nula(proveedor) || nula(reglas) || nula(reloj) {
		return nil, ErrServicioMiBolsaInvalido
	}
	return &Portal{registro: registro, autorizador: autorizador, proveedor: proveedor, reglas: reglas, reloj: reloj}, nil
}

// SolicitarPausa pide no ser llamado hasta la fecha indicada. La pausa no
// empieza hasta que RRHH la valida (regla b29).
func (p *Portal) SolicitarPausa(ctx context.Context, orden Orden, bolsa string, hasta time.Time, clave string) (puertosbolsa.ReciboSolicitudPortal, error) {
	return p.solicitar(ctx, orden, puertosbolsa.SolicitudPortalPausa, bolsa, &hasta, clave)
}

// SolicitarReactivacion pide volver a estar disponible.
func (p *Portal) SolicitarReactivacion(ctx context.Context, orden Orden, bolsa, clave string) (puertosbolsa.ReciboSolicitudPortal, error) {
	return p.solicitar(ctx, orden, puertosbolsa.SolicitudPortalReactivacion, bolsa, nil, clave)
}

func (p *Portal) solicitar(ctx context.Context, orden Orden, tipo, bolsa string, hasta *time.Time, clave string) (puertosbolsa.ReciboSolicitudPortal, error) {
	var vacio puertosbolsa.ReciboSolicitudPortal
	if err := p.listo(ctx); err != nil {
		return vacio, err
	}
	if !bolsaRefPortal.MatchString(bolsa) || !claveIdempotenciaPortal.MatchString(clave) || (tipo == puertosbolsa.SolicitudPortalPausa) != (hasta != nil) {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	accion := puertosbolsa.AccionSolicitarReactivacionPropia
	audiencia := puertosbolsa.AudienciaSolicitarReactivacionPropia
	if tipo == puertosbolsa.SolicitudPortalPausa {
		accion, audiencia = puertosbolsa.AccionSolicitarPausaPropia, puertosbolsa.AudienciaSolicitarPausaPropia
	}
	situaciones, reglaRef, err := p.reglas.SituacionesAdmitidas(ctx, tipo)
	if err != nil || len(situaciones) == 0 || reglaRef == "" {
		return vacio, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	solicitud := puertosbolsa.SolicitudPortalCandidato{Tipo: tipo, Bolsa: bolsa, Clave: clave, SituacionesAdmitidas: situaciones, ReglaRef: reglaRef}
	if hasta != nil {
		ahora := p.reloj.Ahora().UTC()
		maxima, reglaPausa, err := p.reglas.PausaMaxima(ctx, ahora)
		if err != nil || reglaPausa == "" {
			return vacio, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
		}
		fin := hasta.UTC().Truncate(time.Microsecond)
		if !fin.After(ahora) || fin.After(maxima) {
			return vacio, puertosbolsa.ErrPortalPausaFueraDeLimite
		}
		maxima = maxima.UTC().Truncate(time.Microsecond)
		solicitud.PausaHasta, solicitud.PausaMaxima, solicitud.ReglaRef = &fin, &maxima, reglaPausa
	}
	material, candidato, ahora, err := p.autorizar(ctx, orden, accion, audiencia, bolsa)
	if err != nil {
		return vacio, err
	}
	solicitud.CandidatoRef, solicitud.Material, solicitud.RegistradaEn = candidato, material, ahora
	huella := huellaPortal("solicitud", candidato, bolsa, tipo, clave)
	solicitud.SolicitudRef, solicitud.ReciboRef = "solicitud-portal:"+huella, "recibo:solicitud-portal:"+huellaPortal("recibo", huella)
	return p.registro.SolicitarPortal(ctx, solicitud)
}

// Responder registra la respuesta al llamamiento abierto dentro de plazo. El
// modo (firme o propuesta que RRHH confirma) lo fija el catálogo.
func (p *Portal) Responder(ctx context.Context, orden Orden, c ComandoRespuestaPortal) (puertosbolsa.ReciboRespuestaPortal, error) {
	var vacio puertosbolsa.ReciboRespuestaPortal
	if err := p.listo(ctx); err != nil {
		return vacio, err
	}
	if !bolsaRefPortal.MatchString(c.Bolsa) || !claveIdempotenciaPortal.MatchString(c.Clave) {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	switch c.Respuesta {
	case puertosbolsa.RespuestaPortalAcepta, puertosbolsa.RespuestaPortalRenuncia:
		if c.Causa != "" || c.JustificanteRef != "" || c.JustificanteHash != "" {
			return vacio, puertosbolsa.ErrPortalCandidatoInvalido
		}
	case puertosbolsa.RespuestaPortalRenunciaJustificada:
		if !causaPortal.MatchString(c.Causa) || !justificanteRefPortal.MatchString(c.JustificanteRef) || !huellaSHA256Portal.MatchString(c.JustificanteHash) {
			return vacio, puertosbolsa.ErrPortalCandidatoInvalido
		}
		causas, err := p.reglas.CausasRenunciaJustificada(ctx)
		if err != nil {
			return vacio, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
		}
		if !slices.Contains(causas, c.Causa) {
			return vacio, puertosbolsa.ErrPortalCausaNoAdmitida
		}
	default:
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	modo, reglaRef, err := p.reglas.ModoRespuesta(ctx)
	if err != nil || reglaRef == "" || (modo != puertosbolsa.ModoRespuestaPortalFirme && modo != puertosbolsa.ModoRespuestaPortalPropuesta) {
		return vacio, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	efectivos, err := p.reglas.ResultadosContactoEfectivo(ctx)
	if err != nil || len(efectivos) == 0 {
		return vacio, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	material, candidato, ahora, err := p.autorizar(ctx, orden, puertosbolsa.AccionResponderLlamamientoPropio, puertosbolsa.AudienciaResponderLlamamientoPropio, c.Bolsa)
	if err != nil {
		return vacio, err
	}
	huella := huellaPortal("respuesta", candidato, c.Bolsa, c.Clave)
	respuesta := puertosbolsa.RespuestaPortalCandidato{
		RespuestaRef: "respuesta-portal:" + huella, ReciboRef: "recibo:respuesta-portal:" + huellaPortal("recibo", huella),
		CandidatoRef: candidato, Bolsa: c.Bolsa, Respuesta: c.Respuesta, Causa: c.Causa,
		JustificanteRef: c.JustificanteRef, JustificanteSHA256: c.JustificanteHash, Modo: modo,
		ResultadosEfectivos: efectivos, ReglaRef: reglaRef, Clave: c.Clave, RespondidaEn: ahora, Material: material,
	}
	return p.registro.ResponderPortal(ctx, respuesta, plazoPortal{reglas: p.reglas})
}

// autorizar pide la decisión de la acción exacta sobre el recurso propio y
// devuelve su material, ya cotejado con la operación y la audiencia.
func (p *Portal) autorizar(ctx context.Context, orden Orden, accion, audiencia, bolsa string) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, string, time.Time, error) {
	var vacio puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
	ahora := p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	resultadoActor, candidato, err := validarOrden(orden, ahora)
	if err != nil {
		return vacio, "", time.Time{}, err
	}
	recurso := dominiovec.RecursoAutorizable{
		Referencia: "mi-bolsa:" + candidato, ModuloID: puertosbolsa.ModuloMiBolsa, Tipo: puertosbolsa.TipoRecursoMiBolsa,
		Ambitos:   map[string]string{"candidato_ref": candidato},
		Atributos: map[string]string{"propiedad": "candidato", "bolsa_ref": bolsa},
	}
	if recurso.Validar() != nil {
		return vacio, "", time.Time{}, errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrPortalCandidatoInvalido)
	}
	nominal, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: orden.Vinculo, ReferenciaMotivo: orden.Motivo,
		Accion: accion, Recurso: recurso, Finalidad: puertosbolsa.FinalidadPortalCandidato, Correlacion: orden.Correlacion,
	})
	if err != nil {
		return vacio, "", time.Time{}, denegarPortal(err)
	}
	decision, confirmacion, err := p.autorizador.ExigirSolicitudLigadaV3(ctx, nominal, resultadoActor)
	if err != nil {
		return vacio, "", time.Time{}, denegarPortal(err)
	}
	ahora = p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err := ctx.Err(); err != nil {
		return vacio, "", time.Time{}, err
	}
	if !decisionExacta(nominal, decision, confirmacion, resultadoActor, ahora, []string{}) {
		return vacio, "", time.Time{}, denegarPortal(nil)
	}
	exportador, err := p.proveedor.EmitirMaterialPortalCandidato(ctx, accion, nominal, resultadoActor, decision, confirmacion)
	if err != nil || nula(exportador) {
		return vacio, "", time.Time{}, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacio, "", time.Time{}, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
	}
	ahora = p.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err := ctx.Err(); err != nil {
		return vacio, "", time.Time{}, err
	}
	if _, _, err = validarOrden(orden, ahora); err != nil || !materialExacto(material, nominal, decision, confirmacion, resultadoActor, ahora, accion, audiencia) {
		return vacio, "", time.Time{}, denegarPortal(err)
	}
	return material, candidato, ahora, nil
}

func (p *Portal) listo(ctx context.Context) error {
	if ctx == nil || p == nil || nula(p.registro) || nula(p.autorizador) || nula(p.proveedor) || nula(p.reglas) || nula(p.reloj) {
		return ErrServicioMiBolsaInvalido
	}
	return ctx.Err()
}

type plazoPortal struct {
	reglas puertosbolsa.ReglasPortalCandidato
}

func (p plazoPortal) VencimientoRespuesta(ctx context.Context, contacto time.Time) (time.Time, error) {
	vence, _, err := p.reglas.VencimientoRespuesta(ctx, contacto)
	return vence, err
}

func huellaPortal(partes ...string) string {
	h := sha256.New()
	for _, parte := range partes {
		h.Write([]byte(parte))
		h.Write([]byte{0x1f})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func denegarPortal(err error) error {
	return errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrPortalCandidatoInvalido, err)
}
