package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type consultorReciboPrueba struct {
	despues    func()
	encontrado bool
	orden      ports.OrdenConsultaReciboContactoUsuario
	mutar      func(*ports.ResultadoConsultaReciboContactoUsuario)
	mutarOrden bool
	llamadas   int
}

func (c *consultorReciboPrueba) ConsultarReciboContactoUsuario(_ context.Context, o ports.OrdenConsultaReciboContactoUsuario) (ports.ResultadoConsultaReciboContactoUsuario, error) {
	c.llamadas++
	c.orden = clonarOrdenConsultaRecibo(o)
	r, err := resultadoReciboFuentePrueba(o, c.encontrado)
	if c.mutarOrden {
		o.Acceso.Recurso.Ambitos["unidad"] = "ajena"
		o.Acceso.PayloadNegocio[0] = '!'
		o.Acceso.Auditoria.ActorRoles[0] = "rol-alterado"
	}
	if c.mutar != nil {
		c.mutar(&r)
	}
	if c.despues != nil {
		c.despues()
	}
	return r, err
}

// Fixture del JSON de T13/1 + T13/7. Firmas/id de cadena son sintéticos: no se
// verifica criptografía histórica con ellos ni se cambia una firma real.
func resultadoReciboFuentePrueba(o ports.OrdenConsultaReciboContactoUsuario, encontrado bool) (ports.ResultadoConsultaReciboContactoUsuario, error) {
	a := o.Acceso
	r := ports.ResultadoConsultaReciboContactoUsuario{Encontrado: encontrado, SujetoRef: a.SujetoRef, Version: a.Version, ConsumoConsultaRef: "aud_v3_" + strings.Repeat("d", 32), ConsumoConsultaHuellaSHA256: strings.Repeat("d", 64)}
	base, err := reciboCentralContactoFuente(ports.OrdenRegistroContactoUsuario{Preparacion: ports.PreparacionRegistroContactoUsuario{SujetoRef: a.SujetoRef, VersionNueva: a.Version, Auditoria: a.Auditoria, PayloadNegocio: a.PayloadNegocio, Recurso: a.Recurso}, Material: a.Material})
	if err != nil {
		return r, err
	}
	var consulta proyeccionCentralContacto
	if err = json.Unmarshal(base.EvidenciaCentral.JSONOriginal, &consulta); err != nil {
		return r, err
	}
	refOriginal, shaOriginal := "", ""
	if encontrado {
		original := consulta
		original.ID = "acc_" + strings.Repeat("e", 40)
		original.AuthorizationRef = "dec_" + strings.Repeat("e", 32)
		original.Action = AccionActualizarContactoUsuario
		if a.Version == 1 {
			original.Action = AccionAltaContactoUsuario
		}
		original.ActorRoles = []string{"rol-historico-uno", "rol-historico-dos"}
		original.AfterHash = strings.Repeat("a", 64)
		original.CorrelationRef = "correlacion-original-sintetica"
		original.Metadata = map[string]string{"consumo_ref": base.ConsumoRef, "consumo_huella_sha256": base.ConsumoHuellaSHA256, "material_sha256": original.AfterHash, "contexto_recurso_sha256": strings.Repeat("a", 64)}
		raw, err := json.MarshalIndent(original, "", " ")
		if err != nil {
			return r, err
		}
		evidencia, err := envolverEvidenciaCentralContacto(raw)
		if err != nil {
			return r, err
		}
		r.ReciboOriginal = ports.ReciboContactoUsuario{SujetoRef: a.SujetoRef, Version: a.Version, ConsumoRef: base.ConsumoRef, ConsumoHuellaSHA256: base.ConsumoHuellaSHA256, EvidenciaCentral: evidencia}
		refOriginal, shaOriginal = evidencia.Referencia, evidencia.HuellaJSONSHA256
	}
	consulta.Metadata["consumo_ref"] = r.ConsumoConsultaRef
	consulta.Metadata["consumo_huella_sha256"] = r.ConsumoConsultaHuellaSHA256
	consulta.Metadata["recibo_encontrado"] = "false"
	if encontrado {
		consulta.Metadata["recibo_encontrado"] = "true"
	}
	consulta.Metadata["recibo_original_ref"] = refOriginal
	consulta.Metadata["recibo_original_sha256"] = shaOriginal
	raw, err := json.Marshal(consulta)
	if err != nil {
		return r, err
	}
	r.AuditoriaConsulta, err = envolverEvidenciaCentralContacto(raw)
	return r, err
}

func entornoConsultaReciboPrueba(t *testing.T) (*ServicioConsultaReciboContactoUsuario, ports.SolicitudConsultaReciboContactoUsuario, *consultorReciboPrueba, *entornoContacto) {
	t.Helper()
	e := nuevoEntornoContacto(t)
	base := e.solicitud.SolicitudBase
	base.Accion = AccionConsultarContactoUsuario
	base.Finalidad = FinalidadConsultaReciboContactoUsuario
	base.Recurso.Referencia = e.resultado.Contexto.PersonaRef
	base.Recurso.ModuloID = "vec.module.usuarios"
	base.Recurso.Tipo = "contacto_usuario"
	for i := range e.fuente.instantanea.VersionRol.Concesiones {
		c := &e.fuente.instantanea.VersionRol.Concesiones[i]
		c.ModuloID = base.Recurso.ModuloID
		c.TipoRecurso = base.Recurso.Tipo
		c.Finalidades = []string{base.Finalidad}
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	contactoExigir(t, err)
	t.Cleanup(func() { clear(priv) })
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	at, err := NuevoServicioAtestacionesAutorizacionV3(domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3, Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:contacto", Audiencia: "vec/prueba/contacto"}, firmanteContacto{priv, e.ahora})
	contactoExigir(t, err)
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:prueba:contacto", 1, pub, "vec/prueba/contacto", confianza.EstadoClaveAtestacionAutorizacionV3Activa, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{})
	contactoExigir(t, err)
	cfg, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:prueba:recibo", 1, e.ahora.Add(-time.Minute), e.ahora.Add(time.Hour), raiz)
	contactoExigir(t, err)
	ver, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, reloj)
	contactoExigir(t, err)
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:recibo", 1, bytes.Repeat([]byte{0x74}, 32), "emisor:prueba:recibo", AudienciaConsultaReciboContactoUsuario, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
	contactoExigir(t, err)
	em, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	contactoExigir(t, err)
	real, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(e.entornoAutorizacionSolicitudV3Prueba.servicio, at, ver, em)
	contactoExigir(t, err)
	e.emisor = &emisorContactoObservado{real: real}
	correlacion, err := base.Correlacion.ValorCanonico()
	contactoExigir(t, err)
	c := &consultorReciboPrueba{encontrado: true}
	servicio, err := NuevoServicioConsultaReciboContactoUsuario(auditorContactoPrueba{e.ahora, base.Finalidad, correlacion}, e.emisor, c)
	contactoExigir(t, err)
	servicio.ahora = func() time.Time { return e.ahora }
	return servicio, ports.SolicitudConsultaReciboContactoUsuario{ContextoActor: e.resultado.Contexto, Version: 2, Recurso: base.Recurso, SolicitudBase: base, ResultadoContexto: e.resultado}, c, e
}

func TestConsultaReciboContactoEmisorV3RealConservaOriginalYAuditaAcceso(t *testing.T) {
	s, p, c, _ := entornoConsultaReciboPrueba(t)
	c.mutarOrden = true
	r, err := s.Consultar(context.Background(), p)
	contactoExigir(t, err)
	if !r.Encontrado || r.Version != 2 || c.llamadas != 1 || r.ReciboOriginal.EvidenciaCentral.Referencia == r.AuditoriaConsulta.Referencia || r.ReciboOriginal.ConsumoRef == r.ConsumoConsultaRef {
		t.Fatal("confundió recibo histórico y consulta")
	}
	acceso := c.orden.Acceso
	proyeccionVieja := ports.ReciboContactoUsuario{SujetoRef: r.SujetoRef, Version: r.Version, ConsumoRef: r.ConsumoConsultaRef, ConsumoHuellaSHA256: r.ConsumoConsultaHuellaSHA256, EvidenciaCentral: r.AuditoriaConsulta}
	if evidenciaCentralEsperadaContacto(proyeccionVieja, acceso.SujetoRef, acceso.Version, acceso.Auditoria, acceso.PayloadNegocio, acceso.Recurso, acceso.Material.ResumenCapacidad().DecisionRef()) {
		t.Fatal("wrapper anterior aceptó la metadata ampliada")
	}
	esperado, err := resultadoReciboFuentePrueba(c.orden, true)
	contactoExigir(t, err)
	if !bytes.Equal(r.ReciboOriginal.EvidenciaCentral.JSONOriginal, esperado.ReciboOriginal.EvidenciaCentral.JSONOriginal) {
		t.Fatal("alteró los bytes originales")
	}
	if _, err := DecodificarResultadoConsultaReciboContactoUsuario(c.orden, true, r.SujetoRef, r.Version, r.ReciboOriginal.EvidenciaCentral.JSONOriginal, r.ReciboOriginal.ConsumoRef, r.ReciboOriginal.ConsumoHuellaSHA256, r.AuditoriaConsulta.JSONOriginal, r.ConsumoConsultaRef, r.ConsumoConsultaHuellaSHA256); err != nil {
		t.Fatal(err)
	}
}

func TestConsultaReciboContactoAusenteTambienExigeAuditoriaYMetadataExacta(t *testing.T) {
	s, p, c, _ := entornoConsultaReciboPrueba(t)
	c.encontrado = false
	r, err := s.Consultar(context.Background(), p)
	contactoExigir(t, err)
	if r.Encontrado || r.AuditoriaConsulta.Referencia == "" || r.ReciboOriginal.EvidenciaCentral.Referencia != "" {
		t.Fatal("ausencia no auditada")
	}
	var central map[string]any
	contactoExigir(t, json.Unmarshal(r.AuditoriaConsulta.JSONOriginal, &central))
	metadata := central["metadata"].(map[string]any)
	delete(metadata, "recibo_original_ref")
	metadata["campo_ajeno"] = ""
	raw, err := json.Marshal(central)
	contactoExigir(t, err)
	r.AuditoriaConsulta, err = envolverEvidenciaCentralContacto(raw)
	contactoExigir(t, err)
	if ValidarResultadoConsultaReciboContactoUsuario(c.orden, r) == nil {
		t.Fatal("clave requerida vacía ausente fue sustituida por desconocida")
	}
}

func TestConsultaReciboContactoRechazaMutacionesAutoridadYResultado(t *testing.T) {
	for _, nombre := range []string{"sujeto", "finalidad", "cuerpo", "version_original", "bytes_original", "consumo_original", "decision_consulta", "extra_metadata"} {
		t.Run(nombre, func(t *testing.T) {
			s, p, c, e := entornoConsultaReciboPrueba(t)
			switch nombre {
			case "sujeto":
				p.Recurso.Referencia = "per_" + strings.Repeat("z", 22)
			case "finalidad":
				p.SolicitudBase.Finalidad = "envio_llamamiento"
			case "cuerpo":
				e.emisor.mutar = func(d *domain.DatosSolicitudAutorizacionLigadaV3) {
					d.Recurso.Atributos["material_sha256"] = strings.Repeat("a", 64)
				}
			default:
				c.mutar = func(r *ports.ResultadoConsultaReciboContactoUsuario) {
					switch nombre {
					case "version_original":
						r.ReciboOriginal.Version++
					case "bytes_original":
						r.ReciboOriginal.EvidenciaCentral.JSONOriginal = append(r.ReciboOriginal.EvidenciaCentral.JSONOriginal, ' ')
						h := sha256.Sum256(r.ReciboOriginal.EvidenciaCentral.JSONOriginal)
						r.ReciboOriginal.EvidenciaCentral.HuellaJSONSHA256 = hex.EncodeToString(h[:])
					case "consumo_original":
						r.ReciboOriginal.ConsumoHuellaSHA256 = strings.Repeat("f", 64)
					default:
						var central map[string]any
						contactoExigir(t, json.Unmarshal(r.AuditoriaConsulta.JSONOriginal, &central))
						if nombre == "decision_consulta" {
							central["authorization_ref"] = "dec_" + strings.Repeat("e", 32)
						} else {
							central["metadata"].(map[string]any)["extra"] = "x"
						}
						raw, err := json.Marshal(central)
						contactoExigir(t, err)
						r.AuditoriaConsulta, err = envolverEvidenciaCentralContacto(raw)
						contactoExigir(t, err)
					}
				}
			}
			if _, err := s.Consultar(context.Background(), p); !errors.Is(err, ErrContactoUsuarioNoDisponible) {
				t.Fatal("mutación admitida")
			}
		})
	}
}

func TestConsultaReciboContactoDeniegaExposicionTrasCaducidadDuranteIO(t *testing.T) {
	s, p, c, e := entornoConsultaReciboPrueba(t)
	c.despues = func() { e.ahora = e.ahora.Add(time.Hour) }
	r, err := s.Consultar(context.Background(), p)
	if !errors.Is(err, ErrContactoUsuarioNoDisponible) || r.Encontrado || r.AuditoriaConsulta.Referencia != "" || c.llamadas != 1 {
		t.Fatal("expuso evidencia tras caducidad o confundió denegación con ausencia de acceso")
	}
}
