package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Este archivo prueba exclusivamente el cierre seguro del contrato V1. La
// autorizacion positiva pertenece al camino V2, que relee la situacion
// operativa actual y usa componentes atestados por rol.

type catalogoFormatosLegacyPrueba struct {
	descriptores []ports.DescriptorFormatoDocumental
	err          error
	llamadas     int
}

func (c *catalogoFormatosLegacyPrueba) BuscarDescriptoresFormatoDocumental(
	_ context.Context,
	_ ports.ConsultaFormatoDocumental,
) ([]ports.DescriptorFormatoDocumental, error) {
	c.llamadas++
	return append([]ports.DescriptorFormatoDocumental(nil), c.descriptores...), c.err
}

type renderizadorLegacyPrueba struct {
	perfil   domain.ReferenciaPerfilDocumental
	digest   string
	conector domain.ReferenciaConectorDocumental
}

func (r *renderizadorLegacyPrueba) PerfilDocumental() domain.ReferenciaPerfilDocumental {
	return r.perfil
}

func (r *renderizadorLegacyPrueba) DigestPerfilSHA256() string { return r.digest }

func (r *renderizadorLegacyPrueba) ConectorDocumental() domain.ReferenciaConectorDocumental {
	return r.conector
}

func (*renderizadorLegacyPrueba) Renderizar(
	context.Context,
	domain.ContenidoDocumento,
) ([]byte, error) {
	return []byte("salida que nunca debe producirse"), nil
}

func (*renderizadorLegacyPrueba) ValidarSalida(context.Context, []byte) error { return nil }

type registroRenderizadoresLegacyPrueba struct {
	candidatos []ports.RenderizadorDocumentalPorPerfil
	llamadas   int
}

func (r *registroRenderizadoresLegacyPrueba) BuscarRenderizadoresDocumentales(
	_ context.Context,
	_ domain.ReferenciaPerfilDocumental,
	_ domain.ReferenciaConectorDocumental,
) ([]ports.RenderizadorDocumentalPorPerfil, error) {
	r.llamadas++
	return append([]ports.RenderizadorDocumentalPorPerfil(nil), r.candidatos...), nil
}

func TestConstructorPerfilLegacySiempreDeniegaAutoridadPositiva(t *testing.T) {
	identidad, referencia, capacidades := parametrosPerfilLegacyPrueba(t)

	for _, estado := range []domain.EstadoPerfilDocumental{
		domain.EstadoPerfilDocumentalVigente,
		domain.EstadoPerfilDocumentalRetirado,
	} {
		t.Run(string(estado), func(t *testing.T) {
			perfil, err := domain.NuevoPerfilFormatoDocumental(
				referencia, identidad, "application/pdf", "pdf", "binario", estado, capacidades,
			)
			if !errors.Is(err, domain.ErrPerfilFormatoDocumentalInvalido) ||
				perfil.Validar() == nil || perfil.Estado() != "" {
				t.Fatalf("el constructor legacy concedio autoridad: %#v, %v", perfil, err)
			}
		})
	}
}

func TestResolucionLegacyCierraInclusoConPerfilV2YCatalogoExactos(t *testing.T) {
	consulta, descriptor, renderizador := escenarioResolucionLegacyPrueba(t)
	catalogo := &catalogoFormatosLegacyPrueba{
		descriptores: []ports.DescriptorFormatoDocumental{descriptor},
	}
	registro := &registroRenderizadoresLegacyPrueba{
		candidatos: []ports.RenderizadorDocumentalPorPerfil{renderizador},
	}
	servicio, err := NuevoServicioResolucionFormatoDocumental(catalogo, registro)
	if err != nil {
		t.Fatal(err)
	}

	resultado, err := servicio.Resolver(context.Background(), consulta)
	if !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) ||
		errors.Is(err, ports.ErrFormatoDocumentalNoResuelto) {
		t.Fatalf("el contrato V1 no cerro uniformemente: %#v, %v", resultado, err)
	}
	if catalogo.llamadas != 1 || registro.llamadas != 0 {
		t.Fatalf("el cierre V1 alcanzo ejecutores: catalogo=%d registro=%d", catalogo.llamadas, registro.llamadas)
	}
	if _, err := resultado.Descriptor(); !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) {
		t.Fatalf("el resultado cero expuso descriptor: %v", err)
	}
	if _, err := resultado.Renderizador(); !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) {
		t.Fatalf("el resultado cero expuso ejecutor: %v", err)
	}
	if _, err := resultado.Evidencia(); !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) {
		t.Fatalf("el resultado cero expuso evidencia: %v", err)
	}
}

func TestResolucionLegacyValidaAntesDeConsultarYSaneaFallos(t *testing.T) {
	catalogo := &catalogoFormatosLegacyPrueba{}
	servicio, err := NuevoServicioResolucionFormatoDocumental(
		catalogo,
		&registroRenderizadoresLegacyPrueba{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := servicio.Resolver(context.Background(), ports.ConsultaFormatoDocumental{}); !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) {
		t.Fatalf("consulta invalida no cerro: %v", err)
	}
	if catalogo.llamadas != 0 {
		t.Fatalf("la consulta invalida alcanzo el catalogo %d veces", catalogo.llamadas)
	}

	consulta, _, _ := escenarioResolucionLegacyPrueba(t)
	const secreto = "postgres://usuario:clave@catalogo-interno"
	catalogo.err = errors.New(secreto)
	_, err = servicio.Resolver(context.Background(), consulta)
	if !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) ||
		errors.Is(err, ports.ErrCatalogoFormatosDocumentalesNoDisponible) ||
		strings.Contains(err.Error(), secreto) {
		t.Fatalf("el fallo del adaptador no se saneo: %v", err)
	}

	if _, err := RestaurarEvidenciaResolucionFormatoDocumental(
		DatosEvidenciaResolucionFormatoDocumental{},
	); !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) {
		t.Fatalf("una evidencia V1 vacia fue restaurada: %v", err)
	}
}

type registroMarcadoresLegacyPrueba struct{ llamadas int }

func (r *registroMarcadoresLegacyPrueba) BuscarMarcadoresMetadatoInstitucional(
	context.Context,
	domain.ReferenciaPerfilDocumental,
	domain.ReferenciaConectorDocumental,
) ([]ports.MarcadorMetadatoInstitucionalDocumental, error) {
	r.llamadas++
	return nil, nil
}

type verificadorSemanticoLegacyPrueba struct{ llamadas int }

func (v *verificadorSemanticoLegacyPrueba) VerificarEquivalenciaSemantica(
	context.Context,
	domain.PerfilFormatoDocumental,
	[]byte,
	[]byte,
) error {
	v.llamadas++
	return nil
}

func TestMetadatoLegacyCierraAntesDeConsultarComponentes(t *testing.T) {
	solicitud := solicitudMetadatoLegacyPrueba(t)
	if solicitud.Perfil.Validar() != nil || solicitud.Perfil.Estado() != "" {
		t.Fatal("el escenario no usa un perfil V2 valido y sin estado incrustado")
	}
	if !errors.Is(solicitud.Validar(), ports.ErrMetadatoInstitucionalDocumentalInvalido) {
		t.Fatal("la solicitud legacy obtuvo autoridad a partir de un perfil V2")
	}

	registro := &registroMarcadoresLegacyPrueba{}
	verificador := &verificadorSemanticoLegacyPrueba{}
	servicio, err := NuevoServicioMetadatoInstitucionalDocumental(registro, verificador)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := servicio.Incorporar(context.Background(), solicitud)
	if !errors.Is(err, ErrMetadatoInstitucionalNoIncorporado) || len(resultado.Contenido) != 0 {
		t.Fatalf("el metadato V1 no cerro: %+v, %v", resultado, err)
	}
	if registro.llamadas != 0 || verificador.llamadas != 0 {
		t.Fatalf("el cierre alcanzo componentes: registro=%d verificador=%d", registro.llamadas, verificador.llamadas)
	}
}

func TestConstructoresServiciosLegacyRechazanNilTipado(t *testing.T) {
	var catalogo *catalogoFormatosLegacyPrueba
	var renderizadores *registroRenderizadoresLegacyPrueba
	var marcadores *registroMarcadoresLegacyPrueba
	var verificador *verificadorSemanticoLegacyPrueba

	if servicio, err := NuevoServicioResolucionFormatoDocumental(
		catalogo,
		&registroRenderizadoresLegacyPrueba{},
	); servicio != nil || !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) {
		t.Fatalf("catalogo nil tipado aceptado: %#v, %v", servicio, err)
	}
	if servicio, err := NuevoServicioResolucionFormatoDocumental(
		&catalogoFormatosLegacyPrueba{},
		renderizadores,
	); servicio != nil || !errors.Is(err, ErrResolucionFormatoDocumentalCerrada) {
		t.Fatalf("registro nil tipado aceptado: %#v, %v", servicio, err)
	}
	if servicio, err := NuevoServicioMetadatoInstitucionalDocumental(
		marcadores,
		&verificadorSemanticoLegacyPrueba{},
	); servicio != nil || !errors.Is(err, ErrMetadatoInstitucionalNoIncorporado) {
		t.Fatalf("marcadores nil tipado aceptados: %#v, %v", servicio, err)
	}
	if servicio, err := NuevoServicioMetadatoInstitucionalDocumental(
		&registroMarcadoresLegacyPrueba{},
		verificador,
	); servicio != nil || !errors.Is(err, ErrMetadatoInstitucionalNoIncorporado) {
		t.Fatalf("verificador nil tipado aceptado: %#v, %v", servicio, err)
	}
}

func escenarioResolucionLegacyPrueba(
	t *testing.T,
) (ports.ConsultaFormatoDocumental, ports.DescriptorFormatoDocumental, *renderizadorLegacyPrueba) {
	t.Helper()
	perfil := perfilV2ParaContratoLegacyPrueba(t)
	revision, err := domain.NuevaRevisionCatalogoFormatosDocumentales(21, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	conector, err := domain.NuevaReferenciaConectorDocumental(
		"renderizador-pdfa", 5, "homologacion:renderizador-pdfa:5",
		strings.Repeat("b", 64), strings.Repeat("c", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	descriptor, err := ports.NuevoDescriptorFormatoDocumental(
		"descriptor:pdfa-4:2", perfil, revision, conector,
	)
	if err != nil {
		t.Fatal(err)
	}
	consulta := ports.ConsultaFormatoDocumental{
		Identidad: perfil.Identidad(), PerfilRef: perfil.Referencia(),
		DigestPerfilSHA256: perfil.DigestSHA256(), RevisionCatalogo: revision,
	}
	return consulta, descriptor, &renderizadorLegacyPrueba{
		perfil: perfil.Referencia(), digest: perfil.DigestSHA256(), conector: conector,
	}
}

func solicitudMetadatoLegacyPrueba(t *testing.T) ports.SolicitudIncorporarMetadatoInstitucional {
	t.Helper()
	perfil := perfilV2ParaContratoLegacyPrueba(t)
	_, descriptor, _ := escenarioResolucionLegacyPrueba(t)
	institucion, err := domain.NuevaReferenciaInstitucionalDocumento(
		"entidad:diputacion_granada", "organo:servicio_seleccion",
	)
	if err != nil {
		t.Fatal(err)
	}
	marca, err := domain.NuevaMarcaInstitucionalDocumento(
		institucion,
		"018f6f2a-6d92-4cc3-8c5e-3fb3e8e8a123",
		perfil.Referencia(),
		time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC),
		"manifiesto:documento:001",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido := []byte("%PDF-1.7 contenido sin firma")
	return ports.SolicitudIncorporarMetadatoInstitucional{
		Perfil: perfil, Conector: descriptor.Conector(),
		Etapa:             ports.EtapaMetadatoInstitucionalAntesFirma,
		ContenidoSinFirma: contenido, HuellaEntradaSHA256: huellaContenidoLegacyPrueba(contenido),
		Metadato: marca,
	}
}

func perfilV2ParaContratoLegacyPrueba(t *testing.T) domain.PerfilFormatoDocumental {
	t.Helper()
	identidad, referencia, capacidades := parametrosPerfilLegacyPrueba(t)
	conformidad, err := domain.NuevaReferenciaConformidadDocumental(
		"conformidad:pdfa-4", 3,
		"esquema:pdfa-4", "dialecto:pdfa-4f", "canon:pdfa-4",
		"reglas:pdfa-4:3", strings.Repeat("d", 64),
		"politica:documental:7", strings.Repeat("e", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	perfil, err := domain.NuevoPerfilFormatoDocumentalConforme(
		referencia, identidad, "application/pdf", "pdf", "binario",
		capacidades, conformidad, 16*1024*1024,
	)
	if err != nil {
		t.Fatal(err)
	}
	return perfil
}

func parametrosPerfilLegacyPrueba(
	t *testing.T,
) (
	domain.IdentidadSintacticaDocumental,
	domain.ReferenciaPerfilDocumental,
	domain.CapacidadesPerfilFormatoDocumental,
) {
	t.Helper()
	identidad, err := domain.NuevaIdentidadSintacticaDocumental("pdf")
	if err != nil {
		t.Fatal(err)
	}
	referencia, err := domain.NuevaReferenciaPerfilDocumental("pdfa-4", 2)
	if err != nil {
		t.Fatal(err)
	}
	capacidades, err := domain.NuevasCapacidadesPerfilFormatoDocumental(
		domain.CapacidadPerfilRenderizar,
		domain.CapacidadPerfilMetadatoInstitucional,
		domain.CapacidadPerfilFirmaElectronica,
	)
	if err != nil {
		t.Fatal(err)
	}
	return identidad, referencia, capacidades
}

func huellaContenidoLegacyPrueba(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}
