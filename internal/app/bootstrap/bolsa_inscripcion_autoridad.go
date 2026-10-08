package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// DecisorLecturaActualInscripcionBolsa es el seam pendiente de la autoridad
// común: debe revalidar la sesión revocable y la concesión central ACTUAL en
// memoria, con acción, recurso, finalidad, perfil y campos exactos. No puede
// implementarse con una instantánea histórica ni con los roles del principal.
type DecisorLecturaActualInscripcionBolsa interface {
	DecidirLecturaActual(context.Context, contextoSeguridadComunDesarrollo, string, string, inscripcion.Filtro) (DecisionLecturaActualInscripcionBolsa, error)
}

// La decisión conserva todos los datos de una única lectura. El constructor
// exige un decisor real; ningún valor por defecto concede acceso.
type DecisionLecturaActualInscripcionBolsa struct {
	Concedida                                                     bool
	PersonaRef, PerfilRef, CuentaRef, SesionRef, AutenticacionRef string
	CertificadoHuellaSHA256, Canal, Accion, RecursoRef, Finalidad string
	CorrelacionRef                                                string
	RevisionPermisos                                              uint64
	Campos                                                        []string
	Filtro                                                        inscripcion.Filtro
	EmitidaEn, ValidaHasta                                        time.Time
}

// DescriptorInscripcionBolsa se aporta desde el catálogo publicado. Su motivo
// se vuelve a resolver en el validador central para cada acto V3.
type DescriptorInscripcionBolsa struct {
	Accion, ModuloID, TipoRecurso, Finalidad string
	Motivo                                   vecdomain.ReferenciaEntradaCatalogo
}

type DescriptorLecturaInscripcionBolsa struct {
	Accion, Finalidad string
	Campos            []string
}

type ConfiguracionAutoridadInscripcionBolsa struct {
	Lectura      DecisorLecturaActualInscripcionBolsa
	PDP          *vecapp.ServicioAutorizacionSolicitudLigadaV3
	Material     *proveedorMaterialAltaContratacionTemporalDesarrollo
	Motivos      vecports.ValidadorReferenciaMotivoAutorizacionV2
	Reloj        interface{ Ahora() time.Time }
	Descriptores map[string]DescriptorInscripcionBolsa
	Lecturas     map[string]DescriptorLecturaInscripcionBolsa
}

type autoridadNominalInscripcionBolsa struct {
	c ConfiguracionAutoridadInscripcionBolsa
}

var _ AutoridadInscripcionBolsa = (*autoridadNominalInscripcionBolsa)(nil)

func NuevaAutoridadInscripcionBolsa(c ConfiguracionAutoridadInscripcionBolsa) (AutoridadInscripcionBolsa, error) {
	if nuloInscripcionBolsa(c.Lectura) || c.PDP == nil || c.Material == nil ||
		nuloInscripcionBolsa(c.Motivos) || nuloInscripcionBolsa(c.Reloj) ||
		len(c.Descriptores) != 3 || len(c.Lecturas) != 7 {
		return nil, inscripcion.ErrNoDisponible
	}
	for _, accion := range []string{inscripcion.AccionPresentar, inscripcion.AccionDecidir, inscripcion.AccionIncorporar} {
		d, ok := c.Descriptores[accion]
		if !ok || d.Accion != accion || d.ModuloID == "" || d.TipoRecurso == "" || d.Finalidad == "" || d.Motivo.Validar() != nil {
			return nil, inscripcion.ErrNoDisponible
		}
	}
	lecturas := make(map[string]DescriptorLecturaInscripcionBolsa, 7)
	for accion, d := range c.Lecturas {
		if !accionLecturaInscripcion(accion) || d.Accion != accion || d.Finalidad == "" || len(d.Campos) == 0 {
			return nil, inscripcion.ErrNoDisponible
		}
		vistos := make(map[string]struct{}, len(d.Campos))
		for _, campo := range d.Campos {
			if campo == "" || campo == "*" {
				return nil, inscripcion.ErrNoDisponible
			}
			if _, duplicado := vistos[campo]; duplicado {
				return nil, inscripcion.ErrNoDisponible
			}
			vistos[campo] = struct{}{}
		}
		d.Campos = slices.Clone(d.Campos)
		lecturas[accion] = d
	}
	clon := make(map[string]DescriptorInscripcionBolsa, 3)
	for k, v := range c.Descriptores {
		clon[k] = v
	}
	c.Descriptores = clon
	c.Lecturas = lecturas
	return &autoridadNominalInscripcionBolsa{c: c}, nil
}

func (a *autoridadNominalInscripcionBolsa) CapturarLectura(ctx context.Context, s contextoSeguridadComunDesarrollo, accion, recurso string, filtro inscripcion.Filtro) (inscripcion.CapturaLectura, error) {
	vacia := inscripcion.CapturaLectura{}
	if a == nil || ctx == nil || ctx.Err() != nil || nuloInscripcionBolsa(a.c.Lectura) || nuloInscripcionBolsa(a.c.Reloj) ||
		!accionLecturaInscripcion(accion) || s.Resultado.Validar() != nil || s.Vinculo.ValidarPara(s.Resultado) != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora := a.c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !s.Vinculo.VigenteEn(ahora, s.Resultado) {
		return vacia, inscripcion.ErrSesionAusente
	}
	v, err := s.Vinculo.Datos()
	if err != nil {
		return vacia, inscripcion.ErrSesionAusente
	}
	decision, err := a.c.Lectura.DecidirLecturaActual(ctx, s, accion, recurso, filtro)
	if err != nil || ctx.Err() != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora = a.c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	descriptor, publicado := a.c.Lecturas[accion]
	if !decision.Concedida || !s.Vinculo.VigenteEn(ahora, s.Resultado) ||
		!publicado || descriptor.Accion != accion || descriptor.Finalidad == "" || len(descriptor.Campos) == 0 ||
		decision.PersonaRef != s.Resultado.Contexto.PersonaRef || decision.PerfilRef != v.PerfilActivoRef ||
		decision.CuentaRef != v.CuentaRef || decision.SesionRef != v.SesionRef || decision.AutenticacionRef != v.AutenticacionRef ||
		!huellaCertificadoInscripcionValida(decision.CertificadoHuellaSHA256) ||
		decision.Accion != accion || decision.RecursoRef != recurso || decision.Filtro != filtro ||
		decision.RevisionPermisos == 0 || decision.Finalidad != descriptor.Finalidad ||
		!vecdomain.ReferenciaCorrelacionAutorizacionV2Valida(decision.CorrelacionRef) ||
		!slices.Equal(decision.Campos, descriptor.Campos) || decision.EmitidaEn.IsZero() || decision.EmitidaEn.After(ahora) ||
		!decision.ValidaHasta.After(ahora) || decision.ValidaHasta.After(ahora.Add(30*time.Second)) ||
		decision.ValidaHasta.After(v.SesionValidaHasta) ||
		!canalLecturaInscripcion(accion, decision.Canal, v.Superficie) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	return inscripcion.CapturaLectura{
		PersonaRef: decision.PersonaRef, PerfilRef: decision.PerfilRef, CuentaRef: decision.CuentaRef,
		SesionRef: decision.SesionRef, AutenticacionRef: decision.AutenticacionRef,
		CertificadoHuellaSHA256: decision.CertificadoHuellaSHA256, Canal: decision.Canal,
		Accion: accion, RecursoRef: recurso, Finalidad: decision.Finalidad,
		CorrelacionRef: decision.CorrelacionRef, RevisionPermisos: decision.RevisionPermisos,
		Filtro: filtro, EmitidaEn: decision.EmitidaEn, ValidaHasta: decision.ValidaHasta,
	}, nil
}

func accionLecturaInscripcion(a string) bool {
	switch a {
	case inscripcion.AccionListarAbiertas, inscripcion.AccionDetalleAbierta, inscripcion.AccionListarPropias,
		inscripcion.AccionDetallePropia, inscripcion.AccionListarRRHH, inscripcion.AccionDetalleRRHH, inscripcion.AccionMotivosRRHH:
		return true
	}
	return false
}

func canalLecturaInscripcion(a, canal string, superficie vecdomain.SuperficieAutenticacionActorV1) bool {
	if a == inscripcion.AccionListarRRHH || a == inscripcion.AccionDetalleRRHH || a == inscripcion.AccionMotivosRRHH {
		return canal == "interna_corporativa" && superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1
	}
	return canal == "externa_personal" && superficie == vecdomain.SuperficieAutenticacionExternaPersonalV1 ||
		canal == "interna_corporativa" && superficie == vecdomain.SuperficieAutenticacionInternaCorporativaV1
}

func (a *autoridadNominalInscripcionBolsa) AutorizarEscritura(ctx context.Context, s contextoSeguridadComunDesarrollo, accion, recurso string, material, recursoCanonico []byte) (AutorizacionEscrituraInscripcionBolsa, error) {
	vacia := AutorizacionEscrituraInscripcionBolsa{}
	if a == nil || ctx == nil || ctx.Err() != nil || a.c.PDP == nil || a.c.Material == nil ||
		nuloInscripcionBolsa(a.c.Motivos) || nuloInscripcionBolsa(a.c.Reloj) ||
		s.Resultado.Validar() != nil || s.Vinculo.ValidarPara(s.Resultado) != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	d, ok := a.c.Descriptores[accion]
	if !ok || d.Accion != accion || d.Motivo.Validar() != nil || d.ModuloID == "" || d.TipoRecurso == "" || d.Finalidad == "" {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	ahora := a.c.Reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !s.Vinculo.VigenteEn(ahora, s.Resultado) || !materialEscrituraInscripcionExacto(accion, s.Resultado.Contexto.PersonaRef, recurso, material, recursoCanonico) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	v, err := s.Vinculo.Datos()
	if err != nil || (accion != inscripcion.AccionPresentar && v.Superficie != vecdomain.SuperficieAutenticacionInternaCorporativaV1) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	if err := a.c.Motivos.ValidarReferenciaMotivoAutorizacionV2(ctx, d.Motivo, ahora); err != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	var rc struct {
		Ambitos   map[string]string `json:"ambitos"`
		Atributos map[string]string `json:"atributos"`
	}
	if json.Unmarshal(recursoCanonico, &rc) != nil || len(rc.Atributos) != 1 {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	h := sha256.Sum256(material)
	if rc.Atributos["material_sha256"] != hex.EncodeToString(h[:]) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, inscripcion.ErrNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: s.Vinculo, ReferenciaMotivo: d.Motivo, Accion: accion,
		Recurso:   vecdomain.RecursoAutorizable{Referencia: recurso, ModuloID: d.ModuloID, Tipo: d.TipoRecurso, Ambitos: rc.Ambitos, Atributos: rc.Atributos},
		Finalidad: d.Finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	decision, confirmacion, err := a.c.PDP.ExigirSolicitudLigadaV3(ctx, solicitud, s.Resultado)
	concedida, _, errResultado := decision.Resultado()
	if err != nil || errResultado != nil || !concedida || decision.ValidarPara(solicitud) != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	materialV3, err := a.c.Material.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, d.Motivo, s.Resultado)
	if err != nil || materialV3.ValidarEstructura() != nil {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	resumen := materialV3.ResumenCapacidad()
	hr := sha256.Sum256(recursoCanonico)
	if resumen.Operacion() != accion || resumen.AudienciaConsumo() != audienciaEscrituraInscripcion(accion) ||
		resumen.EfectoRef() != recurso || resumen.EfectoHuellaSHA256() != hex.EncodeToString(hr[:]) ||
		resumen.ContextoRef() != s.Resultado.RegistroContextoRef || resumen.ContextoHuellaSHA256() != s.Resultado.HuellaSHA256 ||
		!bytes.Equal(materialV3.ContextoActorCanonico(), s.Resultado.RepresentacionCanonica) {
		return vacia, inscripcion.ErrAccesoDenegado
	}
	return AutorizacionEscrituraInscripcionBolsa{Accion: accion, Recurso: recurso, MaterialSHA256: hex.EncodeToString(h[:]), Material: materialV3}, nil
}

func audienciaEscrituraInscripcion(accion string) string {
	switch accion {
	case inscripcion.AccionPresentar:
		return "vec_bolsa_llamamientos.inscripcion.presentar.v1"
	case inscripcion.AccionDecidir:
		return "vec_bolsa_llamamientos.inscripcion.revisar.v1"
	case inscripcion.AccionIncorporar:
		return "vec_bolsa_llamamientos.inscripcion.incorporar.v1"
	default:
		return ""
	}
}

func materialEscrituraInscripcionExacto(accion, persona, recurso string, material, rc []byte) bool {
	if len(material) == 0 || len(material) > 64*1024 || len(rc) == 0 || len(rc) > 8*1024 {
		return false
	}
	var esperadoMaterial, esperadoRecurso []byte
	var huella, ref string
	var err error
	switch accion {
	case inscripcion.AccionPresentar:
		var v struct {
			Esquema           string                    `json:"esquema"`
			ConvocatoriaRef   string                    `json:"convocatoria_ref"`
			CategoriaRef      string                    `json:"categoria_ref"`
			CatalogoVersion   uint64                    `json:"catalogo_version"`
			ClaveIdempotencia string                    `json:"clave_idempotencia"`
			Declaraciones     []inscripcion.Declaracion `json:"declaraciones"`
		}
		if json.Unmarshal(material, &v) != nil || v.Esquema != inscripcion.EsquemaMaterialPresentacion {
			return false
		}
		p := inscripcion.Presentacion{ConvocatoriaRef: v.ConvocatoriaRef, CategoriaRef: v.CategoriaRef, CatalogoVersion: v.CatalogoVersion, ClaveIdempotencia: v.ClaveIdempotencia, Declaraciones: v.Declaraciones}
		esperadoMaterial, huella, err = inscripcion.MaterialPresentacion(p)
		if err == nil {
			esperadoRecurso, err = inscripcion.RecursoPresentacion(p, huella)
		}
		if err == nil {
			ref, err = inscripcion.ReferenciaSolicitud(persona, p)
		}
	case inscripcion.AccionDecidir:
		var v struct {
			Esquema           string `json:"esquema"`
			SolicitudRef      string `json:"solicitud_ref"`
			Decision          string `json:"decision"`
			MotivoCodigo      string `json:"motivo_codigo"`
			VersionEsperada   uint64 `json:"version_esperada"`
			ClaveIdempotencia string `json:"clave_idempotencia"`
		}
		if json.Unmarshal(material, &v) != nil || v.Esquema != inscripcion.EsquemaMaterialDecision {
			return false
		}
		d := inscripcion.Decision{SolicitudRef: v.SolicitudRef, Tipo: v.Decision, MotivoCodigo: v.MotivoCodigo, VersionEsperada: v.VersionEsperada, ClaveIdempotencia: v.ClaveIdempotencia}
		esperadoMaterial, huella, err = inscripcion.MaterialDecision(d)
		if err == nil {
			esperadoRecurso, err = inscripcion.RecursoDecision(d, huella)
		}
		ref = d.SolicitudRef
	case inscripcion.AccionIncorporar:
		var v struct {
			Esquema           string `json:"esquema"`
			SolicitudRef      string `json:"solicitud_ref"`
			EvidenciaRef      string `json:"evidencia_ref"`
			VersionEsperada   uint64 `json:"version_esperada"`
			ClaveIdempotencia string `json:"clave_idempotencia"`
		}
		if json.Unmarshal(material, &v) != nil || v.Esquema != inscripcion.EsquemaMaterialIncorporacion {
			return false
		}
		i := inscripcion.Incorporacion{SolicitudRef: v.SolicitudRef, EvidenciaRef: v.EvidenciaRef, VersionEsperada: v.VersionEsperada, ClaveIdempotencia: v.ClaveIdempotencia}
		esperadoMaterial, huella, err = inscripcion.MaterialIncorporacion(i)
		if err == nil {
			esperadoRecurso, err = inscripcion.RecursoIncorporacion(i, huella)
		}
		ref = i.SolicitudRef
	default:
		return false
	}
	return err == nil && ref == recurso && bytes.Equal(esperadoMaterial, material) && bytes.Equal(esperadoRecurso, rc)
}
