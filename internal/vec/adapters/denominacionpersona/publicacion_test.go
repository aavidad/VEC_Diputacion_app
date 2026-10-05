package denominacionpersona

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type registroPublicacionPrueba struct {
	llamadas        int
	fallo           error
	alterar         bool
	invalidarRecibo bool
}

type intentosPublicacionPrueba struct {
	intentosPrueba
	ultima ports.OrdenIntentoAuditoria
}

func (i *intentosPublicacionPrueba) AppendIntentoAuditoria(ctx context.Context, o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	i.ultima = o
	return i.intentosPrueba.AppendIntentoAuditoria(ctx, o)
}

func (r *registroPublicacionPrueba) PublicarDenominacionPersona(_ context.Context, o ports.OrdenDenominacionPersona) (ports.ReciboDenominacionPersona, error) {
	r.llamadas++
	if r.alterar {
		o.Acceso.Recurso.Referencia = "recurso:alterado"
		o.Acceso.Recurso.Ambitos["unidad_ref"] = "unidad:alterada"
		o.Preparacion.Sobre.Cifrado[0] ^= 1
	}
	if r.fallo != nil {
		return ports.ReciboDenominacionPersona{}, r.fallo
	}
	p := o.Preparacion
	recibo := ports.ReciboDenominacionPersona{PersonaRef: p.PersonaRef, ProcedenciaRef: p.ProcedenciaRef, SobreSHA256: p.SobreSHA256, AuditoriaRef: "auditoria:prueba", Version: p.Sobre.Version}
	if r.invalidarRecibo {
		recibo.PersonaRef = personaDos
	}
	return recibo, nil
}

func ordenPublicacionPrueba(t *testing.T) ports.OrdenDenominacionPersona {
	t.Helper()
	protector, _ := escenario(t)
	a := accesoPrueba(t)
	a.Recurso.ModuloID, a.Recurso.Tipo, a.Recurso.Referencia = "vec", "persona_denominacion", personaUno
	a.Recurso.Ambitos = map[string]string{"unidad_ref": "unidad:prueba"}
	return ports.OrdenDenominacionPersona{Preparacion: preparar(t, protector, personaUno, 0), Acceso: a}
}

func configPublicacionPrueba() ConfiguracionIntentosPublicacion {
	c := configIntentosPrueba()
	denegado := c.MotivoError
	denegado.EntradaClave = "denegado_observado"
	return ConfiguracionIntentosPublicacion{Proceso: c.Proceso, Canal: c.Canal, MotivoError: c.MotivoError, MotivoDenegado: denegado, PlazoAuditoria: time.Second}
}

func TestPublicacionConservaConfirmacionYClasificaResultados(t *testing.T) {
	for _, caso := range []string{"confirmado", "denegado", "commit_desconocido", "recibo_ajeno"} {
		t.Run(caso, func(t *testing.T) {
			registro := &registroPublicacionPrueba{}
			if caso == "denegado" {
				registro.fallo = domain.ErrAutorizacionDenegada
			}
			if caso == "commit_desconocido" {
				registro.fallo = errors.New("commit_desconocido")
			}
			registro.invalidarRecibo = caso == "recibo_ajeno"
			intentos := &intentosPublicacionPrueba{}
			p, err := NuevoPublicador(registro, intentos, configPublicacionPrueba())
			if err != nil {
				t.Fatal(err)
			}
			recibo, err := p.PublicarDenominacionPersona(context.Background(), ordenPublicacionPrueba(t))
			if registro.llamadas != 1 {
				t.Fatal("publicacion_repetida")
			}
			if caso == "confirmado" {
				if err != nil || recibo.PersonaRef != personaUno || intentos.llamadas != 0 {
					t.Fatal("confirmacion")
				}
				return
			}
			if recibo != (ports.ReciboDenominacionPersona{}) || intentos.llamadas != 1 {
				t.Fatal("resultado_sin_auditoria")
			}
			d, fallo := intentos.ultima.Datos()
			if fallo != nil || d.Datos.Accion != ports.AccionPublicarDenominacionPersona {
				t.Fatal("intento")
			}
			if caso == "denegado" {
				if !errors.Is(err, domain.ErrAutorizacionDenegada) || d.Datos.Resultado != domain.ResultadoIntentoAuditoriaDenegado || d.Datos.Motivo.EntradaClave != "denegado_observado" {
					t.Fatal("clasificacion_denegacion")
				}
			} else if !errors.Is(err, ErrNoDisponible) || d.Datos.Resultado != domain.ResultadoIntentoAuditoriaError {
				t.Fatal("clasificacion_error")
			}
		})
	}
}

func TestPublicacionAislaEvidenciaYReutilizaIntentoAmbiguo(t *testing.T) {
	orden := ordenPublicacionPrueba(t)
	original := orden.Preparacion.Sobre.Cifrado[0]
	registro := &registroPublicacionPrueba{fallo: errors.New("commit_desconocido"), alterar: true}
	intentos := &intentosPrueba{ambiguo: true}
	p, _ := NuevoPublicador(registro, intentos, configPublicacionPrueba())
	_, err := p.PublicarDenominacionPersona(context.Background(), orden)
	d, fallo := intentos.orden.Datos()
	if !errors.Is(err, ErrNoDisponible) || fallo != nil || registro.llamadas != 1 || intentos.llamadas != 2 || d.Datos.RecursoRef != orden.Acceso.Recurso.Referencia || orden.Acceso.Recurso.Ambitos["unidad_ref"] != "unidad:prueba" || orden.Preparacion.Sobre.Cifrado[0] != original {
		t.Fatal("evidencia_o_idempotencia")
	}
}

type intentoCancelacionPrueba struct {
	intentosPrueba
	cancelado bool
}

func (i *intentoCancelacionPrueba) AppendIntentoAuditoria(ctx context.Context, o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	i.cancelado = ctx.Err() != nil
	return i.intentosPrueba.AppendIntentoAuditoria(ctx, o)
}

func TestPublicacionCanceladaConservaAuditoriaConPlazoPropio(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	registro := &registroPublicacionPrueba{fallo: context.Canceled}
	intentos := &intentoCancelacionPrueba{}
	p, _ := NuevoPublicador(registro, intentos, configPublicacionPrueba())
	_, err := p.PublicarDenominacionPersona(ctx, ordenPublicacionPrueba(t))
	if !errors.Is(err, ErrNoDisponible) || intentos.cancelado || intentos.llamadas != 1 {
		t.Fatal("cancelacion_sin_intento")
	}
}

func TestPublicacionNoAtribuyeIdentidadSinEvidencia(t *testing.T) {
	registro := &registroPublicacionPrueba{}
	intentos := &intentosPrueba{}
	p, _ := NuevoPublicador(registro, intentos, configPublicacionPrueba())
	orden := ordenPublicacionPrueba(t)
	orden.Acceso.Contexto.PersonaRef = personaDos
	_, err := p.PublicarDenominacionPersona(context.Background(), orden)
	if !errors.Is(err, ErrNoDisponible) || registro.llamadas != 0 || intentos.llamadas != 0 {
		t.Fatal("actor_inventado")
	}
}

func TestPublicacionNoDesviaElRecursoNominalDeAuditoria(t *testing.T) {
	for _, campo := range []string{"persona", "recurso", "modulo", "tipo", "finalidad"} {
		t.Run(campo, func(t *testing.T) {
			orden := ordenPublicacionPrueba(t)
			switch campo {
			case "persona":
				orden.Acceso.PersonaRef = personaDos
			case "recurso":
				orden.Acceso.Recurso.Referencia = personaDos
			case "modulo":
				orden.Acceso.Recurso.ModuloID = "otro_modulo"
			case "tipo":
				orden.Acceso.Recurso.Tipo = "otro_tipo"
			case "finalidad":
				orden.Acceso.FinalidadRef = "otra_finalidad"
			}
			registro := &registroPublicacionPrueba{}
			intentos := &intentosPublicacionPrueba{}
			p, _ := NuevoPublicador(registro, intentos, configPublicacionPrueba())
			_, err := p.PublicarDenominacionPersona(context.Background(), orden)
			d, fallo := intentos.ultima.Datos()
			if !errors.Is(err, ErrNoDisponible) || fallo != nil || registro.llamadas != 0 || intentos.llamadas != 1 || d.Datos.ModuloID != "vec" || d.Datos.RecursoRef != personaUno || d.Datos.FinalidadRef != "presentacion_persona" || d.Datos.Resultado != domain.ResultadoIntentoAuditoriaError {
				t.Fatal("recurso_nominal_divergente")
			}
		})
	}
}
