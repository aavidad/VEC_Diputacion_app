package ports

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func imagenAlmacenV3Prueba(t *testing.T, accion string, superficies ...domain.SuperficieAutenticacionActorV1) (
	AutorizacionImagenAlmacenV3, VinculosOperacionAlmacen, ImagenAlmacenVinculada, time.Time,
) {
	t.Helper()
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	superficie := domain.SuperficieAutenticacionInternaCorporativaV1
	if len(superficies) != 0 {
		superficie = superficies[0]
	}
	_, _, v, _ := autorizacionAlmacenPrueba(t, AccionNegocioPrepararCargaDocumental,
		[]string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"}, false)
	campos := []string{"contenido_png256", "objeto_cuarentena"}
	objeto := false
	if accion == AccionNegocioPromoverImagenProcesada {
		campos = []string{"objeto_admitido", "estado"}
		objeto = true
	}
	if accion == AccionNegocioAbrirImagenPropiaActiva || accion == AccionNegocioAbrirImagenAjenaActiva {
		campos = []string{"contenido_png256"}
		objeto = true
	}
	i := ImagenAlmacenVinculada{
		DocumentoRef: "documento:imagen:0001", ActorPersonaRef: e.resultado.Contexto.PersonaRef,
		TitularPersonaRef: e.resultado.Contexto.PersonaRef,
		Audiencia:         audienciaImagenInterna, Finalidad: finalidadImagenPropia,
		HuellaSHA256: strings.Repeat("a", 64), Tamano: 1024,
		ClaveIdempotencia: "documento:imagen:0001:cuarentena",
	}
	if accion == AccionNegocioPromoverImagenProcesada {
		i.ClaveIdempotencia = "documento:imagen:0001:admitida"
		i.EvidenciaAnalisisRef = "analisis:limpio:0001"
	}
	if accion == AccionNegocioAbrirImagenPropiaActiva || accion == AccionNegocioAbrirImagenAjenaActiva {
		i.ClaveIdempotencia = ""
	}
	if accion == AccionNegocioAbrirImagenAjenaActiva {
		i.TitularPersonaRef = "per_titular_0000000001"
		i.Audiencia = audienciaImagenInterna
		i.Finalidad = finalidadImagenInterna
	}
	if superficie == domain.SuperficieAutenticacionExternaPersonalV1 &&
		accion != AccionNegocioAbrirImagenAjenaActiva {
		i.Audiencia = audienciaImagenPersonal
	}
	v.CargaRef = i.DocumentoRef
	a := map[string]string{
		AtributoAlmacenOperacionRef:            v.OperacionRef,
		AtributoAlmacenCargaRef:                v.CargaRef,
		AtributoAlmacenClasificacion:           v.Clasificacion,
		AtributoAlmacenSujetoSeudonimoHMAC:     v.SujetoSeudonimoHMAC,
		AtributoAlmacenHuellaSolicitudHMAC:     v.HuellaSolicitudHMAC,
		AtributoAlmacenEfectoRef:               v.EfectoRef,
		AtributoAlmacenImagenDocumentoRef:      i.DocumentoRef,
		AtributoAlmacenImagenActorPersonaRef:   i.ActorPersonaRef,
		AtributoAlmacenImagenTitularPersonaRef: i.TitularPersonaRef,
		AtributoAlmacenImagenAudiencia:         i.Audiencia,
		AtributoAlmacenImagenFinalidad:         i.Finalidad,
		AtributoAlmacenImagenHuellaSHA256:      i.HuellaSHA256,
		AtributoAlmacenImagenTamano:            strconv.FormatInt(i.Tamano, 10),
	}
	if i.ClaveIdempotencia != "" {
		a[AtributoAlmacenImagenClaveIdempotencia] = i.ClaveIdempotencia
	}
	if i.EvidenciaAnalisisRef != "" {
		a[AtributoAlmacenImagenEvidenciaAnalisisRef] = i.EvidenciaAnalisisRef
	}
	if objeto {
		v.ObjetoVinculado = ReferenciaObjetoAlmacen{Referencia: "objeto:imagen:0001", Version: "version:1"}
		a[AtributoAlmacenObjetoRef] = v.ObjetoVinculado.Referencia
		a[AtributoAlmacenObjetoVersion] = v.ObjetoVinculado.Version
	}
	datos, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	if superficie != domain.SuperficieAutenticacionInternaCorporativaV1 {
		original, err := datos.VinculoAutenticacionActor.Datos()
		if err != nil {
			t.Fatal(err)
		}
		autenticacion := original.Autenticacion()
		autenticacion.Superficie = superficie
		cuenta := domain.CuentaAutenticadaContextoActor{
			CuentaRef: e.resultado.Contexto.Instantanea.CuentaRef,
			Metodo:    e.resultado.Contexto.Principal.AuthMethod,
			Garantia:  e.resultado.Contexto.Principal.AuthAssurance,
		}
		vinculo, err := domain.CrearVinculoAutenticacionActorV2(
			context.Background(), revalidadorOrdenAutorizacionV3Prueba{autenticacion},
			domain.SolicitudRevalidacionAutenticacionActorV1{
				AutenticacionRef: autenticacion.AutenticacionRef, SesionRef: autenticacion.SesionRef,
			}, resolutorOrdenAutorizacionV3Prueba{e.resultado},
			domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: e.resultado.Contexto.PerfilActivoRef},
			relojOrdenAutorizacionV3Prueba{e.ahora})
		if err != nil {
			t.Fatal(err)
		}
		datos.VinculoAutenticacionActor = vinculo
	}
	datos.Accion = accion
	datos.Finalidad = i.Finalidad
	datos.Recurso = domain.RecursoAutorizable{
		Referencia: i.DocumentoRef, ModuloID: "documentos", Tipo: "imagen_usuario",
		Ambitos: map[string]string{"unidad": "seleccion"}, Atributos: a,
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(datos)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := e.instantnea
	instantanea.VersionRol.Concesiones = append([]domain.ConcesionRol(nil), instantanea.VersionRol.Concesiones...)
	instantanea.VersionRol.Concesiones[0].Accion = accion
	instantanea.VersionRol.Concesiones[0].ModuloID = "documentos"
	instantanea.VersionRol.Concesiones[0].TipoRecurso = "imagen_usuario"
	instantanea.VersionRol.Concesiones[0].Finalidades = []string{i.Finalidad}
	instantanea.VersionRol.Concesiones[0].CamposPermitidos = campos
	evidencia, err := domain.NuevaEvidenciaEvaluacionAutorizacionV3(
		solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", e.ahora,
		e.ahora.Add(90*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	orden, err := NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, e.motivo, e.resultado)
	if err != nil {
		t.Fatal(err)
	}
	confirmacion, err := nuevaConfirmacionRegistroConcesionAutorizacionLigadaV3(orden, e.ahora.Add(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}
	confirmada, err := confirmacion.Datos()
	if err != nil {
		t.Fatal(err)
	}
	huellaDecision, err := domain.HuellaSHA256DecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(e.motivo)
	if err != nil {
		t.Fatal(err)
	}
	huellaRecurso, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	resumen, err := NuevoResumenCapacidadAtestacionAutorizacionV3(
		confirmada.DecisionRef, huellaDecision, huellaMotivo, e.resultado.RegistroContextoRef,
		e.resultado.HuellaSHA256, accion, i.DocumentoRef, huellaRecurso,
		audienciaConsumoImagenV3(accion), e.ahora.Add(2*time.Microsecond), e.ahora.Add(4*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decisionCanonica, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		t.Fatal(err)
	}
	motivoCanonico, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(e.motivo)
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	material, err := NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte("c"), 512), resumen, decisionCanonica, motivoCanonico,
		e.resultado.RepresentacionCanonica, e.resultado.Contexto.Instantanea.PersonaVersion,
		e.resultado.Contexto.Instantanea.PerfilVersion, []byte("payload"), []byte("cose"),
		[]byte("verificacion"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return AutorizacionImagenAlmacenV3{Solicitud: solicitud, Decision: decision,
		Confirmacion: confirmacion, ContextoActor: e.resultado, Motivo: e.motivo,
		Material: material}, v, i, e.ahora.Add(3 * time.Microsecond)
}

func TestImagenAlmacenV3EscrituraExactaYSinDegradacion(t *testing.T) {
	a, v, i, instante := imagenAlmacenV3Prueba(t, AccionNegocioEscribirImagenProcesada)
	c, err := NuevoContextoEscribirImagenProcesadaAlmacen(a, v, i, instante)
	if err != nil || c.ValidarParaEn(AccionAlmacenEscribir, instante) != nil {
		t.Fatalf("V3: %v", err)
	}
	if _, err := c.EvidenciaAutorizacion(); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatal("V1 expuesta")
	}
	if _, err := c.MaterialAutorizacionImagenV3(); err != nil {
		t.Fatalf("material V3: %v", err)
	}
	s := SolicitudEscribirObjeto{Contexto: c, ClaveIdempotencia: i.ClaveIdempotencia,
		Zona: ZonaAlmacenCuarentena, MIME: "image/png", Tamano: i.Tamano,
		HuellaSHA256: i.HuellaSHA256, Contenido: strings.NewReader("png procesado")}
	if s.Validar() != nil {
		t.Fatal("escritura exacta denegada")
	}
	for _, mutar := range []func(*SolicitudEscribirObjeto){
		func(s *SolicitudEscribirObjeto) { s.ClaveIdempotencia = "otra:clave" },
		func(s *SolicitudEscribirObjeto) { s.Zona = ZonaAlmacenAdmitida },
		func(s *SolicitudEscribirObjeto) { s.MIME = "image/jpeg" },
		func(s *SolicitudEscribirObjeto) { s.Tamano++ },
		func(s *SolicitudEscribirObjeto) { s.HuellaSHA256 = strings.Repeat("b", 64) },
	} {
		mutada := s
		mutar(&mutada)
		if mutada.Validar() == nil {
			t.Fatal("escritura alterada aceptada")
		}
	}
	if _, err := NuevoContextoEscribirImagenProcesadaAlmacen(AutorizacionImagenAlmacenV3{}, v, i, instante); err == nil {
		t.Fatal("sin V3")
	}
	if _, err := c.DerivarPaso(PasoAlmacenPromoverImagenProcesada); err == nil {
		t.Fatal("paso cruzado")
	}
}

func TestImagenAlmacenV3PromocionLigaClaveYAnalisis(t *testing.T) {
	a, v, i, instante := imagenAlmacenV3Prueba(t, AccionNegocioPromoverImagenProcesada)
	c, err := NuevoContextoPromoverImagenProcesadaAlmacen(a, v, i, instante)
	if err != nil {
		t.Fatal(err)
	}
	s := SolicitudPromoverObjeto{Contexto: c, Origen: v.ObjetoVinculado,
		ClaveIdempotencia: i.ClaveIdempotencia, EvidenciaAnalisisRef: i.EvidenciaAnalisisRef}
	if s.Validar() != nil {
		t.Fatal("promocion exacta denegada")
	}
	s.ClaveIdempotencia = "otra:clave"
	if s.Validar() == nil {
		t.Fatal("clave de destino cambiada")
	}
	s.ClaveIdempotencia = i.ClaveIdempotencia
	s.EvidenciaAnalisisRef = "analisis:otro"
	if s.Validar() == nil {
		t.Fatal("evidencia distinta aceptada")
	}
	v.ObjetoVinculado.Version = "version:otra"
	if _, err := NuevoContextoPromoverImagenProcesadaAlmacen(a, v, i, instante); err == nil {
		t.Fatal("version ajena")
	}
	v.ObjetoVinculado.Version = "version:1"
	i.EvidenciaAnalisisRef = "analisis:otro"
	if _, err := NuevoContextoPromoverImagenProcesadaAlmacen(a, v, i, instante); err == nil {
		t.Fatal("evidencia del recurso cambiada")
	}
	i.EvidenciaAnalisisRef = "analisis:limpio:0001"
	i.HuellaSHA256 = strings.Repeat("b", 64)
	if _, err := NuevoContextoPromoverImagenProcesadaAlmacen(a, v, i, instante); err == nil {
		t.Fatal("huella del recurso cambiada")
	}
}

func TestImagenAlmacenV3DeniegaAudienciaYLecturaAjenaExterior(t *testing.T) {
	a, v, i, instante := imagenAlmacenV3Prueba(t, AccionNegocioAbrirImagenAjenaActiva)
	c, err := NuevoContextoAbrirImagenActivaAlmacen(a, v, i, instante)
	if err != nil || c.ValidarParaEn(AccionAlmacenLeer, instante) != nil {
		t.Fatalf("lectura V3: %v", err)
	}
	i.Audiencia = audienciaImagenPersonal
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(a, v, i, instante); err == nil {
		t.Fatal("externa ajena")
	}
	i.Audiencia = "mi_bolsa_publica"
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(a, v, i, instante); err == nil {
		t.Fatal("publica")
	}
	i.Audiencia = audienciaImagenInterna
	otra, _, _, _ := imagenAlmacenV3Prueba(t, AccionNegocioAbrirImagenPropiaActiva)
	a.Material = otra.Material
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(a, v, i, instante); err == nil {
		t.Fatal("material de audiencia propia aceptado para ajena")
	}
	a.Material = ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(a, v, i, instante); err == nil {
		t.Fatal("sin material")
	}
	externa, vinculosExternos, imagenExterna, ahoraExterno := imagenAlmacenV3Prueba(
		t, AccionNegocioAbrirImagenAjenaActiva,
		domain.SuperficieAutenticacionExternaPersonalV1)
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(
		externa, vinculosExternos, imagenExterna, ahoraExterno); err == nil {
		t.Fatal("vinculo exterior aceptado para lectura ajena interna")
	}
	propiaExterna, vinculosPropios, imagenPropia, ahoraPropio := imagenAlmacenV3Prueba(
		t, AccionNegocioAbrirImagenPropiaActiva,
		domain.SuperficieAutenticacionExternaPersonalV1)
	if _, err := NuevoContextoAbrirImagenActivaAlmacen(
		propiaExterna, vinculosPropios, imagenPropia, ahoraPropio); err != nil {
		t.Fatalf("lectura propia exterior válida: %v", err)
	}
	if superficieImagenCoincide(domain.SuperficieAutenticacionAdministracionPrivilegiadaV1, audienciaImagenInterna) ||
		superficieImagenCoincide(domain.SuperficieAutenticacionAdministracionPrivilegiadaV1, audienciaImagenPersonal) {
		t.Fatal("administración privilegiada convertida en audiencia de imagen")
	}
}
