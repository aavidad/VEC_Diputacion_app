package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	ctapplication "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
	docautorizacion "vec-diputacion-granada/internal/vec/documentos/adapters/autorizacion"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func solicitudOriginalFirmablePrueba() (puertosvec.SolicitudOriginalFirmableCT, string, string) {
	s := puertosvec.SolicitudOriginalFirmableCT{OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef: "expediente:ct:original:001", Documento: "informe_definitivo", OriginalVersion: 7}
	ref := almacencanonico.IdentidadOriginalCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		Documento: s.Documento, Version: s.OriginalVersion}.Referencia()
	expediente, _ := ctapplication.ReferenciaExpedienteDocumentalFormalizacion(s.ExpedienteRef)
	return s, ref, expediente
}

func recursoDocumentosPrueba(tipo, ref string, atributos map[string]string) dominiovec.RecursoAutorizable {
	return dominiovec.RecursoAutorizable{Referencia: ref, ModuloID: "documentos", Tipo: tipo,
		Ambitos: map[string]string{"organizacion_ref": docports.OrganizacionRefV3}, Atributos: atributos}
}

func datosOriginalPrueba(accion, finalidad string, r dominiovec.RecursoAutorizable) dominiovec.DatosSolicitudAutorizacionLigadaV3 {
	return dominiovec.DatosSolicitudAutorizacionLigadaV3{Accion: accion, Finalidad: finalidad,
		ReferenciaMotivo: motivoOriginalFirmableCTDesarrollo(), Recurso: r}
}

func sha256HexPrueba(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// El rol del perfil fijo solo cubre las cuatro operaciones de Documentos sobre
// el original, con los campos exactos de cada consumidor y sin obligaciones.
func TestInstantaneaOriginalFirmableConcedeSoloOperacionesDelOriginal(t *testing.T) {
	i, err := instantaneaOriginalFirmableCTDesarrollo("per_0123456789abcdefghijkl", "perfil:ct:original", time.Now().UTC())
	if err != nil || i.Validar() != nil {
		t.Fatalf("instantánea inválida: %v", err)
	}
	esperadas := map[string][]string{
		docports.AccionDescargar:                         {"contenido", "documento"},
		docports.AccionReservarOriginalFirmable:          {"intento", "reserva"},
		docports.AccionConfirmarOriginalFirmable:         {"documento", "recibo"},
		puertosvec.AccionNegocioEscribirOriginalFirmable: {"evidencia_almacen", "original_firmable.contenido"},
	}
	if len(i.VersionRol.Concesiones) != len(esperadas) {
		t.Fatalf("concesiones: %d", len(i.VersionRol.Concesiones))
	}
	for _, c := range i.VersionRol.Concesiones {
		campos, ok := esperadas[c.Accion]
		if !ok || c.ModuloID != "documentos" || !slices.Equal(c.CamposPermitidos, campos) || len(c.Obligaciones) != 0 ||
			c.GarantiaMinima != dominiovec.AuthAssuranceHigh || len(c.Finalidades) != 1 {
			t.Fatalf("concesión inesperada: %+v", c)
		}
	}
}

func TestPredicadoOriginalFirmableLigadoAlOriginalEnCurso(t *testing.T) {
	s, ref, expediente := solicitudOriginalFirmablePrueba()
	consulta := docports.ConsultaDocumento{DocumentoID: ref, Version: s.OriginalVersion}
	preimagen, err := consulta.PreimagenDescargar()
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := docports.RecursoV3(docports.AccionDescargar, ref, preimagen)
	if err != nil {
		t.Fatal(err)
	}
	e := &esperadoOriginalFirmableCTDesarrollo{documentoRef: ref, expedienteRef: expediente, version: s.OriginalVersion}
	ctx := context.WithValue(context.Background(), claveOriginalFirmableCTDesarrollo{}, e)
	descarga := datosOriginalPrueba(docports.AccionDescargar, finalidadDescargaDocumento, recurso)
	if solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, descarga) {
		t.Fatal("descarga admitida sin preimagen fijada")
	}
	e.fijarPreimagen(docports.AccionDescargar, preimagen)
	if !solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, descarga) {
		t.Fatal("la descarga exacta no se admite")
	}
	if solicitudAutorizacionOriginalFirmableCTDesarrolloValida(context.Background(), descarga) {
		t.Fatal("admitida sin original en curso")
	}
	for nombre, alterar := range map[string]func(*dominiovec.DatosSolicitudAutorizacionLigadaV3){
		"otro documento": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Referencia = "ref:" + strings.Repeat("a", 64)
		},
		"otro motivo": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.ReferenciaMotivo = motivoFirmaDocumentoCTDesarrollo()
		},
		"otra finalidad": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Finalidad = docports.FinalidadOriginalFirmable
		},
		"otro tipo":   func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.Tipo = "documento_firmado" },
		"otro módulo": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) { d.Recurso.ModuloID = "contratacion_temporal" },
		"otra organización": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Ambitos = map[string]string{"organizacion_ref": "organizacion:ajena"}
		},
		"otra preimagen": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Recurso.Atributos = map[string]string{"preimagen_sha256": strings.Repeat("9", 64)}
		},
		"reserva en lectura": func(d *dominiovec.DatosSolicitudAutorizacionLigadaV3) {
			d.Accion, d.Finalidad, d.Recurso.Tipo = docports.AccionReservarOriginalFirmable, docports.FinalidadOriginalFirmable, tipoRecursoOriginalFirmableCT
		},
	} {
		x := descarga
		x.Recurso.Ambitos, x.Recurso.Atributos = maps.Clone(descarga.Recurso.Ambitos), maps.Clone(descarga.Recurso.Atributos)
		alterar(&x)
		if solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, x) {
			t.Errorf("%s: admitida", nombre)
		}
	}

	// Lectura del objeto: solo para la descarga consumida de esta petición.
	lectura := descarga
	lectura.Recurso.Atributos = map[string]string{docautorizacion.AtributoDecisionDescarga: "decision:descarga",
		docautorizacion.AtributoCorrelacionDescarga: "correlacion:descarga", "documento_expediente_ref": expediente,
		"documento_modulo_productor": moduloProductorCustodiaCT, "documento_version": "7"}
	if solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, lectura) {
		t.Fatal("lectura del objeto sin descarga consumida")
	}
	e.decisionDescarga, e.correlacionDescarga = "decision:descarga", "correlacion:descarga"
	if !solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, lectura) {
		t.Fatal("la lectura exacta del objeto no se admite")
	}
	for clave, valor := range map[string]string{docautorizacion.AtributoDecisionDescarga: "decision:otra",
		docautorizacion.AtributoCorrelacionDescarga: "correlacion:otra", "documento_expediente_ref": "ref:" + strings.Repeat("f", 64),
		"documento_modulo_productor": "dietas", "documento_version": "8"} {
		x := lectura
		x.Recurso.Atributos = maps.Clone(lectura.Recurso.Atributos)
		x.Recurso.Atributos[clave] = valor
		if solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, x) {
			t.Errorf("lectura con %s ajeno admitida", clave)
		}
	}

	// Reserva y escritura solo dentro de una custodia, en orden.
	reservaPre := []byte(`{"accion":"documentos.original_firmable.reservar"}`)
	reserva := datosOriginalPrueba(docports.AccionReservarOriginalFirmable, docports.FinalidadOriginalFirmable,
		recursoDocumentosPrueba(tipoRecursoOriginalFirmableCT, ref, map[string]string{"preimagen_sha256": sha256HexPrueba(reservaPre)}))
	e.fijarPreimagen(docports.AccionReservarOriginalFirmable, reservaPre)
	if solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, reserva) {
		t.Fatal("reserva admitida en una lectura")
	}
	w := &esperadoOriginalFirmableCTDesarrollo{documentoRef: ref, expedienteRef: expediente, version: 7, escritura: true,
		tipoRef: "ref:" + strings.Repeat("c", 64), huellaSHA256: strings.Repeat("1", 64)}
	cw := context.WithValue(context.Background(), claveOriginalFirmableCTDesarrollo{}, w)
	w.fijarPreimagen(docports.AccionReservarOriginalFirmable, reservaPre)
	if !solicitudAutorizacionOriginalFirmableCTDesarrolloValida(cw, reserva) {
		t.Fatal("la reserva exacta no se admite")
	}
	escritura := datosOriginalPrueba(puertosvec.AccionNegocioEscribirOriginalFirmable, docports.FinalidadOriginalFirmable,
		recursoDocumentosPrueba(tipoRecursoOriginalFirmableCT, ref, map[string]string{
			"documentos_original_decision_reserva_ref": "decision:reserva", "documentos_original_huella_sha256": w.huellaSHA256,
			"documentos_original_expediente_ref": expediente, "documentos_original_tipo_ref": w.tipoRef,
			"documentos_original_version": "7", puertosvec.AtributoAlmacenEfectoRef: ref}))
	if solicitudAutorizacionOriginalFirmableCTDesarrolloValida(cw, escritura) {
		t.Fatal("escritura del objeto antes de la reserva")
	}
	w.fijarDecisionReserva("decision:reserva")
	if !solicitudAutorizacionOriginalFirmableCTDesarrolloValida(cw, escritura) {
		t.Fatal("la escritura exacta no se admite")
	}
	for clave, valor := range map[string]string{"documentos_original_decision_reserva_ref": "decision:otra",
		"documentos_original_huella_sha256": strings.Repeat("2", 64), "documentos_original_tipo_ref": "ref:" + strings.Repeat("b", 64),
		"documentos_original_version": "8", puertosvec.AtributoAlmacenEfectoRef: "ref:" + strings.Repeat("a", 64)} {
		x := escritura
		x.Recurso.Atributos = maps.Clone(escritura.Recurso.Atributos)
		x.Recurso.Atributos[clave] = valor
		if solicitudAutorizacionOriginalFirmableCTDesarrolloValida(cw, x) {
			t.Errorf("escritura con %s ajeno admitida", clave)
		}
	}
}

// pdpOriginalPrueba emula el PDP: concede solo si el predicado de la ruta
// admite la solicitud con el contexto recibido.
type pdpOriginalPrueba struct {
	err      error
	actor    dominiovec.DatosVinculoAutenticacionActorV2
	pedidas  []string
	material int
}

func (p *pdpOriginalPrueba) solicitarOriginalV3(ctx context.Context, accion, finalidad string, r dominiovec.RecursoAutorizable) (solicitudCustodiaCTDesarrollo, error) {
	p.pedidas = append(p.pedidas, accion)
	if p.err != nil {
		return solicitudCustodiaCTDesarrollo{}, p.err
	}
	if !solicitudAutorizacionOriginalFirmableCTDesarrolloValida(ctx, datosOriginalPrueba(accion, finalidad, r)) {
		return solicitudCustodiaCTDesarrollo{}, errOriginalFirmableCTDenegado
	}
	return solicitudCustodiaCTDesarrollo{actor: p.actor, correlacion: "correlacion:prueba"}, nil
}

func (p *pdpOriginalPrueba) materialOriginalV3(context.Context, solicitudCustodiaCTDesarrollo) (puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.material++
	return puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

func nuevoOriginalPrueba(t *testing.T, pdp *pdpOriginalPrueba) *originalFirmableCTDesarrollo {
	t.Helper()
	// Solo el catálogo v2 reserva los seis tipos del original CT.
	catalogo, err := conservacion.NuevoCatalogoProvisionalV2(relojRutasDietas{})
	if err != nil {
		t.Fatal(err)
	}
	o, err := nuevoOriginalFirmableCTDesarrollo(pdp, nuevoSeudonimizadorAlmacenDesarrollo([32]byte{7}), catalogo, relojRutasDietas{})
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestOriginalFirmableAutorizaLecturaDelOriginalExacto(t *testing.T) {
	pdp := &pdpOriginalPrueba{actor: dominiovec.DatosVinculoAutenticacionActorV2{PrincipalID: "per_prueba", PerfilActivoRef: "perfil:original"}}
	o := nuevoOriginalPrueba(t, pdp)
	s, ref, expediente := solicitudOriginalFirmablePrueba()
	c, err := o.AutorizarLecturaOriginalCT(context.Background(), s, ref)
	if err != nil {
		t.Fatal(err)
	}
	a := c.Autorizacion
	if c.DocumentoID != ref || c.Version != 7 || a.Accion != docports.AccionDescargar || a.Finalidad != finalidadDescargaDocumento ||
		a.RecursoRef != ref || a.AmbitoRef != expediente || a.PrincipalID != "per_prueba" || a.PerfilActivoRef != "perfil:original" ||
		pdp.material != 1 {
		t.Fatalf("consulta inesperada: %+v", c)
	}
	otra := s
	otra.OrganizacionRef = "organizacion:ajena"
	for nombre, caso := range map[string]struct {
		s   puertosvec.SolicitudOriginalFirmableCT
		ref string
	}{"otra referencia": {s, "ref:" + strings.Repeat("a", 64)}, "otra organización": {otra, ref}} {
		if _, err := o.AutorizarLecturaOriginalCT(context.Background(), caso.s, caso.ref); !errors.Is(err, puertosvec.ErrOriginalFirmableCTInvalido) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	pdp.err = docports.ErrCapacidadNoDisponible
	if _, err := o.AutorizarLecturaOriginalCT(context.Background(), s, ref); !errors.Is(err, docports.ErrCapacidadNoDisponible) {
		t.Fatalf("indisponibilidad perdida: %v", err)
	}
}

func preimagenReservaPrueba(e *esperadoOriginalFirmableCTDesarrollo, cambios map[string]any) []byte {
	p := map[string]any{"accion": docports.AccionReservarOriginalFirmable, "id": e.documentoRef, "clave_idempotencia": e.claveIdempotencia,
		"modulo_id": moduloProductorCustodiaCT, "expediente_ref": e.expedienteRef, "tipo_ref": e.tipoRef, "version": e.version,
		"mime": "application/pdf", "tamano": e.tamano, "huella_sha256": e.huellaSHA256}
	maps.Copy(p, cambios)
	b, _ := json.Marshal(p)
	return b
}

func TestOriginalFirmablePreparaCustodiaLigadaAlPDF(t *testing.T) {
	pdp := &pdpOriginalPrueba{actor: dominiovec.DatosVinculoAutenticacionActorV2{PrincipalID: "per_prueba", PerfilActivoRef: "perfil:original"}}
	o := nuevoOriginalPrueba(t, pdp)
	s, ref, expediente := solicitudOriginalFirmablePrueba()
	tipoRef, err := o.politicas.TipoDocumentalRef("contratacion_temporal.borrador.informe_definitivo.v1")
	if err != nil {
		t.Fatal(err)
	}
	pdf := puertosvec.PDFOriginalCT{TipoRef: tipoRef, Contenido: []byte("%PDF-1.7\noriginal\n%%EOF")}
	orden, autoridad, err := o.PrepararCustodiaOriginalCT(context.Background(), s, pdf, ref)
	if err != nil || autoridad == nil {
		t.Fatal(err)
	}
	identidad := almacencanonico.IdentidadOriginalCT{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, Documento: s.Documento, Version: 7}
	if orden.ID != ref || orden.ClaveIdempotencia != identidad.ClaveLogica() || orden.ModuloID != moduloProductorCustodiaCT ||
		orden.ExpedienteRef != expediente || orden.TipoRef != tipoRef || orden.Version != 7 || orden.MIME != "application/pdf" ||
		orden.SolicitudPolitica.Validar() != nil || orden.SolicitudPolitica.TipoDocumentalRef() != tipoRef ||
		orden.SolicitudPolitica.ExpedienteRef() != expediente {
		t.Fatalf("orden inesperada: %+v", orden)
	}
	pdf.Contenido[0] = 'X'
	if orden.Contenido[0] != '%' {
		t.Fatal("la orden comparte el buffer de la fuente")
	}
	ajeno, err := o.politicas.TipoDocumentalRef("contratacion_temporal.resolucion_firmada.v1")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := o.PrepararCustodiaOriginalCT(context.Background(), s, puertosvec.PDFOriginalCT{TipoRef: ajeno, Contenido: orden.Contenido}, ref); !errors.Is(err, puertosvec.ErrOriginalFirmableCTNoDisponible) {
		t.Fatalf("tipo no reservado al original: %v", err)
	}

	// La autoridad devuelta lleva el original al contexto; el emisor solo
	// acepta la preimagen de este PDF y nunca fuera de esa custodia.
	e := autoridad.(autoridadOriginalFirmablePeticionCTDesarrollo).esperado
	ctx := context.WithValue(context.Background(), claveOriginalFirmableCTDesarrollo{}, e)
	if _, err := o.AutorizarOperacionOriginalFirmable(context.Background(), docports.AccionReservarOriginalFirmable,
		preimagenReservaPrueba(e, nil), ref, expediente); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("reserva fuera de la custodia: %v", err)
	}
	for nombre, cambio := range map[string]map[string]any{"otro PDF": {"huella_sha256": strings.Repeat("2", 64)},
		"otro tipo": {"tipo_ref": ajeno}, "otra versión": {"version": 8}, "otra clave": {"clave_idempotencia": "ref:" + strings.Repeat("a", 64)},
		"otro módulo": {"modulo_id": "dietas"}} {
		if _, err := o.AutorizarOperacionOriginalFirmable(ctx, docports.AccionReservarOriginalFirmable,
			preimagenReservaPrueba(e, cambio), ref, expediente); !errors.Is(err, docports.ErrAccesoDenegado) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	if _, err := o.AutorizarOperacionOriginalFirmable(ctx, docports.AccionReservarOriginalFirmable,
		preimagenReservaPrueba(e, nil), ref, expediente); err != nil {
		t.Fatalf("reserva exacta: %v", err)
	}
	confirmacion, _ := json.Marshal(map[string]any{"accion": docports.AccionConfirmarOriginalFirmable, "reserva_ref": "ref:" + strings.Repeat("e", 64),
		"intento_num": 1, "clave_almacen_ref": "ref:" + strings.Repeat("f", 64), "id": ref, "huella_sha256": e.huellaSHA256,
		"objeto": map[string]any{"huella_sha256": e.huellaSHA256}})
	if _, err := o.AutorizarOperacionOriginalFirmable(ctx, docports.AccionConfirmarOriginalFirmable, confirmacion, ref, expediente); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("confirmación sin escritura concedida: %v", err)
	}
	e.fijarIntento("ref:"+strings.Repeat("e", 64), "1", "ref:"+strings.Repeat("f", 64))
	if _, err := o.AutorizarOperacionOriginalFirmable(ctx, docports.AccionConfirmarOriginalFirmable, confirmacion, ref, expediente); err != nil {
		t.Fatalf("confirmación del intento concedido: %v", err)
	}
}

func TestOriginalFirmableConcesionAlmacenSoloDelActorYLaOperacion(t *testing.T) {
	pdp := &pdpOriginalPrueba{actor: dominiovec.DatosVinculoAutenticacionActorV2{PrincipalID: "per_prueba", PerfilActivoRef: "perfil:original"}}
	o := nuevoOriginalPrueba(t, pdp)
	_, ref, expediente := solicitudOriginalFirmablePrueba()
	e := &esperadoOriginalFirmableCTDesarrollo{documentoRef: ref, expedienteRef: expediente, version: 7,
		decisionDescarga: "decision:descarga", correlacionDescarga: "correlacion:descarga"}
	ctx := context.WithValue(context.Background(), claveOriginalFirmableCTDesarrollo{}, e)
	lectura := docautorizacion.SolicitudConcesionAlmacenV3{PrincipalID: "per_prueba", PerfilActivoRef: "perfil:original",
		Accion: puertosvec.AccionNegocioLeerOriginalDocumentoGenerado, Finalidad: finalidadDescargaDocumento,
		Recurso: recursoDocumentosPrueba(tipoRecursoDescargaOriginalCT, ref, map[string]string{
			docautorizacion.AtributoDecisionDescarga: "decision:descarga", docautorizacion.AtributoCorrelacionDescarga: "correlacion:descarga",
			"documento_expediente_ref": expediente, "documento_modulo_productor": moduloProductorCustodiaCT, "documento_version": "7"})}
	if _, err := o.EmitirConcesionAlmacenV3(ctx, lectura); err != nil {
		t.Fatalf("lectura exacta: %v", err)
	}
	if _, err := o.EmitirConcesionAlmacenV3(context.Background(), lectura); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("sin descarga en curso: %v", err)
	}
	otro := lectura
	otro.PerfilActivoRef = "perfil:otro"
	if _, err := o.EmitirConcesionAlmacenV3(ctx, otro); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("otro perfil: %v", err)
	}
	escritura := lectura
	escritura.Accion, escritura.Finalidad = puertosvec.AccionNegocioEscribirOriginalFirmable, docports.FinalidadOriginalFirmable
	if _, err := o.EmitirConcesionAlmacenV3(ctx, escritura); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("escritura en una lectura: %v", err)
	}
}

// El servicio que ve la custodia común lleva al contexto la descarga ya
// autorizada; nada más puede cubrir la concesión de lectura del objeto.
func TestServicioOriginalFirmableSinServicioNoDisponible(t *testing.T) {
	var s servicioDocumentosOriginalCTDesarrollo
	if _, _, err := s.DescargarOriginalConDocumento(context.Background(), docports.ConsultaDocumento{}); !errors.Is(err, docports.ErrCapacidadNoDisponible) {
		t.Fatal(err)
	}
	if _, err := s.CustodiarOriginalFirmable(context.Background(), docports.OrdenCustodiarOriginalFirmable{}, nil); !errors.Is(err, docports.ErrCapacidadNoDisponible) {
		t.Fatal(err)
	}
}

type escenarioOriginalPDP struct {
	pdp       pdpCTOriginalFirmableDesarrollo
	soporte   *soporteAltaContratacionTemporalDesarrollo
	perfil    *perfilFijoCTDesarrollo
	principal dominiovec.Principal
}

func nuevoEscenarioOriginalPDP(t *testing.T) escenarioOriginalPDP {
	t.Helper()
	s, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ahora := s.reloj.Ahora()
	p, err := nuevoPerfilFijoCTDesarrollo(principal, s.contexto, ahora, clavePerfilFijoOriginalFirmableCTDesarrollo,
		[]string{httpinterno.RutaOriginalFirmableCT},
		func(actor, ref string) (dominiovec.InstantaneaAutorizacion, error) {
			return instantaneaOriginalFirmableCTDesarrollo(actor, ref, ahora)
		})
	if err != nil || s.registrarPerfilFijoCTDesarrollo(p) != nil {
		t.Fatal(err)
	}
	p.contextoEsperadoRegistrado = p.contexto.Resultado
	p.sesionOperativa = proveedorSesionOperativaCTPrueba{contexto: p.contexto}
	asignaciones := s.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	asignaciones.asignaciones = map[string]instantaneaPublicadaDesarrollo{p.perfilRef(): {
		instantanea: clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(p.plantilla), actoAsignacion: actoAsignacionPerfilFijoCTDesarrollo}}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(s, s, s, s, s.reloj, seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	return escenarioOriginalPDP{pdp: pdpCTOriginalFirmableDesarrollo{alta: &dependenciasAltaContratacionTemporalDesarrollo{
		soporte: s, autorizador: servicio}}, soporte: s, perfil: p, principal: principal}
}

func (e escenarioOriginalPDP) descarga(t *testing.T, ruta string) (context.Context, dominiovec.RecursoAutorizable) {
	t.Helper()
	s, ref, expediente := solicitudOriginalFirmablePrueba()
	preimagen, err := (docports.ConsultaDocumento{DocumentoID: ref, Version: s.OriginalVersion}).PreimagenDescargar()
	if err != nil {
		t.Fatal(err)
	}
	recurso, err := docports.RecursoV3(docports.AccionDescargar, ref, preimagen)
	if err != nil {
		t.Fatal(err)
	}
	esperado := &esperadoOriginalFirmableCTDesarrollo{documentoRef: ref, expedienteRef: expediente, version: s.OriginalVersion}
	esperado.fijarPreimagen(docports.AccionDescargar, preimagen)
	ctx := contextoRutaCoberturaDesarrolloPrueba(e.soporte, e.principal, ruta)
	return context.WithValue(ctx, claveOriginalFirmableCTDesarrollo{}, esperado), recurso
}

// El PDP de CT consume el perfil fijo del original y deriva los campos de su
// plantilla; otra ruta, un recurso ajeno o un perfil revocado se deniegan, y
// una fuente caída es indisponibilidad, no denegación.
func TestOriginalFirmablePDPConsumePerfilFijoDeLaRuta(t *testing.T) {
	e := nuevoEscenarioOriginalPDP(t)
	ctx, recurso := e.descarga(t, httpinterno.RutaOriginalFirmableCT)
	pedida, err := e.pdp.solicitarOriginalV3(ctx, docports.AccionDescargar, finalidadDescargaDocumento, recurso)
	if err != nil {
		t.Fatal(err)
	}
	restricciones, err := pedida.decision.RestriccionesProyeccionPara(pedida.solicitud)
	if err != nil || !slices.Equal(restricciones.CamposPermitidos, []string{"contenido", "documento"}) {
		t.Fatalf("campos derivados: %v %v", restricciones.CamposPermitidos, err)
	}
	if pedida.actor.PerfilActivoRef != e.perfil.perfilRef() {
		t.Fatal("decisión con otro perfil")
	}
	if motivo, ok := e.soporte.motivoAutorizacionParaRuta(httpinterno.RutaOriginalFirmableCT); !ok || motivo != motivoOriginalFirmableCTDesarrollo() {
		t.Fatal("motivo de la ruta del original")
	}

	// La reserva lleva los campos en el orden canónico de la atestación
	// (intento, reserva). AD3-158 compara hoy contra ["reserva","intento"]:
	// la reserva quedará rechazada en SQL hasta que una migración lo corrija.
	pre := []byte(`{"accion":"documentos.original_firmable.reservar"}`)
	w := &esperadoOriginalFirmableCTDesarrollo{documentoRef: recurso.Referencia, escritura: true}
	w.fijarPreimagen(docports.AccionReservarOriginalFirmable, pre)
	cw := context.WithValue(contextoRutaCoberturaDesarrolloPrueba(e.soporte, e.principal, httpinterno.RutaOriginalFirmableCT), claveOriginalFirmableCTDesarrollo{}, w)
	reserva, err := e.pdp.solicitarOriginalV3(cw, docports.AccionReservarOriginalFirmable, docports.FinalidadOriginalFirmable,
		recursoDocumentosPrueba(tipoRecursoOriginalFirmableCT, recurso.Referencia, map[string]string{"preimagen_sha256": sha256HexPrueba(pre)}))
	if err != nil {
		t.Fatal(err)
	}
	if r, err := reserva.decision.RestriccionesProyeccionPara(reserva.solicitud); err != nil || !slices.Equal(r.CamposPermitidos, []string{"intento", "reserva"}) {
		t.Fatalf("campos de la reserva: %v %v", r.CamposPermitidos, err)
	}

	otraRuta, _ := e.descarga(t, httpinterno.RutaFirmaDocumento)
	if _, err := e.pdp.solicitarOriginalV3(otraRuta, docports.AccionDescargar, finalidadDescargaDocumento, recurso); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("otra ruta: %v", err)
	}
	ajeno := recurso
	ajeno.Referencia = "ref:" + strings.Repeat("a", 64)
	if _, err := e.pdp.solicitarOriginalV3(ctx, docports.AccionDescargar, finalidadDescargaDocumento, ajeno); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("otro documento: %v", err)
	}
	if _, err := e.pdp.solicitarOriginalV3(ctx, docports.AccionReservarOriginalFirmable, docports.FinalidadOriginalFirmable, recurso); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("reserva en una lectura: %v", err)
	}

	a := e.soporte.autoridadAsignaciones.(*autoridadAsignacionesContratacionTemporalDesarrolloPrueba)
	e.soporte.autoridadAsignaciones = fuentePerfilFirmaCaidaPrueba{a}
	if _, err := e.pdp.solicitarOriginalV3(ctx, docports.AccionDescargar, finalidadDescargaDocumento, recurso); !errors.Is(err, docports.ErrCapacidadNoDisponible) ||
		errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("fuente caída: %v", err)
	}
	e.soporte.autoridadAsignaciones = a
	publicada := a.asignaciones[e.perfil.perfilRef()]
	ahora := e.soporte.reloj.Ahora()
	publicada.instantanea.AsignacionPerfil.Estado = dominiovec.EstadoAsignacionPerfilRevocada
	publicada.instantanea.AsignacionPerfil.RevocadaEn, publicada.instantanea.AsignacionPerfil.RevocadaPor, publicada.instantanea.AsignacionPerfil.RevocacionRef = ahora, "revocador:prueba", "revocacion:prueba"
	a.asignaciones[e.perfil.perfilRef()] = publicada
	if _, err := e.pdp.solicitarOriginalV3(ctx, docports.AccionDescargar, finalidadDescargaDocumento, recurso); !errors.Is(err, docports.ErrAccesoDenegado) {
		t.Fatalf("perfil revocado: %v", err)
	}
}
