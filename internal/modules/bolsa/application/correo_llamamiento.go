package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
	"unicode/utf8"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// componerCorreos personaliza, en el servidor, el asunto y el cuerpo de cada
// destinatario. Solo consulta a la fuente de personas si la plantilla usa un
// marcador personal; una persona ausente de la bolsa invalida la emisión.
func (s *ServicioEmisionLlamamiento) componerCorreos(ctx context.Context, bolsaRef string, participaciones []string, c puertosbolsa.ConfiguracionLlamamiento, ahora time.Time) ([]dominiobolsa.CorreoPersonalizado, error) {
	cat := s.correo.Catalogo
	datos := map[string]puertosbolsa.DatosPersonalizacionLlamamiento{}
	if cat.RequiereDatosPersonales(c.PlantillaVersion, c.Asunto, c.Cuerpo) {
		var err error
		datos, err = s.correo.Personalizacion.DatosPersonalizacionLlamamiento(ctx, bolsaRef, append([]string(nil), participaciones...))
		if err != nil {
			return nil, puertosbolsa.ErrEmisionLlamamientoNoDisponible
		}
		for _, p := range participaciones {
			if _, ok := datos[p]; !ok {
				return nil, fmt.Errorf("%w: %w", puertosbolsa.ErrEmisionLlamamientoInvalida, dominiobolsa.ErrDatosCorreoLlamamientoIncompletos)
			}
		}
	}
	correos := make([]dominiobolsa.CorreoPersonalizado, 0, len(participaciones))
	for _, p := range participaciones {
		d := datos[p]
		correo, err := cat.Personalizar(c.PlantillaVersion, c.Asunto, c.Cuerpo, dominiobolsa.ValoresCorreoLlamamiento{
			Nombre: d.Nombre, Apellidos: d.Apellidos, Bolsa: d.Bolsa, Posicion: d.Posicion,
			Categoria: c.Categoria, Referencia: c.Referencia, Centro: c.Centro, Modalidad: c.Modalidad,
			FechaInicio: c.FechaInicio, Plazo: c.Plazo, FechaEnvio: ahora,
		})
		if err != nil {
			return nil, fmt.Errorf("%w: %w", puertosbolsa.ErrEmisionLlamamientoInvalida, err)
		}
		correos = append(correos, correo)
	}
	return correos, nil
}

func huellasCorreos(participaciones []string, correos []dominiobolsa.CorreoPersonalizado) []puertosbolsa.HuellaCuerpoContacto {
	out := make([]puertosbolsa.HuellaCuerpoContacto, len(participaciones))
	for i, p := range participaciones {
		asunto, cuerpo := sha256.Sum256([]byte(correos[i].Asunto)), sha256.Sum256([]byte(correos[i].Cuerpo))
		out[i] = puertosbolsa.HuellaCuerpoContacto{ParticipacionRef: p, HuellaAsuntoSHA256: hex.EncodeToString(asunto[:]), HuellaCuerpoSHA256: hex.EncodeToString(cuerpo[:]), Caracteres: utf8.RuneCountInString(correos[i].Cuerpo)}
	}
	return out
}

// VistaPreviaLlamamiento muestra el correo exacto que recibiría una persona.
// No reserva, no envía ni consume autorización: exige el mismo ámbito de
// bolsa que la recuperación de emisiones.
func (s *ServicioEmisionLlamamiento) VistaPreviaLlamamiento(ctx context.Context, q puertosbolsa.SolicitudVistaPreviaLlamamiento) (puertosbolsa.VistaPreviaCorreoLlamamiento, error) {
	if ctx == nil || s == nil || q.ContextoActor.PersonaRef == "" || q.BolsaRef == "" || q.ParticipacionRef == "" || len(q.ParticipacionRef) > 256 {
		return puertosbolsa.VistaPreviaCorreoLlamamiento{}, puertosbolsa.ErrEmisionLlamamientoInvalida
	}
	resuelto, err := s.contextoBolsa.ResolverContextoContactosBolsa(ctx, q.ContextoActor, q.BolsaRef)
	if err != nil || resuelto.Validar() != nil {
		return puertosbolsa.VistaPreviaCorreoLlamamiento{}, errorDependenciaSituacion(err)
	}
	if err := s.validarConfiguracion(q.Configuracion); err != nil {
		return puertosbolsa.VistaPreviaCorreoLlamamiento{}, err
	}
	correos, err := s.componerCorreos(ctx, q.BolsaRef, []string{q.ParticipacionRef}, q.Configuracion, s.reloj().UTC())
	if err != nil {
		return puertosbolsa.VistaPreviaCorreoLlamamiento{}, err
	}
	return puertosbolsa.VistaPreviaCorreoLlamamiento{Asunto: correos[0].Asunto, Cuerpo: correos[0].Cuerpo, Caracteres: utf8.RuneCountInString(correos[0].Cuerpo), Limite: s.correo.Catalogo.LimiteCaracteres()}, nil
}

// PlantillaCorreoLlamamiento entrega al asistente la plantilla vigente del
// catálogo, sus textos por defecto y los marcadores admitidos.
func (s *ServicioEmisionLlamamiento) PlantillaCorreoLlamamiento(ctx context.Context, actor dominiovec.ContextoActor, idioma string) (puertosbolsa.PlantillaCorreoLlamamientoPublica, error) {
	if ctx == nil || s == nil || actor.PersonaRef == "" {
		return puertosbolsa.PlantillaCorreoLlamamientoPublica{}, dominiovec.ErrAutorizacionDenegada
	}
	cat := s.correo.Catalogo
	textos := cat.TextosDefecto(idioma)
	personalizada, _ := cat.PlantillaAdmitida(cat.PlantillaVigente())
	out := puertosbolsa.PlantillaCorreoLlamamientoPublica{PlantillaVersion: cat.PlantillaVigente(), Personalizada: personalizada, Limite: cat.LimiteCaracteres(), LimiteAsunto: cat.LimiteCaracteresAsunto(), Asunto: textos.Asunto, Cuerpo: textos.Cuerpo, Marcadores: []puertosbolsa.MarcadorCorreoLlamamientoPublico{}}
	if personalizada {
		for _, m := range cat.Marcadores() {
			out.Marcadores = append(out.Marcadores, puertosbolsa.MarcadorCorreoLlamamientoPublico{Clave: m.Clave})
		}
	}
	return out, nil
}
