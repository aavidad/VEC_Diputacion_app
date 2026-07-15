package domain

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	huellaIndiceCotejoPrueba  = "hmac-sha256:indice-cotejo-v1:1111111111111111111111111111111111111111111111111111111111111111"
	huellaEmisionCotejoPrueba = "2222222222222222222222222222222222222222222222222222222222222222"
)

func politicaCotejoBorradorPrueba() PoliticaCotejo {
	fecha := time.Date(2026, time.July, 14, 8, 0, 0, 0, time.UTC)
	return PoliticaCotejo{
		ID:                       "cotejo_documentos_emitidos",
		Version:                  3,
		Revision:                 1,
		VersionAnteriorRef:       "politica-cotejo:cotejo_documentos_emitidos:v2",
		Nombre:                   "Cotejo de documentos emitidos",
		Descripcion:              "Politica aplicable a documentos emitidos por recursos humanos",
		Modulos:                  []string{"personal", "bolsa"},
		TiposDocumentales:        []string{"resolucion", "contrato"},
		Clasificaciones:          []string{"publica", "datos_personales_alta"},
		ClaseAcceso:              ClaseAccesoCotejoPublico,
		CamposPublicos:           []CampoPublicoCotejo{CampoPublicoCotejoOrgano, CampoPublicoCotejoHuellaSHA256, CampoPublicoCotejoFechaEmision},
		PermiteDescargaDocumento: true,
		RequiereTitularidad:      false,
		GarantiaMinima:           AuthAssuranceLow,
		DiasPlazoActivacion:      7,
		DiasDisponibilidad:       30,
		Estado:                   EstadoPoliticaCotejoBorrador,
		FuenteRef:                "norma-cotejo-2026-001",
		MotivoCreacion:           "Implantacion del servicio de cotejo documental",
		CreadaPor:                "tecnico-rrhh-1",
		CreadaEn:                 fecha,
	}
}

func politicaCotejoPublicadaPrueba(t *testing.T) PoliticaCotejo {
	t.Helper()
	borrador := politicaCotejoBorradorPrueba()
	publicada, err := borrador.Publicar(
		"responsable-rrhh-1",
		"aprobacion-cotejo-2026-001",
		"Aprobacion de la politica de cotejo",
		borrador.CreadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("PoliticaCotejo.Publicar() error = %v", err)
	}
	return publicada
}

func codigoCotejoReservadoPrueba(t *testing.T) CodigoCotejo {
	t.Helper()
	politica := politicaCotejoPublicadaPrueba(t)
	aplicacion, err := politica.Aplicacion()
	if err != nil {
		t.Fatalf("PoliticaCotejo.Aplicacion() error = %v", err)
	}
	reservadoEn := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	codigo := CodigoCotejo{
		ID:               "codigo-cotejo-001",
		Revision:         1,
		Documento:        ReferenciaDocumento{ID: "documento-001", Version: 1},
		ModuloID:         "bolsa",
		TipoDocumental:   "contrato",
		Clasificacion:    "datos_personales_alta",
		Organo:           "Diputacion de Granada",
		ExpedienteRef:    "expediente-042",
		IndiceCodigoHMAC: huellaIndiceCotejoPrueba,
		ProteccionRef:    "proteccion-csv-001",
		VersionGenerador: "generador-csv-v1",
		EntropiaBits:     160,
		Politica:         aplicacion,
		Estado:           EstadoCodigoCotejoReservado,
		ReservadoPor:     "tecnico-rrhh-1",
		ReservadoEn:      reservadoEn,
		ReservaExpiraEn:  reservadoEn.Add(15 * time.Minute),
		MotivoReserva:    "Reserva para la emision del contrato",
		CorrelacionRef:   "correlacion-cotejo-001",
	}
	if err := codigo.Validar(); err != nil {
		t.Fatalf("CodigoCotejo.Validar() fixture error = %v", err)
	}
	return codigo
}

func versionEmitidaCotejoPrueba(emitidaEn time.Time) VersionEmitidaCotejo {
	return VersionEmitidaCotejo{
		RepresentacionID:      "documento-001:representacion:firma:pdf:2",
		ReferenciaContenido:   "almacen:v1:documento-firmado-001",
		HuellaContenidoSHA256: huellaEmisionCotejoPrueba,
		MIME:                  "application/pdf",
		Tamano:                4_096,
		FirmaRefs:             []string{"firma-002", "firma-001"},
		SelloTiempoRefs:       []string{"sello-tiempo-002", "sello-tiempo-001"},
		ValidacionFirmaRef:    "validacion-dss-001",
		RegistroRef:           "registro-salida-001",
		EmitidaEn:             emitidaEn.UTC(),
	}
}

func evidenciaEmisionCotejoPrueba(codigo CodigoCotejo, emitidaEn time.Time) EvidenciaEmisionDocumento {
	return EvidenciaEmisionDocumento{
		Documento:      codigo.Documento,
		VersionEmitida: versionEmitidaCotejoPrueba(emitidaEn),
		Apta:           true,
		EvidenciaRef:   "evidencia-emision-001",
	}
}

func activarCodigoCotejoPrueba(t *testing.T, codigo CodigoCotejo, fecha time.Time) CodigoCotejo {
	t.Helper()
	evidencia := evidenciaEmisionCotejoPrueba(codigo, fecha.Add(-time.Minute))
	activo, err := codigo.Activar("responsable-registro-1", "activacion-cotejo-001", "Emision comprobada", evidencia, fecha)
	if err != nil {
		t.Fatalf("CodigoCotejo.Activar() error = %v", err)
	}
	return activo
}

func TestPoliticaCotejoCanonizaYCalculaHuellaEstable(t *testing.T) {
	politica := politicaCotejoBorradorPrueba()
	canonico, err := politica.ClonarCanonica()
	if err != nil {
		t.Fatalf("ClonarCanonica() error = %v", err)
	}
	if !reflect.DeepEqual(canonico.Modulos, []string{"bolsa", "personal"}) ||
		!reflect.DeepEqual(canonico.TiposDocumentales, []string{"contrato", "resolucion"}) ||
		!reflect.DeepEqual(canonico.Clasificaciones, []string{"datos_personales_alta", "publica"}) ||
		!reflect.DeepEqual(canonico.CamposPublicos, []CampoPublicoCotejo{
			CampoPublicoCotejoFechaEmision,
			CampoPublicoCotejoHuellaSHA256,
			CampoPublicoCotejoOrgano,
		}) {
		t.Fatalf("politica no canonica: %+v", canonico)
	}
	canonico.Modulos[0] = "alterado"
	canonico.CamposPublicos[0] = CampoPublicoCotejoTipoDocumental
	if politica.Modulos[0] != "personal" || politica.CamposPublicos[0] != CampoPublicoCotejoOrgano {
		t.Fatal("ClonarCanonica() comparte listas con la politica original")
	}

	huellaOriginal, err := politica.HuellaSHA256()
	if err != nil {
		t.Fatalf("HuellaSHA256() error = %v", err)
	}
	reordenada := politicaCotejoBorradorPrueba()
	reordenada.Modulos = []string{"bolsa", "personal"}
	reordenada.TiposDocumentales = []string{"contrato", "resolucion"}
	reordenada.Clasificaciones = []string{"datos_personales_alta", "publica"}
	reordenada.CamposPublicos = []CampoPublicoCotejo{
		CampoPublicoCotejoFechaEmision,
		CampoPublicoCotejoOrgano,
		CampoPublicoCotejoHuellaSHA256,
	}
	huellaReordenada, err := reordenada.HuellaSHA256()
	if err != nil {
		t.Fatalf("HuellaSHA256() reordenada error = %v", err)
	}
	if huellaOriginal != huellaReordenada || len(huellaOriginal) != 64 {
		t.Fatalf("huella no estable: original=%q reordenada=%q", huellaOriginal, huellaReordenada)
	}

	modificada := politicaCotejoBorradorPrueba()
	modificada.DiasDisponibilidad++
	huellaModificada, err := modificada.HuellaSHA256()
	if err != nil {
		t.Fatalf("HuellaSHA256() modificada error = %v", err)
	}
	if huellaModificada == huellaOriginal {
		t.Fatal("la huella no cambia al modificar la politica")
	}
}

func TestPoliticaCotejoRechazaListasDuplicadasYCamposInvalidos(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*PoliticaCotejo)
	}{
		{"modulos duplicados", func(p *PoliticaCotejo) { p.Modulos = []string{"bolsa", "bolsa"} }},
		{"tipos duplicados", func(p *PoliticaCotejo) { p.TiposDocumentales = []string{"contrato", "contrato"} }},
		{"clasificaciones duplicadas", func(p *PoliticaCotejo) { p.Clasificaciones = []string{"publica", "publica"} }},
		{"lista obligatoria vacia", func(p *PoliticaCotejo) { p.Modulos = nil }},
		{"clave no canonica", func(p *PoliticaCotejo) { p.Modulos = []string{"Bolsa"} }},
		{"campos publicos duplicados", func(p *PoliticaCotejo) {
			p.CamposPublicos = []CampoPublicoCotejo{CampoPublicoCotejoOrgano, CampoPublicoCotejoOrgano}
		}},
		{"campo publico desconocido", func(p *PoliticaCotejo) {
			p.CamposPublicos = []CampoPublicoCotejo{"dni"}
		}},
		{"campos en acceso interno", func(p *PoliticaCotejo) { p.ClaseAcceso = ClaseAccesoCotejoInterno }},
		{"plazo activacion nulo", func(p *PoliticaCotejo) { p.DiasPlazoActivacion = 0 }},
		{"plazo activacion excesivo", func(p *PoliticaCotejo) { p.DiasPlazoActivacion = maximoDiasActivacionCotejo + 1 }},
		{"disponibilidad nula", func(p *PoliticaCotejo) { p.DiasDisponibilidad = 0 }},
		{"disponibilidad excesiva", func(p *PoliticaCotejo) { p.DiasDisponibilidad = maximoDiasCotejo + 1 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			politica := politicaCotejoBorradorPrueba()
			caso.mutar(&politica)
			if err := politica.Validar(); !errors.Is(err, ErrPoliticaCotejoInvalida) {
				t.Fatalf("Validar() error = %v", err)
			}
		})
	}

	version := versionEmitidaCotejoPrueba(time.Date(2026, time.July, 14, 10, 1, 0, 0, time.UTC))
	version.FirmaRefs = []string{"firma-001", "firma-001"}
	if err := version.Validar(); !errors.Is(err, ErrVersionCotejoInvalida) {
		t.Fatalf("referencias de firma duplicadas: error = %v", err)
	}
}

func TestPoliticaCotejoPublicarRetirarYCrearAplicacionInmutable(t *testing.T) {
	borrador := politicaCotejoBorradorPrueba()
	if _, err := borrador.Aplicacion(); !errors.Is(err, ErrPoliticaCotejoNoPublicada) {
		t.Fatalf("Aplicacion() de borrador error = %v", err)
	}
	if _, err := borrador.Publicar("responsable-rrhh-1", "aprobacion-1", "Aprobada", borrador.CreadaEn.Add(-time.Nanosecond)); !errors.Is(err, ErrPoliticaCotejoInvalida) {
		t.Fatalf("publicacion anterior a creacion: error = %v", err)
	}

	publicadaEn := borrador.CreadaEn.Add(time.Hour)
	publicada, err := borrador.Publicar("responsable-rrhh-1", "aprobacion-1", "Aprobada", publicadaEn)
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	if publicada.Estado != EstadoPoliticaCotejoPublicada || publicada.Revision != 2 ||
		publicada.PublicadaPor != "responsable-rrhh-1" || publicada.AprobacionRef != "aprobacion-1" ||
		!publicada.PublicadaEn.Equal(publicadaEn) || borrador.Estado != EstadoPoliticaCotejoBorrador || borrador.Revision != 1 {
		t.Fatalf("transicion de publicacion incorrecta: borrador=%+v publicada=%+v", borrador, publicada)
	}
	aplicacion, err := publicada.Aplicacion()
	if err != nil {
		t.Fatalf("Aplicacion() error = %v", err)
	}
	huella, _ := publicada.HuellaSHA256()
	if aplicacion.Referencia.ID != publicada.ID || aplicacion.Referencia.Version != publicada.Version ||
		aplicacion.Referencia.HuellaSHA256 != huella ||
		aplicacion.DiasPlazoActivacion != publicada.DiasPlazoActivacion || aplicacion.Validar() != nil {
		t.Fatalf("aplicacion no fija la politica exacta: %+v", aplicacion)
	}
	campoPublicado := publicada.CamposPublicos[0]
	aplicacion.CamposPublicos[0] = CampoPublicoCotejoTipoDocumental
	if publicada.CamposPublicos[0] != campoPublicado {
		t.Fatal("Aplicacion() comparte campos con la politica publicada")
	}

	retiradaEn := publicadaEn.Add(time.Hour)
	retirada, err := publicada.Retirar("responsable-rrhh-2", "retirada-1", "Sustituida por nueva version", retiradaEn)
	if err != nil {
		t.Fatalf("Retirar() error = %v", err)
	}
	if retirada.Estado != EstadoPoliticaCotejoRetirada || retirada.Revision != 3 ||
		retirada.RetiradaPor != "responsable-rrhh-2" || retirada.RetiradaAprobacionRef != "retirada-1" ||
		!retirada.RetiradaEn.Equal(retiradaEn) || publicada.Estado != EstadoPoliticaCotejoPublicada {
		t.Fatalf("transicion de retirada incorrecta: publicada=%+v retirada=%+v", publicada, retirada)
	}
	if _, err := retirada.Aplicacion(); !errors.Is(err, ErrPoliticaCotejoNoPublicada) {
		t.Fatalf("Aplicacion() de retirada error = %v", err)
	}
	if _, err := retirada.Retirar("responsable-rrhh-2", "retirada-2", "Segunda retirada", retiradaEn.Add(time.Hour)); !errors.Is(err, ErrPoliticaCotejoInvalida) {
		t.Fatalf("segunda retirada: error = %v", err)
	}
}

func TestPoliticaCotejoAdmitePorModuloTipoYClasificacion(t *testing.T) {
	publicada := politicaCotejoPublicadaPrueba(t)
	documento := documentoLogicoValidoPrueba()
	if !publicada.Admite(documento) {
		t.Fatal("la politica publicada no admite el documento coincidente")
	}

	casos := []struct {
		nombre string
		mutar  func(*DocumentoLogico)
	}{
		{"otro modulo", func(d *DocumentoLogico) { d.ModuloID = "dietas" }},
		{"otro tipo", func(d *DocumentoLogico) { d.TipoDocumental = "certificado" }},
		{"otra clasificacion", func(d *DocumentoLogico) { d.Clasificacion = "interna" }},
		{"documento invalido", func(d *DocumentoLogico) { d.ID = "" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			otro := documentoLogicoValidoPrueba()
			caso.mutar(&otro)
			if publicada.Admite(otro) {
				t.Fatalf("Admite() = true para %+v", otro)
			}
		})
	}

	borrador := politicaCotejoBorradorPrueba()
	if borrador.Admite(documento) {
		t.Fatal("una politica no publicada admite documentos")
	}
	retirada, err := publicada.Retirar("responsable-rrhh-2", "retirada-1", "Fin de vigencia", publicada.PublicadaEn.Add(time.Hour))
	if err != nil {
		t.Fatalf("Retirar() error = %v", err)
	}
	if retirada.Admite(documento) {
		t.Fatal("una politica retirada admite documentos")
	}
}

func TestNormalizarCodigoCotejoYNoExponerCamposDeCustodiaEnJSON(t *testing.T) {
	const esperado = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	normalizado, err := NormalizarValorCodigoCotejo("  2345-6789 abcd-efgh jklm-npqr stuv-wxyz  ")
	if err != nil || normalizado != esperado {
		t.Fatalf("NormalizarValorCodigoCotejo() = %q, %v", normalizado, err)
	}
	invalidos := []string{
		strings.Repeat("A", 25),
		strings.Repeat("A", 97),
		strings.Repeat("A", 25) + "0",
		strings.Repeat("A", 25) + "I",
		strings.Repeat("A", 25) + "/",
		"",
	}
	for _, invalido := range invalidos {
		if _, err := NormalizarValorCodigoCotejo(invalido); !errors.Is(err, ErrCodigoCotejoInvalido) {
			t.Fatalf("codigo invalido %q: error = %v", invalido, err)
		}
	}

	codigo := codigoCotejoReservadoPrueba(t)
	contenido, err := json.Marshal(codigo)
	if err != nil {
		t.Fatalf("json.Marshal(CodigoCotejo) error = %v", err)
	}
	jsonTexto := string(contenido)
	for _, secreto := range []string{
		"expediente_ref", "indice_codigo_hmac", "proteccion_ref",
		codigo.ExpedienteRef, codigo.IndiceCodigoHMAC, codigo.ProteccionRef, esperado,
	} {
		if strings.Contains(jsonTexto, secreto) {
			t.Fatalf("JSON expone %q: %s", secreto, jsonTexto)
		}
	}
}

func TestHuellaEstadoCodigoCotejoIncluyeCamposOcultos(t *testing.T) {
	codigo := codigoCotejoReservadoPrueba(t)
	huellaBase, err := codigo.HuellaEstadoSHA256()
	if err != nil || len(huellaBase) != 64 {
		t.Fatalf("HuellaEstadoSHA256() = %q, %v", huellaBase, err)
	}
	casos := []struct {
		nombre string
		mutar  func(*CodigoCotejo)
	}{
		{"expediente", func(c *CodigoCotejo) { c.ExpedienteRef = "expediente-043" }},
		{"indice", func(c *CodigoCotejo) {
			c.IndiceCodigoHMAC = "hmac-sha256:indice-cotejo-v2:" + strings.Repeat("3", 64)
		}},
		{"proteccion", func(c *CodigoCotejo) { c.ProteccionRef = "proteccion-csv-002" }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			modificado := codigo
			caso.mutar(&modificado)
			huella, err := modificado.HuellaEstadoSHA256()
			if err != nil {
				t.Fatalf("HuellaEstadoSHA256() error = %v", err)
			}
			if huella == huellaBase {
				t.Fatalf("la huella no incluye el campo oculto %s", caso.nombre)
			}
		})
	}
}

func TestCodigoCotejoActivacionExigeDocumentoExactoYLimitesDeReserva(t *testing.T) {
	codigo := codigoCotejoReservadoPrueba(t)
	fechaValida := codigo.ReservaExpiraEn
	evidenciaValida := evidenciaEmisionCotejoPrueba(codigo, fechaValida)

	casos := []struct {
		nombre    string
		fecha     time.Time
		evidencia func() EvidenciaEmisionDocumento
	}{
		{
			nombre: "otra version documental",
			fecha:  fechaValida,
			evidencia: func() EvidenciaEmisionDocumento {
				e := evidenciaValida
				e.Documento.Version++
				return e
			},
		},
		{
			nombre:    "antes de la reserva",
			fecha:     codigo.ReservadoEn.Add(-time.Nanosecond),
			evidencia: func() EvidenciaEmisionDocumento { return evidenciaValida },
		},
		{
			nombre:    "despues de expirar",
			fecha:     codigo.ReservaExpiraEn.Add(time.Nanosecond),
			evidencia: func() EvidenciaEmisionDocumento { return evidenciaValida },
		},
		{
			nombre: "emision posterior a activacion",
			fecha:  fechaValida,
			evidencia: func() EvidenciaEmisionDocumento {
				e := evidenciaValida
				e.VersionEmitida.EmitidaEn = fechaValida.Add(time.Nanosecond)
				return e
			},
		},
		{
			nombre: "evidencia no apta",
			fecha:  fechaValida,
			evidencia: func() EvidenciaEmisionDocumento {
				e := evidenciaValida
				e.Apta = false
				return e
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := codigo.Activar("responsable-registro-1", "activacion-cotejo-001", "Emision comprobada", caso.evidencia(), caso.fecha); !errors.Is(err, ErrTransicionCodigoCotejo) {
				t.Fatalf("Activar() error = %v", err)
			}
		})
	}

	activo, err := codigo.Activar("responsable-registro-1", "activacion-cotejo-001", "Emision comprobada", evidenciaValida, fechaValida)
	if err != nil {
		t.Fatalf("Activar() en limite exacto error = %v", err)
	}
	if activo.Estado != EstadoCodigoCotejoActivo || activo.Revision != 2 ||
		!activo.ActivadoEn.Equal(fechaValida) || !activo.DisponibleDesde.Equal(fechaValida) ||
		!activo.DisponibleHasta.Equal(fechaValida.AddDate(0, 0, codigo.Politica.DiasDisponibilidad)) ||
		activo.VersionEmitida == nil || activo.VersionEmitida.HuellaContenidoSHA256 != huellaEmisionCotejoPrueba ||
		codigo.Estado != EstadoCodigoCotejoReservado || codigo.Revision != 1 {
		t.Fatalf("activacion incorrecta: reservado=%+v activo=%+v", codigo, activo)
	}
	if !reflect.DeepEqual(activo.VersionEmitida.FirmaRefs, []string{"firma-001", "firma-002"}) ||
		!reflect.DeepEqual(activo.VersionEmitida.SelloTiempoRefs, []string{"sello-tiempo-001", "sello-tiempo-002"}) {
		t.Fatalf("version emitida no canonica: %+v", activo.VersionEmitida)
	}
	evidenciaValida.VersionEmitida.FirmaRefs[0] = "firma-alterada"
	if activo.VersionEmitida.FirmaRefs[0] != "firma-001" {
		t.Fatal("Activar() comparte referencias con la evidencia de entrada")
	}
}

func TestCodigoCotejoDisponibilidadUsaIntervaloSemiabierto(t *testing.T) {
	codigo := codigoCotejoReservadoPrueba(t)
	activadoEn := codigo.ReservadoEn.Add(5 * time.Minute)
	activo := activarCodigoCotejoPrueba(t, codigo, activadoEn)

	casos := []struct {
		nombre     string
		instante   time.Time
		disponible bool
	}{
		{"antes", activo.DisponibleDesde.Add(-time.Nanosecond), false},
		{"inicio incluido", activo.DisponibleDesde, true},
		{"antes del final", activo.DisponibleHasta.Add(-time.Nanosecond), true},
		{"final excluido", activo.DisponibleHasta, false},
		{"despues", activo.DisponibleHasta.Add(time.Nanosecond), false},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if obtenido := activo.DisponibleEn(caso.instante); obtenido != caso.disponible {
				t.Fatalf("DisponibleEn(%s) = %t, quiere %t", caso.instante, obtenido, caso.disponible)
			}
		})
	}
}

func TestCodigoCotejoRetiraSustituyeYClonaSinAlias(t *testing.T) {
	codigo := codigoCotejoReservadoPrueba(t)
	activadoEn := codigo.ReservadoEn.Add(5 * time.Minute)
	activo := activarCodigoCotejoPrueba(t, codigo, activadoEn)
	if !activo.TieneCampoPublico(CampoPublicoCotejoOrgano) || activo.TieneCampoPublico(CampoPublicoCotejoTipoDocumental) {
		t.Fatalf("TieneCampoPublico() no respeta la aplicacion: %+v", activo.Politica.CamposPublicos)
	}

	clon, err := activo.ClonarCanonico()
	if err != nil {
		t.Fatalf("ClonarCanonico() error = %v", err)
	}
	clon.Politica.CamposPublicos[0] = CampoPublicoCotejoTipoDocumental
	clon.VersionEmitida.FirmaRefs[0] = "firma-alterada"
	clon.VersionEmitida.SelloTiempoRefs[0] = "sello-alterado"
	if activo.Politica.CamposPublicos[0] == CampoPublicoCotejoTipoDocumental ||
		activo.VersionEmitida.FirmaRefs[0] == "firma-alterada" || activo.VersionEmitida.SelloTiempoRefs[0] == "sello-alterado" {
		t.Fatal("ClonarCanonico() comparte listas con el codigo original")
	}

	if _, err := activo.Retirar("responsable-rrhh-1", "retirada-cotejo-001", "Retirada anticipada", activadoEn.Add(-time.Nanosecond)); !errors.Is(err, ErrTransicionCodigoCotejo) {
		t.Fatalf("retirada anterior a activacion: error = %v", err)
	}
	retiradoEn := activadoEn.Add(time.Hour)
	retirado, err := activo.Retirar("responsable-rrhh-1", "retirada-cotejo-001", "Retirada anticipada", retiradoEn)
	if err != nil {
		t.Fatalf("Retirar() error = %v", err)
	}
	if retirado.Estado != EstadoCodigoCotejoRetirado || retirado.Revision != activo.Revision+1 ||
		retirado.RetiradoPor != "responsable-rrhh-1" || retirado.RetiradaRef != "retirada-cotejo-001" ||
		!retirado.RetiradoEn.Equal(retiradoEn) || retirado.SustituidoPorRef != "" || retirado.DisponibleEn(retiradoEn) ||
		activo.Estado != EstadoCodigoCotejoActivo {
		t.Fatalf("retirada incorrecta: activo=%+v retirado=%+v", activo, retirado)
	}

	if _, err := activo.Sustituir("responsable-rrhh-1", "sustitucion-001", "Nueva emision", activo.Referencia(), retiradoEn); !errors.Is(err, ErrTransicionCodigoCotejo) {
		t.Fatalf("sustitucion por si mismo: error = %v", err)
	}
	const sustituto = "cotejo:codigo-cotejo-002"
	sustituido, err := activo.Sustituir("responsable-rrhh-1", "sustitucion-001", "Nueva emision", sustituto, retiradoEn)
	if err != nil {
		t.Fatalf("Sustituir() error = %v", err)
	}
	if sustituido.Estado != EstadoCodigoCotejoSustituido || sustituido.Revision != activo.Revision+1 ||
		sustituido.SustituidoPorRef != sustituto || sustituido.RetiradaRef != "sustitucion-001" ||
		sustituido.DisponibleEn(retiradoEn) {
		t.Fatalf("sustitucion incorrecta: %+v", sustituido)
	}
	if _, err := sustituido.Retirar("responsable-rrhh-1", "retirada-2", "No permitida", retiradoEn.Add(time.Hour)); !errors.Is(err, ErrTransicionCodigoCotejo) {
		t.Fatalf("retirada tras sustitucion: error = %v", err)
	}
}
