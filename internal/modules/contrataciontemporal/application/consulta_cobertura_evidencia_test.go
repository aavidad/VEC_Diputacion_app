package application

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	finalidadCoberturaClave = "decidir_via_cobertura"
	finalidadCoberturaRef   = "finalidad_decidir_via_cobertura_01"
)

func TestConsultarConEvidenciaNoPreconsume(t *testing.T) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	evidencia := prepararEvidenciaCoberturaPrueba(
		t, entorno, entorno.solicitud, "recibo_consulta_bolsa_012345",
		domain.ComprobacionAfirmativa,
	)
	comprobacion, err := evidencia.Comprobacion()
	if err != nil || comprobacion.Resultado !=
		domain.ComprobacionAfirmativa {
		t.Fatalf("comprobacion inesperada: %#v, %v", comprobacion, err)
	}
	if _, err := evidencia.OrdenPendienteEn(entorno.reloj.Ahora()); err != nil {
		t.Fatalf("la orden preparada debe seguir vigente: %v", err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.ordenes) != 0 ||
		len(entorno.consumidor.registros) != 0 {
		t.Fatal("preparar evidencia no puede producir consumo durable")
	}
}

func TestPreparadorConsultaCoberturaNoTieneConsumidor(t *testing.T) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	preparador, err := NuevoPreparadorConsultaCobertura(
		entorno.fuente,
		entorno.verificador,
		entorno.publicador,
		entorno.autenticador,
		entorno.reloj,
		time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := preparador.ConsultarConEvidencia(
		context.Background(),
		entorno.solicitud,
	); err != nil {
		t.Fatal(err)
	}
	entorno.consumidor.mu.Lock()
	defer entorno.consumidor.mu.Unlock()
	if len(entorno.consumidor.ordenes) != 0 {
		t.Fatal("el preparador aislado no puede alcanzar un consumidor")
	}
}

func TestEvidenciaRechazaRollbackPosteriorAVerificacion(t *testing.T) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	evidencia := prepararEvidenciaCoberturaPrueba(
		t, entorno, entorno.solicitud, "recibo_consulta_bolsa_012345",
		domain.ComprobacionAfirmativa,
	)
	orden, err := evidencia.OrdenPendienteEn(entorno.reloj.Ahora())
	if err != nil {
		t.Fatal(err)
	}
	suelo := entorno.inicio.Add(3 * time.Second)
	conSuelo, err := cobertura.NuevaEvidenciaConsultaCobertura(orden, suelo)
	if err != nil {
		t.Fatal(err)
	}
	rollback := entorno.inicio.Add(2500 * time.Millisecond)
	if _, err := conSuelo.OrdenPendienteEn(rollback); !errors.Is(
		err,
		ports.ErrResultadoFuenteCoberturaNoConfiable,
	) {
		t.Fatalf("rollback aceptado: %v", err)
	}
}

func TestConjuntoNormalizaDuplicadoYRenovacionSinDependerDelOrden(
	t *testing.T,
) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	antigua := prepararEvidenciaCoberturaPrueba(
		t, entorno, entorno.solicitud, "recibo_consulta_bolsa_012345",
		domain.ComprobacionAfirmativa,
	)
	renovadaSolicitud := entorno.solicitud
	renovadaSolicitud.PeticionRef = "peticion_cobertura_renovada_012345"
	renovadaSolicitud.SolicitadaEn = renovadaSolicitud.SolicitadaEn.Add(
		100 * time.Millisecond,
	)
	renovada := prepararEvidenciaCoberturaPrueba(
		t, entorno, renovadaSolicitud, "recibo_consulta_bolsa_renovado_012345",
		domain.ComprobacionAfirmativa,
	)
	politica := politicaCoberturaPrueba(t, entorno.catalogo, entorno.inicio)
	coordenadas := coordenadasCoberturaPrueba(
		entorno.solicitud, politica,
	)
	primero := nuevoConjuntoCoberturaPrueba(
		t, coordenadas, entorno.catalogo, politica,
		[]cobertura.EvidenciaConsultaCobertura{antigua, renovada},
		entorno.reloj.Ahora(),
	)
	segundo := nuevoConjuntoCoberturaPrueba(
		t, coordenadas, entorno.catalogo, politica,
		[]cobertura.EvidenciaConsultaCobertura{renovada, antigua},
		entorno.reloj.Ahora(),
	)
	duplicada := nuevoConjuntoCoberturaPrueba(
		t, coordenadas, entorno.catalogo, politica,
		[]cobertura.EvidenciaConsultaCobertura{renovada, renovada},
		entorno.reloj.Ahora(),
	)
	huellaPrimero, _ := primero.HuellaSHA256()
	huellaSegundo, _ := segundo.HuellaSHA256()
	huellaDuplicada, _ := duplicada.HuellaSHA256()
	if huellaPrimero != huellaSegundo ||
		huellaPrimero != huellaDuplicada {
		t.Fatal("la normalizacion depende del orden o conserva duplicados")
	}
	ordenes, err := primero.OrdenesPendientesEn(entorno.reloj.Ahora())
	if err != nil || len(ordenes) != 1 {
		t.Fatalf("debe quedar una orden: %d, %v", len(ordenes), err)
	}
	resumen, err := ordenes[0].ResumenPendienteEn(entorno.reloj.Ahora())
	if err != nil || resumen.PeticionRef != renovadaSolicitud.PeticionRef {
		t.Fatalf("no se eligio la prueba mas reciente: %#v, %v", resumen, err)
	}
	comprobaciones, err := primero.ComprobacionesEn(entorno.reloj.Ahora())
	if err != nil || len(comprobaciones) != 1 {
		t.Fatalf("lectura revalidada inesperada: %d, %v", len(comprobaciones), err)
	}
	if _, err := primero.ComprobacionesEn(
		entorno.inicio.Add(6 * time.Second),
	); !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("lectura caducada aceptada: %v", err)
	}
}

func TestConjuntoRechazaConflictoSemanticoMismaClave(t *testing.T) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	afirmativa := prepararEvidenciaCoberturaPrueba(
		t, entorno, entorno.solicitud, "recibo_consulta_bolsa_012345",
		domain.ComprobacionAfirmativa,
	)
	conflictiva := prepararEvidenciaCoberturaPrueba(
		t, entorno, entorno.solicitud, "recibo_consulta_bolsa_012345",
		domain.ComprobacionNegativa,
	)
	politica := politicaCoberturaPrueba(t, entorno.catalogo, entorno.inicio)
	_, err := cobertura.NuevoConjuntoEvidenciasCobertura(
		coordenadasCoberturaPrueba(entorno.solicitud, politica),
		entorno.catalogo,
		politica,
		[]cobertura.EvidenciaConsultaCobertura{afirmativa, conflictiva},
		entorno.reloj.Ahora(),
	)
	if !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("conflicto semantico aceptado: %v", err)
	}
}

func TestConjuntoExigeCompletitudYDistingueNoConsta(t *testing.T) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	catalogo, comprobaciones := catalogoDosComprobacionesPrueba(
		t, entorno.inicio,
	)
	entorno.catalogo = catalogo
	entorno.publicador.publicar = func(
		context.Context,
		ports.SolicitudConsultarCobertura,
	) (ports.ConfirmacionPublicacionCobertura, error) {
		return ports.NuevaConfirmacionPublicacionCobertura(
			entorno.publicador.identidad.AutoridadRef(),
			catalogo.Publicacion(),
			entorno.reloj.Ahora(),
		)
	}
	solicitud := entorno.solicitud
	solicitud.Catalogo = catalogo.Identidad()
	solicitud.Comprobacion = comprobaciones[0]
	evidencia := prepararEvidenciaCoberturaPrueba(
		t, entorno, solicitud, "recibo_primera_comprobacion_012345",
		domain.ComprobacionAfirmativa,
	)
	politica := politicaCoberturaPrueba(t, catalogo, entorno.inicio)
	coordenadas := coordenadasCoberturaPrueba(solicitud, politica)
	if _, err := cobertura.NuevoConjuntoEvidenciasCobertura(
		coordenadas,
		catalogo,
		politica,
		[]cobertura.EvidenciaConsultaCobertura{evidencia},
		entorno.reloj.Ahora(),
	); !errors.Is(err, ports.ErrResultadoFuenteCoberturaNoConfiable) {
		t.Fatalf("omision de comprobacion aceptada: %v", err)
	}
	segundaSolicitud := solicitud
	segundaSolicitud.PeticionRef = "peticion_segunda_comprobacion_012345"
	segundaSolicitud.Comprobacion = comprobaciones[1]
	segunda := prepararEvidenciaCoberturaPrueba(
		t, entorno, segundaSolicitud, "recibo_segunda_comprobacion_012345",
		domain.ComprobacionAfirmativa,
	)
	ordenDirecto := nuevoConjuntoCoberturaPrueba(
		t, coordenadas, catalogo, politica,
		[]cobertura.EvidenciaConsultaCobertura{evidencia, segunda},
		entorno.reloj.Ahora(),
	)
	ordenInverso := nuevoConjuntoCoberturaPrueba(
		t, coordenadas, catalogo, politica,
		[]cobertura.EvidenciaConsultaCobertura{segunda, evidencia},
		entorno.reloj.Ahora(),
	)
	huellaDirecta, _ := ordenDirecto.HuellaSHA256()
	huellaInversa, _ := ordenInverso.HuellaSHA256()
	if huellaDirecta != huellaInversa {
		t.Fatal("la huella cambia con el orden de comprobaciones")
	}

	entornoUno := nuevoEntornoCoberturaAplicacionPrueba(t)
	noConsta := prepararEvidenciaCoberturaPrueba(
		t, entornoUno, entornoUno.solicitud,
		"recibo_no_consta_bolsa_012345",
		domain.ComprobacionNoConsta,
	)
	politicaUno := politicaCoberturaPrueba(
		t, entornoUno.catalogo, entornoUno.inicio,
	)
	_ = nuevoConjuntoCoberturaPrueba(
		t,
		coordenadasCoberturaPrueba(entornoUno.solicitud, politicaUno),
		entornoUno.catalogo,
		politicaUno,
		[]cobertura.EvidenciaConsultaCobertura{noConsta},
		entornoUno.reloj.Ahora(),
	)
}

func TestArtefactosEvidenciaCoberturaRedactanTodosLosFormatos(t *testing.T) {
	entorno := nuevoEntornoCoberturaAplicacionPrueba(t)
	evidencia := prepararEvidenciaCoberturaPrueba(
		t, entorno, entorno.solicitud, "recibo_consulta_bolsa_012345",
		domain.ComprobacionAfirmativa,
	)
	resumen, err := evidencia.Resumen()
	if err != nil {
		t.Fatal(err)
	}
	politica := politicaCoberturaPrueba(t, entorno.catalogo, entorno.inicio)
	coordenadas := coordenadasCoberturaPrueba(entorno.solicitud, politica)
	conjunto := nuevoConjuntoCoberturaPrueba(
		t, coordenadas, entorno.catalogo, politica,
		[]cobertura.EvidenciaConsultaCobertura{evidencia},
		entorno.reloj.Ahora(),
	)
	secretos := []string{
		entorno.solicitud.ExpedienteRef,
		entorno.solicitud.CategoriaRef,
		entorno.solicitud.PeticionRef,
		entorno.solicitud.Catalogo.Referencia,
		politica.Identidad().Referencia,
		finalidadCoberturaRef,
	}
	comprobarRedaccionCobertura(t, evidencia, secretos)
	comprobarRedaccionCobertura(t, resumen, secretos)
	comprobarRedaccionCobertura(t, coordenadas, secretos)
	comprobarRedaccionCobertura(t, conjunto, secretos)
}

func prepararEvidenciaCoberturaPrueba(
	t *testing.T,
	entorno *entornoCoberturaAplicacionPrueba,
	solicitud ports.SolicitudConsultarCobertura,
	reciboRef string,
	resultado domain.ResultadoComprobacion,
) cobertura.EvidenciaConsultaCobertura {
	t.Helper()
	entorno.fuente.consultar = func(
		_ context.Context,
		s ports.SolicitudConsultarCobertura,
	) (ports.ResultadoConsultaCobertura, error) {
		return resultadoCoberturaAplicacionPrueba(
			t,
			s,
			func(datos *ports.DatosResultadoConsultaCobertura) {
				datos.Comprobacion.ReciboRef = reciboRef
				datos.Comprobacion.Resultado = resultado
			},
		), nil
	}
	evidencia, err := entorno.servicio.ConsultarConEvidencia(
		context.Background(),
		solicitud,
	)
	if err != nil {
		t.Fatal(err)
	}
	return evidencia
}

func politicaCoberturaPrueba(
	t *testing.T,
	catalogo domain.CatalogoViasCobertura,
	inicio time.Time,
) domain.PoliticaDecisionCobertura {
	t.Helper()
	viasCatalogo := catalogo.Vias()
	vias := make([]domain.ReglaViaDecisionCobertura, len(viasCatalogo))
	for indice, via := range viasCatalogo {
		reglas := make(
			[]domain.ReglaComprobacionDecisionCobertura,
			len(via.Comprobaciones),
		)
		for j, comprobacion := range via.Comprobaciones {
			reglas[j] = domain.ReglaComprobacionDecisionCobertura{
				Clave: comprobacion.Clave,
				ResultadosHabilitantes: []domain.ResultadoComprobacion{
					domain.ComprobacionAfirmativa,
				},
				TratamientoAusencia: domain.AusenciaCoberturaAdmitida,
			}
		}
		vias[indice] = domain.ReglaViaDecisionCobertura{
			ViaClave: via.Clave, Prioridad: uint16(indice + 1),
			Comprobaciones: reglas,
		}
	}
	politica, err := domain.PublicarPoliticaDecisionCobertura(
		domain.BorradorPoliticaDecisionCobertura{
			Referencia:      "politica_decision_cobertura_01",
			Version:         1,
			Catalogo:        catalogo.Identidad(),
			OrganizacionRef: organizacionCoberturaPrueba,
			FinalidadClave:  finalidadCoberturaClave,
			FinalidadRef:    finalidadCoberturaRef,
			PublicadaEn:     inicio.Add(-45 * time.Minute),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: inicio.Add(-30 * time.Minute),
				Hasta: inicio.Add(30 * time.Minute),
			},
			ProcedenciaRef: "procedimiento_politica_cobertura_01",
			Vias:           vias,
		},
		catalogo,
	)
	if err != nil {
		t.Fatal(err)
	}
	return politica
}

func coordenadasCoberturaPrueba(
	solicitud ports.SolicitudConsultarCobertura,
	politica domain.PoliticaDecisionCobertura,
) cobertura.CoordenadasConjuntoEvidencias {
	return cobertura.CoordenadasConjuntoEvidencias{
		OrganizacionRef:   solicitud.OrganizacionRef,
		ExpedienteRef:     solicitud.ExpedienteRef,
		VersionExpediente: solicitud.VersionExpediente,
		Catalogo:          solicitud.Catalogo,
		Politica:          politica.Identidad(),
		FinalidadClave:    finalidadCoberturaClave,
		FinalidadRef:      finalidadCoberturaRef,
		ViaClave:          solicitud.ViaClave,
		CategoriaRef:      solicitud.CategoriaRef,
		Periodo:           solicitud.Periodo,
	}
}

func nuevoConjuntoCoberturaPrueba(
	t *testing.T,
	coordenadas cobertura.CoordenadasConjuntoEvidencias,
	catalogo domain.CatalogoViasCobertura,
	politica domain.PoliticaDecisionCobertura,
	evidencias []cobertura.EvidenciaConsultaCobertura,
	instante time.Time,
) cobertura.ConjuntoEvidenciasCobertura {
	t.Helper()
	conjunto, err := cobertura.NuevoConjuntoEvidenciasCobertura(
		coordenadas, catalogo, politica, evidencias, instante,
	)
	if err != nil {
		t.Fatal(err)
	}
	return conjunto
}

func catalogoDosComprobacionesPrueba(
	t *testing.T,
	inicio time.Time,
) (
	domain.CatalogoViasCobertura,
	[]domain.ComprobacionExigibleCobertura,
) {
	t.Helper()
	comprobaciones := []domain.ComprobacionExigibleCobertura{
		{
			Clave: "existe_bolsa_vigente", Orden: 1, Obligatoria: true,
			Procedencia: domain.ProcedenciaComprobacionCobertura{
				Clave: "bolsa", DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
			},
		},
		{
			Clave: "cumple_requisito_adicional", Orden: 2, Obligatoria: false,
			Procedencia: domain.ProcedenciaComprobacionCobertura{
				Clave: "bolsa", DefinicionFuenteRef: "fuente_definicion_bolsa_v3",
			},
		},
	}
	catalogo, err := domain.PublicarCatalogoViasCobertura(
		domain.BorradorCatalogoViasCobertura{
			Referencia: "catalogo_cobertura_dos_comprobaciones",
			Version:    8, PublicadoEn: inicio.Add(-time.Hour),
			Vigencia: domain.VigenciaCatalogoCobertura{
				Desde: inicio.Add(-time.Hour), Hasta: inicio.Add(time.Hour),
			},
			ProcedenciaRef: "procedimiento_gobierno_catalogo_02",
			Vias: []domain.DefinicionViaCobertura{{
				Clave: "bolsa_vigente", Orden: 1,
				Comprobaciones: comprobaciones,
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return catalogo, comprobaciones
}

type artefactoCoberturaRedactado interface {
	fmt.Stringer
	fmt.GoStringer
	slog.LogValuer
	encoding.TextMarshaler
}

func comprobarRedaccionCobertura(
	t *testing.T,
	artefacto artefactoCoberturaRedactado,
	secretos []string,
) {
	t.Helper()
	texto, err := artefacto.MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	jsonArtefacto, err := json.Marshal(artefacto)
	if err != nil {
		t.Fatal(err)
	}
	representaciones := []string{
		artefacto.String(),
		artefacto.GoString(),
		fmt.Sprintf("%v", artefacto),
		fmt.Sprintf("%+v", artefacto),
		fmt.Sprintf("%#v", artefacto),
		artefacto.LogValue().String(),
		string(texto),
		string(jsonArtefacto),
	}
	for _, representacion := range representaciones {
		for _, secreto := range secretos {
			if strings.Contains(representacion, secreto) {
				t.Fatalf("formato filtra %q: %s", secreto, representacion)
			}
		}
	}
}
