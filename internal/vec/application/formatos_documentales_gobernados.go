package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strconv"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	// Un unico error externo evita revelar si falta un perfil, existen dos
	// entradas contradictorias, fue retirado o no hay conector homologado.
	ErrResolucionFormatoDocumentalCerrada = errors.New("vec: resolucion de formato documental cerrada")
	ErrMetadatoInstitucionalNoIncorporado = errors.New("vec: metadato institucional no incorporado")
)

type ServicioResolucionFormatoDocumental struct {
	catalogo       ports.CatalogoFormatosDocumentales
	renderizadores ports.RegistroRenderizadoresDocumentales
}

func NuevoServicioResolucionFormatoDocumental(
	catalogo ports.CatalogoFormatosDocumentales,
	renderizadores ports.RegistroRenderizadoresDocumentales,
) (*ServicioResolucionFormatoDocumental, error) {
	if dependenciaDocumentalGobernadaNula(catalogo) || dependenciaDocumentalGobernadaNula(renderizadores) {
		return nil, ErrResolucionFormatoDocumentalCerrada
	}
	return &ServicioResolucionFormatoDocumental{catalogo: catalogo, renderizadores: renderizadores}, nil
}

// EvidenciaResolucionFormatoDocumental compromete la consulta y la unica
// respuesta aceptada, incluido el digest del artefacto instalado. Es inmutable
// en memoria; un puerto duradero de auditoria se incorporara al integrar el
// puente, sin reconstruir esta evidencia desde logs o mapas.
type EvidenciaResolucionFormatoDocumental struct {
	consulta   ports.ConsultaFormatoDocumental
	descriptor ports.DescriptorFormatoDocumental
	huella     string
}

// DatosEvidenciaResolucionFormatoDocumental es una instantanea tipada y
// autoconsistente para un futuro puerto duradero. Sus campos son valores
// inmutables o copias; modificarlos no altera la evidencia original.
type DatosEvidenciaResolucionFormatoDocumental struct {
	Consulta     ports.ConsultaFormatoDocumental
	Descriptor   ports.DescriptorFormatoDocumental
	HuellaSHA256 string
}

func nuevaEvidenciaResolucionFormatoDocumental(
	consulta ports.ConsultaFormatoDocumental,
	descriptor ports.DescriptorFormatoDocumental,
) (EvidenciaResolucionFormatoDocumental, error) {
	evidencia := EvidenciaResolucionFormatoDocumental{consulta: consulta, descriptor: descriptor}
	evidencia.huella = evidencia.calcularHuella()
	if evidencia.Validar() != nil {
		return EvidenciaResolucionFormatoDocumental{}, ErrResolucionFormatoDocumentalCerrada
	}
	return evidencia, nil
}

func (e EvidenciaResolucionFormatoDocumental) Validar() error {
	if e.consulta.Validar() != nil || e.descriptor.Validar() != nil ||
		!e.descriptor.Coincide(e.consulta) || len(e.huella) != 64 ||
		e.calcularHuella() != e.huella {
		return ErrResolucionFormatoDocumentalCerrada
	}
	return nil
}

func (e EvidenciaResolucionFormatoDocumental) HuellaSHA256() (string, error) {
	if e.Validar() != nil {
		return "", ErrResolucionFormatoDocumentalCerrada
	}
	return e.huella, nil
}

func (e EvidenciaResolucionFormatoDocumental) Datos() (
	DatosEvidenciaResolucionFormatoDocumental,
	error,
) {
	if e.Validar() != nil {
		return DatosEvidenciaResolucionFormatoDocumental{}, ErrResolucionFormatoDocumentalCerrada
	}
	return DatosEvidenciaResolucionFormatoDocumental{
		Consulta: e.consulta, Descriptor: e.descriptor, HuellaSHA256: e.huella,
	}, nil
}

func RestaurarEvidenciaResolucionFormatoDocumental(
	datos DatosEvidenciaResolucionFormatoDocumental,
) (EvidenciaResolucionFormatoDocumental, error) {
	evidencia := EvidenciaResolucionFormatoDocumental{
		consulta: datos.Consulta, descriptor: datos.Descriptor, huella: datos.HuellaSHA256,
	}
	if evidencia.Validar() != nil {
		return EvidenciaResolucionFormatoDocumental{}, ErrResolucionFormatoDocumentalCerrada
	}
	return evidencia, nil
}

func (e EvidenciaResolucionFormatoDocumental) calcularHuella() string {
	perfil := e.descriptor.Perfil()
	revision := e.descriptor.Revision()
	conector := e.descriptor.Conector()
	valores := []string{
		"vec.evidencia-resolucion-formato-documental.v1", e.consulta.Identidad.Identificador(),
		e.consulta.PerfilRef.Identificador(), strconv.FormatUint(e.consulta.PerfilRef.Version(), 10),
		e.consulta.DigestPerfilSHA256, strconv.FormatUint(e.consulta.RevisionCatalogo.Numero(), 10),
		e.consulta.RevisionCatalogo.HuellaSHA256(), e.descriptor.Referencia(),
		perfil.DigestSHA256(), strconv.FormatUint(revision.Numero(), 10), revision.HuellaSHA256(),
		conector.Identificador(), strconv.FormatUint(conector.Version(), 10), conector.HomologacionRef(),
		conector.HuellaHomologacionSHA256(), conector.HuellaArtefactoSHA256(),
	}
	calculador := sha256.New()
	for _, valor := range valores {
		_, _ = calculador.Write([]byte(strconv.Itoa(len(valor))))
		_, _ = calculador.Write([]byte{':'})
		_, _ = calculador.Write([]byte(valor))
		_, _ = calculador.Write([]byte{'\n'})
	}
	return hex.EncodeToString(calculador.Sum(nil))
}

type FormatoDocumentalResuelto struct {
	descriptor   ports.DescriptorFormatoDocumental
	renderizador ports.RenderizadorDocumentalPorPerfil
	evidencia    EvidenciaResolucionFormatoDocumental
}

func (r FormatoDocumentalResuelto) Descriptor() (ports.DescriptorFormatoDocumental, error) {
	if r.descriptor.Validar() != nil || ports.RenderizadorDocumentalNulo(r.renderizador) ||
		r.evidencia.Validar() != nil {
		return ports.DescriptorFormatoDocumental{}, ErrResolucionFormatoDocumentalCerrada
	}
	return r.descriptor, nil
}

func (r FormatoDocumentalResuelto) Renderizador() (ports.RenderizadorDocumentalPorPerfil, error) {
	if r.descriptor.Validar() != nil || ports.RenderizadorDocumentalNulo(r.renderizador) ||
		r.evidencia.Validar() != nil {
		return nil, ErrResolucionFormatoDocumentalCerrada
	}
	return r.renderizador, nil
}

func (r FormatoDocumentalResuelto) Evidencia() (EvidenciaResolucionFormatoDocumental, error) {
	if r.evidencia.Validar() != nil || r.descriptor.Validar() != nil ||
		ports.RenderizadorDocumentalNulo(r.renderizador) {
		return EvidenciaResolucionFormatoDocumental{}, ErrResolucionFormatoDocumentalCerrada
	}
	return r.evidencia, nil
}

func (s *ServicioResolucionFormatoDocumental) Resolver(
	ctx context.Context,
	consulta ports.ConsultaFormatoDocumental,
) (FormatoDocumentalResuelto, error) {
	if ctx == nil || ctx.Err() != nil || s == nil ||
		dependenciaDocumentalGobernadaNula(s.catalogo) ||
		dependenciaDocumentalGobernadaNula(s.renderizadores) || consulta.Validar() != nil {
		return FormatoDocumentalResuelto{}, ErrResolucionFormatoDocumentalCerrada
	}
	descriptores, err := s.catalogo.BuscarDescriptoresFormatoDocumental(ctx, consulta)
	if err != nil || ctx.Err() != nil || len(descriptores) != 1 {
		return FormatoDocumentalResuelto{}, ErrResolucionFormatoDocumentalCerrada
	}
	// Desconocido y ambiguo son el mismo cierre. Nunca se elige la primera
	// entrada ni se intenta otra revision o perfil.
	descriptor := descriptores[0]
	perfil := descriptor.Perfil()
	if descriptor.Validar() != nil || !descriptor.Coincide(consulta) ||
		perfil.Estado() != domain.EstadoPerfilDocumentalVigente ||
		!perfil.Capacidades().Tiene(domain.CapacidadPerfilRenderizar) {
		return FormatoDocumentalResuelto{}, ErrResolucionFormatoDocumentalCerrada
	}
	candidatos, err := s.renderizadores.BuscarRenderizadoresDocumentales(
		ctx, perfil.Referencia(), descriptor.Conector(),
	)
	if err != nil || ctx.Err() != nil || len(candidatos) != 1 ||
		ports.RenderizadorDocumentalNulo(candidatos[0]) {
		return FormatoDocumentalResuelto{}, ErrResolucionFormatoDocumentalCerrada
	}
	renderizador := candidatos[0]
	if renderizador.PerfilDocumental() != perfil.Referencia() ||
		renderizador.DigestPerfilSHA256() != perfil.DigestSHA256() ||
		renderizador.ConectorDocumental() != descriptor.Conector() {
		return FormatoDocumentalResuelto{}, ErrResolucionFormatoDocumentalCerrada
	}
	evidencia, err := nuevaEvidenciaResolucionFormatoDocumental(consulta, descriptor)
	if err != nil {
		return FormatoDocumentalResuelto{}, ErrResolucionFormatoDocumentalCerrada
	}
	return FormatoDocumentalResuelto{
		descriptor: descriptor, renderizador: renderizador, evidencia: evidencia,
	}, nil
}

type ServicioMetadatoInstitucionalDocumental struct {
	marcadores  ports.RegistroMarcadoresMetadatoInstitucional
	verificador ports.VerificadorEquivalenciaSemanticaDocumental
}

// Este servicio es un contrato en sombra. Separar interfaces evita la
// autocertificacion directa, pero no prueba por si solo que sean componentes
// independientes. El despliegue productivo debe permanecer cerrado hasta que
// el verificador tenga identidad, digest y homologacion atestados, distintos
// de los del marcador, y el bootstrap coteje esa segregacion.
func NuevoServicioMetadatoInstitucionalDocumental(
	marcadores ports.RegistroMarcadoresMetadatoInstitucional,
	verificador ports.VerificadorEquivalenciaSemanticaDocumental,
) (*ServicioMetadatoInstitucionalDocumental, error) {
	if dependenciaDocumentalGobernadaNula(marcadores) || dependenciaDocumentalGobernadaNula(verificador) {
		return nil, ErrMetadatoInstitucionalNoIncorporado
	}
	return &ServicioMetadatoInstitucionalDocumental{
		marcadores: marcadores, verificador: verificador,
	}, nil
}

func (s *ServicioMetadatoInstitucionalDocumental) Incorporar(
	ctx context.Context,
	solicitud ports.SolicitudIncorporarMetadatoInstitucional,
) (ports.ResultadoMetadatoInstitucional, error) {
	if ctx == nil || ctx.Err() != nil || s == nil ||
		dependenciaDocumentalGobernadaNula(s.marcadores) ||
		dependenciaDocumentalGobernadaNula(s.verificador) {
		return ports.ResultadoMetadatoInstitucional{}, ErrMetadatoInstitucionalNoIncorporado
	}
	// Se toma una unica instantanea defensiva antes de validar y antes de
	// entregar nada al conector. Esa misma copia se usa en todas las
	// comprobaciones posteriores; no se mezcla con el slice del llamador.
	instantanea := solicitud
	instantanea.ContenidoSinFirma = append([]byte(nil), solicitud.ContenidoSinFirma...)
	if instantanea.Validar() != nil {
		return ports.ResultadoMetadatoInstitucional{}, ErrMetadatoInstitucionalNoIncorporado
	}
	candidatos, err := s.marcadores.BuscarMarcadoresMetadatoInstitucional(
		ctx, instantanea.Perfil.Referencia(), instantanea.Conector,
	)
	if err != nil || ctx.Err() != nil || len(candidatos) != 1 ||
		ports.MarcadorMetadatoInstitucionalNulo(candidatos[0]) {
		return ports.ResultadoMetadatoInstitucional{}, ErrMetadatoInstitucionalNoIncorporado
	}
	marcador := candidatos[0]
	if marcador.PerfilDocumental() != instantanea.Perfil.Referencia() ||
		marcador.DigestPerfilSHA256() != instantanea.Perfil.DigestSHA256() ||
		marcador.ConectorDocumental() != instantanea.Conector {
		return ports.ResultadoMetadatoInstitucional{}, ErrMetadatoInstitucionalNoIncorporado
	}
	resultado, err := marcador.IncorporarMetadatoInstitucional(ctx, instantanea)
	if err != nil || ctx.Err() != nil {
		return ports.ResultadoMetadatoInstitucional{}, ErrMetadatoInstitucionalNoIncorporado
	}
	// El resultado se desacopla una vez antes de verificarlo y devolverlo. Si
	// el adaptador conserva o reutiliza su buffer, no puede cambiar lo ya
	// validado por el servicio.
	seguro := resultado
	seguro.Contenido = append([]byte(nil), resultado.Contenido...)
	if instantanea.Validar() != nil || seguro.ValidarContra(instantanea) != nil ||
		s.verificador.VerificarEquivalenciaSemantica(
			ctx, instantanea.Perfil, instantanea.ContenidoSinFirma, seguro.Contenido,
		) != nil || ctx.Err() != nil {
		return ports.ResultadoMetadatoInstitucional{}, ErrMetadatoInstitucionalNoIncorporado
	}
	return seguro, nil
}

func dependenciaDocumentalGobernadaNula(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}
