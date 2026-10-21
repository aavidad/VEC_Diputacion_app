package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	importacionapp "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
	importacion "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func solicitudIdentidadDocumentoPrueba(t *testing.T) ports.SolicitudIdentidadDocumentoAceptacionCT {
	t.Helper()
	a := solicitudPersonaAceptacionPrueba(t).ActorConfiable
	return ports.SolicitudIdentidadDocumentoAceptacionCT{Selector: ports.SelectorAceptacionNativaCT{
		OrganizacionRef: "organizacion:demo", ExpedienteRef: "expediente:demo", ClaveCT: "clave:demo",
		UnidadRef: "unidad:demo", CategoriaRef: "categoria:demo", NecesidadRef: "necesidad:demo",
		BolsaRef: "bolsa:demo", LlamamientoRef: "llamamiento:demo",
		ReciboEmisionRef: "recibo:emision:demo", ReciboRespuestaRef: "recibo:respuesta:demo",
	}, ActorConfiable: a}
}

func relojIdentidadDocumentoPrueba() time.Time {
	return time.Date(2026, 9, 30, 11, 0, 0, 1000, time.UTC)
}

func materialIdentidadDocumentoPrueba(t *testing.T, p ports.PreparacionIdentidadDocumentoAceptacionCT) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	c := p.Solicitud.ActorConfiable.Resultado
	h, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	instante := relojIdentidadDocumentoPrueba()
	x, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", strings.Repeat("c", 64), strings.Repeat("d", 64), c.RegistroContextoRef, c.HuellaSHA256,
		ports.AccionIdentidadDocumentoAceptacionCT, p.Recurso.Referencia, h, ports.AudienciaIdentidadDocumentoAceptacionCT,
		instante.Add(-time.Microsecond), instante.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	m, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), x,
		[]byte("d"), []byte("m"), c.RepresentacionCanonica, c.Contexto.Instantanea.PersonaVersion,
		c.Contexto.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type proveedorIdentidadDocumentoPrueba struct{ t *testing.T }

func (p proveedorIdentidadDocumentoPrueba) AutorizarIdentidadDocumentoAceptacionCT(_ context.Context, q ports.PreparacionIdentidadDocumentoAceptacionCT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return materialIdentidadDocumentoPrueba(p.t, q), nil
}

type repositorioIdentidadDocumentoPrueba struct {
	fuente *ports.FuenteIdentidadDocumentoAceptacionCT
	estado string
}

func (r repositorioIdentidadDocumentoPrueba) ConsultarFuenteIdentidadDocumentoAceptacionCT(_ context.Context, o ports.OrdenIdentidadDocumentoAceptacionCT) (ports.LecturaFuenteIdentidadDocumentoAceptacionCT, error) {
	x := o.Material.ResumenCapacidad()
	return ports.LecturaFuenteIdentidadDocumentoAceptacionCT{Estado: r.estado, Fuente: r.fuente,
		Evidencia: ports.EvidenciaConsultaPersonaAceptacionCT{DecisionRef: x.DecisionRef(),
			ConsumoHuellaSHA256: strings.Repeat("e", 64), AuditoriaRef: "auditoria:demo", ConsultadaEn: relojIdentidadDocumentoPrueba()}}, nil
}

type recuperadorIdentidadDocumentoPrueba struct {
	lote   importacion.LoteValidado
	estado importacionapp.EstadoImportacion
	err    error
}

func (r recuperadorIdentidadDocumentoPrueba) RecuperarLote(_ context.Context, huella, categoria string) (importacion.LoteValidado, importacionapp.EstadoImportacion, bool, error) {
	if r.err != nil {
		return importacion.LoteValidado{}, importacionapp.EstadoImportacion{}, false, r.err
	}
	return r.lote, r.estado, r.lote.Acta.HuellaFicheroSHA256 == huella && r.lote.Acta.CategoriaRef == categoria, nil
}

func loteIdentidadDocumentoPrueba(filas ...importacion.FilaAceptada) (importacion.LoteValidado, importacionapp.EstadoImportacion) {
	huella := strings.Repeat("a", 64)
	categoria := "categoria:demo"
	ref := importacion.ReferenciaContexto(huella, categoria)
	acta := importacion.ActaImportacion{CategoriaRef: categoria, BolsaRef: "bolsa:demo",
		ActaRef: "acta:importacion-convoca:" + ref, ImportacionRef: "importacion:convoca:" + ref,
		HuellaFicheroSHA256: huella, FicheroCustodiadoRef: "custodia:demo", NombreFichero: "lista_demo.xls",
		ActorRef: "actor:rrhh:demo", RegistradaEn: time.Date(2026, 9, 29, 9, 0, 0, 0, time.UTC),
		Esquema: importacion.EsquemaResumenPersona, FilasLeidas: len(filas), FilasAceptadas: len(filas),
		Procedencia: importacion.NuevaProcedenciaNoAutoritativa()}
	lote := importacion.LoteValidado{Acta: acta, Aceptadas: filas}
	estado := importacionapp.EstadoImportacion{Acta: acta, EstadoConciliacion: importacionapp.EstadoConciliacionPendiente,
		EstadoStaging: importacionapp.EstadoStagingDisponible, PoliticaRetencionRef: "politica:demo",
		PoliticaRetencionVersion: 1, ConservarStagingHasta: acta.RegistradaEn.Add(365 * 24 * time.Hour), Version: 1}
	return lote, estado
}

func filaIdentidadDocumentoPrueba(numero int, doc, apellido, nombre string) importacion.FilaAceptada {
	return importacion.FilaAceptada{Numero: numero, Esquema: importacion.EsquemaResumenPersona,
		Identidad: importacion.IdentidadEnmascarada{Documento: doc, PrimerApellido: apellido, Nombre: nombre},
		Turno:     "Libre", Resumen: &importacion.ResumenPersona{Experiencia: "1", Formacion: "1", Total: "2"}}
}

func TestIdentidadDocumentoAceptacionCTVectorCanonicoYContexto(t *testing.T) {
	s := solicitudIdentidadDocumentoPrueba(t)
	p, err := PrepararIdentidadDocumentoAceptacionCT(s)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.bolsa.aceptacion-nativa-ct.identidad-documento.consulta.v1","organizacion_ref":"organizacion:demo","expediente_ref":"expediente:demo","clave_ct":"clave:demo","unidad_ref":"unidad:demo","categoria_ref":"categoria:demo","necesidad_ref":"necesidad:demo","bolsa_ref":"bolsa:demo","llamamiento_ref":"llamamiento:demo","recibo_emision_ref":"recibo:emision:demo","recibo_respuesta_ref":"recibo:respuesta:demo"}`
	contexto, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || string(p.MaterialCanonico) != esperado || p.Recurso.Atributos["material_sha256"] != "3df28a8e095cba2ca0dd1ccb8e98e35fe06fb792e69834bb6431f400e8e23fdc" || contexto != "3486e5f48107b6c674a9b7a26e7531ef16ae3d671ce8cf9cd7f2bef30786add7" {
		t.Fatal("vector Go/SQL divergente")
	}
	for _, cambiar := range []func(*ports.SelectorAceptacionNativaCT){
		func(x *ports.SelectorAceptacionNativaCT) { x.UnidadRef = "unidad:ajena" },
		func(x *ports.SelectorAceptacionNativaCT) { x.ExpedienteRef = "expediente:ajeno" },
		func(x *ports.SelectorAceptacionNativaCT) { x.ReciboRespuestaRef = "recibo:respuesta:ajeno" },
	} {
		otra := s
		cambiar(&otra.Selector)
		p2, err := PrepararIdentidadDocumentoAceptacionCT(otra)
		if err != nil || bytes.Equal(p.MaterialCanonico, p2.MaterialCanonico) || p.Recurso.Atributos["material_sha256"] == p2.Recurso.Atributos["material_sha256"] {
			t.Fatal("selector ajeno sin ligadura")
		}
	}
	m := materialIdentidadDocumentoPrueba(t, p)
	for _, cambiar := range []func(*ports.SelectorAceptacionNativaCT){
		func(x *ports.SelectorAceptacionNativaCT) { x.UnidadRef = "unidad:ajena" },
		func(x *ports.SelectorAceptacionNativaCT) { x.ExpedienteRef = "expediente:ajeno" },
		func(x *ports.SelectorAceptacionNativaCT) { x.ReciboRespuestaRef = "recibo:respuesta:ajeno" },
	} {
		otra := s
		cambiar(&otra.Selector)
		if err := ValidarOrdenIdentidadDocumentoAceptacionCT(ports.OrdenIdentidadDocumentoAceptacionCT{Solicitud: otra, Material: m}, relojIdentidadDocumentoPrueba()); !errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTDenegada) {
			t.Fatal("capacidad reutilizada en selector ajeno")
		}
	}
}

func TestIdentidadDocumentoAceptacionCTSeleccionYCierreDeFuente(t *testing.T) {
	lote, estado := loteIdentidadDocumentoPrueba(
		filaIdentidadDocumentoPrueba(2, "***1111**", "Uno", "Ana"),
		filaIdentidadDocumentoPrueba(3, "***4567**", "Dos", "Bea"))
	if lote.Validar() != nil || estado.Validar() != nil {
		t.Fatal("fixture protegida inválida")
	}
	s := solicitudIdentidadDocumentoPrueba(t)
	f := &ports.FuenteIdentidadDocumentoAceptacionCT{ActaRef: lote.Acta.ActaRef, FilaNumero: 3,
		InstantaneaRef: "instantanea:demo", VersionInstantanea: 4,
		HuellaInstantaneaSHA256: strings.Repeat("b", 64), BolsaRef: s.Selector.BolsaRef,
		HuellaListadoSHA256: lote.Acta.HuellaFicheroSHA256,
		ReciboEmisionRef:    s.Selector.ReciboEmisionRef, ReciboRespuestaRef: s.Selector.ReciboRespuestaRef}
	r := repositorioIdentidadDocumentoPrueba{fuente: f, estado: "acreditado"}
	rec := recuperadorIdentidadDocumentoPrueba{lote: lote, estado: estado}
	servicio, err := NuevoServicioIdentidadDocumentoAceptacionCT(proveedorIdentidadDocumentoPrueba{t}, r, rec, relojIdentidadDocumentoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	salida, err := servicio.ConsultarIdentidadDocumentoAceptacionCT(context.Background(), s)
	if err != nil || salida.Estado != "acreditado" || salida.Identidad == nil || salida.Identidad.Nombre != "Bea" || salida.Identidad.Apellidos != "Dos" || salida.Identidad.DocumentoEnmascarado != "***4567**" || salida.Identidad.VersionFuente != 4 {
		t.Fatalf("selección de fila: %v", err)
	}
	for _, caso := range []string{"ambigua", "ausente", "mascara", "fuente"} {
		t.Run(caso, func(t *testing.T) {
			copia, estadoCopia := lote, estado
			fuente := *f
			fallo := error(nil)
			switch caso {
			case "ambigua":
				copia, estadoCopia = loteIdentidadDocumentoPrueba(filaIdentidadDocumentoPrueba(2, "***4567**", "Dos", "Bea"), filaIdentidadDocumentoPrueba(3, "***4567**", "Dos", "Bea"))
			case "ausente":
				fuente.FilaNumero = 8
			case "mascara":
				copia.Aceptadas = append([]importacion.FilaAceptada(nil), lote.Aceptadas...)
				copia.Aceptadas[1].Identidad.Documento = "invalido"
			case "fuente":
				fallo = errors.New("origen inaccesible")
			}
			svc, _ := NuevoServicioIdentidadDocumentoAceptacionCT(proveedorIdentidadDocumentoPrueba{t}, repositorioIdentidadDocumentoPrueba{fuente: &fuente, estado: "acreditado"}, recuperadorIdentidadDocumentoPrueba{lote: copia, estado: estadoCopia, err: fallo}, relojIdentidadDocumentoPrueba)
			v, e := svc.ConsultarIdentidadDocumentoAceptacionCT(context.Background(), s)
			if v.Identidad != nil || (e == nil && v.Estado != "pendiente") {
				t.Fatalf("%s dejó pasar identidad: %v", caso, e)
			}
			if fallo != nil && (!errors.Is(e, fallo) || !errors.Is(e, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible)) {
				t.Fatalf("causa de recuperación perdida: %v", e)
			}
		})
	}
}

func TestIdentidadDocumentoAceptacionCTErrorDesconocidoConservaCausa(t *testing.T) {
	causa := errors.New("dependencia privada")
	err := errorIdentidadDocumentoAceptacionCT(context.Background(), causa)
	if !errors.Is(err, causa) || !errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTNoDisponible) {
		t.Fatal("causa privada perdida para diagnóstico interno")
	}
}

type emisorIdentidadDocumentoPrueba struct {
	emisorNominalPersonaAceptacionPrueba
}

// El doble prueba el protocolo nominal Go; no verifica COSE ni escribe SQL.
func (e *emisorIdentidadDocumentoPrueba) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3, c core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	d, confirmacion, exportador, err := e.emisorNominalPersonaAceptacionPrueba.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	if err != nil || exportador == nil {
		return d, confirmacion, exportador, err
	}
	m, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil {
		e.t.Fatal(err)
	}
	datos, _ := s.Datos()
	dh, _ := core.HuellaSHA256DecisionAutorizacionV3(d)
	mh, _ := core.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	rh, _ := datos.Recurso.HuellaContextoAutorizacionSHA256()
	anterior := m.ResumenCapacidad()
	x, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(anterior.DecisionRef(), dh, mh,
		c.RegistroContextoRef, c.HuellaSHA256, datos.Accion, datos.Recurso.Referencia, rh,
		ports.AudienciaIdentidadDocumentoAceptacionCT, anterior.EmitidaEn(), anterior.ExpiraEn())
	if err != nil {
		e.t.Fatal(err)
	}
	m, err = vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(m.CapacidadCanonica(), x,
		m.DecisionCanonica(), m.MotivoCanonico(), c.RepresentacionCanonica, m.PersonaVersion(), m.PerfilVersion(),
		m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
	if err != nil {
		e.t.Fatal(err)
	}
	return d, confirmacion, exportadorBorradorPrueba{material: m}, nil
}

func TestIdentidadDocumentoAceptacionCTProveedorExigeConcesionPropia(t *testing.T) {
	e := &emisorIdentidadDocumentoPrueba{emisorNominalPersonaAceptacionPrueba{t: t, campos: camposIdentidadDocumentoAceptacionCT}}
	p, err := NuevoProveedorNominalIdentidadDocumentoAceptacionCT(e, motivoBorradorPrueba(),
		func(context.Context) (core.ReferenciaCorrelacionAutorizacionV2, error) {
			return correlacionBorradorPrueba(t), nil
		},
		relojIdentidadDocumentoPrueba)
	if err != nil {
		t.Fatal(err)
	}
	s := solicitudIdentidadDocumentoPrueba(t)
	entrada, _ := PrepararIdentidadDocumentoAceptacionCT(s)
	m, err := p.AutorizarIdentidadDocumentoAceptacionCT(context.Background(), entrada)
	datos, _ := e.ultima.Datos()
	if err != nil || datos.Accion != ports.AccionIdentidadDocumentoAceptacionCT ||
		datos.Finalidad != ports.FinalidadIdentidadDocumentoAceptacionCT ||
		datos.Recurso.Tipo != ports.TipoIdentidadDocumentoAceptacionCT ||
		m.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaIdentidadDocumentoAceptacionCT {
		t.Fatalf("concesión documental incorrecta: %v", err)
	}
	entrada.Recurso.Referencia = "recibo:ajeno"
	if _, err := p.AutorizarIdentidadDocumentoAceptacionCT(context.Background(), entrada); !errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTInvalida) {
		t.Fatal("recurso inyectado admitido")
	}
	entrada, _ = PrepararIdentidadDocumentoAceptacionCT(s)
	entrada.Recurso.Atributos["nonce_ajeno"] = "x"
	if _, err := p.AutorizarIdentidadDocumentoAceptacionCT(context.Background(), entrada); !errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTInvalida) {
		t.Fatal("atributo ajeno admitido")
	}
	entrada, _ = PrepararIdentidadDocumentoAceptacionCT(s)
	for _, campos := range [][]string{{"aceptacion", "persona", "vinculo"}, {"nombre", "documento_enmascarado", "version_fuente"}} {
		e.campos = campos
		if _, err := p.AutorizarIdentidadDocumentoAceptacionCT(context.Background(), entrada); !errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTDenegada) {
			t.Fatal("proyección ajena admitida")
		}
	}
	e.campos = camposIdentidadDocumentoAceptacionCT
	e.denegada = true
	if _, err := p.AutorizarIdentidadDocumentoAceptacionCT(context.Background(), entrada); !errors.Is(err, ports.ErrIdentidadDocumentoAceptacionCTDenegada) {
		t.Fatal("perfil sin concesión admitido")
	}
}
