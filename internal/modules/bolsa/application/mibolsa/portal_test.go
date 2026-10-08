package mibolsa

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type registroPortalPrueba struct {
	solicitudes           []bolsa.SolicitudPortalCandidato
	documentales          []bolsa.SolicitudDocumentalPortal
	estadoDocumental      string
	reutilizadaDocumental bool
	respuestas            []bolsa.RespuestaPortalCandidato
	plazo                 bolsa.PlazoRespuestaPortal
}

func (r *registroPortalPrueba) SolicitarDocumentalPortal(_ context.Context, s bolsa.SolicitudDocumentalPortal) (bolsa.ReciboSolicitudDocumentalPortal, error) {
	r.documentales = append(r.documentales, s)
	estado := r.estadoDocumental
	if estado == "" {
		estado = "pendiente_rrhh"
	}
	return bolsa.ReciboSolicitudDocumentalPortal{SolicitudRef: s.SolicitudRef, ReciboRef: s.ReciboRef, ContenidoSHA256: s.ContenidoSHA256, RegistradaEn: s.RegistradaEn, Version: 1, Estado: estado, Reutilizada: r.reutilizadaDocumental}, nil
}

func (r *registroPortalPrueba) SolicitarPortal(_ context.Context, s bolsa.SolicitudPortalCandidato) (bolsa.ReciboSolicitudPortal, error) {
	r.solicitudes = append(r.solicitudes, s)
	return bolsa.ReciboSolicitudPortal{SolicitudRef: s.SolicitudRef, ReciboRef: s.ReciboRef, RegistradaEn: s.RegistradaEn}, nil
}

func (r *registroPortalPrueba) ResponderPortal(_ context.Context, s bolsa.RespuestaPortalCandidato, plazo bolsa.PlazoRespuestaPortal) (bolsa.ReciboRespuestaPortal, error) {
	r.respuestas = append(r.respuestas, s)
	r.plazo = plazo
	return bolsa.ReciboRespuestaPortal{RespuestaRef: s.RespuestaRef, ReciboRef: s.ReciboRef, RespondidaEn: s.RespondidaEn, Modo: s.Modo}, nil
}

type reglasPortalPrueba struct {
	modo      string
	maxima    time.Time
	maximaErr error
}

func (r reglasPortalPrueba) ModoRespuesta(context.Context) (string, string, error) {
	return r.modo, "vec.bolsa.reglas:1:b29.portal_candidato", nil
}
func (reglasPortalPrueba) ResultadosContactoEfectivo(context.Context) ([]string, error) {
	return []string{"contactado"}, nil
}
func (reglasPortalPrueba) SituacionesAdmitidas(_ context.Context, tipo string) ([]string, string, error) {
	if tipo == bolsa.SolicitudPortalPausa {
		return []string{"disponible"}, "vec.bolsa.reglas:1:b29.portal_candidato", nil
	}
	return []string{"no_disponible"}, "vec.bolsa.reglas:1:b29.portal_candidato", nil
}
func (r reglasPortalPrueba) PausaMaxima(context.Context, time.Time) (time.Time, string, error) {
	if r.maximaErr != nil {
		return time.Time{}, "", r.maximaErr
	}
	return r.maxima, "vec.bolsa.reglas:1:b18.pausa_voluntaria", nil
}
func (reglasPortalPrueba) VencimientoRespuesta(_ context.Context, contacto time.Time) (time.Time, string, error) {
	return contacto.Add(24 * time.Hour), "vec.bolsa.reglas:1:b05.plazo_respuesta", nil
}
func (reglasPortalPrueba) CausasRenunciaJustificada(context.Context) ([]string, error) {
	return []string{"enfermedad", "matrimonio_union_hecho"}, nil
}

type proveedorPortalPrueba struct {
	emisor   *emisorObservado
	acciones []string
}

func (p *proveedorPortalPrueba) EmitirMaterialPortalCandidato(ctx context.Context, accion string, s domain.SolicitudAutorizacionLigadaV3, r domain.ResultadoContextoActorRegistradoV2, d domain.DecisionAutorizacionLigadaV3, c ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3) (ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	p.acciones = append(p.acciones, accion)
	return p.emisor.EmitirMaterialMiBolsa(ctx, s, r, d, c)
}

type entornoPortal struct {
	*entornoMiBolsa
	portal    *Portal
	registro  *registroPortalPrueba
	proveedor *proveedorPortalPrueba
}

// nuevoEntornoPortal concede solo la acción indicada y emite capacidades para
// la audiencia indicada; así se prueba también el cruce de acción y audiencia.
func nuevoEntornoPortal(t *testing.T, accion, audiencia string, reglas reglasPortalPrueba) *entornoPortal {
	t.Helper()
	e := nuevoEntorno(t)
	c := &e.fuente.instantanea.VersionRol.Concesiones[0]
	c.Accion, c.Finalidades, c.CamposPermitidos = accion, []string{bolsa.FinalidadPortalCandidato}, nil
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:portal", 1, bytes.Repeat([]byte{0x72}, 32), "emisor:prueba:portal", audiencia, confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("8", 64))
	exigir(t, err)
	e.emisor.capacidades, err = confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
	exigir(t, err)
	p := &entornoPortal{entornoMiBolsa: e, registro: new(registroPortalPrueba), proveedor: &proveedorPortalPrueba{emisor: e.emisor}}
	p.portal, err = NuevoPortal(p.registro, e.autorizador, p.proveedor, reglas, reloj)
	exigir(t, err)
	return p
}

func TestPortalSolicitudDocumentalPropiaPendiente(t *testing.T) {
	p := nuevoEntornoPortal(t, bolsa.AccionPresentarSolicitudDocumentalPropia, bolsa.AudienciaPresentarSolicitudDocumentalPropia, reglasPortalPrueba{})
	comando := ComandoSolicitudDocumentalPortal{Bolsa: "bolsa:auxiliar", DocumentoRef: "documento:parte-1", DocumentoSHA256: strings.Repeat("a", 64), FechaFinCausa: "2026-10-02", Clave: "clave-documental-1"}
	recibo, err := p.portal.PresentarSolicitudDocumental(context.Background(), p.orden, comando)
	exigir(t, err)
	if len(p.registro.documentales) != 1 || len(p.registro.solicitudes) != 0 || p.firmas != 1 || recibo.Estado != "pendiente_rrhh" {
		t.Fatal("la solicitud documental no quedó separada de la pausa y pendiente de RRHH")
	}
	s := p.registro.documentales[0]
	if s.CandidatoRef != referenciaServicioContextoActorPrueba("can_", "c") || s.Material.ValidarEstructura() != nil ||
		s.ContenidoSHA256 != huellaPortal("contenido-solicitud-documental", s.CandidatoRef, comando.Bolsa, comando.DocumentoRef, comando.DocumentoSHA256, comando.FechaFinCausa) ||
		!strings.HasPrefix(recibo.SolicitudRef, "solicitud-documental:") || !strings.HasPrefix(recibo.ReciboRef, "recibo:solicitud-documental:") {
		t.Fatalf("material documental no ligado: %+v", s)
	}
	comando.FechaFinCausa = "2026-02-30"
	if _, err := p.portal.PresentarSolicitudDocumental(context.Background(), p.orden, comando); !errors.Is(err, bolsa.ErrPortalCandidatoInvalido) || len(p.registro.documentales) != 1 {
		t.Fatalf("fecha civil imposible admitida: %v", err)
	}
	comando.FechaFinCausa = "2026-10-02"
	comando.DocumentoRef = "dni:prueba"
	if _, err := p.portal.PresentarSolicitudDocumental(context.Background(), p.orden, comando); !errors.Is(err, bolsa.ErrPortalCandidatoInvalido) || len(p.registro.documentales) != 1 {
		t.Fatalf("referencia de identidad dejó pendiente irresoluble: %v", err)
	}
	comando.DocumentoRef = "documento:parte-1"
	comando.FechaFinCausa = ""
	comando.Clave = "clave-documental-sin-fecha"
	sinFecha, err := p.portal.PresentarSolicitudDocumental(context.Background(), p.orden, comando)
	if err != nil || sinFecha.Estado != "pendiente_rrhh" || p.registro.documentales[len(p.registro.documentales)-1].FechaFinCausa != "" {
		t.Fatalf("solicitud sin fecha inventó fin de causa: %+v, %v", sinFecha, err)
	}
	for _, estado := range []string{"validada", "rechazada"} {
		p.registro.estadoDocumental, p.registro.reutilizadaDocumental = estado, true
		repetida, err := p.portal.PresentarSolicitudDocumental(context.Background(), p.orden, ComandoSolicitudDocumentalPortal{Bolsa: "bolsa:auxiliar", DocumentoRef: "documento:parte-1", DocumentoSHA256: strings.Repeat("a", 64), FechaFinCausa: "2026-10-02", Clave: "clave-documental-1"})
		if err != nil || !repetida.Reutilizada || repetida.Estado != estado || repetida.ReciboRef != recibo.ReciboRef || repetida.ContenidoSHA256 != recibo.ContenidoSHA256 {
			t.Fatalf("replay %s no conserva recibo: %+v, %v", estado, repetida, err)
		}
	}
}

func TestHuellaSolicitudDocumentalSinFechaCoincideConSQLB77(t *testing.T) {
	got := huellaPortal("contenido-solicitud-documental", "can_"+strings.Repeat("A", 22),
		"bolsa:sintetica:rrhh17", "documento:sintetico:rrhh17", strings.Repeat("a", 64), "")
	if got != "9bff03e2c2b23f401a5208d2073851a14dcf26094f57539c07cc6e10b9e7fff2" {
		t.Fatalf("canon documental sin fecha no coincide con B77: %s", got)
	}
}

func TestPortalSolicitaPausaConAccionPropiaYReglas(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	p := nuevoEntornoPortal(t, bolsa.AccionSolicitarPausaPropia, bolsa.AudienciaSolicitarPausaPropia, reglasPortalPrueba{modo: "firme", maxima: ahora.AddDate(1, 0, 0)})
	hasta := p.ahora.Add(30 * 24 * time.Hour)
	recibo, err := p.portal.SolicitarPausa(context.Background(), p.orden, "bolsa:auxiliar", hasta, "clave-pausa-0001")
	exigir(t, err)
	if len(p.registro.solicitudes) != 1 || p.firmas != 1 || len(p.proveedor.acciones) != 1 || p.proveedor.acciones[0] != bolsa.AccionSolicitarPausaPropia {
		t.Fatal("no recorrió una sola cadena completa")
	}
	s := p.registro.solicitudes[0]
	if s.Tipo != "pausa" || s.PausaHasta == nil || !s.PausaHasta.Equal(hasta) || s.PausaMaxima == nil || s.ReglaRef != "vec.bolsa.reglas:1:b18.pausa_voluntaria" ||
		len(s.SituacionesAdmitidas) != 1 || s.CandidatoRef != referenciaServicioContextoActorPrueba("can_", "c") || s.Material.ValidarEstructura() != nil ||
		!strings.HasPrefix(s.SolicitudRef, "solicitud-portal:") || !strings.HasPrefix(recibo.ReciboRef, "recibo:solicitud-portal:") {
		t.Fatalf("solicitud inexacta: %+v", s)
	}
	datos, err := p.concesiones.orden.Datos()
	exigir(t, err)
	nominal, err := datos.Solicitud.Datos()
	exigir(t, err)
	if nominal.Accion != bolsa.AccionSolicitarPausaPropia || nominal.Recurso.Referencia != "mi-bolsa:"+s.CandidatoRef ||
		nominal.Recurso.Atributos["bolsa_ref"] != "bolsa:auxiliar" || nominal.Finalidad != bolsa.FinalidadPortalCandidato {
		t.Fatal("decisión sobre otro recurso o acción")
	}
	// La misma clave repite las mismas referencias: el replay es idempotente.
	_, err = p.portal.SolicitarPausa(context.Background(), p.orden, "bolsa:auxiliar", hasta, "clave-pausa-0001")
	exigir(t, err)
	if p.registro.solicitudes[1].SolicitudRef != s.SolicitudRef {
		t.Fatal("la repetición cambia la referencia")
	}
}

func TestPortalSinReglaPausaNoAutorizaNiEscribe(t *testing.T) {
	p := nuevoEntornoPortal(t, bolsa.AccionSolicitarPausaPropia, bolsa.AudienciaSolicitarPausaPropia,
		reglasPortalPrueba{modo: bolsa.ModoRespuestaPortalFirme, maximaErr: bolsa.ErrPausaPortalNoConfigurada})
	_, err := p.portal.SolicitarPausa(t.Context(), p.orden, "bolsa:auxiliar",
		p.ahora.Add(24*time.Hour), "clave-pausa-retirada")
	if !errors.Is(err, bolsa.ErrPortalCandidatoNoDisponible) ||
		p.firmas != 0 || len(p.proveedor.acciones) != 0 || len(p.registro.solicitudes) != 0 {
		t.Fatalf("pausa sin regla produjo efecto o permiso: %v", err)
	}
}

func TestPortalRechazaAntesDeAutorizar(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	p := nuevoEntornoPortal(t, bolsa.AccionSolicitarPausaPropia, bolsa.AudienciaSolicitarPausaPropia, reglasPortalPrueba{modo: "firme", maxima: ahora.Add(24 * time.Hour)})
	if _, err := p.portal.SolicitarPausa(context.Background(), p.orden, "bolsa:auxiliar", p.ahora.Add(48*time.Hour), "clave-pausa-0001"); !errors.Is(err, bolsa.ErrPortalPausaFueraDeLimite) {
		t.Fatalf("pausa más allá del máximo: %v", err)
	}
	for nombre, llamada := range map[string]func() error{
		"clave corta": func() error {
			_, err := p.portal.SolicitarReactivacion(context.Background(), p.orden, "bolsa:auxiliar", "corta")
			return err
		},
		"causa fuera del catálogo": func() error {
			_, err := p.portal.Responder(context.Background(), p.orden, ComandoRespuestaPortal{Bolsa: "bolsa:auxiliar", Respuesta: "renuncia_justificada", Causa: "viaje", JustificanteRef: "justificante:1", JustificanteHash: strings.Repeat("a", 64), Clave: "clave-respuesta-1"})
			return err
		},
		"renuncia con causa": func() error {
			_, err := p.portal.Responder(context.Background(), p.orden, ComandoRespuestaPortal{Bolsa: "bolsa:auxiliar", Respuesta: "renuncia", Causa: "enfermedad", Clave: "clave-respuesta-1"})
			return err
		},
	} {
		if err := llamada(); err == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
	if p.firmas != 0 || len(p.registro.solicitudes)+len(p.registro.respuestas) != 0 {
		t.Fatal("se autorizó o escribió una petición rechazada")
	}
}

func TestPortalNoUsaUnaAccionParaOtra(t *testing.T) {
	// Se concede pausa pero se pide reactivación: el PDP deniega sin material.
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	p := nuevoEntornoPortal(t, bolsa.AccionSolicitarPausaPropia, bolsa.AudienciaSolicitarReactivacionPropia, reglasPortalPrueba{modo: "firme", maxima: ahora.AddDate(1, 0, 0)})
	if _, err := p.portal.SolicitarReactivacion(context.Background(), p.orden, "bolsa:auxiliar", "clave-reactiva-01"); !errors.Is(err, domain.ErrAutorizacionDenegada) {
		t.Fatalf("acción no concedida: %v", err)
	}
	// Concedida la pausa pero con capacidades de otra audiencia: sin escritura.
	if _, err := p.portal.SolicitarPausa(context.Background(), p.orden, "bolsa:auxiliar", p.ahora.Add(time.Hour), "clave-pausa-0002"); err == nil {
		t.Fatal("material de otra audiencia aceptado")
	}
	if len(p.registro.solicitudes) != 0 {
		t.Fatal("escritura con material cruzado")
	}
}

func TestPortalRespondeConModoYPlazoDelCatalogo(t *testing.T) {
	ahora := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	p := nuevoEntornoPortal(t, bolsa.AccionResponderLlamamientoPropio, bolsa.AudienciaResponderLlamamientoPropio, reglasPortalPrueba{modo: "propuesta_rrhh", maxima: ahora})
	_, err := p.portal.Responder(context.Background(), p.orden, ComandoRespuestaPortal{
		Bolsa: "bolsa:auxiliar", Respuesta: "renuncia_justificada", Causa: "enfermedad",
		JustificanteRef: "justificante:parte-medico", JustificanteHash: strings.Repeat("b", 64), Clave: "clave-respuesta-1",
	})
	exigir(t, err)
	if len(p.registro.respuestas) != 1 || p.registro.plazo == nil {
		t.Fatal("no registró la respuesta")
	}
	r := p.registro.respuestas[0]
	if r.Modo != "propuesta_rrhh" || r.ReglaRef != "vec.bolsa.reglas:1:b29.portal_candidato" || len(r.ResultadosEfectivos) != 1 || r.Causa != "enfermedad" || r.Material.ValidarEstructura() != nil {
		t.Fatalf("respuesta inexacta: %+v", r)
	}
	vence, err := p.registro.plazo.VencimientoRespuesta(context.Background(), ahora)
	if err != nil || !vence.Equal(ahora.Add(24*time.Hour)) {
		t.Fatal("el plazo no procede del catálogo")
	}
}
