package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	seg "vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// El PDP y el registro CAS son dobles declarados. Decisión, firma, verificación,
// capacidad y exportación usan implementaciones reales; no acreditan PostgreSQL.
type pdpConsultaAdministracionPrueba struct {
	campos       []string
	obligaciones []string
	llamadas     int
	solicitud    core.SolicitudAutorizacionLigadaV3
}

func (p *pdpConsultaAdministracionPrueba) ExigirSolicitudLigadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	p.llamadas++
	p.solicitud = s
	falla := func(err error) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
		return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, err
	}
	d, err := s.Datos()
	if err != nil {
		return falla(err)
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	rol := core.VersionRol{RolID: "admin_consulta", Version: 1, Nombre: "Consulta SMTP sintética", Estado: core.EstadoVersionRolPublicada, Concesiones: []core.ConcesionRol{{Accion: d.Accion, ModuloID: d.Recurso.ModuloID, TipoRecurso: d.Recurso.Tipo, Finalidades: []string{d.Finalidad}, GarantiaMinima: core.AuthAssuranceHigh, CamposPermitidos: p.campos, Obligaciones: p.obligaciones}}, PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-time.Hour)}
	huella, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		return falla(err)
	}
	i := core.InstantaneaAutorizacion{AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-consulta-admin", Version: 1, PerfilActivoRef: r.Contexto.PerfilActivoRef, PrincipalID: r.Contexto.PersonaRef, VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, Ambitos: []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionConfiguracionCorreoAdministracionV3}}}, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-time.Hour)}, VersionRol: rol, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
	e, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(s, i, "dec_0123456789abcdef0123456789abcdef", ahora, ahora.Add(30*time.Second))
	if err != nil {
		return falla(err)
	}
	decision, err := core.NuevaDecisionAutorizacionLigadaV3(s, e)
	if err != nil {
		return falla(err)
	}
	orden, err := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(s, decision, d.ReferenciaMotivo, r)
	if err != nil {
		return falla(err)
	}
	confirmacion, err := vp.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, registroConsultaAdministracionPrueba{}, orden)
	return decision, confirmacion, err
}

type registroConsultaAdministracionPrueba struct{}

func (registroConsultaAdministracionPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(_ context.Context, o vp.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	_, err := o.Datos()
	return time.Now().UTC().Truncate(time.Microsecond), err
}

type emisorConsultaAdministracionPrueba func(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)

func (f emisorConsultaAdministracionPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	return f(ctx, s, r)
}

func nuevosEmisoresConsultaAdministracionPrueba(t *testing.T, e escenarioAuditoriaAdministracionPrueba, pdp *pdpConsultaAdministracionPrueba, audiencia string) (*confianza.EmisorMaterialAutorizacionAtestadaV3, *confianza.EmisorMaterialAutorizacionAtestadaV3, archivoEmisorAdministracionDesarrollo) {
	t.Helper()
	c, semilla, secreto, _ := materialEmisorAdministracionPrueba(t)
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	c.Firma.Desde, c.Firma.Hasta = ahora.Add(-time.Hour), ahora.Add(time.Hour)
	c.Confianza.PublicadaEn, c.Confianza.ExpiraEn = c.Firma.Desde, c.Firma.Hasta
	c.Capacidad.Desde, c.Capacidad.Hasta = c.Firma.Desde, c.Firma.Hasta
	c.CapacidadConsulta.Desde, c.CapacidadConsulta.Hasta = c.Firma.Desde, c.Firma.Hasta
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(c.Firma.ClaveID, c.Firma.Version, ed25519.NewKeyFromSeed(semilla).Public().(ed25519.PublicKey), audienciaAtestacionAdministracionDesarrollo, confianza.EstadoClaveAtestacionAutorizacionV3Activa, c.Firma.Desde, c.Firma.Hasta, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	config, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(c.Confianza.Referencia, c.Confianza.Orden, c.Confianza.PublicadaEn, c.Confianza.ExpiraEn, raiz)
	if err != nil {
		t.Fatal(err)
	}
	c.Confianza.HuellaSHA256, err = config.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	reloj := relojContratacionTemporalDesarrollo{}
	atestador, verificador, capacidades, firmante, err := materialEmisorAdministracionDesarrollo(c, semilla, secreto, reloj)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(firmante.cerrar)
	capacidadesConsulta, err := materialCapacidadConsultaAdministracionDesarrollo(c, semilla, secreto, bytes.Repeat([]byte{11}, 32), reloj)
	if err != nil {
		t.Fatal(err)
	}
	if audiencia == audienciaConfiguracionCorreoAdministracionV3 {
		// Inyección deliberadamente incoherente para probar la validación final
		// del autorizador. La factoría productiva nunca utiliza esta asignación.
		capacidadesConsulta = capacidades
	} else if audiencia != audienciaConsultaConfiguracionCorreoAdministracionV3 {
		t.Fatal("audiencia de prueba desconocida")
	}
	autoridad := &autoridadEmisorAdministracionDesarrollo{delegado: pdp, identidad: *e.transporte.identidad, motivo: c.Motivo, reloj: reloj}
	escritura, lectura, err := nuevosEmisoresMaterialAdministracionDesarrollo(autoridad, atestador, verificador, capacidades, capacidadesConsulta)
	if err != nil {
		t.Fatal(err)
	}
	return escritura, lectura, c
}

func nuevoEmisorConsultaAdministracionPrueba(t *testing.T, e escenarioAuditoriaAdministracionPrueba, pdp *pdpConsultaAdministracionPrueba, audiencia string) (*confianza.EmisorMaterialAutorizacionAtestadaV3, core.ReferenciaEntradaCatalogo) {
	t.Helper()
	_, lectura, c := nuevosEmisoresConsultaAdministracionPrueba(t, e, pdp, audiencia)
	return lectura, c.Motivo
}

func camposConsultaAdministracionPrueba() []string {
	return []string{"configurada", "host", "modo_autenticacion", "modo_tls", "puerto", "referencia_ca", "remitente_fijo", "secreto_configurado", "server_name", "tiempo_maximo_ms", "usuario", "version"}
}

func preparacionConsultaAdministracionPrueba(ctx context.Context, e escenarioAuditoriaAdministracionPrueba) (adminports.PreparacionConsultaConfiguracionCorreo, error) {
	a, err := e.preparador.PrepararAuditoriaConsultaConfiguracionCorreo(ctx, e.sesion.valor.Principal)
	if err != nil {
		return adminports.PreparacionConsultaConfiguracionCorreo{}, err
	}
	b, err := adminapp.PayloadConsultaConfiguracionCorreo(a)
	return adminports.PreparacionConsultaConfiguracionCorreo{Auditoria: a, PayloadNegocio: b}, err
}

func TestConsultaAutorizacionAdministracionGETEmiteMaterialRealLigado(t *testing.T) {
	e := nuevaAuditoriaAdministracionPrueba(t)
	pdp := &pdpConsultaAdministracionPrueba{campos: camposConsultaAdministracionPrueba()}
	emisor, motivo := nuevoEmisorConsultaAdministracionPrueba(t, e, pdp, audienciaConsultaConfiguracionCorreoAdministracionV3)
	var errorEmision error
	llamadasEmisor := 0
	observador := emisorConsultaAdministracionPrueba(func(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
		llamadasEmisor++
		d, c, m, err := emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
		errorEmision = err
		return d, c, m, err
	})
	a, err := nuevoAutorizadorConsultaConfiguracionCorreoV3(e.sesion, e.preparador, observador, motivo, seg.GeneradorReferenciasCriptograficas{}, relojContratacionTemporalDesarrollo{})
	if err != nil {
		t.Fatal(err)
	}
	comprobarAuditoriaAdministracionTLSMetodo(t, e, http.MethodGet, func(ctx context.Context) error {
		p, err := preparacionConsultaAdministracionPrueba(ctx, e)
		if err != nil {
			return err
		}
		material, err := a.AutorizarConsultaConfiguracionCorreo(ctx, p)
		if err != nil {
			return fmt.Errorf("consulta válida: llamadas emisor=%d PDP=%d; error emisión=%v; error final=%w", llamadasEmisor, pdp.llamadas, errorEmision, err)
		}
		d, err := pdp.solicitud.Datos()
		h := sha256.Sum256(p.PayloadNegocio)
		if err != nil || pdp.llamadas != 1 || d.Accion != accionConsultaConfiguracionCorreoAdministracionV3 || len(d.Recurso.Ambitos) != 1 || d.Recurso.Ambitos["organizacion_ref"] != organizacionConfiguracionCorreoAdministracionV3 || len(d.Recurso.Atributos) != 1 || d.Recurso.Atributos["material_sha256"] != hex.EncodeToString(h[:]) || d.VinculoAutenticacionActor.ValidarPara(e.sesion.valor.Resultado) != nil {
			return errors.New("consulta no ligada al recurso, actor o payload")
		}
		if material.ValidarEstructura() != nil || material.ResumenCapacidad().AudienciaConsumo() != audienciaConsultaConfiguracionCorreoAdministracionV3 || material.ResumenCapacidad().Operacion() != accionConsultaConfiguracionCorreoAdministracionV3 || !camposDecisionConsultaConfiguracionCorreoExactos(material.DecisionCanonica()) {
			return errors.New("material GET incompleto o de escritura")
		}
		return nil
	})
}

func TestConsultaAutorizacionAdministracionRechazaAntesDelEmisor(t *testing.T) {
	for _, caso := range []string{"payload_ajeno", "actor_ajeno", "sesion_cruzada", "auditoria_escritura", "cancelacion", "canal_put"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevaAuditoriaAdministracionPrueba(t)
			c, _, _, _ := materialEmisorAdministracionPrueba(t)
			llamadas := 0
			emisor := emisorConsultaAdministracionPrueba(func(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vp.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
				llamadas++
				return core.DecisionAutorizacionLigadaV3{}, vp.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, errors.New("no debe alcanzarse")
			})
			a, err := nuevoAutorizadorConsultaConfiguracionCorreoV3(e.sesion, e.preparador, emisor, c.Motivo, seg.GeneradorReferenciasCriptograficas{}, relojContratacionTemporalDesarrollo{})
			if err != nil {
				t.Fatal(err)
			}
			metodo := http.MethodGet
			if caso == "canal_put" {
				metodo = http.MethodPut
			}
			comprobarAuditoriaAdministracionTLSMetodo(t, e, metodo, func(ctx context.Context) error {
				var p adminports.PreparacionConsultaConfiguracionCorreo
				var err error
				if caso != "canal_put" {
					p, err = preparacionConsultaAdministracionPrueba(ctx, e)
					if err != nil {
						return err
					}
				}
				switch caso {
				case "payload_ajeno":
					p.PayloadNegocio = append(p.PayloadNegocio, ' ')
				case "actor_ajeno":
					s, _ := vp.NuevaSolicitudSeudonimizarSujetoAlmacen("per_eeeeeeeeeeeeeeeeeeeeee", ambitoSeudonimoAuditoriaAdministracion)
					p.Auditoria.ActorID, err = e.seudonimizador.SeudonimizarSujetoAlmacen(ctx, s)
					if err == nil {
						p.PayloadNegocio, err = adminapp.PayloadConsultaConfiguracionCorreo(p.Auditoria)
					}
					if err != nil {
						return err
					}
				case "sesion_cruzada":
					e.sesion.valor.Principal.ID = "per_eeeeeeeeeeeeeeeeeeeeee"
				case "auditoria_escritura":
					p.Auditoria.Action = accionConfiguracionCorreoAdministracionV3
					p.Auditoria.Result = "accepted"
				case "cancelacion":
					var cancelar context.CancelFunc
					ctx, cancelar = context.WithCancel(ctx)
					cancelar()
				}
				m, err := a.AutorizarConsultaConfiguracionCorreo(ctx, p)
				if err != errAutorizacionConsultaConfiguracionCorreoV3 || llamadas != 0 || !reflect.DeepEqual(m, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}) {
					return fmt.Errorf("negativa %s alcanzó emisor o devolvió material", caso)
				}
				return nil
			})
		})
	}
}

func TestConsultaAutorizacionAdministracionRechazaCamposYAudienciaDeEscritura(t *testing.T) {
	for _, caso := range []string{"campos_extra", "campos_ausentes", "obligacion", "audiencia_put"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevaAuditoriaAdministracionPrueba(t)
			pdp := &pdpConsultaAdministracionPrueba{campos: camposConsultaAdministracionPrueba()}
			audiencia := audienciaConsultaConfiguracionCorreoAdministracionV3
			switch caso {
			case "campos_extra":
				pdp.campos = append(pdp.campos, "secreto")
			case "campos_ausentes":
				pdp.campos = nil
			case "obligacion":
				pdp.obligaciones = []string{"doble_control"}
			case "audiencia_put":
				audiencia = audienciaConfiguracionCorreoAdministracionV3
			}
			emisor, motivo := nuevoEmisorConsultaAdministracionPrueba(t, e, pdp, audiencia)
			a, err := nuevoAutorizadorConsultaConfiguracionCorreoV3(e.sesion, e.preparador, emisor, motivo, seg.GeneradorReferenciasCriptograficas{}, relojContratacionTemporalDesarrollo{})
			if err != nil {
				t.Fatal(err)
			}
			comprobarAuditoriaAdministracionTLSMetodo(t, e, http.MethodGet, func(ctx context.Context) error {
				p, err := preparacionConsultaAdministracionPrueba(ctx, e)
				if err != nil {
					return err
				}
				m, err := a.AutorizarConsultaConfiguracionCorreo(ctx, p)
				if err == nil || pdp.llamadas != 1 || !reflect.DeepEqual(m, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}) {
					return fmt.Errorf("concesión %s aceptada", caso)
				}
				return nil
			})
		})
	}
}

func TestEmisoresAdministracionSeparanGETPUTYConservanGobierno(t *testing.T) {
	for _, metodo := range []string{http.MethodGet, http.MethodPut} {
		t.Run(metodo, func(t *testing.T) {
			e := nuevaAuditoriaAdministracionPrueba(t)
			pdp := &pdpConsultaAdministracionPrueba{campos: camposConsultaAdministracionPrueba()}
			escritura, lectura, c := nuevosEmisoresConsultaAdministracionPrueba(t, e, pdp, audienciaConsultaConfiguracionCorreoAdministracionV3)
			comprobarAuditoriaAdministracionTLSMetodo(t, e, metodo, func(ctx context.Context) error {
				accion, audiencia, claveEsperada, secreto := accionConsultaConfiguracionCorreoAdministracionV3, audienciaConsultaConfiguracionCorreoAdministracionV3, c.CapacidadConsulta, bytes.Repeat([]byte{11}, 32)
				propio, ajeno := lectura, escritura
				if metodo == http.MethodPut {
					accion, audiencia, claveEsperada, secreto = accionConfiguracionCorreoAdministracionV3, audienciaConfiguracionCorreoAdministracionV3, c.Capacidad, bytes.Repeat([]byte{9}, 32)
					propio, ajeno = escritura, lectura
				}
				correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seg.GeneradorReferenciasCriptograficas{})
				if err != nil {
					return err
				}
				s, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: e.sesion.valor.Vinculo, ReferenciaMotivo: c.Motivo, Accion: accion, Recurso: core.RecursoAutorizable{Referencia: referenciaConfiguracionCorreoAdministracionV3, ModuloID: "vec.module.administracion", Tipo: tipoRecursoConfiguracionCorreoAdministracion, Ambitos: map[string]string{"organizacion_ref": organizacionConfiguracionCorreoAdministracionV3}, Atributos: map[string]string{"material_sha256": c.Firma.SPKISHA256}}, Finalidad: finalidadConfiguracionCorreoAdministracionV3, Correlacion: correlacion})
				if err != nil {
					return err
				}
				if _, _, material, err := ajeno.EmitirMaterialAutorizacionAtestadaV3(ctx, s, e.sesion.valor.Resultado); err == nil || material != nil || pdp.llamadas != 0 {
					return errors.New("emisor de otra acción alcanzó PDP")
				}
				_, _, exportador, err := propio.EmitirMaterialAutorizacionAtestadaV3(ctx, s, e.sesion.valor.Resultado)
				if err != nil || exportador == nil || pdp.llamadas != 1 {
					return fmt.Errorf("emisor propio incompleto: %w", err)
				}
				material, err := exportador.ExportarMaterialParaConsumidor()
				if err != nil || material.ResumenCapacidad().AudienciaConsumo() != audiencia || material.ResumenCapacidad().Operacion() != accion {
					return errors.New("material cruza acción o audiencia")
				}
				clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(claveEsperada.ClaveID, claveEsperada.Version, secreto, claveEsperada.EmisorID, audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Verificacion, claveEsperada.Desde, claveEsperada.Hasta, time.Time{}, claveEsperada.RevisionGobierno, claveEsperada.HuellaGobierno)
				if err != nil {
					return err
				}
				verificador, err := confianza.NuevoVerificadorCapacidadesAtestacionAutorizacionV3(relojContratacionTemporalDesarrollo{}, clave)
				if err != nil || verificador.VerificarExportacionCanonica(ctx, material.CapacidadCanonica()) != nil {
					return errors.New("capacidad no conserva clave o gobierno suministrados")
				}
				// Mismos metadatos pero secreto de otra operación: prueba el MAC,
				// no sólo una comparación de nombres de clave o audiencia.
				if metodo == http.MethodPut {
					secreto = bytes.Repeat([]byte{11}, 32)
				} else {
					secreto = bytes.Repeat([]byte{9}, 32)
				}
				clave, err = confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(claveEsperada.ClaveID, claveEsperada.Version, secreto, claveEsperada.EmisorID, audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Verificacion, claveEsperada.Desde, claveEsperada.Hasta, time.Time{}, claveEsperada.RevisionGobierno, claveEsperada.HuellaGobierno)
				if err != nil {
					return err
				}
				verificador, err = confianza.NuevoVerificadorCapacidadesAtestacionAutorizacionV3(relojContratacionTemporalDesarrollo{}, clave)
				if err != nil || verificador.VerificarExportacionCanonica(ctx, material.CapacidadCanonica()) == nil {
					return errors.New("capacidad acepta secreto de otra operación")
				}
				return nil
			})
		})
	}
}
