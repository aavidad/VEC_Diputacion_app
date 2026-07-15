package memory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type relojAlmacenPrueba struct {
	mu    sync.RWMutex
	ahora time.Time
}

func (r *relojAlmacenPrueba) Ahora() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ahora
}

func (r *relojAlmacenPrueba) Avanzar(duracion time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ahora = r.ahora.Add(duracion)
}

func instanteAlmacenPrueba() time.Time {
	return time.Date(2026, time.July, 14, 20, 0, 0, 0, time.UTC)
}

func contextoAlmacenPrueba(
	t *testing.T,
	sufijo string,
	accionTecnica string,
	objeto ports.ReferenciaObjetoAlmacen,
) ports.ContextoOperacionAlmacen {
	t.Helper()
	instante := instanteAlmacenPrueba()
	accionNegocio := ports.AccionNegocioCustodiarDecisionBaremacion
	campos := []string{"documento_custodiado", "evidencia_custodia"}
	requiereObjeto := false
	segunAccion := func(
		decision domain.DecisionAutorizacion,
		recurso domain.RecursoAutorizable,
		vinculos ports.VinculosOperacionAlmacen,
	) (ports.ContextoOperacionAlmacen, error) {
		return ports.NuevoContextoCustodiarDecisionBaremacionAlmacen(decision, recurso, vinculos, instante)
	}
	switch accionTecnica {
	case ports.AccionAlmacenEscribir:
	case ports.AccionAlmacenLeer:
		accionNegocio = ports.AccionNegocioAnalizarCargaDocumental
		campos = []string{"analisis_seguridad", "estado"}
		requiereObjeto = true
		segunAccion = func(
			decision domain.DecisionAutorizacion,
			recurso domain.RecursoAutorizable,
			vinculos ports.VinculosOperacionAlmacen,
		) (ports.ContextoOperacionAlmacen, error) {
			return ports.NuevoContextoAnalizarCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
		}
	case ports.AccionAlmacenPromover:
		accionNegocio = ports.AccionNegocioPromoverCargaDocumental
		campos = []string{"contenido_admitido", "estado"}
		requiereObjeto = true
		segunAccion = func(
			decision domain.DecisionAutorizacion,
			recurso domain.RecursoAutorizable,
			vinculos ports.VinculosOperacionAlmacen,
		) (ports.ContextoOperacionAlmacen, error) {
			return ports.NuevoContextoPromoverCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
		}
	case ports.AccionAlmacenAplicarRetencion:
		accionNegocio = ports.AccionNegocioRetenerDocumentoFirmado
		campos = []string{"documento_firmado.retencion", "evidencia_retencion"}
		requiereObjeto = true
		segunAccion = func(
			decision domain.DecisionAutorizacion,
			recurso domain.RecursoAutorizable,
			vinculos ports.VinculosOperacionAlmacen,
		) (ports.ContextoOperacionAlmacen, error) {
			return ports.NuevoContextoRetenerDocumentoFirmadoAlmacen(decision, recurso, vinculos, instante)
		}
	default:
		t.Fatalf("la prueba intento acuñar una capacidad tecnica no publicada: %s", accionTecnica)
	}
	if requiereObjeto && objeto.Validar() != nil {
		t.Fatal("la capacidad de prueba exige objeto y version exactos")
	}
	vinculos := ports.VinculosOperacionAlmacen{
		OperacionRef: "operacion:" + sufijo, CargaRef: "carga:" + sufijo,
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_v1:" + strings.Repeat("1", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_v1:" + strings.Repeat("2", 64),
		EfectoRef:           "efecto:" + sufijo,
		ObjetoVinculado:     objeto,
	}
	atributos := map[string]string{
		ports.AtributoAlmacenOperacionRef:        vinculos.OperacionRef,
		ports.AtributoAlmacenCargaRef:            vinculos.CargaRef,
		ports.AtributoAlmacenClasificacion:       vinculos.Clasificacion,
		ports.AtributoAlmacenSujetoSeudonimoHMAC: vinculos.SujetoSeudonimoHMAC,
		ports.AtributoAlmacenHuellaSolicitudHMAC: vinculos.HuellaSolicitudHMAC,
		ports.AtributoAlmacenEfectoRef:           vinculos.EfectoRef,
	}
	if requiereObjeto {
		atributos[ports.AtributoAlmacenObjetoRef] = objeto.Referencia
		atributos[ports.AtributoAlmacenObjetoVersion] = objeto.Version
	}
	recurso := domain.RecursoAutorizable{
		Referencia: "solicitud-bolsa:" + sufijo, ModuloID: "bolsa", Tipo: "documento_bolsa",
		Ambitos: map[string]string{"organizacion": "diputacion_granada"}, Atributos: atributos,
	}
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	_, vinculoActor, err := pruebasvec.NuevoContextoYVinculo(
		instante, "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh,
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaCatalogo, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	decision := domain.DecisionAutorizacion{
		DecisionRef: "decision:" + sufijo, Concedida: true, Codigo: "concedida",
		PrincipalID: "per_0123456789abcdefghijkl", PerfilActivoRef: "prf_0123456789abcdefghijkl",
		Accion: accionNegocio, RecursoRef: recurso.Referencia, ModuloID: recurso.ModuloID,
		TipoRecurso: recurso.Tipo, ContextoRecursoHuellaSHA256: huellaRecurso,
		Finalidad: "tramitar_documentacion_bolsa", CorrelacionRef: "correlacion:" + sufijo,
		VinculoAutenticacionActor: vinculoActor,
		AsignacionRef:             "asignacion:almacen:v1", AsignacionHuellaSHA256: strings.Repeat("a", 64),
		VersionRolRef: "rol:almacen:v1", VersionRolHuellaSHA256: strings.Repeat("b", 64),
		ControlVigenciaVersionRolRef: "rol:almacen:v1", ControlVigenciaVersionRolRevision: 1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("c", 64),
		RevisionCatalogoPoliticas:             1, CatalogoPoliticasHuellaSHA256: huellaCatalogo,
		PoliticasEvaluadasHuellasSHA256: map[string]string{}, GarantiaMinima: domain.AuthAssuranceHigh,
		CamposPermitidos: campos, EmitidaEn: instante, ValidaHasta: instante.Add(5 * time.Minute),
	}
	contexto, err := segunAccion(decision, recurso, vinculos)
	if err != nil {
		t.Fatalf("crear capacidad opaca de almacen: %v", err)
	}
	return contexto
}

func solicitudEscrituraAlmacenPrueba(
	contenido []byte,
	clave string,
	contexto ports.ContextoOperacionAlmacen,
) ports.SolicitudEscribirObjeto {
	suma := sha256.Sum256(contenido)
	return ports.SolicitudEscribirObjeto{
		Contexto: contexto, ClaveIdempotencia: clave, Zona: ports.ZonaAlmacenCuarentena,
		MIME: "application/pdf", Tamano: int64(len(contenido)), HuellaSHA256: hex.EncodeToString(suma[:]),
		Contenido: bytes.NewReader(contenido),
	}
}

func nuevoAlmacenObjetosPrueba(t *testing.T) (*AlmacenObjetosMemoria, *relojAlmacenPrueba) {
	t.Helper()
	reloj := &relojAlmacenPrueba{ahora: instanteAlmacenPrueba()}
	almacen, err := NuevoAlmacenObjetosMemoria("memoria-pruebas", 4*1024*1024, reloj)
	if err != nil {
		t.Fatalf("crear almacen de prueba: %v", err)
	}
	return almacen, reloj
}

func TestAlmacenObjetosMemoriaDeclaraLimitesSinFingirPerfilProductivo(t *testing.T) {
	almacen, _ := nuevoAlmacenObjetosPrueba(t)
	capacidades, err := almacen.Capacidades(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := ports.VerificarCapacidadesAlmacen(capacidades, ports.RequisitosAlmacenObjetos{
		EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true,
		IntegridadSHA256: true, PreservaObjetoOriginal: true, TamanoMinimoObjeto: 1024,
	}); err != nil {
		t.Fatalf("perfil de pruebas compatible: %v", err)
	}
	if err := ports.VerificarCapacidadesAlmacen(capacidades, ports.RequisitosAlmacenObjetos{
		CifradoEnTransito: true, CifradoEnReposo: true, CifradoPorObjeto: true, CargaDirectaTemporal: true,
	}); !errors.Is(err, ports.ErrCapacidadAlmacenNoDisponible) {
		t.Fatalf("el adaptador de memoria fingio aptitud productiva: %v", err)
	}
}

func TestAlmacenObjetosMemoriaEscrituraIdempotenteLigaCapacidadCompleta(t *testing.T) {
	almacen, _ := nuevoAlmacenObjetosPrueba(t)
	contenido := []byte("%PDF-1.7\nexpediente opaco\n%%EOF")
	contexto := contextoAlmacenPrueba(t, "alta-001", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{})
	solicitud := solicitudEscrituraAlmacenPrueba(contenido, "idempotencia-carga-001", contexto)
	guardado, err := almacen.Escribir(context.Background(), solicitud)
	if err != nil || guardado.Validar() != nil {
		t.Fatalf("escritura valida: %+v, %v", guardado, err)
	}
	exigirEvidenciaAlmacenLigada(t, guardado.Evidencia, contexto, "", false)
	solicitud.Contenido = bytes.NewReader(contenido)
	repetida, err := almacen.Escribir(context.Background(), solicitud)
	if err != nil || repetida.Objeto.Objeto != guardado.Objeto.Objeto || !repetida.Evidencia.ReintentoIdempotente {
		t.Fatalf("reintento idempotente: %+v, %v", repetida, err)
	}
	exigirEvidenciaAlmacenLigada(t, repetida.Evidencia, contexto, "", true)

	otroContexto := contextoAlmacenPrueba(t, "alta-otra", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{})
	solicitud.Contexto, solicitud.Contenido = otroContexto, bytes.NewReader(contenido)
	if _, err := almacen.Escribir(context.Background(), solicitud); !errors.Is(err, ports.ErrIdempotenciaAlmacenReutilizada) {
		t.Fatalf("una misma clave cruzo capacidades: %v", err)
	}
}

func TestAlmacenObjetosMemoriaRevalidaVigenciaJustoAntesDelEfecto(t *testing.T) {
	almacen, reloj := nuevoAlmacenObjetosPrueba(t)
	contenido := []byte("documento que no debe persistirse")
	contexto := contextoAlmacenPrueba(t, "caducada", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{})
	reloj.Avanzar(5 * time.Minute)
	_, err := almacen.Escribir(context.Background(), solicitudEscrituraAlmacenPrueba(
		contenido, "idempotencia-caducada", contexto,
	))
	if !errors.Is(err, ports.ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("capacidad caducada no se denego: %v", err)
	}
	almacen.mu.RLock()
	defer almacen.mu.RUnlock()
	if len(almacen.objetos) != 0 || len(almacen.idempotencias) != 0 {
		t.Fatal("una denegacion temporal produjo efectos")
	}
}

func TestAlmacenObjetosMemoriaLecturaYPromocionExigenObjetoVersionExactos(t *testing.T) {
	almacen, _ := nuevoAlmacenObjetosPrueba(t)
	contenido := []byte("contenido limpio de cuarentena")
	alta := contextoAlmacenPrueba(t, "alta-flujo", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{})
	guardado, err := almacen.Escribir(context.Background(), solicitudEscrituraAlmacenPrueba(
		contenido, "idempotencia-alta-flujo", alta,
	))
	if err != nil {
		t.Fatal(err)
	}
	objeto := guardado.Objeto.Objeto
	lectura := abrirObjetoAlmacenPrueba(t, almacen, objeto, ports.ZonaAlmacenCuarentena, contenido)
	defer lectura.Contenido.Close()

	objetoDistinto := objeto
	objetoDistinto.Version = "2"
	contextoOtroObjeto := contextoAlmacenPrueba(t, "lectura-otra-version", ports.AccionAlmacenLeer, objetoDistinto)
	if _, err := almacen.Abrir(context.Background(), ports.SolicitudAbrirObjeto{
		Contexto: contextoOtroObjeto, Objeto: objeto, Zona: ports.ZonaAlmacenCuarentena, Limite: int64(len(contenido)),
	}); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("una capacidad para otra version leyo el objeto: %v", err)
	}

	contextoPromocion := contextoAlmacenPrueba(t, "promocion", ports.AccionAlmacenPromover, objeto)
	solicitudPromocion := ports.SolicitudPromoverObjeto{
		Contexto: contextoPromocion, ClaveIdempotencia: "idempotencia-promocion",
		Origen: objeto, EvidenciaAnalisisRef: "analisis-antivirus-limpio-001",
	}
	promovido, err := almacen.Promover(context.Background(), solicitudPromocion)
	if err != nil || promovido.Validar() != nil || promovido.Objeto.Zona != ports.ZonaAlmacenAdmitida ||
		promovido.Objeto.Objeto == objeto || promovido.Objeto.HuellaSHA256 != guardado.Objeto.HuellaSHA256 {
		t.Fatalf("promocion: %+v, %v", promovido, err)
	}
	exigirEvidenciaAlmacenLigada(
		t, promovido.Evidencia, contextoPromocion, solicitudPromocion.EvidenciaAnalisisRef, false,
	)
	_ = abrirObjetoAlmacenPrueba(t, almacen, promovido.Objeto.Objeto, ports.ZonaAlmacenAdmitida, contenido).Contenido.Close()
}

func TestAlmacenObjetosMemoriaRetencionExactaNoAcortaConservacion(t *testing.T) {
	almacen, reloj := nuevoAlmacenObjetosPrueba(t)
	contenido := []byte("documento firmado")
	guardado, err := almacen.Escribir(context.Background(), solicitudEscrituraAlmacenPrueba(
		contenido, "idempotencia-firmado",
		contextoAlmacenPrueba(t, "custodia-firmado", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{}),
	))
	if err != nil {
		t.Fatal(err)
	}
	hasta := reloj.Ahora().Add(24 * time.Hour)
	contextoRetencion := contextoAlmacenPrueba(t, "retencion-firmado", ports.AccionAlmacenAplicarRetencion, guardado.Objeto.Objeto)
	solicitud := ports.SolicitudRetenerObjeto{
		Contexto: contextoRetencion, Objeto: guardado.Objeto.Objeto,
		PoliticaRef: "politica-conservacion-expediente-v3", Hasta: hasta,
	}
	retenido, err := almacen.AplicarRetencion(context.Background(), solicitud)
	if err != nil || !retenido.Objeto.RetenidoHasta.Equal(hasta) ||
		retenido.ValidarRetencion(solicitud, guardado.Objeto) != nil {
		t.Fatalf("retencion: %+v, %v", retenido, err)
	}
	exigirEvidenciaAlmacenLigada(t, retenido.Evidencia, contextoRetencion, solicitud.PoliticaRef, false)
	solicitud.Hasta = hasta.Add(-time.Hour)
	if _, err := almacen.AplicarRetencion(context.Background(), solicitud); !errors.Is(err, ports.ErrRetencionObjetoAlmacenVigente) {
		t.Fatalf("se acorto la retencion: %v", err)
	}
}

func TestAlmacenObjetosMemoriaDetectaContenidoAlterado(t *testing.T) {
	almacen, _ := nuevoAlmacenObjetosPrueba(t)
	contenido := []byte("contenido original")
	contexto := contextoAlmacenPrueba(t, "integridad", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{})
	solicitud := solicitudEscrituraAlmacenPrueba(contenido, "idempotencia-integridad", contexto)
	solicitud.HuellaSHA256 = strings.Repeat("0", 64)
	if _, err := almacen.Escribir(context.Background(), solicitud); !errors.Is(err, ports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("huella falsa aceptada: %v", err)
	}
	solicitud = solicitudEscrituraAlmacenPrueba(contenido, "idempotencia-integridad-valida", contexto)
	guardado, err := almacen.Escribir(context.Background(), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	almacen.mu.Lock()
	clave := claveObjetoAlmacen(guardado.Objeto.Objeto)
	alterado := almacen.objetos[clave]
	alterado.contenido[0] ^= 0xff
	almacen.objetos[clave] = alterado
	almacen.mu.Unlock()
	contextoLectura := contextoAlmacenPrueba(t, "lectura-alterada", ports.AccionAlmacenLeer, guardado.Objeto.Objeto)
	if _, err := almacen.Abrir(context.Background(), ports.SolicitudAbrirObjeto{
		Contexto: contextoLectura, Objeto: guardado.Objeto.Objeto,
		Zona: ports.ZonaAlmacenCuarentena, Limite: int64(len(contenido)),
	}); !errors.Is(err, ports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("contenido alterado no detectado: %v", err)
	}
}

func TestAlmacenObjetosMemoriaIdempotenciaConcurrente(t *testing.T) {
	almacen, _ := nuevoAlmacenObjetosPrueba(t)
	contenido := []byte("contenido concurrente")
	contexto := contextoAlmacenPrueba(t, "concurrente", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{})
	const intentos = 24
	resultados := make(chan ports.ResultadoOperacionObjeto, intentos)
	errores := make(chan error, intentos)
	var grupo sync.WaitGroup
	for range intentos {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			resultado, err := almacen.Escribir(context.Background(), solicitudEscrituraAlmacenPrueba(
				contenido, "idempotencia-concurrente", contexto,
			))
			if err != nil {
				errores <- err
				return
			}
			resultados <- resultado
		}()
	}
	grupo.Wait()
	close(resultados)
	close(errores)
	for err := range errores {
		t.Fatal(err)
	}
	var referencia ports.ReferenciaObjetoAlmacen
	reintentos := 0
	cuenta := 0
	for resultado := range resultados {
		if cuenta == 0 {
			referencia = resultado.Objeto.Objeto
		} else if resultado.Objeto.Objeto != referencia {
			t.Fatal("la idempotencia creo mas de un objeto")
		}
		if resultado.Evidencia.ReintentoIdempotente {
			reintentos++
		}
		cuenta++
	}
	if cuenta != intentos || reintentos != intentos-1 {
		t.Fatalf("resultados=%d reintentos=%d", cuenta, reintentos)
	}
}

func TestAlmacenObjetosMemoriaOperacionesSinFabricaPublicadaSeDeniegan(t *testing.T) {
	almacen, _ := nuevoAlmacenObjetosPrueba(t)
	contenido := []byte("objeto que debe permanecer intacto")
	guardado, err := almacen.Escribir(context.Background(), solicitudEscrituraAlmacenPrueba(
		contenido, "idempotencia-sin-ampliacion",
		contextoAlmacenPrueba(t, "alta-sin-ampliacion", ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{}),
	))
	if err != nil {
		t.Fatal(err)
	}
	objeto := guardado.Objeto.Objeto
	if _, err := almacen.Inmovilizar(context.Background(), ports.SolicitudInmovilizarObjeto{
		Contexto: ports.ContextoOperacionAlmacen{}, Objeto: objeto,
		AprobacionRef: "aprobacion-no-autorizada", Motivo: "sin capacidad positiva publicada",
	}); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("inmovilizacion sin capacidad: %v", err)
	}
	if _, err := almacen.Eliminar(context.Background(), ports.SolicitudEliminarObjeto{
		Contexto: ports.ContextoOperacionAlmacen{}, Objeto: objeto,
		AprobacionRef: "aprobacion-no-autorizada", Motivo: "sin capacidad positiva publicada",
	}); !errors.Is(err, ports.ErrSolicitudAlmacenInvalida) {
		t.Fatalf("eliminacion sin capacidad: %v", err)
	}
	_ = abrirObjetoAlmacenPrueba(t, almacen, objeto, ports.ZonaAlmacenCuarentena, contenido).Contenido.Close()
}

func abrirObjetoAlmacenPrueba(
	t *testing.T,
	almacen ports.AlmacenObjetos,
	objeto ports.ReferenciaObjetoAlmacen,
	zona ports.ZonaAlmacen,
	esperado []byte,
) ports.LecturaObjetoAlmacen {
	t.Helper()
	contexto := contextoAlmacenPrueba(t, "lectura-"+objeto.Referencia, ports.AccionAlmacenLeer, objeto)
	lectura, err := almacen.Abrir(context.Background(), ports.SolicitudAbrirObjeto{
		Contexto: contexto, Objeto: objeto, Zona: zona, Limite: int64(len(esperado)),
	})
	if err != nil {
		t.Fatalf("abrir objeto: %v", err)
	}
	contenido, err := io.ReadAll(lectura.Contenido)
	if err != nil || !bytes.Equal(contenido, esperado) {
		_ = lectura.Contenido.Close()
		t.Fatalf("contenido leido=%q err=%v", contenido, err)
	}
	return lectura
}

func exigirEvidenciaAlmacenLigada(
	t *testing.T,
	evidencia ports.EvidenciaOperacionAlmacen,
	contexto ports.ContextoOperacionAlmacen,
	fundamentoRef string,
	reintento bool,
) {
	t.Helper()
	proyeccion, err := contexto.Proyeccion()
	if err != nil || evidencia.EsquemaContexto != proyeccion.Esquema ||
		evidencia.OperacionRef != proyeccion.OperacionRef ||
		evidencia.CorrelacionRef != proyeccion.CorrelacionRef ||
		evidencia.AutorizacionRef != proyeccion.AutorizacionRef ||
		evidencia.Finalidad != proyeccion.Finalidad || evidencia.Clasificacion != proyeccion.Clasificacion ||
		evidencia.AccionNegocio != proyeccion.AccionNegocio || evidencia.Accion != proyeccion.AccionTecnica ||
		evidencia.EfectoRef != proyeccion.EfectoRef ||
		evidencia.HuellaPlanEfectoSHA256 != proyeccion.HuellaPlanEfectoSHA256 ||
		evidencia.PasoRef != proyeccion.PasoRef || evidencia.HuellaDecisionSHA256 != proyeccion.HuellaDecisionSHA256 ||
		evidencia.CargaRef != proyeccion.CargaRef ||
		evidencia.SujetoSeudonimoHMAC != proyeccion.SujetoSeudonimoHMAC ||
		evidencia.RecursoRef != proyeccion.RecursoRef || evidencia.ModuloID != proyeccion.ModuloID ||
		evidencia.HuellaSolicitudHMAC != proyeccion.HuellaSolicitudHMAC ||
		evidencia.FundamentoRef != fundamentoRef || evidencia.ReintentoIdempotente != reintento {
		t.Fatalf("evidencia no ligada exactamente: %+v, %v", evidencia, err)
	}
}
