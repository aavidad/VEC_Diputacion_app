package mibolsa

import (
	"context"
	"errors"
	"regexp"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var tipoEventoHistorial = regexp.MustCompile(`^[a-z][a-z0-9_]{1,39}$`)

// ServicioHistorial exige una concesión propia para las tres clases de datos.
// No reutiliza la autorización de la instantánea minimizada de Mi Bolsa.
type ServicioHistorial struct {
	consulta    puertosbolsa.ConsultaHistorialMiBolsa
	autorizador puertosvec.AutorizadorSolicitudLigadaV3
	proveedor   puertosbolsa.ProveedorMaterialHistorialMiBolsa
	reloj       puertosvec.Reloj
}

func NuevoHistorial(consulta puertosbolsa.ConsultaHistorialMiBolsa, autorizador puertosvec.AutorizadorSolicitudLigadaV3, proveedor puertosbolsa.ProveedorMaterialHistorialMiBolsa, reloj puertosvec.Reloj) (*ServicioHistorial, error) {
	if nula(consulta) || nula(autorizador) || nula(proveedor) || nula(reloj) {
		return nil, puertosbolsa.ErrHistorialMiBolsaNoDisponible
	}
	return &ServicioHistorial{consulta: consulta, autorizador: autorizador, proveedor: proveedor, reloj: reloj}, nil
}

func (s *ServicioHistorial) ConsultarHistorial(ctx context.Context, orden Orden, pagina int) (puertosbolsa.PaginaHistorialMiBolsa, error) {
	var vacia puertosbolsa.PaginaHistorialMiBolsa
	if ctx == nil || s == nil || nula(s.consulta) || nula(s.autorizador) || nula(s.proveedor) || nula(s.reloj) || pagina < 1 || pagina > puertosbolsa.MaximaPaginaHistorialMiBolsa {
		return vacia, puertosbolsa.ErrConsultaHistorialMiBolsaInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	actor, candidato, err := validarOrden(orden, ahora)
	if err != nil {
		return vacia, err
	}
	recurso := dominiovec.RecursoAutorizable{Referencia: "mi-bolsa:" + candidato, ModuloID: puertosbolsa.ModuloMiBolsa, Tipo: puertosbolsa.TipoRecursoMiBolsa, Ambitos: map[string]string{"candidato_ref": candidato}, Atributos: map[string]string{"propiedad": "candidato"}}
	if recurso.Validar() != nil {
		return vacia, denegar(nil)
	}
	nominal, err := dominiovec.NuevaSolicitudAutorizacionLigadaV3(dominiovec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: orden.Vinculo, ReferenciaMotivo: orden.Motivo,
		Accion: puertosbolsa.AccionConsultarHistorialPropio, Recurso: recurso,
		Finalidad: puertosbolsa.FinalidadHistorialMiBolsa, Correlacion: orden.Correlacion,
	})
	if err != nil {
		return vacia, denegar(err)
	}
	decision, confirmacion, err := s.autorizador.ExigirSolicitudLigadaV3(ctx, nominal, actor)
	if err != nil {
		return vacia, denegar(err)
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ctx.Err() != nil {
		return vacia, ctx.Err()
	}
	if !decisionExacta(nominal, decision, confirmacion, actor, ahora, puertosbolsa.CamposHistorialMiBolsa()) {
		return vacia, denegar(nil)
	}
	exportador, err := s.proveedor.EmitirMaterialHistorialMiBolsa(ctx, nominal, actor, decision, confirmacion)
	if err != nil || nula(exportador) {
		return vacia, errors.Join(puertosbolsa.ErrHistorialMiBolsaNoDisponible, err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		return vacia, errors.Join(puertosbolsa.ErrHistorialMiBolsaNoDisponible, err)
	}
	ahora = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ctx.Err() != nil {
		return vacia, ctx.Err()
	}
	if _, _, err = validarOrden(orden, ahora); err != nil || !materialExacto(material, nominal, decision, confirmacion, actor, ahora, puertosbolsa.AccionConsultarHistorialPropio, puertosbolsa.AudienciaHistorialMiBolsa) {
		return vacia, denegar(err)
	}
	resultado, err := s.consulta.ConsultarHistorialMiBolsa(ctx, puertosbolsa.SolicitudConsultaHistorialMiBolsa{CandidatoRef: candidato, Material: material, ConsultadaEn: ahora, Pagina: pagina})
	if err != nil {
		return vacia, err
	}
	if !historialValido(resultado, ahora, pagina) {
		return vacia, puertosbolsa.ErrResultadoHistorialMiBolsaInvalido
	}
	return resultado, nil
}

func historialValido(r puertosbolsa.PaginaHistorialMiBolsa, ahora time.Time, pagina int) bool {
	if !r.ConsultadaEn.Equal(ahora) || r.Pagina != pagina || r.Tamano != puertosbolsa.TamanoPaginaHistorialMiBolsa || len(r.Items) > r.Tamano || (r.HayMas && len(r.Items) != r.Tamano) {
		return false
	}
	for i, h := range r.Items {
		if h.Bolsa == "" || h.Categoria == "" || h.OcurridoEn.IsZero() || h.OcurridoEn.After(ahora) || i > 0 && h.OcurridoEn.After(r.Items[i-1].OcurridoEn) {
			return false
		}
		switch h.Clase {
		case "contrato_bolsa":
			if !tipoEventoHistorial.MatchString(h.Tipo) || h.Procedencia != "evento_ct_recibido" || h.FinPrevisto != nil && h.Inicio != nil && h.FinPrevisto.Before(*h.Inicio) || h.Canal != "" || h.Respuesta != "" || h.Modo != "" || h.Estado != "" {
				return false
			}
		case "llamamiento":
			if h.Canal != "correo" || h.Resultado != "enviado" && h.Resultado != "no_enviado" && h.Resultado != "aviso_pendiente" || h.Tipo != "" || h.Procedencia != "" || h.Respuesta != "" || h.Modo != "" || h.Estado != "" {
				return false
			}
		case "renuncia":
			if h.Respuesta != "renuncia" && h.Respuesta != "renuncia_justificada" || h.Modo != "firme" && h.Modo != "propuesta_rrhh" || h.Estado != "respuesta_registrada" && h.Estado != "propuesta_pendiente_rrhh" || h.Modo == "firme" && h.Estado != "respuesta_registrada" || h.Modo == "propuesta_rrhh" && h.Estado != "propuesta_pendiente_rrhh" || h.Tipo != "" || h.Procedencia != "" || h.Canal != "" || h.Resultado != "" {
				return false
			}
		default:
			return false
		}
	}
	return true
}
