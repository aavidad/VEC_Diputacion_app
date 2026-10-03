package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	ctadapters "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	vecapp "vec-diputacion-granada/internal/vec/application"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestOriginalDocumentosCTMontajeIncompletoDeniega(t *testing.T) {
	for _, configuraciones := range [][]configuracionAutorizacionesOriginalDocumentosCT{nil, {{}}, {{}, {}}} {
		proveedor, err := nuevasAutorizacionesOriginalDocumentosCTDesarrollo(configuraciones...)
		if proveedor != nil || !errors.Is(err, errAutoridadOriginalDocumentosCTNoDisponible) {
			t.Fatalf("montaje incompleto: %T %v", proveedor, err)
		}
	}
	if solicitudAutorizacionOriginalDocumentosCTValida(context.Background(), solicitudOriginalDocumentosCT{}.datos) {
		t.Fatal("contexto sin sello aceptado")
	}
}

// Registro y fuente de prueba; PDP, decisión, COSE/Ed25519 y HMAC son reales.
// No acredita instalación ni consumo transaccional de PostgreSQL.
type gobiernoOriginalCTPrueba struct {
	*soporteAltaContratacionTemporalDesarrollo
	perfil       *perfilFijoCTDesarrollo
	motivo       core.ReferenciaEntradaCatalogo
	registros    int
	revocarEnCAS bool
}

func (g *gobiernoOriginalCTPrueba) ObtenerInstantaneaAutorizacion(ctx context.Context, actor, perfil string) (core.InstantaneaAutorizacion, error) {
	d, ok := ctx.Value(claveSolicitudAutorizacionContratacionTemporalDesarrollo{}).(core.DatosSolicitudAutorizacionLigadaV3)
	if !ok || !solicitudAutorizacionOriginalDocumentosCTValida(ctx, d) || actor != g.perfil.plantilla.AsignacionPerfil.PrincipalID || perfil != g.perfil.perfilRef() {
		return core.InstantaneaAutorizacion{}, vecports.ErrFuenteAutorizacionNoDisponible
	}
	return clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(g.perfil.plantilla), nil
}
func (g *gobiernoOriginalCTPrueba) ValidarReferenciaMotivoAutorizacionV2(_ context.Context, m core.ReferenciaEntradaCatalogo, _ time.Time) error {
	if m != g.motivo {
		return core.ErrSolicitudAutorizacionInvalida
	}
	return nil
}
func (g *gobiernoOriginalCTPrueba) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx context.Context, o vecports.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	if g.revocarEnCAS {
		return time.Time{}, vecports.ErrInstantaneaAutorizacionObsoleta
	}
	if _, estado := g.consumirPerfilFijoCTDesarrolloConEstado(ctx, g.perfil); estado != perfilFijoConsumoVigente {
		return time.Time{}, vecports.ErrInstantaneaAutorizacionObsoleta
	}
	if _, err := o.Datos(); err != nil {
		return time.Time{}, err
	}
	g.registros++
	return g.reloj.Ahora(), nil
}

type escenarioOriginalAutorizacionCT struct {
	a   *autorizacionesOriginalDocumentosCT
	g   *gobiernoOriginalCTPrueba
	ctx context.Context
	s   vecports.SolicitudOriginalFirmableCT
	pdf vecports.PDFOriginalCT
	ref string
}

func nuevoEscenarioOriginalAutorizacionCT(t *testing.T) escenarioOriginalAutorizacionCT {
	t.Helper()
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	const ruta = httpinterno.RutaFirmaDocumento
	operaciones := map[string]operacionAutorizacionOriginalDocumentosCT{
		docports.AccionDescargar:                       {finalidad: "descargar_documento_original", tipo: "documento_original", campos: []string{"contenido", "documento"}},
		docports.AccionReservarOriginalFirmable:        {finalidad: docports.FinalidadOriginalFirmable, tipo: "documento_original_firmable"},
		docports.AccionConfirmarOriginalFirmable:       {finalidad: docports.FinalidadOriginalFirmable, tipo: "documento_original_firmable"},
		vecports.AccionNegocioEscribirOriginalFirmable: {finalidad: docports.FinalidadOriginalFirmable, tipo: "documento_original_firmable", campos: []string{"evidencia_almacen", "original_firmable.contenido"}},
	}
	concesiones := make([]core.ConcesionRol, 0, len(operaciones))
	for accion, o := range operaciones {
		concesiones = append(concesiones, core.ConcesionRol{Accion: accion, ModuloID: "documentos", TipoRecurso: o.tipo,
			Finalidades: []string{o.finalidad}, GarantiaMinima: core.AuthAssuranceHigh, CamposPermitidos: o.campos})
	}
	slices.SortFunc(concesiones, func(a, b core.ConcesionRol) int { return strings.Compare(a.Accion, b.Accion) })
	perfil, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, "original-ct-prueba", []string{ruta}, func(actor, ref string) (core.InstantaneaAutorizacion, error) {
		return nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(actor, ref, ahora, "original-ct-prueba", "Original CT sintético", "original-ct-prueba",
			concesiones, []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{docports.OrganizacionRefV3}}})
	})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(perfil) != nil {
		t.Fatal(err)
	}
	perfil.contextoEsperadoRegistrado, perfil.sesionOperativa = perfil.contexto.Resultado, proveedorSesionOperativaCTPrueba{contexto: perfil.contexto}
	asignaciones := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	asignaciones.asignaciones = map[string]instantaneaPublicadaDesarrollo{perfil.perfilRef(): {instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(perfil.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	g := &gobiernoOriginalCTPrueba{soporteAltaContratacionTemporalDesarrollo: s, perfil: perfil, motivo: s.motivo}
	pdp, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(g, g, g, g, s.reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(nuevoDerivadorIdempotenciaPrueba(t, 31, 1), ahora)
	if err != nil {
		t.Fatal(err)
	}
	defer material.borrarCopiasEfimeras()
	material.capacidad, err = confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3(material.claveHMACID, material.claveHMACVersion, material.claveHMAC, material.emisorID, docports.AudienciaV3,
		confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, material.validaDesde, material.validaHasta, time.Time{}, material.claveHMACRevision, material.claveHMACHuella)
	if err != nil {
		t.Fatal(err)
	}
	exportador, err := nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(material, s, s.reloj)
	if err != nil {
		t.Fatal(err)
	}
	for accion, o := range operaciones {
		o.rutas = []string{ruta}
		o.perfil = perfil
		o.motivo = s.motivo
		o.exportador = exportador
		operaciones[accion] = o
	}
	catalogo, err := conservacion.NuevoCatalogoProvisionalV2(s.reloj)
	if err != nil {
		t.Fatal(err)
	}
	tipos, err := ctadapters.NuevosTiposOriginalFirmableRRHH(catalogo)
	if err != nil {
		t.Fatal(err)
	}
	a := &autorizacionesOriginalDocumentosCT{soporte: s, pdp: pdp, reloj: s.reloj, politicas: catalogo, tipos: tipos,
		seudonimizador: nuevoSeudonimizadorAlmacenDesarrollo([sha256.Size]byte{42}), operaciones: operaciones}
	solicitud := vecports.SolicitudOriginalFirmableCT{OrganizacionRef: docports.OrganizacionRefV3, ExpedienteRef: "expediente:ct:uno", Documento: "informe_definitivo", OriginalVersion: 7}
	identidad := almacencanonico.IdentidadOriginalCT{OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef, Documento: solicitud.Documento, Version: solicitud.OriginalVersion}
	solicitud.OriginalRef = identidad.Referencia()
	tipo, err := tipos.ResolverTipoOriginalCT(context.Background(), solicitud.Documento)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextoRutaCoberturaDesarrolloPrueba(s, principal, ruta)
	cap := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	cap.metodo = http.MethodPost
	ctx = context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, cap)
	return escenarioOriginalAutorizacionCT{a: a, g: g, ctx: ctx, s: solicitud, pdf: vecports.PDFOriginalCT{TipoRef: tipo, Contenido: []byte("%PDF-1.7\noriginal CT sintético")}, ref: solicitud.OriginalRef}
}

func TestOriginalDocumentosCTEmisionYRecuperacionRevalidadas(t *testing.T) {
	e := nuevoEscenarioOriginalAutorizacionCT(t)
	consulta, err := e.a.AutorizarLecturaOriginalCT(e.ctx, e.s, e.ref)
	if err != nil || consulta.Autorizacion.ValidarPara(docports.AccionDescargar, e.a.reloj.Ahora()) != nil {
		t.Fatalf("lectura nominal: %v", err)
	}
	if consulta.Autorizacion.PrincipalID != e.g.perfil.plantilla.AsignacionPerfil.PrincipalID || consulta.Autorizacion.PerfilActivoRef != e.g.perfil.perfilRef() {
		t.Fatal("actor inventado")
	}
	otra, err := e.a.AutorizarLecturaOriginalCT(e.ctx, e.s, e.ref)
	if err != nil || otra.DocumentoID != consulta.DocumentoID || otra.Autorizacion.CorrelacionRef == consulta.Autorizacion.CorrelacionRef || e.g.registros != 2 {
		t.Fatalf("replay no reautorizado: %v", err)
	}
	orden, autoridad, err := e.a.PrepararCustodiaOriginalCT(e.ctx, e.s, e.pdf, e.ref)
	if err != nil || autoridad == nil {
		t.Fatalf("preparación: %v", err)
	}
	politica, err := vecapp.ResolverPoliticaConservacionDocumentalAdmitiendoProvisional(e.ctx, e.a.politicas, e.a.reloj, orden.SolicitudPolitica)
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(orden.Contenido)
	reserva := docports.ReservaOriginalFirmable{ID: orden.ID, ClaveIdempotencia: orden.ClaveIdempotencia, ModuloID: orden.ModuloID,
		ExpedienteRef: orden.ExpedienteRef, TipoRef: orden.TipoRef, Version: orden.Version, MIME: orden.MIME, HuellaSHA256: hex.EncodeToString(suma[:]), Tamano: int64(len(orden.Contenido)), Politica: politica}
	preimagen, err := docapp.PreimagenReservaOriginalFirmable(reserva)
	if err != nil {
		t.Fatal(err)
	}
	antesReserva := e.g.registros
	ctxAjeno := context.WithValue(e.ctx, struct{}{}, true)
	if _, err := autoridad.AutorizarReservaOriginal(ctxAjeno, preimagen, orden.ID, orden.ExpedienteRef); err == nil || e.g.registros != antesReserva {
		t.Fatal("otra petición reutilizó autoridad ligada")
	}
	preimagenAlterada := bytes.Replace(preimagen, []byte(reserva.HuellaSHA256), []byte(strings.Repeat("f", 64)), 1)
	if _, err := autoridad.AutorizarReservaOriginal(e.ctx, preimagenAlterada, orden.ID, orden.ExpedienteRef); err == nil || e.g.registros != antesReserva {
		t.Fatal("PDF diferente obtuvo reserva")
	}
	reserva.Autorizacion, err = autoridad.AutorizarReservaOriginal(e.ctx, preimagen, orden.ID, orden.ExpedienteRef)
	if err != nil || docapp.ValidarAutorizacionOriginalFirmable(reserva.Autorizacion, docports.AccionReservarOriginalFirmable, e.a.reloj.Ahora()) != nil {
		t.Fatalf("reserva: %v", err)
	}
	intento := docports.IntentoOriginalFirmable{ReservaRef: "ref:" + strings.Repeat("a", 64), Estado: "pendiente", DocumentoID: orden.ID,
		HuellaSHA256: reserva.HuellaSHA256, ClaveAlmacenRef: "ref:" + strings.Repeat("b", 64), Numero: 1}
	contexto, err := autoridad.ContextoEscrituraOriginal(e.ctx, reserva, intento)
	if err != nil || contexto.ValidarParaEn(vecports.AccionAlmacenEscribir, e.a.reloj.Ahora()) != nil {
		t.Fatalf("almacén ligado: %v", err)
	}
	confirmacion := docports.ConfirmacionOriginalFirmable{Intento: intento, Objeto: docports.ObjetoOriginalFirmable{
		ClaveAlmacenRef: intento.ClaveAlmacenRef, ObjetoRef: "objeto:ct:original", ObjetoVersion: "v1", ConectorRef: "almacen_prueba", ReciboObjetoRef: "recibo:ct:original",
		ReciboObjetoHuellaSHA256: strings.Repeat("c", 64), MIME: orden.MIME, Tamano: reserva.Tamano, HuellaSHA256: reserva.HuellaSHA256}}
	preimagen, err = docapp.PreimagenConfirmacionOriginalFirmable(confirmacion)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = autoridad.AutorizarConfirmacionOriginal(e.ctx, preimagen, orden.ID, orden.ExpedienteRef); err != nil {
		t.Fatalf("confirmación: %v", err)
	}
	asignaciones := e.g.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	if asignaciones.preparadas != 0 || asignaciones.publicadas != 0 {
		t.Fatal("publicó permisos durante emisión")
	}
	delete(asignaciones.asignaciones, e.g.perfil.perfilRef())
	antes := e.g.registros
	if _, err = e.a.AutorizarLecturaOriginalCT(e.ctx, e.s, e.ref); !errors.Is(err, docports.ErrAccesoDenegado) || e.g.registros != antes {
		t.Fatalf("recuperación revocada: %v", err)
	}
	if _, err = autoridad.AutorizarConfirmacionOriginal(e.ctx, preimagen, orden.ID, orden.ExpedienteRef); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("confirmación revocada: %v", err)
	}
}

func TestOriginalDocumentosCTDeniegaCanalCertificadoPerfilYMaterial(t *testing.T) {
	e := nuevoEscenarioOriginalAutorizacionCT(t)
	original := e.ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	for nombre, alterar := range map[string]func(*capacidadConsultaContratacionTemporalDesarrollo){
		"actor": func(c *capacidadConsultaContratacionTemporalDesarrollo) { c.principal.ID = "per_otro" },
		"certificado": func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.principal.Attributes = maps.Clone(c.principal.Attributes)
			c.principal.Attributes["certificate_sha256"] = strings.Repeat("0", 64)
		},
		"certificado vencido": func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.certificadoValidoHasta = e.a.reloj.Ahora().Add(-time.Second)
		},
		"certificado futuro": func(c *capacidadConsultaContratacionTemporalDesarrollo) {
			c.certificadoVerificadoEn = e.a.reloj.Ahora().Add(time.Hour)
		},
		"ruta":   func(c *capacidadConsultaContratacionTemporalDesarrollo) { c.ruta = httpinterno.RutaAltaSolicitudes },
		"método": func(c *capacidadConsultaContratacionTemporalDesarrollo) { c.metodo = http.MethodGet },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := original
			alterar(&c)
			ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, c)
			if _, err := e.a.AutorizarLecturaOriginalCT(ctx, e.s, e.ref); err == nil {
				t.Fatal("concedió")
			}
		})
	}
	if e.g.registros != 0 {
		t.Fatal("rechazo posterior al registro")
	}
	s := e.s
	s.OrganizacionRef = "organizacion:otra"
	if _, err := e.a.AutorizarLecturaOriginalCT(e.ctx, s, e.ref); err == nil {
		t.Fatal("organización ajena")
	}
	if _, err := e.a.AutorizarLecturaOriginalCT(context.WithValue(context.Background(), claveContextoDocumentos{}, contextoDocumentos{}), e.s, e.ref); err == nil {
		t.Fatal("contexto Documentos sustituyó CT")
	}
	pdf := e.pdf
	pdf.TipoRef, _ = e.a.tipos.ResolverTipoOriginalCT(e.ctx, "resolucion")
	if _, _, err := e.a.PrepararCustodiaOriginalCT(e.ctx, e.s, pdf, e.ref); err == nil {
		t.Fatal("tipo de otro documento")
	}
	e.g.revocarEnCAS = true
	if _, err := e.a.AutorizarLecturaOriginalCT(e.ctx, e.s, e.ref); err == nil || e.g.registros != 0 {
		t.Fatal("CAS obsoleto concedió")
	}
}

func TestOriginalDocumentosCTPredicadoLigaMotivoAccionRecursoYVinculo(t *testing.T) {
	e := nuevoEscenarioOriginalAutorizacionCT(t)
	recurso, _ := docports.RecursoV3(docports.AccionDescargar, e.ref, []byte(`{"version":7}`))
	d := core.DatosSolicitudAutorizacionLigadaV3{Accion: docports.AccionDescargar, Finalidad: e.a.operaciones[docports.AccionDescargar].finalidad,
		ReferenciaMotivo: e.g.motivo, Recurso: recurso, VinculoAutenticacionActor: e.g.perfil.contexto.Vinculo}
	ctx := context.WithValue(e.ctx, claveSolicitudOriginalDocumentosCT{}, solicitudOriginalDocumentosCT{datos: d})
	if !solicitudAutorizacionOriginalDocumentosCTValida(ctx, d) {
		t.Fatal("exacta rechazada")
	}
	for nombre, alterar := range map[string]func(*core.DatosSolicitudAutorizacionLigadaV3){
		"acción":    func(d *core.DatosSolicitudAutorizacionLigadaV3) { d.Accion = docports.AccionCustodiarFirmado },
		"finalidad": func(d *core.DatosSolicitudAutorizacionLigadaV3) { d.Finalidad = docports.FinalidadCustodiarFirmado },
		"motivo": func(d *core.DatosSolicitudAutorizacionLigadaV3) {
			d.ReferenciaMotivo = e.a.soporte.motivoPropuestaCobertura
		},
		"recurso": func(d *core.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Referencia = "ref:" + strings.Repeat("e", 64)
		},
		"huella": func(d *core.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = map[string]string{"preimagen_sha256": strings.Repeat("e", 64)}
		},
		"vínculo": func(d *core.DatosSolicitudAutorizacionLigadaV3) {
			d.VinculoAutenticacionActor = core.VinculoAutenticacionActorV2{}
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			otro := d
			alterar(&otro)
			if solicitudAutorizacionOriginalDocumentosCTValida(ctx, otro) {
				t.Fatal("material ajeno aceptado")
			}
		})
	}
}

func TestOriginalDocumentosCTLecturaFirmadoAnteriorExacta(t *testing.T) {
	e := nuevoEscenarioOriginalAutorizacionCT(t)
	q := ctports.SolicitudPDFFirmaAnterior{OrganizacionRef: e.s.OrganizacionRef, ExpedienteRef: e.s.ExpedienteRef, Documento: e.s.Documento,
		FirmaRef: "firma:ct:prueba", ReciboRef: "recibo:ct:prueba", DocumentoRef: "ref:" + strings.Repeat("a", 64), DocumentoVersion: 2, DocumentoHuella: strings.Repeat("b", 64)}
	consulta, err := e.a.AutorizarLecturaPDFFirmaAnteriorCT(e.ctx, q)
	if err != nil || consulta.DocumentoID != q.DocumentoRef || consulta.Version != q.DocumentoVersion || e.g.registros != 1 {
		t.Fatalf("descarga PDF anterior: %v", err)
	}
	if consulta.DocumentoID == e.ref {
		t.Fatal("regeneró original en lugar de PDF previo")
	}
	q.DocumentoHuella = ""
	if _, err = e.a.AutorizarLecturaPDFFirmaAnteriorCT(e.ctx, q); err == nil || e.g.registros != 1 {
		t.Fatal("antecedente incompleto concedió")
	}
}
