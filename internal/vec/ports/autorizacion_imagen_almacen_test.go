package ports

import (
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func imagenAlmacenPrueba(t *testing.T, accion string, objeto bool) (
	domain.DecisionAutorizacion, domain.RecursoAutorizable, VinculosOperacionAlmacen,
	ImagenAlmacenVinculada, time.Time,
) {
	t.Helper()
	campos := []string{"contenido_png256", "objeto_cuarentena"}
	switch accion {
	case AccionNegocioPromoverImagenProcesada:
		campos = []string{"objeto_admitido", "estado"}
	case AccionNegocioAbrirImagenPropiaActiva, AccionNegocioAbrirImagenAjenaActiva:
		campos = []string{"contenido_png256"}
	}
	d, r, v, instante := autorizacionAlmacenPrueba(t, accion, campos, objeto)
	i := ImagenAlmacenVinculada{
		DocumentoRef: "documento:imagen:0001", ActorPersonaRef: "per_actor_00000001",
		TitularPersonaRef: "per_actor_00000001", Audiencia: audienciaImagenPersonal,
		Finalidad: finalidadImagenPropia, HuellaSHA256: strings.Repeat("a", 64),
		Tamano: 1024, ClaveIdempotencia: "documento:imagen:0001:cuarentena",
	}
	if accion == AccionNegocioPromoverImagenProcesada {
		i.ClaveIdempotencia = "documento:imagen:0001:admitida"
	}
	if accion == AccionNegocioAbrirImagenPropiaActiva || accion == AccionNegocioAbrirImagenAjenaActiva {
		i.ClaveIdempotencia = ""
	}
	if accion == AccionNegocioAbrirImagenAjenaActiva {
		i.ActorPersonaRef = "per_otra_00000001"
		i.Audiencia = audienciaImagenInterna
		i.Finalidad = finalidadImagenInterna
	}
	v.CargaRef = i.DocumentoRef
	r.Referencia = i.DocumentoRef
	r.ModuloID = "documentos"
	r.Tipo = "imagen_usuario"
	r.Atributos[AtributoAlmacenCargaRef] = v.CargaRef
	r.Atributos[AtributoAlmacenImagenDocumentoRef] = i.DocumentoRef
	r.Atributos[AtributoAlmacenImagenActorPersonaRef] = i.ActorPersonaRef
	r.Atributos[AtributoAlmacenImagenTitularPersonaRef] = i.TitularPersonaRef
	r.Atributos[AtributoAlmacenImagenAudiencia] = i.Audiencia
	r.Atributos[AtributoAlmacenImagenFinalidad] = i.Finalidad
	r.Atributos[AtributoAlmacenImagenHuellaSHA256] = i.HuellaSHA256
	r.Atributos[AtributoAlmacenImagenTamano] = strconv.FormatInt(i.Tamano, 10)
	if i.ClaveIdempotencia != "" {
		r.Atributos[AtributoAlmacenImagenClaveIdempotencia] = i.ClaveIdempotencia
	}
	d.RecursoRef = r.Referencia
	d.ModuloID = r.ModuloID
	d.TipoRecurso = r.Tipo
	d.Finalidad = i.Finalidad
	d.ContextoRecursoHuellaSHA256, _ = r.HuellaContextoAutorizacionSHA256()
	return d, r, v, i, instante
}

func TestImagenProcesadaAlmacenCierraEscrituraPNG256YAccion(t *testing.T) {
	d, r, v, i, instante := imagenAlmacenPrueba(t, AccionNegocioEscribirImagenProcesada, false)
	c, err := NuevoContextoEscribirImagenProcesadaAlmacen(d, r, v, i, instante)
	if err != nil || c.ValidarParaEn(AccionAlmacenEscribir, instante) != nil {
		t.Fatalf("capacidad de imagen: %v", err)
	}
	s := SolicitudEscribirObjeto{Contexto: c, ClaveIdempotencia: i.ClaveIdempotencia,
		Zona: ZonaAlmacenCuarentena, MIME: "image/png", Tamano: i.Tamano,
		HuellaSHA256: i.HuellaSHA256, Contenido: strings.NewReader("png procesado")}
	if err := s.Validar(); err != nil {
		t.Fatalf("solicitud exacta: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*SolicitudEscribirObjeto)
	}{
		{"clave", func(s *SolicitudEscribirObjeto) { s.ClaveIdempotencia = "otra:clave" }},
		{"zona", func(s *SolicitudEscribirObjeto) { s.Zona = ZonaAlmacenAdmitida }},
		{"mime", func(s *SolicitudEscribirObjeto) { s.MIME = "image/jpeg" }},
		{"tamano", func(s *SolicitudEscribirObjeto) { s.Tamano++ }},
		{"huella", func(s *SolicitudEscribirObjeto) { s.HuellaSHA256 = strings.Repeat("b", 64) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := s
			caso.mutar(&mutada)
			if !errors.Is(mutada.Validar(), ErrAutorizacionAlmacenInvalida) {
				t.Fatal("metadatos cambiados aceptados")
			}
		})
	}
	if _, err := c.DerivarPaso(PasoAlmacenPromoverImagenProcesada); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("escritura derivo promocion")
	}
	if c.ValidarParaEn(AccionAlmacenLeer, instante) == nil {
		t.Fatal("escritura habilito lectura")
	}
	cruzada := d
	cruzada.Accion = AccionNegocioAbrirImagenPropiaActiva
	if _, err := NuevoContextoEscribirImagenProcesadaAlmacen(cruzada, r, v, i, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("permiso de otra operacion aceptado")
	}
}

func TestImagenProcesadaAlmacenDeniegaRefVersionHuellaYAudienciaCruzadas(t *testing.T) {
	d, r, v, i, instante := imagenAlmacenPrueba(t, AccionNegocioPromoverImagenProcesada, true)
	c, err := NuevoContextoPromoverImagenProcesadaAlmacen(d, r, v, i, instante)
	if err != nil || !c.coincideObjeto(v.ObjetoVinculado) {
		t.Fatalf("promocion exacta: %v", err)
	}
	if c.coincideObjeto(ReferenciaObjetoAlmacen{Referencia: v.ObjetoVinculado.Referencia, Version: "version:otra"}) {
		t.Fatal("version ajena aceptada")
	}
	v2 := v
	v2.ObjetoVinculado.Version = "version:otra"
	if _, err := NuevoContextoPromoverImagenProcesadaAlmacen(d, r, v2, i, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("promocion con version ajena aceptada")
	}
	i2 := i
	i2.HuellaSHA256 = strings.Repeat("b", 64)
	if _, err := NuevoContextoPromoverImagenProcesadaAlmacen(d, r, v, i2, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("huella ajena aceptada")
	}
	i2 = i
	i2.DocumentoRef = "documento:imagen:0002"
	if _, err := NuevoContextoPromoverImagenProcesadaAlmacen(d, r, v, i2, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("documento ajeno aceptado")
	}
	r2 := clonarRecursoAlmacenPrueba(r)
	r2.Atributos[AtributoAlmacenImagenTamano] = "2048"
	if _, err := NuevoContextoPromoverImagenProcesadaAlmacen(d, r2, v, i, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("tamano alterado aceptado")
	}
	r2 = clonarRecursoAlmacenPrueba(r)
	r2.Atributos[AtributoAlmacenImagenTitularPersonaRef] = "per_otra_00000001"
	if _, err := NuevoContextoPromoverImagenProcesadaAlmacen(d, r2, v, i, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("titular alterado aceptado")
	}
	if _, err := c.DerivarPaso(PasoAlmacenLeerParaAnalisis); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("promocion derivo analisis")
	}
}

func TestImagenActivaAlmacenSeparaPropiaAjenaYRetirada(t *testing.T) {
	d, r, v, i, instante := imagenAlmacenPrueba(t, AccionNegocioAbrirImagenAjenaActiva, true)
	c, err := NuevoContextoAbrirImagenActivaAlmacen(d, r, v, i, instante)
	if err != nil || c.ValidarParaEn(AccionAlmacenLeer, instante) != nil {
		t.Fatalf("lectura interna nominal: %v", err)
	}
	if !c.coincideObjeto(v.ObjetoVinculado) || c.coincideObjeto(ReferenciaObjetoAlmacen{Referencia: "otro:objeto", Version: v.ObjetoVinculado.Version}) {
		t.Fatal("objeto ajeno aceptado")
	}
	i2 := i
	i2.Audiencia = audienciaImagenPersonal
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(d, r, v, i2, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("audiencia externa aceptada")
	}
	i2 = i
	i2.Finalidad = finalidadImagenPropia
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(d, r, v, i2, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("finalidad propia en lectura ajena aceptada")
	}
	i2 = i
	i2.Audiencia = "mi_bolsa_publica"
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(d, r, v, i2, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("audiencia publica aceptada")
	}
	desconocida := d
	desconocida.Accion = "documentos.imagen.almacen.leer_cualquiera"
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(desconocida, r, v, i, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("operacion no catalogada aceptada")
	}
	if c.ValidarEn(d.ValidaHasta) == nil {
		t.Fatal("decision caducada aceptada")
	}
	retirada := d
	retirada.Concedida = false
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(retirada, r, v, i, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("lectura denegada aceptada")
	}
	if _, err := c.DerivarPaso(PasoAlmacenPromoverImagenProcesada); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("lectura derivo promocion")
	}
}
