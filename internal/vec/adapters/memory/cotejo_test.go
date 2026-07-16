package memory

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	cotejoMemoriaPruebaSecreto        = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	cotejoMemoriaPruebaIndiceA        = "hmac-sha256:indice_cotejo_memoria:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	cotejoMemoriaPruebaIndiceB        = "hmac-sha256:indice_cotejo_memoria:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	cotejoMemoriaPruebaIndiceC        = "hmac-sha256:indice_cotejo_memoria:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	cotejoMemoriaPruebaSolicitudA     = "hmac-sha256:solicitud_cotejo_memoria:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	cotejoMemoriaPruebaSolicitudB     = "hmac-sha256:solicitud_cotejo_memoria:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	cotejoMemoriaPruebaSolicitudC     = "hmac-sha256:solicitud_cotejo_memoria:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	cotejoMemoriaPruebaProteccionA    = "proteccion-cotejo-memoria-001"
	cotejoMemoriaPruebaProteccionB    = "proteccion-cotejo-memoria-002"
	cotejoMemoriaPruebaProteccionC    = "proteccion-cotejo-memoria-003"
	cotejoMemoriaPruebaExpediente     = "expediente-cotejo-memoria-001"
	cotejoMemoriaPruebaPrincipal      = "tecnico-rrhh-cotejo-memoria-001"
	cotejoMemoriaPruebaPoliticaID     = "politica_cotejo_memoria"
	cotejoMemoriaPruebaCorrelacionRef = "correlacion-cotejo-memoria-001"
)

func TestCotejoMemoriaGobiernoPoliticasEsAtomicoVersionadoYOptimista(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	ahora := time.Date(2026, time.July, 14, 9, 0, 0, 0, time.UTC)
	borrador := cotejoMemoriaPruebaPoliticaBorrador(1, ahora)

	traza, evento := cotejoMemoriaPruebaEvidenciaPolitica(borrador, domain.AccionPoliticaCotejoBorradorCreada, "")
	if err := store.ConfirmarAltaBorradorPoliticaCotejo(ctx, borrador, traza, evento); err != nil {
		t.Fatalf("confirmar alta de politica: %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)

	if err := store.ConfirmarAltaBorradorPoliticaCotejo(ctx, borrador, traza, evento); !errors.Is(err, ports.ErrVersionPoliticaCotejoYaExiste) {
		t.Fatalf("alta duplicada: error = %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)

	huellaInicial, err := borrador.HuellaSHA256()
	if err != nil {
		t.Fatalf("calcular huella inicial: %v", err)
	}
	propuesta := borrador
	propuesta.Descripcion = "Politica actualizada para verificar documentos de recursos humanos"
	actualizada, err := borrador.ActualizarBorrador(
		propuesta, "responsable-rrhh-cotejo-memoria-001", "actualizacion revisada de la politica", ahora.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("crear actualizacion de politica: %v", err)
	}
	traza, evento = cotejoMemoriaPruebaEvidenciaPolitica(actualizada, domain.AccionPoliticaCotejoBorradorActualizada, huellaInicial)
	if err := store.ConfirmarActualizacionBorradorPoliticaCotejo(ctx, huellaInicial, actualizada, traza, evento); err != nil {
		t.Fatalf("confirmar actualizacion: %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 2)

	if err := store.ConfirmarActualizacionBorradorPoliticaCotejo(ctx, huellaInicial, actualizada, traza, evento); !errors.Is(err, ports.ErrRevisionPoliticaCotejoConflicto) {
		t.Fatalf("repeticion con huella obsoleta: error = %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 2)

	huellaActualizada, err := actualizada.HuellaSHA256()
	if err != nil {
		t.Fatalf("calcular huella actualizada: %v", err)
	}
	publicada, err := actualizada.Publicar(
		"responsable-seguridad-cotejo-memoria-001", "aprobacion-cotejo-memoria-001",
		"publicacion aprobada de la politica", ahora.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("crear publicacion: %v", err)
	}
	traza, evento = cotejoMemoriaPruebaEvidenciaPolitica(publicada, domain.AccionPoliticaCotejoPublicada, huellaActualizada)
	if err := store.ConfirmarPublicacionPoliticaCotejo(ctx, huellaActualizada, publicada, traza, evento); err != nil {
		t.Fatalf("confirmar publicacion: %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 3)

	borradorV2 := cotejoMemoriaPruebaPoliticaBorrador(2, ahora.Add(3*time.Minute))
	traza, evento = cotejoMemoriaPruebaEvidenciaPolitica(borradorV2, domain.AccionPoliticaCotejoBorradorCreada, "")
	if err := store.ConfirmarAltaBorradorPoliticaCotejo(ctx, borradorV2, traza, evento); err != nil {
		t.Fatalf("confirmar version 2: %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 4)

	borradorV3 := cotejoMemoriaPruebaPoliticaBorrador(3, ahora.Add(4*time.Minute))
	traza, evento = cotejoMemoriaPruebaEvidenciaPolitica(borradorV3, domain.AccionPoliticaCotejoBorradorCreada, "")
	if err := store.ConfirmarAltaBorradorPoliticaCotejo(ctx, borradorV3, traza, evento); !errors.Is(err, ports.ErrSecuenciaPoliticaCotejoInvalida) {
		t.Fatalf("version 3 con version 2 no publicada: error = %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 4)

	versiones, err := store.ListarVersionesPoliticaCotejo(ctx, cotejoMemoriaPruebaPoliticaID)
	if err != nil || len(versiones) != 2 || versiones[0].Version != 1 || versiones[1].Version != 2 ||
		versiones[0].Estado != domain.EstadoPoliticaCotejoPublicada || versiones[1].Estado != domain.EstadoPoliticaCotejoBorrador {
		t.Fatalf("versiones guardadas = %+v; error = %v", versiones, err)
	}
	versiones[0].Descripcion = "mutacion externa"
	guardada, err := store.ObtenerPoliticaCotejo(ctx, cotejoMemoriaPruebaPoliticaID, 1)
	if err != nil || guardada.Descripcion == "mutacion externa" {
		t.Fatalf("la lectura comparte memoria con el almacen: %+v, %v", guardada, err)
	}
}

func TestCotejoMemoriaConfirmacionImponeUnicidadDeDocumentoEIndice(t *testing.T) {
	store := NewStore()
	ahora := time.Date(2026, time.July, 14, 11, 0, 0, 0, time.UTC)
	documentoA := domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-unicidad-001", Version: 1}
	documentoB := domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-unicidad-002", Version: 1}
	documentoC := domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-unicidad-003", Version: 1}

	codigoA := cotejoMemoriaPruebaCodigoReservado(
		"codigo-cotejo-memoria-unicidad-001", documentoA, cotejoMemoriaPruebaIndiceA,
		cotejoMemoriaPruebaProteccionA, ahora,
	)
	if err := cotejoMemoriaPruebaReservarYConfirmar(
		t, store, "idempotencia-cotejo-memoria-unicidad-001", cotejoMemoriaPruebaSolicitudA, codigoA,
	); err != nil {
		t.Fatalf("confirmar codigo inicial: %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)

	codigoMismoDocumento := cotejoMemoriaPruebaCodigoReservado(
		"codigo-cotejo-memoria-unicidad-002", documentoA, cotejoMemoriaPruebaIndiceB,
		cotejoMemoriaPruebaProteccionB, ahora.Add(time.Minute),
	)
	if err := cotejoMemoriaPruebaReservarYConfirmar(
		t, store, "idempotencia-cotejo-memoria-unicidad-002", cotejoMemoriaPruebaSolicitudB, codigoMismoDocumento,
	); !errors.Is(err, ports.ErrDocumentoConCodigoCotejo) {
		t.Fatalf("segundo codigo para el documento: error = %v", err)
	}
	cotejoMemoriaPruebaExigirSinParcialidadCodigo(t, store, codigoMismoDocumento, 1)

	codigoMismoIndice := cotejoMemoriaPruebaCodigoReservado(
		"codigo-cotejo-memoria-unicidad-003", documentoB, cotejoMemoriaPruebaIndiceA,
		cotejoMemoriaPruebaProteccionC, ahora.Add(2*time.Minute),
	)
	if err := cotejoMemoriaPruebaReservarYConfirmar(
		t, store, "idempotencia-cotejo-memoria-unicidad-003", cotejoMemoriaPruebaSolicitudC, codigoMismoIndice,
	); !errors.Is(err, ports.ErrIndiceCodigoCotejoYaExiste) {
		t.Fatalf("segundo codigo para el indice: error = %v", err)
	}
	cotejoMemoriaPruebaExigirSinParcialidadCodigo(t, store, codigoMismoIndice, 1)

	codigoMismoID := cotejoMemoriaPruebaCodigoReservado(
		codigoA.ID, documentoC, cotejoMemoriaPruebaIndiceC,
		"proteccion-cotejo-memoria-004", ahora.Add(3*time.Minute),
	)
	if err := cotejoMemoriaPruebaReservarYConfirmar(
		t, store, "idempotencia-cotejo-memoria-unicidad-004", cotejoMemoriaPruebaSolicitudB, codigoMismoID,
	); !errors.Is(err, ports.ErrCodigoCotejoYaExiste) {
		t.Fatalf("segundo agregado con el mismo id: error = %v", err)
	}
	cotejoMemoriaPruebaExigirSinParcialidadCodigo(t, store, codigoMismoID, 1)
}

func TestCotejoMemoriaBusquedaAdmiteRotacionYDetectaAmbiguedad(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	ahora := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	codigoHistorico := cotejoMemoriaPruebaCodigoReservado(
		"codigo-cotejo-memoria-busqueda-001",
		domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-busqueda-001", Version: 1},
		cotejoMemoriaPruebaIndiceA, cotejoMemoriaPruebaProteccionA, ahora,
	)
	cotejoMemoriaPruebaSembrarCodigo(store, codigoHistorico)

	encontrado, err := store.BuscarCodigoCotejoPorIndices(ctx, []string{cotejoMemoriaPruebaIndiceB, cotejoMemoriaPruebaIndiceA})
	if err != nil || !reflect.DeepEqual(encontrado, codigoHistorico) {
		t.Fatalf("busqueda con indice historico = %+v, %v", encontrado, err)
	}
	encontrado.MotivoReserva = "mutacion externa"
	guardado, err := store.ObtenerCodigoCotejo(ctx, codigoHistorico.ID)
	if err != nil || guardado.MotivoReserva == "mutacion externa" {
		t.Fatalf("la busqueda comparte memoria con el almacen: %+v, %v", guardado, err)
	}

	codigoActual := cotejoMemoriaPruebaCodigoReservado(
		"codigo-cotejo-memoria-busqueda-002",
		domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-busqueda-002", Version: 1},
		cotejoMemoriaPruebaIndiceB, cotejoMemoriaPruebaProteccionB, ahora,
	)
	cotejoMemoriaPruebaSembrarCodigo(store, codigoActual)
	if _, err := store.BuscarCodigoCotejoPorIndices(ctx, []string{cotejoMemoriaPruebaIndiceA, cotejoMemoriaPruebaIndiceB}); !errors.Is(err, ports.ErrIndicesCodigoCotejoAmbiguos) {
		t.Fatalf("indices de dos codigos: error = %v", err)
	}
	if _, err := store.BuscarCodigoCotejoPorIndices(ctx, []string{cotejoMemoriaPruebaIndiceA, cotejoMemoriaPruebaIndiceA}); !errors.Is(err, ports.ErrMaterialCodigoCotejoInvalido) {
		t.Fatalf("indices duplicados: error = %v", err)
	}
}

func TestCotejoMemoriaConsultasSonAuditadasYRechazanSecretos(t *testing.T) {
	ctx := context.Background()
	store := NewStore()
	ahora := time.Date(2026, time.July, 14, 13, 0, 0, 0, time.UTC)
	reservado := cotejoMemoriaPruebaCodigoReservado(
		"codigo-cotejo-memoria-consulta-001",
		domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-consulta-001", Version: 1},
		cotejoMemoriaPruebaIndiceA, cotejoMemoriaPruebaProteccionA, ahora.Add(-10*time.Minute),
	)
	activo := cotejoMemoriaPruebaActivarCodigo(t, reservado, ahora.Add(-5*time.Minute))
	cotejoMemoriaPruebaSembrarCodigo(store, activo)
	traza, evento := cotejoMemoriaPruebaConsultaPublica(activo, ahora)

	if err := store.RegistrarConsultaCotejo(ctx, traza, evento); err != nil {
		t.Fatalf("registrar consulta publica: %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)
	if store.events[0].Payload["auditoria_ref"] != store.audit[0].ID || store.audit[0].Signature == "" {
		t.Fatalf("consulta sin enlace de integridad: auditoria=%+v evento=%+v", store.audit[0], store.events[0])
	}
	contenido, err := json.Marshal(struct {
		Auditoria []domain.AuditEntry `json:"auditoria"`
		Eventos   []domain.Event      `json:"eventos"`
	}{Auditoria: store.audit, Eventos: store.events})
	if err != nil {
		t.Fatalf("serializar evidencia: %v", err)
	}
	for _, prohibido := range []string{
		cotejoMemoriaPruebaSecreto, cotejoMemoriaPruebaIndiceA, cotejoMemoriaPruebaProteccionA,
		"hmac-sha256", "proteccion_ref", "valor_csv",
	} {
		if strings.Contains(strings.ToLower(string(contenido)), strings.ToLower(prohibido)) {
			t.Fatalf("la consulta auditada contiene %q: %s", prohibido, contenido)
		}
	}

	trazaInsegura, eventoInseguro := cotejoMemoriaPruebaConsultaPublica(activo, ahora.Add(time.Minute))
	trazaInsegura.Metadata["valor_csv"] = cotejoMemoriaPruebaSecreto
	if err := store.RegistrarConsultaCotejo(ctx, trazaInsegura, eventoInseguro); !errors.Is(err, domain.ErrCodigoCotejoInvalido) {
		t.Fatalf("consulta con secreto en auditoria: error = %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)

	trazaInsegura, eventoInseguro = cotejoMemoriaPruebaConsultaPublica(activo, ahora.Add(2*time.Minute))
	eventoInseguro.Payload["indice"] = cotejoMemoriaPruebaIndiceA
	if err := store.RegistrarConsultaCotejo(ctx, trazaInsegura, eventoInseguro); !errors.Is(err, domain.ErrCodigoCotejoInvalido) {
		t.Fatalf("consulta con HMAC en evento: error = %v", err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)
}

func TestCotejoMemoriaTransicionesSonAtomicasYDetectanConflictoOptimista(t *testing.T) {
	t.Run("activacion", func(t *testing.T) {
		store := NewStore()
		ahora := time.Date(2026, time.July, 14, 14, 0, 0, 0, time.UTC)
		reservado := cotejoMemoriaPruebaCodigoReservado(
			"codigo-cotejo-memoria-transicion-001",
			domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-transicion-001", Version: 1},
			cotejoMemoriaPruebaIndiceA, cotejoMemoriaPruebaProteccionA, ahora.Add(-10*time.Minute),
		)
		cotejoMemoriaPruebaSembrarCodigo(store, reservado)
		huellaAnterior, _ := reservado.HuellaEstadoSHA256()
		activo := cotejoMemoriaPruebaActivarCodigo(t, reservado, ahora)
		cotejoMemoriaPruebaPrepararDocumentoParaActivacion(store, activo)
		traza, evento := cotejoMemoriaPruebaEvidenciaCodigo(activo, domain.AccionCodigoCotejoActivado, huellaAnterior, activo.ActivadoEn)

		if err := store.ConfirmarActivacionCodigoCotejo(context.Background(), huellaAnterior, activo, traza, evento); err != nil {
			t.Fatalf("confirmar activacion: %v", err)
		}
		cotejoMemoriaPruebaExigirTransicionConfirmada(t, store, activo)
		if err := store.ConfirmarActivacionCodigoCotejo(context.Background(), huellaAnterior, activo, traza, evento); !errors.Is(err, ports.ErrRevisionCodigoCotejoConflicto) {
			t.Fatalf("activacion con huella obsoleta: error = %v", err)
		}
		cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)
	})

	t.Run("retirada", func(t *testing.T) {
		store := NewStore()
		ahora := time.Date(2026, time.July, 14, 15, 0, 0, 0, time.UTC)
		reservado := cotejoMemoriaPruebaCodigoReservado(
			"codigo-cotejo-memoria-transicion-002",
			domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-transicion-002", Version: 1},
			cotejoMemoriaPruebaIndiceA, cotejoMemoriaPruebaProteccionA, ahora.Add(-20*time.Minute),
		)
		activo := cotejoMemoriaPruebaActivarCodigo(t, reservado, ahora.Add(-10*time.Minute))
		cotejoMemoriaPruebaSembrarCodigo(store, activo)
		huellaAnterior, _ := activo.HuellaEstadoSHA256()
		retirado, err := activo.Retirar(
			cotejoMemoriaPruebaPrincipal, "acuerdo-retirada-cotejo-memoria-001",
			"retirada autorizada del codigo", ahora,
		)
		if err != nil {
			t.Fatalf("crear retirada: %v", err)
		}
		traza, evento := cotejoMemoriaPruebaEvidenciaCodigo(retirado, domain.AccionCodigoCotejoRetirado, huellaAnterior, ahora)

		if err := store.ConfirmarRetiradaCodigoCotejo(context.Background(), huellaAnterior, retirado, traza, evento); err != nil {
			t.Fatalf("confirmar retirada: %v", err)
		}
		cotejoMemoriaPruebaExigirTransicionConfirmada(t, store, retirado)
		if err := store.ConfirmarRetiradaCodigoCotejo(context.Background(), huellaAnterior, retirado, traza, evento); !errors.Is(err, ports.ErrRevisionCodigoCotejoConflicto) {
			t.Fatalf("retirada con huella obsoleta: error = %v", err)
		}
		cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)
	})

	t.Run("sustitucion", func(t *testing.T) {
		store := NewStore()
		ahora := time.Date(2026, time.July, 14, 16, 0, 0, 0, time.UTC)
		reservado := cotejoMemoriaPruebaCodigoReservado(
			"codigo-cotejo-memoria-transicion-003",
			domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-transicion-003", Version: 1},
			cotejoMemoriaPruebaIndiceA, cotejoMemoriaPruebaProteccionA, ahora.Add(-20*time.Minute),
		)
		activo := cotejoMemoriaPruebaActivarCodigo(t, reservado, ahora.Add(-10*time.Minute))
		reservadoSustituto := cotejoMemoriaPruebaCodigoReservado(
			"codigo-cotejo-memoria-transicion-004",
			domain.ReferenciaDocumento{ID: "documento-cotejo-memoria-transicion-004", Version: 1},
			cotejoMemoriaPruebaIndiceB, cotejoMemoriaPruebaProteccionB, ahora.Add(-20*time.Minute),
		)
		sustituto := cotejoMemoriaPruebaActivarCodigo(t, reservadoSustituto, ahora.Add(-10*time.Minute))
		cotejoMemoriaPruebaSembrarCodigo(store, activo)
		cotejoMemoriaPruebaSembrarCodigo(store, sustituto)
		huellaAnterior, _ := activo.HuellaEstadoSHA256()
		sustituido, err := activo.Sustituir(
			cotejoMemoriaPruebaPrincipal, "acuerdo-sustitucion-cotejo-memoria-001",
			"sustitucion autorizada del codigo", sustituto.Referencia(), ahora,
		)
		if err != nil {
			t.Fatalf("crear sustitucion: %v", err)
		}
		traza, evento := cotejoMemoriaPruebaEvidenciaCodigo(sustituido, domain.AccionCodigoCotejoSustituido, huellaAnterior, ahora)

		if err := store.ConfirmarSustitucionCodigoCotejo(context.Background(), huellaAnterior, sustituido, traza, evento); err != nil {
			t.Fatalf("confirmar sustitucion: %v", err)
		}
		cotejoMemoriaPruebaExigirTransicionConfirmada(t, store, sustituido)
		if err := store.ConfirmarSustitucionCodigoCotejo(context.Background(), huellaAnterior, sustituido, traza, evento); !errors.Is(err, ports.ErrRevisionCodigoCotejoConflicto) {
			t.Fatalf("sustitucion con huella obsoleta: error = %v", err)
		}
		cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)
	})
}

func cotejoMemoriaPruebaPoliticaBorrador(version int, creadaEn time.Time) domain.PoliticaCotejo {
	versionAnterior := ""
	if version > 1 {
		versionAnterior = "politica-cotejo:" + cotejoMemoriaPruebaPoliticaID + ":v" + strconv.Itoa(version-1)
	}
	return domain.PoliticaCotejo{
		ID:                       cotejoMemoriaPruebaPoliticaID,
		Version:                  version,
		Revision:                 1,
		VersionAnteriorRef:       versionAnterior,
		Nombre:                   "Politica de cotejo en memoria",
		Descripcion:              "Politica para verificar documentos de recursos humanos",
		Modulos:                  []string{"bolsa", "personal"},
		TiposDocumentales:        []string{"certificado", "resolucion"},
		Clasificaciones:          []string{"restringido"},
		ClaseAcceso:              domain.ClaseAccesoCotejoProtegido,
		CamposPublicos:           []domain.CampoPublicoCotejo{domain.CampoPublicoCotejoOrgano},
		PermiteDescargaDocumento: true,
		RequiereFirma:            true,
		RequiereSelloTiempo:      true,
		RequiereRegistro:         true,
		GarantiaMinima:           domain.AuthAssuranceHigh,
		DiasPlazoActivacion:      5,
		DiasDisponibilidad:       365,
		Estado:                   domain.EstadoPoliticaCotejoBorrador,
		FuenteRef:                "normativa-cotejo-memoria-2026",
		MotivoCreacion:           "alta gobernada de la politica de cotejo",
		CreadaPor:                "responsable-rrhh-cotejo-memoria-001",
		CreadaEn:                 creadaEn,
	}
}

func cotejoMemoriaPruebaPoliticaPublicada() domain.PoliticaCotejo {
	creadaEn := time.Date(2026, time.January, 2, 9, 0, 0, 0, time.UTC)
	borrador := cotejoMemoriaPruebaPoliticaBorrador(1, creadaEn)
	publicada, _ := borrador.Publicar(
		"responsable-seguridad-cotejo-memoria-fixture",
		"aprobacion-cotejo-memoria-fixture",
		"publicacion estable para las pruebas del repositorio",
		creadaEn.Add(time.Hour),
	)
	return publicada
}

func cotejoMemoriaPruebaEvidenciaPolitica(
	politica domain.PoliticaCotejo,
	accion, huellaAnterior string,
) (domain.AuditEntry, domain.Event) {
	huella, _ := politica.HuellaSHA256()
	actor, instante, regla, motivo := politica.CreadaPor, politica.CreadaEn, politica.FuenteRef, politica.MotivoCreacion
	switch accion {
	case domain.AccionPoliticaCotejoBorradorActualizada:
		actor, instante, motivo = politica.ActualizadaPor, politica.ActualizadaEn, politica.MotivoActualizacion
	case domain.AccionPoliticaCotejoPublicada:
		actor, instante, regla, motivo = politica.PublicadaPor, politica.PublicadaEn, politica.AprobacionRef, politica.MotivoPublicacion
	case domain.AccionPoliticaCotejoRetirada:
		actor, instante, regla, motivo = politica.RetiradaPor, politica.RetiradaEn, politica.RetiradaAprobacionRef, politica.MotivoRetirada
	}
	traza := domain.AuditEntry{
		ActorID:          actor,
		ActorProfile:     "responsable_rrhh",
		AuthMethod:       domain.AuthMethodCertificate,
		AuthAssurance:    domain.AuthAssuranceHigh,
		AuthorizationRef: "decision-cotejo-memoria-001",
		Purpose:          "gobernar_cotejo_documental",
		Action:           accion,
		ModuleID:         "documentos",
		SubjectRef:       politica.Referencia(),
		ObjectVersion:    politica.Revision,
		RuleRef:          regla,
		Reason:           motivo,
		Result:           "correcto",
		BeforeHash:       huellaAnterior,
		AfterHash:        huella,
		CorrelationRef:   cotejoMemoriaPruebaCorrelacionRef,
		Metadata:         map[string]string{"estado": string(politica.Estado)},
		OccurredAt:       instante,
	}
	evento := domain.Event{
		Type:       accion,
		ModuleID:   "documentos",
		SubjectRef: politica.Referencia(),
		ActorID:    actor,
		OccurredAt: instante,
		Payload: map[string]string{
			"politica_id":      politica.ID,
			"politica_version": strconv.Itoa(politica.Version),
			"revision":         strconv.Itoa(politica.Revision),
			"estado":           string(politica.Estado),
			"huella_sha256":    huella,
		},
	}
	return traza, evento
}

func cotejoMemoriaPruebaSolicitudReserva(
	clave string,
	documento domain.ReferenciaDocumento,
	huella string,
	solicitadaEn time.Time,
) ports.SolicitudReservarEmisionCodigoCotejo {
	politica := cotejoMemoriaPruebaPoliticaPublicada()
	aplicacion, _ := politica.Aplicacion()
	return ports.SolicitudReservarEmisionCodigoCotejo{
		ClaveIdempotencia:   clave,
		PrincipalID:         cotejoMemoriaPruebaPrincipal,
		HuellaSolicitudHMAC: huella,
		Documento:           documento,
		Politica:            aplicacion.Referencia,
		SolicitadaEn:        solicitadaEn,
		ExpiraEn:            solicitadaEn.Add(5 * time.Minute),
	}
}

func cotejoMemoriaPruebaCodigoReservado(
	id string,
	documento domain.ReferenciaDocumento,
	indice, proteccion string,
	reservadoEn time.Time,
) domain.CodigoCotejo {
	politica := cotejoMemoriaPruebaPoliticaPublicada()
	aplicacion, _ := politica.Aplicacion()
	return domain.CodigoCotejo{
		ID:               id,
		Revision:         1,
		Documento:        documento,
		ModuloID:         "bolsa",
		TipoDocumental:   "certificado",
		Clasificacion:    "restringido",
		Organo:           "L01000000",
		ExpedienteRef:    cotejoMemoriaPruebaExpediente,
		IndiceCodigoHMAC: indice,
		ProteccionRef:    proteccion,
		VersionGenerador: "generador-cotejo-memoria-v1",
		EntropiaBits:     160,
		Politica:         aplicacion,
		Estado:           domain.EstadoCodigoCotejoReservado,
		ReservadoPor:     cotejoMemoriaPruebaPrincipal,
		ReservadoEn:      reservadoEn,
		ReservaExpiraEn:  reservadoEn.Add(30 * time.Minute),
		MotivoReserva:    "reserva de codigo de cotejo en memoria",
		CorrelacionRef:   cotejoMemoriaPruebaCorrelacionRef,
	}
}

func cotejoMemoriaPruebaEvidenciaEmision(codigo domain.CodigoCotejo, emitidaEn time.Time) domain.EvidenciaEmisionDocumento {
	return domain.EvidenciaEmisionDocumento{
		Documento: codigo.Documento,
		VersionEmitida: domain.VersionEmitidaCotejo{
			RepresentacionID:      "representacion-" + codigo.ID,
			ReferenciaContenido:   "almacen:cotejo-memoria:" + codigo.ID,
			HuellaContenidoSHA256: strings.Repeat("2", 64),
			MIME:                  domain.FormatoDocumentoPDF.MIME(),
			Tamano:                8_192,
			FirmaRefs:             []string{"firma-" + codigo.ID},
			SelloTiempoRefs:       []string{"sello-tiempo-" + codigo.ID},
			ValidacionFirmaRef:    "validacion-firma-" + codigo.ID,
			RegistroRef:           "registro-" + codigo.ID,
			EmitidaEn:             emitidaEn,
		},
		Apta:         true,
		EvidenciaRef: "evidencia-emision-" + codigo.ID,
	}
}

func cotejoMemoriaPruebaActivarCodigo(t *testing.T, reservado domain.CodigoCotejo, activadoEn time.Time) domain.CodigoCotejo {
	t.Helper()
	activo, err := reservado.Activar(
		cotejoMemoriaPruebaPrincipal, "activacion-"+reservado.ID,
		"activacion del codigo tras validar su emision",
		cotejoMemoriaPruebaEvidenciaEmision(reservado, activadoEn.Add(-time.Minute)), activadoEn,
	)
	if err != nil {
		t.Fatalf("activar codigo de prueba: %v", err)
	}
	return activo
}

func cotejoMemoriaPruebaEvidenciaCodigo(
	codigo domain.CodigoCotejo,
	accion, huellaAnterior string,
	instante time.Time,
) (domain.AuditEntry, domain.Event) {
	huella, _ := codigo.HuellaEstadoSHA256()
	actor, motivo := codigo.ReservadoPor, codigo.MotivoReserva
	regla := codigo.Politica.Referencia.ID + ":" + strconv.Itoa(codigo.Politica.Referencia.Version)
	switch accion {
	case domain.AccionCodigoCotejoActivado:
		actor, motivo, regla = codigo.ActivadoPor, codigo.MotivoActivacion, codigo.EvidenciaEmisionRef
	case domain.AccionCodigoCotejoRetirado, domain.AccionCodigoCotejoSustituido:
		actor, motivo, regla = codigo.RetiradoPor, codigo.MotivoRetirada, codigo.RetiradaRef
	}
	traza := domain.AuditEntry{
		ActorID:          actor,
		ActorProfile:     "tecnico_rrhh",
		AuthMethod:       domain.AuthMethodCertificate,
		AuthAssurance:    domain.AuthAssuranceHigh,
		AuthorizationRef: "decision-cotejo-memoria-001",
		Purpose:          "gestionar_cotejo_documental",
		Action:           accion,
		ModuleID:         codigo.ModuloID,
		SubjectRef:       codigo.Referencia(),
		ObjectVersion:    codigo.Revision,
		ExpedienteRef:    codigo.ExpedienteRef,
		DocumentRef:      claveDocumentoLogico(codigo.Documento),
		RuleRef:          regla,
		Reason:           motivo,
		Result:           "correcto",
		BeforeHash:       huellaAnterior,
		AfterHash:        huella,
		CorrelationRef:   codigo.CorrelacionRef,
		Metadata:         map[string]string{"estado": string(codigo.Estado)},
		OccurredAt:       instante,
	}
	evento := domain.Event{
		Type:       accion,
		ModuleID:   codigo.ModuloID,
		SubjectRef: codigo.Referencia(),
		ActorID:    actor,
		OccurredAt: instante,
		Payload: map[string]string{
			"codigo_ref":    codigo.Referencia(),
			"documento_ref": claveDocumentoLogico(codigo.Documento),
			"revision":      strconv.Itoa(codigo.Revision),
			"estado":        string(codigo.Estado),
			"huella_estado": huella,
		},
	}
	return traza, evento
}

func cotejoMemoriaPruebaConsultaPublica(codigo domain.CodigoCotejo, instante time.Time) (domain.AuditEntry, domain.Event) {
	traza := domain.AuditEntry{
		ActorID:        "publico-anonimo",
		ActorProfile:   "publico",
		Purpose:        "verificacion_documental_publica",
		Action:         domain.AccionConsultaPublicaCotejo,
		ModuleID:       codigo.ModuloID,
		SubjectRef:     codigo.Referencia(),
		ObjectVersion:  codigo.Revision,
		Result:         "disponible",
		CorrelationRef: "correlacion-consulta-cotejo-memoria-001",
		Metadata:       map[string]string{"origen_tecnico": "http-publico-cotejo-memoria"},
		OccurredAt:     instante,
	}
	evento := domain.Event{
		Type:       traza.Action,
		ModuleID:   traza.ModuleID,
		SubjectRef: traza.SubjectRef,
		ActorID:    traza.ActorID,
		OccurredAt: instante,
		Payload:    map[string]string{"resultado_consulta": traza.Result},
	}
	return traza, evento
}

func cotejoMemoriaPruebaSembrarPolitica(store *Store) {
	politica := cotejoMemoriaPruebaPoliticaPublicada()
	store.politicasCotejo[clavePoliticaCotejo(politica.ID, politica.Version)] = politica
}

func cotejoMemoriaPruebaDocumentoLogico(
	codigo domain.CodigoCotejo,
	estado domain.EstadoDocumentoLogico,
) domain.DocumentoLogico {
	return domain.DocumentoLogico{
		ID:       codigo.Documento.ID,
		Version:  codigo.Documento.Version,
		Revision: 1,
		Plantilla: domain.ReferenciaPlantillaDocumento{
			ID: "certificado_cotejo_memoria", Version: 1, HuellaSHA256: strings.Repeat("3", 64),
		},
		ModuloID:       codigo.ModuloID,
		TipoDocumental: codigo.TipoDocumental,
		Clasificacion:  codigo.Clasificacion,
		Relaciones: []domain.RelacionDocumento{{
			Tipo: domain.TipoRelacionExpediente, Referencia: codigo.ExpedienteRef, Rol: "principal",
		}},
		Estado:           estado,
		HuellaDatosHMAC:  "hmac-sha256:datos_cotejo_memoria:" + strings.Repeat("4", 64),
		HuellaFuenteHMAC: "hmac-sha256:fuente_cotejo_memoria:" + strings.Repeat("5", 64),
		CreadoPor:        cotejoMemoriaPruebaPrincipal,
		CreadoEn:         codigo.ReservadoEn.Add(-time.Hour),
		CorrelacionRef:   "correlacion-documento-cotejo-memoria-001",
		Motivo:           "generacion del documento logico para cotejo",
		ENI: domain.MetadatosENI{
			Identificador:     "ES_L01000000_2026_" + codigo.Documento.ID,
			Organo:            codigo.Organo,
			Origen:            "administracion",
			EstadoElaboracion: "original",
			TipoDocumental:    codigo.TipoDocumental,
			FechaCaptura:      codigo.ReservadoEn.Add(-time.Hour),
		},
	}
}

func cotejoMemoriaPruebaRepresentacion(
	codigo domain.CodigoCotejo,
	documento domain.DocumentoLogico,
) domain.RepresentacionDocumento {
	version := codigo.VersionEmitida
	return domain.RepresentacionDocumento{
		ID:                    version.RepresentacionID,
		Documento:             codigo.Documento,
		Tipo:                  domain.TipoRepresentacionFirma,
		Formato:               domain.FormatoDocumentoPDF,
		MIME:                  version.MIME,
		NombreFichero:         version.RepresentacionID + ".pdf",
		Tamano:                version.Tamano,
		HuellaContenidoSHA256: version.HuellaContenidoSHA256,
		HuellaFuenteHMAC:      documento.HuellaFuenteHMAC,
		ReferenciaContenido:   version.ReferenciaContenido,
		EstadoTecnico:         domain.EstadoRepresentacionDisponible,
		EstadoAntivirus:       domain.EstadoAntivirusLimpio,
		GeneradaPor:           "servicio-firma-cotejo-memoria",
		GeneradaEn:            version.EmitidaEn.Add(-time.Minute),
		DerivadaDeRef:         "representacion-origen-" + codigo.ID,
	}
}

func cotejoMemoriaPruebaSembrarDependenciasReserva(store *Store, codigo domain.CodigoCotejo) {
	cotejoMemoriaPruebaSembrarPolitica(store)
	clave := claveDocumentoLogico(codigo.Documento)
	if _, existe := store.documentosLogicos[clave]; !existe {
		store.documentosLogicos[clave] = cotejoMemoriaPruebaDocumentoLogico(codigo, domain.EstadoDocumentoLogicoBorrador)
	}
}

func cotejoMemoriaPruebaPrepararDocumentoParaActivacion(store *Store, codigo domain.CodigoCotejo) {
	documento := cotejoMemoriaPruebaDocumentoLogico(codigo, domain.EstadoDocumentoLogicoRegistrado)
	store.documentosLogicos[claveDocumentoLogico(codigo.Documento)] = documento
	representacion := cotejoMemoriaPruebaRepresentacion(codigo, documento)
	store.representaciones[representacion.ID] = representacion
}

func cotejoMemoriaPruebaReservarYConfirmar(
	t *testing.T,
	store *Store,
	clave, huellaSolicitud string,
	codigo domain.CodigoCotejo,
) error {
	t.Helper()
	cotejoMemoriaPruebaSembrarDependenciasReserva(store, codigo)
	solicitud := cotejoMemoriaPruebaSolicitudReserva(
		clave, codigo.Documento, huellaSolicitud, codigo.ReservadoEn,
	)
	reserva, err := store.ReservarEmisionCodigoCotejo(context.Background(), solicitud)
	if err != nil {
		return err
	}
	traza, evento := cotejoMemoriaPruebaEvidenciaCodigo(
		codigo, domain.AccionCodigoCotejoReservado, "", codigo.ReservadoEn.Add(time.Minute),
	)
	if err := codigo.Validar(); err != nil || !evidenciaCodigoCotejoValida(
		codigo, traza, evento, domain.AccionCodigoCotejoReservado, "", codigo.ReservadoEn.Add(time.Minute),
	) {
		t.Fatalf("fixture de reserva invalido: codigo=%v traza=%+v evento=%+v", err, traza, evento)
	}
	return store.ConfirmarReservaCodigoCotejo(
		context.Background(), reserva.Token, huellaSolicitud, codigo.ReservadoEn.Add(time.Minute), codigo, traza, evento,
	)
}

func cotejoMemoriaPruebaSembrarCodigo(store *Store, codigo domain.CodigoCotejo) {
	cotejoMemoriaPruebaSembrarPolitica(store)
	estadoDocumento := domain.EstadoDocumentoLogicoBorrador
	if codigo.Estado != domain.EstadoCodigoCotejoReservado {
		estadoDocumento = domain.EstadoDocumentoLogicoRegistrado
	}
	documento := cotejoMemoriaPruebaDocumentoLogico(codigo, estadoDocumento)
	store.documentosLogicos[claveDocumentoLogico(codigo.Documento)] = documento
	if codigo.VersionEmitida != nil {
		representacion := cotejoMemoriaPruebaRepresentacion(codigo, documento)
		store.representaciones[representacion.ID] = representacion
	}
	store.codigosCotejo[codigo.ID] = codigo
	store.cotejoPorDocumento[claveDocumentoLogico(codigo.Documento)] = codigo.ID
	store.cotejoPorIndice[codigo.IndiceCodigoHMAC] = codigo.ID
}

func cotejoMemoriaPruebaExigirEvidenciaAtomica(t *testing.T, store *Store, esperadas int) {
	t.Helper()
	if len(store.audit) != esperadas || len(store.events) != esperadas {
		t.Fatalf("evidencia no atomica: auditoria=%d eventos=%d esperados=%d", len(store.audit), len(store.events), esperadas)
	}
	for indice := range store.events {
		if store.events[indice].Payload["auditoria_ref"] != store.audit[indice].ID ||
			store.audit[indice].Signature == "" || store.audit[indice].IntegrityAlgorithm != "sha256-chain-v1" {
			t.Fatalf("evidencia %d no enlazada: auditoria=%+v evento=%+v", indice, store.audit[indice], store.events[indice])
		}
	}
}

func cotejoMemoriaPruebaExigirSinParcialidadCodigo(
	t *testing.T,
	store *Store,
	rechazado domain.CodigoCotejo,
	evidenciasEsperadas int,
) {
	t.Helper()
	if _, existe := store.codigosCotejo[rechazado.ID]; existe && rechazado.ID != "codigo-cotejo-memoria-unicidad-001" {
		t.Fatalf("el codigo rechazado fue persistido: %s", rechazado.ID)
	}
	if id := store.cotejoPorIndice[rechazado.IndiceCodigoHMAC]; id == rechazado.ID && rechazado.ID != "codigo-cotejo-memoria-unicidad-001" {
		t.Fatalf("el indice rechazado fue persistido para %s", rechazado.ID)
	}
	if id := store.cotejoPorDocumento[claveDocumentoLogico(rechazado.Documento)]; id == rechazado.ID && rechazado.ID != "codigo-cotejo-memoria-unicidad-001" {
		t.Fatalf("el documento rechazado fue persistido para %s", rechazado.ID)
	}
	if len(store.codigosCotejo) != 1 || len(store.cotejoPorDocumento) != 1 || len(store.cotejoPorIndice) != 1 {
		t.Fatalf("la confirmacion rechazada dejo indices parciales: codigos=%d documentos=%d indices=%d",
			len(store.codigosCotejo), len(store.cotejoPorDocumento), len(store.cotejoPorIndice))
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, evidenciasEsperadas)
}

func cotejoMemoriaPruebaExigirTransicionConfirmada(t *testing.T, store *Store, esperada domain.CodigoCotejo) {
	t.Helper()
	guardada, err := store.ObtenerCodigoCotejo(context.Background(), esperada.ID)
	if err != nil || !reflect.DeepEqual(guardada, esperada) {
		t.Fatalf("transicion guardada = %+v; esperada %+v; error = %v", guardada, esperada, err)
	}
	cotejoMemoriaPruebaExigirEvidenciaAtomica(t, store, 1)
}
